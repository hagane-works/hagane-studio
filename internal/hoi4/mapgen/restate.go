package mapgen

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

type RestateState struct {
	Provinces []int
	Owner     string
}

type RestateNew struct {
	Provinces []int
	Owner     string
	From      int
}

type RestatePlan struct {
	Owner map[int]string
	Lists map[int][]int
	New   []RestateNew
}

func PlanRestate(states map[int]RestateState, neighbours map[int]map[int]bool, assign map[int]string, regionOf map[int]int, land func(int) bool) RestatePlan {
	if land == nil {
		land = func(int) bool { return true }
	}
	plan := RestatePlan{Owner: map[int]string{}, Lists: map[int][]int{}}
	stateOf := map[int]int{}
	ids := make([]int, 0, len(states))
	for id, st := range states {
		ids = append(ids, id)
		for _, p := range st.Provinces {
			stateOf[p] = id
		}
	}
	sort.Ints(ids)

	type piece struct {
		provinces []int
		owner     string
		from      int
	}
	kept := map[int][]int{}
	var pieces []piece
	for _, id := range ids {
		st := states[id]
		touched := false
		groups := map[string][]int{}
		var lakes []int
		for _, p := range st.Provinces {
			if !land(p) {
				lakes = append(lakes, p)
				continue
			}
			owner := st.Owner
			if tag, ok := assign[p]; ok && tag != "" {
				owner = tag
			}
			if owner != st.Owner {
				touched = true
			}
			groups[owner] = append(groups[owner], p)
		}
		if !touched {
			continue
		}
		keeper := st.Owner
		if len(groups[keeper]) == 0 {
			keeper = ""
			for tag, ps := range groups {
				if keeper == "" || len(ps) > len(groups[keeper]) || (len(ps) == len(groups[keeper]) && tag < keeper) {
					keeper = tag
				}
			}
			plan.Owner[id] = keeper
		}
		kept[id] = append(groups[keeper], lakes...)
		tags := make([]string, 0, len(groups))
		for tag := range groups {
			if tag != keeper {
				tags = append(tags, tag)
			}
		}
		sort.Strings(tags)
		for _, tag := range tags {
			for _, part := range byRegion(groups[tag], regionOf) {
				for _, c := range components(part, neighbours) {
					pieces = append(pieces, piece{c, tag, id})
				}
			}
		}
	}

	ownerOf := func(id int) string {
		if tag, ok := plan.Owner[id]; ok {
			return tag
		}
		return states[id].Owner
	}
	keeps := func(id, p int) bool {
		list, cut := kept[id]
		if !cut {
			return true
		}
		for _, q := range list {
			if q == p {
				return true
			}
		}
		return false
	}

	joined := map[int][]int{}
	var orphans []piece
	for _, pc := range pieces {
		border := map[int]int{}
		for _, p := range pc.provinces {
			for q := range neighbours[p] {
				t, ok := stateOf[q]
				if !ok || t == pc.from || ownerOf(t) != pc.owner || !keeps(t, q) || regionOf[q] != regionOf[p] {
					continue
				}
				border[t]++
			}
		}
		best, most := 0, 0
		for t, n := range border {
			if n > most || (n == most && t < best) {
				best, most = t, n
			}
		}
		if best == 0 {
			orphans = append(orphans, pc)
			continue
		}
		joined[best] = append(joined[best], pc.provinces...)
	}

	parent := make([]int, len(orphans))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	pieceOf := map[int]int{}
	for i, pc := range orphans {
		for _, p := range pc.provinces {
			pieceOf[p] = i
		}
	}
	for i, pc := range orphans {
		for _, p := range pc.provinces {
			for q := range neighbours[p] {
				if j, ok := pieceOf[q]; ok && orphans[j].owner == pc.owner && regionOf[q] == regionOf[p] {
					if a, b := find(i), find(j); a != b {
						parent[b] = a
					}
				}
			}
		}
	}
	merged := map[int]*RestateNew{}
	from := map[int]map[int]int{}
	var roots []int
	for i, pc := range orphans {
		root := find(i)
		n, ok := merged[root]
		if !ok {
			n = &RestateNew{Owner: pc.owner}
			merged[root] = n
			from[root] = map[int]int{}
			roots = append(roots, root)
		}
		n.Provinces = append(n.Provinces, pc.provinces...)
		from[root][pc.from] += len(pc.provinces)
	}
	for _, root := range roots {
		n := merged[root]
		sort.Ints(n.Provinces)
		most := -1
		for id, count := range from[root] {
			if count > most || (count == most && id < n.From) {
				n.From, most = id, count
			}
		}
	}
	sort.Slice(roots, func(a, b int) bool { return merged[roots[a]].Provinces[0] < merged[roots[b]].Provinces[0] })
	for _, root := range roots {
		plan.New = append(plan.New, *merged[root])
	}

	same := func(a, b []int) bool {
		if len(a) != len(b) {
			return false
		}
		a, b = append([]int(nil), a...), append([]int(nil), b...)
		sort.Ints(a)
		sort.Ints(b)
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	for id, list := range kept {
		next := append(append([]int(nil), list...), joined[id]...)
		sort.Ints(next)
		if !same(next, states[id].Provinces) {
			plan.Lists[id] = next
		}
	}
	for id, list := range joined {
		if _, done := plan.Lists[id]; done {
			continue
		}
		next := append(append([]int(nil), states[id].Provinces...), list...)
		sort.Ints(next)
		plan.Lists[id] = next
	}
	return plan
}

func SplitStatesByRegion(states map[int]RestateState, neighbours map[int]map[int]bool, regionOf map[int]int, capitalOf map[int]int) RestatePlan {
	plan := RestatePlan{Owner: map[int]string{}, Lists: map[int][]int{}}
	ids := make([]int, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		st := states[id]
		count := map[int]int{}
		for _, p := range st.Provinces {
			if r := regionOf[p]; r > 0 {
				count[r]++
			}
		}
		if len(count) < 2 {
			continue
		}
		capital := regionOf[capitalOf[id]]
		main, most := 0, -1
		for r, n := range count {
			if n > most || (n == most && (r == capital || (main != capital && r < main))) {
				main, most = r, n
			}
		}
		var kept, away []int
		for _, p := range st.Provinces {
			if r := regionOf[p]; r == 0 || r == main {
				kept = append(kept, p)
			} else {
				away = append(away, p)
			}
		}
		sort.Ints(kept)
		plan.Lists[id] = kept
		for _, part := range byRegion(away, regionOf) {
			for _, c := range components(part, neighbours) {
				plan.New = append(plan.New, RestateNew{Provinces: c, Owner: st.Owner, From: id})
			}
		}
	}
	return plan
}

func (b *Base) LandForLakeStates(states map[int]RestateState, regionOf map[int]int) RestatePlan {
	plan := RestatePlan{Owner: map[int]string{}, Lists: map[int][]int{}}
	kinds := b.kinds()
	isLand := func(id int) bool { return id > 0 && id < len(kinds) && kinds[id] == kindLand }
	ids := make([]int, 0, len(states))
	lists := map[int][]int{}
	stateOf := map[int]int{}
	var lakes []int
	for id, st := range states {
		ids = append(ids, id)
		lists[id] = append([]int(nil), st.Provinces...)
		landed := false
		for _, p := range st.Provinces {
			stateOf[p] = id
			landed = landed || isLand(p)
		}
		if !landed {
			lakes = append(lakes, st.Provinces...)
		}
	}
	if len(lakes) == 0 {
		return plan
	}
	sort.Ints(ids)
	borders := b.Borders(lakes)
	landIn := func(id int) int {
		n := 0
		for _, p := range lists[id] {
			if isLand(p) {
				n++
			}
		}
		return n
	}
	for _, id := range ids {
		if len(lists[id]) == 0 || landIn(id) > 0 {
			continue
		}
		best, most := 0, 0
		for _, lake := range lists[id] {
			for q, n := range borders[lake] {
				donor, ok := stateOf[q]
				if !ok || donor == id || !isLand(q) || regionOf[q] != regionOf[lake] || landIn(donor) < 2 {
					continue
				}
				if n > most || (n == most && q < best) {
					best, most = q, n
				}
			}
		}
		if best == 0 {
			continue
		}
		donor := stateOf[best]
		var rest []int
		for _, p := range lists[donor] {
			if p != best {
				rest = append(rest, p)
			}
		}
		lists[donor] = rest
		lists[id] = append(lists[id], best)
		sort.Ints(lists[id])
		stateOf[best] = id
		plan.Lists[donor], plan.Lists[id] = lists[donor], lists[id]
		owner := states[donor].Owner
		if tag, ok := plan.Owner[donor]; ok {
			owner = tag
		}
		if owner != states[id].Owner {
			plan.Owner[id] = owner
		}
	}
	return plan
}

func byRegion(provinces []int, regionOf map[int]int) [][]int {
	parts := map[int][]int{}
	for _, p := range provinces {
		parts[regionOf[p]] = append(parts[regionOf[p]], p)
	}
	keys := make([]int, 0, len(parts))
	for k := range parts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([][]int, 0, len(keys))
	for _, k := range keys {
		out = append(out, parts[k])
	}
	return out
}

func components(provinces []int, neighbours map[int]map[int]bool) [][]int {
	in := map[int]bool{}
	for _, p := range provinces {
		in[p] = true
	}
	seen := map[int]bool{}
	sorted := append([]int(nil), provinces...)
	sort.Ints(sorted)
	var out [][]int
	for _, start := range sorted {
		if seen[start] {
			continue
		}
		var c []int
		queue := []int{start}
		seen[start] = true
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			c = append(c, p)
			for q := range neighbours[p] {
				if in[q] && !seen[q] {
					seen[q] = true
					queue = append(queue, q)
				}
			}
		}
		sort.Ints(c)
		out = append(out, c)
	}
	return out
}

