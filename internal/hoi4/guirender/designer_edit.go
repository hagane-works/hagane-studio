package guirender

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

var designerGroups = map[string]bool{
	"infantry": true, "mobile": true, "armor": true,
	"combat_support": true, "mobile_combat_support": true, "armor_combat_support": true,
	"support": true,
}

func withDesignerCells(template DivisionTemplate, cells string) DivisionTemplate {
	template.RegimentCells = nil
	template.SupportCells = nil
	template.RegimentalSupportCells = nil
	template.Regiments = nil
	template.Support = nil
	template.RegimentalSupport = nil
	if strings.TrimSpace(cells) == "-" {
		return template
	}
	count := func(list []RegimentCount, kind string) []RegimentCount {
		for at := range list {
			if strings.EqualFold(list[at].Type, kind) {
				list[at].Count++
				return list
			}
		}
		return append(list, RegimentCount{Type: kind, Count: 1})
	}
	for _, part := range strings.Split(cells, ",") {
		fields := strings.Split(strings.TrimSpace(part), ":")
		if len(fields) != 4 || fields[3] == "" {
			continue
		}
		x, errX := strconv.Atoi(fields[1])
		y, errY := strconv.Atoi(fields[2])
		if errX != nil || errY != nil || x < 0 || y < 0 || x > 9 || y > 9 {
			continue
		}
		cell := RegimentCell{Type: fields[3], X: x, Y: y}
		switch fields[0] {
		case "r":
			template.RegimentCells = append(template.RegimentCells, cell)
			template.Regiments = count(template.Regiments, cell.Type)
		case "s":
			template.SupportCells = append(template.SupportCells, cell)
			template.Support = count(template.Support, cell.Type)
		case "rs":
			template.RegimentalSupportCells = append(template.RegimentalSupportCells, cell)
			template.RegimentalSupport = count(template.RegimentalSupport, cell.Type)
		}
	}
	return template
}

func (idx *Index) designerAvailable(tag, date string, armyHQ bool) []map[string]any {
	enabled := map[string]int{}
	for _, name := range idx.ResearchedBy(tag, date) {
		if record := idx.technologyFor(name); record != nil {
			for _, unit := range record.EnableSubunits {
				if _, seen := enabled[strings.ToLower(unit)]; !seen {
					enabled[strings.ToLower(unit)] = len(enabled)
				}
			}
		}
	}

	armed := map[string]bool{}
	for name := range idx.FieldedEquipment(tag, date) {
		if record := idx.equipmentFor(name); record != nil && !record.IsArchetype {
			armed[strings.ToLower(record.Archetype)] = true
		}
	}
	seen := map[string]bool{}
	units := []*subUnitRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key := range layer.subUnits {
			if seen[key] {
				continue
			}
			seen[key] = true
			unit := idx.subUnitFor(key)

			if unit == nil || !designerGroups[strings.ToLower(unit.Group)] || (armyHQ && !unit.ArmyHQ) || (!armyHQ && !unit.NonArmyHQ) {
				continue
			}
			if _, unlocked := enabled[key]; !unit.Active && !unlocked {
				continue
			}
			equipped := true
			for archetype := range unit.Need {
				if !armed[archetype] {
					equipped = false
				}
			}
			if !equipped {
				continue
			}
			units = append(units, unit)
		}
	}
	sort.Slice(units, func(a, b int) bool {
		if units[a].Active != units[b].Active {
			return units[a].Active
		}
		if !units[a].Active {
			if ra, rb := enabled[strings.ToLower(units[a].Name)], enabled[strings.ToLower(units[b].Name)]; ra != rb {
				return ra < rb
			}
		}
		return units[a].Order < units[b].Order
	})
	out := make([]map[string]any, 0, len(units))
	for _, unit := range units {
		item := map[string]any{
			"type":          unit.Name,
			"group":         strings.ToLower(unit.Group),
			"special_force": unit.SpecialForce,
			"paradrop":      unit.Paradrop,

			"divisional": unit.Divisional,
			"regimental": unit.Regimental,
			"priority":   unit.Priority,
		}
		if label, ok := idx.Localise(unit.Name); ok {
			item["label"] = label
		}
		if icon := idx.subUnitIcon(unit); icon != "" {
			item["icon"] = icon
		}

		if armyHQ {
			item["deployment_cost"] = unit.DeploymentCost
			bonuses, abilities := idx.subUnitCommanderBonuses(unit)
			if len(bonuses) > 0 {
				item["commander_bonuses"] = bonuses
			}
			if len(abilities) > 0 {
				item["abilities"] = abilities
			}
		}
		out = append(out, item)
	}
	return out
}

