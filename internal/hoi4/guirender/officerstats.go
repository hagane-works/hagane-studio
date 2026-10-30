package guirender

import (
	"fmt"
	"math"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type officerStat struct {
	Modifier string

	Icon string

	Base string

	Form string

	LowerIsBetter bool
}

func stat(modifier, icon, base, form string, lower bool) officerStat {
	return officerStat{Modifier: modifier, Icon: icon, Base: base, Form: form, LowerIsBetter: lower}
}

var officerStats = map[string][]officerStat{
	"army": {
		stat("land_night_attack", "GFX_MODIFIER_LAND_NIGHT_ATTACK", "NMilitary.BASE_NIGHT_ATTACK_PENALTY", "percent", false),
		stat("army_speed_factor", "GFX_MODIFIER_ARMY_SPEED_FACTOR", "", "percent", false),
		stat("planning_speed", "GFX_MODIFIER_PLANNING_SPEED", "", "factor", false),
		stat("max_planning", "GFX_MODIFIER_MAX_PLANNING", "NMilitary.PLANNING_MAX", "percent", false),
		stat("dig_in_speed", "GFX_MODIFIER_DIG_IN_SPEED", "", "factor", false),
		stat("max_dig_in", "GFX_MODIFIER_MAX_DIG_IN", "NMilitary.UNIT_DIGIN_CAP", "number", false),
		stat("experience_loss_factor", "GFX_MODIFIER_EXPERIENCE_LOSS_FACTOR", "", "factor", true),
		stat("army_bonus_air_superiority_factor", "GFX_MODIFIER_ARMY_BONUS_AIR_SUPERIORITY_FACTOR", "", "factor", false),
		stat("land_reinforce_rate", "GFX_MODIFIER_LAND_REINFORCE_RATE", "NMilitary.REINFORCE_CHANCE", "percent", false),
		stat("supply_consumption_factor", "GFX_MODIFIER_SUPPLY_CONSUMPTION_FACTOR", "", "factor", true),
		stat("attrition", "GFX_MODIFIER_ATTRITION", "", "factor", true),
		stat("out_of_supply_factor", "GFX_MODIFIER_OUT_OF_SUPPLY_FACTOR", "", "factor", true),
		stat("no_supply_grace", "GFX_MODIFIER_NO_SUPPLY_GRACE", "NMilitary.SUPPLY_GRACE", "hours", false),
		stat("org_loss_when_moving", "GFX_MODIFIER_ORG_LOSS_WHEN_MOVING", "NMilitary.HOURLY_ORG_MOVEMENT_IMPACT", "scaled", true),
	},
	"navy": {
		stat("convoy_escort_efficiency", "GFX_MODIFIER_MISSION_CONVOY_ESCORT_EFFICIENCY", "", "percent", false),
		stat("sortie_efficiency", "GFX_CARRIER_SORTIE_EFFICIENCY_FACTOR", "NNavy.BASE_CARRIER_SORTIE_EFFICIENCY", "percent", false),
		stat("naval_strike_targetting_factor", "GFX_MODIFIER_NAVAL_STRIKE_TARGETTING_FACTOR", "", "percent", false),
		stat("port_strike", "GFX_MODIFIER_PORT_STRIKE_ATTACK_FACTOR", "", "factor", false),
		stat("carrier_capacity_penalty_reduction", "GFX_MODIFIER_CARRIER_CAPACITY_PENALTY_REDUCTION", "", "factor", false),
		stat("convoy_raiding_efficiency_factor", "GFX_MODIFIER_CONVOY_RAIDING_EFFICIENCY_FACTOR", "", "factor", false),
		stat("naval_invasion_division_cap", "GFX_MODIFIER_NAVAL_INVASION_DIVISION_CAP", "NNavy.BASE_NAVAL_INVASION_DIVISION_CAP", "number", false),
		stat("naval_invasion_plan_cap", "GFX_MODIFIER_NAVAL_INVASION_PLAN_CAP", "NNavy.NAVAL_INVASION_PLAN_CAP", "number", false),
		stat("naval_invasion_prep_days", "GFX_MODIFIER_NAVAL_INVASION_PREPARATION_DAYS", "NNavy.NAVAL_INVASION_PREPARE_DAYS", "number", true),
		stat("naval_supply_consumption_factor", "GFX_MODIFIER_NAVAL_SUPPLY_CONSUMPTION_FACTOR", "", "factor", true),
	},
	"air": {
		stat("air_superiority_efficiency", "GFX_MODIFIER_AIR_SUPERIORITY_EFFICIENCY", "", "factor", false),
		stat("air_escort_efficiency", "GFX_MODIFIER_AIR_ESCORT_EFFICIENCY", "", "factor", false),
		stat("air_intercept_efficiency", "GFX_MODIFIER_AIR_INTERCEPT_EFFICIENCY", "", "factor", false),
		stat("air_cas_efficiency", "GFX_MODIFIER_AIR_CAS_EFFICIENCY", "", "factor", false),
		stat("air_nav_efficiency", "GFX_MODIFIER_AIR_NAV_EFFICIENCY", "", "factor", false),
		stat("air_strategic_bomber_defence_factor", "GFX_MODIFIER_STRATEGIC_BOMBER_DEFENCE_FACTOR", "", "factor", false),
		stat("air_strategic_bomber_bombing_factor", "GFX_MODIFIER_STRATEGIC_BOMBER_BOMBING_FACTOR", "", "factor", false),
		stat("air_ace_generation_chance_factor", "GFX_MODIFIER_AIR_ACE_GENERATION_CHANCE_FACTOR", "", "factor", false),
		stat("air_strategic_bomber_night_penalty", "GFX_MODIFIER_AIR_STRAT_BOMBER_NIGHT_PENALTY", "", "factor", true),
		stat("strategic_bomb_visibility", "GFX_MODIFIER_STRAT_BOMBING_VISIBILITY", "", "factor", true),
		stat("air_superiority_detect_factor", "GFX_MODIFIER_AIR_SUPERIORITY_DETECT_FACTOR", "", "factor", false),
		stat("air_interception_detect_factor", "GFX_MODIFIER_AIR_INTERCEPTION_DETECT_FACTOR", "", "factor", false),
		stat("air_cas_present_factor", "GFX_MODIFIER_AIR_CAS_PRESENT_FACTOR", "NMilitary.AIR_SUPPORT_BASE", "factor", false),
		stat("enemy_army_bonus_air_superiority_factor", "GFX_MODIFIER_ENEMY_ARMY_BONUS_AIR_SUPERIORITY_FACTOR", "", "factor", true),
	},
	"special_forces": {
		stat("special_forces_cap", "GFX_MODIFIER_SPECIAL_FORCES_CAP", "", "factor", false),
		stat("special_forces_attack_factor", "GFX_MODIFIER_SPECIAL_FORCES_ATTACK_FACTOR", "", "factor", false),
		stat("special_forces_training_time_factor", "GFX_MODIFIER_SPECIAL_FORCES_TRAINING_TIME_FACTOR", "", "factor", true),
		stat("special_forces_out_of_supply_factor", "GFX_MODIFIER_SPECIAL_FORCES_OUT_OF_SUPPLY_FACTOR", "", "factor", true),
		stat("special_forces_defence_factor", "GFX_MODIFIER_SPECIAL_FORCES_DEFENCE_FACTOR", "", "factor", false),
		stat("mountaineers_special_forces_contribution_factor", "GFX_MODIFIER_MOUNTAINEERS_SPECIAL_FORCES_CONTRIBUTION_FACTOR", "", "factor", false),
		stat("marines_special_forces_contribution_factor", "GFX_MODIFIER_MARINES_SPECIAL_FORCES_CONTRIBUTION_FACTOR", "", "factor", false),
		stat("paratroopers_special_forces_contribution_factor", "GFX_MODIFIER_PARATROOPERS_SPECIAL_FORCES_CONTRIBUTION_FACTOR", "", "factor", false),
		stat("rangers_special_forces_contribution_factor", "GFX_MODIFIER_RANGERS_SPECIAL_FORCES_CONTRIBUTION_FACTOR", "", "factor", false),
	},
}

func numericEntries(entries []*paradox.Entry, variables map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for _, field := range entries {
		if field == nil || field.Key == nil || field.Block != nil {
			continue
		}
		value, ok := toFloat(scalarValue(field.Scalar))
		if !ok {
			value, ok = variables[toString(scalarValue(field.Scalar))]
		}
		if ok {
			out[strings.ToLower(*field.Key)] += value
		}
	}
	return out
}

type modifierSource struct {
	Name   string
	Values map[string]float64
}

func (idx *Index) startingModifierSources(tag, date string) []modifierSource {
	out := idx.baseModifierSources(tag, date)
	sums := map[string]float64{}
	for _, source := range out {
		for key, value := range source.Values {
			sums[key] += value
		}
	}
	stability, _ := idx.startingStability(tag, date, sums)
	if source, ok := idx.stabilitySource(stability); ok {
		out = append(out, source)
	}
	return out
}

func (idx *Index) baseModifierSources(tag, date string) []modifierSource {
	variables := idx.StartingVariables(tag, date)
	out := []modifierSource{}
	held := idx.StartingIdeas(tag, date)
	groups := map[string]bool{}
	for _, token := range held {
		groups[idx.ideaCategoryOf(token)] = true
	}
	for _, category := range idx.IdeaCategories() {
		for _, slot := range category.Slots {
			if slot.Character || groups[slot.Name] {
				continue
			}
			if law := idx.DefaultIdeaOf(slot.Name); law != "" {
				held = append(held, law)
				groups[slot.Name] = true
			}
		}
	}
	for _, token := range held {
		if block := idx.ideaModifierBlockFor(token); block != nil {
			out = append(out, modifierSource{"idea " + token, numericEntries(block.Entries, variables)})
		}
		if block := idx.dynamicModifierBlockFor(token); block != nil {
			out = append(out, modifierSource{"dynamic modifier " + token, numericEntries(block.Entries, variables)})
		}
	}
	for _, name := range idx.ResearchedBy(tag, date) {
		if record := idx.technologyFor(name); record != nil && len(record.CountryModifiers) > 0 {
			out = append(out, modifierSource{"technology " + name, record.CountryModifiers})
		}
	}
	for _, token := range idx.StartingGrandDoctrines(tag, date) {
		if doctrine, ok := idx.grandDoctrineFor(token); ok && len(doctrine.Modifiers) > 0 {
			out = append(out, modifierSource{"grand doctrine " + token, doctrine.Modifiers})
		}
	}
	traits := []string{}
	if leader := idx.countryLeaderOf(tag, date); leader != nil {
		traits = append(traits, leader.LeaderTraits...)
	}
	hired := map[string]bool{}
	for _, token := range held {
		hired[token] = true
	}
	for _, person := range idx.peopleOf(tag, date) {
		if hired[person.Token] || (person.IdeaToken != "" && hired[person.IdeaToken]) {
			traits = append(traits, person.AdvisorTraits...)
		}
	}
	for _, trait := range traits {
		if block := idx.countryLeaderTraitFor(trait); block != nil {
			out = append(out, modifierSource{"trait " + trait, numericEntries(block.Entries, variables)})
		}
	}

	if block := idx.staticModifierFor(playerDifficulty); block != nil {
		out = append(out, modifierSource{"difficulty " + playerDifficulty, numericEntries(block.Entries, variables)})
	}
	out = append(out, idx.buildingModifierSources(tag, date)...)
	if idx.hasPrideOfTheFleet(tag, date) {
		if block := idx.staticModifierFor(prideOfTheFleet); block != nil {
			out = append(out, modifierSource{"pride of the fleet", numericEntries(block.Entries, variables)})
		}
	}
	return out
}

const playerDifficulty = "diff_normal_player"

func (idx *Index) indexStaticModifierFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry != nil && entry.Key != nil && entry.Block != nil {
			idx.staticModifiers[*entry.Key] = entry.Block
		}
	}
}

