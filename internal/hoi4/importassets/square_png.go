package importassets

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	xdraw "golang.org/x/image/draw"
)

func normalizeSquarePNG(raw []byte, filename string, maxDimension, maxBytes int, what string) ([]byte, error) {
	img, err := DecodeImage(raw, filename)
	if err != nil {
		return nil, errors.New(what + " could not be decoded")
	}
	img = applyEXIFOrientation(img, jpegEXIFOrientation(raw))
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx()*b.Dy() > maxCoverSourcePixels {
		return nil, errors.New(what + " dimensions are too large")
	}
	cropSide := min(b.Dx(), b.Dy())
	left, top := b.Min.X+(b.Dx()-cropSide)/2, b.Min.Y+(b.Dy()-cropSide)/2
	cropped := image.NewNRGBA(image.Rect(0, 0, cropSide, cropSide))
	draw.Draw(cropped, cropped.Bounds(), img, image.Pt(left, top), draw.Src)
	for side := min(cropSide, maxDimension); side >= 1; side = max(1, side/2) {
		var normalized image.Image = cropped
		if side != cropSide {
			dst := image.NewNRGBA(image.Rect(0, 0, side, side))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Src, nil)
			normalized = dst
		}
		var out bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&out, normalized); err != nil {
			return nil, errors.New(what + " could not be encoded")
		}
		if out.Len() < maxBytes {
			return out.Bytes(), nil
		}
		if side == 1 {
			break
		}
	}
	return nil, fmt.Errorf("%s could not be reduced below %d bytes", what, maxBytes)
}
