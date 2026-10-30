package mapsketch

import "math"

func PadForModel(p *Picture) (*Picture, Rect, string) {
	ratio, size := 1.5, "1536x1024"
	if p.H > p.W {
		ratio, size = 1/1.5, "1024x1536"
	}
	w := max(p.W, int(math.Ceil(float64(p.H)*ratio)))
	h := max(p.H, int(math.Ceil(float64(p.W)/ratio)))
	frame := Rect{X: (w - p.W) / 2, Y: (h - p.H) / 2, W: p.W, H: p.H}
	fill := edgeColour(p)
	out := &Picture{W: w, H: h, Pix: make([]RGB, w*h)}
	for i := range out.Pix {
		out.Pix[i] = fill
	}
	for y := 0; y < p.H; y++ {
		for x := 0; x < p.W; x++ {
			i := y*p.W + x
			if !p.clear(i) {
				out.Pix[(y+frame.Y)*w+x+frame.X] = p.Pix[i]
			}
		}
	}
	return out, frame, size
}

func Unpad(result, padded *Picture, frame Rect, w, h int) *Picture {
	fx, fy := float64(result.W)/float64(padded.W), float64(result.H)/float64(padded.H)
	box := Rect{
		X: int(math.Round(float64(frame.X) * fx)), Y: int(math.Round(float64(frame.Y) * fy)),
		W: max(1, int(math.Round(float64(frame.W)*fx))), H: max(1, int(math.Round(float64(frame.H)*fy))),
	}
	return result.Crop(box).Resized(w, h)
}

func edgeColour(p *Picture) RGB {
	counts := map[int][2]int{}
	best, bestN := RGB{40, 80, 170}, 0
	add := func(i int) {
		if p.clear(i) {
			return
		}
		c := p.Pix[i]
		k := int(c[0]>>3)<<10 | int(c[1]>>3)<<5 | int(c[2]>>3)
		v := counts[k]
		v[0]++
		v[1] = i
		counts[k] = v
		if v[0] > bestN {
			best, bestN = c, v[0]
		}
	}
	for x := 0; x < p.W; x++ {
		add(x)
		add((p.H-1)*p.W + x)
	}
	for y := 0; y < p.H; y++ {
		add(y * p.W)
		add(y*p.W + p.W - 1)
	}
	return best
}
