package mapgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Definition struct {
	ID        int
	RGB       uint32
	Kind      string
	Coastal   bool
	Terrain   string
	Continent int
}

func ParseDefinitions(csv []byte) []Definition {
	var out []Definition
	for _, line := range strings.Split(string(csv), "\n") {
		fields := strings.Split(strings.TrimSpace(line), ";")
		if len(fields) < 8 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil || id <= 0 {
			continue
		}
		var rgb [3]int
		ok := true
		for i := range rgb {
			rgb[i], err = strconv.Atoi(strings.TrimSpace(fields[1+i]))
			ok = ok && err == nil && rgb[i] >= 0 && rgb[i] <= 255
		}
		if !ok {
			continue
		}
		continent, _ := strconv.Atoi(strings.TrimSpace(fields[7]))
		out = append(out, Definition{
			ID:        id,
			RGB:       uint32(rgb[0])<<16 | uint32(rgb[1])<<8 | uint32(rgb[2]),
			Kind:      strings.ToLower(strings.TrimSpace(fields[4])),
			Coastal:   strings.EqualFold(strings.TrimSpace(fields[5]), "true"),
			Terrain:   strings.TrimSpace(fields[6]),
			Continent: continent,
		})
	}
	return out
}

type CheckInput struct {
	Width, Height int

	ProvinceRGB []uint32
	Definitions []Definition

	HeightSize, TerrainSize, RiverSize [2]int

	Heights []uint8

	Terrain        []uint8
	TerrainIndexes map[uint8]bool
}

type Finding struct {
	Code     string   `json:"code"`
	Severity string   `json:"severity"`
	Message  string   `json:"message"`
	Count    int      `json:"count"`
	At       [][2]int `json:"at"`
}

const findingPlaces = 20

type findings struct {
	byCode map[string]*Finding
	order  []string
}

func (f *findings) add(code, severity, message string, x, y int) {
	if f.byCode == nil {
		f.byCode = map[string]*Finding{}
	}
	found := f.byCode[code]
	if found == nil {
		found = &Finding{Code: code, Severity: severity, Message: message, At: [][2]int{}}
		f.byCode[code] = found
		f.order = append(f.order, code)
	}
	found.Count++
	if len(found.At) < findingPlaces && x >= 0 {
		found.At = append(found.At, [2]int{x, y})
	}
}

func (f *findings) list() []Finding {
	out := make([]Finding, 0, len(f.order))
	for _, code := range f.order {
		out = append(out, *f.byCode[code])
	}
	return out
}

const MaxProvinces = 30000

const (
	MaxProvinceBoxWidth  = 384
	MaxProvinceBoxHeight = 254
)

const SmallProvincePixels = 8

