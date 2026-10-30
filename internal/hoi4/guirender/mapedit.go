package guirender

import "strings"

func (idx *Index) StateOfProvince(province int) (int, bool) {
	for _, state := range idx.effectiveStates() {
		for _, p := range state.Provinces {
			if p == province {
				return state.ID, true
			}
		}
	}
	return 0, false
}

func (idx *Index) StateProvinces(state int) ([]int, bool) {
	for _, record := range idx.effectiveStates() {
		if record.ID == state {
			return append([]int(nil), record.Provinces...), true
		}
	}
	return nil, false
}

func (idx *Index) ContinentCount() int {
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.continents) > 0 {
			return len(layer.continents)
		}
	}
	return 0
}

func (idx *Index) StrategicRegionFile(id int) (string, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if region := layer.strategicRegions[id]; region != nil {
			return region.File, true
		}
	}
	return "", false
}

func (idx *Index) CapitalStates() map[int][]string {
	out := map[int][]string{}
	for _, tag := range idx.ListCountryTags() {
		if politics := idx.countryPoliticsFor(tag); politics != nil && politics.Capital > 0 {
			out[politics.Capital] = append(out[politics.Capital], tag)
		}
	}
	return out
}

func (idx *Index) DivisionLocations(date string) map[int]int {
	out := map[int]int{}
	seen := map[string]bool{}
	for _, tag := range idx.ListCountryTags() {
		for _, name := range idx.OOBOf(tag, date) {
			key := strings.ToLower(strings.Trim(name, "\""))
			if seen[key] {
				continue
			}
			seen[key] = true
			if record, ok := idx.oobRecordFor(name); ok {
				for _, division := range record.Divisions {
					if division.Location > 0 {
						out[division.Location]++
					}
				}
			}
		}
	}
	return out
}
