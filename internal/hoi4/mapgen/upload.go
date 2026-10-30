package mapgen

import (
	"errors"
	"fmt"
	"image"
	"image/color"
)

type UploadInput struct {
	Provinces, Heightmap, Terrain, Rivers image.Image
	Definitions                           []byte

	FillStrayPixels bool
}

type Upload struct {
	Base *Base

	Check CheckInput

	MadeDefinitions bool

	MadeRivers bool

	FilledPixels int
}

func FillStrayPixels(rgb []uint32, w, h int, province func(uint32) bool) int {
	var stray []int
	for p, c := range rgb {
		if !province(c) {
			stray = append(stray, p)
		}
	}
	filled := 0
	for len(stray) > 0 {
		type change struct {
			p int
			c uint32
		}
		var changes []change
		var left []int
		for _, p := range stray {
			x, y := p%w, p/w
			var colours [4]uint32
			var counts [4]int
			n := 0
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				c := rgb[ny*w+nx]
				if !province(c) {
					continue
				}
				i := 0
				for i < n && colours[i] != c {
					i++
				}
				if i == n {
					colours[n] = c
					n++
				}
				counts[i]++
			}
			if n == 0 {
				left = append(left, p)
				continue
			}
			best := 0
			for i := 1; i < n; i++ {
				if counts[i] > counts[best] || (counts[i] == counts[best] && colours[i] < colours[best]) {
					best = i
				}
			}
			changes = append(changes, change{p, colours[best]})
		}
		if len(changes) == 0 {
			break
		}
		for _, c := range changes {
			rgb[c.p] = c.c
		}
		filled += len(changes)
		stray = left
	}
	return filled
}

type UploadError struct {
	Message string
	Code    string
	Values  map[string]any
}

func (e *UploadError) Error() string { return e.Message }

func ReadUpload(in UploadInput, game *Base) (*Upload, error) {
	if in.Provinces == nil || in.Heightmap == nil || in.Terrain == nil {
		return nil, &UploadError{Message: "要有省份图、高度图和地表图三张", Code: "upload_pictures_missing"}
	}
	bounds := in.Provinces.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	for _, picture := range []struct {
		name, file string
		img        image.Image
	}{{"高度图", "heightmap.bmp", in.Heightmap}, {"地表图", "terrain.bmp", in.Terrain}, {"河流图", "rivers.bmp", in.Rivers}} {
		if img := picture.img; img != nil && (img.Bounds().Dx() != w || img.Bounds().Dy() != h) {
			return nil, &UploadError{
				Message: fmt.Sprintf("%s是 %d×%d,和省份图的 %d×%d 不一样大", picture.name, img.Bounds().Dx(), img.Bounds().Dy(), w, h),
				Code:    "upload_size_mismatch",
				Values:  map[string]any{"file": picture.file, "width": img.Bounds().Dx(), "height": img.Bounds().Dy(), "map_width": w, "map_height": h},
			}
		}
	}
	if game == nil || len(game.TerrainPalette) == 0 {
		return nil, errors.New("upload: the base game's map is needed")
	}

	rgb := make([]uint32, w*h)
	for p := range rgb {
		r, g, b, _ := in.Provinces.At(bounds.Min.X+p%w, bounds.Min.Y+p/w).RGBA()
		rgb[p] = uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(b>>8)
	}
	heights := greyValues(in.Heightmap, w, h)
	terrain := paletteIndexes(in.Terrain, game.TerrainPalette, w, h)

	out := &Upload{}
	var definitions []Definition
	if len(in.Definitions) > 0 {
		definitions = ParseDefinitions(in.Definitions)
		if len(definitions) == 0 {
			return nil, &UploadError{Message: "definition.csv 里没有省份行", Code: "upload_definitions_empty"}
		}
	} else {
		definitions = makeDefinitions(rgb, terrain, heights, w, h, game.TerrainNames())
		out.MadeDefinitions = true
	}
	if in.FillStrayPixels {
		colours := make(map[uint32]bool, len(definitions))
		for _, d := range definitions {
			if d.RGB != 0 {
				colours[d.RGB] = true
			}
		}
		out.FilledPixels = FillStrayPixels(rgb, w, h, func(c uint32) bool { return colours[c] })
	}

	base := &Base{
		Width: w, Height: h, Heights: heights, Terrain: terrain,
		TerrainPalette: game.TerrainPalette, RiverPalette: game.RiverPalette, Definitions: definitions,
	}
	idOf := make(map[uint32]int32, len(definitions))
	water := map[int32]bool{}
	for _, d := range definitions {
		idOf[d.RGB] = int32(d.ID)
		if d.Kind != "land" {
			water[int32(d.ID)] = true
		}
	}
	base.Province = make([]int32, w*h)
	for p, c := range rgb {
		base.Province[p] = idOf[c]
	}
	if in.Rivers != nil {
		base.Rivers = paletteIndexes(in.Rivers, game.RiverPalette, w, h)
	} else {
		base.Rivers = make([]uint8, w*h)
		for p, id := range base.Province {
			if water[id] {
				base.Rivers[p] = RiverWater
			} else {
				base.Rivers[p] = RiverLand
			}
		}
		out.MadeRivers = true
	}
	out.Base = base
	size := [2]int{w, h}
	out.Check = CheckInput{
		Width: w, Height: h, ProvinceRGB: rgb, Definitions: definitions,
		HeightSize: size, TerrainSize: size, RiverSize: size, Heights: heights,
		Terrain: terrain, TerrainIndexes: game.TerrainIndexes(),
	}
	return out, nil
}

