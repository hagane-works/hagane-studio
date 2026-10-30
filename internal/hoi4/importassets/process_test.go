package importassets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"testing"
)

func TestNormalizePNGUsesCanonicalSize(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 200, G: 40, B: 10, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizePNG(encoded.Bytes(), "x.png", "focus_icon")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(got))
	if err != nil || decoded.Bounds().Dx() != 100 || decoded.Bounds().Dy() != 88 {
		t.Fatalf("bounds=%v err=%v", decoded.Bounds(), err)
	}
}

func TestNormalizePNGWithCropHonorsExplicitRegion(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				source.Set(x, y, color.NRGBA{R: 255, A: 255})
			} else {
				source.Set(x, y, color.NRGBA{B: 255, A: 255})
			}
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizePNGWithCrop(encoded.Bytes(), "x.png", "idea_picture", Crop{
		X: 10, Y: 0, Width: 10, Height: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 65 || decoded.Bounds().Dy() != 67 {
		t.Fatalf("size=%v want=65x67", decoded.Bounds())
	}
	r, _, b, _ := decoded.At(32, 33).RGBA()
	if b <= r {
		t.Fatalf("explicit crop lost selected blue half: r=%d b=%d", r, b)
	}
}

func TestNormalizePNGWithCropRejectsOutsideImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizePNGWithCrop(encoded.Bytes(), "x.png", "idea_picture", Crop{
		X: 10, Y: 10, Width: 2, Height: 2,
	}); err == nil {
		t.Fatal("expected outside-image crop error")
	}
}

func TestNormalizePNGWithCropToSizeDoesNotChangeImportCanonicalPolicy(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 36, 20))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	creator, err := NormalizePNGWithCropToSize(encoded.Bytes(), "x.png", "bookmark_picture", Crop{}, Size{Width: 180, Height: 104})
	if err != nil {
		t.Fatal(err)
	}
	creatorImage, err := png.Decode(bytes.NewReader(creator))
	if err != nil {
		t.Fatal(err)
	}
	if got := creatorImage.Bounds().Size(); got.X != 180 || got.Y != 104 {
		t.Fatalf("creator size=%dx%d want=180x104", got.X, got.Y)
	}
	imported, err := NormalizePNG(encoded.Bytes(), "x.png", "bookmark_picture")
	if err != nil {
		t.Fatal(err)
	}
	importedImage, err := png.Decode(bytes.NewReader(imported))
	if err != nil {
		t.Fatal(err)
	}
	if got := importedImage.Bounds().Size(); got.X != 36 || got.Y != 20 {
		t.Fatalf("import size=%dx%d want=36x20", got.X, got.Y)
	}
}

func TestNormalizePNGWithCropToSizeCropsExtremeAspectBeforeScaling(t *testing.T) {

	source := image.NewNRGBA(image.Rect(0, 0, 1, 32768))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizePNGWithCropToSize(encoded.Bytes(), "needle.png", "gui_sprite_override", Crop{}, Size{Width: 100, Height: 88})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if size := decoded.Bounds().Size(); size.X != 100 || size.Y != 88 {
		t.Fatalf("normalized size=%dx%d want=100x88", size.X, size.Y)
	}
	if crop := centerCoverCrop(source.Bounds(), Size{Width: 100, Height: 88}); crop.Dx() != 1 || crop.Dy() != 1 {
		t.Fatalf("source crop=%v want centered 1x1", crop)
	}
}

func TestDecodeImageRejectsUnsafeDimensionsBeforeAllocation(t *testing.T) {
	raw := make([]byte, 128)
	copy(raw, "DDS ")
	binary.LittleEndian.PutUint32(raw[12:16], 32768)
	binary.LittleEndian.PutUint32(raw[16:20], 32768)
	if _, err := DecodeImage(raw, "huge.dds"); err == nil || !strings.Contains(err.Error(), "safe decode limit") {
		t.Fatalf("expected dimension guard, got %v", err)
	}
}

