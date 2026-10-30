package imagebatch

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func overlayFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 40, A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestOverlayTextContractAndDeterminism(t *testing.T) {
	raw := overlayFixture(t)
	spec := OverlaySpec{Text: "GER", Position: "bottom-right", Margin: 3, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}, StrokeColor: color.RGBA{A: 255}, StrokeWidth: 1}
	first, err := OverlayText(raw, "source.png", spec)
	if err != nil {
		t.Fatal(err)
	}
	second, err := OverlayText(raw, "source.png", spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.Width != 96 || first.Height != 64 || !bytes.Equal(first.PNG, second.PNG) || bytes.Equal(first.PNG, raw) {
		t.Fatalf("result=%dx%d deterministic=%v changed=%v", first.Width, first.Height, bytes.Equal(first.PNG, second.PNG), !bytes.Equal(first.PNG, raw))
	}
}

func TestOverlayTextValidationAndFontPathSecurity(t *testing.T) {
	raw := overlayFixture(t)
	for _, tc := range []struct {
		name string
		spec OverlaySpec
	}{
		{"empty", OverlaySpec{Position: "bottom"}},
		{"position", OverlaySpec{Text: "x", Position: "outside"}},
		{"stroke bound", OverlaySpec{Text: "x", Position: "bottom", StrokeWidth: 65}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := OverlayText(raw, "source.png", tc.spec); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if overlayFontPathAllowed(t.TempDir() + "/private.ttf") {
		t.Fatal("arbitrary request font path must not be readable")
	}
}
