package mapgen

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/image/bmp"
)

func (b *Base) Flood(ids []int) (*Addition, error) { return b.drown(ids, Lake) }

func (b *Base) Sink(ids []int) (*Addition, error) { return b.drown(ids, Sea) }

func (b *Base) drown(ids []int, class Class) (*Addition, error) {
	rows := map[int]*Definition{}
	for i := range b.Definitions {
		rows[b.Definitions[i].ID] = &b.Definitions[i]
	}
	flooded := map[int32]bool{}
	for _, id := range ids {
		row, ok := rows[id]
		if !ok {
			return nil, NoProvince(id)
		}
		if row.Kind != "land" {
			return nil, &AddError{Message: fmt.Sprintf("省份 #%d 已经是水面", id), Code: "already_water",
				Values: map[string]any{"id": id}, Provinces: []int{id}}
		}
		flooded[int32(id)] = true
	}
	add := &Addition{pixels: map[int][]int32{}}
	changed := image.Rectangle{}
	for p, id := range b.Province {
		if flooded[id] {
			add.pixels[int(id)] = append(add.pixels[int(id)], int32(p))
			x, y := p%b.Width, p/b.Width
			changed = changed.Union(image.Rect(x, y, x+1, y+1))
		}
	}
	add.Changed = changed
	sort.Ints(ids)
	water := Legend[class]
	kind := "lake"
	if class == Sea {
		kind = "sea"
	}
	for _, id := range ids {
		row := rows[id]
		row.Kind, row.Terrain, row.Coastal, row.Continent = kind, water.Key, false, 0
		add.Provinces = append(add.Provinces, Province{ID: id, Kind: kind, Terrain: water.Key})
	}
	if changed.Empty() {
		return add, nil
	}

	area := changed.Inset(-1).Intersect(image.Rect(0, 0, b.Width, b.Height))
	w, h := area.Dx(), area.Dy()
	depth := make([]int32, w*h)
	var queue []int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if flooded[b.Province[(area.Min.Y+y)*b.Width+area.Min.X+x]] {
				depth[y*w+x] = -1
			} else {
				queue = append(queue, y*w+x)
			}
		}
	}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w, i/w
		for _, n := range [4][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
			if n[0] < 0 || n[1] < 0 || n[0] >= w || n[1] >= h {
				continue
			}
			if j := n[1]*w + n[0]; depth[j] == -1 {
				depth[j] = depth[i] + 1
				queue = append(queue, j)
			}
		}
	}
	before := append([]uint8(nil), b.Rivers...)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := depth[y*w+x]
			if d == 0 {
				continue
			}
			if d < 0 {
				d = 99
			}
			p := (area.Min.Y+y)*b.Width + area.Min.X + x
			height := water.Height
			if class == Lake {
				height += float64(max(0, 4-int(d)))
			}
			b.Heights[p] = uint8(math.Min(float64(b.Heights[p]), height))
			b.Terrain[p] = water.Terrain
			b.Rivers[p] = RiverWater
		}
	}
	b.mendCutRivers(before, flooded)
	if class == Sea {
		add.NowCoastal = b.recoast(flooded)
	}
	return add, nil
}

func NoProvince(id int) *AddError {
	return &AddError{Message: fmt.Sprintf("没有省份 #%d", id), Code: "no_province", Values: map[string]any{"id": id}}
}

func (b *Base) recoast(sunk map[int32]bool) []int {
	kinds := b.kinds()
	near := map[int32]bool{}
	touchesLand := map[int32]bool{}
	note := func(a, c int32) {
		if a == c {
			return
		}
		if sunk[a] {
			near[c] = true
		}
		if sunk[c] {
			near[a] = true
		}
		if int(c) < len(kinds) && kinds[c] == kindLand {
			touchesLand[a] = true
		}
		if int(a) < len(kinds) && kinds[a] == kindLand {
			touchesLand[c] = true
		}
	}
	b.eachEdge(note)
	var nowCoastal []int
	for i := range b.Definitions {
		d := &b.Definitions[i]
		id := int32(d.ID)
		switch {
		case d.Kind == "land" && near[id] && !d.Coastal:
			d.Coastal = true
			nowCoastal = append(nowCoastal, d.ID)
		case d.Kind == "sea" && (sunk[id] || near[id]):
			d.Coastal = touchesLand[id]
		}
	}
	sort.Ints(nowCoastal)
	return nowCoastal
}

func (b *Base) eachEdge(note func(a, c int32)) {
	for p, a := range b.Province {
		x := p % b.Width
		right := p + 1
		if x+1 == b.Width {
			right = p - x
		}
		note(a, b.Province[right])
		if p+b.Width < len(b.Province) {
			note(a, b.Province[p+b.Width])
		}
	}
}

