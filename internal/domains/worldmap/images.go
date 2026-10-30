package worldmap

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/hoi4/mapsketch"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	worldImageMaxBytes  = 16 << 20
	worldImageMaxPixels = 50_000_000
	worldImageMaxSide   = 3072

	worldImageMaxTags = 32

	worldImagePreviewSide = 1024
)

var (
	worldImageIDPattern = regexp.MustCompile(`^wimg_[0-9a-f]{16}$`)

	worldImageGate = make(chan struct{}, 2)
	errWorldImage  = errors.New("world image missing")
)

type worldImageObjects interface {
	Put(key, contentType string, size int64, body io.Reader) error
	GetOptional(key string, maxBytes int64) ([]byte, bool, error)
	EnvPrefix() string
}

type blobObjects struct{ blobs *project.BlobStore }

func (b blobObjects) Put(key, contentType string, size int64, body io.Reader) error {
	return b.blobs.Put(key, contentType, size, body)
}

func (b blobObjects) GetOptional(key string, maxBytes int64) ([]byte, bool, error) {
	if !b.blobs.Exists(key) {
		return nil, false, nil
	}
	body, err := b.blobs.Get(key, maxBytes)
	return body, err == nil, err
}

func (b blobObjects) EnvPrefix() string { return b.blobs.EnvPrefix() }

type worldImageMeta struct {
	ID string `json:"id"`

	Kind string `json:"kind"`

	Format string `json:"format,omitempty"`

	Origin     string `json:"origin"`
	SourceKind string `json:"source_kind,omitempty"`
	From       string `json:"from,omitempty"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`

	SourceWidth  int                  `json:"source_width,omitempty"`
	SourceHeight int                  `json:"source_height,omitempty"`
	Placement    *mapsketch.Placement `json:"placement,omitempty"`

	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func worldImageStore(p *project.Project) worldImageObjects { return blobObjects{blobs: p.Blobs()} }

func worldImageKey(env string, projectID int64, id, ext string) string {
	return fmt.Sprintf("%s/world-images/%d/%s.%s", strings.Trim(strings.TrimSpace(env), "/"), projectID, id, ext)
}

func newWorldImageID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "wimg_" + hex.EncodeToString(b[:])
}

func putWorldImage(objects worldImageObjects, projectID int64, meta *worldImageMeta, img image.Image) error {
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, img); err != nil {
		return err
	}
	meta.Width, meta.Height = img.Bounds().Dx(), img.Bounds().Dy()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}
	if err := objects.Put(worldImageKey(objects.EnvPrefix(), projectID, meta.ID, "png"), "image/png", int64(buf.Len()), bytes.NewReader(buf.Bytes())); err != nil {
		return err
	}
	encoded, _ := json.Marshal(meta)
	return objects.Put(worldImageKey(objects.EnvPrefix(), projectID, meta.ID, "json"), "application/json", int64(len(encoded)), bytes.NewReader(encoded))
}

func worldImageKeepPreview(objects worldImageObjects, projectID int64, from, dataURL string) string {
	header, payload, ok := strings.Cut(dataURL, ",")
	format := map[string]string{"data:image/jpeg;base64": "jpg", "data:image/png;base64": "png"}[header]
	if !ok || format == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	meta := &worldImageMeta{ID: newWorldImageID(), Kind: "preview", Origin: "preview", From: from, Format: format,
		Width: cfg.Width, Height: cfg.Height, CreatedAt: time.Now().UTC()}
	contentType := map[string]string{"jpg": "image/jpeg", "png": "image/png"}[format]
	if err := objects.Put(worldImageKey(objects.EnvPrefix(), projectID, meta.ID, format), contentType, int64(len(data)), bytes.NewReader(data)); err != nil {
		log.Printf("world image preview %d: %v", projectID, err)
		return ""
	}
	encoded, _ := json.Marshal(meta)
	if err := objects.Put(worldImageKey(objects.EnvPrefix(), projectID, meta.ID, "json"), "application/json", int64(len(encoded)), bytes.NewReader(encoded)); err != nil {
		log.Printf("world image preview %d: %v", projectID, err)
		return ""
	}
	return worldImageURL(projectID, meta.ID)
}