func (b *Base) Neighbours() map[int]map[int]bool {
	out := map[int]map[int]bool{}
	link := func(a, c int32) {
		if a == c || a == 0 || c == 0 {
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
		row := y * b.Width
		for x := 0; x < b.Width; x++ {
			here := b.Province[row+x]
			link(here, b.Province[row+(x+1)%b.Width])
			if y+1 < b.Height {
				link(here, b.Province[row+b.Width+x])
			}
		}
	}
	return out
}

func (b *Base) RestateBuildings(text string, stateOf map[int]int, ensure map[int][]int, seed int64) (string, int, int) {
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	body := strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r")
	ended := len(body) != len(text)
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if body == "" {
		lines = nil
	}
	kinds := b.kinds()
	stateWide := map[string]bool{}
	for _, building := range stateBuildings {
		stateWide[building.kind] = true
	}
	renamed := 0
	have := map[int]map[string]int{}
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, ";")
		if len(fields) < 7 {
			kept = append(kept, line)
			continue
		}
		state, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			kept = append(kept, line)
			continue
		}
		if p := b.landAtLine(fields, kinds); p > 0 {
			if next, ok := stateOf[p]; ok && next != state {
				fields[0] = strconv.Itoa(next)
				line = strings.Join(fields, ";")
				state = next
				renamed++
			}
		}
		kind := strings.TrimSpace(fields[1])

		if _, redo := ensure[state]; redo && stateWide[kind] && !b.overLand(fields, kinds) {
			continue
		}
		kept = append(kept, line)
		if have[state] == nil {
			have[state] = map[string]int{}
		}
		have[state][kind]++
	}
	lines = kept

	rng := rand.New(rand.NewSource(seed + 211))
	rotation := func() float64 { return math.Round(rng.Float64()*2*math.Pi*100) / 100 }
	states := make([]int, 0, len(ensure))
	for id := range ensure {
		states = append(states, id)
	}
	sort.Ints(states)
	pixels := b.pixelsOf(ensure)
	var added []string
	for _, id := range states {

		var capital []int32
		for _, p := range ensure[id] {
			if p > 0 && p < len(kinds) && kinds[p] == kindLand && len(pixels[p]) > 0 {
				capital = pixels[p]
				break
			}
		}
		if len(capital) == 0 {
			continue
		}
		provinces := ensure[id]
		for _, building := range stateBuildings {
			for i := have[id][building.kind]; i < building.count; i++ {
				added = append(added, b.line(id, building.kind, int(capital[rng.Intn(len(capital))]), rotation(), 0))
			}
		}
		if have[id]["dockyard"] == 0 {
			for _, p := range provinces {
				if land, _, _, ok := b.shore(pixels[p], kinds, rng); ok {
					added = append(added, b.line(id, "dockyard", land, rotation(), 0))
					break
				}
			}
		}
	}
	lines = append(lines, added...)
	out := strings.Join(lines, newline)
	if ended && len(lines) > 0 {
		out += newline
	}
	return out, renamed, len(added)
}