func (b *Base) Borders(ids []int) map[int]map[int]int {
	want := map[int32]bool{}
	out := map[int]map[int]int{}
	for _, id := range ids {
		want[int32(id)] = true
		out[id] = map[int]int{}
	}
	note := func(a, c int32) {
		if a == c {
			return
		}
		if want[a] {
			out[int(a)][int(c)]++
		}
		if want[c] {
			out[int(c)][int(a)]++
		}
	}
	b.eachEdge(note)
	return out
}

func (b *Base) UnitStacks(ids []int) []string {
	pixels := map[int32][]int32{}
	want := map[int32]bool{}
	for _, id := range ids {
		want[int32(id)] = true
	}
	for p, id := range b.Province {
		if want[id] {
			pixels[id] = append(pixels[id], int32(p))
		}
	}
	var out []string
	for _, id := range ids {
		px := pixels[int32(id)]
		if len(px) == 0 {
			continue
		}
		x, y, z := b.at(b.middle(px))
		for _, kind := range unitStackTypes {
			out = append(out, fmt.Sprintf("%d;%d;%.2f;%.2f;%.2f;0.00;0.30", id, kind, x, y, z))
		}
	}
	return out
}

func (b *Base) AddCoastalBuildings(text string, provinces []int, stateOf map[int]int, seed int64) (string, int) {
	lines, newline, ended := textLines(text)
	kinds := b.kinds()
	coastal := map[string]bool{}
	for _, kind := range coastalBuildings {
		coastal[kind] = true
	}
	has := map[int]bool{}
	for _, line := range lines {
		fields := strings.Split(line, ";")
		if len(fields) >= 7 && coastal[strings.TrimSpace(fields[1])] {
			if p := b.landAtLine(fields, kinds); p > 0 {
				has[p] = true
			}
		}
	}
	want := map[int32]bool{}
	for _, id := range provinces {
		if !has[id] {
			want[int32(id)] = true
		}
	}
	pixels := map[int32][]int32{}
	for p, id := range b.Province {
		if want[id] {
			pixels[id] = append(pixels[id], int32(p))
		}
	}
	rng := rand.New(rand.NewSource(seed + 307))
	rotation := func() float64 { return math.Round(rng.Float64()*2*math.Pi*100) / 100 }
	added := 0
	for _, id := range provinces {
		px := pixels[int32(id)]
		if len(px) == 0 {
			continue
		}
		land, water, sea, ok := b.shore(px, kinds, rng)
		if !ok {
			continue
		}
		state := stateOf[id]
		for _, kind := range coastalBuildings {
			switch kind {
			case "naval_base_spawn":
				lines = append(lines, b.line(state, kind, land, rotation(), sea))
			case "floating_harbor":
				lines = append(lines, b.line(state, kind, water, rotation(), sea))
			default:
				lines = append(lines, b.line(state, kind, land, rotation(), 0))
			}
			added++
		}
	}
	return joinLines(lines, newline, ended), added
}

func (b *Base) mendCutRivers(before []uint8, flooded map[int32]bool) {
	w, h := b.Width, b.Height
	besideFlood := func(p int) bool {
		x, y := p%w, p/w
		return (x > 0 && flooded[b.Province[p-1]]) || (x+1 < w && flooded[b.Province[p+1]]) ||
			(y > 0 && flooded[b.Province[p-w]]) || (y+1 < h && flooded[b.Province[p+w]])
	}
	seen := map[int]bool{}
	ranked := map[int]int{}
	for p := range b.Rivers {
		if !IsRiver(b.Rivers[p]) || seen[p] || !besideFlood(p) {
			continue
		}
		network := riverNetwork(b.Rivers, w, h, p)
		for _, q := range network {
			seen[q] = true
		}
		if _, ok := ranked[p]; !ok {
			for q, r := range riverFlow(before, w, h, p) {
				ranked[q] = r
			}
		}
		sources := 0
		best, bestRank := -1, 0
		for _, q := range network {
			if b.Rivers[q] == RiverSource {
				sources++
			}
			if riverDegree(b.Rivers, w, h, q) == 1 {
				if r := ranked[q]; best < 0 || r < bestRank || (r == bestRank && q < best) {
					best, bestRank = q, r
				}
			}
		}
		for _, q := range network {
			if v := b.Rivers[q]; (v == RiverJoin || v == RiverSplit) && riverDegree(b.Rivers, w, h, q) != 2 {
				b.Rivers[q] = RiverNarrow
			}
		}
		if sources == 0 && best >= 0 {
			b.Rivers[best] = RiverSource
		}
	}
}

func riverSides(rivers []uint8, w, h, p int) []int {
	x, y := p%w, p/w
	out := make([]int, 0, 4)
	for _, q := range [4]int{p - 1, p + 1, p - w, p + w} {
		if (q == p-1 && x == 0) || (q == p+1 && x+1 == w) || (q == p-w && y == 0) || (q == p+w && y+1 == h) {
			continue
		}
		if IsRiver(rivers[q]) {
			out = append(out, q)
		}
	}
	return out
}

