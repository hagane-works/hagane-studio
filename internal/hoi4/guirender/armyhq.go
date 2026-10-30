package guirender

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type hqTemplateRule struct {
	limits   []*paradox.Block
	template DivisionTemplate

	file string
	line int
}

type leaderModifier struct {
	Key   string
	Value float64
}

func (idx *Index) indexGeneralHistoryFile(file string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	var walk func(entries []*paradox.Entry, limits []*paradox.Block)
	walk = func(entries []*paradox.Entry, limits []*paradox.Block) {
		scope := limits
		for _, entry := range entries {
			if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "limit") {
				scope = append(append([]*paradox.Block{}, limits...), entry.Block)
			}
		}
		for _, entry := range entries {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			switch strings.ToLower(*entry.Key) {
			case "division_template":
				if template := idx.parseDivisionTemplate(entry.Block); template.Name != "" {
					idx.hqTemplateRules = append(idx.hqTemplateRules, hqTemplateRule{scope, template, file, entry.Line})
				}
			case "if", "every_possible_country", "every_country":
				walk(entry.Block.Entries, scope)
			}
		}
	}
	walk(root.Entries, nil)
}

func (idx *Index) ArmyHQTemplatesOf(tag, date string) []DivisionTemplate {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	out := []DivisionTemplate{}
	seen := map[string]bool{}
	for _, oob := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(oob)
		if !ok {
			continue
		}
		for _, template := range record.templates {
			if template.IsArmyHQ && !seen[template.Name] {
				seen[template.Name] = true
				out = append(out, template)
			}
		}
	}
	state := idx.decisionStartOf(tag, date)
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.hqTemplateRules) == 0 {
			continue
		}
		for _, rule := range layer.hqTemplateRules {
			if !rule.template.IsArmyHQ || seen[rule.template.Name] {
				continue
			}
			holds := true
			for _, limit := range rule.limits {
				if ok, decided := state.verdict(limit); !ok || !decided {
					holds = false
					break
				}
			}
			if holds {
				seen[rule.template.Name] = true
				out = append(out, rule.template)
			}
		}
		break
	}
	return out
}

func (idx *Index) ArmyHQSource(tag, date, name string) (string, []byte, int, bool) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	state := idx.decisionStartOf(tag, date)
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.hqTemplateRules) == 0 {
			continue
		}
		for _, rule := range layer.hqTemplateRules {
			if !rule.template.IsArmyHQ || rule.template.Name != name || rule.file == "" {
				continue
			}
			holds := true
			for _, limit := range rule.limits {
				if ok, decided := state.verdict(limit); !ok || !decided {
					holds = false
					break
				}
			}
			if holds {
				return rule.file, layer.files[rule.file], rule.line, true
			}
		}
		break
	}
	return "", nil, 0, false
}

func (idx *Index) armyHQDeployment(template DivisionTemplate) (commandPower float64, days int) {
	manpower := 0.0
	for _, count := range append(append([]RegimentCount{}, template.Regiments...), template.Support...) {
		unit := idx.subUnitFor(count.Type)
		if unit == nil {
			continue
		}
		commandPower += unit.DeploymentCost * float64(count.Count)
		manpower += unit.Manpower * float64(count.Count)
	}
	base := idx.defineNumber("NMilitary.UNIT_LEADER_MODIFIER_COOLDOWN_ON_DEPLOY")
	floor := idx.defineNumber("NMilitary.UNIT_LEADER_MODIFIER_COOLDOWN_ON_DEPLOY_MIN")
	reference := idx.defineNumber("NMilitary.UNIT_LEADER_MODIFIER_COOLDOWN_REFERENCE_MANPOWER")
	value := base
	if reference > 0 {
		value = base * manpower / reference
	}
	return commandPower, int(math.Round(math.Max(value, floor)))
}

var commanderBonusOrder = []string{
	"air_invasion_preparation", "amphibious_invasion_defence", "recon_factor", "cas_damage_reduction",
	"air_superiority_bonus_in_combat", "skill_bonus_factor", "fortification_damage", "terrain_trait_xp_gain_factor",
}

