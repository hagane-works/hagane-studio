package imagebatch

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"haganeworks/internal/hoi4/importassets"
)

var overlayPositions = map[string]bool{
	"top-left": true, "top": true, "top-right": true,
	"left": true, "center": true, "right": true,
	"bottom-left": true, "bottom": true, "bottom-right": true,
}

const overlayPositionList = "['bottom', 'bottom-left', 'bottom-right', 'center', 'left', 'right', 'top', 'top-left', 'top-right']"

var overlayFontCandidates = []string{
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/opentype/noto/NotoSansCJKsc-Regular.otf",
	"/usr/share/fonts/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
	"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	"/usr/share/fonts/truetype/droid/DroidSansFallback.ttf",
	"C:/Windows/Fonts/msyh.ttc", "C:/Windows/Fonts/msyhbd.ttc",
	"C:/Windows/Fonts/simhei.ttf", "C:/Windows/Fonts/simsun.ttc",
	"/System/Library/Fonts/PingFang.ttc", "/Library/Fonts/Arial Unicode.ttf",
}

type OverlaySpec struct {
	Text        string
	Position    string
	Margin      int
	FontSize    *int
	FontPath    string
	Color       color.RGBA
	StrokeColor color.RGBA
	StrokeWidth int
}

type OverlayResult struct {
	PNG      []byte
	Width    int
	Height   int
	UsedFont string
}

func overlayFontPathAllowed(path string) bool {
	wanted, err := filepath.Abs(filepath.Clean(path))
	if err != nil || strings.TrimSpace(path) == "" {
		return false
	}
	allowed := append([]string{}, overlayFontCandidates...)
	if env := strings.TrimSpace(os.Getenv("HOI4_OVERLAY_FONT")); env != "" {
		allowed = append(allowed, env)
	}
	for _, candidate := range allowed {
		absolute, err := filepath.Abs(filepath.Clean(candidate))
		if err == nil && strings.EqualFold(wanted, absolute) {
			return true
		}
	}
	return false
}

func FindOverlayFont() string {
	if env := strings.TrimSpace(os.Getenv("HOI4_OVERLAY_FONT")); env != "" {
		if info, err := os.Stat(filepath.Clean(env)); err == nil && !info.IsDir() {
			return filepath.Clean(env)
		}
	}
	for _, candidate := range overlayFontCandidates {
		if info, err := os.Stat(filepath.Clean(candidate)); err == nil && !info.IsDir() {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func loadOverlayFace(requested string, size int) (font.Face, string) {
	path := ""
	if overlayFontPathAllowed(requested) {
		if info, err := os.Stat(filepath.Clean(requested)); err == nil && !info.IsDir() {
			path = filepath.Clean(requested)
		}
	}
	if path == "" {
		path = FindOverlayFont()
	}
	if path == "" {
		return basicfont.Face7x13, "go-basicfont"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return basicfont.Face7x13, "go-basicfont"
	}
	var parsed *sfnt.Font
	if collection, parseErr := sfnt.ParseCollection(raw); parseErr == nil {
		parsed, _ = collection.Font(0)
	} else if single, parseErr := opentype.Parse(raw); parseErr == nil {
		parsed = single
	}
	if parsed == nil {
		return basicfont.Face7x13, "go-basicfont"
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return basicfont.Face7x13, "go-basicfont"
	}
	return face, path
}

func overlayTextSize(face font.Face, text string, stroke int) (int, int) {
	width := font.MeasureString(face, text).Ceil() + stroke*2
	metrics := face.Metrics()
	height := metrics.Ascent.Ceil() + metrics.Descent.Ceil() + stroke*2
	return max(1, width), max(1, height)
}

func overlayXY(iw, ih, tw, th int, position string, margin int) (int, int) {
	x := (iw - tw) / 2
	y := (ih - th) / 2
	if strings.Contains(position, "left") {
		x = margin
	} else if strings.Contains(position, "right") {
		x = iw - tw - margin
	}
	if strings.Contains(position, "top") {
		y = margin
	} else if strings.Contains(position, "bottom") {
		y = ih - th - margin
	}
	return max(0, x), max(0, y)
}

func drawOverlayText(dst draw.Image, face font.Face, text string, x, baseline, stroke int, foreground, outline color.RGBA) {
	if stroke > 0 {
		for dy := -stroke; dy <= stroke; dy++ {
			for dx := -stroke; dx <= stroke; dx++ {
				if dx*dx+dy*dy > stroke*stroke || (dx == 0 && dy == 0) {
					continue
				}
				d := font.Drawer{Dst: dst, Src: image.NewUniform(outline), Face: face, Dot: fixed.P(x+dx, baseline+dy)}
				d.DrawString(text)
			}
		}
	}
	d := font.Drawer{Dst: dst, Src: image.NewUniform(foreground), Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(text)
}

func OverlayText(raw []byte, filename string, spec OverlaySpec) (OverlayResult, error) {
	text := strings.TrimSpace(spec.Text)
	if text == "" {
		return OverlayResult{}, errors.New("text must not be empty")
	}
	if utf8.RuneCountInString(text) > 4096 {
		return OverlayResult{}, errors.New("text is too long (max 4096 characters)")
	}
	position := strings.TrimSpace(spec.Position)
	if position == "" {
		position = "bottom"
	}
	if !overlayPositions[position] {
		return OverlayResult{}, fmt.Errorf("unknown position %q · expected one of %s", position, overlayPositionList)
	}
	width, height, err := Dimensions(raw, filename)
	if err != nil || width <= 0 || height <= 0 || int64(width)*int64(height) > MaxPixels {
		return OverlayResult{}, errors.New("could not decode source image bytes")
	}
	source, err := importassets.DecodeImage(raw, filename)
	if err != nil {
		return OverlayResult{}, errors.New("could not decode source image bytes")
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), source, source.Bounds().Min, draw.Src)
	margin := max(0, spec.Margin)
	stroke := max(0, spec.StrokeWidth)
	if margin > 32768 {
		return OverlayResult{}, errors.New("margin is too large")
	}
	if stroke > 64 {
		return OverlayResult{}, errors.New("stroke_width is too large (max 64)")
	}
	fontSize := max(12, min(width, height)*8/100)
	explicitSize := spec.FontSize != nil
	if explicitSize {
		if *spec.FontSize <= 0 || *spec.FontSize > 1024 {
			return OverlayResult{}, errors.New("font_size must be between 1 and 1024")
		}
		fontSize = *spec.FontSize
	}
	face, usedFont := loadOverlayFace(spec.FontPath, fontSize)
	tw, th := overlayTextSize(face, text, stroke)
	available := max(1, width-2*margin)
	for !explicitSize && tw > available && fontSize > 8 {
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		fontSize--
		face, usedFont = loadOverlayFace(spec.FontPath, fontSize)
		tw, th = overlayTextSize(face, text, stroke)
	}
	if closer, ok := face.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	x, y := overlayXY(width, height, tw, th, position, margin)
	baseline := y + stroke + face.Metrics().Ascent.Ceil()
	drawOverlayText(canvas, face, text, x+stroke, baseline, stroke, spec.Color, spec.StrokeColor)
	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return OverlayResult{}, errors.New("could not encode overlay image")
	}
	return OverlayResult{PNG: out.Bytes(), Width: width, Height: height, UsedFont: usedFont}, nil
}
