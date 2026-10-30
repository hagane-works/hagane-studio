package mapgen

import (
	"math"
	"math/rand"
	"sort"
)

const (
	stepStraight = 5
	stepDiagonal = 7
	bucketRing   = 8
)

var steps = [8]struct{ dx, dy, cost int }{
	{-1, 0, stepStraight}, {1, 0, stepStraight}, {0, -1, stepStraight}, {0, 1, stepStraight},
	{-1, -1, stepDiagonal}, {1, -1, stepDiagonal}, {-1, 1, stepDiagonal}, {1, 1, stepDiagonal},
}

func grow(width, height int, area []int32, seeds []int32) []int32 {
	owner := make([]int32, width*height)
	dist := make([]int32, width*height)
	for i := range dist {
		dist[i] = math.MaxInt32
		owner[i] = -1
	}
	var buckets [bucketRing][]int32
	pending := 0
	for k, p := range seeds {
		dist[p] = 0
		owner[p] = int32(k)
		buckets[0] = append(buckets[0], p)
		pending++
	}
	for d := int32(0); pending > 0; d++ {
		ring := &buckets[d%bucketRing]
		for len(*ring) > 0 {
			p := (*ring)[len(*ring)-1]
			*ring = (*ring)[:len(*ring)-1]
			pending--
			if dist[p] != d {
				continue
			}
			x, y := int(p)%width, int(p)/width
			for _, step := range steps {
				nx, ny := x+step.dx, y+step.dy
				if nx < 0 || ny < 0 || nx >= width || ny >= height {
					continue
				}
				q := int32(ny*width + nx)
				if area[q] != area[p] {
					continue
				}

				if step.dx != 0 && step.dy != 0 && area[int32(y*width+nx)] != area[p] && area[int32(ny*width+x)] != area[p] {
					continue
				}
				nd := d + int32(step.cost)
				if nd < dist[q] {
					dist[q] = nd
					owner[q] = owner[p]
					buckets[nd%bucketRing] = append(buckets[nd%bucketRing], q)
					pending++
				}
			}
		}
	}
	return owner
}

func relax(width int, owner []int32, seeds []int32) {
	sumX := make([]float64, len(seeds))
	sumY := make([]float64, len(seeds))
	count := make([]float64, len(seeds))
	for p, k := range owner {
		if k < 0 {
			continue
		}
		sumX[k] += float64(p % width)
		sumY[k] += float64(p / width)
		count[k]++
	}
	best := make([]float64, len(seeds))
	for k := range best {
		best[k] = math.MaxFloat64
	}
	for p, k := range owner {
		if k < 0 {
			continue
		}
		dx := float64(p%width) - sumX[k]/count[k]
		dy := float64(p/width) - sumY[k]/count[k]
		if d := dx*dx + dy*dy; d < best[k] {
			best[k] = d
			seeds[k] = int32(p)
		}
	}
}

func placeSeeds(area []int32, sizes []int, targets []int, rng *rand.Rand) []int32 {

	start := make([]int, len(sizes)+1)
	for a, size := range sizes {
		start[a+1] = start[a] + size
	}
	next := append([]int(nil), start[:len(sizes)]...)
	pixels := make([]int32, len(area))
	for p, a := range area {
		if a < 0 {
			continue
		}
		pixels[next[a]] = int32(p)
		next[a]++
	}
	var seeds []int32
	for a, size := range sizes {
		n := int(math.Round(float64(size) / float64(targets[a])))
		if n < 1 {
			n = 1
		}
		if n > size {
			n = size
		}
		chosen := map[int]bool{}
		for len(chosen) < n {
			chosen[start[a]+rng.Intn(size)] = true
		}
		keys := make([]int, 0, n)
		for i := range chosen {
			keys = append(keys, i)
		}
		sort.Ints(keys)
		for _, i := range keys {
			seeds = append(seeds, pixels[i])
		}
	}
	return seeds
}

func borders(width, height int, area, cell []int32) map[[2]int32]int {
	out := map[[2]int32]int{}
	add := func(p, q int) {
		a, b := cell[p], cell[q]
		if a == b || area[p] != area[q] {
			return
		}
		if a > b {
			a, b = b, a
		}
		out[[2]int32{a, b}]++
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := y*width + x
			if x+1 < width {
				add(p, p+1)
			}
			if y+1 < height {
				add(p, p+width)
			}
		}
	}
	return out
}

