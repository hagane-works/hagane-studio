package mapgen

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"math"

	"golang.org/x/image/bmp"
)

const latitudeBands = 16

const seaTerrain = "~sea"

func (b *Base) newPixels(add *Addition) map[int32]bool {
	out := map[int32]bool{}
	for _, pr := range add.Provinces {
		if pr.Kind == "land" {
			for _, p := range add.Pixels(pr.ID) {
				out[p] = true
			}
		}
	}
	return out
}

func (b *Base) terrainOf() map[int32]string {
	out := make(map[int32]string, len(b.Definitions))
	for _, d := range b.Definitions {
		out[int32(d.ID)] = d.Terrain
	}
	return out
}

func newTerrain(index uint8) string {
	for _, entry := range Legend {
		if entry.Terrain == index && entry.Class.Land() {
			return entry.Key
		}
	}
	return "plains"
}

var ErrTexture = errors.New("texture format not supported")

func colormapLayout(dds []byte) (w, h, pitch int, err error) {
	if len(dds) < 128 || string(dds[:4]) != "DDS " {
		return 0, 0, 0, ErrTexture
	}
	header := func(at int) uint32 { return binary.LittleEndian.Uint32(dds[at:]) }
	h, w, pitch, mips := int(header(12)), int(header(16)), int(header(20)), int(header(28))
	flags, bits := header(80), header(88)
	if flags&0x4 != 0 || bits != 32 || header(92) != 0xff0000 || header(96) != 0xff00 || header(100) != 0xff || mips > 1 || w <= 0 || h <= 0 {
		return 0, 0, 0, ErrTexture
	}
	if pitch == 0 {
		pitch = w * 4
	}
	if len(dds) < 128+pitch*h {
		return 0, 0, 0, ErrTexture
	}
	return w, h, pitch, nil
}

func (b *Base) texel(w, h, tx, ty int) int {
	return min(ty*b.Height/h, b.Height-1)*b.Width + min(tx*b.Width/w, b.Width-1)
}

type colourModel struct {
	byBand  map[string]*[latitudeBands]colourSum
	overall map[string]*colourSum
}

type colourSum struct{ r, g, bl, n float64 }

func (b *Base) learnColours(dds []byte, skip map[int32]bool) (colourModel, error) {
	w, h, pitch, err := colormapLayout(dds)
	if err != nil {
		return colourModel{}, err
	}
	terrains := b.terrainOf()
	kinds := b.kinds()
	model := colourModel{byBand: map[string]*[latitudeBands]colourSum{}, overall: map[string]*colourSum{}}
	add := func(t string, band int, i int) {
		if model.byBand[t] == nil {
			model.byBand[t] = &[latitudeBands]colourSum{}
			model.overall[t] = &colourSum{}
		}
		for _, s := range []*colourSum{&model.byBand[t][band], model.overall[t]} {
			s.bl += float64(dds[i])
			s.g += float64(dds[i+1])
			s.r += float64(dds[i+2])
			s.n++
		}
	}
	for cy := 0; cy < h; cy += 2 {
		band := cy * latitudeBands / h
		for cx := 0; cx < w; cx += 2 {
			p := b.texel(w, h, cx, cy)
			id := b.Province[p]
			if skip[int32(p)] || int(id) >= len(kinds) {
				continue
			}
			i := 128 + cy*pitch + cx*4
			switch kinds[id] {
			case kindLand:

				add(terrains[id], band, i)
				add("", band, i)
			case kindSea, kindLake:
				add(seaTerrain, band, i)
			}
		}
	}
	return model, nil
}

func (m colourModel) colour(t string, band int) (float64, float64, float64) {
	if bands := m.byBand[t]; bands != nil {
		for d := 0; d < latitudeBands; d++ {
			for _, k := range []int{band - d, band + d} {
				if k >= 0 && k < latitudeBands && bands[k].n >= 20 {
					s := bands[k]
					return s.r / s.n, s.g / s.n, s.bl / s.n
				}
			}
		}
		if s := m.overall[t]; s.n > 0 {
			return s.r / s.n, s.g / s.n, s.bl / s.n
		}
	}
	switch t {
	case "":
		return 68, 69, 42
	case seaTerrain:
		return 52, 74, 92
	}
	return m.colour("", band)
}

func (m colourModel) smooth(t string, cy, h int) (float64, float64, float64) {
	f := (float64(cy)+0.5)*latitudeBands/float64(h) - 0.5
	lo := int(math.Floor(f))
	w := f - float64(lo)
	r0, g0, b0 := m.colour(t, min(max(lo, 0), latitudeBands-1))
	r1, g1, b1 := m.colour(t, min(max(lo+1, 0), latitudeBands-1))
	return r0 + (r1-r0)*w, g0 + (g1-g0)*w, b0 + (b1-b0)*w
}

