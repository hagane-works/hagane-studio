package mapgen

import (
	"fmt"
	"sort"
)

func (b *Base) LoadRules(regions, states map[int][]int) []Finding {
	var f findings
	at := map[int32]int{}
	for p, id := range b.Province {
		if _, seen := at[id]; !seen {
			at[id] = p
		}
	}
	place := func(id int) (int, int) {
		if p, ok := at[int32(id)]; ok {
			return p % b.Width, p / b.Width
		}
		return -1, -1
	}
	regionOf := map[int]int{}
	for region, provinces := range regions {
		for _, p := range provinces {
			regionOf[p] = region
		}
	}
	stateOf := map[int]int{}
	for state, provinces := range states {
		for _, p := range provinces {
			stateOf[p] = state
		}
	}
	defs := append([]Definition(nil), b.Definitions...)
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	for _, d := range defs {
		if _, drawn := at[int32(d.ID)]; d.ID <= 0 || !drawn {
			continue
		}
		x, y := place(d.ID)
		if d.Kind == "land" && stateOf[d.ID] == 0 {
			f.add("province_no_state", "error", "有陆地省份不属于任何州,游戏读到这里会退出", x, y)
		}
		if regionOf[d.ID] == 0 {
			f.add("province_no_region", "error", "有省份不属于任何战略区,游戏读到这里会退出", x, y)
		}
	}

	ids := make([]int, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for k, id := range ids {
		if id != k+1 {
			f.add("state_ids_gap", "error", fmt.Sprintf("州的编号要从 1 连到 %d,中间缺了 #%d,游戏读到这里会退出", len(ids), k+1), -1, -1)
			break
		}
	}
	kinds := b.kinds()
	isLand := func(id int) bool { return id > 0 && id < len(kinds) && kinds[id] == kindLand }
	for _, id := range ids {
		landed := false
		for _, p := range states[id] {
			landed = landed || isLand(p)
		}
		if !landed && len(states[id]) > 0 {
			x, y := place(states[id][0])
			f.add("state_no_land", "error", "有州只剩湖泊、没有陆地,放不下机场和要塞,游戏开局会退出", x, y)
		}
	}
	for _, id := range ids {
		seen := map[int]bool{}
		for _, p := range states[id] {
			seen[regionOf[p]] = true
		}
		if len(seen) > 1 {
			x, y := place(states[id][0])
			f.add("state_two_regions", "error", "有州的省份分在了几个战略区,游戏读到这里会退出", x, y)
		}
	}

	isSea := func(id int) bool { return id > 0 && id < len(kinds) && kinds[id] == kindSea }
	var all, naval []int
	for region, provinces := range regions {
		all = append(all, provinces...)
		for _, p := range provinces {
			if isSea(p) {
				naval = append(naval, region)
				break
			}
		}
	}
	sort.Ints(naval)
	if len(naval) > 0 {
		borders := b.Borders(all)
		for _, region := range naval {
			if parts := seaParts(regions[region], borders, isSea); len(parts) > 1 {
				x, y := place(parts[1][0])
				f.add("sea_region_split", "error", "有海上战略区被切成了不相连的几块,游戏读到这里会退出", x, y)
			}
		}
	}
	return f.list()
}

func (w *World) LoadRules() []Finding {
	regions := map[int][]int{}
	for _, region := range w.Regions {
		regions[region.ID] = region.Provinces
	}
	states := map[int][]int{}
	for k, state := range w.All.States {
		states[k+1] = state.Provinces
	}
	return w.Base.LoadRules(regions, states)
}