func (idx *Index) staticModifierFor(name string) *paradox.Block {
	for layer := idx; layer != nil; layer = layer.fallback {
		if block, ok := layer.staticModifiers[name]; ok {
			return block
		}
	}
	return nil
}

func (idx *Index) countryLeaderOf(tag, date string) *Character {
	promoted := idx.PromotedLeader(tag, date)
	party := idx.RulingPartyOf(tag)
	var first, ofParty *Character
	for _, person := range idx.peopleOf(tag, date) {
		isLeader := false
		for _, role := range person.Roles {
			if role == "country_leader" {
				isLeader = true
			}
		}
		if !isLeader {
			continue
		}
		if promoted != "" && person.Token == promoted {
			return person
		}
		if first == nil {
			first = person
		}
		if ofParty == nil && party != "" && idx.PartyOfIdeology(person.Ideology) == party {
			ofParty = person
		}
	}
	if ofParty != nil {
		return ofParty
	}
	return first
}

func (idx *Index) startingModifiers(tag, date string) map[string]float64 {
	out := map[string]float64{}
	for _, source := range idx.startingModifierSources(tag, date) {
		for key, value := range source.Values {
			out[key] += value
		}
	}
	return out
}

func (idx *Index) OfficerStatsPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	sums := idx.startingModifiers(tag, date)
	out := map[string]any{}
	sprites := map[string]any{}
	for branch, stats := range officerStats {
		items := []map[string]any{}
		for _, s := range stats {
			base := 0.0
			if s.Base != "" {
				base = idx.defineNumber(s.Base)
			}
			delta := sums[s.Modifier]
			value := base + delta

			change := delta
			text := ""
			switch s.Form {
			case "factor":
				value = 1 + base + delta
				change = value - 1
				text = fmt.Sprintf("%d%%", int(math.Round(value*100)))
			case "percent":
				text = fmt.Sprintf("%d%%", int(math.Round(value*100)))
			case "scaled":
				value = base * (1 + delta)
				text = fmt.Sprintf("%d%%", int(math.Round(value*100)))
			case "hours":
				text = fmt.Sprintf("%d H", int(math.Round(value)))
			default:
				text = fmt.Sprintf("%d", int(math.Round(value)))
			}
			colour := "Y"
			if math.Abs(change) > 1e-9 {
				if (change > 0) != s.LowerIsBetter {
					colour = "G"
				} else {
					colour = "R"
				}
			}
			items = append(items, map[string]any{"modifier": s.Modifier, "icon": s.Icon, "text": text, "colour": colour})
			sprites[s.Icon] = idx.spritePayload(s.Icon)
		}
		out[branch] = items
	}
	return map[string]any{"tag": tag, "stats": out, "sprites": sprites}
}

