package guirender

import (
	"sort"
	"strconv"
	"strings"
)

var planeRoleFrames = map[string]int{
	"cas": 1, "fighter": 2, "naval_bomber": 3, "heavy_fighter": 4, "tactical_bomber": 5,
	"strategic_bomber": 6, "air_transport": 7, "scout_plane": 8, "maritime_patrol_plane": 9,
}

const airWingSize = 100

func (idx *Index) indexUnitNamesFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	if idx.unitNames == nil {
		idx.unitNames = map[string]map[string][]string{}
	}
	for _, country := range root.Entries {
		if country == nil || country.Key == nil || country.Block == nil {
			continue
		}
		tag := strings.ToUpper(*country.Key)
		for _, kind := range country.Block.Entries {
			if kind == nil || kind.Key == nil || kind.Block == nil {
				continue
			}
			unique := childBlock(kind.Block.Entries, "unique")
			if unique == nil {
				continue
			}
			if idx.unitNames[tag] == nil {
				idx.unitNames[tag] = map[string][]string{}
			}
			idx.unitNames[tag][strings.ToLower(*kind.Key)] = blockTokens(unique.Entries)
		}
	}
}

func (idx *Index) uniqueUnitNames(tag, archetype string) []string {
	for layer := idx; layer != nil; layer = layer.fallback {
		if names, ok := layer.unitNames[strings.ToUpper(tag)][strings.ToLower(archetype)]; ok {
			return names
		}
	}
	return nil
}

func (idx *Index) planeRoleKey(record, archetype *equipmentRecord, modules map[string]string) string {
	if archetype != nil && archetype.TypeOverride != "" {
		if module := idx.equipmentModuleFor(modules[archetype.TypeOverride]); module != nil {
			for _, role := range module.AddTypes {
				if _, known := planeRoleFrames[role]; known && (len(archetype.AllowedTypes) == 0 || containsWord(archetype.AllowedTypes, role)) {
					return role
				}
			}
		}
	}
	for _, candidate := range []*equipmentRecord{record, archetype} {
		if candidate == nil {
			continue
		}
		for _, kind := range candidate.Types {
			if _, known := planeRoleFrames[kind]; known {
				return kind
			}
		}
	}
	return ""
}

func (idx *Index) AirOverviewPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	designs := map[string]equipmentVariant{}
	for _, variant := range idx.EquipmentVariantsOf(tag, date) {
		designs[strings.ToLower(variant.Type)+"|"+variant.Name] = variant
	}
	states := map[int]*stateRecord{}
	for _, state := range idx.effectiveStates() {
		states[state.ID] = state
	}
	unlockedBy := idx.enablingTech()
	type wing struct {
		oobAirWing
		name, design, icon, role string
		order                    int

		oob            string
		ordinal, local int
	}
	wings := []wing{}
	for _, oob := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(oob)
		if !ok {
			continue
		}
		perState := map[int]int{}
		for ordinal, item := range record.AirWings {
			wings = append(wings, wing{oobAirWing: item, order: len(wings), oob: strings.Trim(oob, "\""), ordinal: ordinal, local: perState[item.State]})
			perState[item.State]++
		}
	}

	used := map[string]bool{}
	for _, item := range wings {
		if item.Name != "" {
			used[item.Name] = true
		}
	}
	next := map[string]int{}
	pictures := []string{"GFX_air_overview_plane_strip", "GFX_navieslist_bar_str"}
	for at := range wings {
		item := &wings[at]
		equipment := idx.equipmentFor(item.Type)
		var archetype *equipmentRecord
		archetypeName := item.Type
		if equipment != nil {
			archetypeName = firstNonEmpty(equipment.Archetype, equipment.Name)
			archetype = idx.equipmentFor(equipment.Archetype)
		}
		modules := map[string]string{}
		item.design = item.Version
		if variant, ok := designs[item.Type+"|"+item.Version]; ok && item.Version != "" {
			if block := childBlock(variant.Block.Entries, "modules"); block != nil {
				for _, slot := range block.Entries {
					if slot != nil && slot.Key != nil && slot.Block == nil {
						modules[strings.ToLower(*slot.Key)] = strings.ToLower(strings.Trim(toString(scalarValue(slot.Scalar)), `"`))
					}
				}
			}
			if icon := strings.Trim(ciScalar(variant.Block.Entries, "icon"), `"`); icon != "" {
				item.icon = icon
			}
		}
		if item.design == "" && equipment != nil {
			item.design = idx.localiseOr(equipment.Name, equipment.Name)
		}
		item.role = idx.planeRoleKey(equipment, archetype, modules)
		if item.icon == "" && equipment != nil && archetype != nil {
			item.icon = idx.designIcon(tag, equipment, archetype, unlockedBy)
		}
		if item.icon != "" {
			pictures = append(pictures, item.icon)
		}
		item.name = item.Name
		if item.name == "" {
			names := idx.uniqueUnitNames(tag, archetypeName)
			for next[archetypeName] < len(names) && used[names[next[archetypeName]]] {
				next[archetypeName]++
			}
			if next[archetypeName] < len(names) {
				item.name = names[next[archetypeName]]
				used[item.name] = true
				next[archetypeName]++
			}
		}
		if item.name == "" {
			item.name = item.design
		}
	}

	sort.SliceStable(wings, func(a, b int) bool {
		if (wings[a].Version == "") != (wings[b].Version == "") {
			return wings[a].Version != ""
		}
		return wings[a].design < wings[b].design
	})
	counts := map[string]float64{}
	rows := []map[string]any{}
	for _, item := range wings {
		counts[item.role] += item.Amount
		base := ""
		if state := states[item.State]; state != nil {
			base = idx.stateLabel(state, tag)
		}
		rows = append(rows, map[string]any{
			"name": item.name, "design": item.design, "type": item.Type, "role": item.role, "icon": item.icon,
			"count": int(item.Amount), "limit": airWingSize, "base": base, "state": item.State,
			"oob": item.oob, "ordinal": item.ordinal, "state_index": item.local, "raw_name": item.Name,
		})
	}
	roles := make([]string, 0, len(counts))
	for role := range counts {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(a, b int) bool { return planeRoleFrames[roles[a]] < planeRoleFrames[roles[b]] })
	pills := []map[string]any{}
	for _, role := range roles {
		pills = append(pills, map[string]any{"role": role, "frame": planeRoleFrames[role], "count": int(counts[role])})
	}
	say := func(key, fallback string, value int) string {
		text, ok := idx.locValue(key)
		if !ok {
			text = fallback
		}
		return locSlot.ReplaceAllString(text, strconv.Itoa(value))
	}
	return map[string]any{
		"tag": tag, "wings": rows, "pills": pills,
		"texts": map[string]any{
			"title":    idx.localiseOr("ARMY_AIR_LABEL", "Air Forces Overview"),
			"wings":    say("ARMY_AIR_WINGS_ACTIVE", "$N$ Air Wings", len(rows)),
			"theatres": say("ARMY_AIR_WINGS_THEATRES", "Active in $N$ Strategic area(s)", 0),
		},
		"sprites": idx.SpritesPayload(pictures),
	}
}
