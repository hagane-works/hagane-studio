package mapsketch

import (
	"errors"
	"fmt"
	"image"
	"math"
	"strings"
)

type Seed struct {
	At  Point
	Tag string
}

type RegionOptions struct {
	Seeds []Seed

	Land func(Point) bool

	Tolerance float64

	Dark float64

	SeaCost float64
}

type Region struct {
	Tag string `json:"tag"`

	Colours []string `json:"colours"`
	Seeds   []Point  `json:"seeds"`

	Share float64 `json:"share"`

	Parts  int   `json:"parts"`
	Box    Rect  `json:"box"`
	Centre Point `json:"centre"`
}

type RegionResult struct {
	Tags    []string
	Regions []Region

	Seeds []Seed

	Warnings []string
	work     *Picture
	srcW     int
	srcH     int

	tagOf []uint8
}

func Regions(src *Picture, opt RegionOptions) (*RegionResult, error) {
	if len(opt.Seeds) == 0 {
		return nil, errors.New("至少要给一个国家的位置")
	}
	if opt.Tolerance <= 0 {
		opt.Tolerance = 28
	}
	if opt.Dark == 0 {
		opt.Dark = DefaultDark
	}
	if opt.SeaCost <= 0 {
		opt.SeaCost = 6
	}
	work := src.Scaled(1400)
	w, h := work.W, work.H
	toSource := func(i int) Point {
		return Point{X: int((float64(i%w) + 0.5) * float64(src.W) / float64(w)), Y: int((float64(i/w) + 0.5) * float64(src.H) / float64(h))}
	}
	toWork := func(p Point) int {
		x := min(w-1, max(0, p.X*w/src.W))
		y := min(h-1, max(0, p.Y*h/src.H))
		return y*w + x
	}
	land := make([]bool, len(work.Pix))
	dark := make([]bool, len(work.Pix))
	landCount := 0
	for i, c := range work.Pix {
		land[i] = !work.clear(i) && (opt.Land == nil || opt.Land(toSource(i)))
		if land[i] {
			landCount++
		}
		dark[i] = opt.Dark > 0 && luminance(c) < opt.Dark
	}
	if landCount == 0 {
		return nil, errors.New("图上没有陆地")
	}

	result := &RegionResult{work: work, srcW: src.W, srcH: src.H, Seeds: append([]Seed(nil), opt.Seeds...)}
	seeds := result.Seeds
	tagIndex := map[string]int{}
	seedTag := make([]int, len(seeds))
	seedColour := make([]RGB, len(seeds))
	seedAt := make([]int, len(seeds))

	reach := max(4, max(w, h)/25)

	inland := func(x, y, r int) bool {
		d := r * 7 / 10
		for _, o := range [8][2]int{{-r, 0}, {r, 0}, {0, -r}, {0, r}, {-d, -d}, {d, -d}, {-d, d}, {d, d}} {
			nx, ny := x+o[0], y+o[1]
			if nx < 0 || ny < 0 || nx >= w || ny >= h || !land[ny*w+nx] {
				return false
			}
		}
		return true
	}
	nearestLand := func(i int) (int, bool) {
		x0, y0 := i%w, i/w
		hit, hitDist := -1, reach*reach+1
		for y := max(0, y0-reach); y <= min(h-1, y0+reach); y++ {
			for x := max(0, x0-reach); x <= min(w-1, x0+reach); x++ {
				if d := (x-x0)*(x-x0) + (y-y0)*(y-y0); land[y*w+x] && d < hitDist {
					hit, hitDist = y*w+x, d
				}
			}
		}
		if hit < 0 {
			return -1, false
		}

		hx, hy := hit%w, hit/w
		seen := map[int]bool{hit: true}
		queue, fallback := []int{hit}, -1
		for len(queue) > 0 {
			j := queue[0]
			queue = queue[1:]
			x, y := j%w, j/w
			if !dark[j] {
				if inland(x, y, 8) {
					return j, true
				}
				if fallback < 0 && inland(x, y, 3) {
					fallback = j
				}
			}
			for _, n := range [4]int{j - 1, j + 1, j - w, j + w} {
				nx, ny := n%w, n/w
				if n < 0 || n >= len(land) || (n == j-1 && x == 0) || (n == j+1 && x == w-1) || seen[n] || !land[n] ||
					(nx-hx)*(nx-hx)+(ny-hy)*(ny-hy) > reach*reach {
					continue
				}
				seen[n] = true
				queue = append(queue, n)
			}
		}
		if fallback >= 0 {
			return fallback, true
		}
		return hit, true
	}
	for k, seed := range seeds {
		tag := strings.ToUpper(strings.TrimSpace(seed.Tag))
		if _, ok := tagIndex[tag]; !ok {
			tagIndex[tag] = len(result.Tags)
			result.Tags = append(result.Tags, tag)
		}
		seedTag[k] = tagIndex[tag]
		if seed.At.X < 0 || seed.At.Y < 0 || seed.At.X >= src.W || seed.At.Y >= src.H {
			return nil, fmt.Errorf("%s 的位置 (%d, %d) 不在图上(图是 %d×%d)", tag, seed.At.X, seed.At.Y, src.W, src.H)
		}
		i := toWork(seed.At)
		if !land[i] {
			if j, ok := nearestLand(i); ok {
				moved := toSource(j)
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"%s 的位置 (%d, %d) 在水上,挪到了最近的陆地 (%d, %d):看预览确认那里是它的地方", tag, seed.At.X, seed.At.Y, moved.X, moved.Y))
				seed.At, seeds[k].At, i = moved, moved, j
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s 的位置 (%d, %d) 在水上,附近也没有陆地:它只能从海上长到最近的陆地", tag, seed.At.X, seed.At.Y))
			}
		}
		seedAt[k] = i
		c, _ := src.SampleColour(seed.At, opt.Dark)
		seedColour[k] = c
		if dark[i] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s 的位置 (%d, %d) 落在线条或文字上,颜色取自旁边的 %s", tag, seed.At.X, seed.At.Y, c.Hex()))
		}
	}
	if len(result.Tags) > 255 {
		return nil, errors.New("国家太多了")
	}

	owner := make([]int16, len(work.Pix))
	best := make([]float32, len(work.Pix))
	for i := range owner {
		owner[i] = -1
		best[i] = float32(math.Inf(1))
	}
	var queue growHeap
	for k := range seeds {
		i := seedAt[k]
		if best[i] > 0 {
			best[i] = 0
			queue.push(growItem{cost: 0, at: int32(i), seed: int16(k)})
		}
	}
	step := func(i int, seed int16) float32 {
		switch {
		case !land[i]:
			return float32(opt.SeaCost)
		case dark[i]:
			return 2
		}
		excess := Distance(work.Pix[i], seedColour[seed]) - opt.Tolerance
		if excess <= 0 {
			return 1
		}
		return float32(1 + excess*0.6)
	}
	for queue.len() > 0 {
		item := queue.pop()
		i := int(item.at)
		if owner[i] >= 0 {
			continue
		}
		owner[i] = item.seed
		x, y := i%w, i/w
		for _, j := range [4]int{i - 1, i + 1, i - w, i + w} {
			if (j == i-1 && x == 0) || (j == i+1 && x == w-1) || (j == i-w && y == 0) || (j == i+w && y == h-1) {
				continue
			}
			if owner[j] >= 0 {
				continue
			}
			cost := item.cost + step(j, item.seed)
			if cost < best[j] {
				best[j] = cost
				queue.push(growItem{cost: cost, at: int32(j), seed: item.seed})
			}
		}
	}

	result.tagOf = make([]uint8, len(work.Pix))
	for i := range owner {
		if land[i] && owner[i] >= 0 {
			result.tagOf[i] = uint8(seedTag[owner[i]] + 1)
		}
	}

	touch := map[[2]uint8]bool{}
	for i, t := range result.tagOf {
		if t == 0 {
			continue
		}
		x := i % w
		if x < w-1 {
			if u := result.tagOf[i+1]; u != 0 && u != t {
				touch[[2]uint8{min(t, u), max(t, u)}] = true
			}
		}
		if i+w < len(result.tagOf) {
			if u := result.tagOf[i+w]; u != 0 && u != t {
				touch[[2]uint8{min(t, u), max(t, u)}] = true
			}
		}
	}
	warned := map[[2]uint8]bool{}
	var alike []string
	for a := range seeds {
		for b := a + 1; b < len(seeds); b++ {
			ta, tb := uint8(seedTag[a]+1), uint8(seedTag[b]+1)
			pair := [2]uint8{min(ta, tb), max(ta, tb)}
			if ta == tb || warned[pair] || !touch[pair] || Distance(seedColour[a], seedColour[b]) >= opt.Tolerance {
				continue
			}
			warned[pair] = true
			alike = append(alike, fmt.Sprintf("%s 与 %s 相邻而颜色几乎一样(%s / %s)",
				result.Tags[seedTag[a]], result.Tags[seedTag[b]], seedColour[a].Hex(), seedColour[b].Hex()))
		}
	}
	const byDistance = "它们之间的边界按到各个位置的远近划,而不是按图;不对的话在边界两侧各加几个位置"
	switch {
	case len(alike) > 3:

		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"有 %d 对相邻的国家颜色几乎一样(这张图没有用颜色区分国家):%s;要贴着图上画的边界,就沿边界两侧多加位置",
			len(alike), byDistance))
	default:
		for _, pair := range alike {
			result.Warnings = append(result.Warnings, pair+":"+byDistance)
		}
	}

	type acc struct {
		n                      int
		sumX, sumY             float64
		minX, minY, maxX, maxY int
	}
	accs := make([]acc, len(result.Tags))
	for t := range accs {
		accs[t].minX, accs[t].minY = w, h
	}
	for i, t := range result.tagOf {
		if t == 0 {
			continue
		}
		a := &accs[t-1]
		x, y := i%w, i/w
		a.n++
		a.sumX += float64(x)
		a.sumY += float64(y)
		a.minX, a.minY = min(a.minX, x), min(a.minY, y)
		a.maxX, a.maxY = max(a.maxX, x), max(a.maxY, y)
	}
	_, starts, members := components(w, h, func(a, b int) bool { return result.tagOf[a] == result.tagOf[b] })
	parts := make([]int, len(result.Tags))
	speck := max(4, landCount/2000)
	for c := 0; c+1 < len(starts); c++ {
		size := starts[c+1] - starts[c]
		if t := result.tagOf[members[starts[c]]]; t > 0 && size >= speck {
			parts[t-1]++
		}
	}
	sx, sy := float64(src.W)/float64(w), float64(src.H)/float64(h)
	for t, tag := range result.Tags {
		a := accs[t]
		region := Region{Tag: tag, Colours: []string{}, Seeds: []Point{}, Parts: parts[t]}
		for k, seed := range seeds {
			if seedTag[k] == t {
				region.Seeds = append(region.Seeds, seed.At)
				hex := seedColour[k].Hex()
				if !contains(region.Colours, hex) {
					region.Colours = append(region.Colours, hex)
				}
			}
		}
		if a.n > 0 {
			region.Share = round4(float64(a.n) / float64(landCount))
			region.Box = Rect{X: int(float64(a.minX) * sx), Y: int(float64(a.minY) * sy), W: int(float64(a.maxX-a.minX+1) * sx), H: int(float64(a.maxY-a.minY+1) * sy)}
			region.Centre = Point{X: int((a.sumX/float64(a.n) + 0.5) * sx), Y: int((a.sumY/float64(a.n) + 0.5) * sy)}
		}
		if region.Share < 0.003 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s 只分到 %.1f%% 的陆地:位置可能点错了,或者被旁边颜色相近的国家吞了", tag, region.Share*100))
		}
		result.Regions = append(result.Regions, region)
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	return result, nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func (r *RegionResult) TagAt(p Point) uint8 {
	w, h := r.work.W, r.work.H
	x := min(w-1, max(0, p.X*w/r.srcW))
	y := min(h-1, max(0, p.Y*h/r.srcH))
	return r.tagOf[y*w+x]
}

