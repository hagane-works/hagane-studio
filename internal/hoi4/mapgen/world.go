package mapgen

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"sort"
)

type WorldOptions struct {
	ProvincesPerState, StatesPerRegion, SeaProvincesPerRegion int
	Seed                                                      int64
}

func (o WorldOptions) withDefaults() WorldOptions {
	if o.ProvincesPerState <= 0 {
		o.ProvincesPerState = 4
	}
	if o.StatesPerRegion <= 0 {
		o.StatesPerRegion = 4
	}
	if o.SeaProvincesPerRegion <= 0 {
		o.SeaProvincesPerRegion = 6
	}
	return o
}

type Region struct {
	ID        int
	Provinces []int
	Sea       bool

	Weather string
}

type World struct {
	Base    *Base
	All     *Addition
	Regions []Region
}

func NewWorld(m *Map, terrainPalette, riverPalette color.Palette, options WorldOptions) (*World, error) {
	o := options.withDefaults()
	encode := func(name string, img image.Image) ([]byte, error) {
		out, err := EncodeBMP(img)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return out, nil
	}
	grey := image.NewPaletted(image.Rect(0, 0, m.Width, m.Height), GreyPalette())
	copy(grey.Pix, m.Heights)
	files := map[string][]byte{}
	for name, img := range map[string]image.Image{
		"provinces": m.ProvincesImage(), "heightmap": grey,
		"terrain": m.TerrainImage(terrainPalette), "rivers": m.RiversImage(riverPalette),
	} {
		data, err := encode(name, img)
		if err != nil {
			return nil, err
		}
		files[name] = data
	}
	base, err := LoadBase(files["provinces"], files["heightmap"], files["terrain"], files["rivers"], m.DefinitionCSV())
	if err != nil {
		return nil, err
	}
	return worldOf(base, m.Provinces, o), nil
}

func WorldOfBase(base *Base, options WorldOptions) *World {
	sumX := map[int32]float64{}
	sumY := map[int32]float64{}
	count := map[int32]int{}
	for p, id := range base.Province {
		sumX[id] += float64(p % base.Width)
		sumY[id] += float64(p / base.Width)
		count[id]++
	}
	var provinces []Province
	for _, d := range base.Definitions {
		n := count[int32(d.ID)]
		if n == 0 {
			continue
		}
		provinces = append(provinces, Province{
			ID: d.ID, Kind: d.Kind, Coastal: d.Coastal, Terrain: d.Terrain, Continent: d.Continent, Pixels: n,
			X: sumX[int32(d.ID)] / float64(n), Y: sumY[int32(d.ID)] / float64(n),
		})
	}
	return worldOf(base, provinces, options.withDefaults())
}

