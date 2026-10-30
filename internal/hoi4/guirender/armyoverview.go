package guirender

import (
	"sort"
	"strconv"
	"strings"
)

func (idx *Index) namesGroupFor(key string) *namesGroup {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return nil
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		if group, ok := layer.namesGroups[key]; ok {
			return group
		}
	}
	return nil
}

func (idx *Index) orderedDivisionName(groupKey string, order int) string {
	group := idx.namesGroupFor(groupKey)
	if group == nil || order <= 0 {
		return ""
	}
	for _, item := range group.Ordered {
		if item.Number == order {
			return divisionName(item.Name, order)
		}
	}
	if group.Fallback != "" {
		return divisionName(group.Fallback, order)
	}
	return ""
}

func experienceLevel(experience float64) int {
	level := 0
	for _, threshold := range []float64{0.1, 0.3, 0.75, 0.9} {
		if experience+1e-9 >= threshold {
			level++
		}
	}
	return level
}

func (idx *Index) ArmyOverviewPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	templates := map[string]DivisionTemplate{}
	for _, template := range idx.DivisionTemplatesOf(tag, date) {
		templates[template.Name] = template
	}
	type division struct {
		oobDivision
		name  string
		icon  string
		order int

		oob     string
		ordinal int
	}
	divisions := []division{}
	battalions, special := 0, 0
	for _, oob := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(oob)
		if !ok {
			continue
		}
		for ordinal, item := range record.Divisions {
			template, known := templates[item.Template]
			name := item.Name
			if name != "" {
				name = idx.localiseOr(name, name)
			} else if known {
				name = idx.orderedDivisionName(template.NamesGroup, item.NameOrder)
			}
			if name == "" {
				name = item.Template
			}
			entry := division{oobDivision: item, name: name, order: len(divisions), oob: strings.Trim(oob, "\""), ordinal: ordinal}
			if known {
				entry.icon = idx.templateIcon(template)
				for _, cell := range template.AllCells() {
					battalions++
					if unit := idx.subUnitFor(cell.Type); unit != nil && unit.SpecialForce {
						special++
					}
				}
			}
			divisions = append(divisions, entry)
		}
	}
	sort.SliceStable(divisions, func(a, b int) bool { return divisions[a].name < divisions[b].name })

	counts := map[string]int{}
	first := map[string]int{}
	for _, item := range divisions {
		if _, seen := first[item.icon]; !seen {
			first[item.icon] = item.order
		}
		counts[item.icon]++
	}
	types := make([]string, 0, len(counts))
	for icon := range counts {
		types = append(types, icon)
	}
	sort.Slice(types, func(a, b int) bool {
		if counts[types[a]] != counts[types[b]] {
			return counts[types[a]] > counts[types[b]]
		}
		return first[types[a]] < first[types[b]]
	})
	pictures := []string{"GFX_unit_level", "GFX_navieslist_bar_org", "GFX_navieslist_bar_str"}
	pills := []map[string]any{}
	for _, icon := range types {
		pills = append(pills, map[string]any{"icon": icon, "count": counts[icon]})
		if icon != "" {
			pictures = append(pictures, icon)
		}
	}
	rows := []map[string]any{}
	for _, item := range divisions {
		rows = append(rows, map[string]any{
			"name": item.name, "template": item.Template, "icon": item.icon,
			"level": experienceLevel(item.Experience), "strength": item.Equipment, "organisation": 1.0,

			"oob": item.oob, "ordinal": item.ordinal, "experience": item.Experience, "equipment": item.Equipment,
			"location": item.Location, "officer": item.Officer,
		})
	}
	capBase := idx.defineNumber("NCountry.SPECIAL_FORCES_CAP_BASE")
	capMin := idx.defineNumber("NCountry.SPECIAL_FORCES_CAP_MIN")
	specialCap := int(capBase * float64(battalions-special))
	if float64(specialCap) < capMin {
		specialCap = int(capMin)
	}
	say := func(key, fallback string, value int) string {
		text, ok := idx.locValue(key)
		if !ok {
			text = fallback
		}
		return locSlot.ReplaceAllString(text, strconv.Itoa(value))
	}
	templateNames := make([]string, 0, len(templates))
	for _, template := range idx.DivisionTemplatesOf(tag, date) {
		templateNames = append(templateNames, template.Name)
	}
	return map[string]any{
		"tag": tag, "divisions": rows, "pills": pills,

		"templates":          templateNames,
		"special_forces_cap": specialCap,
		"texts": map[string]any{
			"title":       idx.localiseOr("ARMY_ARMY_LABEL", "Division Overview"),
			"divisions":   say("ARMY_DIVISIONS_ACTIVE", "$N$ Divisions", len(divisions)),
			"battalions":  say("BATTALIONS_COUNT", "$NUM$ Battalions", battalions),
			"theatres":    say("ARMY_DIVISIONS_THEATRES", "Active in $N$ Theater(s)", 0),
			"special_cap": strconv.Itoa(specialCap),
		},
		"sprites": idx.SpritesPayload(pictures),
	}
}
