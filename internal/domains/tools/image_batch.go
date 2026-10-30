package tools

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/domains"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/hoi4/imagebatch"
)

const (
	imageBatchMaxFiles        = 200
	imageBatchMaxFileBytes    = int64(10 << 20)
	imageBatchMaxPayloadBytes = int64(256 << 20)
	imageBatchMultipartSlack  = int64(4 << 20)
	imageBatchMaxTotalPixels  = int64(64 * 1024 * 1024)
	imageBatchMaxOutputBytes  = int64(128 << 20)
)

func registerImageBatch(mux *http.ServeMux, deps domains.Deps) {
	mux.HandleFunc("POST /api/go/image-batch/convert-dds", handleImageBatchConvertDDS)
	mux.HandleFunc("POST /api/go/image-batch/generate-placeholder", handleImageBatchGeneratePlaceholder)
	mux.HandleFunc("POST /api/go/image-batch/overlay-text", handleImageBatchOverlayText)
	mux.HandleFunc("GET /api/go/image-batch/vanilla-root-status", func(w http.ResponseWriter, r *http.Request) {
		handleImageBatchVanillaRootStatus(w, deps.Game)
	})
	mux.HandleFunc("GET /api/go/image-batch/vanilla-extract/{type...}", func(w http.ResponseWriter, r *http.Request) {
		handleImageBatchVanillaExtract(w, r, deps.Game)
	})
}

type overlayItemRequest struct {
	B64Image    string `json:"b64_image"`
	Text        string `json:"text"`
	Position    string `json:"position"`
	Margin      *int   `json:"margin"`
	FontSize    *int   `json:"font_size"`
	FontPath    string `json:"font_path"`
	Color       []int  `json:"color"`
	StrokeColor []int  `json:"stroke_color"`
	StrokeWidth *int   `json:"stroke_width"`
}

type overlayBatchRequest struct {
	Items  []overlayItemRequest `json:"items"`
	Output string               `json:"output"`
}

type imageBatchBudget struct {
	pixels int64
	output int64
}

func (b *imageBatchBudget) addPixels(pixels int64) error {
	if pixels < 0 || b.pixels+pixels > imageBatchMaxTotalPixels {
		return errors.New("Batch decoded dimensions exceed safe limit; split into smaller batches")
	}
	b.pixels += pixels
	return nil
}

func (b *imageBatchBudget) addOutput(size int64) error {
	if size < 0 || b.output+size > imageBatchMaxOutputBytes {
		return errors.New("Batch output exceeds 128MB; split into smaller batches")
	}
	b.output += size
	return nil
}

func overlayRGB(values []int, fallback color.RGBA) (color.RGBA, error) {
	if len(values) != 3 {
		return fallback, nil
	}
	for _, value := range values {
		if value < 0 || value > 255 {
			return color.RGBA{}, errors.New("RGB values must be integers between 0 and 255")
		}
	}
	return color.RGBA{R: uint8(values[0]), G: uint8(values[1]), B: uint8(values[2]), A: 255}, nil
}