var commanderBonusLabels = map[string]string{
	"skill_bonus_factor":          "MODIFIER_UNIT_LEADER_SKILL_BONUS_FACTOR",
	"fortification_damage":        "MODIFIER_FORTIFICATION_COLLATERAL_DAMAGE",
	"amphibious_invasion_defence": "MODIFIER_NAVAL_INVASION_DEFENSE",
}

var commanderBonusPercent = map[string]bool{
	"air_invasion_preparation": true, "cas_damage_reduction": true, "air_superiority_bonus_in_combat": true,
	"amphibious_invasion_defence": true, "fortification_damage": true,
}

func sortCommanderBonuses(order []string) {
	given := append([]string{}, order...)
	rank := func(key string) int {
		for at, known := range commanderBonusOrder {
			if known == key {
				return at
			}
		}
		for at, known := range given {
			if known == key {
				return len(commanderBonusOrder) + at
			}
		}
		return math.MaxInt
	}
	sort.SliceStable(order, func(a, b int) bool { return rank(order[a]) < rank(order[b]) })
}

func (idx *Index) commanderBonusLine(key string, value float64) string {
	label := key
	for _, candidate := range []string{commanderBonusLabels[key], "MODIFIER_" + strings.ToUpper(key)} {
		if candidate == "" {
			continue
		}
		if text, ok := idx.Localise(candidate); ok {
			label = stripColour(text)
			break
		}
	}
	decimals := 1
	if at, ok := idx.modifierDecimalsOf(key); ok {
		decimals = at
	}
	number := fmt.Sprintf("%+.*f", decimals, value)
	if strings.HasSuffix(key, "_factor") || commanderBonusPercent[key] {
		number = fmt.Sprintf("%+.*f%%", decimals, value*100)
	}
	return strings.TrimSuffix(strings.TrimSpace(label), ":") + ": §G" + number + "§!"
}

func (idx *Index) abilityName(ability string) string {
	return idx.localiseOr("ABILITY_"+strings.ToUpper(ability), ability)
}

func (idx *Index) subUnitCommanderBonuses(unit *subUnitRecord) (bonuses, abilities []string) {
	values := map[string]float64{}
	order := []string{}
	for _, mod := range unit.LeaderModifiers {
		if _, seen := values[mod.Key]; !seen {
			order = append(order, mod.Key)
		}
		values[mod.Key] += mod.Value
	}
	sortCommanderBonuses(order)
	for _, key := range order {
		bonuses = append(bonuses, idx.commanderBonusLine(key, values[key]))
	}
	for _, ability := range unit.Abilities {
		abilities = append(abilities, idx.abilityName(ability))
	}
	return bonuses, abilities
}

func (idx *Index) commanderBonuses(template DivisionTemplate) []string {
	abilities := []string{}
	sums := map[string]float64{}
	order := []string{}
	for _, count := range append(append([]RegimentCount{}, template.Regiments...), template.Support...) {
		unit := idx.subUnitFor(count.Type)
		if unit == nil {
			continue
		}
		for _, ability := range unit.Abilities {
			if !containsFold(abilities, ability) {
				abilities = append(abilities, ability)
			}
		}
		for _, mod := range unit.LeaderModifiers {
			if _, seen := sums[mod.Key]; !seen {
				order = append(order, mod.Key)
			}
			sums[mod.Key] += mod.Value * float64(count.Count)
		}
	}
	sortCommanderBonuses(order)
	out := []string{}
	enables := idx.localiseOr("ENABLES_ABILITY", "Enables Ability: $NAME|H$")
	for _, ability := range abilities {
		name := idx.abilityName(ability)
		out = append(out, strings.NewReplacer("$NAME|H$", "§H"+name+"§!", "$NAME$", name, "NAME|H", "§H"+name+"§!").Replace(enables))
	}
	for _, key := range order {
		out = append(out, idx.commanderBonusLine(key, sums[key]))
	}
	return out
}

func (idx *Index) ArmyHQTemplatesPayload(tag, date string) []map[string]any {
	out := []map[string]any{}
	for _, template := range idx.ArmyHQTemplatesOf(tag, date) {
		power, days := idx.armyHQDeployment(template)
		out = append(out, map[string]any{
			"name":          template.Name,
			"icon":          idx.templateIcon(template),
			"priority":      template.Priority,
			"command_power": power,
			"deploy_days":   days,
		})
	}
	return out
}
