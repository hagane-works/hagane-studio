package guirender

import "strings"

type strategicRegion struct {
	ID   int
	Name string
	File string
}

func (idx *Index) indexStrategicRegionFile(file string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.strategicRegions == nil {
		idx.strategicRegions = map[int]*strategicRegion{}
		idx.provinceRegions = map[int]int{}
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "strategic_region") {
			continue
		}
		id, ok := toFloat(ciScalarValue(entry.Block.Entries, "id"))
		if !ok {
			continue
		}
		region := &strategicRegion{ID: int(id), Name: strings.Trim(ciScalar(entry.Block.Entries, "name"), "\""), File: file}
		idx.strategicRegions[region.ID] = region
		if provinces := childBlock(entry.Block.Entries, "provinces"); provinces != nil {
			for _, token := range blockTokens(provinces.Entries) {
				if province, ok := toFloat(token); ok {
					idx.provinceRegions[int(province)] = region.ID
				}
			}
		}
	}
}

func (idx *Index) strategicRegionOf(province int) *strategicRegion {
	for layer := idx; layer != nil; layer = layer.fallback {
		if id, ok := layer.provinceRegions[province]; ok {
			if region := layer.strategicRegions[id]; region != nil {
				return region
			}
		}
	}
	return nil
}

func (idx *Index) StrategicRegionOf(province int) (id int, file string, ok bool) {
	if region := idx.strategicRegionOf(province); region != nil {
		return region.ID, region.File, true
	}
	return 0, "", false
}

func (idx *Index) MaxStrategicRegionID() int {
	most := 0
	for layer := idx; layer != nil; layer = layer.fallback {
		for id := range layer.strategicRegions {
			most = max(most, id)
		}
	}
	return most
}
