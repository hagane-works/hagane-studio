package mapgen

import (
	"fmt"
	"image"
	"math"
)

var terrainIndexes = map[string]uint8{
	"plains": 0, "forest": 1, "hills": 17, "mountain": 6, "desert": 3, "marsh": 9, "jungle": 21, "urban": 13,
}

func TerrainIndex(terrain string) (uint8, bool) {
	index, ok := terrainIndexes[terrain]
	return index, ok
}

func reliefOf(terrain string) (float64, float64) {
	for _, entry := range Legend {
		if entry.Key == terrain && entry.Class.Land() {
			return entry.Height, entry.Rough
		}
	}
	return Legend[Plains].Height, Legend[Plains].Rough
}

func (b *Base) Reterrain(terrains map[int]string, seed int64) (*Addition, error) {
	rows := map[int]*Definition{}
	for i := range b.Definitions {
		rows[b.Definitions[i].ID] = &b.Definitions[i]
	}
	for id, terrain := range terrains {
		row, ok := rows[id]
		if !ok {
			return nil, NoProvince(id)
		}
		if row.Kind != "land" {
			return nil, WaterTerrain(id)
		}
		if _, ok := terrainIndexes[terrain]; !ok {
			return nil, &AddError{Message: fmt.Sprintf("不认识的地形 %s", terrain), Code: "bad_request"}
		}
	}
	add := &Addition{pixels: map[int][]int32{}, was: map[int]string{}}
	changed := image.Rectangle{}
	for p, id := range b.Province {
		if _, painted := terrains[int(id)]; painted {
			add.pixels[int(id)] = append(add.pixels[int(id)], int32(p))
			x, y := p%b.Width, p/b.Width
			changed = changed.Union(image.Rect(x, y, x+1, y+1))
		}
	}
	for id, terrain := range terrains {
		add.was[id] = rows[id].Terrain
		rows[id].Terrain = terrain
		add.Provinces = append(add.Provinces, Province{ID: id, Kind: "land", Terrain: terrain, Coastal: rows[id].Coastal, Continent: rows[id].Continent})
		index := terrainIndexes[terrain]
		for _, p := range add.pixels[id] {
			b.Terrain[p] = index
		}
	}
	add.Changed = changed
	if changed.Empty() {
		return add, nil
	}

	area := changed.Inset(-2).Intersect(image.Rect(0, 0, b.Width, b.Height))
	w, h := area.Dx(), area.Dy()
	target := make([]float32, w*h)
	depth := make([]int32, w*h)
	cell := math.Max(8, float64(b.Width)/90)
	var queue []int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := (area.Min.Y+y)*b.Width + area.Min.X + x
			terrain, ok := terrains[int(b.Province[p])]
			if !ok {
				queue = append(queue, y*w+x)
				continue
			}
			depth[y*w+x] = -1
			base, rough := reliefOf(terrain)
			mx, my := float64(area.Min.X+x), float64(area.Min.Y+y)
			v := base + rough*fractalNoise(seed+17, mx, my, cell) + 1.5*valueNoise(seed+29, mx, my, 4)
			if rough >= 8 {

				ridge := 1 - math.Abs(2*valueNoise(seed+53, mx, my, cell/3)-1)
				v += rough * 1.2 * (ridge - 0.5)
			}
			target[y*w+x] = float32(v)
		}
	}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w, i/w
		for _, n := range [4][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
			if n[0] < 0 || n[1] < 0 || n[0] >= w || n[1] >= h {
				continue
			}
			j := n[1]*w + n[0]
			if depth[j] == -1 {
				depth[j] = depth[i] + 1
				queue = append(queue, j)
			}
		}
	}
	const falloff = 14.0
	smooth := boxBlur(target, w, h, 2)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if depth[i] == 0 {
				continue
			}
			d := float64(depth[i])
			if depth[i] < 0 {
				d = falloff
			}
			t := math.Min(1, d/falloff)
			t = t * t * (3 - 2*t)
			p := (area.Min.Y+y)*b.Width + area.Min.X + x
			old := float64(b.Heights[p])
			b.Heights[p] = uint8(math.Max(LandHeight, math.Min(255, math.Round(old+(float64(smooth[i])-old)*t))))
		}
	}
	return add, nil
}

func WaterTerrain(id int) *AddError {
	return &AddError{Message: fmt.Sprintf("省份 #%d 是水面,只有陆地能换地形", id), Code: "water_terrain",
		Values: map[string]any{"id": id}, Provinces: []int{id}}
}