func Check(in CheckInput) []Finding {
	var f findings
	w, h := in.Width, in.Height
	for name, size := range map[string][2]int{"heightmap.bmp": in.HeightSize, "terrain.bmp": in.TerrainSize, "rivers.bmp": in.RiverSize} {
		if size != [2]int{} && size != [2]int{w, h} {
			f.add("size_mismatch", "error", fmt.Sprintf("%s 是 %d×%d,和 provinces.bmp 的 %d×%d 不一样大", name, size[0], size[1], w, h), -1, -1)
		}
	}
	if w%256 != 0 || h%256 != 0 {
		f.add("size_not_256", "warning", fmt.Sprintf("地图 %d×%d 的宽、高不是 256 的倍数(待在游戏里核实是否必须)", w, h), -1, -1)
	}

	defined := map[uint32]Definition{}
	ids := map[int]bool{}
	for _, d := range in.Definitions {
		if _, dup := defined[d.RGB]; dup {
			f.add("duplicate_color", "error", "definition.csv 里有两个省份用同一种颜色", -1, -1)
		}
		if ids[d.ID] {
			f.add("duplicate_id", "error", "definition.csv 里有重复的省份编号", -1, -1)
		}
		defined[d.RGB] = d
		ids[d.ID] = true
	}
	if len(in.Definitions) > MaxProvinces {
		f.add("too_many_provinces", "warning", fmt.Sprintf("%d 个省份,超过 %d(上限待在游戏里核实)", len(in.Definitions), MaxProvinces), -1, -1)
	}

	pixels := map[uint32]int{}
	undefinedSeen := map[uint32]bool{}
	for p, rgb := range in.ProvinceRGB {
		pixels[rgb]++
		x, y := p%w, p/w
		if rgb == 0 {
			f.add("black_pixel", "error", "省份图里有纯黑 (0,0,0) 的像素,这个颜色是保留的", x, y)
			continue
		}
		if _, ok := defined[rgb]; !ok && !undefinedSeen[rgb] {
			undefinedSeen[rgb] = true
			f.add("undefined_color", "error", "省份图里的颜色在 definition.csv 里找不到", x, y)
		}
	}
	for _, d := range in.Definitions {
		if pixels[d.RGB] == 0 {
			f.add("unused_definition", "warning", "definition.csv 里有省份在地图上一个像素都没有", -1, -1)
		}
	}

	labels, sizes, _ := regions(w, h, func(a, b int) bool { return in.ProvinceRGB[a] == in.ProvinceRGB[b] })
	pieces := map[uint32][]int32{}
	firstAt := make([]int, len(sizes))
	for i := range firstAt {
		firstAt[i] = -1
	}
	for p, label := range labels {
		if firstAt[label] < 0 {
			firstAt[label] = p
			pieces[in.ProvinceRGB[p]] = append(pieces[in.ProvinceRGB[p]], label)
		}
	}
	colors := make([]uint32, 0, len(pieces))
	for rgb := range pieces {
		colors = append(colors, rgb)
	}
	sort.Slice(colors, func(i, j int) bool { return colors[i] < colors[j] })
	for _, rgb := range colors {
		list := pieces[rgb]
		if rgb != 0 && len(list) > 1 {

			biggest := list[0]
			for _, piece := range list {
				if sizes[piece] > sizes[biggest] {
					biggest = piece
				}
			}
			for _, piece := range list {
				if piece != biggest {
					p := firstAt[piece]
					f.add("disconnected_province", "warning", "有省份分成了几块不相连的部分(原版地图有 491 个,游戏能读;地图上会是同一个省份的几块)", p%w, p/w)
					break
				}
			}
		}
		if total := pixels[rgb]; rgb != 0 && total < SmallProvincePixels {
			p := firstAt[list[0]]
			f.add("small_province", "warning", fmt.Sprintf("有省份不到 %d 个像素", SmallProvincePixels), p%w, p/w)
		}
	}

	type extent struct{ x0, y0, x1, y1 int }
	boxes := map[uint32]*extent{}
	for p, rgb := range in.ProvinceRGB {
		x, y := p%w, p/w
		if e := boxes[rgb]; e == nil {
			boxes[rgb] = &extent{x, y, x, y}
		} else {
			e.x0, e.y0, e.x1, e.y1 = min(e.x0, x), min(e.y0, y), max(e.x1, x), max(e.y1, y)
		}
	}
	for _, rgb := range colors {
		e := boxes[rgb]
		if rgb == 0 || e == nil {
			continue
		}
		if bw, bh := e.x1-e.x0+1, e.y1-e.y0+1; bw > MaxProvinceBoxWidth || bh > MaxProvinceBoxHeight {
			f.add("province_box", "error", fmt.Sprintf("有省份的外框超过游戏上限(宽 %d、高 %d 像素),游戏读到这里会崩溃", MaxProvinceBoxWidth, MaxProvinceBoxHeight), e.x0, e.y0)
		}
	}

	for y := 0; y+1 < h; y++ {
		for x := 0; x+1 < w; x++ {
			p := y*w + x
			a, b, c, d := in.ProvinceRGB[p], in.ProvinceRGB[p+1], in.ProvinceRGB[p+w], in.ProvinceRGB[p+w+1]
			if (a == d && b == c && a != b) || (a != b && a != c && a != d && b != c && b != d && c != d) {
				f.add("x_crossing", "warning", "有四个省份交于一点,或两个省份只在一个角上相接(原版地图有 174 处,游戏能读)", x, y)
			}
		}
	}

	if len(in.Heights) == w*h {
		below := map[uint32]int{}
		for p, rgb := range in.ProvinceRGB {
			if in.Heights[p] < LandHeight {
				below[rgb]++
			}
		}
		for _, rgb := range colors {
			d, ok := defined[rgb]
			if !ok {
				continue
			}
			share := float64(below[rgb]) / float64(pixels[rgb])
			p := firstAt[pieces[rgb][0]]
			if d.Kind == "land" && share > 0.5 {
				f.add("land_under_water", "warning", "陆地省份大半低于水面高度(高度图低于 95),游戏里会画成水(原版地图有 96 个)", p%w, p/w)
			} else if (d.Kind == "sea" || d.Kind == "lake") && share < 0.5 {
				f.add("water_above_land", "warning", "海洋或湖泊省份大半高出水面(高度图 95 以上),游戏里会画成陆地(原版地图有 1 个)", p%w, p/w)
			}
		}
	}

	if len(in.Terrain) == w*h && in.TerrainIndexes != nil {
		seen := map[uint8]bool{}
		for p, index := range in.Terrain {
			if !in.TerrainIndexes[index] && !seen[index] {
				seen[index] = true
				f.add("terrain_index", "warning", fmt.Sprintf("地表图用了 00_terrain.txt 没定义的索引 %d(原版地图也有几个像素,游戏能读)", index), p%w, p/w)
			}
		}
	}
	return f.list()
}

func (m *Map) CheckInput() CheckInput {
	rgb := make([]uint32, len(m.Province))
	for p, k := range m.Province {
		c := m.Provinces[k].Color
		rgb[p] = uint32(c[0])<<16 | uint32(c[1])<<8 | uint32(c[2])
	}
	terrain := make([]uint8, len(m.Classes))
	indexes := map[uint8]bool{}
	for _, entry := range Legend {
		indexes[entry.Terrain] = true
	}
	for p, class := range m.Classes {
		terrain[p] = entryOf(class).Terrain
	}
	size := [2]int{m.Width, m.Height}
	return CheckInput{
		Width: m.Width, Height: m.Height, ProvinceRGB: rgb,
		Definitions: ParseDefinitions(m.DefinitionCSV()),
		HeightSize:  size, TerrainSize: size, RiverSize: size,
		Heights: m.Heights, Terrain: terrain, TerrainIndexes: indexes,
	}
}

func Errors(list []Finding) []Finding {
	var out []Finding
	for _, finding := range list {
		if finding.Severity == "error" {
			out = append(out, finding)
		}
	}
	return out
}