func (r *RegionResult) Owners(placement Placement) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, placement.CanvasW, placement.CanvasH))
	for cy := 0; cy < placement.CanvasH; cy++ {
		for cx := 0; cx < placement.CanvasW; cx++ {
			o := out.PixOffset(cx, cy)
			out.Pix[o+3] = 255
			if s, ok := placement.SourceOf(cx, cy); ok {
				out.Pix[o] = r.TagAt(s)
			}
		}
	}
	return out
}

func (r *RegionResult) Preview(maxSide int, seeds []Seed) *image.NRGBA {
	q := r.work.Scaled(maxSide)
	out := q.Image()
	w, h := q.W, q.H
	tag := func(x, y int) uint8 {
		return r.TagAt(Point{X: int((float64(x) + 0.5) * float64(r.srcW) / float64(w)), Y: int((float64(y) + 0.5) * float64(r.srcH) / float64(h))})
	}
	tags := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tags[y*w+x] = tag(x, y)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			o := i * 4
			out.Pix[o+3] = 255
			t := tags[i]
			if t == 0 {

				out.Pix[o], out.Pix[o+1], out.Pix[o+2] = out.Pix[o]/3+30, out.Pix[o+1]/3+50, out.Pix[o+2]/3+90
				continue
			}
			border := (x > 0 && tags[i-1] != t) || (x < w-1 && tags[i+1] != t) || (y > 0 && tags[i-w] != t) || (y < h-1 && tags[i+w] != t)
			if border {
				out.Pix[o], out.Pix[o+1], out.Pix[o+2] = 0, 0, 0
				continue
			}
			c := TagColour(int(t) - 1)
			for ch := 0; ch < 3; ch++ {
				out.Pix[o+ch] = uint8((int(c[ch])*6 + int(out.Pix[o+ch])*4) / 10)
			}
		}
	}
	marks := make([]Mark, 0, len(seeds))
	for _, s := range seeds {
		t := strings.ToUpper(strings.TrimSpace(s.Tag))
		n := 0
		for k, v := range r.Tags {
			if v == t {
				n = k
			}
		}
		marks = append(marks, Mark{At: s.At, Label: t, Colour: TagColour(n)})
	}
	view := Rect{W: r.srcW, H: r.srcH}
	drawGrid(out, view)
	drawMarks(out, view, marks)
	return out
}

type growItem struct {
	cost float32
	at   int32
	seed int16
}

type growHeap struct{ items []growItem }

func (q *growHeap) len() int { return len(q.items) }

func (q *growHeap) push(it growItem) {
	q.items = append(q.items, it)
	i := len(q.items) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if q.items[parent].cost <= q.items[i].cost {
			break
		}
		q.items[parent], q.items[i] = q.items[i], q.items[parent]
		i = parent
	}
}

func (q *growHeap) pop() growItem {
	top := q.items[0]
	last := len(q.items) - 1
	q.items[0] = q.items[last]
	q.items = q.items[:last]
	i := 0
	for {
		l, r := 2*i+1, 2*i+2
		smallest := i
		if l < len(q.items) && q.items[l].cost < q.items[smallest].cost {
			smallest = l
		}
		if r < len(q.items) && q.items[r].cost < q.items[smallest].cost {
			smallest = r
		}
		if smallest == i {
			break
		}
		q.items[i], q.items[smallest] = q.items[smallest], q.items[i]
		i = smallest
	}
	return top
}
