package mapgen

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"sort"
)

type AddOptions struct {
	Place image.Rectangle
	Seed  int64

	LandProvincePixels int

	SpeckPixels       int
	SketchSpeckPixels int
	Rounds            int

	ProvincesPerState int

	Continent int
}

func (o AddOptions) withDefaults() AddOptions {
	if o.LandProvincePixels <= 0 {
		o.LandProvincePixels = 400
	}
	if o.SpeckPixels <= 0 {
		o.SpeckPixels = 8
	}
	if o.SketchSpeckPixels <= 0 {
		o.SketchSpeckPixels = 3
	}
	if o.Rounds <= 0 {
		o.Rounds = 2
	}
	if o.ProvincesPerState <= 0 {
		o.ProvincesPerState = 8
	}
	return o
}

type NewState struct {
	Provinces []int
	Capital   int
}

type Addition struct {
	Provinces []Province
	States    []NewState

	CutFrom map[int]int

	Covered map[int]int

	NowCoastal []int

	Ignored int

	Changed image.Rectangle

	pixels map[int][]int32

	was map[int]string
}

func (a *Addition) Pixels(id int) []int32 { return a.pixels[id] }

type AddError struct {
	Message   string
	Code      string
	Values    map[string]any
	Provinces []int
}

func (e *AddError) Error() string { return e.Message }

var ErrNoLand = &AddError{Message: "草图放的地方没有落在海上的陆地:新土地只能画在海上", Code: "land_off_sea"}

const (
	kindNone byte = iota
	kindLand
	kindSea
	kindLake
)

