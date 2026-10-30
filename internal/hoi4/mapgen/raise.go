package mapgen

import (
	"fmt"
	"image"
)

func (b *Base) Raise(id int, terrain string, continent int) (*Addition, error) {
	if continent <= 0 {
		return nil, fmt.Errorf("continent must be positive")
	}
	var entry Entry
	foundTerrain := false
	for _, candidate := range Legend {
		if candidate.Class.Land() && candidate.Key == terrain {
			entry, foundTerrain = candidate, true
			break
		}
	}
	if !foundTerrain {
		return nil, fmt.Errorf("unknown land terrain %q", terrain)
	}
	index := -1
	for i := range b.Definitions {
		if b.Definitions[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, NoProvince(id)
	}
	row := &b.Definitions[index]
	if row.Kind == "land" {
		return nil, &AddError{Message: fmt.Sprintf("省份 #%d 已经是陆地", id), Code: "kind_land_water", Values: map[string]any{"id": id}, Provinces: []int{id}}
	}
	add := &Addition{pixels: map[int][]int32{id: {}}}
	changed := image.Rectangle{}
	for p, province := range b.Province {
		if province != int32(id) {
			continue
		}
		add.pixels[id] = append(add.pixels[id], int32(p))
		x, y := p%b.Width, p/b.Width
		changed = changed.Union(image.Rect(x, y, x+1, y+1))
		height := uint8(entry.Height)
		if height < LandHeight {
			height = LandHeight
		}
		if b.Heights[p] < height {
			b.Heights[p] = height
		}
		b.Terrain[p] = entry.Terrain
		b.Rivers[p] = RiverLand
	}
	if len(add.pixels[id]) == 0 {
		return nil, fmt.Errorf("province #%d has no pixels", id)
	}
	row.Kind, row.Terrain, row.Continent = "land", terrain, continent
	kindOf := map[int]string{}
	definitionAt := map[int]int{}
	for i, definition := range b.Definitions {
		kindOf[definition.ID] = definition.Kind
		definitionAt[definition.ID] = i
	}
	affected := []int{id}
	for other := range b.Borders([]int{id})[id] {
		affected = append(affected, other)
	}
	for province, neighbours := range b.Borders(affected) {
		at, ok := definitionAt[province]
		if !ok {
			continue
		}
		coastal := false
		for neighbour := range neighbours {
			coastal = coastal || kindOf[province] == "land" && kindOf[neighbour] == "sea" ||
				kindOf[province] == "sea" && kindOf[neighbour] == "land"
		}
		b.Definitions[at].Coastal = coastal
	}
	add.Changed = changed
	add.Provinces = []Province{{ID: id, Kind: "land", Coastal: row.Coastal, Terrain: terrain, Continent: continent}}
	return add, nil
}
