package mapgen

import (
	"bytes"
	"fmt"
	"image"

	"golang.org/x/image/bmp"
)

const (
	RiverSource = 0
	RiverJoin   = 1
	RiverSplit  = 2
	RiverNarrow = 3
	RiverWide   = 11
)

func IsRiver(v uint8) bool { return v <= RiverWide }

func RiverPixels(rivers []uint8) (at []int, values []uint8) {
	for p, v := range rivers {
		if IsRiver(v) {
			at = append(at, p)
			values = append(values, v)
		}
	}
	return at, values
}

func DecodeRivers(data []byte) (width, height int, rivers []uint8, err error) {
	img, err := bmp.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("rivers.bmp: %w", err)
	}
	paletted, ok := img.(*image.Paletted)
	if !ok {
		return 0, 0, nil, fmt.Errorf("rivers.bmp is not an 8-bit indexed image")
	}
	return paletted.Rect.Dx(), paletted.Rect.Dy(), packed(paletted), nil
}

func (b *Base) EditRivers(pixels map[int]uint8) (int, error) {
	kinds := b.kinds()
	for p, v := range pixels {
		if p < 0 || p >= len(b.Rivers) {
			return 0, &AddError{Message: fmt.Sprintf("河流的位置不在地图上(%d)", p), Code: "bad_request"}
		}
		where := map[string]any{"x": p % b.Width, "y": p / b.Width}
		switch {
		case IsRiver(v):
			id := int(b.Province[p])
			if id <= 0 || id >= len(kinds) || kinds[id] != kindLand {
				return 0, &AddError{Message: fmt.Sprintf("河流画到了水面上(%d, %d)", p%b.Width, p/b.Width),
					Code: "river_on_water", Values: where, Provinces: []int{id}}
			}
		case v == RiverLand:
			if b.Rivers[p] == RiverWater {
				return 0, &AddError{Message: fmt.Sprintf("(%d, %d) 是水面,不是河", p%b.Width, p/b.Width),
					Code: "river_on_water", Values: where, Provinces: []int{int(b.Province[p])}}
			}
		default:
			return 0, &AddError{Message: fmt.Sprintf("河流图里没有 %d 这个值", v), Code: "bad_request"}
		}
	}
	around := b.riverNeighbourhood(pixels)
	before := riverFaults(b.Rivers, b.Width, b.Height, around)
	old := make(map[int]uint8, len(pixels))
	changed := 0
	for p, v := range pixels {
		old[p] = b.Rivers[p]
		if b.Rivers[p] != v {
			changed++
		}
		b.Rivers[p] = v
	}
	if after := riverFaults(b.Rivers, b.Width, b.Height, around); after > before {
		for p, v := range old {
			b.Rivers[p] = v
		}
		return 0, &AddError{Message: fmt.Sprintf("这样改会让河流接错(%d 处):源头要在河的一端、支流只能并排接进别的河、河只有一格宽", after-before),
			Code: "river_faults", Values: map[string]any{"count": after - before}}
	}
	return changed, nil
}

func (b *Base) riverNeighbourhood(pixels map[int]uint8) []int {
	seen := map[int]bool{}
	var out []int
	for p := range pixels {
		x, y := p%b.Width, p/b.Width
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= b.Width || ny >= b.Height {
					continue
				}
				q := ny*b.Width + nx
				if !seen[q] {
					seen[q] = true
					out = append(out, q)
				}
			}
		}
	}
	return out
}

func riverDegree(rivers []uint8, w, h, p int) int {
	x, y := p%w, p/w
	n := 0
	if x > 0 && IsRiver(rivers[p-1]) {
		n++
	}
	if x+1 < w && IsRiver(rivers[p+1]) {
		n++
	}
	if y > 0 && IsRiver(rivers[p-w]) {
		n++
	}
	if y+1 < h && IsRiver(rivers[p+w]) {
		n++
	}
	return n
}

func besideMarker(rivers []uint8, w, h, p int) bool {
	x, y := p%w, p/w
	marker := func(q int) bool { return rivers[q] == RiverJoin || rivers[q] == RiverSplit }
	return (x > 0 && marker(p-1)) || (x+1 < w && marker(p+1)) || (y > 0 && marker(p-w)) || (y+1 < h && marker(p+w))
}

func riverFaults(rivers []uint8, w, h int, pixels []int) int {
	faults := 0
	for _, p := range pixels {
		v := rivers[p]
		if !IsRiver(v) {
			continue
		}
		deg := riverDegree(rivers, w, h, p)
		switch {
		case v == RiverSource && deg != 1:
			faults++
		case (v == RiverJoin || v == RiverSplit) && deg != 2:
			faults++
		case deg > 3:
			faults++
		case deg == 3 && !besideMarker(rivers, w, h, p):
			faults++
		}
		x, y := p%w, p/w
		if x+1 < w && y+1 < h && IsRiver(rivers[p+1]) && IsRiver(rivers[p+w]) && IsRiver(rivers[p+w+1]) {
			faults++
		}
	}
	return faults
}