func (b *Base) AddLand(sketch image.Image, options AddOptions) (*Addition, error) {
	opt := options.withDefaults()
	place := opt.Place
	if place.Dx() < 4 || place.Dy() < 4 || !place.In(image.Rect(0, 0, b.Width, b.Height)) {
		return nil, &AddError{Message: fmt.Sprintf("草图要整个放在地图里(地图 %d×%d),且至少 4×4 像素", b.Width, b.Height),
			Code: "land_outside_map", Values: map[string]any{"width": b.Width, "height": b.Height}}
	}
	W := b.Width
	kinds := b.kinds()
	kindAt := func(p int) byte {
		id := b.Province[p]
		if int(id) < len(kinds) {
			return kinds[id]
		}
		return kindNone
	}

	src, sw, sh := ClassifySketch(sketch)
	removeSpecks(src, sw, sh, opt.SketchSpeckPixels)
	pw, ph := place.Dx(), place.Dy()
	classes := Resample(src, sw, sh, pw, ph, opt.Seed)

	const margin = 2
	win := place.Inset(-margin).Intersect(image.Rect(0, 0, b.Width, b.Height))
	ww, wh := win.Dx(), win.Dy()
	toMap := func(wp int) int { return (win.Min.Y+wp/ww)*W + win.Min.X + wp%ww }
	inWin := func(p int) (int, bool) {
		x, y := p%W, p/W
		if x < win.Min.X || x >= win.Max.X || y < win.Min.Y || y >= win.Max.Y {
			return 0, false
		}
		return (y-win.Min.Y)*ww + x - win.Min.X, true
	}
	land := make([]bool, ww*wh)
	landClass := make([]Class, ww*wh)
	add := &Addition{CutFrom: map[int]int{}, Covered: map[int]int{}, pixels: map[int][]int32{}}
	for y := place.Min.Y; y < place.Max.Y; y++ {
		for x := place.Min.X; x < place.Max.X; x++ {
			c := classes[(y-place.Min.Y)*pw+x-place.Min.X]
			if !c.Land() {
				continue
			}
			p := y*W + x
			if kindAt(p) != kindSea {
				add.Ignored++
				continue
			}
			wp, _ := inWin(p)
			land[wp], landClass[wp] = true, c
		}
	}

	labels, sizes, _ := regions(ww, wh, func(a, c int) bool { return land[a] == land[c] })
	for wp := range land {
		if land[wp] && sizes[labels[wp]] < opt.SpeckPixels {
			land[wp] = false
		}
	}
	for wp, isLand := range land {
		if isLand {
			add.Covered[int(b.Province[toMap(wp)])]++
		}
	}
	if len(add.Covered) == 0 {
		return nil, ErrNoLand
	}

	scan := b.scan()
	type piece struct {
		from   int
		pixels []int32
		lake   bool
	}
	var pieces []piece
	touched := make([]int, 0, len(add.Covered))
	for id := range add.Covered {
		touched = append(touched, id)
	}
	sort.Ints(touched)
	for _, id := range touched {
		box := scan.box[id]
		bw, bh := box.Dx(), box.Dy()

		state := make([]byte, bw*bh)
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				p := y*W + x
				if int(b.Province[p]) != id {
					continue
				}
				s := byte(1)
				if wp, ok := inWin(p); ok && land[wp] {
					s = 2
				}
				state[(y-box.Min.Y)*bw+x-box.Min.X] = s
			}
		}
		parts, partSizes, _ := regions(bw, bh, func(a, c int) bool { return state[a] == state[c] })
		nextToLand := make([]bool, len(partSizes))
		for q, s := range state {
			if s != 1 {
				continue
			}
			x, y := q%bw, q/bw
			for _, n := range [4][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				if n[0] >= 0 && n[1] >= 0 && n[0] < bw && n[1] < bh && state[n[1]*bw+n[0]] == 2 {
					nextToLand[parts[q]] = true
				}
			}
		}
		left, keep := 0, int32(-1)
		for q, s := range state {
			if s != 1 {
				continue
			}
			part := parts[q]
			left++
			if nextToLand[part] && (keep < 0 || partSizes[part] > partSizes[keep]) {
				keep = part
			}
		}
		if left < SmallProvincePixels {
			return nil, &AddError{Message: fmt.Sprintf("海洋省份 #%d 几乎被新土地整个盖住(只剩 %d 个像素):海洋省份要留下一部分,编号才保得住。把草图挪一挪或缩小", id, left),
				Code: "land_covers_sea", Values: map[string]any{"id": id, "pixels": left}, Provinces: []int{id}}
		}
		cut := map[int32][]int32{}
		var kept []int32
		for q, s := range state {
			if s != 1 {
				continue
			}
			p := int32((box.Min.Y+q/bw)*W + box.Min.X + q%bw)
			if part := parts[q]; nextToLand[part] && part != keep {
				cut[part] = append(cut[part], p)
			} else {
				kept = append(kept, p)
			}
		}
		add.pixels[id] = kept
		order := make([]int32, 0, len(cut))
		for part := range cut {
			order = append(order, part)
		}
		sort.Slice(order, func(i, j int) bool { return cut[order[i]][0] < cut[order[j]][0] })
		for _, part := range order {
			px := cut[part]
			if len(px) < opt.SpeckPixels {
				if b.fillWithLand(px, inWin, land, landClass) {
					continue
				}
			}
			pieces = append(pieces, piece{from: id, pixels: px, lake: b.enclosed(px, kindAt, inWin, land)})
		}
	}

	landLabels, _, _ := regions(ww, wh, func(a, c int) bool { return land[a] && land[c] })
	area := make([]int32, ww*wh)
	compact := map[int32]int32{}
	var areaSizes []int
	for wp := range area {
		if !land[wp] {
			area[wp] = -int32(wp) - 2
			continue
		}
		a, ok := compact[landLabels[wp]]
		if !ok {
			a = int32(len(areaSizes))
			compact[landLabels[wp]] = a
			areaSizes = append(areaSizes, 0)
		}
		area[wp] = a
		areaSizes[a]++
	}
	targets := make([]int, len(areaSizes))
	for a := range targets {
		targets[a] = opt.LandProvincePixels
	}
	rng := rand.New(rand.NewSource(opt.Seed))
	seeds := placeSeeds(area, areaSizes, targets, rng)
	cell := grow(ww, wh, area, seeds)
	for round := 0; round < opt.Rounds; round++ {
		relax(ww, cell, seeds)
		cell = grow(ww, wh, area, seeds)
	}
	count := make([]int, len(seeds))
	for _, c := range cell {
		if c >= 0 {
			count[c]++
		}
	}
	absorb(ww, wh, area, cell, func(c int32) bool { return c >= 0 && count[c] < opt.LandProvincePixels/6 })

	pieceOf := map[int32]int32{}
	for k, pc := range pieces {
		for _, p := range pc.pixels {
			pieceOf[p] = int32(k)
		}
	}
	for wp := range cell {
		if land[wp] {
			continue
		}
		p := int32(toMap(wp))
		if k, ok := pieceOf[p]; ok {
			cell[wp] = -1_000_000 - k
		} else {
			cell[wp] = -b.Province[p] - 2
		}
	}
	fixCrossings(ww, wh, area, cell)
	joinPieces(ww, wh, area, cell)
	fixCrossings(ww, wh, area, cell)

	firstLand := b.MaxProvinceID() + 1
	next := firstLand
	idOfCell := map[int32]int{}
	for wp, c := range cell {
		if c < 0 || !land[wp] {
			continue
		}
		if _, ok := idOfCell[c]; !ok {
			idOfCell[c] = next
			next++
		}
	}
	firstWater := next
	for k := range pieces {
		add.CutFrom[firstWater+k] = pieces[k].from
	}
	continent := opt.Continent
	if continent <= 0 {
		continent = b.nearestContinent(place, kinds, scan)
	}
	used := make(map[uint32]bool, len(b.Definitions)+next)
	used[0] = true
	for _, d := range b.Definitions {
		used[d.RGB] = true
	}
	newColor := func() uint32 {
		for {
			c := uint32(rng.Intn(1 << 24))
			if !used[c] {
				used[c] = true
				return c
			}
		}
	}

	winClasses := make([]Class, ww*wh)
	classCount := map[int][9]int{}
	for wp, isLand := range land {
		p := toMap(wp)
		switch {
		case isLand:
			id := idOfCell[cell[wp]]
			b.Province[p] = int32(id)
			add.pixels[id] = append(add.pixels[id], int32(p))
			counts := classCount[id]
			counts[landClass[wp]]++
			classCount[id] = counts
			winClasses[wp] = landClass[wp]
			b.Terrain[p] = entryOf(landClass[wp]).Terrain
			b.Rivers[p] = RiverLand
		case kindAt(p) == kindLand:
			winClasses[wp] = Plains
		default:
			winClasses[wp] = Sea
		}
	}
	for k, pc := range pieces {
		id := firstWater + k
		for _, p := range pc.pixels {
			b.Province[p] = int32(id)
			if pc.lake {
				b.Terrain[p] = entryOf(Lake).Terrain
			}
		}
		add.pixels[id] = pc.pixels
	}
	field := heights(winClasses, ww, wh, opt.Seed)
	for wp, isLand := range land {
		if isLand {
			b.Heights[toMap(wp)] = field[wp]
		}
	}

	kinds = b.kinds()
	for id := len(kinds); id < firstWater+len(pieces); id++ {
		kinds = append(kinds, kindNone)
	}
	for id := firstLand; id < firstWater; id++ {
		kinds[id] = kindLand
	}
	for k, pc := range pieces {
		if pc.lake {
			kinds[firstWater+k] = kindLake
		} else {
			kinds[firstWater+k] = kindSea
		}
	}
	touchesSea := map[int]bool{}
	touchesLand := map[int]bool{}
	bounds := add.changedBounds(W, firstLand)
	for y := max(bounds.Min.Y-1, 0); y < min(bounds.Max.Y+1, b.Height); y++ {
		for x := max(bounds.Min.X-1, 0); x < min(bounds.Max.X+1, W); x++ {
			p := y*W + x
			for _, q := range [2]int{p + 1, p + W} {
				if (q == p+1 && x+1 >= W) || q >= len(b.Province) {
					continue
				}
				a, c := int(b.Province[p]), int(b.Province[q])
				if a == c || a >= len(kinds) || c >= len(kinds) {
					continue
				}
				if kinds[a] == kindLand && kinds[c] == kindSea {
					touchesSea[a], touchesLand[c] = true, true
				} else if kinds[c] == kindLand && kinds[a] == kindSea {
					touchesSea[c], touchesLand[a] = true, true
				}
			}
		}
	}
	at := b.definitionIndex()
	for id := range touchesLand {
		if i, ok := at[id]; ok && !b.Definitions[i].Coastal {
			b.Definitions[i].Coastal = true
			add.NowCoastal = append(add.NowCoastal, id)
		}
	}
	for id := range touchesSea {
		if i, ok := at[id]; ok && id < firstLand && !b.Definitions[i].Coastal {
			b.Definitions[i].Coastal = true
			add.NowCoastal = append(add.NowCoastal, id)
		}
	}
	sort.Ints(add.NowCoastal)

	landIDs := make([]int, 0, len(idOfCell))
	for _, id := range idOfCell {
		landIDs = append(landIDs, id)
	}
	sort.Ints(landIDs)
	for _, id := range landIDs {
		counts := classCount[id]
		major, most := Plains, -1
		for class, n := range counts {
			if n > most {
				major, most = Class(class), n
			}
		}
		pr := Province{ID: id, Kind: "land", Coastal: touchesSea[id], Terrain: entryOf(major).Key, Continent: continent}
		add.Provinces = append(add.Provinces, pr)
	}
	for k, pc := range pieces {
		pr := Province{ID: firstWater + k, Kind: "sea", Terrain: "ocean", Coastal: touchesLand[firstWater+k]}
		if pc.lake {
			pr.Kind, pr.Terrain, pr.Coastal = "lake", "lakes", false
		}
		add.Provinces = append(add.Provinces, pr)
	}
	for i := range add.Provinces {
		pr := &add.Provinces[i]
		rgb := newColor()
		pr.Color = [3]uint8{uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb)}
		px := add.pixels[pr.ID]
		pr.Pixels = len(px)
		var sx, sy float64
		for _, p := range px {
			sx += float64(int(p) % W)
			sy += float64(int(p) / W)
		}
		pr.X, pr.Y = sx/float64(len(px)), sy/float64(len(px))
		b.Definitions = append(b.Definitions, Definition{ID: pr.ID, RGB: rgb, Kind: pr.Kind, Coastal: pr.Coastal, Terrain: pr.Terrain, Continent: pr.Continent})
	}
	add.Changed = bounds
	add.States = b.groupStates(add, landIDs, opt.ProvincesPerState)
	return add, nil
}

