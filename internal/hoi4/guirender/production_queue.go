package guirender

import (
	"fmt"
	"math"
	"strings"
	"time"
)

var queueCategories = []string{"infantry", "armor", "air", "capital_ships", "screen_ships", "other_ships"}

func (idx *Index) productionQueue(tag, date string, items []map[string]any) map[string]any {
	sums := idx.startingModifiers(tag, date)
	built := idx.builtBuildings(tag, date)
	left := map[bool]int{false: built["arms_factory"], true: built["dockyard"]}
	used := map[bool]int{}

	byName := map[string]map[string]any{}
	for _, item := range items {
		name, _ := item["name"].(string)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if design, _ := item["design"].(bool); design {
			model, _ := item["type"].(string)
			key += "|" + strings.ToLower(model)
		}
		byName[key] = item
	}
	variants := idx.EquipmentVariantsOf(tag, date)
	unlockedBy := idx.enablingTech()
	start := startTime(date)
	inUse := map[string]int{}
	lines := []map[string]any{}
	for at, line := range queueOrder(idx, idx.ProductionLinesOf(tag, date)) {
		naval := idx.onDockyards(line.Equipment)
		item := byName[strings.ToLower(line.Version)+"|"+strings.ToLower(line.Equipment)]
		if item == nil && line.Version == "" {
			item = byName[strings.ToLower(line.Equipment)]
		}
		assigned := min(line.Factories, left[naval])
		left[naval] -= assigned
		used[naval] += assigned
		row := map[string]any{
			"priority":  at + 1,
			"naval":     naval,
			"equipment": line.Equipment,
			"factories": assigned,
			"requested": line.Factories,
		}
		name, general := line.Version, idx.localiseOr(line.Equipment, line.Equipment)
		cost := 0.0
		if item != nil {
			if name == "" {
				name, _ = item["label"].(string)
			}
			if text, _ := item["general_name"].(string); text != "" {
				general = text
			} else if text, _ := item["label"].(string); text != "" && line.Version != "" {
				general = text
			}
			cost, _ = item["cost"].(float64)
			row["icon"] = item["icon"]

			if line.Version == "" {
				if tech := unlockedBy[strings.ToLower(line.Equipment)]; tech != "" {
					if _, ok := idx.spriteRecordFor("GFX_" + tech + "_medium"); ok {
						row["icon"] = "GFX_" + tech + "_medium"
					}
				}
			}
			group, _ := item["group"].(string)
			if naval && isHeavyCruiser(line, variants) {
				group = "capital_ships"
			}
			row["group"] = group
			inUse[group] += assigned

			if resources, ok := item["resources"].(map[string]float64); ok {
				scaled := map[string]float64{}
				for resource, amount := range resources {
					scaled[resource] = amount * float64(assigned)
				}
				row["resources"] = scaled
			}
		}
		if name == "" {
			name = general
		}
		row["name"], row["general_name"] = name, general

		switch {
		case assigned == 0:

			none, ok := idx.locValue("PRODUCTION_PRODUCING_NONE")
			if !ok {
				none = "§RNone§!"
			}
			row["speed"] = none
		case naval:

			daily := float64(assigned) * idx.defineNumber("NProduction.BASE_FACTORY_SPEED_NAV") * (1 + sums["industrial_capacity_dockyard"])
			if cost > 0 && daily > 0 {
				days := int(math.Ceil((1 - line.Progress) * cost / daily))
				row["speed"] = shipDateText(start.AddDate(0, 0, days))
			}
		default:
			efficiency := line.Efficiency / 100
			if efficiency <= 0 {
				efficiency = idx.defineNumber("NProduction.BASE_FACTORY_START_EFFICIENCY_FACTOR") / 100
			}
			daily := float64(assigned) * idx.defineNumber("NProduction.BASE_FACTORY_SPEED_MIL") * efficiency * (1 + sums["industrial_capacity_factory"])
			if cost > 0 {
				daily /= cost
				row["daily"] = math.Round(daily*100) / 100
				row["speed"] = idx.producingText(daily)

				row["stock"] = int(math.Floor(line.Progress + daily))
			}
		}
		if naval && line.Amount > 0 {
			row["amount"] = line.Amount
		}
		lines = append(lines, row)
	}
	categories := map[string]int{}
	for _, group := range queueCategories {
		categories[group] = inUse[group]
	}
	maxEfficiency := idx.defineNumber("NProduction.BASE_FACTORY_MAX_EFFICIENCY_FACTOR") * (1 + sums["production_factory_max_efficiency_factor"])
	return map[string]any{
		"lines":          lines,
		"military_used":  used[false],
		"military_total": built["arms_factory"],
		"naval_used":     used[true],
		"naval_total":    built["dockyard"],
		"categories":     categories,
		"modifiers": map[string]any{
			"production_output":          sums["industrial_capacity_factory"],
			"dockyards_output":           sums["industrial_capacity_dockyard"],
			"efficiency_cap":             maxEfficiency / 100,
			"efficiency_growth":          sums["production_factory_efficiency_gain_factor"],
			"production_efficiency_loss": sums["line_change_production_efficiency_factor"],
			"factory_bombing_defence":    -sums["industry_air_damage_factor"],
		},
	}
}

func queueOrder(idx *Index, lines []oobProductionLine) []oobProductionLine {
	out := make([]oobProductionLine, 0, len(lines))
	for _, line := range lines {
		if !idx.onDockyards(line.Equipment) {
			out = append(out, line)
		}
	}
	for _, line := range lines {
		if idx.onDockyards(line.Equipment) {
			out = append(out, line)
		}
	}
	return out
}

func isHeavyCruiser(line oobProductionLine, variants []equipmentVariant) bool {
	if !strings.Contains(line.Equipment, "cruiser") {
		return false
	}
	for _, variant := range variants {
		if !strings.EqualFold(variant.Name, line.Version) || !strings.EqualFold(variant.Type, line.Equipment) || variant.Block == nil {
			continue
		}
		for _, entry := range variant.Block.Entries {
			if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "modules") {
				continue
			}
			battery := strings.ToLower(ciScalar(entry.Block.Entries, "fixed_ship_battery_slot"))
			return strings.Contains(battery, "medium_battery") || strings.Contains(battery, "heavy_battery")
		}
	}
	return false
}

func (idx *Index) producingText(daily float64) string {
	key, fallback, amount := "PRODUCTION_PRODUCING_DAILY", "$AMOUNT$/day", daily
	switch {
	case daily >= 1:
	case daily*7 >= 1:
		key, fallback, amount = "PRODUCTION_PRODUCING_WEEKLY", "$AMOUNT$/week", daily*7
	default:
		key, fallback, amount = "PRODUCTION_PRODUCING_MONTHLY", "$AMOUNT$/month", daily*30
	}
	text, ok := idx.locValue(key)
	if !ok {
		text = fallback
	}
	return strings.Replace(text, "$AMOUNT$", fmt.Sprintf("%.2f", amount), 1)
}

func startTime(date string) time.Time {
	parts := parsedDate(scenarioDate(date))
	return time.Date(parts[0], time.Month(max(1, parts[1])), max(1, parts[2]), 12, 0, 0, 0, time.UTC)
}

func shipDateText(at time.Time) string {
	return fmt.Sprintf("%d %s %d", at.Day(), at.Month().String()[:3], at.Year())
}
