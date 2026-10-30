package mapgen

import (
	"encoding/binary"
	"math"
	"sort"

	"haganeworks/internal/hoi4/imgcodec"
)

var WaterColormapFiles = []string{
	"map/terrain/colormap_water_0.dds", "map/terrain/colormap_water_1.dds", "map/terrain/colormap_water_2.dds",
}

const FogTexture = "map/terrain/fow_rgb_waterspec_a.dds"

const coastBands = 8

type waterModel struct {
	sums [2][coastBands][latitudeBands]colourSum
	all  [2]colourSum
}

func (b *Base) surfaces(w, h int) []uint8 {
	kinds := b.kinds()
	out := make([]uint8, w*h)
	for ty := 0; ty < h; ty++ {
		for tx := 0; tx < w; tx++ {
			out[ty*w+tx] = 1
			if id := b.Province[b.texel(w, h, tx, ty)]; int(id) < len(kinds) && kinds[id] == kindLand {
				out[ty*w+tx] = 0
			}
		}
	}
	return out
}

func coastDistance(surface []uint8, w, h int) []float32 {
	const far = math.MaxInt32 / 2
	d := make([]int32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			d[i] = far
			for _, n := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				if n[0] >= 0 && n[0] < w && n[1] >= 0 && n[1] < h && surface[n[1]*w+n[0]] != surface[i] {
					d[i] = 3
					break
				}
			}
		}
	}
	relax := func(i, x, y, dx, dy int, cost int32) {
		if nx, ny := x+dx, y+dy; nx >= 0 && nx < w && ny >= 0 && ny < h {
			if v := d[ny*w+nx] + cost; v < d[i] {
				d[i] = v
			}
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			relax(i, x, y, -1, 0, 3)
			relax(i, x, y, 0, -1, 3)
			relax(i, x, y, -1, -1, 4)
			relax(i, x, y, 1, -1, 4)
		}
	}
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			i := y*w + x
			relax(i, x, y, 1, 0, 3)
			relax(i, x, y, 0, 1, 3)
			relax(i, x, y, 1, 1, 4)
			relax(i, x, y, -1, 1, 4)
		}
	}
	out := make([]float32, w*h)
	for i, v := range d {
		out[i] = float32(v) / 3
	}
	return out
}

func coastBand(distance float32) float64 {
	return math.Min(coastBands-1, math.Log2(math.Max(1, float64(distance))))
}

func (b *Base) learnWater(pix []byte, w, h int) waterModel {
	var model waterModel
	surface := b.surfaces(w, h)
	distance := coastDistance(surface, w, h)
	for ty := 0; ty < h; ty += 2 {
		band := ty * latitudeBands / h
		for tx := 0; tx < w; tx += 2 {
			t := ty*w + tx
			coast := int(coastBand(distance[t]))
			i := t * 4
			for _, s := range []*colourSum{&model.sums[surface[t]][coast][band], &model.all[surface[t]]} {
				s.r += float64(pix[i])
				s.g += float64(pix[i+1])
				s.bl += float64(pix[i+2])
				s.n++
			}
		}
	}
	return model
}

func (m *waterModel) colour(surface, coast, band int) (float64, float64, float64) {
	for l := 0; l < latitudeBands; l++ {
		for _, j := range []int{band - l, band + l} {
			if j < 0 || j >= latitudeBands {
				continue
			}
			for d := 0; d < coastBands; d++ {
				for _, k := range []int{coast - d, coast + d} {
					if k >= 0 && k < coastBands {
						if s := m.sums[surface][k][j]; s.n >= 20 {
							return s.r / s.n, s.g / s.n, s.bl / s.n
						}
					}
				}
			}
		}
	}
	if s := m.all[surface]; s.n > 0 {
		return s.r / s.n, s.g / s.n, s.bl / s.n
	}
	if surface == 0 {
		return 40, 70, 60
	}
	return 30, 50, 90
}