func (b *Base) kinds() []byte {
	out := make([]byte, b.MaxProvinceID()+1)
	for _, d := range b.Definitions {
		switch d.Kind {
		case "land":
			out[d.ID] = kindLand
		case "sea":
			out[d.ID] = kindSea
		case "lake":
			out[d.ID] = kindLake
		}
	}
	return out
}

func (b *Base) IsLand() func(int) bool {
	kinds := b.kinds()
	return func(id int) bool { return id > 0 && id < len(kinds) && kinds[id] == kindLand }
}

type provinceScan struct {
	box      []image.Rectangle
	count    []int
	sumX     []float64
	sumY     []float64
	maxIndex int
}

func (b *Base) scan() provinceScan {
	n := b.MaxProvinceID() + 1
	out := provinceScan{box: make([]image.Rectangle, n), count: make([]int, n), sumX: make([]float64, n), sumY: make([]float64, n), maxIndex: n - 1}
	for p, id := range b.Province {
		if id <= 0 || int(id) >= n {
			continue
		}
		x, y := p%b.Width, p/b.Width
		if out.count[id] == 0 {
			out.box[id] = image.Rect(x, y, x+1, y+1)
		} else {
			r := &out.box[id]
			r.Min.X, r.Min.Y = min(r.Min.X, x), min(r.Min.Y, y)
			r.Max.X, r.Max.Y = max(r.Max.X, x+1), max(r.Max.Y, y+1)
		}
		out.count[id]++
		out.sumX[id] += float64(x)
		out.sumY[id] += float64(y)
	}
	return out
}

