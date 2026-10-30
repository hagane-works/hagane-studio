package mapsketch

import (
	"errors"
	"image"
	"math"

	"haganeworks/internal/hoi4/mapgen"
)

const (
	DefaultCanvasW = 2816
	DefaultCanvasH = 1024

	workSide = 2048
)

const Ignore = "ignore"

type Rule struct {
	Colour    RGB
	Class     string
	Tolerance float64
}

type Area struct {
	Box   Rect
	Class string
}

type TerrainOptions struct {
	Rules []Rule

	Areas []Area

	Unmatched string

	Dark float64

	MinPatch float64

	Smooth int

	Crop Rect

	Margin float64

	CanvasW, CanvasH int
}

type Placement struct {
	Crop    Rect    `json:"crop"`
	Scale   float64 `json:"scale"`
	OffsetX int     `json:"offset_x"`
	OffsetY int     `json:"offset_y"`
	CanvasW int     `json:"canvas_width"`
	CanvasH int     `json:"canvas_height"`
}

func Place(crop Rect, canvasW, canvasH int, margin float64) Placement {
	scale := math.Min(float64(canvasW)*(1-2*margin)/float64(crop.W), float64(canvasH)*(1-2*margin)/float64(crop.H))
	w := int(math.Round(float64(crop.W) * scale))
	h := int(math.Round(float64(crop.H) * scale))
	return Placement{Crop: crop, Scale: scale, OffsetX: (canvasW - w) / 2, OffsetY: (canvasH - h) / 2, CanvasW: canvasW, CanvasH: canvasH}
}

func (p Placement) SourceOf(cx, cy int) (Point, bool) {
	sx := int(math.Floor((float64(cx-p.OffsetX) + 0.5) / p.Scale))
	sy := int(math.Floor((float64(cy-p.OffsetY) + 0.5) / p.Scale))
	if sx < 0 || sy < 0 || sx >= p.Crop.W || sy >= p.Crop.H {
		return Point{}, false
	}
	return Point{X: p.Crop.X + sx, Y: p.Crop.Y + sy}, true
}

func (p Placement) CanvasOf(s Point) (Point, bool) {
	if s.X < p.Crop.X || s.Y < p.Crop.Y || s.X >= p.Crop.X+p.Crop.W || s.Y >= p.Crop.Y+p.Crop.H {
		return Point{}, false
	}
	return Point{
		X: min(p.CanvasW-1, p.OffsetX+int((float64(s.X-p.Crop.X)+0.5)*p.Scale)),
		Y: min(p.CanvasH-1, p.OffsetY+int((float64(s.Y-p.Crop.Y)+0.5)*p.Scale)),
	}, true
}

type RuleUse struct {
	Colour    string  `json:"colour"`
	Class     string  `json:"class"`
	Share     float64 `json:"share"`
	Tolerance float64 `json:"tolerance"`
}

type TerrainResult struct {
	Sketch    *image.NRGBA
	Placement Placement

	Shares map[string]float64

	Filled float64

	Unmatched float64
	Rules     []RuleUse
}

const (
	labelIgnore = 255
)