func (b *Base) TerrainNames() map[uint8]string {
	terrainOf := make(map[int32]string, len(b.Definitions))
	for _, d := range b.Definitions {
		terrainOf[int32(d.ID)] = d.Terrain
	}
	votes := map[uint8]map[string]int{}
	for p, id := range b.Province {
		name := terrainOf[id]
		if name == "" || name == "unknown" {
			continue
		}
		index := b.Terrain[p]
		if votes[index] == nil {
			votes[index] = map[string]int{}
		}
		votes[index][name]++
	}
	out := map[uint8]string{}
	for index, counts := range votes {
		out[index] = mostVotes(counts)
	}
	return out
}

func (b *Base) TerrainIndexes() map[uint8]bool {
	count := map[uint8]int{}
	for _, index := range b.Terrain {
		count[index]++
	}
	out := map[uint8]bool{}
	for index, n := range count {
		if n >= 100 {
			out[index] = true
		}
	}
	return out
}

func mostVotes(counts map[string]int) string {
	best, bestN := "", -1
	for name, n := range counts {
		if n > bestN || (n == bestN && name < best) {
			best, bestN = name, n
		}
	}
	return best
}

func makeDefinitions(rgb []uint32, terrain, heights []uint8, w, h int, names map[uint8]string) []Definition {
	at := map[uint32]int{}
	var definitions []Definition
	var votes []map[string]int
	var below, pixels []int
	edge := []bool{}
	for p, c := range rgb {
		if c == 0 {
			continue
		}
		k, ok := at[c]
		if !ok {
			k = len(definitions)
			at[c] = k
			definitions = append(definitions, Definition{ID: k + 1, RGB: c})
			votes = append(votes, map[string]int{})
			below = append(below, 0)
			pixels = append(pixels, 0)
			edge = append(edge, false)
		}
		pixels[k]++
		if heights[p] < LandHeight {
			below[k]++
		}
		if x, y := p%w, p/w; x == 0 || y == 0 || x == w-1 || y == h-1 {
			edge[k] = true
		}
		if name := names[terrain[p]]; name != "" && name != "ocean" && name != "lakes" {
			votes[k][name]++
		}
	}
	water := make([]bool, len(definitions))
	for k := range definitions {
		water[k] = below[k]*2 > pixels[k]
	}

	neighbours := make([]map[int]bool, len(definitions))
	link := func(a, b int) {
		if rgb[a] == rgb[b] || rgb[a] == 0 || rgb[b] == 0 {
			return
		}
		ka, kb := at[rgb[a]], at[rgb[b]]
		if neighbours[ka] == nil {
			neighbours[ka] = map[int]bool{}
		}
		if neighbours[kb] == nil {
			neighbours[kb] = map[int]bool{}
		}
		neighbours[ka][kb], neighbours[kb][ka] = true, true
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x+1 < w {
				link(p, p+1)
			}
			if y+1 < h {
				link(p, p+w)
			}
		}
	}

	lakeMax := w * h / 400
	body := make([]int, len(definitions))
	for k := range body {
		body[k] = -1
	}
	for start := range definitions {
		if !water[start] || body[start] >= 0 {
			continue
		}
		members, size, reachesEdge := []int{start}, 0, false
		body[start] = start
		for i := 0; i < len(members); i++ {
			k := members[i]
			size += pixels[k]
			reachesEdge = reachesEdge || edge[k]
			for other := range neighbours[k] {
				if water[other] && body[other] < 0 {
					body[other] = start
					members = append(members, other)
				}
			}
		}
		kind := "sea"
		if size < lakeMax && !reachesEdge {
			kind = "lake"
		}
		for _, k := range members {
			definitions[k].Kind = kind
		}
	}
	for k := range definitions {
		switch definitions[k].Kind {
		case "sea":
			definitions[k].Terrain = "ocean"
		case "lake":
			definitions[k].Terrain = "lakes"
		default:
			definitions[k].Kind = "land"
			definitions[k].Continent = 1
			definitions[k].Terrain = mostVotes(votes[k])
			if definitions[k].Terrain == "" {
				definitions[k].Terrain = "plains"
			}
		}
	}
	for k := range definitions {
		for other := range neighbours[k] {
			switch mine, theirs := definitions[k].Kind, definitions[other].Kind; {
			case mine == "land" && theirs == "sea", mine == "sea" && theirs == "land", mine == "lake" && theirs == "sea":
				definitions[k].Coastal = true
			}
		}
	}
	return definitions
}

