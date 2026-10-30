package states

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/bmp"
)

type stateShape struct {
	Rings    [][][2]int `json:"rings"`
	BBox     [4]int     `json:"bbox"`
	Centroid [2]int     `json:"centroid"`
}

const outlineTolerance = 0.35

func parseDefinitionColors(csv []byte) map[uint32]int32 {
	out := map[uint32]int32{}
	text := strings.TrimPrefix(string(csv), "\ufeff")
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		parts := strings.Split(strings.TrimRight(line, "\r"), ";")
		if len(parts) < 4 || parts[0] == "" || strings.Trim(parts[0], "0123456789") != "" {
			continue
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		var rgb [3]int
		ok := true
		for i := range rgb {
			rgb[i], err = strconv.Atoi(strings.TrimSpace(parts[1+i]))
			ok = ok && err == nil
		}
		if !ok {
			continue
		}
		out[uint32(rgb[0]<<16|rgb[1]<<8|rgb[2])] = int32(id)
	}
	return out
}

func loadProvinces(provinces, definitions []byte) (width, height int, province []int32, err error) {
	img, err := bmp.Decode(bytes.NewReader(provinces))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("provinces.bmp: %w", err)
	}
	colors := parseDefinitionColors(definitions)
	if len(colors) == 0 {
		return 0, 0, nil, errors.New("definition.csv: no province rows")
	}
	bounds := img.Bounds()
	width, height = bounds.Dx(), bounds.Dy()
	province = make([]int32, width*height)
	if rgba, ok := img.(*image.RGBA); ok {
		lastRGB, lastID := uint32(1<<31), int32(0)
		for p := range province {
			pix := rgba.Pix[(p/width)*rgba.Stride+(p%width)*4:]
			rgb := uint32(pix[0])<<16 | uint32(pix[1])<<8 | uint32(pix[2])
			if rgb != lastRGB {
				lastRGB, lastID = rgb, colors[rgb]
			}
			province[p] = lastID
		}
		return width, height, province, nil
	}
	for p := range province {
		r, g, b, _ := img.At(bounds.Min.X+p%width, bounds.Min.Y+p/width).RGBA()
		province[p] = colors[uint32(r>>8)<<16|uint32(g>>8)<<8|uint32(b>>8)]
	}
	return width, height, province, nil
}

func stateOutlines(width, height int, province []int32, factor int, states map[int][]int) map[int]stateShape {
	if factor < 1 {
		factor = 1
	}
	most := 0
	for _, list := range states {
		for _, id := range list {
			most = max(most, id)
		}
	}
	stateOf := make([]int32, most+1)
	for state, list := range states {
		for _, id := range list {
			if id > 0 {
				stateOf[id] = int32(state)
			}
		}
	}
	gridW, gridH := (width+factor-1)/factor, (height+factor-1)/factor
	at := func(x, y int) int32 {
		id := province[y*factor*width+x*factor]
		if id <= 0 || int(id) >= len(stateOf) {
			return 0
		}
		return stateOf[id]
	}
	type extent struct {
		x0, y0, x1, y1 int
		sumX, sumY     int64
		count          int64
	}
	extents := map[int32]*extent{}
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			state := at(x, y)
			if state == 0 {
				continue
			}
			e := extents[state]
			if e == nil {
				e = &extent{x0: x, y0: y, x1: x, y1: y}
				extents[state] = e
			}
			e.x0, e.y0, e.x1, e.y1 = min(e.x0, x), min(e.y0, y), max(e.x1, x), max(e.y1, y)
			e.sumX += int64(x)
			e.sumY += int64(y)
			e.count++
		}
	}
	out := map[int]stateShape{}
	for state, e := range extents {
		w, h := e.x1-e.x0+1, e.y1-e.y0+1
		mask := make([]bool, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				mask[y*w+x] = at(e.x0+x, e.y0+y) == state
			}
		}
		var rings [][][2]int
		for _, ring := range boundaryRings(mask, w, h) {
			simple := simplifyRing(ring, outlineTolerance)
			if len(simple) < 3 {
				continue
			}
			placed := make([][2]int, len(simple))
			for i, p := range simple {
				placed[i] = [2]int{p[0] + e.x0, p[1] + e.y0}
			}
			rings = append(rings, placed)
		}
		if len(rings) == 0 {
			continue
		}
		out[int(state)] = stateShape{
			Rings: rings,
			BBox:  [4]int{e.x0, e.y0, e.x1 + 1, e.y1 + 1},
			Centroid: [2]int{
				int(math.RoundToEven(float64(e.sumX) / float64(e.count))),
				int(math.RoundToEven(float64(e.sumY) / float64(e.count))),
			},
		}
	}
	return out
}