func handleImageBatchOverlayText(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, imageBatchMaxPayloadBytes+imageBatchMultipartSlack)
	var in overlayBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"detail": "Batch too large (> 256MB total). Split into smaller batches."})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "Invalid JSON body"})
		return
	}
	if len(in.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "items list is empty"})
		return
	}
	if len(in.Items) > imageBatchMaxFiles {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": fmt.Sprintf("Too many items (%d). Max 200 per batch.", len(in.Items))})
		return
	}
	output := strings.ToLower(strings.TrimSpace(in.Output))
	if output == "" {
		output = "json"
	}
	if output != "json" && output != "zip" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "output must be 'json' or 'zip'"})
		return
	}
	type result struct {
		Index  int
		Status string
		PNG    []byte
		Width  int
		Height int
		Error  string
	}
	results := make([]result, 0, len(in.Items))
	var decodedTotal int64
	var budget imageBatchBudget
	for index, item := range in.Items {
		row := result{Index: index, Status: "error"}
		raw, err := base64.StdEncoding.Strict().DecodeString(item.B64Image)
		if err != nil {
			row.Error = "b64_image is not valid base64"
			results = append(results, row)
			continue
		}
		decodedTotal += int64(len(raw))
		if int64(len(raw)) > imageBatchMaxFileBytes {
			row.Error = fmt.Sprintf("File too large (%.1f MB · max 10 MB)", float64(len(raw))/(1024*1024))
			results = append(results, row)
			continue
		}
		if decodedTotal > imageBatchMaxPayloadBytes {
			row.Error = "Batch too large (> 256MB total). Split into smaller batches."
			results = append(results, row)
			continue
		}
		width, height, dimensionErr := imagebatch.Dimensions(raw, "overlay.png")
		pixels, dimensionsOK := imagebatch.BoundedPixelCount(width, height)
		if dimensionErr != nil || !dimensionsOK {
			row.Error = "could not decode source image bytes"
			results = append(results, row)
			continue
		}
		if err := budget.addPixels(pixels); err != nil {
			row.Error = err.Error()
			results = append(results, row)
			continue
		}
		margin, stroke := 16, 2
		if item.Margin != nil {
			margin = *item.Margin
		}
		if item.StrokeWidth != nil {
			stroke = *item.StrokeWidth
		}
		foreground, colorErr := overlayRGB(item.Color, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		outline, strokeColorErr := overlayRGB(item.StrokeColor, color.RGBA{A: 255})
		if colorErr != nil || strokeColorErr != nil {
			if colorErr != nil {
				row.Error = colorErr.Error()
			} else {
				row.Error = strokeColorErr.Error()
			}
			results = append(results, row)
			continue
		}
		overlay, err := imagebatch.OverlayText(raw, "overlay.png", imagebatch.OverlaySpec{
			Text: item.Text, Position: item.Position, Margin: margin,
			FontSize: item.FontSize, FontPath: item.FontPath,
			Color: foreground, StrokeColor: outline, StrokeWidth: stroke,
		})
		if err != nil {
			row.Error = err.Error()
			results = append(results, row)
			continue
		}
		if err := budget.addOutput(int64(len(overlay.PNG))); err != nil {
			row.Error = err.Error()
			results = append(results, row)
			continue
		}
		row.Status, row.PNG, row.Width, row.Height = "ok", overlay.PNG, overlay.Width, overlay.Height
		results = append(results, row)
	}
	okCount := 0
	for _, row := range results {
		if row.Status == "ok" {
			okCount++
		}
	}
	var fontUsed any
	if path := imagebatch.FindOverlayFont(); path != "" {
		fontUsed = path
	}
	summary := map[string]any{"count": len(results), "ok": okCount, "errors": len(results) - okCount, "font_used": fontUsed}
	if output == "json" {
		rows := make([]map[string]any, 0, len(results))
		for _, row := range results {
			var encoded any
			var width, height any
			if len(row.PNG) != 0 {
				encoded = base64.StdEncoding.EncodeToString(row.PNG)
				width, height = row.Width, row.Height
			}
			var rowError any
			if row.Error != "" {
				rowError = row.Error
			}
			rows = append(rows, map[string]any{"index": row.Index, "status": row.Status, "width": width, "height": height, "b64_output": encoded, "error": rowError})
		}
		summary["results"] = rows
		writeJSON(w, http.StatusOK, summary)
		return
	}
	var payload bytes.Buffer
	zw := zip.NewWriter(&payload)
	errorsDetail := make([]map[string]any, 0)
	for _, row := range results {
		if row.Status != "ok" {
			errorsDetail = append(errorsDetail, map[string]any{"index": row.Index, "error": row.Error})
			continue
		}
		entry, err := zw.Create(fmt.Sprintf("overlay_%03d.png", row.Index))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "Could not build overlay ZIP"})
			return
		}
		_, _ = entry.Write(row.PNG)
	}
	summary["errors_detail"] = errorsDetail
	manifest, _ := json.MarshalIndent(summary, "", "  ")
	entry, err := zw.Create("manifest.json")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "Could not build overlay ZIP"})
		return
	}
	_, _ = entry.Write(manifest)
	if err := zw.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "Could not build overlay ZIP"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="overlays.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload.Bytes())
}

type placeholderEntityRequest struct {
	EntityID string `json:"entity_id"`
	BGColor  []int  `json:"bg_color"`
	Label    string `json:"label"`
}

type placeholderBatchRequest struct {
	Entities []placeholderEntityRequest `json:"entities"`
	Kind     string                     `json:"kind"`
	Width    *int                       `json:"width"`
	Height   *int                       `json:"height"`
	Output   string                     `json:"output"`
}

var placeholderKindSizes = map[string][2]int{
	"idea_picture": {235, 104}, "idea": {235, 104},
	"event": {165, 165}, "event_picture": {165, 165},
	"decision": {152, 80}, "decision_icon": {152, 80},
	"focus": {95, 95}, "focus_icon": {95, 95},
}

