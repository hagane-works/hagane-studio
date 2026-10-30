package importassets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"haganeworks/internal/hoi4/imgcodec"
)

type Size struct{ Width, Height int }

const maxDecodedPixels int64 = 64 * 1024 * 1024

const (
	maxCoverSourcePixels = 4096 * 4096
	maxCoverDimension    = 1024
	maxCoverBytes        = 1_000_000
)

type Crop struct {
	X, Y, Width, Height float64
}

var CanonicalSizes = map[string]Size{
	"character_portrait": {156, 210}, "character_advisor": {65, 67},
	"character_trait": {23, 33}, "idea_picture": {65, 67},
	"focus_icon": {100, 88}, "event_picture": {210, 176},
	"decision_icon": {33, 32}, "tech_icon": {64, 64},
	"mio_icon": {65, 67}, "mio_trait_icon": {64, 64},
	"custom_trait_icon": {23, 33}, "country_flag": {82, 52}, "super_event_picture": {460, 310},

	"decision_category_icon": {52, 40},

	"ideology_icon": {66, 68},

	"intel_agency_logo": {117, 119},
}

var NewsEventPictureSize = Size{397, 153}

func NormalizePNG(raw []byte, filename, kind string) ([]byte, error) {
	return NormalizePNGWithCrop(raw, filename, kind, Crop{})
}

func NormalizePNGWithCrop(raw []byte, filename, kind string, crop Crop) ([]byte, error) {
	return normalizePNGWithCrop(raw, filename, kind, crop, nil)
}

func NormalizePNGWithCropToSize(raw []byte, filename, kind string, crop Crop, target Size) ([]byte, error) {
	return normalizePNGWithCrop(raw, filename, kind, crop, &target)
}