func (idx *Index) RegimentalSupportAvailable(tag, date string) []string {
	out := []string{}
	for _, unit := range idx.designerAvailable(tag, date, false) {
		if unit["group"] == "support" && unit["regimental"] == true {
			out = append(out, unit["type"].(string))
		}
	}
	return out
}

func (idx *Index) designerGroupTitles() map[string]string {
	out := map[string]string{}
	for group := range designerGroups {
		if title, ok := idx.Localise("group_" + group + "_title"); ok {
			out[group] = title
		}
	}
	return out
}

func (idx *Index) designerExperience(tag, date string) int {
	start, _ := idx.countryStartAt(strings.ToUpper(strings.TrimSpace(tag)), date)
	return idx.experience(start.ArmyXP, "NMilitary.MAX_ARMY_EXPERIENCE")
}

func (idx *Index) localiseOr(key, fallback string) string {
	if text, ok := idx.Localise(key); ok {
		return text
	}
	return fallback
}

func (idx *Index) inEquipmentOrder(archetypes []string) []string {
	order := func(name string) int {
		if record := idx.equipmentFor(name); record != nil {
			return record.Order
		}
		return math.MaxInt
	}
	out := append([]string{}, archetypes...)
	sort.SliceStable(out, func(a, b int) bool { return order(out[a]) < order(out[b]) })
	return out
}

func (idx *Index) productionRange(need map[string]float64, year int, fielded map[string]bool, designs ...designSet) (float64, float64) {
	cheapest, dearest := 0.0, 0.0
	for archetype, amount := range need {
		low, high := math.Inf(1), math.Inf(-1)
		if len(designs) > 0 && designs[0][strings.ToLower(archetype)] != nil {
			for _, design := range designs[0][strings.ToLower(archetype)].All {
				low = math.Min(low, design.Stats["build_cost_ic"])
				high = math.Max(high, design.Stats["build_cost_ic"])
			}
		} else if len(fielded) > 0 {
			for _, variant := range idx.variantsOf(archetype, fielded) {
				_, values, _ := idx.equipmentAt(archetype, year, map[string]bool{strings.ToLower(variant.Name): true})
				cost := values["build_cost_ic"]
				low = math.Min(low, cost)
				high = math.Max(high, cost)
			}
		}
		if math.IsInf(low, 1) {
			_, values, _ := idx.equipmentAt(archetype, year, fielded)
			low, high = values["build_cost_ic"], values["build_cost_ic"]
		}
		cheapest += low * amount
		dearest += high * amount
	}
	return cheapest, dearest
}

func templateIcons(idx *Index, templates []DivisionTemplate) map[string]string {
	out := map[string]string{}
	for _, template := range templates {
		if icon := idx.templateIcon(template); icon != "" {
			out[template.Name] = icon
		}
	}
	return out
}

type terrainMod struct {
	Attack   float64
	Defence  float64
	Movement float64
}

var designerTerrainOrder = []string{
	"forest", "hills", "mountain", "plains", "urban", "jungle", "marsh", "desert", "fort", "river", "amphibious",
}

func (idx *Index) designerTerrain(template DivisionTemplate) []map[string]any {
	lines := 0.0
	line := map[string]terrainMod{}
	support := map[string]terrainMod{}
	add := func(into map[string]terrainMod, unit *subUnitRecord, copies float64) {
		for terrain, mod := range unit.Terrain {
			sum := into[terrain]
			sum.Attack += mod.Attack * copies
			sum.Defence += mod.Defence * copies
			sum.Movement += mod.Movement * copies
			into[terrain] = sum
		}
	}
	for _, count := range template.Regiments {
		if unit := idx.subUnitFor(count.Type); unit != nil && count.Count > 0 {
			lines += float64(count.Count)
			add(line, unit, float64(count.Count))
		}
	}
	for _, count := range append(append([]RegimentCount{}, template.Support...), template.RegimentalSupport...) {
		if unit := idx.subUnitFor(count.Type); unit != nil && count.Count > 0 {
			add(support, unit, float64(count.Count))
		}
	}
	out := []map[string]any{}
	for _, terrain := range designerTerrainOrder {
		mod := support[terrain]
		if lines > 0 {
			sum := line[terrain]
			mod.Attack += sum.Attack / lines
			mod.Defence += sum.Defence / lines
			mod.Movement += sum.Movement / lines
		}
		if math.Abs(mod.Attack) < 1e-9 && math.Abs(mod.Defence) < 1e-9 && math.Abs(mod.Movement) < 1e-9 {
			continue
		}
		out = append(out, map[string]any{
			"terrain":  terrain,
			"attack":   round(mod.Attack, 4),
			"defence":  round(mod.Defence, 4),
			"movement": round(mod.Movement, 4),
		})
	}
	return out
}