func absorb(width, height int, area, cell []int32, small func(c int32) bool) {
	shared := borders(width, height, area, cell)
	type best struct {
		with   int32
		length int
	}
	choice := map[int32]best{}
	consider := func(c, with int32, length int) {
		if !small(c) {
			return
		}
		if b, ok := choice[c]; !ok || length > b.length || (length == b.length && with < b.with) {
			choice[c] = best{with, length}
		}
	}
	for pair, length := range shared {
		consider(pair[0], pair[1], length)
		consider(pair[1], pair[0], length)
	}

	final := func(c int32) int32 {
		seen := map[int32]bool{}
		for {
			b, ok := choice[c]
			if !ok || seen[c] {
				return c
			}
			seen[c] = true
			c = b.with
		}
	}
	for p, c := range cell {
		if _, ok := choice[c]; ok {
			cell[p] = final(c)
		}
	}
}

func fixCrossings(width, height int, area, cell []int32) bool {
	for pass := 0; pass < 12; pass++ {
		changed, left := false, false
		for y := 0; y+1 < height; y++ {
			for x := 0; x+1 < width; x++ {
				p := y*width + x
				a, b, c, d := cell[p], cell[p+1], cell[p+width], cell[p+width+1]
				diagonal := a == d && b == c && a != b
				four := a != b && a != c && a != d && b != c && b != d && c != d
				if !diagonal && !four {
					continue
				}

				switch q := p + width + 1; {
				case area[q] == area[p+width]:
					cell[q] = c
					changed = true
				case area[q] == area[p+1]:
					cell[q] = b
					changed = true
				case area[p] == area[p+1]:
					cell[p] = b
					changed = true
				default:
					left = true
				}
			}
		}
		if !changed {
			return !left
		}
	}
	return false
}

func splitLarge(width int, cell []int32, maxW, maxH int) bool {
	type extent struct{ x0, y0, x1, y1 int }
	boxes := map[int32]*extent{}
	next := int32(0)
	for p, c := range cell {
		if c < 0 {
			continue
		}
		next = max(next, c+1)
		x, y := p%width, p/width
		if e := boxes[c]; e == nil {
			boxes[c] = &extent{x, y, x, y}
		} else {
			e.x0, e.y0, e.x1, e.y1 = min(e.x0, x), min(e.y0, y), max(e.x1, x), max(e.y1, y)
		}
	}
	type cut struct {
		across bool
		at     int
		into   int32
	}
	cuts := map[int32]cut{}
	for c, e := range boxes {
		bw, bh := e.x1-e.x0+1, e.y1-e.y0+1
		if bw <= maxW && bh <= maxH {
			continue
		}
		if float64(bw)/float64(maxW) >= float64(bh)/float64(maxH) {
			cuts[c] = cut{true, (e.x0 + e.x1 + 1) / 2, next}
		} else {
			cuts[c] = cut{false, (e.y0 + e.y1 + 1) / 2, next}
		}
		next++
	}
	if len(cuts) == 0 {
		return false
	}
	for p, c := range cell {
		k, ok := cuts[c]
		if !ok {
			continue
		}
		if (k.across && p%width >= k.at) || (!k.across && p/width >= k.at) {
			cell[p] = k.into
		}
	}
	return true
}

func joinPieces(width, height int, area, cell []int32) {
	for round := 0; round < 4; round++ {
		labels, sizes, _ := regions(width, height, func(a, b int) bool { return cell[a] == cell[b] })
		biggest := map[int32]int32{}
		for p, piece := range labels {
			c := cell[p]
			if b, ok := biggest[c]; !ok || sizes[piece] > sizes[b] || (sizes[piece] == sizes[b] && piece < b) {
				biggest[c] = piece
			}
		}
		stray := map[int32]bool{}
		for p, piece := range labels {
			if cell[p] >= 0 && biggest[cell[p]] != piece {
				stray[piece] = true
			}
		}
		if len(stray) == 0 {
			return
		}

		pieceCell := make([]int32, len(cell))

		offset := int32(0)
		for _, c := range cell {
			if c >= offset {
				offset = c + 1
			}
		}
		for p, piece := range labels {
			if stray[piece] {
				pieceCell[p] = offset + piece
			} else {
				pieceCell[p] = cell[p]
			}
		}
		absorb(width, height, area, pieceCell, func(c int32) bool { return c >= offset })
		copy(cell, pieceCell)
	}
}