func TestImageDimensionsReadsUploadHeadersWithoutPixelDecode(t *testing.T) {
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewNRGBA(image.Rect(0, 0, 13, 7))); err != nil {
		t.Fatal(err)
	}
	if width, height, err := ImageDimensions(pngBytes.Bytes(), "source.png"); err != nil || width != 13 || height != 7 {
		t.Fatalf("PNG dimensions = %dx%d, err %v", width, height, err)
	}

	dds := make([]byte, 128)
	copy(dds, "DDS ")
	binary.LittleEndian.PutUint32(dds[12:16], 9)
	binary.LittleEndian.PutUint32(dds[16:20], 21)
	if width, height, err := ImageDimensions(dds, "source.dds"); err != nil || width != 21 || height != 9 {
		t.Fatalf("DDS dimensions = %dx%d, err %v", width, height, err)
	}

	tga := make([]byte, 18)
	binary.LittleEndian.PutUint16(tga[12:14], 5)
	binary.LittleEndian.PutUint16(tga[14:16], 3)
	if width, height, err := ImageDimensions(tga, "source.tga"); err != nil || width != 5 || height != 3 {
		t.Fatalf("TGA dimensions = %dx%d, err %v", width, height, err)
	}
}

func TestDecodeTGAPreservesTopOriginPixelOrder(t *testing.T) {
	raw := make([]byte, 18, 18+4*3)
	raw[2] = 2
	binary.LittleEndian.PutUint16(raw[12:14], 2)
	binary.LittleEndian.PutUint16(raw[14:16], 2)
	raw[16] = 24
	raw[17] = 0x20
	raw = append(raw,
		0, 0, 255,
		0, 255, 0,
		255, 0, 0,
		255, 255, 255,
	)
	decoded, err := DecodeImage(raw, "pixels.tga")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		x, y int
		want color.NRGBA
	}{
		{x: 0, y: 0, want: color.NRGBA{R: 255, A: 255}},
		{x: 1, y: 0, want: color.NRGBA{G: 255, A: 255}},
		{x: 0, y: 1, want: color.NRGBA{B: 255, A: 255}},
		{x: 1, y: 1, want: color.NRGBA{R: 255, G: 255, B: 255, A: 255}},
	} {
		if got := color.NRGBAModel.Convert(decoded.At(tc.x, tc.y)).(color.NRGBA); got != tc.want {
			t.Errorf("pixel (%d,%d) = %#v, want %#v", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestDecodeTGAValidatesDataBeforeAllocation(t *testing.T) {
	t.Run("truncated image ID", func(t *testing.T) {
		raw := make([]byte, 18)
		raw[0] = 255
		raw[2] = 2
		binary.LittleEndian.PutUint16(raw[12:14], 1024)
		binary.LittleEndian.PutUint16(raw[14:16], 1024)
		raw[16] = 32
		if _, err := DecodeImage(raw, "truncated-id.tga"); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("expected truncated ID error, got %v", err)
		}
	})

	t.Run("truncated uncompressed payload", func(t *testing.T) {
		raw := make([]byte, 18)
		raw[2] = 2
		binary.LittleEndian.PutUint16(raw[12:14], 1024)
		binary.LittleEndian.PutUint16(raw[14:16], 1024)
		raw[16] = 32
		if _, err := DecodeImage(raw, "truncated-pixels.tga"); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("expected truncated pixel payload error, got %v", err)
		}
	})

	t.Run("truncated RLE payload", func(t *testing.T) {
		raw := make([]byte, 19)
		raw[2] = 10
		binary.LittleEndian.PutUint16(raw[12:14], 1)
		binary.LittleEndian.PutUint16(raw[14:16], 1)
		raw[16] = 24
		raw[18] = 0x80
		if _, err := DecodeImage(raw, "truncated-rle.tga"); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("expected truncated RLE payload error, got %v", err)
		}
	})
}

func TestDecodeTGARejectsRLEPacketBeyondImage(t *testing.T) {
	raw := make([]byte, 18, 22)
	raw[2] = 10
	binary.LittleEndian.PutUint16(raw[12:14], 2)
	binary.LittleEndian.PutUint16(raw[14:16], 1)
	raw[16] = 24
	raw = append(raw, 0x82, 0, 0, 255)
	if _, err := DecodeImage(raw, "overflow-rle.tga"); err == nil || !strings.Contains(err.Error(), "exceeds image bounds") {
		t.Fatalf("expected overflowing RLE packet error, got %v", err)
	}
}

func TestDecodeTGASupportsRightOrigin(t *testing.T) {
	raw := make([]byte, 18, 24)
	raw[2] = 2
	binary.LittleEndian.PutUint16(raw[12:14], 2)
	binary.LittleEndian.PutUint16(raw[14:16], 1)
	raw[16] = 24
	raw[17] = 0x30
	raw = append(raw,
		0, 0, 255,
		0, 255, 0,
	)
	decoded, err := DecodeImage(raw, "right-origin.tga")
	if err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(decoded.At(0, 0)).(color.NRGBA); got != (color.NRGBA{G: 255, A: 255}) {
		t.Fatalf("left pixel = %#v, want green", got)
	}
	if got := color.NRGBAModel.Convert(decoded.At(1, 0)).(color.NRGBA); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("right pixel = %#v, want red", got)
	}
}