func Terrain(src *Picture, opt TerrainOptions) (*TerrainResult, error) {
	if opt.CanvasW <= 0 || opt.CanvasH <= 0 {
		opt.CanvasW, opt.CanvasH = DefaultCanvasW, DefaultCanvasH
	}
	if opt.Margin <= 0 {
		opt.Margin = 0.03
	}
	if opt.Margin > 0.4 {
		return nil, errors.New("留边太大:最多 0.4")
	}
	if opt.Dark == 0 {
		opt.Dark = DefaultDark
	}
	if opt.MinPatch <= 0 {
		opt.MinPatch = 0.0002
	}
	if opt.Smooth == 0 {
		opt.Smooth = 3
	}
	if opt.Smooth > 8 {
		return nil, errors.New("平滑半径最多 8")
	}
	crop := opt.Crop
	if crop.empty() {
		crop = Rect{W: src.W, H: src.H}
	}
	area := src.Crop(crop)
	if area.W < 8 || area.H < 8 {
		return nil, errors.New("裁出来的地图太小了")
	}
	crop = Rect{X: max(0, crop.X), Y: max(0, crop.Y), W: area.W, H: area.H}
	work := area.Scaled(workSide)

	unmatched := mapgen.Class(0)
	unmatchedMode := "nearest"
	switch opt.Unmatched {
	case "", "nearest":
	case Ignore:
		unmatchedMode = Ignore
	default:
		entry, ok := TerrainClass(opt.Unmatched)
		if !ok {
			return nil, errors.New("没有这种地形:" + opt.Unmatched)
		}
		unmatchedMode, unmatched = "class", entry.Class
	}
	ruleClass := make([]uint8, len(opt.Rules))
	for i, rule := range opt.Rules {
		if rule.Class == Ignore {
			ruleClass[i] = labelIgnore
			continue
		}
		entry, ok := TerrainClass(rule.Class)
		if !ok {
			return nil, errors.New("没有这种地形:" + rule.Class)
		}
		ruleClass[i] = uint8(entry.Class)
		if opt.Rules[i].Tolerance <= 0 {
			opt.Rules[i].Tolerance = 40
		}
	}

	n := len(work.Pix)

	forced := make([]uint8, n)
	areaClass := make([]uint8, len(opt.Areas))
	fx, fy := float64(work.W)/float64(crop.W), float64(work.H)/float64(crop.H)
	for k, area := range opt.Areas {
		if area.Class == Ignore {
			areaClass[k] = labelIgnore
		} else {
			entry, ok := TerrainClass(area.Class)
			if !ok {
				return nil, errors.New("没有这种地形:" + area.Class)
			}
			areaClass[k] = uint8(entry.Class)
		}
		x0 := max(0, int(float64(area.Box.X-crop.X)*fx))
		y0 := max(0, int(float64(area.Box.Y-crop.Y)*fy))
		x1 := min(work.W, int(math.Ceil(float64(area.Box.X+area.Box.W-crop.X)*fx)))
		y1 := min(work.H, int(math.Ceil(float64(area.Box.Y+area.Box.H-crop.Y)*fy)))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				forced[y*work.W+x] = uint8(min(k+1, 255))
			}
		}
	}
	labels := make([]uint8, n)
	ruleCount := make([]int, len(opt.Rules))
	content, filled, unclaimed := 0, 0, 0
	for i, c := range work.Pix {
		if work.clear(i) {
			labels[i] = uint8(mapgen.Sea)
			continue
		}
		content++
		if a := forced[i]; a > 0 {
			labels[i] = areaClass[a-1]
			if labels[i] == labelIgnore {
				filled++
			}
			continue
		}
		best, bestD := -1, math.MaxFloat64
		for k, rule := range opt.Rules {
			if d := Distance(c, rule.Colour); d <= rule.Tolerance && d < bestD {
				best, bestD = k, d
			}
		}
		switch {
		case best >= 0:
			labels[i] = ruleClass[best]
			ruleCount[best]++
		case opt.Dark > 0 && luminance(c) < opt.Dark:
			labels[i] = labelIgnore
		default:
			unclaimed++
			switch unmatchedMode {
			case Ignore:
				labels[i] = labelIgnore
			case "class":
				labels[i] = uint8(unmatched)
			default:
				entry, _ := NearestTerrain(c)
				labels[i] = uint8(entry.Class)
			}
		}
		if labels[i] == labelIgnore {
			filled++
		}
	}
	if filled == n {
		return nil, errors.New("整张图都被当成了线条或要忽略的颜色,没有可用的地形")
	}
	fillFromAround(labels, work.W, work.H, func(l uint8) bool { return l == labelIgnore })
	if opt.Smooth > 0 {
		labels = majority(labels, work.W, work.H, opt.Smooth, forced)
	}
	removeSpecks(labels, work.W, work.H, max(4, int(opt.MinPatch*float64(n))))

	placement := Place(crop, opt.CanvasW, opt.CanvasH, opt.Margin)
	sketch := image.NewNRGBA(image.Rect(0, 0, opt.CanvasW, opt.CanvasH))
	sea := mapgen.Legend[mapgen.Sea].Color
	for i := 0; i < opt.CanvasW*opt.CanvasH; i++ {
		o := i * 4
		sketch.Pix[o], sketch.Pix[o+1], sketch.Pix[o+2], sketch.Pix[o+3] = sea[0], sea[1], sea[2], 255
	}
	for cy := 0; cy < opt.CanvasH; cy++ {
		for cx := 0; cx < opt.CanvasW; cx++ {
			s, ok := placement.SourceOf(cx, cy)
			if !ok {
				continue
			}
			wx := min(work.W-1, int(float64(s.X-crop.X)*fx))
			wy := min(work.H-1, int(float64(s.Y-crop.Y)*fy))
			c := mapgen.Legend[labels[wy*work.W+wx]].Color
			o := sketch.PixOffset(cx, cy)
			sketch.Pix[o], sketch.Pix[o+1], sketch.Pix[o+2] = c[0], c[1], c[2]
		}
	}

	shares := map[string]float64{}
	counts := make([]int, len(mapgen.Legend))
	for _, l := range labels {
		counts[l]++
	}
	for k, c := range counts {
		if c > 0 {
			shares[mapgen.Legend[k].Key] = round4(float64(c) / float64(n))
		}
	}
	uses := make([]RuleUse, len(opt.Rules))
	for i, rule := range opt.Rules {
		uses[i] = RuleUse{Colour: rule.Colour.Hex(), Class: rule.Class, Share: round4(float64(ruleCount[i]) / float64(max(1, content))), Tolerance: rule.Tolerance}
	}
	return &TerrainResult{
		Sketch: sketch, Placement: placement, Shares: shares,
		Filled:    round4(float64(filled) / float64(max(1, content))),
		Unmatched: round4(float64(unclaimed) / float64(max(1, content))),
		Rules:     uses,
	}, nil
}

