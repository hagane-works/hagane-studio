package mapsketch

import (
	"fmt"
	"math"
)

type ColourSeedOptions struct {
	Colour    RGB
	Tolerance float64
	Box       *Rect
	At        *Point
	Land      func(Point) bool
}

type ColourSeedCandidate struct {
	Box          Rect   `json:"box"`
	Pixels       int    `json:"pixels"`
	SourceColour string `json:"source_colour"`
}

type ColourSeedResult struct {
	RequestedColour   string                `json:"requested_colour"`
	SourceColour      string                `json:"source_colour,omitempty"`
	ActualSeed        Point                 `json:"actual_seed"`
	Tolerance         float64               `json:"colour_tolerance"`
	SourceMatchPixels int                   `json:"source_match_pixels"`
	MatchedLandPixels int                   `json:"matched_land_pixels"`
	SelectedPixels    int                   `json:"selected_region_pixels"`
	MatchShare        float64               `json:"match_share"`
	SearchBox         Rect                  `json:"search_box"`
	OtherRegions      int                   `json:"other_colour_regions"`
	Candidates        []ColourSeedCandidate `json:"candidates,omitempty"`
	Warnings          []string              `json:"warnings"`
}

func FindColourSeed(src *Picture, opt ColourSeedOptions) (ColourSeedResult, error) {
	result := ColourSeedResult{RequestedColour: opt.Colour.Hex(), Tolerance: opt.Tolerance, Warnings: []string{}}
	if result.Tolerance == 0 {
		result.Tolerance = 20
	}
	if result.Tolerance < 0 || result.Tolerance > 100 || math.IsNaN(result.Tolerance) {
		return result, fmt.Errorf("colour_tolerance must be greater than 0 and at most 100")
	}
	box := Rect{W: src.W, H: src.H}
	if opt.Box != nil {
		box = *opt.Box
	}
	result.SearchBox = box
	if box.X < 0 || box.Y < 0 || box.W < 1 || box.H < 1 || box.X > src.W-box.W || box.Y > src.H-box.H {
		return result, fmt.Errorf("colour seed box must lie inside the source image %d×%d", src.W, src.H)
	}
	if opt.At != nil && (opt.At.X < box.X || opt.At.Y < box.Y || opt.At.X >= box.X+box.W || opt.At.Y >= box.Y+box.H) {
		return result, fmt.Errorf("colour seed at hint must lie inside its source search box")
	}

	labels := make([]int32, box.W*box.H)
	for y := 0; y < box.H; y++ {
		for x := 0; x < box.W; x++ {
			i := y*box.W + x
			labels[i] = -1
			point := Point{X: box.X + x, Y: box.Y + y}
			source := point.Y*src.W + point.X
			if src.clear(source) || Distance(src.Pix[source], opt.Colour) > result.Tolerance {
				continue
			}
			result.SourceMatchPixels++
			if opt.Land != nil && !opt.Land(point) {
				continue
			}
			labels[i] = 0
			result.MatchedLandPixels++
		}
	}
	result.MatchShare = float64(result.MatchedLandPixels) / float64(len(labels))
	if result.MatchedLandPixels == 0 {
		if result.SourceMatchPixels > 0 {
			return result, fmt.Errorf("colour %s matches %d source pixels but the terrain marks them all as water; correct the terrain sketch before splitting countries", result.RequestedColour, result.SourceMatchPixels)
		}
		return result, fmt.Errorf("colour %s matches no opaque source pixels in the search box; inspect the actual source colour or correct colour_tolerance/box", result.RequestedColour)
	}
	type region struct {
		ColourSeedCandidate
		near float64
	}
	regions := []region{}
	queue := []int{}
	for start, label := range labels {
		if label != 0 {
			continue
		}
		id := int32(len(regions) + 1)
		labels[start] = id
		queue = append(queue[:0], start)
		item := region{ColourSeedCandidate: ColourSeedCandidate{Box: Rect{X: box.X + start%box.W, Y: box.Y + start/box.W}}, near: math.Inf(1)}
		colourCounts := map[RGB]int{}
		maxX, maxY := item.Box.X, item.Box.Y
		for head := 0; head < len(queue); head++ {
			i := queue[head]
			x, y := i%box.W, i/box.W
			px, py := box.X+x, box.Y+y
			colourCounts[src.Pix[py*src.W+px]]++
			item.Box.X, item.Box.Y = min(item.Box.X, px), min(item.Box.Y, py)
			maxX, maxY = max(maxX, px), max(maxY, py)
			if opt.At != nil {
				dx, dy := float64(px-opt.At.X), float64(py-opt.At.Y)
				item.near = min(item.near, dx*dx+dy*dy)
			}
			visit := func(j int) {
				if labels[j] == 0 {
					labels[j] = id
					queue = append(queue, j)
				}
			}
			if x > 0 {
				visit(i - 1)
			}
			if x+1 < box.W {
				visit(i + 1)
			}
			if y > 0 {
				visit(i - box.W)
			}
			if y+1 < box.H {
				visit(i + box.W)
			}
		}
		item.Pixels, item.Box.W, item.Box.H = len(queue), maxX-item.Box.X+1, maxY-item.Box.Y+1
		most := 0
		for colour, count := range colourCounts {
			if count > most || (count == most && colour.Hex() < item.SourceColour) {
				most, item.SourceColour = count, colour.Hex()
			}
		}
		regions = append(regions, item)
	}
	selected := 0
	for i, item := range regions {
		if (opt.At != nil && item.near < regions[selected].near) || (opt.At == nil && item.Pixels > regions[selected].Pixels) {
			selected = i
		}
	}
	result.OtherRegions = len(regions) - 1
	if opt.At != nil {
		for i, item := range regions {
			if i != selected && item.near == regions[selected].near {
				return result, fmt.Errorf("at hint is equally close to different colour regions; supply a tighter box")
			}
		}
	}
	threshold := max(4, regions[selected].Pixels/10)
	if opt.At == nil {

		for _, item := range regions {
			if item.Pixels >= threshold {
				result.Candidates = append(result.Candidates, item.ColourSeedCandidate)
			}
		}
		if len(result.Candidates) > 1 {
			return result, fmt.Errorf("colour %s matches %d substantial disconnected regions; supply box or at to select this country's region (use several seeds for its islands); candidates=%v", result.RequestedColour, len(result.Candidates), result.Candidates)
		}
	}
	for i, item := range regions {
		if i != selected && item.Pixels >= threshold {
			result.Warnings = append(result.Warnings, "selected one of several substantial colour regions; inspect the preview and add colour+box/at seeds for any islands or exclaves")
			break
		}
	}
	at := deepest(box.W, box.H, func(i int) bool { return labels[i] == int32(selected+1) })
	result.ActualSeed = Point{X: box.X + at%box.W, Y: box.Y + at/box.W}
	result.SourceColour = src.Pix[result.ActualSeed.Y*src.W+result.ActualSeed.X].Hex()
	result.SelectedPixels = regions[selected].Pixels
	if opt.At != nil && regions[selected].near > 0 {
		result.Warnings = append(result.Warnings, "at was a location hint outside the matching colour; the actual seed is a verified deep interior source pixel")
	}
	return result, nil
}