func TestDecodeTGA32BitAlphaFollowsDescriptor(t *testing.T) {
	for _, tc := range []struct {
		name       string
		descriptor byte
		wantAlpha  uint8
	}{
		{name: "no attribute bits is opaque", descriptor: 0x20, wantAlpha: 255},
		{name: "eight attribute bits preserves alpha", descriptor: 0x28, wantAlpha: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := make([]byte, 18, 22)
			raw[2] = 2
			binary.LittleEndian.PutUint16(raw[12:14], 1)
			binary.LittleEndian.PutUint16(raw[14:16], 1)
			raw[16] = 32
			raw[17] = tc.descriptor
			raw = append(raw, 3, 2, 1, 0)
			decoded, err := DecodeImage(raw, "alpha.tga")
			if err != nil {
				t.Fatal(err)
			}
			got := color.NRGBAModel.Convert(decoded.At(0, 0)).(color.NRGBA)
			if got.A != tc.wantAlpha {
				t.Fatalf("alpha = %d, want %d (pixel %#v)", got.A, tc.wantAlpha, got)
			}
		})
	}
}

func TestNormalizeProjectCoverIsSquareAndBounded(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 1400, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 1400; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: byte(x), G: byte(y), B: byte(x + y), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeProjectCover(encoded.Bytes(), "cover.png")
	if err != nil {
		t.Fatal(err)
	}
	cover, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	if cover.Bounds().Dx() != 900 || cover.Bounds().Dy() != 900 || len(got) >= maxCoverBytes {
		t.Fatalf("cover bounds=%v bytes=%d", cover.Bounds(), len(got))
	}
}

func TestProjectCoverJPEGEXIFOrientation(t *testing.T) {

	tiff := make([]byte, 26)
	copy(tiff[:2], "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)
	binary.LittleEndian.PutUint16(tiff[8:10], 1)
	binary.LittleEndian.PutUint16(tiff[10:12], 0x0112)
	binary.LittleEndian.PutUint16(tiff[12:14], 3)
	binary.LittleEndian.PutUint32(tiff[14:18], 1)
	binary.LittleEndian.PutUint16(tiff[18:20], 6)
	segment := append([]byte("Exif\x00\x00"), tiff...)
	raw := []byte{0xff, 0xd8, 0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(raw[4:6], uint16(len(segment)+2))
	raw = append(raw, segment...)
	raw = append(raw, 0xff, 0xd9)
	if got := jpegEXIFOrientation(raw); got != 6 {
		t.Fatalf("orientation=%d want=6", got)
	}

	source := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	source.Set(0, 0, color.NRGBA{R: 255, A: 255})
	oriented := applyEXIFOrientation(source, 6)
	if oriented.Bounds().Dx() != 3 || oriented.Bounds().Dy() != 2 {
		t.Fatalf("bounds=%v want=3x2", oriented.Bounds())
	}
	r, _, _, _ := oriented.At(2, 0).RGBA()
	if r == 0 {
		t.Fatal("orientation=6 did not rotate top-left source pixel to top-right")
	}
}

func TestNormalizePNGWithCropOfAnotherShapeIsCroppedNotStretched(t *testing.T) {

	source := image.NewNRGBA(image.Rect(0, 0, 30, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 30; x++ {
			band := 1
			if x < 6 {
				band = 0
			} else if x >= 24 {
				band = 2
			}
			source.Set(x, y, []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}[band])
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizePNGWithCrop(encoded.Bytes(), "x.png", "focus_icon", Crop{Width: 30, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(got))
	if err != nil || decoded.Bounds().Dx() != 100 || decoded.Bounds().Dy() != 88 {
		t.Fatalf("decoded %v err=%v", decoded, err)
	}
	for _, x := range []int{3, 50, 96} {
		r, g, b, _ := decoded.At(x, 44).RGBA()
		if g < 0xc000 || r > 0x4000 || b > 0x4000 {
			t.Fatalf("x=%d is %d/%d/%d, want the green middle only", x, r>>8, g>>8, b>>8)
		}
	}
}