func majority(labels []uint8, w, h, r int, forced []uint8) []uint8 {
	out := make([]uint8, len(labels))
	var count [256]int32
	for y := 0; y < h; y++ {
		y0, y1 := max(0, y-r), min(h-1, y+r)
		for x := 0; x < w; x++ {
			i := y*w + x
			if forced[i] > 0 {
				out[i] = labels[i]
				continue
			}
			x0, x1 := max(0, x-r), min(w-1, x+r)
			best, bestN := labels[i], int32(0)
			var seen [16]uint8
			nSeen := 0
			for yy := y0; yy <= y1; yy++ {
				row := yy * w
				for xx := x0; xx <= x1; xx++ {
					l := labels[row+xx]
					if count[l] == 0 && nSeen < len(seen) {
						seen[nSeen] = l
						nSeen++
					}
					count[l]++
				}
			}
			own := count[labels[i]]
			bestN = own
			for k := 0; k < nSeen; k++ {
				l := seen[k]
				if count[l] > bestN {
					best, bestN = l, count[l]
				}
				count[l] = 0
			}
			out[i] = best
		}
	}
	return out
}

func fillFromAround(labels []uint8, w, h int, hole func(uint8) bool) {
	queue := make([]int32, 0, len(labels)/4)
	for i, l := range labels {
		if hole(l) {
			continue
		}
		x, y := i%w, i/w
		if (x > 0 && hole(labels[i-1])) || (x < w-1 && hole(labels[i+1])) ||
			(y > 0 && hole(labels[i-w])) || (y < h-1 && hole(labels[i+w])) {
			queue = append(queue, int32(i))
		}
	}
	for head := 0; head < len(queue); head++ {
		i := int(queue[head])
		x, y := i%w, i/w
		for _, j := range [4]int{i - 1, i + 1, i - w, i + w} {
			if (j == i-1 && x == 0) || (j == i+1 && x == w-1) || (j == i-w && y == 0) || (j == i+w && y == h-1) {
				continue
			}
			if hole(labels[j]) {
				labels[j] = labels[i]
				queue = append(queue, int32(j))
			}
		}
	}
}

func removeSpecks(labels []uint8, w, h, minSize int) {
	for pass := 0; pass < 3; pass++ {
		comp, starts, members := components(w, h, func(a, b int) bool { return labels[a] == labels[b] })
		changed := false
		for c := 0; c+1 < len(starts); c++ {
			pixels := members[starts[c]:starts[c+1]]
			if len(pixels) >= minSize {
				continue
			}
			var votes [256]int
			for _, p := range pixels {
				i := int(p)
				x, y := i%w, i/w
				for _, j := range [4]int{i - 1, i + 1, i - w, i + w} {
					if (j == i-1 && x == 0) || (j == i+1 && x == w-1) || (j == i-w && y == 0) || (j == i+w && y == h-1) {
						continue
					}
					if comp[j] != int32(c) {
						votes[labels[j]]++
					}
				}
			}
			best, bestVotes := -1, 0
			for l, v := range votes {
				if v > bestVotes {
					best, bestVotes = l, v
				}
			}
			if best < 0 || uint8(best) == labels[pixels[0]] {
				continue
			}
			for _, p := range pixels {
				labels[p] = uint8(best)
			}
			changed = true
		}
		if !changed {
			return
		}
	}
}

func components(w, h int, same func(a, b int) bool) (comp []int32, starts []int, members []int32) {
	n := w * h
	comp = make([]int32, n)
	for i := range comp {
		comp[i] = -1
	}
	members = make([]int32, 0, n)
	starts = []int{0}
	for i := 0; i < n; i++ {
		if comp[i] >= 0 {
			continue
		}
		id := int32(len(starts) - 1)
		comp[i] = id
		head := len(members)
		members = append(members, int32(i))
		for ; head < len(members); head++ {
			p := int(members[head])
			x, y := p%w, p/w
			try := func(q int) {
				if comp[q] < 0 && same(p, q) {
					comp[q] = id
					members = append(members, int32(q))
				}
			}
			if x > 0 {
				try(p - 1)
			}
			if x < w-1 {
				try(p + 1)
			}
			if y > 0 {
				try(p - w)
			}
			if y < h-1 {
				try(p + w)
			}
		}
		starts = append(starts, len(members))
	}
	return comp, starts, members
}