func boundaryRings(mask []bool, w, h int) [][][2]int {
	in := func(x, y int) bool { return x >= 0 && y >= 0 && x < w && y < h && mask[y*w+x] }
	type corner struct {
		at   [2]int
		outs [][2]int
		done bool
	}
	var corners []corner
	index := map[[2]int]int{}
	add := func(a, b [2]int) {
		i, ok := index[a]
		if !ok {
			i = len(corners)
			index[a] = i
			corners = append(corners, corner{at: a})
		}
		corners[i].outs = append(corners[i].outs, b)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if in(x, y) && !in(x, y-1) {
				add([2]int{x, y}, [2]int{x + 1, y})
			}
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if in(x, y) && !in(x, y+1) {
				add([2]int{x + 1, y + 1}, [2]int{x, y + 1})
			}
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if in(x, y) && !in(x-1, y) {
				add([2]int{x, y + 1}, [2]int{x, y})
			}
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if in(x, y) && !in(x+1, y) {
				add([2]int{x + 1, y}, [2]int{x + 1, y + 1})
			}
		}
	}
	var rings [][][2]int
	next := 0
	for {
		for next < len(corners) && corners[next].done {
			next++
		}
		if next == len(corners) {
			break
		}
		start := corners[next].at
		ring := [][2]int{start}
		current := start
		for {
			c := &corners[index[current]]
			to := c.outs[len(c.outs)-1]
			c.outs = c.outs[:len(c.outs)-1]
			if len(c.outs) == 0 {
				c.done = true
			}
			if to == start {
				break
			}
			ring = append(ring, to)
			current = to
		}
		var slim [][2]int
		n := len(ring)
		for i := range ring {
			a, b, c := ring[(i+n-1)%n], ring[i], ring[(i+1)%n]
			if (a[0] == b[0] && b[0] == c[0]) || (a[1] == b[1] && b[1] == c[1]) {
				continue
			}
			slim = append(slim, b)
		}
		if len(slim) >= 3 {
			rings = append(rings, slim)
		}
	}
	return rings
}

func simplifyRing(points [][2]int, tolerance float64) [][2]int {
	if len(points) <= 4 {
		return points
	}
	low, high := 0, 0
	for i, p := range points {
		if p[0] < points[low][0] {
			low = i
		}
		if p[0] > points[high][0] {
			high = i
		}
	}
	if low > high {
		low, high = high, low
	}
	var open func(pts [][2]int) [][2]int
	open = func(pts [][2]int) [][2]int {
		if len(pts) <= 2 {
			return pts
		}
		ax, ay := pts[0][0], pts[0][1]
		bx, by := pts[len(pts)-1][0], pts[len(pts)-1][1]
		dx, dy := bx-ax, by-ay
		norm := math.Sqrt(float64(dx*dx + dy*dy))
		if norm == 0 {
			norm = 1e-9
		}
		most, at := -1.0, 0
		for i := 1; i < len(pts)-1; i++ {
			px, py := pts[i][0], pts[i][1]
			cross := dx*(ay-py) - dy*(ax-px)
			if cross < 0 {
				cross = -cross
			}
			if d := float64(cross) / norm; d > most {
				most, at = d, i
			}
		}
		if most <= tolerance {
			return [][2]int{pts[0], pts[len(pts)-1]}
		}
		left := open(pts[:at+1])
		right := open(pts[at:])
		return append(append([][2]int(nil), left[:len(left)-1]...), right...)
	}
	first := open(points[low : high+1])
	wrapped := append(append([][2]int(nil), points[high:]...), points[:low+1]...)
	second := open(wrapped)
	ring := append(append([][2]int(nil), first[:len(first)-1]...), second[:len(second)-1]...)
	if len(ring) < 3 {
		return points
	}
	return ring
}
