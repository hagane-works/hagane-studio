package guirender

import (
	"sort"
	"strconv"
	"strings"
)

var shipClassOrder = []string{"battleship", "battle_cruiser", "carrier", "heavy_cruiser", "light_cruiser", "destroyer", "submarine"}

func shipClassRank(class string) int {
	for at, known := range shipClassOrder {
		if known == class {
			return at
		}
	}
	return len(shipClassOrder)
}

func (idx *Index) NavyOverviewPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	type placed struct {
		oobTaskForce
		oob     string
		ordinal int
	}
	forces := []placed{}
	for _, oob := range idx.OOBOf(tag, date) {
		if record, ok := idx.oobRecordFor(oob); ok {
			for ordinal, force := range record.TaskForces {
				forces = append(forces, placed{force, strings.Trim(oob, "\""), ordinal})
			}
		}
	}
	sort.SliceStable(forces, func(a, b int) bool { return forces[a].Name < forces[b].Name })
	classes := map[string]int{}
	regions := map[int]bool{}
	pictures := []string{"GFX_navieslist_bar_org", "GFX_navieslist_bar_str", "GFX_NV_icon_pride"}
	rows := []map[string]any{}
	for _, force := range forces {
		lead, pride := "", false
		for _, ship := range force.Ships {
			classes[ship.Definition]++
			if lead == "" || shipClassRank(ship.Definition) < shipClassRank(lead) {
				lead = ship.Definition
			}
			pride = pride || ship.Pride
		}
		icon := "GFX_unit_" + lead + "_icon_medium"
		pictures = append(pictures, icon)
		region := ""
		if found := idx.strategicRegionOf(force.Location); found != nil {
			region = idx.localiseOr(found.Name, found.Name)
			regions[found.ID] = true
		}
		rows = append(rows, map[string]any{
			"name": idx.localiseOr(force.Name, force.Name), "fleet": force.Fleet, "region": region,
			"size": len(force.Ships), "icon": icon, "class": lead, "pride": pride,
			"strength": 1.0, "organisation": 1.0,
			"oob": force.oob, "ordinal": force.ordinal, "raw_name": force.Name,
		})
	}
	present := make([]string, 0, len(classes))
	for class := range classes {
		present = append(present, class)
	}
	sort.SliceStable(present, func(a, b int) bool {
		if shipClassRank(present[a]) != shipClassRank(present[b]) {
			return shipClassRank(present[a]) < shipClassRank(present[b])
		}
		return present[a] < present[b]
	})
	pills := []map[string]any{}
	for _, class := range present {
		icon := "GFX_navalcombat_ship_icon_" + class
		pictures = append(pictures, icon)
		pills = append(pills, map[string]any{"class": class, "icon": icon, "count": classes[class]})
	}
	say := func(key, fallback string, value int) string {
		text, ok := idx.locValue(key)
		if !ok {
			text = fallback
		}
		return locSlot.ReplaceAllString(text, strconv.Itoa(value))
	}
	return map[string]any{
		"tag": tag, "task_forces": rows, "pills": pills,
		"texts": map[string]any{
			"title":    idx.localiseOr("ARMY_NAVY_LABEL", "Navy Overview"),
			"fleets":   say("ARMY_NAVIES_ACTIVE", "$N$ Fleets", len(rows)),
			"theatres": say("ARMY_NAVIES_THEATRES", "Active in $N$ Strategic area(s)", len(regions)),
		},
		"sprites": idx.SpritesPayload(pictures),
	}
}
