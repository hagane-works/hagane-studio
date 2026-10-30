package guirender

import (
	"math"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type leaderTrait struct {
	Token     string
	Types     []string
	TraitType string
	Skills    map[string]int
	GuiRow    int
	GuiColumn int
	HasGui    bool
	AnyParent []string
	AllParent []string
	Exclusive []string

	Block *paradox.Block
}

var leaderSkillKeys = []string{"attack_skill", "defense_skill", "planning_skill", "logistics_skill", "maneuvering_skill", "coordination_skill"}

func (idx *Index) indexLeaderTraitFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, "leader_traits") {
			continue
		}
		for _, item := range entry.Block.Entries {
			if item == nil || item.Key == nil || item.Block == nil {
				continue
			}
			trait := &leaderTrait{Token: *item.Key, Skills: map[string]int{}, Block: item.Block}
			for _, field := range item.Block.Entries {
				if field == nil || field.Key == nil {
					continue
				}
				key := strings.ToLower(*field.Key)
				switch key {
				case "type":
					if field.Block != nil {
						trait.Types = blockTokens(field.Block.Entries)
					} else {
						trait.Types = []string{toString(scalarValue(field.Scalar))}
					}
				case "trait_type":
					trait.TraitType = toString(scalarValue(field.Scalar))
				case "gui_row":
					if value, ok := toFloat(scalarValue(field.Scalar)); ok {
						trait.GuiRow, trait.HasGui = int(value), true
					}
				case "gui_column":
					if value, ok := toFloat(scalarValue(field.Scalar)); ok {
						trait.GuiColumn, trait.HasGui = int(value), true
					}
				case "any_parent", "parent":
					if field.Block != nil {
						trait.AnyParent = append(trait.AnyParent, parentTokens(field.Block)...)
					} else {
						trait.AnyParent = append(trait.AnyParent, toString(scalarValue(field.Scalar)))
					}
				case "all_parents":
					if field.Block != nil {
						trait.AllParent = append(trait.AllParent, blockTokens(field.Block.Entries)...)
					}
				case "mutually_exclusive":
					if field.Block != nil {
						trait.Exclusive = append(trait.Exclusive, blockTokens(field.Block.Entries)...)
					} else {
						trait.Exclusive = append(trait.Exclusive, toString(scalarValue(field.Scalar)))
					}
				default:
					for _, skill := range leaderSkillKeys {
						if key == skill {
							if value, ok := toFloat(scalarValue(field.Scalar)); ok {
								trait.Skills[skill] += int(value)
							}
						}
					}
				}
			}
			if _, seen := idx.leaderTraits[trait.Token]; !seen {
				idx.leaderTraitOrder = append(idx.leaderTraitOrder, trait.Token)
			}
			idx.leaderTraits[trait.Token] = trait
		}
	}
}

func parentTokens(block *paradox.Block) []string {
	bare := []*paradox.Entry{}
	out := []string{}
	for _, field := range block.Entries {
		switch {
		case field == nil:
		case field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "traits"):
			out = append(out, blockTokens(field.Block.Entries)...)
		case field.Op == nil:
			bare = append(bare, field)
		}
	}
	return append(blockTokens(bare), out...)
}

func (idx *Index) leaderTraitFor(token string) *leaderTrait {
	for layer := idx; layer != nil; layer = layer.fallback {
		if trait, ok := layer.leaderTraits[token]; ok {
			return trait
		}
	}
	return nil
}

func (idx *Index) leaderSkills(person *Character) map[string]int {
	out := map[string]int{}
	for key, value := range person.Skills {
		out[key] = value
	}
	for _, token := range person.CommanderTraits {
		if trait := idx.leaderTraitFor(token); trait != nil {
			for key, value := range trait.Skills {
				out[key] += value
			}
		}
	}
	return out
}

var rosterRoles = map[string][]string{
	"army": {"field_marshal", "corps_commander"},
	"navy": {"navy_leader"},
}

func leaderRank(person *Character, kind string) string {
	for _, role := range rosterRoles[kind] {
		for _, held := range person.Roles {
			if held == role {
				return role
			}
		}
	}
	return ""
}

var rankLabels = map[string]string{"field_marshal": "fm_leader", "corps_commander": "cc_leader", "navy_leader": "navy_leader"}

func (idx *Index) CommanderRosterPayload(tag, date, kind string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	if kind != "navy" {
		kind = "army"
	}
	state := idx.decisionStartOf(tag, date)
	type row struct {
		person *Character
		rank   string
		skills map[string]int
	}
	rows := []row{}
	existing := 0
	for _, person := range idx.peopleOf(tag, date) {
		rank := leaderRank(person, kind)
		if rank == "" {
			continue
		}
		existing++
		if !state.holds(person.CommanderVisible) {
			continue
		}
		rows = append(rows, row{person, rank, idx.leaderSkills(person)})
	}
	sort.SliceStable(rows, func(a, b int) bool {
		if rows[a].skills["skill"] != rows[b].skills["skill"] {
			return rows[a].skills["skill"] > rows[b].skills["skill"]
		}
		return rows[a].person.Order < rows[b].person.Order
	})
	items := []map[string]any{}
	for _, r := range rows {
		name := r.person.Name
		if text, ok := idx.Localise(name); ok {
			name = text
		}
		item := map[string]any{
			"token":  r.person.Token,
			"name":   name,
			"rank":   r.rank,
			"skills": r.skills,
		}
		if label, ok := idx.Localise(rankLabels[r.rank]); ok {
			item["rank_label"] = label
		}
		portrait := firstNonEmpty(r.person.Large, r.person.Small)
		if portrait != "" {
			item["portrait"] = portrait
			item["portrait_sprite"] = idx.spritePayload(portrait)
		}
		traits := []map[string]any{}
		for _, token := range r.person.CommanderTraits {
			icon := "GFX_trait_" + token
			trait := map[string]any{"token": token, "icon": icon, "icon_sprite": idx.spritePayload(icon)}
			if label, ok := idx.Localise(token); ok {
				trait["label"] = label
			}
			traits = append(traits, trait)
		}
		item["traits"] = traits
		switch r.rank {
		case "corps_commander":
			item["capacity"] = int(idx.defineNumber("NMilitary.CORPS_COMMANDER_DIVISIONS_CAP"))
		case "field_marshal":
			item["capacity"] = int(idx.defineNumber("NMilitary.FIELD_MARSHAL_DIVISIONS_CAP"))
		case "navy_leader":
			item["capacity"] = int(idx.defineNumber("NNavy.ADMIRAL_TASKFORCE_CAP"))
		}
		items = append(items, item)
	}

	group := "ARMY"
	if kind == "navy" {
		group = "NAVY"
	}
	cost := math.Min(idx.defineNumber("NPolitics."+group+"_LEADER_COST")*float64(existing), idx.defineNumber("NPolitics."+group+"_LEADER_MAX_COST"))
	return map[string]any{"tag": tag, "kind": kind, "items": items, "new_leader_cost": int(cost)}
}