func worldImageURL(projectID int64, id string) string {
	return fmt.Sprintf("/api/go/projects/%d/gui-render/world-map/image?id=%s", projectID, id)
}

func loadWorldImageMeta(objects worldImageObjects, projectID int64, id string) (*worldImageMeta, error) {
	if !worldImageIDPattern.MatchString(id) {
		return nil, errWorldImage
	}
	raw, found, err := objects.GetOptional(worldImageKey(objects.EnvPrefix(), projectID, id, "json"), 1<<20)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errWorldImage
	}
	var meta worldImageMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func loadWorldImage(objects worldImageObjects, projectID int64, id string) (*worldImageMeta, image.Image, error) {
	meta, err := loadWorldImageMeta(objects, projectID, id)
	if err != nil {
		return nil, nil, err
	}
	raw, found, err := objects.GetOptional(worldImageKey(objects.EnvPrefix(), projectID, id, "png"), 64<<20)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, errWorldImage
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, err
	}
	return meta, img, nil
}

func worldImageLoadFailed(w http.ResponseWriter, projectID int64, id string, err error) {
	if errors.Is(err, errWorldImage) {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "没有这张图:" + id + "(图片编号来自 image-import 等步骤的返回)"})
		return
	}
	log.Printf("world image %d %s: %v", projectID, id, err)
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "读不到这张图(图片存储出错)"})
}

func decodeWorldImage(data []byte) (image.Image, string) {
	if len(data) == 0 {
		return nil, "图片是空的"
	}
	if len(data) > worldImageMaxBytes {
		return nil, fmt.Sprintf("图片太大:最多 %d MB", worldImageMaxBytes>>20)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "读不出这张图:要是 PNG 或 JPEG"
	}
	if cfg.Width < 16 || cfg.Height < 16 || cfg.Width*cfg.Height > worldImageMaxPixels {
		return nil, fmt.Sprintf("图片尺寸 %d×%d 不行:至少 16×16,最多 %d 万像素", cfg.Width, cfg.Height, worldImageMaxPixels/10000)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "读不出这张图:要是 PNG 或 JPEG"
	}
	picture := mapsketch.FromImage(img).Scaled(worldImageMaxSide)
	return picture.Image(), ""
}

func decodeWorldImageJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, worldImageMaxBytes*2))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "请求读不懂:" + err.Error()})
		return false
	}
	return true
}

func worldImagePreview(img image.Image) string {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func worldImageGateAcquire(w http.ResponseWriter) (func(), bool) {
	select {
	case worldImageGate <- struct{}{}:
		return func() { <-worldImageGate }, true
	default:
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"detail": "另外两张图正在处理,稍等几秒再试"})
		return nil, false
	}
}

func (h *handlers) worldImageRead(w http.ResponseWriter, r *http.Request, p *project.Project) {
	projectID, objects := p.ID(), worldImageStore(p)
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	meta, err := loadWorldImageMeta(objects, projectID, id)
	if err != nil {
		worldImageLoadFailed(w, projectID, id, err)
		return
	}
	format, contentType := "png", "image/png"
	if meta.Format == "jpg" {
		format, contentType = "jpg", "image/jpeg"
	}
	raw, found, err := objects.GetOptional(worldImageKey(objects.EnvPrefix(), projectID, id, format), 64<<20)
	if err != nil || !found {
		worldImageLoadFailed(w, projectID, id, errors.Join(err, errWorldImage))
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))

	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

type worldImageImportBody struct {
	DataBase64 string          `json:"data_base64,omitempty"`
	ImageID    string          `json:"image_id,omitempty"`
	Crop       *mapsketch.Rect `json:"crop,omitempty"`
}

