package mapsketch

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type Mark struct {
	At     Point
	Label  string
	Colour RGB
}

func GridPreview(p *Picture, maxSide int, marks []Mark) *image.NRGBA {
	q := p.Scaled(maxSide)
	out := q.Image()
	for i := range out.Pix {
		if i%4 == 3 && out.Pix[i] == 0 {

			out.Pix[i-3], out.Pix[i-2], out.Pix[i-1], out.Pix[i] = 200, 200, 200, 255
		}
	}
	view := Rect{W: p.W, H: p.H}
	drawGrid(out, view)
	drawMarks(out, view, marks)
	return out
}

func ZoomPreview(p *Picture, box Rect, maxSide int, marks []Mark) *image.NRGBA {
	part := p.Crop(box)
	box = Rect{X: max(0, box.X), Y: max(0, box.Y), W: part.W, H: part.H}
	w, h := part.W, part.H
	long := max(w, h)
	w = max(1, int(math.Round(float64(w)*float64(maxSide)/float64(long))))
	h = max(1, int(math.Round(float64(h)*float64(maxSide)/float64(long))))
	out := part.Resized(w, h).Image()
	drawGrid(out, box)
	drawMarks(out, box, marks)
	return out
}

func gridStep(long int) int {
	for _, step := range []int{10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 2500, 5000} {
		if long/step <= 12 {
			return step
		}
	}
	return 10000
}

func drawGrid(out *image.NRGBA, view Rect) {
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	fx, fy := float64(w)/float64(view.W), float64(h)/float64(view.H)
	step := gridStep(max(view.W, view.H))
	dash := func(x, y int, along int) {

		c := color.NRGBA{0, 0, 0, 200}
		if (along/4)%2 == 1 {
			c = color.NRGBA{255, 255, 255, 200}
		}
		blend(out, x, y, c)
	}
	for v := (view.X/step + 1) * step; v < view.X+view.W; v += step {
		x := int(math.Round(float64(v-view.X) * fx))
		for y := 0; y < h; y++ {
			dash(x, y, y)
		}
		label(out, x+2, 1, strconv.Itoa(v))
	}
	for v := (view.Y/step + 1) * step; v < view.Y+view.H; v += step {
		y := int(math.Round(float64(v-view.Y) * fy))
		for x := 0; x < w; x++ {
			dash(x, y, x)
		}
		label(out, 1, y+2, strconv.Itoa(v))
	}
}

func drawMarks(out *image.NRGBA, view Rect, marks []Mark) {
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	for _, m := range marks {
		x := int((float64(m.At.X-view.X) + 0.5) * float64(w) / float64(view.W))
		y := int((float64(m.At.Y-view.Y) + 0.5) * float64(h) / float64(view.H))
		for dy := -5; dy <= 5; dy++ {
			for dx := -5; dx <= 5; dx++ {
				r := dx*dx + dy*dy
				switch {
				case r <= 12:
					blend(out, x+dx, y+dy, nrgba(m.Colour))
				case r <= 25:
					blend(out, x+dx, y+dy, color.NRGBA{0, 0, 0, 255})
				}
			}
		}
		if m.Label != "" {
			label(out, x+7, y-6, m.Label)
		}
	}
}

func Label(out *image.NRGBA, x, y int, text string) { label(out, x, y, text) }

func label(out *image.NRGBA, x, y int, text string) {
	face := basicfont.Face7x13
	width := font.MeasureString(face, text).Ceil()
	box := image.Rect(x, y, x+width+2, y+13).Intersect(out.Bounds())
	draw.Draw(out, box, image.NewUniform(color.NRGBA{255, 255, 255, 220}), image.Point{}, draw.Over)
	d := font.Drawer{Dst: out, Src: image.NewUniform(color.NRGBA{0, 0, 0, 255}), Face: face, Dot: fixed.P(x+1, y+10)}
	d.DrawString(text)
}

func blend(out *image.NRGBA, x, y int, c color.NRGBA) {
	if !(image.Point{x, y}.In(out.Bounds())) {
		return
	}
	o := out.PixOffset(x, y)
	a := int(c.A)
	out.Pix[o] = uint8((int(c.R)*a + int(out.Pix[o])*(255-a)) / 255)
	out.Pix[o+1] = uint8((int(c.G)*a + int(out.Pix[o+1])*(255-a)) / 255)
	out.Pix[o+2] = uint8((int(c.B)*a + int(out.Pix[o+2])*(255-a)) / 255)
	out.Pix[o+3] = 255
}

var tagColours = []RGB{
	{230, 25, 75}, {60, 180, 75}, {255, 225, 25}, {0, 130, 200}, {245, 130, 48},
	{145, 30, 180}, {70, 240, 240}, {240, 50, 230}, {210, 245, 60}, {250, 190, 212},
	{0, 128, 128}, {220, 190, 255}, {170, 110, 40}, {255, 250, 200}, {128, 0, 0},
	{170, 255, 195}, {128, 128, 0}, {255, 215, 180}, {0, 0, 128}, {128, 128, 128},
}

func TagColour(n int) RGB { return tagColours[n%len(tagColours)] }

func TerrainPreview(sketch *image.NRGBA, placement Placement, srcW, srcH, maxSide int, marks []Mark) *image.NRGBA {
	w, h := srcW, srcH
	if long := max(w, h); long > maxSide && maxSide > 0 {
		w = max(1, int(math.Round(float64(srcW)*float64(maxSide)/float64(long))))
		h = max(1, int(math.Round(float64(srcH)*float64(maxSide)/float64(long))))
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := out.PixOffset(x, y)
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = 90, 90, 90, 255
			s := Point{X: int((float64(x) + 0.5) * float64(srcW) / float64(w)), Y: int((float64(y) + 0.5) * float64(srcH) / float64(h))}
			if c, ok := placement.CanvasOf(s); ok && image.Pt(c.X, c.Y).In(sketch.Bounds()) {
				so := sketch.PixOffset(c.X, c.Y)
				copy(out.Pix[o:o+3], sketch.Pix[so:so+3])
			}
		}
	}
	view := Rect{W: srcW, H: srcH}
	drawGrid(out, view)
	drawMarks(out, view, marks)
	return out
}