func (m *waterModel) smooth(surface int, coast float64, ty, h int) (float64, float64, float64) {
	fl := (float64(ty)+0.5)*latitudeBands/float64(h) - 0.5
	c0, l0 := int(math.Floor(coast)), int(math.Floor(fl))
	wc, wl := coast-float64(c0), fl-float64(l0)
	clamp := func(v, n int) int { return min(max(v, 0), n-1) }
	var r, g, bl float64
	for _, c := range []struct {
		c, l int
		w    float64
	}{
		{c0, l0, (1 - wc) * (1 - wl)}, {c0 + 1, l0, wc * (1 - wl)},
		{c0, l0 + 1, (1 - wc) * wl}, {c0 + 1, l0 + 1, wc * wl},
	} {
		cr, cg, cb := m.colour(surface, clamp(c.c, coastBands), clamp(c.l, latitudeBands))
		r, g, bl = r+cr*c.w, g+cg*c.w, bl+cb*c.w
	}
	return r, g, bl
}

func (b *Base) PaintWorldWater(source *Base, sourceDDS []byte, seed int64) ([][]byte, error) {
	pix, w, h, ok := imgcodec.DecodeDDS(sourceDDS)
	if !ok {
		return nil, ErrTexture
	}
	model := source.learnWater(pix, w, h)
	w, h = max(1, b.Width/2), max(1, b.Height/2)
	surface := b.surfaces(w, h)
	distance := coastDistance(surface, w, h)
	level := make([]byte, w*h*4)
	for ty := 0; ty < h; ty++ {
		for tx := 0; tx < w; tx++ {
			t := ty*w + tx
			r, g, bl := model.smooth(int(surface[t]), coastBand(distance[t]), ty, h)
			shade := 1 + 0.04*valueNoise(seed+53, float64(tx), float64(ty), 8)
			i := (ty*w + tx) * 4
			level[i] = uint8(math.Min(255, r*shade))
			level[i+1] = uint8(math.Min(255, g*shade))
			level[i+2] = uint8(math.Min(255, bl*shade))
			level[i+3] = 255
		}
	}
	out := make([][]byte, 0, len(WaterColormapFiles))
	for range WaterColormapFiles {
		dds := imgcodec.EncodeDXT5DDS(level, w, h)
		if dds == nil {
			return nil, ErrTexture
		}
		out = append(out, dds)
		level, w, h = halve(level, w, h)
	}
	return out, nil
}

func (b *Base) PaintWorldFog(source *Base, sourceDDS []byte, seed int64) ([]byte, error) {
	pix, w, h, ok := imgcodec.DecodeDDS(sourceDDS)
	if !ok {
		return nil, ErrTexture
	}
	paper, ok := source.learnFog(pix, w, h)
	if !ok {
		return nil, ErrTexture
	}
	kinds := b.kinds()
	ow, oh := max(1, b.Width/2), max(1, b.Height/2)
	level := make([]byte, ow*oh*4)
	sea := paper.sea
	dx := int(hash01(seed+71, 0, 0) * float64(2*sea.size))
	dy := int(hash01(seed+73, 0, 0) * float64(2*sea.size))
	for ty := 0; ty < oh; ty++ {
		for tx := 0; tx < ow; tx++ {
			i := (ty*ow + tx) * 4
			if id := b.Province[b.texel(ow, oh, tx, ty)]; int(id) < len(kinds) && kinds[id] == kindLand {
				x, y := float64(tx), float64(ty)

				grain := 4 * (0.5*valueNoise(seed+79, x, y, 2) + 0.3*valueNoise(seed+83, x, y, 7) + 0.2*valueNoise(seed+89, x, y, 23))
				for c := 0; c < 4; c++ {
					level[i+c] = uint8(math.Round(math.Min(255, math.Max(0, paper.mean[c]+paper.spread[c]*grain))))
				}
				continue
			}
			sx := sea.x + mirror(tx+dx, sea.size)
			sy := sea.y + mirror(ty+dy, sea.size)
			copy(level[i:i+4], pix[(sy*w+sx)*4:(sy*w+sx)*4+4])
		}
	}
	return encodeDXT5Mips(level, ow, oh), nil
}

type fogSwatch struct{ x, y, size int }

type fogPaper struct {
	sea          fogSwatch
	mean, spread [4]float64
}