func (h *handlers) worldImageImport(w http.ResponseWriter, r *http.Request, p *project.Project) {
	projectID, objects := p.ID(), worldImageStore(p)
	release, ok := worldImageGateAcquire(w)
	if !ok {
		return
	}
	defer release()
	meta := &worldImageMeta{ID: newWorldImageID(), Kind: "picture"}
	var img image.Image
	problem := ""
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, worldImageMaxBytes+(1<<20))
		file, _, err := r.FormFile("image")
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "没有收到图片(表单字段 image)"})
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, worldImageMaxBytes+1))
		_ = file.Close()
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "图片没有收完整"})
			return
		}
		meta.Origin = "upload"
		img, problem = decodeWorldImage(data)
	} else {
		var body worldImageImportBody
		if !decodeWorldImageJSON(w, r, &body) {
			return
		}
		var data []byte
		switch {
		case body.DataBase64 != "":
			meta.Origin = "data"
			encoded := body.DataBase64
			if comma := strings.Index(encoded, ","); strings.HasPrefix(encoded, "data:") && comma > 0 {
				encoded = encoded[comma+1:]
			}
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
			if err != nil {
				problem = "data_base64 不是 base64"
			}
			data = decoded
		case body.ImageID != "":
			meta.Origin, meta.From = "crop", body.ImageID
			source, picture, err := loadWorldImage(objects, projectID, body.ImageID)
			if err != nil {
				worldImageLoadFailed(w, projectID, body.ImageID, err)
				return
			}
			if source.Kind != "picture" {
				problem = "只能从图片(kind=picture)裁剪"
				break
			}
			if body.Crop == nil || body.Crop.W < 16 || body.Crop.H < 16 {
				problem = "裁剪要给 crop {x, y, width, height},至少 16×16"
				break
			}
			cropped := mapsketch.FromImage(picture).Crop(*body.Crop)
			if cropped.W < 16 || cropped.H < 16 {
				problem = fmt.Sprintf("裁剪框不在图上(图是 %d×%d)", source.Width, source.Height)
				break
			}
			img = cropped.Image()
		default:
			problem = "要给 data_base64,或者 image_id + crop,或者用表单上传 image"
		}
		if problem == "" && img == nil {
			img, problem = decodeWorldImage(data)
		}
	}
	if problem != "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": problem})
		return
	}
	if err := putWorldImage(objects, projectID, meta, img); err != nil {
		log.Printf("world image import %d: %v", projectID, err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "图片没能存下(图片存储出错)"})
		return
	}
	picture := mapsketch.FromImage(img)
	writeJSON(w, http.StatusCreated, map[string]any{
		"image":   meta,
		"preview": worldImagePreview(mapsketch.GridPreview(picture, worldImagePreviewSide, nil)),
	})
}

type worldImageMarkIn struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Label string `json:"label,omitempty"`
}

func worldImageMarks(in []worldImageMarkIn) []mapsketch.Mark {
	marks := make([]mapsketch.Mark, 0, len(in))
	for k, m := range in {
		marks = append(marks, mapsketch.Mark{At: mapsketch.Point{X: m.X, Y: m.Y}, Label: m.Label, Colour: mapsketch.TagColour(k)})
	}
	return marks
}

func (h *handlers) worldImageInspect(w http.ResponseWriter, r *http.Request, p *project.Project) {
	projectID := p.ID()
	var body struct {
		ImageID    string             `json:"image_id"`
		MaxColours int                `json:"max_colours,omitempty"`
		Marks      []worldImageMarkIn `json:"marks,omitempty"`
		Crop       *mapsketch.Rect    `json:"zoom,omitempty"`
	}
	if !decodeWorldImageJSON(w, r, &body) {
		return
	}
	objects := worldImageStore(p)
	release, ok := worldImageGateAcquire(w)
	if !ok {
		return
	}
	defer release()
	meta, img, err := loadWorldImage(objects, projectID, body.ImageID)
	if err != nil {
		worldImageLoadFailed(w, projectID, body.ImageID, err)
		return
	}
	if body.MaxColours <= 0 || body.MaxColours > 40 {
		body.MaxColours = 16
	}
	picture := mapsketch.FromImage(img)
	colours := mapsketch.Palette(picture, body.MaxColours)
	marks := worldImageMarks(body.Marks)
	var preview image.Image
	if body.Crop != nil {

		zoomed := picture.Crop(*body.Crop)
		if zoomed.W < 8 || zoomed.H < 8 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("放大的框不在图上(图是 %d×%d)", meta.Width, meta.Height)})
			return
		}
		preview = mapsketch.ZoomPreview(picture, *body.Crop, worldImagePreviewSide, marks)
	} else {
		preview = mapsketch.GridPreview(picture, worldImagePreviewSide, marks)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"image": meta, "colours": colours,
		"legend":  worldImageLegend(),
		"preview": worldImagePreview(preview),
	})
}