func (b *Base) fillWithLand(px []int32, inWin func(int) (int, bool), land []bool, landClass []Class) bool {
	class := Plains
	found := false
	for _, p := range px {
		if _, ok := inWin(int(p)); !ok {
			return false
		}
		for _, q := range [4]int{int(p) - 1, int(p) + 1, int(p) - b.Width, int(p) + b.Width} {
			if wq, ok := inWin(q); ok && land[wq] && !found {
				class, found = landClass[wq], true
			}
		}
	}
	for _, p := range px {
		wp, _ := inWin(int(p))
		land[wp], landClass[wp] = true, class
	}
	return true
}

func (b *Base) enclosed(px []int32, kindAt func(int) byte, inWin func(int) (int, bool), land []bool) bool {
	own := make(map[int32]bool, len(px))
	for _, p := range px {
		own[p] = true
	}
	for _, p := range px {
		x, y := int(p)%b.Width, int(p)/b.Width
		if x == 0 || y == 0 || x == b.Width-1 || y == b.Height-1 {
			return false
		}
		for _, q := range [4]int{int(p) - 1, int(p) + 1, int(p) - b.Width, int(p) + b.Width} {
			if own[int32(q)] {
				continue
			}
			if wq, ok := inWin(q); ok && land[wq] {
				continue
			}
			if k := kindAt(q); k == kindSea || k == kindLake {
				return false
			}
		}
	}
	return true
}

