package mapgen

import (
	"image"
	"math"
)

func ClassifySketch(sketch image.Image) (classes []Class, width, height int) {
	bounds := sketch.Bounds()
	width, height = bounds.Dx(), bounds.Dy()
	classes = make([]Class, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := sketch.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a < 0x8000 {
				classes[y*width+x] = Sea
				continue
			}
			classes[y*width+x] = nearestClass(uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}
	}
	return classes, width, height
}

func hash01(seed int64, x, y int) float64 {
	h := uint64(seed)*0x9E3779B97F4A7C15 ^ uint64(int64(x))*0xBF58476D1CE4E5B9 ^ uint64(int64(y))*0x94D049BB133111EB
	h ^= h >> 31
	h *= 0xD6E8FEB86659FD93
	h ^= h >> 32
	return float64(h>>11) / float64(1<<53)
}

func valueNoise(seed int64, x, y, cell float64) float64 {
	gx, gy := x/cell, y/cell
	x0, y0 := int(math.Floor(gx)), int(math.Floor(gy))
	fx, fy := gx-float64(x0), gy-float64(y0)
	sx, sy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	a := hash01(seed, x0, y0)
	b := hash01(seed, x0+1, y0)
	c := hash01(seed, x0, y0+1)
	d := hash01(seed, x0+1, y0+1)
	top := a + (b-a)*sx
	bottom := c + (d-c)*sx
	return (top+(bottom-top)*sy)*2 - 1
}

func fractalNoise(seed int64, x, y, cell float64) float64 {
	return 0.6*valueNoise(seed, x, y, cell) + 0.3*valueNoise(seed+1, x, y, cell/2) + 0.1*valueNoise(seed+2, x, y, cell/4)
}

func Resample(src []Class, sw, sh, width, height int, seed int64) []Class {
	out := make([]Class, width*height)
	scaleX := float64(sw) / float64(width)
	scaleY := float64(sh) / float64(height)
	cell := math.Max(6, math.Min(float64(width), float64(height))/48)
	at := func(x, y int) Class {
		if x < 0 {
			x = 0
		} else if x >= sw {
			x = sw - 1
		}
		if y < 0 {
			y = 0
		} else if y >= sh {
			y = sh - 1
		}
		return src[y*sw+x]
	}
	landAt := func(x, y int) float64 {
		if at(x, y).Land() {
			return 1
		}
		return 0
	}
	for y := 0; y < height; y++ {
		v := (float64(y)+0.5)*scaleY - 0.5
		y0 := int(math.Floor(v))
		fy := v - float64(y0)
		for x := 0; x < width; x++ {
			u := (float64(x)+0.5)*scaleX - 0.5
			x0 := int(math.Floor(u))
			fx := u - float64(x0)
			share := landAt(x0, y0)*(1-fx)*(1-fy) + landAt(x0+1, y0)*fx*(1-fy) + landAt(x0, y0+1)*(1-fx)*fy + landAt(x0+1, y0+1)*fx*fy
			if share > 0 && share < 1 {
				share += 0.22 * fractalNoise(seed, float64(x), float64(y), cell)
			}
			nearest := at(int(math.Round(u)), int(math.Round(v)))
			i := y*width + x
			switch {
			case share >= 0.5 && nearest.Land():
				out[i] = nearest
			case share >= 0.5:

				best, weight := Plains, -1.0
				for _, corner := range [4]struct {
					x, y int
					w    float64
				}{{x0, y0, (1 - fx) * (1 - fy)}, {x0 + 1, y0, fx * (1 - fy)}, {x0, y0 + 1, (1 - fx) * fy}, {x0 + 1, y0 + 1, fx * fy}} {
					if class := at(corner.x, corner.y); class.Land() && corner.w > weight {
						best, weight = class, corner.w
					}
				}
				out[i] = best
			case nearest == Lake:
				out[i] = Lake
			default:
				out[i] = Sea
			}
		}
	}
	return out
}