func worldImageLegend() []map[string]string {
	out := make([]map[string]string, 0, len(mapgen.Legend))
	for _, entry := range mapgen.Legend {
		out = append(out, map[string]string{"terrain": entry.Key, "colour": mapsketch.RGB(entry.Color).Hex(), "name": entry.Name})
	}
	return out
}

type worldImageRuleIn struct {
	Colour    string           `json:"colour,omitempty"`
	At        *mapsketch.Point `json:"at,omitempty"`
	Terrain   string           `json:"terrain"`
	Tolerance float64          `json:"tolerance,omitempty"`
}

type worldImageAreaIn struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	W       int    `json:"width"`
	H       int    `json:"height"`
	Terrain string `json:"terrain"`
}

func (h *handlers) worldImageTerrain(w http.ResponseWriter, r *http.Request, p *project.Project) {
	projectID := p.ID()
	var body struct {
		ImageID    string             `json:"image_id"`
		SourceKind string             `json:"source_kind,omitempty"`
		Rules      []worldImageRuleIn `json:"rules,omitempty"`
		Areas      []worldImageAreaIn `json:"areas,omitempty"`
		Unmatched  string             `json:"unmatched,omitempty"`
		Dark       float64            `json:"dark,omitempty"`
		Smooth     int                `json:"smooth,omitempty"`
		MinPatch   float64            `json:"min_patch,omitempty"`
		Crop       *mapsketch.Rect    `json:"crop,omitempty"`
		Margin     float64            `json:"margin,omitempty"`
	}
	if !decodeWorldImageJSON(w, r, &body) {
		return
	}
	if len(body.Rules) > 64 || len(body.Areas) > 64 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "规则和区域各最多 64 条"})
		return
	}
	var err error
	body.SourceKind, body.Unmatched, err = normalizeWorldTerrainSource(body.SourceKind, body.Unmatched, body.Rules, body.Areas)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": err.Error(), "code": "invalid_political_terrain", "recoverable": true})
		return
	}
	objects := worldImageStore(p)
	release, ok := worldImageGateAcquire(w)
	if !ok {
		return
	}
	defer release()
	meta, img, err := loadWorldImage(objects, projectID, body.ImageID)
	if err != nil {
		worldImageLoadFailed(w, projectID, body.ImageID, err)
		return
	}
	if meta.Kind != "picture" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "地形草图要从图片(kind=picture)做,这张是 " + meta.Kind})
		return
	}
	picture := mapsketch.FromImage(img)
	dark := body.Dark
	if dark == 0 {
		dark = mapsketch.DefaultDark
	}
	opt := mapsketch.TerrainOptions{Unmatched: body.Unmatched, Dark: body.Dark, Smooth: body.Smooth, MinPatch: body.MinPatch, Margin: body.Margin}
	if body.Crop != nil {
		opt.Crop = *body.Crop
	}
	for k, rule := range body.Rules {
		var colour mapsketch.RGB
		switch {
		case rule.Colour != "":
			c, ok := mapsketch.ParseHex(rule.Colour)
			if !ok {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("第 %d 条规则的颜色 %q 要写成 #rrggbb", k+1, rule.Colour)})
				return
			}
			colour = c
		case rule.At != nil:
			c, ok := picture.SampleColour(*rule.At, dark)
			if !ok {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("第 %d 条规则的位置 (%d, %d) 不在图上(图是 %d×%d)", k+1, rule.At.X, rule.At.Y, picture.W, picture.H)})
				return
			}
			colour = c
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("第 %d 条规则要给 colour 或 at", k+1)})
			return
		}
		opt.Rules = append(opt.Rules, mapsketch.Rule{Colour: colour, Class: strings.ToLower(strings.TrimSpace(rule.Terrain)), Tolerance: rule.Tolerance})
	}
	for _, area := range body.Areas {
		opt.Areas = append(opt.Areas, mapsketch.Area{Box: mapsketch.Rect{X: area.X, Y: area.Y, W: area.W, H: area.H}, Class: strings.ToLower(strings.TrimSpace(area.Terrain))})
	}
	result, err := mapsketch.Terrain(picture, opt)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": err.Error()})
		return
	}
	placement := result.Placement
	out := &worldImageMeta{
		ID: newWorldImageID(), Kind: "terrain", Origin: "terrain", From: meta.ID,
		SourceWidth: picture.W, SourceHeight: picture.H, Placement: &placement, SourceKind: body.SourceKind,
	}
	if err := putWorldImage(objects, projectID, out, result.Sketch); err != nil {
		log.Printf("world image terrain %d: %v", projectID, err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "地形草图没能存下(图片存储出错)"})
		return
	}
	preview := worldImagePreview(mapsketch.TerrainPreview(result.Sketch, placement, picture.W, picture.H, worldImagePreviewSide, nil))
	writeJSON(w, http.StatusCreated, map[string]any{
		"image": out, "shares": result.Shares, "filled": result.Filled, "unmatched": result.Unmatched, "rules": result.Rules,
		"preview": preview, "preview_url": worldImageKeepPreview(objects, projectID, out.ID, preview),
		"source_kind": body.SourceKind, "unmatched_terrain": body.Unmatched,
	})
}

