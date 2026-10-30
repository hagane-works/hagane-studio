package mapsketch

import (
	"math"
	"sort"

	"haganeworks/internal/hoi4/mapgen"
)

type Swatch struct {
	Hex string `json:"hex"`

	Share float64 `json:"share"`

	Sample Point `json:"sample"`

	SampleSourceColour string `json:"sample_source_colour,omitempty"`

	NearestTerrain  string  `json:"nearest_terrain"`
	TerrainDistance float64 `json:"terrain_distance"`

	Dark bool `json:"dark"`
}

const DefaultDark = 45

func Palette(p *Picture, maxColours int) []Swatch {
	if maxColours <= 0 {
		maxColours = 16
	}
	q := p.Scaled(320)
	type bin struct{ n, r, g, b int }
	bins := make([]bin, 1<<15)
	opaque := 0
	for i, c := range q.Pix {
		if q.clear(i) {
			continue
		}
		opaque++
		k := int(c[0]>>3)<<10 | int(c[1]>>3)<<5 | int(c[2]>>3)
		bins[k].n++
		bins[k].r += int(c[0])
		bins[k].g += int(c[1])
		bins[k].b += int(c[2])
	}
	if opaque == 0 {
		return []Swatch{}
	}
	order := make([]int, 0, 1024)
	for k := range bins {
		if bins[k].n > 0 {
			order = append(order, k)
		}
	}
	sort.Slice(order, func(a, b int) bool { return bins[order[a]].n > bins[order[b]].n })
	type cluster struct {
		sum [3]float64
		n   int
	}
	centre := func(c cluster) RGB {
		return RGB{uint8(c.sum[0] / float64(c.n)), uint8(c.sum[1] / float64(c.n)), uint8(c.sum[2] / float64(c.n))}
	}
	var clusters []cluster
	for _, k := range order {
		b := bins[k]
		mean := RGB{uint8(b.r / b.n), uint8(b.g / b.n), uint8(b.b / b.n)}
		best, bestD := -1, math.MaxFloat64
		for i, c := range clusters {
			if d := Distance(mean, centre(c)); d < bestD {
				best, bestD = i, d
			}
		}
		if best >= 0 && (bestD < 26 || len(clusters) >= 64) {
			clusters[best].sum[0] += float64(b.r)
			clusters[best].sum[1] += float64(b.g)
			clusters[best].sum[2] += float64(b.b)
			clusters[best].n += b.n
			continue
		}
		clusters = append(clusters, cluster{sum: [3]float64{float64(b.r), float64(b.g), float64(b.b)}, n: b.n})
	}
	centres := make([]RGB, len(clusters))
	for i, c := range clusters {
		centres[i] = centre(c)
	}

	owner := make([]int, len(q.Pix))
	count := make([]int, len(centres))
	for i, c := range q.Pix {
		owner[i] = -1
		if q.clear(i) {
			continue
		}
		best, bestD := 0, math.MaxFloat64
		for k, centreColour := range centres {
			if d := Distance(c, centreColour); d < bestD {
				best, bestD = k, d
			}
		}
		owner[i] = best
		count[best]++
	}
	ranked := make([]int, len(centres))
	for i := range ranked {
		ranked[i] = i
	}
	sort.SliceStable(ranked, func(a, b int) bool { return count[ranked[a]] > count[ranked[b]] })
	out := []Swatch{}
	for _, k := range ranked {
		if len(out) >= maxColours {
			break
		}
		share := float64(count[k]) / float64(opaque)
		if share < 0.002 {
			break
		}
		at := deepest(q.W, q.H, func(i int) bool { return owner[i] == k })
		entry, d := NearestTerrain(centres[k])
		sample := Point{X: int((float64(at%q.W) + 0.5) * float64(p.W) / float64(q.W)), Y: int((float64(at/q.W) + 0.5) * float64(p.H) / float64(q.H))}
		sourceColour := ""
		if source := sample.Y*p.W + sample.X; !p.clear(source) {
			sourceColour = p.Pix[source].Hex()
		}
		out = append(out, Swatch{
			Hex: centres[k].Hex(), Share: round4(share),
			Sample: sample, SampleSourceColour: sourceColour,
			NearestTerrain:  entry.Key,
			TerrainDistance: math.Round(d),
			Dark:            luminance(centres[k]) < DefaultDark,
		})
	}
	return out
}

func deepest(w, h int, in func(int) bool) int {
	const inf = math.MaxInt32 / 2
	d := make([]int32, w*h)
	for i := range d {
		if in(i) {
			d[i] = inf
		}
	}
	at := func(x, y int) int32 {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		return d[y*w+x]
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if d[i] == 0 {
				continue
			}
			d[i] = min(d[i], at(x-1, y)+3, at(x, y-1)+3, at(x-1, y-1)+4, at(x+1, y-1)+4)
		}
	}
	best, bestD := 0, int32(-1)
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			i := y*w + x
			if d[i] == 0 {
				continue
			}
			d[i] = min(d[i], at(x+1, y)+3, at(x, y+1)+3, at(x+1, y+1)+4, at(x-1, y+1)+4)
			if d[i] > bestD {
				best, bestD = i, d[i]
			}
		}
	}
	return best
}

func NearestTerrain(c RGB) (mapgen.Entry, float64) {
	best, bestD := mapgen.Legend[0], math.MaxFloat64
	for _, entry := range mapgen.Legend {
		if d := Distance(c, entry.Color); d < bestD {
			best, bestD = entry, d
		}
	}
	return best, bestD
}

func TerrainClass(name string) (mapgen.Entry, bool) {
	switch name {
	case "sea", "water":
		name = "ocean"
	case "lake":
		name = "lakes"
	}
	for _, entry := range mapgen.Legend {
		if entry.Key == name {
			return entry, true
		}
	}
	return mapgen.Entry{}, false
}

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
