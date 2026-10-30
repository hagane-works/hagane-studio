package imagebatch

import (
	"bytes"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"testing"

	"haganeworks/internal/hoi4/imgcodec"
)

func TestBoundedPixelCountRejectsOverflowSizedHeaders(t *testing.T) {
	if pixels, ok := BoundedPixelCount(4096, 4096); !ok || pixels != 16*1024*1024 {
		t.Fatalf("boundary pixels=%d ok=%v", pixels, ok)
	}
	for _, dimensions := range [][2]int{{0, 1}, {1, 0}, {4097, 4096}, {int(^uint(0) >> 1), int(^uint(0) >> 1)}} {
		if pixels, ok := BoundedPixelCount(dimensions[0], dimensions[1]); ok {
			t.Fatalf("dimensions=%v unexpectedly accepted as %d pixels", dimensions, pixels)
		}
	}
}

func TestConvertDDSLosslessPNGPreservesPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 7, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 7; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 31), G: uint8(y * 47), B: uint8((x + y) * 19), A: uint8(40 + x*17 + y*11)})
		}
	}
	var source bytes.Buffer
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	got, width, height, err := ConvertDDS(source.Bytes(), "fixture.png")
	if err != nil {
		t.Fatal(err)
	}
	want := imgcodec.EncodeDXT5DDS(img.Pix, 7, 5)
	if width != 7 || height != 5 || !bytes.Equal(got, want) {
		t.Fatalf("dimensions=%dx%d bytes=%d want=%d", width, height, len(got), len(want))
	}
}

func TestResizeEquivalentQualityAgainstPillowLanczosGolden(t *testing.T) {

	source := image.NewNRGBA(image.Rect(0, 0, 11, 7))
	for y := 0; y < 7; y++ {
		for x := 0; x < 11; x++ {
			source.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*37 + y*11) % 256), G: uint8((x*17 + y*43) % 256),
				B: uint8((x*29 + y*23) % 256), A: uint8((40 + x*13 + y*17) % 256),
			})
		}
	}
	golden, err := hex.DecodeString("1c1c1736663e5a52c75b996f3c91e18c58c239a82e594353798c796fe3cae08c54d190a9666a35c23ece66739ec9b78faf68a4ab3a3f24c9873e6cf44f738d90b744e5aba9324dca3a6645e39a878e86")
	if err != nil {
		t.Fatal(err)
	}
	got := resizeNRGBA(source, 5, 4).Pix
	if len(got) != len(golden) {
		t.Fatalf("pixels=%d golden=%d", len(got), len(golden))
	}
	total, worst := 0, 0
	for i := range got {
		delta := int(got[i]) - int(golden[i])
		if delta < 0 {
			delta = -delta
		}
		total += delta
		worst = max(worst, delta)
	}
	mean := float64(total) / float64(len(got))
	if mean > 5.0 || worst > 24 {
		t.Fatalf("Catmull-Rom drifted from Pillow LANCZOS: mean_abs=%.3f worst=%d", mean, worst)
	}
}

func TestConvertDDSRejectsOversizedHeaderBeforeDecode(t *testing.T) {

	raw := make([]byte, 18)
	raw[2] = 2
	raw[12], raw[13] = 0x88, 0x13
	raw[14], raw[15] = 0xa0, 0x0f
	raw[16] = 32
	if _, _, _, err := ConvertDDS(raw, "huge.tga"); err == nil || err.Error() != "Could not decode image (dimensions exceed safe limit)" {
		t.Fatalf("err=%v", err)
	}
}