type worldImageSeedIn struct {
	X               *int             `json:"x,omitempty"`
	Y               *int             `json:"y,omitempty"`
	Tag             string           `json:"tag"`
	Colour          string           `json:"colour,omitempty"`
	ColourTolerance float64          `json:"colour_tolerance,omitempty"`
	Box             *mapsketch.Rect  `json:"box,omitempty"`
	At              *mapsketch.Point `json:"at,omitempty"`
}

func (h *handlers) worldImageCountries(w http.ResponseWriter, r *http.Request, p *project.Project) {
	projectID := p.ID()
	var body struct {
		ImageID        string             `json:"image_id"`
		TerrainImageID string             `json:"terrain_image_id"`
		Seeds          []worldImageSeedIn `json:"seeds"`
		Tolerance      float64            `json:"tolerance,omitempty"`
		Dark           float64            `json:"dark,omitempty"`
		SeaCost        float64            `json:"sea_cost,omitempty"`
	}
	if !decodeWorldImageJSON(w, r, &body) {
		return
	}
	if len(body.Seeds) == 0 || len(body.Seeds) > 256 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "seeds 要有 1 到 256 个位置"})
		return
	}
	tags := map[string]bool{}
	for _, seed := range body.Seeds {
		tag := strings.ToUpper(strings.TrimSpace(seed.Tag))
		if !landTagPattern.MatchString(tag) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("国家标签 %q 要是三个字母或数字、字母开头,比如 GER", seed.Tag)})
			return
		}
		tags[tag] = true
	}
	if len(tags) > worldImageMaxTags {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("最多 %d 个国家", worldImageMaxTags)})
		return
	}
	objects := worldImageStore(p)
	release, ok := worldImageGateAcquire(w)
	if !ok {
		return
	}
	defer release()
	terrainMeta, terrainImg, err := loadWorldImage(objects, projectID, body.TerrainImageID)
	if err != nil {
		worldImageLoadFailed(w, projectID, body.TerrainImageID, err)
		return
	}
	if terrainMeta.Kind != "terrain" || terrainMeta.Placement == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "terrain_image_id 要是地形草图(image-terrain 的结果)"})
		return
	}
	meta, img, err := loadWorldImage(objects, projectID, body.ImageID)
	if err != nil {
		worldImageLoadFailed(w, projectID, body.ImageID, err)
		return
	}
	if meta.Kind != "picture" || meta.Width != terrainMeta.SourceWidth || meta.Height != terrainMeta.SourceHeight {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf(
			"分国家的图要与地形草图出自同样大小的图(%d×%d,同一张图或它的重绘),这张是 %s %d×%d",
			terrainMeta.SourceWidth, terrainMeta.SourceHeight, meta.Kind, meta.Width, meta.Height)})
		return
	}
	placement := *terrainMeta.Placement
	sketch := mapsketch.FromImage(terrainImg)
	sea := mapsketch.RGB(mapgen.Legend[mapgen.Sea].Color)
	lake := mapsketch.RGB(mapgen.Legend[mapgen.Lake].Color)
	land := func(at mapsketch.Point) bool {
		c, ok := placement.CanvasOf(at)
		if !ok || c.X >= sketch.W || c.Y >= sketch.H {
			return false
		}
		colour := sketch.Pix[c.Y*sketch.W+c.X]
		return colour != sea && colour != lake
	}
	picture := mapsketch.FromImage(img)
	seeds, seedMatches, err := resolveWorldCountrySeeds(picture, body.Seeds, land)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": err.Error(), "code": "invalid_country_seed", "recoverable": true, "seed_matches": seedMatches})
		return
	}
	result, err := mapsketch.Regions(picture, mapsketch.RegionOptions{Seeds: seeds, Land: land, Tolerance: body.Tolerance, Dark: body.Dark, SeaCost: body.SeaCost})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": err.Error()})
		return
	}
	for i, seed := range result.Seeds {
		seedMatches[i]["actual_seed"] = seed.At
		if seed.At.X >= 0 && seed.At.Y >= 0 && seed.At.X < picture.W && seed.At.Y < picture.H {
			seedMatches[i]["source_colour"] = picture.Pix[seed.At.Y*picture.W+seed.At.X].Hex()
		}
		if warnings, ok := seedMatches[i]["warnings"].([]string); ok {
			for _, warning := range warnings {
				result.Warnings = append(result.Warnings, seed.Tag+": "+warning)
			}
		}
	}

	for _, tag := range result.Tags {
		if workbench.VanillaCountryTag(tag) {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s 是原版游戏的国家代码:用它的国家会沿用原版那个国家的国旗、领导人、国策和事件。新编的国家换一个原版没有的代码", tag))
		}
	}
	out := &worldImageMeta{
		ID: newWorldImageID(), Kind: "owners", Origin: "countries", From: terrainMeta.ID,
		SourceWidth: picture.W, SourceHeight: picture.H, Placement: &placement, Tags: result.Tags, SourceKind: terrainMeta.SourceKind,
	}
	if err := putWorldImage(objects, projectID, out, result.Owners(placement)); err != nil {
		log.Printf("world image countries %d: %v", projectID, err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "国家划分没能存下(图片存储出错)"})
		return
	}
	preview := worldImagePreview(result.Preview(worldImagePreviewSide, result.Seeds))
	writeJSON(w, http.StatusCreated, map[string]any{
		"image": out, "countries": result.Regions, "warnings": result.Warnings, "seed_matches": seedMatches,
		"preview": preview, "preview_url": worldImageKeepPreview(objects, projectID, out.ID, preview),
	})
}

