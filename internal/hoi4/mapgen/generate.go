package mapgen

import (
	"errors"
	"image"
	"math/rand"
)

type Options struct {
	Width, Height int

	Seed int64

	LandProvincePixels, SeaProvincePixels int

	LakeMaxPixels int

	SpeckPixels       int
	SketchSpeckPixels int

	Rounds int

	MaxBoxWidth, MaxBoxHeight int
}

func (o Options) withDefaults() Options {
	if o.LandProvincePixels <= 0 {
		o.LandProvincePixels = 400
	}
	if o.SeaProvincePixels <= 0 {
		o.SeaProvincePixels = 2500
	}
	if o.LakeMaxPixels <= 0 {
		o.LakeMaxPixels = o.Width * o.Height / 400
	}
	if o.SpeckPixels <= 0 {
		o.SpeckPixels = 8
	}
	if o.SketchSpeckPixels <= 0 {
		o.SketchSpeckPixels = 3
	}
	if o.MaxBoxWidth <= 0 || o.MaxBoxWidth > MaxProvinceBoxWidth {
		o.MaxBoxWidth = MaxProvinceBoxWidth
	}
	if o.MaxBoxHeight <= 0 || o.MaxBoxHeight > MaxProvinceBoxHeight {
		o.MaxBoxHeight = MaxProvinceBoxHeight
	}
	if o.Rounds <= 0 {
		o.Rounds = 2
	}
	return o
}

type Province struct {
	ID        int
	Color     [3]uint8
	Kind      string
	Coastal   bool
	Terrain   string
	Continent int
	Pixels    int
	X, Y      float64
}

type Map struct {
	Width, Height int
	Classes       []Class

	Province  []int32
	Provinces []Province
	Heights   []uint8
}

var ErrSize = errors.New("map size out of range")

func Generate(sketch image.Image, options Options) (*Map, error) {
	if options.Width < 16 || options.Height < 16 || options.Width > 16384 || options.Height > 8192 {
		return nil, ErrSize
	}
	opt := options.withDefaults()
	src, sw, sh := ClassifySketch(sketch)
	removeSpecks(src, sw, sh, opt.SketchSpeckPixels)
	classes := Resample(src, sw, sh, opt.Width, opt.Height, opt.Seed)
	return FromClasses(classes, opt)
}

func FromClasses(classes []Class, options Options) (*Map, error) {
	opt := options.withDefaults()
	width, height := opt.Width, opt.Height
	if len(classes) != width*height {
		return nil, ErrSize
	}
	clean(classes, width, height, opt.SpeckPixels, opt.LakeMaxPixels)

	kind := func(i int) Class {
		if classes[i].Land() {
			return Plains
		}
		return classes[i]
	}
	area, sizes, _ := regions(width, height, func(a, b int) bool { return kind(a) == kind(b) })
	targets := make([]int, len(sizes))
	firstPixel := make([]int, len(sizes))
	for i := range firstPixel {
		firstPixel[i] = -1
	}
	for p, a := range area {
		if firstPixel[a] < 0 {
			firstPixel[a] = p
		}
	}
	for a := range sizes {
		if classes[firstPixel[a]].Land() {
			targets[a] = opt.LandProvincePixels
		} else {
			targets[a] = opt.SeaProvincePixels
		}
	}

	rng := rand.New(rand.NewSource(opt.Seed))
	seeds := placeSeeds(area, sizes, targets, rng)
	cell := grow(width, height, area, seeds)
	for round := 0; round < opt.Rounds; round++ {
		relax(width, cell, seeds)
		cell = grow(width, height, area, seeds)
	}

	count := make([]int, len(seeds))
	for _, c := range cell {
		count[c]++
	}
	absorb(width, height, area, cell, func(c int32) bool {
		return count[c] < targets[area[seeds[c]]]/6
	})
	fixCrossings(width, height, area, cell)
	joinPieces(width, height, area, cell)

	for round := 0; round < 8 && splitLarge(width, cell, opt.MaxBoxWidth, opt.MaxBoxHeight); round++ {
		joinPieces(width, height, area, cell)
	}
	fixCrossings(width, height, area, cell)

	m := &Map{Width: width, Height: height, Classes: classes, Province: cell}
	m.number(rng)
	m.Heights = heights(classes, width, height, opt.Seed)
	return m, nil
}

func (m *Map) number(rng *rand.Rand) {
	index := map[int32]int32{}
	for _, c := range m.Province {
		if _, ok := index[c]; !ok {
			index[c] = int32(len(index))
		}
	}
	provinces := make([]Province, len(index))
	sumX := make([]float64, len(index))
	sumY := make([]float64, len(index))
	classCount := make([][9]int, len(index))
	for p, c := range m.Province {
		k := index[c]
		m.Province[p] = k
		provinces[k].Pixels++
		sumX[k] += float64(p % m.Width)
		sumY[k] += float64(p / m.Width)
		classCount[k][m.Classes[p]]++
	}

	touchesSea := make([]bool, len(provinces))
	touchesLand := make([]bool, len(provinces))
	mark := func(p, q int) {
		a, b := m.Classes[p], m.Classes[q]
		if a.Land() && b == Sea {
			touchesSea[m.Province[p]] = true
			touchesLand[m.Province[q]] = true
		} else if b.Land() && a == Sea {
			touchesSea[m.Province[q]] = true
			touchesLand[m.Province[p]] = true
		}
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			p := y*m.Width + x
			if x+1 < m.Width {
				mark(p, p+1)
			}
			if y+1 < m.Height {
				mark(p, p+m.Width)
			}
		}
	}
	used := map[[3]uint8]bool{{0, 0, 0}: true}
	for k := range provinces {
		province := &provinces[k]
		province.ID = k + 1
		province.X = sumX[k] / float64(province.Pixels)
		province.Y = sumY[k] / float64(province.Pixels)
		major, most := Plains, -1
		for class, n := range classCount[k] {
			if n > most {
				major, most = Class(class), n
			}
		}
		switch {
		case major == Sea:
			province.Kind, province.Terrain, province.Coastal = "sea", "ocean", touchesLand[k]
		case major == Lake:
			province.Kind, province.Terrain = "lake", "lakes"
		default:
			province.Kind, province.Terrain, province.Coastal, province.Continent = "land", entryOf(major).Key, touchesSea[k], 1
		}
		for {
			color := [3]uint8{uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256))}
			if !used[color] {
				used[color] = true
				province.Color = color
				break
			}
		}
	}
	m.Provinces = provinces
}

func (m *Map) Summary() map[string]int {
	out := map[string]int{"land": 0, "sea": 0, "lake": 0}
	for _, province := range m.Provinces {
		out[province.Kind]++
	}
	return out
}