func handleImageBatchGeneratePlaceholder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	var in placeholderBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "Invalid JSON body"})
		return
	}
	if len(in.Entities) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "entities list is empty"})
		return
	}
	if len(in.Entities) > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": fmt.Sprintf("Too many entities (%d). Max 1000 per batch.", len(in.Entities))})
		return
	}
	size := [2]int{235, 104}
	if known, ok := placeholderKindSizes[strings.ToLower(strings.TrimSpace(in.Kind))]; ok {
		size = known
	}
	if in.Width != nil {
		size[0] = *in.Width
	}
	if in.Height != nil {
		size[1] = *in.Height
	}
	if size[0] <= 0 || size[1] <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "width/height must be positive"})
		return
	}
	if _, err := placeholderBatchPixels(len(in.Entities), size[0], size[1]); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	specs := make([]imagebatch.PlaceholderSpec, 0, len(in.Entities))
	for _, entity := range in.Entities {
		bg := [3]uint8{60, 60, 80}
		if len(entity.BGColor) == 3 {
			for i, value := range entity.BGColor {
				bg[i] = uint8(min(max(value, 0), 255))
			}
		}
		specs = append(specs, imagebatch.PlaceholderSpec{
			EntityID: entity.EntityID, Label: entity.Label,
			BG: color.RGBA{R: bg[0], G: bg[1], B: bg[2], A: 255},
		})
	}
	results, err := imagebatch.GeneratePlaceholdersLimited(specs, size[0], size[1], imageBatchMaxOutputBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	if strings.EqualFold(strings.TrimSpace(in.Output), "json") {
		rows := make([]map[string]any, 0, len(results))
		for _, result := range results {
			rows = append(rows, map[string]any{"entity_id": result.EntityID, "filename": result.Filename, "b64": base64.StdEncoding.EncodeToString(result.PNG), "size_bytes": len(result.PNG)})
		}
		writeJSON(w, http.StatusOK, map[string]any{"count": len(rows), "results": rows})
		return
	}
	var payload bytes.Buffer
	zw := zip.NewWriter(&payload)
	for _, result := range results {
		entry, err := zw.Create(result.Filename)
		if err == nil {
			_, err = entry.Write(result.PNG)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "Could not build placeholder ZIP"})
			return
		}
	}
	if err := zw.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "Could not build placeholder ZIP"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="placeholders.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload.Bytes())
}

func placeholderBatchPixels(count, width, height int) (int64, error) {
	if count <= 0 || width <= 0 || height <= 0 {
		return 0, errors.New("count/width/height must be positive")
	}
	w, h, n := int64(width), int64(height), int64(count)
	if w > imagebatch.MaxPixels/h {
		return 0, errors.New("placeholder dimensions exceed safe per-image pixel limit")
	}
	perImage := w * h
	if n > imageBatchMaxTotalPixels/perImage {
		return 0, errors.New("Placeholder batch dimensions exceed 64Mpx; split into smaller batches")
	}
	return n * perImage, nil
}

func gameFolderPath(fsys fs.FS) string {
	if fsys == nil {
		return ""
	}
	if value := reflect.ValueOf(fsys); value.Kind() == reflect.String {
		return value.String()
	}
	return ""
}

func handleImageBatchVanillaRootStatus(w http.ResponseWriter, game func() (fs.FS, error)) {
	g, gameErr := gamefiles.Open(game)
	exists := gameErr == nil
	dirs := make([]map[string]any, 0, len(imagebatch.VanillaAssetDirs))
	var fsys fs.FS
	if exists {
		fsys = g.FS()
		for _, assetType := range imagebatch.VanillaAssetTypes {
			subdir := imagebatch.VanillaAssetDirs[assetType]
			count := len(g.Files(subdir, false, ".dds"))
			info, err := fs.Stat(fsys, subdir)
			dirExists := count > 0 || err == nil && info.IsDir()
			dirs = append(dirs, map[string]any{"asset_type": assetType, "subdir": subdir, "exists": dirExists, "dds_count": count})
		}
	}
	var hint any
	if !exists {
		hint = "Set the Hearts of Iron IV folder in settings first."
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured_via_env": false, "env_value": nil, "resolved_path": gameFolderPath(fsys),
		"root_exists": exists, "asset_dirs": dirs, "hint": hint,
	})
}

func handleImageBatchVanillaExtract(w http.ResponseWriter, r *http.Request, game func() (fs.FS, error)) {
	assetType := strings.Trim(r.PathValue("type"), "/")
	subdir, ok := imagebatch.VanillaAssetDirs[assetType]
	if !ok {
		keys := make([]string, 0, len(imagebatch.VanillaAssetDirs))
		for key := range imagebatch.VanillaAssetDirs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": fmt.Sprintf("Unknown asset_type %q. Expected one of: %v", assetType, keys)})
		return
	}
	assets := []imagebatch.VanillaAsset{}
	if g, err := gamefiles.Open(game); err == nil {
		for _, rel := range g.Files(subdir, false, ".dds") {
			name := path.Base(rel)
			assets = append(assets, imagebatch.VanillaAsset{
				Name: strings.TrimSuffix(name, path.Ext(name)), Filename: name,
				RelativePath: rel,
			})
		}
		sort.Slice(assets, func(i, j int) bool { return assets[i].Filename < assets[j].Filename })
	}
	writeJSON(w, http.StatusOK, map[string]any{"asset_type": assetType, "count": len(assets), "assets": assets})
}

