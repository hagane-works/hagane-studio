package worldmap

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strconv"
	"time"

	_ "image/jpeg"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

const (
	sketchPreviewMaxPixels = 4096 * 1536
	sketchUploadMaxBytes   = 12 << 20

	sketchPreviewWidth = 1024
)

func worldMapLegend() []map[string]any {
	out := make([]map[string]any, 0, len(mapgen.Legend))
	for _, entry := range mapgen.Legend {
		if entry.Class == mapgen.Lake {
			continue
		}
		out = append(out, map[string]any{"key": entry.Key, "name": entry.Name, "color": entry.Color})
	}
	return out
}

func (h *handlers) sketchPreview(w http.ResponseWriter, r *http.Request, _ *project.Project) {
	r.Body = http.MaxBytesReader(w, r.Body, sketchUploadMaxBytes+(1<<20))
	if err := r.ParseMultipartForm(sketchUploadMaxBytes); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "草图要是 12 MB 以内的 PNG 或 JPEG"})
		return
	}
	file, _, err := r.FormFile("sketch")
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "没有收到草图"})
		return
	}
	sketch, _, err := image.Decode(file)
	_ = file.Close()
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "草图读不出来,要是 PNG 或 JPEG"})
		return
	}
	number := func(name string, fallback int) int {
		if v, err := strconv.Atoi(r.FormValue(name)); err == nil && v > 0 {
			return v
		}
		return fallback
	}
	options := mapgen.Options{
		Width:              number("width", 2816),
		Height:             number("height", 1024),
		LandProvincePixels: number("land", 0),
		SeaProvincePixels:  number("sea", 0),
	}
	if seed, err := strconv.ParseInt(r.FormValue("seed"), 10, 64); err == nil {
		options.Seed = seed
	}
	if options.Width*options.Height > sketchPreviewMaxPixels {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": fmt.Sprintf("试用最大 %d 像素(如 4096×1536);原版尺寸的地图在正式生成时做", sketchPreviewMaxPixels)})
		return
	}
	started := time.Now()
	world, err := mapgen.Generate(sketch, options)
	if errors.Is(err, mapgen.ErrSize) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "地图尺寸不在可生成的范围里"})
		return
	}
	if err != nil {
		refuseMap(w, http.StatusInternalServerError, problemNotSaved, "生成失败")
		return
	}
	took := time.Since(started).Seconds()

	findings := []mapgen.Finding{}
	for _, finding := range mapgen.Check(world.CheckInput()) {
		if finding.Code != "size_not_256" {
			findings = append(findings, finding)
		}
	}

	legendPalette := mapgen.GreyPalette()
	for _, entry := range mapgen.Legend {
		legendPalette[entry.Terrain] = color.RGBA{entry.Color[0], entry.Color[1], entry.Color[2], 255}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"width": world.Width, "height": world.Height, "seed": options.Seed, "seconds": took,
		"provinces": world.Summary(),
		"findings":  findings,
		"previews": map[string]string{
			"provinces": previewDataURL(world.ProvincesImage()),
			"terrain":   previewDataURL(world.TerrainImage(legendPalette)),
			"heightmap": previewDataURL(readableHeights(world.HeightImage())),
		},
	})
}

func readableHeights(heights *image.Gray) *image.Gray {
	out := image.NewGray(heights.Bounds())
	for i, v := range heights.Pix {
		stretched := (int(v) - 80) * 255 / 100
		out.Pix[i] = uint8(min(max(stretched, 0), 255))
	}
	return out
}

func previewDataURL(img image.Image) string {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > sketchPreviewWidth {
		height = height * sketchPreviewWidth / width
		width = sketchPreviewWidth
	}
	small := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sy := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			small.Set(x, y, img.At(bounds.Min.X+x*bounds.Dx()/width, sy))
		}
	}
	var out bytes.Buffer
	_ = (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&out, small)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
}
