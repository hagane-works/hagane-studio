package guirender

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

var factionTerms = []string{"short_term", "medium_term", "long_term"}

var capitalShips = map[string]bool{"battleship": true, "battle_cruiser": true, "carrier": true, "heavy_cruiser": true, "sh_battleship": true}

func (idx *Index) factionLeaders(date string) []string {
	seen := map[string]bool{}
	var out []string
	for layer := idx; layer != nil; layer = layer.fallback {
		for tag := range layer.factionLed {
			if seen[tag] {
				continue
			}
			seen[tag] = true
			if _, ok := idx.factionTemplateOf(tag, date); ok {
				out = append(out, tag)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (idx *Index) factionMembers(leader, date string) []string {
	seen := map[string]bool{leader: true}
	others := []string{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for tag := range layer.factionMember {
			if seen[tag] {
				continue
			}
			seen[tag] = true
			if led, ok := idx.factionMemberOf(tag, date); ok && led == leader {
				others = append(others, tag)
			}
		}
	}
	sort.Strings(others)
	return append([]string{leader}, others...)
}

func (idx *Index) countryPower(tag, date string) float64 {
	weight := func(name string) float64 { return idx.defineNumber("NDiplomacy." + name) }
	factories, resources := 0, 0
	for _, state := range idx.effectiveStates() {
		if idx.ownerOfState(state, date) != tag {
			continue
		}
		buildings := state.buildingsAt(date)
		factories += buildings["industrial_complex"] + buildings["arms_factory"] + buildings["dockyard"]
		for _, amount := range state.Resources {
			resources += amount
		}
	}
	divisions, capital, ships, planes, bombers := 0, 0, 0, 0.0, 0.0
	for _, oob := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(oob)
		if !ok {
			continue
		}
		divisions += len(record.Divisions)
		for _, force := range record.TaskForces {
			for _, ship := range force.Ships {
				if capitalShips[strings.ToLower(ship.Definition)] {
					capital++
				} else {
					ships++
				}
			}
		}
		for equipment, amount := range record.Fielded {
			name := strings.ToLower(equipment)
			switch {
			case strings.Contains(name, "ship"):
			case strings.Contains(name, "bomber"):
				bombers += amount
			case strings.Contains(name, "plane") || strings.Contains(name, "fighter") || strings.Contains(name, "cas"):
				planes += amount
			}
		}
	}
	return weight("FACTION_POWER_INDUSTRY_WEIGHT")*float64(factories) +
		weight("FACTION_POWER_RESOURCE_WEIGHT")*float64(resources) +
		weight("FACTION_POWER_ARMY_WEIGHT")*float64(divisions) +
		weight("FACTION_POWER_NAVAL_CAPITAL_SHIP_WEIGHT")*float64(capital) +
		weight("FACTION_POWER_NAVAL_WEIGHT")*float64(ships) +
		weight("FACTION_POWER_AIR_WEIGHT")*planes +
		weight("FACTION_POWER_AIR_BOMBER_WEIGHT")*bombers
}

func (idx *Index) factionGoalOf(token string) (factionGoal, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if goal, ok := layer.factionGoals[token]; ok {
			return goal, true
		}
	}
	return factionGoal{}, false
}

func (idx *Index) factionTemplateNamed(token string) (factionTemplate, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if template, ok := layer.factionTemplates[token]; ok {
			return template, true
		}
	}
	return factionTemplate{}, false
}

func (idx *Index) FactionPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	name, nameKey, icon, leader := idx.factionNamed(tag, date)
	out := map[string]any{"tag": tag, "in_faction": name != ""}
	if name == "" {
		return out
	}
	sprites := map[string]any{}
	keep := func(ref string) string {
		if ref == "" {
			return ""
		}
		if sprite := idx.spritePayload(ref); sprite != nil {
			sprites[ref] = sprite
			return ref
		}
		return ""
	}
	out["name"] = name

	out["name_key"] = nameKey
	out["icon"] = keep(icon)
	out["leader"] = leader

	out["initiative"] = 0
	out["manifest_progress"] = 0.0

	token, _ := idx.factionTemplateOf(leader, date)
	template, _ := idx.factionTemplateNamed(token)
	if goal, ok := idx.factionGoalOf(template.Manifest); ok {
		out["manifest"] = idx.localiseOr(goal.Name, goal.Name)
		out["manifest_key"] = goal.Name
	}
	goals := map[string]any{}
	for _, term := range factionTerms {
		goals[term] = nil
	}
	for _, id := range template.Goals {
		goal, ok := idx.factionGoalOf(id)
		if !ok || goals[goal.Category] != nil {
			continue
		}
		if _, column := goals[goal.Category]; !column {
			continue
		}
		goals[goal.Category] = map[string]any{
			"id": id, "name": idx.localiseOr(goal.Name, goal.Name), "icon": keep("GFX_" + goal.Group),
		}
	}
	out["goals"] = goals

	members := idx.factionMembers(leader, date)
	powers := make([]float64, len(members))
	total := 0.0
	for k, member := range members {
		powers[k] = idx.countryPower(member, date)
		total += powers[k]
	}
	rows := []map[string]any{}
	for k, member := range members {
		share := 100.0 / float64(len(members))
		if total > 0 {
			share = 100 * powers[k] / total
		}
		party := idx.RulingPartyOf(member)
		row := map[string]any{"tag": member, "name": idx.CountryName(member, party), "share": int(math.Round(share))}
		if flag := idx.flagPayload(member, party, "medium"); flag != nil {
			row["flag"] = flag
		}
		if colour, ok := idx.CountryMapColor(member); ok {
			row["color"] = []int{int(colour[0]), int(colour[1]), int(colour[2])}
		}
		rows = append(rows, row)
	}
	out["members"] = rows

	type ranked struct {
		leader string
		power  float64
	}
	var all []ranked
	for _, other := range idx.factionLeaders(date) {
		sum := 0.0
		for _, member := range idx.factionMembers(other, date) {
			sum += idx.countryPower(member, date)
		}
		all = append(all, ranked{other, sum})
	}
	sort.SliceStable(all, func(a, b int) bool { return all[a].power > all[b].power })
	rank := len(all)
	for k, r := range all {
		if r.leader == leader {
			rank = k + 1
			break
		}
	}
	out["rank"] = rank

	ranking := []map[string]any{}
	for _, r := range all {
		ranking = append(ranking, map[string]any{"leader": r.leader, "power": math.Round(r.power*10) / 10})
	}
	out["ranking"] = ranking
	rankText, ok := idx.locValue("FACTION_RANKING")
	if !ok {
		rankText = "Global Rank: $VAL|H$"
	}
	out["rank_text"] = locSlot.ReplaceAllStringFunc(rankText, func(slot string) string {
		if strings.Contains(slot, "|H") || strings.Contains(slot, "|Y") {
			return "§H" + strconv.Itoa(rank) + "§!"
		}
		return strconv.Itoa(rank)
	})
	intel := idx.factionIntel(0)

	intel["portrait"] = keep(vacantPortrait)
	out["intel"] = intel
	out["sprites"] = sprites
	return out
}

