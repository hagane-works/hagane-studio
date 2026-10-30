package imagebatch

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type PlaceholderSpec struct {
	EntityID string
	Label    string
	BG       color.RGBA
}

type PlaceholderResult struct {
	EntityID string
	Filename string
	PNG      []byte
}

var (
	placeholderFontOnce sync.Once
	placeholderFontData []byte
)

func loadPlaceholderFont() []byte {
	placeholderFontOnce.Do(func() {
		paths := []string{strings.TrimSpace(os.Getenv("PLACEHOLDER_FONT_PATH"))}
		paths = append(paths,
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		)
		for _, path := range paths {
			if path == "" {
				continue
			}
			if raw, err := os.ReadFile(filepath.Clean(path)); err == nil {
				placeholderFontData = raw
				return
			}
		}
	})
	return placeholderFontData
}

func placeholderFace(size int) font.Face {
	raw := loadPlaceholderFont()
	if len(raw) == 0 {
		return basicfont.Face7x13
	}
	var parsed *sfnt.Font
	if collection, err := sfnt.ParseCollection(raw); err == nil {
		parsed, _ = collection.Font(0)
	} else if single, err := opentype.Parse(raw); err == nil {
		parsed = single
	}
	if parsed == nil {
		return basicfont.Face7x13
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return basicfont.Face7x13
	}
	return face
}

func splitMeasured(face font.Face, text string, maxWidth int) []string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "_", " "))
	if text == "" {
		return []string{"placeholder"}
	}
	words := strings.Fields(text)
	lines := make([]string, 0, 4)
	current := ""
	flush := func() {
		if current != "" && len(lines) < 4 {
			lines = append(lines, current)
		}
		current = ""
	}
	for _, word := range words {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if current == "" || font.MeasureString(face, candidate).Ceil() <= maxWidth {
			current = candidate
			continue
		}
		flush()
		current = word
	}
	flush()

	out := make([]string, 0, 4)
	for _, line := range lines {
		if font.MeasureString(face, line).Ceil() <= maxWidth {
			out = append(out, line)
			continue
		}
		part := ""
		for _, r := range line {
			candidate := part + string(r)
			if part != "" && font.MeasureString(face, candidate).Ceil() > maxWidth {
				out = append(out, part)
				part = string(r)
				if len(out) == 4 {
					return out
				}
			} else {
				part = candidate
			}
		}
		if part != "" && len(out) < 4 {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{text}
	}
	return out
}

func safePlaceholderFilename(entityID string) string {
	var b strings.Builder
	for _, r := range entityID {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	name := b.String()
	if name == "" {
		name = "placeholder"
	}
	return name + ".png"
}

func GeneratePlaceholders(specs []PlaceholderSpec, width, height int) ([]PlaceholderResult, error) {
	return GeneratePlaceholdersLimited(specs, width, height, 0)
}

func GeneratePlaceholdersLimited(specs []PlaceholderSpec, width, height int, maxOutputBytes int64) ([]PlaceholderResult, error) {
	if width <= 0 || height <= 0 || width > 4096 || height > 4096 {
		return nil, fmt.Errorf("width/height must be positive and at most 4096")
	}
	fontSize := min(width, height) / 8
	if fontSize < 10 {
		fontSize = 10
	}
	face := placeholderFace(fontSize)
	if closer, ok := face.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	results := make([]PlaceholderResult, 0, len(specs))
	var outputBytes int64
	for _, spec := range specs {
		entityID := strings.TrimSpace(spec.EntityID)
		if entityID == "" {
			entityID = "placeholder"
		}
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: spec.BG}, image.Point{}, draw.Src)
		label := spec.Label
		if strings.TrimSpace(label) == "" {
			label = entityID
		}
		margin := max(2, width/20)
		lines := splitMeasured(face, label, max(1, width-2*margin))
		metrics := face.Metrics()
		lineHeight := metrics.Height.Ceil() + 2
		y := (height-len(lines)*lineHeight)/2 + metrics.Ascent.Ceil()
		for _, line := range lines {
			lineWidth := font.MeasureString(face, line).Ceil()
			d := &font.Drawer{
				Dst:  canvas,
				Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 230}),
				Face: face,
				Dot:  fixed.P((width-lineWidth)/2, y),
			}
			d.DrawString(line)
			y += lineHeight
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, canvas); err != nil {
			return nil, fmt.Errorf("encode placeholder: %w", err)
		}
		if maxOutputBytes > 0 && (int64(encoded.Len()) > maxOutputBytes-outputBytes) {
			return nil, fmt.Errorf("placeholder batch output exceeds %d bytes; split into smaller batches", maxOutputBytes)
		}
		outputBytes += int64(encoded.Len())
		results = append(results, PlaceholderResult{EntityID: entityID, Filename: safePlaceholderFilename(entityID), PNG: encoded.Bytes()})
	}
	return results, nil
}
