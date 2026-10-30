package preview

import (
	"strings"
)

var ideaTypeOrder = []string{
	"country", "hidden_ideas",
	"tank_manufacturer", "naval_manufacturer", "aircraft_manufacturer",
	"materiel_manufacturer", "industrial_concern",
}

var designerTypes = map[string]bool{
	"tank_manufacturer": true, "naval_manufacturer": true,
	"aircraft_manufacturer": true, "materiel_manufacturer": true,
	"industrial_concern": true,
}

var b2PromotedKeys = map[string]bool{
	"available": true, "visible": true, "allowed_to_remove": true,
	"cancel": true, "do_effect": true, "on_add": true, "on_remove": true,
	"cancel_if_invalid": true, "targeted_modifier": true,
	"equipment_bonus": true, "rule": true, "level": true, "ledger": true,
	"cost": true,
}

type Idea struct {
	Ent *OD
	ID  int
}

func GenerateIdeasScript(ideas []*Idea, pictureRefs map[int]string) string {
	if len(ideas) == 0 {
		return "ideas = {\n}"
	}

	grouped := map[string][]*Idea{}
	var groupOrder []string
	for _, idea := range ideas {
		cat := getStr(idea.Ent, "idea_type")
		if _, ok := grouped[cat]; !ok {
			groupOrder = append(groupOrder, cat)
		}
		grouped[cat] = append(grouped[cat], idea)
	}

	var orderedTypes []string
	for _, t := range ideaTypeOrder {
		if _, ok := grouped[t]; ok {
			orderedTypes = append(orderedTypes, t)
		}
	}
	inOrdered := map[string]bool{}
	for _, t := range orderedTypes {
		inOrdered[t] = true
	}
	for _, t := range groupOrder {
		if !inOrdered[t] {
			orderedTypes = append(orderedTypes, t)
			inOrdered[t] = true
		}
	}

	lines := []string{"ideas = {"}

	for _, ideaType := range orderedTypes {
		typeIdeas := grouped[ideaType]
		lines = append(lines, "")
		lines = append(lines, "\t"+ideaType+" = {")

		var flagSource *Idea
		for _, i := range typeIdeas {
			if pyTruthy(attr(i.Ent, "category_flags")) {
				flagSource = i
				break
			}
		}
		emittedFlagKeys := map[string]bool{}
		if flagSource != nil {
			flags := getOD(flagSource.Ent, "category_flags")
			if flags != nil && flags.Len() > 0 {
				lines = append(lines, "")
				for _, k := range flags.Keys() {
					v, _ := flags.Get(k)
					var valStr string
					if b, ok := v.(bool); ok {
						if b {
							valStr = "yes"
						} else {
							valStr = "no"
						}
					} else {
						valStr = pyStr(v)
					}
					lines = append(lines, "\t\t"+k+" = "+valStr)
					emittedFlagKeys[k] = true
				}
			}
		}

		if designerTypes[ideaType] && !emittedFlagKeys["designer"] {
			lines = append(lines, "")
			lines = append(lines, "\t\tdesigner = yes")
		}

		for _, idea := range typeIdeas {
			lines = append(lines, generateIdeaScriptLines(idea, pictureRefs)...)
			lines = append(lines, "")
		}

		lines = append(lines, "\t}")
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func GenerateSingleIdeaScript(idea *Idea) string {
	lines := generateIdeaScriptLines(idea, nil)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.ReplaceAll(line, "\t", "    "))
	}
	return strings.Join(out, "\n")
}

