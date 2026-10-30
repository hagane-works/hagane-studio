package guirender

import (
	"sort"
	"strings"
)

func (idx *Index) equipmentNeed(tag, date string) map[string]float64 {
	need := map[string]float64{}
	templates := map[string]DivisionTemplate{}
	for _, template := range idx.DivisionTemplatesOf(tag, date) {
		templates[template.Name] = template
	}
	for _, name := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(name)
		if !ok {
			continue
		}

		for templateName, count := range record.Deployed {
			template, known := templates[templateName]
			if !known || count == 0 {
				continue
			}
			for _, cell := range template.AllCells() {
				unit := idx.subUnitFor(cell.Type)
				if unit == nil {
					continue
				}
				for archetype, amount := range unit.Need {
					need[archetype] += amount * float64(count)
				}
			}
		}

		for model, amount := range record.Fielded {
			need[model] += amount
		}
	}
	return need
}

func (idx *Index) ProductionLinesOf(tag, date string) []oobProductionLine {
	var out []oobProductionLine
	for _, name := range idx.OOBOf(tag, date) {
		if record, ok := idx.oobRecordFor(name); ok {
			out = append(out, record.ProductionLines...)
		}
	}
	return out
}

func (idx *Index) onDockyards(equipment string) bool {
	record := idx.equipmentFor(equipment)
	if record == nil {
		return false
	}
	group := record.Group
	if record.Archetype != "" {
		if base := idx.equipmentFor(record.Archetype); base != nil && base.Group != "" {
			group = base.Group
		}
	}
	return strings.HasSuffix(strings.ToLower(group), "_ships")
}

func (idx *Index) factoryUsage(tag, date string) map[string]int {
	built := idx.builtBuildings(tag, date)
	military, naval := 0, 0
	for _, line := range idx.ProductionLinesOf(tag, date) {
		if idx.onDockyards(line.Equipment) {
			naval += line.Factories
		} else {
			military += line.Factories
		}
	}
	return map[string]int{
		"military_used": min(military, built["arms_factory"]), "military_total": built["arms_factory"],
		"naval_used": min(naval, built["dockyard"]), "naval_total": built["dockyard"],
	}
}

func (idx *Index) LogisticsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	need := idx.equipmentNeed(tag, date)
	year := yearOf(date)
	fielded := idx.FieldedEquipment(tag, date)
	unlockedBy := idx.enablingTech()

	byModel := map[string]map[string]any{}
	for key, amount := range need {
		record := idx.equipmentFor(key)
		if record == nil || amount <= 0 {
			continue
		}

		archetype, model := key, key
		if record.Archetype != "" {
			archetype = record.Archetype
		}
		var resources map[string]float64
		if model == archetype {
			model, _, resources = idx.equipmentAt(archetype, year, fielded)
		} else {
			_, _, resources = idx.equipmentAt(archetype, year, map[string]bool{model: true})
		}
		if base := idx.equipmentFor(archetype); base != nil {
			record = base
		}
		row := map[string]any{
			"archetype": archetype,
			"model":     model,
			"group":     productionTabs[record.Group],
			"category":  record.Group,
			"needs":     int(amount + 0.5),
			"resources": resources,
			"icon": idx.equipmentIcon(tag, model, archetype, record.Picture,
				unlockedBy[strings.ToLower(model)]),
		}
		if label, ok := idx.Localise(strings.ToUpper(tag) + "_" + model); ok {
			row["label"] = label
		} else if label, ok := idx.Localise(model); ok {
			row["label"] = label
		} else if label, ok := idx.Localise(archetype); ok {
			row["label"] = label
		}
		if existing, seen := byModel[model]; seen {
			was, _ := existing["needs"].(int)
			add, _ := row["needs"].(int)
			existing["needs"] = was + add
			continue
		}
		byModel[model] = row
	}
	rows := make([]map[string]any, 0, len(byModel))
	for _, row := range byModel {
		rows = append(rows, row)
	}

	order := map[string]int{
		"infantry": 0, "armor": 1, "air": 2,
		"capital_ships": 3, "screen_ships": 4, "other_ships": 5,
	}
	sort.Slice(rows, func(a, b int) bool {
		ga, _ := rows[a]["group"].(string)
		gb, _ := rows[b]["group"].(string)
		if order[ga] != order[gb] {
			return order[ga] < order[gb]
		}
		na, _ := rows[a]["needs"].(int)
		nb, _ := rows[b]["needs"].(int)
		if na != nb {
			return na > nb
		}
		aa, _ := rows[a]["archetype"].(string)
		ab, _ := rows[b]["archetype"].(string)
		return aa < ab
	})

	out := map[string]any{
		"tag":             tag,
		"equipment":       rows,
		"resource_frames": idx.ResourceFrames(),
	}
	if stats := idx.CountryStatsPayload(tag, date); stats != nil {
		out["convoys"] = stats["convoys"]
		out["resources"] = stats["resources"]
		out["nukes"] = stats["nukes"]
	}

	out["factories"] = idx.factoryUsage(tag, date)
	if bar := idx.spritePayload(logisticsUsageBar); bar != nil {
		out["sprites"] = map[string]any{logisticsUsageBar: bar}
	}
	return out
}

const logisticsUsageBar = "GFX_logistics_fuelbar"
