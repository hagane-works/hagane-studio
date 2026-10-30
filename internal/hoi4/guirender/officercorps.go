package guirender

import (
	"sort"
	"strings"
)

type grandDoctrine struct {
	Folder string
	Name   string
	Icon   string

	Tracks []string

	XPCost float64
	XPType string

	Modifiers map[string]float64

	MaxTrackColumns, MaxTrackRows int
}

func (idx *Index) indexGrandDoctrineFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		doctrine := grandDoctrine{}
		for _, field := range entry.Block.Entries {
			if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "tracks") {
				doctrine.Tracks = blockTokens(field.Block.Entries)
			}
			if field == nil || field.Key == nil || field.Block != nil {
				continue
			}
			switch strings.ToLower(*field.Key) {
			case "folder":
				doctrine.Folder = toString(scalarValue(field.Scalar))
			case "name":
				doctrine.Name = toString(scalarValue(field.Scalar))
			case "icon":
				doctrine.Icon = toString(scalarValue(field.Scalar))
			case "xp_cost":
				doctrine.XPCost, _ = toFloat(scalarValue(field.Scalar))
			case "xp_type":
				doctrine.XPType = toString(scalarValue(field.Scalar))
			case "max_track_columns":
				doctrine.MaxTrackColumns = toInt(scalarValue(field.Scalar), 0)
			case "max_track_rows":
				doctrine.MaxTrackRows = toInt(scalarValue(field.Scalar), 0)
			}
		}

		if doctrine.Folder != "" {
			doctrine.Modifiers = numericEntries(entry.Block.Entries, nil)
			delete(doctrine.Modifiers, "xp_cost")
			delete(doctrine.Modifiers, "max_track_columns")
			delete(doctrine.Modifiers, "max_track_rows")
			if _, seen := idx.grandDoctrines[*entry.Key]; !seen {
				idx.grandDoctrineOrder = append(idx.grandDoctrineOrder, *entry.Key)
			}
			idx.grandDoctrines[*entry.Key] = doctrine
		}
	}
}

func (idx *Index) grandDoctrineFor(token string) (grandDoctrine, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if doctrine, ok := layer.grandDoctrines[token]; ok {
			return doctrine, true
		}
	}
	return grandDoctrine{}, false
}

func (idx *Index) StartingGrandDoctrines(tag, date string) map[string]string {
	out := map[string]string{}
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return out
	}
	for _, token := range tokensByOwned(politics.GrandDoctrines, date, idx.ownedDLC) {
		if doctrine, ok := idx.grandDoctrineFor(token); ok {
			out[doctrine.Folder] = token
		}
	}
	return out
}

var officerSlots = []string{"army_chief", "navy_chief", "air_chief", "theorist", "high_command"}

func isSpiritCategory(group string) bool {
	return strings.HasSuffix(group, "_spirit")
}

func (idx *Index) advisorName(person *Character) string {
	for _, key := range []string{person.Name, person.IdeaToken, person.Token} {
		if key == "" {
			continue
		}
		for hop := 0; hop < 3; hop++ {
			value, ok := idx.Localise(key)
			if !ok || value == key {
				break
			}
			if !isLocPointer(value) {
				return value
			}
			key = value
		}
	}
	if person.IdeaToken != "" {
		return person.IdeaToken
	}
	return person.Name
}