func riverNetwork(rivers []uint8, w, h, p int) []int {
	seen := map[int]bool{p: true}
	stack := []int{p}
	var out []int
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, q)
		for _, n := range riverSides(rivers, w, h, q) {
			if !seen[n] {
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	sort.Ints(out)
	return out
}

func riverFlow(rivers []uint8, w, h, p int) map[int]int {
	network := riverNetwork(rivers, w, h, p)
	rank := map[int]int{}
	type step struct{ at, dir int }
	var queue []step
	for _, q := range network {
		if rivers[q] == RiverSource {
			rank[q] = 0
			queue = append(queue, step{q, 1})
		}
	}
	if len(queue) == 0 {
		rank[network[0]] = 0
		queue = append(queue, step{network[0], 1})
	}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, n := range riverSides(rivers, w, h, s.at) {
			if _, ok := rank[n]; ok {
				continue
			}
			dir := s.dir
			switch rivers[n] {
			case RiverJoin:
				dir = -1
			case RiverSplit:
				dir = 1
			}
			rank[n] = rank[s.at] + dir
			queue = append(queue, step{n, dir})
		}
	}
	return rank
}

func (b *Base) ClearTrees(trees []byte, add *Addition) ([]byte, error) {
	img, err := bmp.Decode(bytes.NewReader(trees))
	if err != nil {
		return nil, err
	}
	paletted, ok := img.(*image.Paletted)
	if !ok {
		return nil, ErrTexture
	}
	gone := map[int32]bool{}
	for _, px := range add.pixels {
		for _, p := range px {
			gone[p] = true
		}
	}
	w, h := paletted.Rect.Dx(), paletted.Rect.Dy()
	area := add.Changed
	for ty := max(area.Min.Y*h/b.Height, 0); ty <= min(area.Max.Y*h/b.Height, h-1); ty++ {
		for tx := max(area.Min.X*w/b.Width, 0); tx <= min(area.Max.X*w/b.Width, w-1); tx++ {
			if gone[int32(b.texel(w, h, tx, ty))] {
				paletted.Pix[ty*paletted.Stride+tx] = 0
			}
		}
	}
	return EncodeBMP(paletted)
}

func textLines(text string) ([]string, string, bool) {
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	body := strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r")
	ended := len(body) != len(text)
	if body == "" {
		return nil, newline, ended
	}
	return strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n"), newline, ended
}

func joinLines(lines []string, newline string, ended bool) string {
	out := strings.Join(lines, newline)
	if ended && len(lines) > 0 {
		out += newline
	}
	return out
}

func (b *Base) DropBuildingsIn(text string, provinces map[int]bool) (string, int) {
	lines, newline, ended := textLines(text)
	kinds := b.kinds()
	out := lines[:0]
	dropped := 0
	for _, line := range lines {
		fields := strings.Split(line, ";")
		if len(fields) >= 7 && provinces[b.landAtLine(fields, kinds)] {
			dropped++
			continue
		}
		out = append(out, line)
	}
	return joinLines(out, newline, ended), dropped
}

func CutRailways(text string, provinces map[int]bool) (string, int) {
	lines, newline, ended := textLines(text)
	var out []string
	cut := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			out = append(out, line)
			continue
		}
		var runs [][]string
		var run []string
		through := false
		for _, f := range fields[2:] {
			id, err := strconv.Atoi(f)
			if err == nil && provinces[id] {
				through = true
				runs = append(runs, run)
				run = nil
				continue
			}
			run = append(run, f)
		}
		if !through {
			out = append(out, line)
			continue
		}
		cut++
		runs = append(runs, run)
		for _, r := range runs {
			if len(r) >= 2 {
				out = append(out, fmt.Sprintf("%s %d %s ", fields[0], len(r), strings.Join(r, " ")))
			}
		}
	}
	return joinLines(out, newline, ended), cut
}

func DropProvinceLines(text, sep string, fields []int, provinces map[int]bool) (string, int) {
	lines, newline, ended := textLines(text)
	out := lines[:0]
	dropped := 0
	for _, line := range lines {
		var parts []string
		if sep == " " {
			parts = strings.Fields(line)
		} else {
			parts = strings.Split(line, sep)
		}
		hit := false
		for _, f := range fields {
			if f < len(parts) {
				if id, err := strconv.Atoi(strings.TrimSpace(parts[f])); err == nil && provinces[id] {
					hit = true
				}
			}
		}
		if hit {
			dropped++
			continue
		}
		out = append(out, line)
	}
	return joinLines(out, newline, ended), dropped
}