type datedVariable struct {
	DLC   dlcGuard
	Date  string
	Name  string
	Value float64
}

func collectVariables(entries []*paradox.Entry, date string) []datedVariable {
	return collectVariablesGuarded(entries, date, dlcGuard{})
}

func collectVariablesGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []datedVariable {
	out := []datedVariable{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "if" || key == "else_if":
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectVariablesGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "else":
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectVariablesGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "set_variable":

			name := ciScalar(entry.Block.Entries, "var")
			value, ok := toFloat(ciScalarValue(entry.Block.Entries, "value"))
			if name == "" {
				for _, field := range entry.Block.Entries {
					if field != nil && field.Key != nil && field.Block == nil {
						name = *field.Key
						value, ok = toFloat(scalarValue(field.Scalar))
						break
					}
				}
			}
			if name != "" && ok {
				out = append(out, datedVariable{DLC: guard, Date: date, Name: name, Value: value})
			}
		case isGameDate(key):
			out = append(out, collectVariablesGuarded(entry.Block.Entries, key, guard)...)
		}
	}
	return out
}

func (idx *Index) StartingVariables(tag, date string) map[string]float64 {
	out := map[string]float64{}
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return out
	}
	if date == "" {
		date = "1936.1.1"
	}
	for _, pass := range []bool{false, true} {
		for _, variable := range politics.Variables {
			if (variable.Date != "") != pass || !dateReached(variable.Date, date) || !variable.DLC.holds(idx.ownedDLC) {
				continue
			}
			out[variable.Name] = variable.Value
		}
	}
	return out
}