func (h *handlers) worldImageWorldRoute(preview bool) func(http.ResponseWriter, *http.Request, *project.Project) {
	return func(w http.ResponseWriter, r *http.Request, p *project.Project) { h.worldImageWorld(w, r, p, preview) }
}

func (h *handlers) worldImageWorld(w http.ResponseWriter, r *http.Request, p *project.Project, preview bool) {
	projectID := p.ID()
	var body struct {
		TerrainImageID string   `json:"terrain_image_id"`
		OwnersImageID  string   `json:"owners_image_id,omitempty"`
		Tags           []string `json:"tags,omitempty"`
		Land           int      `json:"land,omitempty"`
		Sea            int      `json:"sea,omitempty"`
		Seed           int64    `json:"seed,omitempty"`
		PerState       int      `json:"per_state,omitempty"`
		Languages      []string `json:"languages,omitempty"`
	}
	if !decodeWorldImageJSON(w, r, &body) {
		return
	}
	objects := worldImageStore(p)
	terrainMeta, terrainImg, err := loadWorldImage(objects, projectID, body.TerrainImageID)
	if err != nil {
		worldImageLoadFailed(w, projectID, body.TerrainImageID, err)
		return
	}
	if terrainMeta.Kind != "terrain" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "terrain_image_id 要是地形草图(image-terrain 的结果)"})
		return
	}
	req := worldRequest{source: "sketch", sketch: terrainImg, seed: body.Seed, land: body.Land, sea: body.Sea, perState: body.PerState}
	if pl := terrainMeta.Placement; pl != nil && pl.CanvasW > 0 && pl.CanvasH > 0 {
		w, h := float64(pl.CanvasW), float64(pl.CanvasH)
		drawnW, drawnH := float64(pl.Crop.W)*pl.Scale, float64(pl.Crop.H)*pl.Scale
		req.focus = &[4]float64{float64(pl.OffsetX) / w, float64(pl.OffsetY) / h, (float64(pl.OffsetX) + drawnW) / w, (float64(pl.OffsetY) + drawnH) / h}
		placement := *pl
		req.picturePlacement = &placement
	}
	if body.OwnersImageID != "" {
		ownersMeta, ownersImg, err := loadWorldImage(objects, projectID, body.OwnersImageID)
		if err != nil {
			worldImageLoadFailed(w, projectID, body.OwnersImageID, err)
			return
		}
		if ownersMeta.Kind != "owners" || ownersMeta.From != terrainMeta.ID {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "owners_image_id 要是用这张地形草图分出的国家(image-countries 的结果)"})
			return
		}
		req.tags = ownersMeta.Tags
		bounds := ownersImg.Bounds()
		req.ownersW, req.ownersH = bounds.Dx(), bounds.Dy()
		owners := mapsketch.FromImage(ownersImg)
		req.owners = make([]uint8, len(owners.Pix))
		for i, c := range owners.Pix {
			if int(c[0]) <= len(req.tags) {
				req.owners[i] = c[0]
			}
		}
	} else {
		for _, tag := range body.Tags {
			tag = strings.ToUpper(strings.TrimSpace(tag))
			if !landTagPattern.MatchString(tag) {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "国家要是三个字母的标签,比如 GER"})
				return
			}
			req.tags = append(req.tags, tag)
		}
	}
	if len(req.tags) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "要给 owners_image_id(分好的国家)或 tags(从西到东分州)"})
		return
	}
	if len(req.tags) > worldImageMaxTags {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("最多 %d 个国家", worldImageMaxTags)})
		return
	}
	if req.land == 0 {
		req.land = 2000
	}
	if req.land < worldLandMin || req.land > worldLandMax {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("省份大小(land)要在 %d 到 %d 像素之间", worldLandMin, worldLandMax)})
		return
	}
	if req.sea <= 0 {
		req.sea = req.land * 5
	}
	if req.perState < 0 || req.perState > 30 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "每州省份数要在 1 到 30 之间"})
		return
	}
	for _, lang := range body.Languages {
		if lang = strings.ToLower(strings.TrimSpace(lang)); lang != "" {
			if _, _, ok := GUITextTarget(lang); !ok {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "不认识的语言 " + lang})
				return
			}
			req.languages = append(req.languages, lang)
		}
	}
	if len(req.languages) == 0 {
		req.languages = []string{"en"}
	}
	if preview {
		req.keepPreviews = func(previews map[string]string) map[string]string {
			urls := map[string]string{}
			for name, dataURL := range previews {
				if url := worldImageKeepPreview(objects, projectID, terrainMeta.ID, dataURL); url != "" {
					urls[name] = url
				}
			}
			return urls
		}
	}
	release, ok := acquireLandGate(w)
	if !ok {
		return
	}
	defer release()
	h.makeWorld(w, r, p, preview, req)
}