func paintColour(out []byte, i int, r, g, bl float64, seed int64, cx, cy int) {
	shade := 1 + 0.08*valueNoise(seed+41, float64(cx), float64(cy), 6)
	out[i] = uint8(math.Min(255, bl*shade))
	out[i+1] = uint8(math.Min(255, g*shade))
	out[i+2] = uint8(math.Min(255, r*shade))
	out[i+3] = 0
}

func (b *Base) PaintColormap(dds []byte, add *Addition, seed int64) ([]byte, error) {
	w, h, pitch, err := colormapLayout(dds)
	if err != nil {
		return nil, err
	}
	fresh := b.newPixels(add)
	model, err := b.learnColours(dds, fresh)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), dds...)
	painted := 0
	bounds := add.Changed
	terrains := b.terrainOf()
	for cy := bounds.Min.Y * h / b.Height; cy <= min(bounds.Max.Y*h/b.Height, h-1); cy++ {
		for cx := bounds.Min.X * w / b.Width; cx <= min(bounds.Max.X*w/b.Width, w-1); cx++ {
			p := b.texel(w, h, cx, cy)
			if !fresh[int32(p)] {
				continue
			}
			i := 128 + cy*pitch + cx*4
			if was, repainted := add.was[int(b.Province[p])]; repainted {

				nr, ng, nb := model.smooth(terrains[b.Province[p]], cy, h)
				or, og, ob := model.smooth(was, cy, h)
				shift := func(v byte, by float64) byte { return uint8(math.Max(0, math.Min(255, math.Round(float64(v)+by)))) }
				out[i] = shift(dds[i], nb-ob)
				out[i+1] = shift(dds[i+1], ng-og)
				out[i+2] = shift(dds[i+2], nr-or)
				painted++
				continue
			}
			r, g, bl := model.colour(newTerrain(b.Terrain[p]), cy*latitudeBands/h)
			paintColour(out, i, r, g, bl, seed, cx, cy)
			painted++
		}
	}
	if painted == 0 {
		return dds, nil
	}
	return out, nil
}

func (b *Base) PaintWorldColormap(source *Base, sourceDDS []byte, seed int64) ([]byte, error) {
	if _, _, _, err := colormapLayout(sourceDDS); err != nil {
		return nil, err
	}
	model, err := source.learnColours(sourceDDS, nil)
	if err != nil {
		return nil, err
	}
	w, h := max(1, b.Width/2), max(1, b.Height/2)
	out := make([]byte, 128+w*h*4)
	copy(out, sourceDDS[:128])
	binary.LittleEndian.PutUint32(out[12:], uint32(h))
	binary.LittleEndian.PutUint32(out[16:], uint32(w))
	binary.LittleEndian.PutUint32(out[20:], uint32(w*4))
	kinds := b.kinds()
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			p := b.texel(w, h, cx, cy)
			t := seaTerrain
			if id := b.Province[p]; int(id) < len(kinds) && kinds[id] == kindLand {
				t = newTerrain(b.Terrain[p])
			}
			r, g, bl := model.smooth(t, cy, h)
			paintColour(out, 128+cy*w*4+cx*4, r, g, bl, seed, cx, cy)
		}
	}
	return out, nil
}

func (b *Base) slopes(rgba *image.RGBA, area image.Rectangle) {
	w, h := rgba.Rect.Dx(), rgba.Rect.Dy()
	sx, sy := float64(b.Width)/float64(w), float64(b.Height)/float64(h)
	height := func(x, y int) float64 {
		x = min(max(x, 0), b.Width-1)
		y = min(max(y, 0), b.Height-1)
		return float64(b.Heights[y*b.Width+x])
	}
	const k = 2.8 / 127
	for ny := max(int(float64(area.Min.Y)/sy), 0); ny <= min(int(float64(area.Max.Y)/sy), h-1); ny++ {
		for nx := max(int(float64(area.Min.X)/sx), 0); nx <= min(int(float64(area.Max.X)/sx), w-1); nx++ {
			x, y := int(float64(nx)*sx), int(float64(ny)*sy)
			dx := height(x+2, y) - height(x-2, y)
			dy := height(x, y+2) - height(x, y-2)
			vx, vy, vz := -k*dx, k*dy, 1.0
			length := math.Sqrt(vx*vx + vy*vy + vz*vz)
			enc := func(v float64) uint8 { return uint8(math.Round(math.Min(255, math.Max(0, 128+127*v/length)))) }
			rgba.SetRGBA(nx, ny, color.RGBA{enc(vx), enc(vy), enc(vz), 255})
		}
	}
}

func (b *Base) PaintNormals(normal []byte, add *Addition) ([]byte, error) {
	img, err := bmp.Decode(bytes.NewReader(normal))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			rgba.SetRGBA(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), 255})
		}
	}
	b.slopes(rgba, add.Changed.Inset(-4))
	return EncodeBMP(rgba)
}