func worldOf(base *Base, provinces []Province, o WorldOptions) *World {
	pixels := map[int][]int32{}
	for p, id := range base.Province {
		if id > 0 {
			pixels[int(id)] = append(pixels[int(id)], int32(p))
		}
	}
	all := &Addition{
		Provinces: append([]Province(nil), provinces...),
		CutFrom:   map[int]int{},
		Covered:   map[int]int{},
		Changed:   image.Rect(0, 0, base.Width, base.Height),
		pixels:    pixels,
	}
	kindOf := map[int]string{}
	centre := map[int][2]float64{}
	var land, sea, lakes []int
	for _, pr := range all.Provinces {
		kindOf[pr.ID] = pr.Kind
		centre[pr.ID] = [2]float64{pr.X, pr.Y}
		switch pr.Kind {
		case "land":
			land = append(land, pr.ID)
		case "sea":
			sea = append(sea, pr.ID)
		default:
			lakes = append(lakes, pr.ID)
		}
	}
	touching := neighbourMap(base)
	among := func(kind string) map[int]map[int]bool {
		out := map[int]map[int]bool{}
		for id, others := range touching {
			if kindOf[id] != kind {
				continue
			}
			out[id] = map[int]bool{}
			for other := range others {
				if kindOf[other] == kind {
					out[id][other] = true
				}
			}
		}
		return out
	}
	all.States = groupNear(land, among("land"), centre, o.ProvincesPerState)

	stateOf := map[int]int{}
	stateCentre := map[int][2]float64{}
	for k, st := range all.States {
		var sx, sy float64
		for _, id := range st.Provinces {
			stateOf[id] = k + 1
			sx += centre[id][0]
			sy += centre[id][1]
		}
		stateCentre[k+1] = [2]float64{sx / float64(len(st.Provinces)), sy / float64(len(st.Provinces))}
	}
	stateNeighbours := map[int]map[int]bool{}
	stateIDs := make([]int, 0, len(all.States))
	for k := range all.States {
		stateIDs = append(stateIDs, k+1)
		stateNeighbours[k+1] = map[int]bool{}
	}
	for id, others := range touching {
		for other := range others {
			if a, b := stateOf[id], stateOf[other]; a > 0 && b > 0 && a != b {
				stateNeighbours[a][b] = true
			}
		}
	}
	world := &World{Base: base, All: all}
	regionOf := map[int]int{}
	for _, group := range groupNear(stateIDs, stateNeighbours, stateCentre, o.StatesPerRegion) {
		region := Region{ID: len(world.Regions) + 1}
		for _, state := range group.Provinces {
			region.Provinces = append(region.Provinces, all.States[state-1].Provinces...)
		}
		for _, id := range region.Provinces {
			regionOf[id] = region.ID
		}
		world.Regions = append(world.Regions, region)
	}
	for _, group := range groupNear(sea, among("sea"), centre, o.SeaProvincesPerRegion) {
		region := Region{ID: len(world.Regions) + 1, Provinces: group.Provinces, Sea: true}
		for _, id := range region.Provinces {
			regionOf[id] = region.ID
		}
		world.Regions = append(world.Regions, region)
	}

	for _, id := range lakes {
		best := 0
		var others []int
		for other := range touching[id] {
			others = append(others, other)
		}
		sort.Ints(others)
		for _, other := range others {
			if r := regionOf[other]; r > 0 && (best == 0 || (kindOf[other] == "land" && world.Regions[best-1].Sea)) {
				best = r
			}
		}
		if best == 0 {
			world.Regions = append(world.Regions, Region{ID: len(world.Regions) + 1, Provinces: []int{id}})
			best = len(world.Regions)
		}
		world.Regions[best-1].Provinces = append(world.Regions[best-1].Provinces, id)
		regionOf[id] = best
	}
	for k := range world.Regions {
		sort.Ints(world.Regions[k].Provinces)
	}
	return world
}

func neighbourMap(b *Base) map[int]map[int]bool {
	out := map[int]map[int]bool{}
	link := func(a, c int32) {
		if a == c || a <= 0 || c <= 0 {
			return
		}
		if out[int(a)] == nil {
			out[int(a)] = map[int]bool{}
		}
		if out[int(c)] == nil {
			out[int(c)] = map[int]bool{}
		}
		out[int(a)][int(c)] = true
		out[int(c)][int(a)] = true
	}
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			p := y*b.Width + x
			if x+1 < b.Width {
				link(b.Province[p], b.Province[p+1])
			}
			if y+1 < b.Height {
				link(b.Province[p], b.Province[p+b.Width])
			}
		}
	}
	return out
}

func (w *World) Positions(seed int64) Positions {
	ids := make([]int, len(w.All.States))
	for k := range ids {
		ids[k] = k + 1
	}
	out := w.Base.Positions(w.All, ids, 0, seed)
	out.Weather = nil
	rng := rand.New(rand.NewSource(seed + 211))
	for _, region := range w.Regions {
		var px []int32
		for _, id := range region.Provinces {
			px = append(px, w.All.Pixels(id)...)
		}
		if len(px) == 0 {
			continue
		}
		for _, size := range []string{"big", "small"} {
			x, y, z := w.Base.at(int(px[rng.Intn(len(px))]))
			out.Weather = append(out.Weather, fmt.Sprintf("%d;%.2f;%.2f;%.2f;%s", region.ID, x, y, z, size))
		}
	}
	return out
}
