package imagebatch

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"strings"

	xdraw "golang.org/x/image/draw"
	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/hoi4/importassets"
)

const MaxPixels = int64(16 * 1024 * 1024)

func BoundedPixelCount(width, height int) (int64, bool) {
	if width <= 0 || height <= 0 {
		return 0, false
	}
	w, h := int64(width), int64(height)
	if h > MaxPixels || w > MaxPixels/h {
		return 0, false
	}
	return w * h, true
}

func ConvertDDS(raw []byte, filename string) ([]byte, int, int, error) {
	return ConvertDDSWithSize(raw, filename, 0, 0)
}

func ConvertDDSWithSize(raw []byte, filename string, targetWidth, targetHeight int) ([]byte, int, int, error) {
	width, height, err := Dimensions(raw, filename)
	if err != nil {
		return nil, 0, 0, errors.New("Could not decode image (corrupt or unsupported format)")
	}
	if _, ok := BoundedPixelCount(width, height); !ok {
		return nil, 0, 0, errors.New("Could not decode image (dimensions exceed safe limit)")
	}
	img, err := importassets.DecodeImage(raw, filename)
	if err != nil {
		return nil, 0, 0, errors.New("Could not decode image (corrupt or unsupported format)")
	}
	bounds := img.Bounds()
	if bounds.Dx() != width || bounds.Dy() != height {
		return nil, 0, 0, errors.New("Could not decode image (dimensions changed during decode)")
	}
	if (targetWidth == 0) != (targetHeight == 0) || targetWidth < 0 || targetHeight < 0 {
		return nil, 0, 0, errors.New("Provide both target_width and target_height or neither")
	}
	if targetWidth > 0 {
		if _, ok := BoundedPixelCount(targetWidth, targetHeight); !ok {
			return nil, 0, 0, errors.New("target dimensions exceed safe limit")
		}
		width, height = targetWidth, targetHeight
	}
	nrgba := resizeNRGBA(img, width, height)
	dds := imgcodec.EncodeDXT5DDS(nrgba.Pix, width, height)
	if dds == nil {
		return nil, 0, 0, errors.New("DDS encode failed")
	}
	return dds, width, height, nil
}

func resizeNRGBA(source image.Image, width, height int) *image.NRGBA {
	bounds := source.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	if bounds.Dx() == width && bounds.Dy() == height {
		draw.Draw(dst, dst.Bounds(), source, bounds.Min, draw.Src)
	} else {
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), source, bounds, draw.Src, nil)
	}
	return dst
}

func Dimensions(raw []byte, filename string) (int, int, error) {
	lower := strings.ToLower(filename)
	switch {
	case bytes.HasPrefix(raw, []byte("DDS ")) || strings.HasSuffix(lower, ".dds"):
		if len(raw) < 20 || !bytes.HasPrefix(raw, []byte("DDS ")) {
			return 0, 0, errors.New("invalid DDS header")
		}
		return int(binary.LittleEndian.Uint32(raw[16:20])), int(binary.LittleEndian.Uint32(raw[12:16])), nil
	case strings.HasSuffix(lower, ".tga"):
		if len(raw) < 18 {
			return 0, 0, errors.New("invalid TGA header")
		}
		return int(binary.LittleEndian.Uint16(raw[12:14])), int(binary.LittleEndian.Uint16(raw[14:16])), nil
	default:
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		return config.Width, config.Height, err
	}
}