func normalizePNGWithCrop(raw []byte, filename, kind string, crop Crop, explicitTarget *Size) ([]byte, error) {
	img, err := DecodeImage(raw, filename)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	if crop.Width > 0 && crop.Height > 0 {
		left := max(bounds.Min.X, bounds.Min.X+int(crop.X))
		top := max(bounds.Min.Y, bounds.Min.Y+int(crop.Y))
		right := min(bounds.Max.X, bounds.Min.X+int(crop.X+crop.Width))
		bottom := min(bounds.Max.Y, bounds.Min.Y+int(crop.Y+crop.Height))
		if right <= left || bottom <= top {
			return nil, errors.New("crop rectangle is outside the image")
		}
		cropped := image.NewNRGBA(image.Rect(0, 0, right-left, bottom-top))
		draw.Draw(cropped, cropped.Bounds(), img, image.Pt(left, top), draw.Src)
		img = cropped
	}
	target, found := CanonicalSizes[kind]
	if explicitTarget != nil {
		target, found = *explicitTarget, true
	}
	if found {
		if target.Width <= 0 || target.Height <= 0 {
			return nil, errors.New("target image dimensions must be positive")
		}
		bounds = img.Bounds()
		if crop.Width > 0 && crop.Height > 0 {

			source := bounds
			if math.Abs(math.Log(float64(bounds.Dx())*float64(target.Height)/(float64(bounds.Dy())*float64(target.Width)))) > 0.02 {
				source = centerCoverCrop(bounds, target)
			}
			dst := image.NewNRGBA(image.Rect(0, 0, target.Width, target.Height))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, source, draw.Over, nil)
			img = dst
		} else {

			sourceRect := centerCoverCrop(bounds, target)
			dst := image.NewNRGBA(image.Rect(0, 0, target.Width, target.Height))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, sourceRect, draw.Over, nil)
			img = dst
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func centerCoverCrop(bounds image.Rectangle, target Size) image.Rectangle {
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || target.Width <= 0 || target.Height <= 0 {
		return bounds
	}
	if int64(width)*int64(target.Height) > int64(height)*int64(target.Width) {
		cropWidth := int(int64(height) * int64(target.Width) / int64(target.Height))
		cropWidth = max(1, min(width, cropWidth))
		left := bounds.Min.X + (width-cropWidth)/2
		return image.Rect(left, bounds.Min.Y, left+cropWidth, bounds.Max.Y)
	}
	if int64(width)*int64(target.Height) < int64(height)*int64(target.Width) {
		cropHeight := int(int64(width) * int64(target.Height) / int64(target.Width))
		cropHeight = max(1, min(height, cropHeight))
		top := bounds.Min.Y + (height-cropHeight)/2
		return image.Rect(bounds.Min.X, top, bounds.Max.X, top+cropHeight)
	}
	return bounds
}

func NormalizeProjectCover(raw []byte, filename string) ([]byte, error) {

	return normalizeSquarePNG(raw, filename, maxCoverDimension, maxCoverBytes, "cover")
}

func jpegEXIFOrientation(raw []byte) int {
	if len(raw) < 4 || raw[0] != 0xff || raw[1] != 0xd8 {
		return 1
	}
	for pos := 2; pos+4 <= len(raw); {
		if raw[pos] != 0xff {
			pos++
			continue
		}
		for pos < len(raw) && raw[pos] == 0xff {
			pos++
		}
		if pos >= len(raw) {
			break
		}
		marker := raw[pos]
		pos++
		if marker == 0xd9 || marker == 0xda {
			break
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if pos+2 > len(raw) {
			break
		}
		length := int(binary.BigEndian.Uint16(raw[pos : pos+2]))
		if length < 2 || pos+length > len(raw) {
			break
		}
		segment := raw[pos+2 : pos+length]
		pos += length
		if marker != 0xe1 || len(segment) < 14 || !bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			continue
		}
		tiff := segment[6:]
		var order binary.ByteOrder
		switch string(tiff[:2]) {
		case "II":
			order = binary.LittleEndian
		case "MM":
			order = binary.BigEndian
		default:
			continue
		}
		if order.Uint16(tiff[2:4]) != 42 {
			continue
		}
		offset := int(order.Uint32(tiff[4:8]))
		if offset < 0 || offset+2 > len(tiff) {
			continue
		}
		count := int(order.Uint16(tiff[offset : offset+2]))
		for i := 0; i < count; i++ {
			entry := offset + 2 + i*12
			if entry+12 > len(tiff) {
				break
			}
			if order.Uint16(tiff[entry:entry+2]) != 0x0112 || order.Uint16(tiff[entry+2:entry+4]) != 3 || order.Uint32(tiff[entry+4:entry+8]) != 1 {
				continue
			}
			orientation := int(order.Uint16(tiff[entry+8 : entry+10]))
			if orientation >= 1 && orientation <= 8 {
				return orientation
			}
		}
	}
	return 1
}

func applyEXIFOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x, y
			switch orientation {
			case 2:
				dx = w - 1 - x
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dy = h - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func DecodeImage(raw []byte, filename string) (image.Image, error) {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".dds") || bytes.HasPrefix(raw, []byte("DDS ")) {
		width, height, err := ImageDimensions(raw, filename)
		if err != nil {
			return nil, err
		}
		pix, width, height, ok := imgcodec.DecodeDDS(raw)
		if !ok {
			return nil, errors.New("unsupported DDS format")
		}
		return &image.NRGBA{Pix: pix, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}, nil
	}
	if strings.HasSuffix(lower, ".tga") {
		return decodeTGA(raw)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if !safeImageDimensions(config.Width, config.Height) {
		return nil, errors.New("image dimensions exceed the safe decode limit")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

func ImageDimensions(raw []byte, filename string) (int, int, error) {
	lower := strings.ToLower(filename)
	var width, height int
	switch {
	case strings.HasSuffix(lower, ".dds") || bytes.HasPrefix(raw, []byte("DDS ")):
		if len(raw) < 20 || !bytes.HasPrefix(raw, []byte("DDS ")) {
			return 0, 0, errors.New("invalid DDS header")
		}
		height = int(binary.LittleEndian.Uint32(raw[12:16]))
		width = int(binary.LittleEndian.Uint32(raw[16:20]))
	case strings.HasSuffix(lower, ".tga"):
		if len(raw) < 18 {
			return 0, 0, io.ErrUnexpectedEOF
		}
		width = int(binary.LittleEndian.Uint16(raw[12:14]))
		height = int(binary.LittleEndian.Uint16(raw[14:16]))
	default:
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return 0, 0, err
		}
		width, height = config.Width, config.Height
	}
	if !safeImageDimensions(width, height) {
		return 0, 0, errors.New("image dimensions exceed the safe decode limit")
	}
	return width, height, nil
}

func safeImageDimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= 32768 && height <= 32768 &&
		int64(width)*int64(height) <= maxDecodedPixels
}

func decodeTGA(raw []byte) (image.Image, error) {
	if len(raw) < 18 {
		return nil, io.ErrUnexpectedEOF
	}
	idLen, imageType := int(raw[0]), raw[2]
	width, height := int(binary.LittleEndian.Uint16(raw[12:14])), int(binary.LittleEndian.Uint16(raw[14:16]))
	bpp := int(raw[16])
	if !safeImageDimensions(width, height) || (bpp != 24 && bpp != 32) || (imageType != 2 && imageType != 10) {
		return nil, errors.New("unsupported TGA format")
	}
	dataStart, bytesPerPixel := 18+idLen, bpp/8
	if dataStart > len(raw) {
		return nil, io.ErrUnexpectedEOF
	}
	pixelCount := width * height
	if imageType == 2 {
		required := pixelCount * bytesPerPixel
		if required > len(raw)-dataStart {
			return nil, io.ErrUnexpectedEOF
		}
	} else {

		scanPos, remaining := dataStart, pixelCount
		for remaining > 0 {
			if scanPos >= len(raw) {
				return nil, io.ErrUnexpectedEOF
			}
			header := raw[scanPos]
			scanPos++
			count := int(header&0x7f) + 1
			if count > remaining {
				return nil, errors.New("TGA RLE packet exceeds image bounds")
			}
			packetPixels := count
			if header&0x80 != 0 {
				packetPixels = 1
			}
			packetBytes := packetPixels * bytesPerPixel
			if packetBytes > len(raw)-scanPos {
				return nil, io.ErrUnexpectedEOF
			}
			scanPos += packetBytes
			remaining -= count
		}
	}

	pos := dataStart
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	topOrigin := raw[17]&0x20 != 0
	rightOrigin := raw[17]&0x10 != 0
	hasAlpha := raw[17]&0x0f != 0
	written := 0
	writePixel := func(p color.NRGBA) {
		x, y := written%width, written/width
		if rightOrigin {
			x = width - 1 - x
		}
		if !topOrigin {
			y = height - 1 - y
		}
		img.SetNRGBA(x, y, p)
		written++
	}
	readPixel := func() (color.NRGBA, error) {
		if pos+bytesPerPixel > len(raw) {
			return color.NRGBA{}, io.ErrUnexpectedEOF
		}
		p := color.NRGBA{R: raw[pos+2], G: raw[pos+1], B: raw[pos], A: 255}
		if bytesPerPixel == 4 && hasAlpha {
			p.A = raw[pos+3]
		}
		pos += bytesPerPixel
		return p, nil
	}
	for written < width*height {
		if imageType == 2 {
			p, err := readPixel()
			if err != nil {
				return nil, err
			}
			writePixel(p)
			continue
		}
		if pos >= len(raw) {
			return nil, io.ErrUnexpectedEOF
		}
		header := raw[pos]
		pos++
		count := int(header&0x7f) + 1
		if header&0x80 != 0 {
			p, err := readPixel()
			if err != nil {
				return nil, err
			}
			for i := 0; i < count; i++ {
				writePixel(p)
			}
		} else {
			for i := 0; i < count; i++ {
				p, err := readPixel()
				if err != nil {
					return nil, err
				}
				writePixel(p)
			}
		}
	}
	return img, nil
}
