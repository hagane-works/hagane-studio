package mapsketch

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"width"`
	H int `json:"height"`
}

func (r Rect) empty() bool { return r.W <= 0 || r.H <= 0 }

type RGB [3]uint8

func (c RGB) Hex() string { return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2]) }

func ParseHex(s string) (RGB, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return RGB{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return RGB{}, false
	}
	return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, true
}

func Distance(a, b RGB) float64 {
	dr := float64(a[0]) - float64(b[0])
	dg := float64(a[1]) - float64(b[1])
	db := float64(a[2]) - float64(b[2])
	return math.Sqrt((2*dr*dr + 4*dg*dg + 3*db*db) / 3)
}

func luminance(c RGB) float64 {
	return 0.299*float64(c[0]) + 0.587*float64(c[1]) + 0.114*float64(c[2])
}

type Picture struct {
	W, H int
	Pix  []RGB

	Clear []bool
}

func FromImage(img image.Image) *Picture {
	b := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	p := &Picture{W: b.Dx(), H: b.Dy(), Pix: make([]RGB, b.Dx()*b.Dy())}
	for i := range p.Pix {
		o := i * 4
		p.Pix[i] = RGB{rgba.Pix[o], rgba.Pix[o+1], rgba.Pix[o+2]}
		if rgba.Pix[o+3] < 128 {
			if p.Clear == nil {
				p.Clear = make([]bool, len(p.Pix))
			}
			p.Clear[i] = true
		}
	}
	return p
}

func (p *Picture) clear(i int) bool { return p.Clear != nil && p.Clear[i] }

func (p *Picture) Image() *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, p.W, p.H))
	for i, c := range p.Pix {
		o := i * 4
		out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = c[0], c[1], c[2], 255
		if p.clear(i) {
			out.Pix[o+3] = 0
		}
	}
	return out
}

func (p *Picture) Crop(r Rect) *Picture {
	box := image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H).Intersect(image.Rect(0, 0, p.W, p.H))
	if box.Empty() {
		return &Picture{}
	}
	out := &Picture{W: box.Dx(), H: box.Dy(), Pix: make([]RGB, box.Dx()*box.Dy())}
	if p.Clear != nil {
		out.Clear = make([]bool, len(out.Pix))
	}
	for y := 0; y < out.H; y++ {
		from := (box.Min.Y+y)*p.W + box.Min.X
		copy(out.Pix[y*out.W:(y+1)*out.W], p.Pix[from:from+out.W])
		if p.Clear != nil {
			copy(out.Clear[y*out.W:(y+1)*out.W], p.Clear[from:from+out.W])
		}
	}
	return out
}

func (p *Picture) Scaled(maxSide int) *Picture {
	long := max(p.W, p.H)
	if maxSide <= 0 || long <= maxSide {
		return p
	}
	w := max(1, int(math.Round(float64(p.W)*float64(maxSide)/float64(long))))
	h := max(1, int(math.Round(float64(p.H)*float64(maxSide)/float64(long))))
	return p.Resized(w, h)
}

func (p *Picture) Resized(w, h int) *Picture {
	out := &Picture{W: w, H: h, Pix: make([]RGB, w*h)}
	if p.Clear != nil {
		out.Clear = make([]bool, w*h)
	}
	for y := 0; y < h; y++ {
		y0 := y * p.H / h
		y1 := max(y0+1, (y+1)*p.H/h)
		for x := 0; x < w; x++ {
			x0 := x * p.W / w
			x1 := max(x0+1, (x+1)*p.W/w)
			var r, g, b, n, clear int
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					i := sy*p.W + sx
					if p.clear(i) {
						clear++
						continue
					}
					c := p.Pix[i]
					r, g, b, n = r+int(c[0]), g+int(c[1]), b+int(c[2]), n+1
				}
			}
			o := y*w + x
			if n > 0 {
				out.Pix[o] = RGB{uint8(r / n), uint8(g / n), uint8(b / n)}
			}
			if out.Clear != nil && clear > n {
				out.Clear[o] = true
			}
		}
	}
	return out
}

func (p *Picture) SampleColour(at Point, dark float64) (RGB, bool) {
	if at.X < 0 || at.Y < 0 || at.X >= p.W || at.Y >= p.H {
		return RGB{}, false
	}
	var all, light []RGB
	for y := at.Y - 3; y <= at.Y+3; y++ {
		for x := at.X - 3; x <= at.X+3; x++ {
			if x < 0 || y < 0 || x >= p.W || y >= p.H || p.clear(y*p.W+x) {
				continue
			}
			c := p.Pix[y*p.W+x]
			all = append(all, c)
			if dark <= 0 || luminance(c) >= dark {
				light = append(light, c)
			}
		}
	}
	if len(light) > 0 {
		return median(light), true
	}
	if len(all) > 0 {
		return median(all), true
	}
	return p.Pix[at.Y*p.W+at.X], true
}

func median(colours []RGB) RGB {
	var out RGB
	values := make([]int, len(colours))
	for ch := 0; ch < 3; ch++ {
		for i, c := range colours {
			values[i] = int(c[ch])
		}

		for i := 1; i < len(values); i++ {
			for j := i; j > 0 && values[j] < values[j-1]; j-- {
				values[j], values[j-1] = values[j-1], values[j]
			}
		}
		out[ch] = uint8(values[len(values)/2])
	}
	return out
}

func nrgba(c RGB) color.NRGBA { return color.NRGBA{c[0], c[1], c[2], 255} }