func (idx *Index) OfficerCorpsPayload(tag, date string) (map[string]any, bool) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	people := idx.peopleOf(tag, date)

	if len(people) == 0 && !idx.hasCountryTag(tag) {
		return nil, false
	}

	holds := map[string]bool{}
	for _, token := range idx.StartingIdeas(tag, date) {
		holds[token] = true
	}
	bySlot := map[string][]map[string]any{}
	for _, person := range people {
		slot := strings.ToLower(person.Slot)
		wanted := false
		for _, name := range officerSlots {
			if slot == name {
				wanted = true
				break
			}
		}
		if !wanted {
			continue
		}
		item := map[string]any{"token": person.Token, "name": idx.advisorName(person), "slot": slot}
		if holds[person.Token] || (person.IdeaToken != "" && holds[person.IdeaToken]) {
			item["hired"] = true
		}

		sprite := person.Small
		if sprite == "" {
			sprite = person.Large
		}
		if sprite != "" {
			item["portrait"] = sprite
		}
		if person.Cost > 0 {
			item["cost"] = person.Cost
		}
		if len(person.Traits) > 0 {
			item["traits"] = idx.traitLabels(person.Traits)
		}
		bySlot[slot] = append(bySlot[slot], item)
	}
	for _, list := range bySlot {
		sort.Slice(list, func(a, b int) bool {
			na, _ := list[a]["name"].(string)
			nb, _ := list[b]["name"].(string)
			return na < nb
		})
	}

	spirits := []map[string]any{}
	for _, token := range idx.StartingIdeas(tag, date) {
		if !isSpiritCategory(idx.ideaCategoryOf(token)) {
			continue
		}
		item := map[string]any{"token": token, "category": idx.ideaCategoryOf(token)}
		if label, ok := idx.Localise(token); ok {
			item["label"] = label
		}
		if picture := idx.ideaPicture(token); picture != "" {
			item["icon"] = "GFX_idea_" + picture
		} else {
			item["icon"] = "GFX_idea_" + token
		}
		spirits = append(spirits, item)
	}

	counts := map[string]int{}
	for _, category := range idx.IdeaCategories() {
		for _, slot := range category.Slots {
			if slot.Character {
				counts[strings.ToLower(slot.Name)]++
			}
		}
	}
	seats := map[string]int{}
	for _, name := range officerSlots {
		seats[name] = counts[name]
	}

	doctrines := map[string]any{}
	for folder, token := range idx.StartingGrandDoctrines(tag, date) {
		doctrine, _ := idx.grandDoctrineFor(token)
		item := map[string]any{"token": token, "icon": doctrine.Icon, "icon_sprite": idx.spritePayload(doctrine.Icon)}
		if label, ok := idx.Localise(doctrine.Name); ok {
			item["label"] = label
		}
		doctrines[folder] = item
	}

	out := map[string]any{
		"tag": tag, "slots": bySlot, "spirits": spirits, "seats": seats, "doctrines": doctrines,
		"spirit_slots": idx.spiritSlots(tag, date),

		"hq_templates": idx.ArmyHQTemplatesPayload(tag, date),
	}
	if flag := idx.flagPayload(tag, idx.RulingPartyOf(tag), ""); flag != nil {
		out["flag"] = flag
	}
	return out, true
}

func (idx *Index) spiritSlots(tag, date string) map[string][]map[string]any {
	types := map[string]string{"army_spirit": "army", "navy_spirit": "navy", "air_spirit": "air"}
	start, _ := idx.countryStartAt(tag, date)
	experience := map[string]int{
		"army": idx.experience(start.ArmyXP, "NMilitary.MAX_ARMY_EXPERIENCE"),
		"navy": idx.experience(start.NavyXP, "NMilitary.MAX_NAVY_EXPERIENCE"),
		"air":  idx.experience(start.AirXP, "NMilitary.MAX_AIR_EXPERIENCE"),
	}
	out := map[string][]map[string]any{}
	seen := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, group := range layer.ideaGroupOrder {
			if seen[group] {
				continue
			}
			seen[group] = true
			kind := ""
			for scan := idx; scan != nil && kind == ""; scan = scan.fallback {
				kind = scan.ideaTypes[group]
			}
			if command, ok := types[kind]; ok {
				cost := 0.0
				for scan := idx; scan != nil; scan = scan.fallback {
					if value, ok := scan.ideaCosts[group]; ok {
						cost = value
						break
					}
				}
				out[command] = append(out[command], map[string]any{
					"group":      group,
					"cost":       cost,
					"affordable": float64(experience[command]) >= cost,
				})
			}
		}
	}
	return out
}

func isLocPointer(value string) bool {
	if !strings.Contains(value, "_") {
		return false
	}
	for _, r := range value {
		if !(r == '_' || r == '.' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}