func (b *Base) landAtLine(fields []string, kinds []byte) int {
	x, err1 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
	z, err2 := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
	if err1 != nil || err2 != nil {
		return 0
	}
	px, py := int(math.Round(x)), b.Height-1-int(math.Round(z))
	land := func(x, y int) int {
		if y < 0 || y >= b.Height {
			return 0
		}
		x = ((x % b.Width) + b.Width) % b.Width
		id := b.Province[y*b.Width+x]
		if int(id) < len(kinds) && kinds[id] == kindLand {
			return int(id)
		}
		return 0
	}
	if p := land(px, py); p > 0 {
		return p
	}
	for r := 1; r <= 4; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if p := land(px+dx, py+dy); p > 0 {
					return p
				}
			}
		}
	}
	return 0
}

func (b *Base) overLand(fields []string, kinds []byte) bool {
	x, err1 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
	z, err2 := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
	if err1 != nil || err2 != nil {
		return true
	}
	px, py := int(math.Round(x)), b.Height-1-int(math.Round(z))
	if py < 0 || py >= b.Height {
		return true
	}
	px = ((px % b.Width) + b.Width) % b.Width
	id := b.Province[py*b.Width+px]
	return int(id) < len(kinds) && kinds[id] == kindLand
}

func (b *Base) pixelsOf(states map[int][]int) map[int][]int32 {
	want := map[int32]bool{}
	for _, provinces := range states {
		for _, p := range provinces {
			want[int32(p)] = true
		}
	}
	out := map[int][]int32{}
	for p, id := range b.Province {
		if want[id] {
			out[int(id)] = append(out[int(id)], int32(p))
		}
	}
	return out
}

func RestateName(from string, n int) string {
	return fmt.Sprintf("%s %d", from, n)
}

func RestateStem(name string) (string, int) {
	cut := strings.LastIndexByte(name, ' ')
	if cut <= 0 {
		return name, 0
	}
	n, err := strconv.Atoi(name[cut+1:])
	if err != nil || n <= 0 {
		return name, 0
	}
	return name[:cut], n
}