func (b *Base) nearestContinent(place image.Rectangle, kinds []byte, scan provinceScan) int {
	cx := float64(place.Min.X+place.Max.X) / 2
	cy := float64(place.Min.Y+place.Max.Y) / 2
	best, bestDistance := 1, math.MaxFloat64
	at := b.definitionIndex()
	for id, count := range scan.count {
		if count == 0 || id >= len(kinds) || kinds[id] != kindLand {
			continue
		}
		dx := math.Abs(scan.sumX[id]/float64(count) - cx)
		dx = math.Min(dx, float64(b.Width)-dx)
		dy := scan.sumY[id]/float64(count) - cy
		if d := dx*dx + dy*dy; d < bestDistance {
			if i, ok := at[id]; ok && b.Definitions[i].Continent > 0 {
				best, bestDistance = b.Definitions[i].Continent, d
			}
		}
	}
	return best
}

func (a *Addition) changedBounds(width, firstNew int) image.Rectangle {
	var r image.Rectangle
	for id, px := range a.pixels {
		if id < firstNew {
			continue
		}
		for _, p := range px {
			x, y := int(p)%width, int(p)/width
			r = r.Union(image.Rect(x, y, x+1, y+1))
		}
	}
	return r
}

func (b *Base) groupStates(add *Addition, landIDs []int, per int) []NewState {
	if len(landIDs) == 0 {
		return nil
	}
	isNew := map[int]bool{}
	for _, id := range landIDs {
		isNew[id] = true
	}
	neighbours := map[int]map[int]bool{}
	for _, id := range landIDs {
		neighbours[id] = map[int]bool{}
		for _, p := range add.pixels[id] {
			for _, q := range [4]int{int(p) - 1, int(p) + 1, int(p) - b.Width, int(p) + b.Width} {
				if q < 0 || q >= len(b.Province) {
					continue
				}
				if other := int(b.Province[q]); other != id && isNew[other] {
					neighbours[id][other] = true
				}
			}
		}
	}
	centre := map[int][2]float64{}
	for _, pr := range add.Provinces {
		centre[pr.ID] = [2]float64{pr.X, pr.Y}
	}
	return groupNear(landIDs, neighbours, centre, per)
}

func groupNear(ids []int, neighbours map[int]map[int]bool, centre map[int][2]float64, per int) []NewState {
	if len(ids) == 0 {
		return nil
	}
	order := append([]int(nil), ids...)
	sort.Slice(order, func(i, j int) bool {
		a, c := centre[order[i]], centre[order[j]]
		if a[0] != c[0] {
			return a[0] < c[0]
		}
		return order[i] < order[j]
	})
	stateOf := map[int]int{}
	var groups [][]int
	for _, start := range order {
		if _, done := stateOf[start]; done {
			continue
		}
		k := len(groups)
		group := []int{start}
		stateOf[start] = k
		for len(group) < per {

			best, bestDistance := 0, math.MaxFloat64
			for _, member := range group {
				for other := range neighbours[member] {
					if _, done := stateOf[other]; done {
						continue
					}
					dx, dy := centre[other][0]-centre[start][0], centre[other][1]-centre[start][1]
					if d := dx*dx + dy*dy; d < bestDistance || (d == bestDistance && other < best) {
						best, bestDistance = other, d
					}
				}
			}
			if best == 0 {
				break
			}
			group = append(group, best)
			stateOf[best] = k
		}
		groups = append(groups, group)
	}

	for k, group := range groups {
		if len(group) == 0 || len(group)*2 >= per || len(groups) == 1 {
			continue
		}
		target := -1
		for _, member := range group {
			for other := range neighbours[member] {
				if s := stateOf[other]; s != k && len(groups[s]) > 0 && (target < 0 || s < target) {
					target = s
				}
			}
		}
		if target < 0 {
			continue
		}
		for _, member := range group {
			stateOf[member] = target
		}
		groups[target] = append(groups[target], group...)
		groups[k] = nil
	}
	var out []NewState
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		sort.Ints(group)
		var sx, sy float64
		for _, id := range group {
			sx += centre[id][0]
			sy += centre[id][1]
		}
		sx, sy = sx/float64(len(group)), sy/float64(len(group))
		capital, bestDistance := group[0], math.MaxFloat64
		for _, id := range group {
			dx, dy := centre[id][0]-sx, centre[id][1]-sy
			if d := dx*dx + dy*dy; d < bestDistance {
				capital, bestDistance = id, d
			}
		}
		out = append(out, NewState{Provinces: group, Capital: capital})
	}
	return out
}