func handleImageBatchConvertDDS(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, imageBatchMaxPayloadBytes+imageBatchMultipartSlack)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"detail": "Batch too large (> 256MB total). Split into smaller batches."})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "Invalid multipart upload"})
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	targetWidthValue := strings.TrimSpace(r.FormValue("target_width"))
	targetHeightValue := strings.TrimSpace(r.FormValue("target_height"))
	if (targetWidthValue == "") != (targetHeightValue == "") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "Provide both target_width and target_height or neither"})
		return
	}
	targetWidth, targetHeight := 0, 0
	if targetWidthValue != "" {
		var widthErr, heightErr error
		targetWidth, widthErr = strconv.Atoi(targetWidthValue)
		targetHeight, heightErr = strconv.Atoi(targetHeightValue)
		if widthErr != nil || heightErr != nil || targetWidth <= 0 || targetHeight <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "target_width/target_height must be positive integers"})
			return
		}
		if _, ok := imagebatch.BoundedPixelCount(targetWidth, targetHeight); !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "target_width/target_height exceed safe pixel limit"})
			return
		}
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "No files uploaded"})
		return
	}
	if len(files) > imageBatchMaxFiles {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": fmt.Sprintf("Too many files (%d). Max %d per batch.", len(files), imageBatchMaxFiles)})
		return
	}
	var totalBytes int64
	for _, file := range files {
		if file.Size > 0 {
			totalBytes += file.Size
		}
		if totalBytes > imageBatchMaxPayloadBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"detail": "Batch too large (> 256MB total). Split into smaller batches."})
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	writeImageBatchSSE(w, flusher, map[string]any{"event": "start", "total": len(files)})
	okCount, errorCount := 0, 0
	var totalPixels int64
	for index, file := range files {
		frame := map[string]any{
			"event": "file", "index": index, "name": file.Filename,
			"progress": int(100 * (index + 1) / len(files)),
		}
		result, width, height, err := convertImageToDDS(file, targetWidth, targetHeight, &totalPixels)
		if err != nil {
			frame["status"] = "error"
			frame["error"] = err.Error()
			errorCount++
		} else {
			frame["status"] = "ok"
			frame["b64_output"] = base64.StdEncoding.EncodeToString(result)
			frame["width"] = width
			frame["height"] = height
			frame["output_size_bytes"] = len(result)
			okCount++
		}
		writeImageBatchSSE(w, flusher, frame)
	}
	writeImageBatchSSE(w, flusher, map[string]any{
		"event": "done", "total": len(files), "ok": okCount, "errors": errorCount,
	})
}

func convertImageToDDS(file *multipart.FileHeader, targetWidth, targetHeight int, totalPixels *int64) ([]byte, int, int, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, 0, 0, errors.New("Could not read image")
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, imageBatchMaxFileBytes+1))
	if err != nil {
		return nil, 0, 0, errors.New("Could not read image")
	}
	if int64(len(raw)) > imageBatchMaxFileBytes {
		return nil, 0, 0, fmt.Errorf("File too large (%.1f MB · max 10 MB)", float64(len(raw))/(1024*1024))
	}
	width, height, err := imagebatch.Dimensions(raw, file.Filename)
	if err != nil {
		return nil, 0, 0, errors.New("Could not decode image (corrupt or unsupported format)")
	}
	if _, ok := imagebatch.BoundedPixelCount(width, height); !ok {
		return nil, 0, 0, errors.New("Could not decode image (dimensions exceed safe limit)")
	}
	workPixels := effectiveConversionPixels(width, height, targetWidth, targetHeight)
	if *totalPixels+workPixels > imageBatchMaxTotalPixels {
		return nil, 0, 0, errors.New("Batch decoded dimensions exceed safe limit; split into smaller batches")
	}
	*totalPixels += workPixels
	return imagebatch.ConvertDDSWithSize(raw, file.Filename, targetWidth, targetHeight)
}

func effectiveConversionPixels(width, height, targetWidth, targetHeight int) int64 {
	pixels, _ := imagebatch.BoundedPixelCount(width, height)
	if targetWidth > 0 && targetHeight > 0 {
		if targetPixels, _ := imagebatch.BoundedPixelCount(targetWidth, targetHeight); targetPixels > pixels {
			pixels = targetPixels
		}
	}
	return pixels
}

func writeImageBatchSSE(w io.Writer, flusher http.Flusher, payload map[string]any) {
	var body strings.Builder
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(payload)
	_, _ = io.WriteString(w, "data: "+strings.TrimSuffix(body.String(), "\n")+"\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}