var factionIntelBranches = []struct{ key, fallback string }{
	{"FACTION_INTEL_HEADER_CRYPTOLOGY", "Cryptology"},
	{"FACTION_INTEL_HEADER_COUNTER", "Counter Intelligence"},
	{"FACTION_INTEL_HEADER_OPERATIONS", "Operations"},
}

const vacantPortrait = "GFX_idea_slot_political_advisor"

func (idx *Index) factionIntel(initiative float64) map[string]any {
	costText := func(key, fallback string, cost float64) string {
		text, ok := idx.locValue(key)
		if !ok {
			text = fallback
		}
		value := strconv.FormatFloat(cost, 'f', -1, 64)
		return strings.TrimSpace(locSlot.ReplaceAllString(text, value))
	}
	unlockCost := idx.defineNumber("NFactions.FACTION_INTELLIGENCE_UNLOCK_COST")
	spymasterCost := idx.defineNumber("NOperatives.BECOME_SPYMASTER_FI_COST")
	unlock := costText("FACTION_UNLOCK_BUTTON", "Unlock £faction_initiative_texticon£ $VALUE$", unlockCost)
	branches := []map[string]any{}
	for _, branch := range factionIntelBranches {
		branches = append(branches, map[string]any{
			"header": idx.localiseOr(branch.key, branch.fallback), "button": unlock, "can_take": initiative >= unlockCost,
		})
	}
	return map[string]any{

		"sharing_bonus":      "§H0%§!",
		"vacant":             idx.localiseOr("FACTION_INTEL_NO_ADVISOR", "No intel advisor assigned"),
		"spymaster_button":   costText("FACTION_INTEL_UNLOCK_SPYMASTER", "Become Spymaster £faction_initiative_texticon£ $VALUE$", spymasterCost),
		"spymaster_can_take": initiative >= spymasterCost,
		"branches":           branches,
	}
}