func (b *Base) learnFog(pix []byte, w, h int) (fogPaper, bool) {
	const cell = 8
	cw, ch := w/cell, h/cell
	if cw == 0 || ch == 0 {
		return fogPaper{}, false
	}
	kinds := b.kinds()

	grid := make([]byte, cw*ch)
	type cellSums struct{ sum, sq [4]float64 }
	sums := make([]cellSums, cw*ch)
	spread := make([]float64, cw*ch)
	var landSpreads []float64
	for cy := 0; cy < ch; cy++ {
		for cx := 0; cx < cw; cx++ {
			var seen byte
			mixed := false
			var s cellSums
			for y := cy * cell; y < (cy+1)*cell && !mixed; y++ {
				for x := cx * cell; x < (cx+1)*cell; x++ {
					var k byte
					if id := b.Province[b.texel(w, h, x, y)]; int(id) < len(kinds) {
						switch kinds[id] {
						case kindLand:
							k = 1
						case kindSea:
							k = 2
						}
					}
					if k == 0 || (seen != 0 && k != seen) {
						mixed = true
						break
					}
					seen = k
					i := (y*w + x) * 4
					for c := 0; c < 4; c++ {
						v := float64(pix[i+c])
						s.sum[c] += v
						s.sq[c] += v * v
					}
				}
			}
			if mixed {
				continue
			}
			i := cy*cw + cx
			grid[i], sums[i] = seen, s
			n := float64(cell * cell)
			for c := 0; c < 4; c++ {
				spread[i] += s.sq[c]/n - (s.sum[c]/n)*(s.sum[c]/n)
			}
			if seen == 1 {
				landSpreads = append(landSpreads, spread[i])
			}
		}
	}
	if len(landSpreads) == 0 {
		return fogPaper{}, false
	}
	sort.Float64s(landSpreads)
	plain := landSpreads[len(landSpreads)/2]
	var paper fogPaper
	var total cellSums
	n := 0.0
	for i, k := range grid {
		if k == 1 && spread[i] <= plain {
			for c := 0; c < 4; c++ {
				total.sum[c] += sums[i].sum[c]
				total.sq[c] += sums[i].sq[c]
			}
			n += cell * cell
		}
	}
	for c := 0; c < 4; c++ {
		paper.mean[c] = total.sum[c] / n
		paper.spread[c] = math.Sqrt(math.Max(0, total.sq[c]/n-paper.mean[c]*paper.mean[c]))
	}
	run := make([]int, cw*ch)
	best, bx, by := 0, 0, 0
	for cy := 0; cy < ch; cy++ {
		for cx := 0; cx < cw; cx++ {
			i := cy*cw + cx
			if grid[i] != 2 {
				continue
			}
			run[i] = 1
			if cx > 0 && cy > 0 {
				run[i] = 1 + min(run[i-1], run[i-cw], run[i-cw-1])
			}
			if run[i] > best {
				best, bx, by = run[i], cx, cy
			}
		}
	}
	if best == 0 {
		return fogPaper{}, false
	}
	paper.sea = fogSwatch{x: (bx - best + 1) * cell, y: (by - best + 1) * cell, size: best * cell}
	return paper, true
}

func mirror(v, size int) int {
	v %= 2 * size
	if v < 0 {
		v += 2 * size
	}
	if v >= size {
		return 2*size - 1 - v
	}
	return v
}

func halve(pix []byte, w, h int) ([]byte, int, int) {
	nw, nh := max(1, w/2), max(1, h/2)
	out := make([]byte, nw*nh*4)
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			for c := 0; c < 4; c++ {
				sum := 0
				for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
					sx, sy := min(2*x+d[0], w-1), min(2*y+d[1], h-1)
					sum += int(pix[(sy*w+sx)*4+c])
				}
				out[(y*nw+x)*4+c] = uint8((sum + 2) / 4)
			}
		}
	}
	return out, nw, nh
}

func encodeDXT5Mips(pix []byte, w, h int) []byte {
	out := imgcodec.EncodeDXT5DDS(pix, w, h)
	mips := 1
	for lw, lh, level := w, h, pix; lw > 1 || lh > 1; {
		level, lw, lh = halve(level, lw, lh)
		out = append(out, imgcodec.EncodeDXT5Blocks(level, lw, lh)...)
		mips++
	}
	const ddsdMipmapCount, ddscapsComplex, ddscapsMipmap = 0x20000, 0x8, 0x400000
	binary.LittleEndian.PutUint32(out[8:], binary.LittleEndian.Uint32(out[8:])|ddsdMipmapCount)
	binary.LittleEndian.PutUint32(out[28:], uint32(mips))
	binary.LittleEndian.PutUint32(out[108:], binary.LittleEndian.Uint32(out[108:])|ddscapsComplex|ddscapsMipmap)
	return out
}