func greyValues(img image.Image, w, h int) []uint8 {
	out := make([]uint8, w*h)
	b := img.Bounds()
	if g, ok := img.(*image.Gray); ok {
		for y := 0; y < h; y++ {
			copy(out[y*w:(y+1)*w], g.Pix[y*g.Stride:y*g.Stride+w])
		}
		return out
	}
	for p := range out {
		c := color.GrayModel.Convert(img.At(b.Min.X+p%w, b.Min.Y+p/w)).(color.Gray)
		out[p] = c.Y
	}
	return out
}

func paletteIndexes(img image.Image, palette color.Palette, w, h int) []uint8 {
	out := make([]uint8, w*h)
	if paletted, ok := img.(*image.Paletted); ok && samePalette(paletted.Palette, palette) {
		for y := 0; y < h; y++ {
			copy(out[y*w:(y+1)*w], paletted.Pix[y*paletted.Stride:y*paletted.Stride+w])
		}
		return out
	}
	b := img.Bounds()
	nearest := map[uint32]uint8{}
	for p := range out {
		r, g, bl, _ := img.At(b.Min.X+p%w, b.Min.Y+p/w).RGBA()
		key := r>>8<<16 | g>>8<<8 | bl>>8
		index, ok := nearest[key]
		if !ok {
			index = uint8(palette.Index(color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), 255}))
			nearest[key] = index
		}
		out[p] = index
	}
	return out
}

func samePalette(a, b color.Palette) bool {
	if len(a) == 0 || len(a) > len(b) {
		return false
	}
	for i := range a {
		r1, g1, b1, _ := a[i].RGBA()
		r2, g2, b2, _ := b[i].RGBA()
		if r1>>8 != r2>>8 || g1>>8 != g2>>8 || b1>>8 != b2>>8 {
			return false
		}
	}
	return true
}
