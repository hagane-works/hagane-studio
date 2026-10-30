package exportengine

import (
	"fmt"
	"image"
	"image/color"
	"math"

	xdraw "golang.org/x/image/draw"

	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/hoi4/importassets"
)

const (
	agencyLogoFrameW = 117
	agencyLogoFrameH = 119
	agencyLogoFrames = 2

	agencyLogoMargin = 5
	agencyGlowRadius = 4
)

var agencyGlowColour = color.NRGBA{R: 236, G: 142, B: 42, A: 255}

func agencyLogoStrip(raw []byte, label string) (*image.NRGBA, error) {
	logo, err := importassets.DecodeImage(raw, label+".png")
	if err != nil {
		return nil, err
	}
	src := logo.Bounds()
	if src.Dx() <= 0 || src.Dy() <= 0 {
		return nil, fmt.Errorf("empty image")
	}
	frame := image.NewNRGBA(image.Rect(0, 0, agencyLogoFrameW, agencyLogoFrameH))
	inner := image.Rect(agencyLogoMargin, agencyLogoMargin, agencyLogoFrameW-agencyLogoMargin, agencyLogoFrameH-agencyLogoMargin)
	scale := math.Min(float64(inner.Dx())/float64(src.Dx()), float64(inner.Dy())/float64(src.Dy()))
	w := max(1, int(math.Round(float64(src.Dx())*scale)))
	h := max(1, int(math.Round(float64(src.Dy())*scale)))
	at := image.Rect(inner.Min.X+(inner.Dx()-w)/2, inner.Min.Y+(inner.Dy()-h)/2, 0, 0)
	at.Max = at.Min.Add(image.Pt(w, h))
	xdraw.CatmullRom.Scale(frame, at, logo, src, xdraw.Over, nil)

	strip := image.NewNRGBA(image.Rect(0, 0, agencyLogoFrameW*agencyLogoFrames, agencyLogoFrameH))
	xdraw.Draw(strip, frame.Bounds(), frame, image.Point{}, xdraw.Src)
	second := frame.Bounds().Add(image.Pt(agencyLogoFrameW, 0))
	xdraw.Draw(strip, second, agencyGlow(frame), image.Point{}, xdraw.Src)
	xdraw.Draw(strip, second, frame, image.Point{}, xdraw.Over)
	return strip, nil
}

func agencyGlow(frame *image.NRGBA) *image.NRGBA {
	b := frame.Bounds()
	w, h := b.Dx(), b.Dy()
	alpha := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			alpha[y*w+x] = float64(frame.NRGBAAt(x, y).A) / 255
		}
	}

	tmp := make([]float64, w*h)
	for pass := 0; pass < 3; pass++ {
		boxBlur(alpha, tmp, w, h, agencyGlowRadius, true)
		boxBlur(tmp, alpha, w, h, agencyGlowRadius, false)
	}
	glow := image.NewNRGBA(b)
	for i, a := range alpha {
		v := math.Min(1, a*2.2) * 0.9
		c := agencyGlowColour
		c.A = uint8(math.Round(v * 255))
		glow.SetNRGBA(i%w, i/w, c)
	}
	return glow
}

func boxBlur(src, dst []float64, w, h, r int, horizontal bool) {
	n := float64(2*r + 1)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sum := 0.0
			for k := -r; k <= r; k++ {
				xx, yy := x, y
				if horizontal {
					xx += k
				} else {
					yy += k
				}
				if xx >= 0 && xx < w && yy >= 0 && yy < h {
					sum += src[yy*w+xx]
				}
			}
			dst[y*w+x] = sum / n
		}
	}
}

func agencyLogoStripDDS(raw []byte, label string) ([]byte, error) {
	strip, err := agencyLogoStrip(raw, label)
	if err != nil {
		return nil, err
	}
	dds := imgcodec.EncodeDXT5DDS(strip.Pix, strip.Bounds().Dx(), strip.Bounds().Dy())
	if dds == nil {
		return nil, fmt.Errorf("DDS encode failed")
	}
	return dds, nil
}
