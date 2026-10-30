package mapgen

import (
	"fmt"
	"sort"
)

type RegionMove struct {
	From, To  int
	Provinces []int
}

func (b *Base) MendSeaRegions(regions map[int][]int, stateOf map[int]int) ([]RegionMove, error) {
	kinds := b.kinds()
	isSea := func(id int) bool { return id > 0 && id < len(kinds) && kinds[id] == kindSea }
	regionOf := map[int]int{}
	var all []int
	for id, provinces := range regions {
		for _, p := range provinces {
			regionOf[p] = id
			all = append(all, p)
		}
	}
	naval := func(provinces []int) bool {
		for _, p := range provinces {
			if isSea(p) {
				return true
			}
		}
		return false
	}
	ids := make([]int, 0, len(regions))
	for id, provinces := range regions {
		if naval(provinces) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	sort.Ints(ids)
	borders := b.Borders(all)
	members := map[int][]int{}
	for p, state := range stateOf {
		if state > 0 {
			members[state] = append(members[state], p)
		}
	}

	var moves []RegionMove
	for _, region := range ids {
		parts := seaParts(regions[region], borders, isSea)
		if len(parts) < 2 {
			continue
		}
		for _, part := range parts[1:] {
			in := map[int]bool{}
			for _, p := range part {
				in[p] = true
			}
			for _, p := range part {
				state := stateOf[p]
				if state <= 0 {
					continue
				}
				for _, q := range members[state] {
					if !in[q] {
						return nil, &AddError{Code: "sea_region_state", Values: map[string]any{"region": region, "id": part[0], "state": state}, Provinces: part,
							Message: fmt.Sprintf("海上战略区 #%d 被切成了几块,切出去的那块(省份 #%d 等)带着州 #%d 的一部分,不能单独并进别的战略区", region, part[0], state)}
					}
				}
			}
			edges := map[int]int{}
			for _, p := range part {
				if !isSea(p) {
					continue
				}
				for other, n := range borders[p] {
					if to := regionOf[other]; to > 0 && to != region && !in[other] && isSea(other) {
						edges[to] += n
					}
				}
			}
			best, bestEdges := 0, 0
			for to, n := range edges {
				if n > bestEdges || (n == bestEdges && to < best) {
					best, bestEdges = to, n
				}
			}
			if best == 0 {
				return nil, &AddError{Code: "sea_region_alone", Values: map[string]any{"region": region, "id": part[0], "count": len(part)}, Provinces: part,
					Message: fmt.Sprintf("海上战略区 #%d 被切成了几块,切出去的那块(省份 #%d 等 %d 个)旁边没有别的海上战略区可以并进去", region, part[0], len(part))}
			}
			moves = append(moves, RegionMove{From: region, To: best, Provinces: part})
		}
	}
	if len(moves) == 0 {
		return nil, nil
	}

	after := map[int][]int{}
	touched := map[int]bool{}
	moved := map[int]int{}
	for _, move := range moves {
		touched[move.From], touched[move.To] = true, true
		for _, p := range move.Provinces {
			moved[p] = move.To
		}
	}
	for id := range touched {
		for _, p := range regions[id] {
			if to, ok := moved[p]; !ok || to == id {
				after[id] = append(after[id], p)
			}
		}
	}
	for _, move := range moves {
		after[move.To] = append(after[move.To], move.Provinces...)
	}
	checked := make([]int, 0, len(touched))
	for id := range touched {
		checked = append(checked, id)
	}
	sort.Ints(checked)
	for _, id := range checked {
		if parts := seaParts(after[id], borders, isSea); len(parts) > 1 {
			return nil, &AddError{Code: "sea_region_alone", Values: map[string]any{"region": id, "id": parts[1][0], "count": len(parts[1])}, Provinces: parts[1],
				Message: fmt.Sprintf("海上战略区 #%d 被切成了几块,并到旁边的战略区后还是连不起来", id)}
		}
	}
	return moves, nil
}

func (b *Base) SeaRegionCutOff(provinces []int) []int {
	kinds := b.kinds()
	isSea := func(id int) bool { return id > 0 && id < len(kinds) && kinds[id] == kindSea }
	parts := seaParts(provinces, b.Borders(provinces), isSea)
	if len(parts) < 2 {
		return nil
	}
	var out []int
	for _, part := range parts[1:] {
		out = append(out, part...)
	}
	sort.Ints(out)
	return out
}

func seaParts(provinces []int, borders map[int]map[int]int, isSea func(int) bool) [][]int {
	inside := map[int]bool{}
	for _, p := range provinces {
		inside[p] = true
	}
	seen := map[int]bool{}
	var parts [][]int
	sorted := append([]int(nil), provinces...)
	sort.Ints(sorted)
	for _, first := range sorted {
		if seen[first] {
			continue
		}
		seen[first] = true
		part := []int{first}
		for i := 0; i < len(part); i++ {
			for other := range borders[part[i]] {
				if inside[other] && !seen[other] {
					seen[other] = true
					part = append(part, other)
				}
			}
		}
		for _, p := range part {
			if isSea(p) {
				sort.Ints(part)
				parts = append(parts, part)
				break
			}
		}
	}
	seas := func(part []int) int {
		n := 0
		for _, p := range part {
			if isSea(p) {
				n++
			}
		}
		return n
	}
	sort.SliceStable(parts, func(i, j int) bool {
		if a, b := seas(parts[i]), seas(parts[j]); a != b {
			return a > b
		}
		return parts[i][0] < parts[j][0]
	})
	return parts
}
