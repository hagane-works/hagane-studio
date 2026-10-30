package mapgen

import (
	"fmt"
	"image"
	"slices"
	"sort"
	"strings"
)

type WorldPartState struct {
	ID        int
	Provinces []int
	Capital   int
}

func WorldOfParts(base *Base, states []WorldPartState, regions []Region) (*World, error) {
	pixels := map[int][]int32{}
	sumX, sumY := map[int]float64{}, map[int]float64{}
	for p, id := range base.Province {
		if id > 0 {
			pixels[int(id)] = append(pixels[int(id)], int32(p))
			sumX[int(id)] += float64(p % base.Width)
			sumY[int(id)] += float64(p / base.Width)
		}
	}
	kind := map[int]string{}
	var provinces []Province
	for _, d := range base.Definitions {
		kind[d.ID] = d.Kind
		n := len(pixels[d.ID])
		if n == 0 {
			continue
		}
		provinces = append(provinces, Province{
			ID: d.ID, Kind: d.Kind, Coastal: d.Coastal, Terrain: d.Terrain, Continent: d.Continent, Pixels: n,
			X: sumX[d.ID] / float64(n), Y: sumY[d.ID] / float64(n),
		})
	}
	sorted := append([]WorldPartState(nil), states...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for k, st := range sorted {
		if st.ID != k+1 {
			return nil, &AddError{
				Message: fmt.Sprintf("州的编号要从 1 连到 %d,中间缺了 #%d", len(sorted), k+1),
				Code:    "mod_state_gap", Values: map[string]any{"count": len(sorted), "missing": k + 1},
			}
		}
	}
	all := &Addition{
		Provinces: provinces, CutFrom: map[int]int{}, Covered: map[int]int{},
		Changed: image.Rect(0, 0, base.Width, base.Height), pixels: pixels,
	}
	var landless, badCapital []string
	for _, st := range sorted {
		var land []int
		for _, p := range st.Provinces {
			if kind[p] == "land" && len(pixels[p]) > 0 {
				land = append(land, p)
			}
		}
		if len(land) == 0 {
			landless = append(landless, fmt.Sprintf("#%d", st.ID))
			continue
		}
		sort.Ints(land)
		capital := st.Capital
		if capital == 0 {
			capital = land[0]
		}
		if kind[capital] != "land" || len(pixels[capital]) == 0 || !slices.Contains(land, capital) {
			badCapital = append(badCapital, fmt.Sprintf("#%d", st.ID))
			continue
		}
		all.States = append(all.States, NewState{Provinces: land, Capital: capital})
	}
	if len(landless) > 0 {
		return nil, &AddError{
			Message: "这些州没有陆地(只有湖泊、海,或省份图上没有它的省份),游戏开局会退出:" + strings.Join(landless, "、"),
			Code:    "mod_state_no_land", Values: map[string]any{"states": strings.Join(landless, ", ")},
		}
	}
	if len(badCapital) > 0 {
		return nil, &AddError{
			Message: "这些州的首府不是它自己的陆地省份:" + strings.Join(badCapital, "、"),
			Code:    "mod_state_capital", Values: map[string]any{"states": strings.Join(badCapital, ", ")},
		}
	}
	world := &World{Base: base, All: all}
	for _, region := range regions {
		r := Region{ID: region.ID, Provinces: append([]int(nil), region.Provinces...), Weather: region.Weather}
		sort.Ints(r.Provinces)
		for _, p := range r.Provinces {
			r.Sea = r.Sea || kind[p] == "sea"
		}
		world.Regions = append(world.Regions, r)
	}
	sort.Slice(world.Regions, func(i, j int) bool { return world.Regions[i].ID < world.Regions[j].ID })
	return world, nil
}