func (b *Base) WorldNormals() ([]byte, error) {
	rgba := image.NewRGBA(image.Rect(0, 0, max(1, b.Width/2), max(1, b.Height/2)))
	b.slopes(rgba, image.Rect(0, 0, b.Width, b.Height))
	return EncodeBMP(rgba)
}

type treeModel map[string]*[latitudeBands]treeTally

type treeTally struct {
	count map[uint8]int
	total int
}

func (b *Base) learnTrees(trees *image.Paletted, skip map[int32]bool) treeModel {
	w, h := trees.Rect.Dx(), trees.Rect.Dy()
	terrains := b.terrainOf()
	learn := treeModel{}
	for ty := 0; ty < h; ty++ {
		band := ty * latitudeBands / h
		for tx := 0; tx < w; tx++ {
			p := b.texel(w, h, tx, ty)
			t := terrains[b.Province[p]]
			if skip[int32(p)] || (t != "forest" && t != "jungle") {
				continue
			}
			if learn[t] == nil {
				learn[t] = &[latitudeBands]treeTally{}
			}
			cell := &learn[t][band]
			if cell.count == nil {
				cell.count = map[uint8]int{}
			}
			cell.count[trees.Pix[ty*trees.Stride+tx]]++
			cell.total++
		}
	}
	return learn
}

func (m treeModel) tree(t string, band int) (uint8, float64) {
	fallback := map[string]uint8{"forest": 6, "jungle": 28}[t]
	bands := m[t]
	if bands == nil {
		return fallback, 0.85
	}
	for d := 0; d < latitudeBands; d++ {
		for _, k := range []int{band - d, band + d} {
			if k < 0 || k >= latitudeBands || bands[k].total < 30 {
				continue
			}
			best, most, trees := uint8(0), 0, 0
			for index, n := range bands[k].count {
				if index == 0 {
					continue
				}
				trees += n
				if n > most || (n == most && index < best) {
					best, most = index, n
				}
			}
			if best != 0 {
				return best, float64(trees) / float64(bands[k].total)
			}
		}
	}
	return fallback, 0.85
}

func (b *Base) plant(model treeModel, p, tx, ty, h int, seed int64) uint8 {
	if t := newTerrain(b.Terrain[p]); t == "forest" || t == "jungle" {
		tree, density := model.tree(t, ty*latitudeBands/h)
		if hash01(seed+53, tx, ty) < density {
			return tree
		}
	}
	return 0
}

func (b *Base) PaintTrees(trees []byte, add *Addition, seed int64) ([]byte, error) {
	img, err := bmp.Decode(bytes.NewReader(trees))
	if err != nil {
		return nil, err
	}
	paletted, ok := img.(*image.Paletted)
	if !ok {
		return nil, ErrTexture
	}
	w, h := paletted.Rect.Dx(), paletted.Rect.Dy()
	fresh := b.newPixels(add)
	model := b.learnTrees(paletted, fresh)
	out := image.NewPaletted(paletted.Rect, paletted.Palette)
	copy(out.Pix, paletted.Pix)
	area := add.Changed
	for ty := max(area.Min.Y*h/b.Height, 0); ty <= min(area.Max.Y*h/b.Height, h-1); ty++ {
		for tx := max(area.Min.X*w/b.Width, 0); tx <= min(area.Max.X*w/b.Width, w-1); tx++ {
			p := b.texel(w, h, tx, ty)
			if !fresh[int32(p)] {
				continue
			}
			out.Pix[ty*out.Stride+tx] = b.plant(model, p, tx, ty, h, seed)
		}
	}
	return EncodeBMP(out)
}

func (b *Base) PaintWorldTrees(source *Base, sourceTrees []byte, seed int64) ([]byte, error) {
	img, err := bmp.Decode(bytes.NewReader(sourceTrees))
	if err != nil {
		return nil, err
	}
	paletted, ok := img.(*image.Paletted)
	if !ok {
		return nil, ErrTexture
	}
	model := source.learnTrees(paletted, nil)
	w := max(1, int(math.Round(float64(b.Width)*float64(paletted.Rect.Dx())/float64(source.Width))))
	h := max(1, int(math.Round(float64(b.Height)*float64(paletted.Rect.Dy())/float64(source.Height))))
	out := image.NewPaletted(image.Rect(0, 0, w, h), paletted.Palette)
	for ty := 0; ty < h; ty++ {
		for tx := 0; tx < w; tx++ {
			out.Pix[ty*out.Stride+tx] = b.plant(model, b.texel(w, h, tx, ty), tx, ty, h, seed)
		}
	}
	return EncodeBMP(out)
}
