package guirender

import (
	"sort"
	"strings"
)

func (idx *Index) StatesOfPayload(tag, date string, want int) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	capital := 0
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		capital = politics.Capital
	}
	buildings := map[string]*buildingRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for name, record := range layer.buildings {
			if _, seen := buildings[name]; !seen {
				buildings[name] = record
			}
		}
	}

	list := []map[string]any{}
	var chosen *stateRecord
	states := idx.effectiveStates()
	for _, state := range states {
		if idx.ownerOfState(state, date) != tag {
			continue
		}
		item := map[string]any{
			"id":       state.ID,
			"manpower": state.Manpower,
			"category": state.Category,
		}
		if label, ok := idx.Localise(state.Name); ok {
			item["name"] = label
		} else if state.Name != "" {
			item["name"] = state.Name
		}
		list = append(list, item)
		if state.ID == want || (want == 0 && chosen == nil && state.ID == capital) {
			chosen = state
		}
	}
	sort.Slice(list, func(a, b int) bool {
		na, _ := list[a]["name"].(string)
		nb, _ := list[b]["name"].(string)
		if na != nb {
			return na < nb
		}
		return list[a]["id"].(int) < list[b]["id"].(int)
	})

	if chosen == nil && len(list) > 0 {

		for _, state := range states {
			if idx.ownerOfState(state, date) == tag {
				chosen = state
				break
			}
		}
	}

	out := map[string]any{"tag": tag, "states": list, "capital": capital}
	if chosen == nil {
		return out
	}

	shelves := map[string][]map[string]any{"state": {}, "shared": {}, "province": {}}
	levels := chosen.buildingsAt(date)
	for _, name := range sortedKeys(levelsAsFloat(levels)) {
		record := buildings[name]
		group := "state"
		if record != nil && record.group() != "" {
			group = record.group()
		}
		if group == "facility" {
			group = "state"
		}
		item := map[string]any{"building": name, "level": levels[name]}
		if record != nil {
			item["icon_frame"] = record.IconFrame
			item["max"] = record.StateMax
		}
		if label, ok := idx.Localise(name); ok {
			item["label"] = label
		}
		shelves[group] = append(shelves[group], item)
	}

	byProvince := chosen.provinceBuildingsAt(date)
	province := []map[string]any{}
	for _, id := range sortedInts(byProvince) {
		for _, name := range sortedKeys(levelsAsFloat(byProvince[id])) {
			item := map[string]any{
				"province": id, "building": name, "level": byProvince[id][name],
			}
			if record := buildings[name]; record != nil {
				item["icon_frame"] = record.IconFrame
			}
			if label, ok := idx.Localise(name); ok {
				item["label"] = label
			}
			province = append(province, item)
		}
	}

	cores := []string{}
	for _, block := range chosen.Cores {
		if !dateReached(block.Date, date) {
			continue
		}
		for _, token := range block.Tokens {
			cores = appendOnce(cores, strings.ToUpper(token))
		}
	}
	points := 0
	for _, value := range chosen.VictoryPoints {
		points += value
	}

	detail := map[string]any{
		"id":                 chosen.ID,
		"manpower":           chosen.Manpower,
		"category":           chosen.Category,
		"resources":          chosen.Resources,
		"cores":              cores,
		"victory_points":     points,
		"provinces":          len(chosen.Provinces),
		"buildings":          shelves["state"],
		"shared":             shelves["shared"],
		"province_buildings": province,
	}
	if label, ok := idx.Localise(chosen.Name); ok {
		detail["name"] = label
	} else if chosen.Name != "" {
		detail["name"] = chosen.Name
	}
	if label, ok := idx.Localise("STATE_CATEGORY_" + chosen.Category); ok {
		detail["category_label"] = label
	} else if label, ok := idx.Localise(chosen.Category); ok {
		detail["category_label"] = label
	}
	out["state"] = detail
	out["resource_frames"] = idx.ResourceFrames()
	return out
}

func levelsAsFloat(levels map[string]int) map[string]float64 {
	out := make(map[string]float64, len(levels))
	for name, level := range levels {
		out[name] = float64(level)
	}
	return out
}

func sortedInts(from map[int]map[string]int) []int {
	out := make([]int, 0, len(from))
	for id := range from {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}
