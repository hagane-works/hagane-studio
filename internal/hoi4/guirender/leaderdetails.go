package guirender

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var traitSlots = map[string][2]float64{
	"field_marshal":   {1, 0.5},
	"corps_commander": {0, 0.5},
	"navy_leader":     {1, 0.5},
}

func (idx *Index) indexCountryLeaderTraitFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "leader_traits") {
			continue
		}
		for _, item := range entry.Block.Entries {
			if item != nil && item.Key != nil && item.Block != nil {
				idx.countryLeaderTraits[*item.Key] = item.Block
			}
		}
	}
}

func (idx *Index) countryLeaderTraitFor(token string) *paradox.Block {
	for layer := idx; layer != nil; layer = layer.fallback {
		if block, ok := layer.countryLeaderTraits[token]; ok {
			return block
		}
	}
	return nil
}

var colourCode = regexp.MustCompile(`§.`)

func stripColour(text string) string { return colourCode.ReplaceAllString(text, "") }

func (idx *Index) modifierLine(key string, value float64) string {
	label := idx.modifierLabel(key)
	if modifierReadsPercent(key) {
		return fmt.Sprintf("%s: %+.1f%%", label, value*100)
	}
	return fmt.Sprintf("%s: %+.1f", label, value)
}

func (idx *Index) modifierLines(entries []*paradox.Entry, skip map[string]bool) []string {
	out := []string{}
	for _, field := range entries {
		if field == nil || field.Key == nil || field.Block != nil || skip[strings.ToLower(*field.Key)] {
			continue
		}
		if value, ok := toFloat(scalarValue(field.Scalar)); ok {
			out = append(out, idx.modifierLine(*field.Key, value))
		}
	}
	return out
}

var skillBonusKeys = map[string]string{
	"attack_skill": "trait_bonus_attack", "defense_skill": "trait_bonus_defense",
	"planning_skill": "trait_bonus_planning", "logistics_skill": "trait_bonus_logistics",
	"maneuvering_skill": "trait_bonus_maneuvering", "coordination_skill": "trait_bonus_coordination",
}

var locValue = regexp.MustCompile(`\$[A-Z]+(\|[^$]*)?\$`)

var proximityLevels = []string{"FRONTLINE", "CLOSE", "MEDIUM", "FAR"}

var modifierBlocks = []string{"modifier", "non_shared_modifier", "corps_commander_modifier", "field_marshal_modifier"}

func (idx *Index) leaderTraitEffects(trait *leaderTrait) []string {
	out := []string{}
	if trait == nil || trait.Block == nil {
		return out
	}
	for _, key := range leaderSkillKeys {
		value := trait.Skills[key]
		if value == 0 {
			continue
		}
		if template, ok := idx.LocaliseMarkup(skillBonusKeys[key]); ok {
			out = append(out, locValue.ReplaceAllString(template, fmt.Sprintf("%+d", value)))
		}
	}
	for _, block := range modifierBlocks {
		for _, field := range trait.Block.Entries {
			if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, block) {
				out = append(out, idx.modifierLines(field.Block.Entries, nil)...)
			}
		}
	}
	proximity := 0
	for _, field := range trait.Block.Entries {
		if field != nil && field.Key != nil && strings.EqualFold(*field.Key, "leader_default_proximity_offset") {
			if value, ok := toFloat(scalarValue(field.Scalar)); ok {
				proximity = int(value)
			}
		}
	}
	if proximity != 0 {
		level := proximityLevels[max(0, min(len(proximityLevels)-1, 1+proximity))]
		name, _ := idx.Localise("LEADER_PROXIMITY_" + level + "_TITLE")
		if template, ok := idx.LocaliseMarkup("trait_bonus_leader_default_proximity_offset"); ok {
			out = append(out, locValue.ReplaceAllString(template, stripColour(name)))
		}
	}
	return out
}

var advisorTraitSkip = map[string]bool{"sprite": true, "command_cap_increase": true, "command_power": true, "cost": true, "random": true}

func leaderTreeColumn(trait *leaderTrait) (tree string, column int, ok bool) {
	if trait == nil || !trait.HasGui {
		return "", 0, false
	}
	has := func(kind string) bool {
		for _, t := range trait.Types {
			if t == kind {
				return true
			}
		}
		return false
	}
	if has("navy") {
		return "", 0, false
	}
	switch trait.TraitType {
	case "basic_terrain_trait":
		return "terrain", 0, true
	case "assignable_terrain_trait":
		return "terrain", 1, true
	case "assignable_trait":
		if has("field_marshal") {
			return "regular", 2, true
		}
		return "regular", 1, true
	case "", "basic_trait":
		return "regular", 0, true
	}
	return "", 0, false
}