func generateIdeaScriptLines(idea *Idea, pictureRefs map[int]string) []string {
	ent := idea.Ent
	var lines []string
	indent := "\t\t"
	inner := "\t\t\t"

	lines = append(lines, indent+getStr(ent, "idea_id")+" = {")

	if pictureRefs != nil {
		if ref, ok := pictureRefs[idea.ID]; ok && ref != "" {
			lines = append(lines, inner+"picture = "+ref)
		}
	} else if pyTruthy(attr(ent, "picture")) {
		pictureValue := pyStr(attr(ent, "picture"))
		if strings.HasPrefix(pictureValue, "GFX_idea_") {
			pictureValue = pictureValue[len("GFX_idea_"):]
		}
		lines = append(lines, inner+"picture = "+pictureValue)
	}

	emitStrippedBlock := func(key string, text string) {
		lines = append(lines, inner+key+" = {")
		for _, al := range pySplitlines(pyStrip(text)) {
			lines = append(lines, inner+"\t"+pyStrip(al))
		}
		lines = append(lines, inner+"}")
	}
	if pyTruthy(attr(ent, "allowed")) && pyStrip(getStr(ent, "allowed")) != "" {
		emitStrippedBlock("allowed", getStr(ent, "allowed"))
	}
	if pyTruthy(attr(ent, "allowed_civil_war")) && pyStrip(getStr(ent, "allowed_civil_war")) != "" {
		emitStrippedBlock("allowed_civil_war", getStr(ent, "allowed_civil_war"))
	}

	removalCost := attr(ent, "removal_cost")
	if !pyEqInt(removalCost, 0) {
		lines = append(lines, inner+"removal_cost = "+pyStr(removalCost))
	}

	costVal := attr(ent, "cost")
	if costVal == nil || !pyTruthy(costVal) {
		costVal = bigFromInt(0)
	}
	if pyTruthy(costVal) {
		z, _ := pyIntFromVal(costVal)
		lines = append(lines, inner+"cost = "+z.String())
	}

	modifier := getOD(ent, "modifier")
	if modifier != nil && modifier.Len() > 0 {
		anyNonDunder := false
		for _, k := range modifier.Keys() {
			if strings.HasPrefix(k, "__") {
				continue
			}
			v, _ := modifier.Get(k)
			if pyTruthy(v) {
				anyNonDunder = true
				break
			}
		}
		if anyNonDunder {
			lines = append(lines, inner+"modifier = {")
			for _, stat := range modifier.Keys() {
				if strings.HasPrefix(stat, "__") {
					continue
				}
				value, _ := modifier.Get(stat)
				emitRawExtras(&lines, stat, value, inner+"\t")
			}
			lines = append(lines, inner+"}")
		}
	}

	researchBonus := getOD(ent, "research_bonus")
	if researchBonus != nil && researchBonus.Len() > 0 {
		lines = append(lines, inner+"research_bonus = {")
		for _, cat := range researchBonus.Keys() {
			bonus, _ := researchBonus.Get(cat)
			lines = append(lines, inner+"\t"+cat+" = "+formatModifierStat(bonus))
		}
		lines = append(lines, inner+"}")
	}

	designerTraits := getList(ent, "designer_traits")
	if len(designerTraits) > 0 {
		lines = append(lines, inner+"traits = { "+joinPyStrList(designerTraits, " ")+" }")
	}

	for _, key := range []string{
		"available", "visible", "allowed_to_remove", "cancel",
		"do_effect", "on_add", "on_remove",
	} {
		text := pyStrip(orEmptyStr(attr(ent, key)))
		if text == "" {
			continue
		}
		lines = append(lines, inner+key+" = {")
		for _, raw := range pySplitlines(text) {
			if s := pyStrip(raw); s != "" {
				lines = append(lines, inner+"\t"+s)
			}
		}
		lines = append(lines, inner+"}")
	}

	if v, ok := attr(ent, "cancel_if_invalid").(bool); ok && !v {
		lines = append(lines, inner+"cancel_if_invalid = no")
	}

	levelVal := attr(ent, "level")
	if levelVal != nil && pyTruthy(levelVal) {
		z, _ := pyIntFromVal(levelVal)
		lines = append(lines, inner+"level = "+z.String())
	}

	ledgerVal := pyStrip(orEmptyStr(attr(ent, "ledger")))
	if ledgerVal != "" {
		lines = append(lines, inner+"ledger = "+ledgerVal)
	}

	for _, entryAny := range getList(ent, "targeted_modifier") {
		entry := asOD(entryAny)
		if entry == nil {
			continue
		}
		tag := pyStrip(orEmptyStr(entry.GetDef("tag", nil)))
		mods := asOD(entry.GetDef("modifier", nil))
		if tag == "" || mods == nil || mods.Len() == 0 {
			continue
		}
		lines = append(lines, inner+"targeted_modifier = {")
		lines = append(lines, inner+"\ttag = "+tag)
		for _, mk := range mods.Keys() {
			mv, _ := mods.Get(mk)
			lines = append(lines, inner+"\t"+mk+" = "+pyStr(mv))
		}
		lines = append(lines, inner+"}")
	}

	eb := getOD(ent, "equipment_bonus")
	if eb != nil && eb.Len() > 0 {
		lines = append(lines, inner+"equipment_bonus = {")
		if pyTruthy(eb.GetDef("instant", nil)) {
			lines = append(lines, inner+"\tinstant = yes")
		}
		for _, ek := range eb.Keys() {
			if ek == "instant" {
				continue
			}
			evAny, _ := eb.Get(ek)
			ev := asOD(evAny)
			if ev == nil || ev.Len() == 0 {
				continue
			}
			lines = append(lines, inner+"\t"+ek+" = {")
			for _, bk := range ev.Keys() {
				bvv, _ := ev.Get(bk)
				lines = append(lines, inner+"\t\t"+bk+" = "+pyStr(bvv))
			}
			lines = append(lines, inner+"\t}")
		}
		lines = append(lines, inner+"}")
	}

	rule := getOD(ent, "rule")
	if rule != nil && rule.Len() > 0 {
		lines = append(lines, inner+"rule = {")
		for _, rk := range rule.Keys() {
			rv, _ := rule.Get(rk)
			var emitVal string
			if b, ok := rv.(bool); ok {
				if b {
					emitVal = "yes"
				} else {
					emitVal = "no"
				}
			} else {
				emitVal = pyStr(rv)
			}
			lines = append(lines, inner+"\t"+rk+" = "+emitVal)
		}
		lines = append(lines, inner+"}")
	}

	aiWillDo := pyStrip(orEmptyStr(attr(ent, "ai_will_do")))
	if aiWillDo != "" {
		lines = append(lines, inner+"ai_will_do = {")

		for _, al := range pySplitlines(pyStrip(aiWillDo)) {
			lines = append(lines, inner+"\t"+pyStrip(al))
		}
		lines = append(lines, inner+"}")
	}

	rawExtras := getOD(ent, "raw_extras")
	if rawExtras != nil && rawExtras.Len() > 0 {
		for _, key := range rawExtras.Keys() {
			if b2PromotedKeys[key] {
				continue
			}
			value, _ := rawExtras.Get(key)
			emitRawExtras(&lines, key, value, inner)
		}
	}

	lines = append(lines, indent+"}")
	return lines
}