func worldPoliticalCloseUp(world *mapgen.World, owner map[int]string, tags []string, focus [4]float64) string {
	base := world.Base
	x0, y0 := int(focus[0]*float64(base.Width)), int(focus[1]*float64(base.Height))
	x1, y1 := int(math.Ceil(focus[2]*float64(base.Width))), int(math.Ceil(focus[3]*float64(base.Height)))
	x0, y0 = max(0, x0), max(0, y0)
	x1, y1 = min(base.Width, max(x0+1, x1)), min(base.Height, max(y0+1, y1))
	bw, bh := x1-x0, y1-y0
	w, h := bw, bh
	if long := max(w, h); long > worldImagePreviewSide {
		w, h = max(1, w*worldImagePreviewSide/long), max(1, h*worldImagePreviewSide/long)
	}
	stateOf := map[int32]int{}
	for k, st := range world.All.States {
		for _, id := range st.Provinces {
			stateOf[int32(id)] = k + 1
		}
	}
	lake := map[int32]bool{}
	for _, pr := range world.All.Provinces {
		if pr.Kind == "lake" {
			lake[int32(pr.ID)] = true
		}
	}
	tagIndex := map[string]int{}
	for k, tag := range tags {
		tagIndex[tag] = k
	}
	state := make([]int, w*h)
	for y := 0; y < h; y++ {
		sy := y0 + y*bh/h
		for x := 0; x < w; x++ {
			id := base.Province[sy*base.Width+x0+x*bw/w]
			if !lake[id] {
				state[y*w+x] = stateOf[id]
			}
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	type acc struct{ n, x, y int }
	centres := map[string]*acc{}
	stateCentres := map[int]*acc{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			o := i * 4
			out.Pix[o+3] = 255
			st := state[i]
			if st == 0 {
				out.Pix[o], out.Pix[o+1], out.Pix[o+2] = 40, 70, 120
				continue
			}
			tag := owner[st]
			c := mapsketch.RGB{150, 150, 150}
			if k, ok := tagIndex[tag]; ok {
				c = mapsketch.TagColour(k)
			}
			if a := centres[tag]; a != nil {
				a.n, a.x, a.y = a.n+1, a.x+x, a.y+y
			} else {
				centres[tag] = &acc{1, x, y}
			}
			if a := stateCentres[st]; a != nil {
				a.n, a.x, a.y = a.n+1, a.x+x, a.y+y
			} else {
				stateCentres[st] = &acc{1, x, y}
			}
			stateEdge, countryEdge := false, false
			for _, j := range []int{i + 1, i + w} {
				if (j == i+1 && x == w-1) || (j == i+w && y == h-1) {
					continue
				}
				if other := state[j]; other != st {
					stateEdge = true
					if other == 0 || owner[other] != tag {
						countryEdge = true
					}
				}
			}
			switch {
			case countryEdge:
				c = mapsketch.RGB{0, 0, 0}
			case stateEdge:
				c = mapsketch.RGB{c[0] * 3 / 5, c[1] * 3 / 5, c[2] * 3 / 5}
			}
			out.Pix[o], out.Pix[o+1], out.Pix[o+2] = c[0], c[1], c[2]
		}
	}

	for stateID, a := range stateCentres {
		label := strconv.Itoa(stateID)
		mapsketch.Label(out, a.x/a.n-len(label)*3, a.y/a.n-7, label)
	}
	for tag, a := range centres {
		if tag != "" {
			mapsketch.Label(out, a.x/a.n-10, a.y/a.n+2, tag)
		}
	}
	return worldImagePreview(out)
}