func (idx *Index) leaderTraitNames() []string {
	var layers []*Index
	for layer := idx; layer != nil; layer = layer.fallback {
		layers = append(layers, layer)
	}
	seen := map[string]bool{}
	out := []string{}
	for i := len(layers) - 1; i >= 0; i-- {
		for _, token := range layers[i].leaderTraitOrder {
			if !seen[token] {
				seen[token] = true
				out = append(out, token)
			}
		}
	}
	return out
}

func (idx *Index) LeaderDetailsPayload(tag, date, token string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	var person *Character
	for _, candidate := range idx.peopleOf(tag, date) {
		if candidate.Token == token {
			person = candidate
		}
	}
	if person == nil {
		return map[string]any{"tag": tag, "token": token}
	}
	label := func(key string) string {
		if text, ok := idx.Localise(key); ok {
			return strings.ReplaceAll(text, "\n", "")
		}
		return key
	}
	rank := leaderRank(person, "army")
	out := map[string]any{
		"tag":    tag,
		"token":  token,
		"name":   label(person.Name),
		"rank":   rank,
		"skills": idx.leaderSkills(person),
	}
	if text, ok := idx.Localise(rankLabels[rank]); ok {
		out["rank_label"] = text
	}
	if portrait := firstNonEmpty(person.Large, person.Small); portrait != "" {
		out["portrait"] = portrait
		out["portrait_sprite"] = idx.spritePayload(portrait)
	}
	traitItem := func(token string) map[string]any {
		icon := "GFX_trait_" + token
		return map[string]any{"token": token, "label": label(token), "icon": icon, "icon_sprite": idx.spritePayload(icon)}
	}

	for _, role := range person.Roles {
		if role != "advisor" || len(person.AdvisorTraits) == 0 {
			continue
		}
		advisor := map[string]any{"token": person.AdvisorTraits[0], "label": label(person.AdvisorTraits[0]), "slot": person.Slot, "effects": []string{}}
		if block := idx.countryLeaderTraitFor(person.AdvisorTraits[0]); block != nil {
			advisor["effects"] = idx.modifierLines(block.Entries, advisorTraitSkip)
			for _, field := range block.Entries {
				if field != nil && field.Key != nil && strings.EqualFold(*field.Key, "sprite") {
					if value, ok := toFloat(scalarValue(field.Scalar)); ok {
						advisor["sprite"] = int(value)
					}
				}
			}
		}
		out["advisor"] = advisor
	}
	held := map[string]bool{}
	background, earned := []map[string]any{}, []map[string]any{}
	assigned := 0
	for i := len(person.CommanderTraits) - 1; i >= 0; i-- {
		token := person.CommanderTraits[i]
		held[token] = true
		trait := idx.leaderTraitFor(token)
		item := traitItem(token)
		item["effects"] = idx.leaderTraitEffects(trait)
		if trait != nil && trait.TraitType == "personality_trait" {
			background = append(background, item)
			continue
		}
		if trait != nil && (trait.TraitType == "assignable_trait" || trait.TraitType == "assignable_terrain_trait") {
			assigned++
		}
		earned = append(earned, item)
	}
	out["background"] = background
	out["traits"] = earned
	if slots, ok := traitSlots[rank]; ok {
		free := int(math.Floor(slots[0]+slots[1]*float64(person.Skills["skill"]))) - assigned
		out["pick_slots"] = max(0, free)
	}
	trees := map[string][]map[string]any{"regular": {}, "terrain": {}}
	for _, name := range idx.leaderTraitNames() {
		trait := idx.leaderTraitFor(name)
		tree, column, ok := leaderTreeColumn(trait)
		if !ok {
			continue
		}
		item := traitItem(name)

		if short, ok := idx.Localise(name + "_short"); ok {
			item["label"] = strings.ReplaceAll(short, "\n", "")
		}
		item["row"] = trait.GuiRow
		item["column"] = column
		item["parents"] = append(append([]string{}, trait.AnyParent...), trait.AllParent...)
		item["exclusive"] = append([]string{}, trait.Exclusive...)
		item["held"] = held[name]
		trees[tree] = append(trees[tree], item)
	}
	out["regular"] = trees["regular"]
	out["terrain"] = trees["terrain"]
	sprites := map[string]any{}
	for _, name := range []string{"GFX_trait_line_vertical_tile_8x8", "GFX_trait_line_horizontal_tile_8x8", "GFX_trait_line_dotted_tile_8x8", "GFX_trait_tree_exclusive_vertical", "GFX_leader_exp_bg"} {
		sprites[name] = idx.spritePayload(name)
	}
	out["sprites"] = sprites
	return out
}
