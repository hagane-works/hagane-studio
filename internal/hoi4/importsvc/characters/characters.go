package characters

import (
	"fmt"
	"math/big"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

var charKnownRoot = map[string]bool{
	"name": true, "portraits": true, "gender": true,
	"corps_commander": true, "field_marshal": true, "navy_leader": true,
	"scientist": true, "advisor": true,
	"country_leader":  true,
	"operative":       true,
	"can_be_captured": true,
}

func extractPortraitPaths(portraits *paradox.Block) (string, string) {
	large, small := "", ""
	for _, ptype := range []string{"civilian", "army", "navy"} {
		for _, entry := range findAll(portraits.Entries, ptype) {
			if entry.Block == nil {
				continue
			}
			sub := entry.Block.Entries
			l := scalarOr(sub, "large", strScalar(""))
			s := scalarOr(sub, "small", strScalar(""))
			if truthy(l) && large == "" {
				large = pyStr(l)
			}
			if truthy(s) && small == "" {
				small = pyStr(s)
			}
		}
	}
	return large, small
}

var clNonModifierScalars = map[string]bool{
	"random": true, "sprite": true, "cost": true,
	"mutually_exclusive": true, "custom_modifier_tooltip": true,
}

var clRawBlockKeys = map[string]bool{
	"equipment_bonus": true, "custom_modifier_tooltip": true,
}

func parseTargetedModifierBlock(block []*paradox.Entry) map[string]any {
	tagVal := ""
	mods := map[string]any{}
	for _, sub := range block {
		if sub.Key == nil {
			continue
		}
		sk := *sub.Key
		if sk == "tag" {
			if sub.Block != nil {

				tagVal = ""
			} else {
				tagVal = pyUpper(pyStrip(pyStr(sub.Scalar)))
			}
		} else if sub.Block != nil {

			mods[sk] = importsvc.BlockToText(sub.Block, 0)
		} else {
			if f, ok := pyFloatOfStr(pyStr(sub.Scalar)); ok {
				mods[sk] = f
			} else {
				mods[sk] = pyStr(sub.Scalar)
			}
		}
	}
	if tagVal == "" {
		return nil
	}
	return map[string]any{"tag": tagVal, "modifier": mods}
}

func entryKeyStr(e *paradox.Entry) string {
	if e.Key == nil {
		return "None"
	}
	return *e.Key
}

func ImportCountryLeaderTraits(text string) ([]map[string]any, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("custom_trait")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, summary, &PyError{"ParseError", err.Error()}
	}
	expandAtConstants(ast.Entries)

	lt := blockOf(ast.Entries, "leader_traits")
	if lt == nil {
		return []map[string]any{}, summary, nil
	}

	results := []map[string]any{}
	for _, traitEntry := range lt.Entries {
		if traitEntry.Block == nil {
			continue
		}
		traitID := entryKeyStr(traitEntry)
		results = append(results, mapCountryLeaderTrait(traitID, traitEntry.Block.Entries, summary))
	}
	summary.Total = len(results)
	return results, summary, nil
}

func mapCountryLeaderTrait(traitID string, block []*paradox.Entry, summary *importsvc.ImportSummary) map[string]any {
	modifier := map[string]any{}
	extraFields := map[string]any{}
	rawExtras := map[string]any{}
	targeted := []any{}
	result := map[string]any{
		"trait_id":   traitID,
		"trait_type": "country_leader",
		"name_en":    "", "name_zh": "", "desc_en": "", "desc_zh": "",
		"modifier":          modifier,
		"ai_will_do":        "",
		"random":            true,
		"extra_fields":      extraFields,
		"raw_extras":        rawExtras,
		"targeted_modifier": targeted,
	}

	for _, entry := range block {
		key := entryKeyStr(entry)
		isBlock := entry.Block != nil

		switch {
		case key == "random" && !isBlock:
			result["random"] = yesEq(entry.Scalar)

		case key == "ai_will_do" && isBlock:
			result["ai_will_do"] = importsvc.BlockToText(entry.Block, 0)

		case key == "targeted_modifier" && isBlock:
			parsed := parseTargetedModifierBlock(entry.Block.Entries)
			if parsed != nil {
				targeted = append(targeted, parsed)
				result["targeted_modifier"] = targeted
			} else {
				accumulateRawExtra(rawExtras, key, importsvc.BlockToText(entry.Block, 0))
				summary.AddPreserved(key)
			}

		case clRawBlockKeys[key] && isBlock:
			accumulateRawExtra(rawExtras, key, importsvc.BlockToText(entry.Block, 0))
			summary.AddPreserved(key)

		case key == "cost" && !isBlock:
			extraFields[key] = traitIntValue(entry.Scalar, summary, traitID, key)

		case clNonModifierScalars[key] && !isBlock:
			extraFields[key] = pyStrip(pyStr(entry.Scalar))

		case !isBlock:
			if f, ok := pyFloatOfStr(pyStr(entry.Scalar)); ok {
				modifier[key] = f
			} else {
				modifier[key] = pyStr(entry.Scalar)
			}

		default:
			accumulateRawExtra(rawExtras, key, importsvc.BlockToText(entry.Block, 0))
			summary.AddPreserved(key)
		}
	}
	return result
}

func traitIntValue(s paradox.Scalar, summary *importsvc.ImportSummary, traitID, fieldName string) any {
	z, raw, isInt := importsvc.CoerceTraitInt(s, summary, traitID, fieldName)
	if isInt {
		return z
	}
	return raw
}

var ulSkillKeys = map[string]bool{
	"attack_skill": true, "defense_skill": true, "logistics_skill": true,
	"planning_skill": true, "maneuvering_skill": true, "coordination_skill": true,
}

var ulSkillFactorKeys = map[string]bool{
	"attack_skill_factor": true, "defense_skill_factor": true,
	"logistics_skill_factor": true, "planning_skill_factor": true,
	"maneuvering_skill_factor": true, "coordination_skill_factor": true,
}

var ulIntKeys = map[string]bool{
	"gui_row": true, "gui_column": true, "gain_xp_on_spotting": true, "cost": true,
}

var ulBoolKeys = map[string]bool{
	"show_in_combat": true,
}

var ulStringKeys = map[string]bool{
	"trait_type": true, "mutually_exclusive": true, "enable_ability": true,
	"override_effect_tooltip": true, "custom_effect_tooltip": true,
	"custom_prerequisite_tooltip": true, "custom_gain_xp_trigger_tooltip": true,
}

var ulScriptKeys = map[string]bool{
	"allowed": true, "prerequisites": true, "gain_xp": true,
	"gain_xp_leader": true, "on_add": true, "on_remove": true,
	"daily_effect": true,
}

var ulModifierKeys = map[string]bool{
	"non_shared_modifier": true, "corps_commander_modifier": true,
	"field_marshal_modifier": true,
}

func ImportUnitLeaderTraits(text string) ([]map[string]any, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("custom_trait")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, summary, &PyError{"ParseError", err.Error()}
	}
	expandAtConstants(ast.Entries)

	lt := blockOf(ast.Entries, "leader_traits")
	if lt == nil {
		return []map[string]any{}, summary, nil
	}

	results := []map[string]any{}
	for _, traitEntry := range lt.Entries {
		if traitEntry.Block == nil {
			continue
		}
		traitID := entryKeyStr(traitEntry)
		results = append(results, mapUnitLeaderTrait(traitID, traitEntry.Block.Entries, summary))
	}
	summary.Total = len(results)
	return results, summary, nil
}

func flatToMap(fm *importsvc.FlatModifiers) map[string]any {
	out := map[string]any{}
	for _, k := range fm.Keys {
		v, _ := fm.Get(k)
		out[k] = sv(v)
	}
	return out
}

func mapUnitLeaderTrait(traitID string, block []*paradox.Entry, summary *importsvc.ImportSummary) map[string]any {
	extraFields := map[string]any{}
	rawExtras := map[string]any{}
	targeted := []any{}
	result := map[string]any{
		"trait_id":   traitID,
		"trait_type": "unit_leader",
		"name_en":    "", "name_zh": "", "desc_en": "", "desc_zh": "",
		"modifier":          map[string]any{},
		"ai_will_do":        "",
		"random":            true,
		"extra_fields":      extraFields,
		"raw_extras":        rawExtras,
		"targeted_modifier": targeted,
	}

	for _, entry := range block {
		key := entryKeyStr(entry)
		isBlock := entry.Block != nil

		switch {
		case key == "ai_will_do" && isBlock:
			result["ai_will_do"] = importsvc.BlockToText(entry.Block, 0)
		case key == "targeted_modifier" && isBlock:
			parsed := parseTargetedModifierBlock(entry.Block.Entries)
			if parsed != nil {
				targeted = append(targeted, parsed)
				result["targeted_modifier"] = targeted
			} else {
				accumulateRawExtra(rawExtras, key, importsvc.BlockToText(entry.Block, 0))
				summary.AddPreserved(key)
			}
		case key == "modifier" && isBlock:
			result["modifier"] = flatToMap(importsvc.ParseFlatModifierBlock(entry.Block.Entries))
		case ulModifierKeys[key] && isBlock:
			extraFields[key] = flatToMap(importsvc.ParseFlatModifierBlock(entry.Block.Entries))
		case ulScriptKeys[key] && isBlock:
			extraFields[key] = importsvc.BlockToText(entry.Block, 0)
		case key == "type" && !isBlock:
			extraFields["type"] = pyStrip(pyStr(entry.Scalar))
		case ulSkillKeys[key] && !isBlock:
			extraFields[key] = traitIntValue(entry.Scalar, summary, traitID, key)
		case ulSkillFactorKeys[key] && !isBlock:
			extraFields[key] = sv(importsvc.CoerceTraitNumber(entry.Scalar))
		case ulIntKeys[key] && !isBlock:
			extraFields[key] = traitIntValue(entry.Scalar, summary, traitID, key)
		case ulBoolKeys[key] && !isBlock:
			extraFields[key] = yesEq(entry.Scalar)
		case ulStringKeys[key] && !isBlock:
			extraFields[key] = pyStrip(pyStr(entry.Scalar))
		case !isBlock:
			accumulateRawExtra(rawExtras, key, pyStrip(pyStr(entry.Scalar)))
			summary.AddPreserved(key)
		default:
			accumulateRawExtra(rawExtras, key, importsvc.BlockToText(entry.Block, 0))
			summary.AddPreserved(key)
		}
	}
	return result
}

func ImportScientistTraits(text string) ([]map[string]any, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("custom_trait")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		return nil, summary, &PyError{"ParseError", err.Error()}
	}
	expandAtConstants(ast.Entries)

	results := []map[string]any{}
	for _, entry := range ast.Entries {
		if entry.Key == nil || *entry.Key == "" || entry.Block == nil {
			continue
		}
		results = append(results, mapScientistTrait(*entry.Key, entry.Block.Entries, summary))
	}
	summary.Total = len(results)
	return results, summary, nil
}

func mapScientistTrait(traitID string, block []*paradox.Entry, summary *importsvc.ImportSummary) map[string]any {
	extraFields := map[string]any{}
	rawExtras := map[string]any{}
	result := map[string]any{
		"trait_id":   traitID,
		"trait_type": "scientist",
		"name_en":    "", "name_zh": "", "desc_en": "", "desc_zh": "",
		"modifier":     map[string]any{},
		"ai_will_do":   "",
		"random":       true,
		"extra_fields": extraFields,
		"raw_extras":   rawExtras,
	}

	for _, entry := range block {
		key := entryKeyStr(entry)
		isBlock := entry.Block != nil

		switch {
		case key == "specialization":
			token := ""
			if isBlock {
				for _, c := range entry.Block.Entries {
					if c.Key == nil && c.Block == nil {
						token = pyStr(c.Scalar)
						break
					}
				}
			} else {
				token = pyStrip(pyStr(entry.Scalar))
			}
			if token != "" {
				extraFields["specialization"] = token
			}
		case key == "icon" && !isBlock:
			extraFields["icon"] = pyStrip(pyStr(entry.Scalar))
		case key == "modifier" && isBlock:
			result["modifier"] = flatToMap(importsvc.ParseFlatModifierBlock(entry.Block.Entries))
		case !isBlock:
			accumulateRawExtra(rawExtras, key, pyStrip(pyStr(entry.Scalar)))
			summary.AddPreserved(key)
		default:
			accumulateRawExtra(rawExtras, key, importsvc.BlockToText(entry.Block, 0))
			summary.AddPreserved(key)
		}
	}
	return result
}

func ImportCharacters(text string) ([]map[string]any, *importsvc.ImportSummary, error) {
	return importCharacters(text, false)
}

func ImportCharactersStrict(text string) ([]map[string]any, *importsvc.ImportSummary, error) {
	return importCharacters(text, true)
}

func importCharacters(text string, strict bool) ([]map[string]any, *importsvc.ImportSummary, error) {
	summary := importsvc.NewImportSummary("character")
	ast, recoveries, err := paradox.Parse(text)
	if err != nil {
		return nil, summary, &PyError{"ParseError", err.Error()}
	}
	if strict {
		for _, recovery := range recoveries {
			if recovery.Code != paradox.RecoveryUnclosedBlock {
				continue
			}
			line, _ := recovery.Details["eof_line"].(int)
			return nil, summary, &PyError{"ParseError", fmt.Sprintf(
				"Expected RBRACE, got EOF '' at line %d", line)}
		}
	}
	expandAtConstants(ast.Entries)

	charsEntries := findAll(ast.Entries, "characters")
	if len(charsEntries) == 0 {
		summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
			"no_content_characters",
			nil,
			"A file in the Characters folder had no characters in it — skipped.",
		))
		return []map[string]any{}, summary, nil
	}

	results := []map[string]any{}
	for _, charsEntry := range charsEntries {
		if charsEntry.Block == nil {
			continue
		}
		for _, entry := range charsEntry.Block.Entries {
			if entry.Key == nil || entry.Block == nil {
				continue
			}
			char, err := mapOneCharacter(*entry.Key, entry.Block.Entries, summary)
			if err != nil {
				return nil, summary, err
			}
			results = append(results, char)
		}
	}
	summary.Total = len(results)
	return results, summary, nil
}

func mapOneCharacter(tag string, body []*paradox.Entry, summary *importsvc.ImportSummary) (map[string]any, error) {
	var instanceEntries []*paradox.Entry
	for _, e := range body {
		if e.Key != nil && *e.Key == "instance" && e.Block != nil {
			instanceEntries = append(instanceEntries, e)
		}
	}
	if len(instanceEntries) > 0 {
		summary.AddPreserved("instance")

		primary := append([]*paradox.Entry{}, instanceEntries[0].Block.Entries...)
		for _, e := range body {
			if !(e.Key != nil && *e.Key == "instance") {
				primary = append(primary, e)
			}
		}
		data, err := mapOneCharacter(tag, primary, summary)
		if err != nil {
			return nil, err
		}
		rawExtras := data["raw_extras"].(map[string]any)
		rawExtras["_origin_has_instance"] = true
		if len(instanceEntries) > 1 {
			additional := []any{}
			for _, e := range instanceEntries[1:] {
				additional = append(additional, importsvc.BlockToText(e.Block, 0))
			}
			rawExtras["additional_instances"] = additional
			n := len(instanceEntries)
			summary.Warnings = append(summary.Warnings, importsvc.CodedWarning(
				"character_multi_instance",
				[]importsvc.WarnParam{{Key: "tag", Value: tag}, {Key: "count", Value: n}},
				fmt.Sprintf("Character %s has %d DLC variant versions. "+
					"The editor shows the first one; the others are kept intact and "+
					"come back unchanged when you export. Use the Script Editor to "+
					"tweak the extra versions.", tag, n),
			))
		}
		return data, nil
	}

	nameVal := pyStr(scalarOr(body, "name", strScalar(tag)))
	rawExtras := map[string]any{}
	data := map[string]any{
		"tag":              tag,
		"name":             nameVal,
		"localisation_key": nameVal,
		"name_en":          nameVal,
		"role_type":        "advisor",
		"has_advisor_role": false,
		"traits":           []string{},
		"advisor_traits":   []string{},
		"skill":            big.NewInt(1), "attack_skill": big.NewInt(1),
		"defense_skill": big.NewInt(1), "planning_skill": big.NewInt(1),
		"logistics_skill": big.NewInt(1), "maneuvering_skill": big.NewInt(1),
		"coordination_skill": big.NewInt(1),
		"advisor_slot":       "", "advisor_cost": big.NewInt(150),
		"scientist_data": map[string]any{},
		"raw_extras":     rawExtras,
	}

	gender := scalarOr(body, "gender", strScalar(""))
	if truthy(gender) {
		data["gender"] = pyStr(gender)
	}

	if cbc, found := scalarFound(body, "can_be_captured"); found {
		data["can_be_captured"] = yesEq(cbc)
	}

	portraits := blockOf(body, "portraits")
	if blockTruthy(portraits) {
		large, small := extractPortraitPaths(portraits)
		data["large_portrait"] = large
		data["small_portrait"] = small
	}

	hasCommander := findOne(body, "corps_commander") != nil
	hasMarshal := findOne(body, "field_marshal") != nil
	hasNavy := findOne(body, "navy_leader") != nil
	hasScientist := findOne(body, "scientist") != nil
	hasAdvisor := findOne(body, "advisor") != nil
	hasLeader := findOne(body, "country_leader") != nil
	hasOperative := findOne(body, "operative") != nil

	switch {
	case hasMarshal:
		data["role_type"] = "field_marshal"
		if err := mapMilitary(data, blockOf(body, "field_marshal"), false); err != nil {
			return nil, err
		}
	case hasCommander:
		data["role_type"] = "general"
		if err := mapMilitary(data, blockOf(body, "corps_commander"), false); err != nil {
			return nil, err
		}
	case hasNavy:
		data["role_type"] = "admiral"
		if err := mapMilitary(data, blockOf(body, "navy_leader"), true); err != nil {
			return nil, err
		}
	case hasScientist:
		data["role_type"] = "scientist"
		if err := mapScientist(data, blockOf(body, "scientist")); err != nil {
			return nil, err
		}
	case hasOperative:
		data["role_type"] = "operative"
		mapOperative(data, blockOf(body, "operative"))
	case hasLeader:
		data["role_type"] = "country_leader"
		mapCountryLeader(data, blockOf(body, "country_leader"))
	case hasAdvisor:
		data["role_type"] = "advisor"
	default:

		data["role_type"] = "country_leader"
	}

	if hasAdvisor && data["role_type"] != "advisor" {
		data["has_advisor_role"] = true
		if err := mapAdvisor(data, blockOf(body, "advisor"), true); err != nil {
			return nil, err
		}
	} else if hasAdvisor && data["role_type"] == "advisor" {
		if err := mapAdvisor(data, blockOf(body, "advisor"), false); err != nil {
			return nil, err
		}
	}

	if hasLeader && data["role_type"] != "country_leader" {
		mapCountryLeader(data, blockOf(body, "country_leader"))
	}

	for _, entry := range body {
		if entry.Key != nil && *entry.Key != "" && !charKnownRoot[*entry.Key] {
			if entry.Block != nil {
				accumulateRawExtra(rawExtras, *entry.Key, importsvc.BlockToText(entry.Block, 0))
			} else {
				accumulateRawExtra(rawExtras, *entry.Key, sv(entry.Scalar))
			}
			summary.AddPreserved(*entry.Key)
		}
	}

	return data, nil
}

func mapCountryLeader(data map[string]any, leaderBlock *paradox.Block) {
	if !blockTruthy(leaderBlock) {
		return
	}
	lb := leaderBlock.Entries
	data["ideology"] = pyStr(scalarOr(lb, "ideology", strScalar("")))
	data["expire"] = pyStr(scalarOr(lb, "expire", strScalar("")))

	traitsBlock := blockOf(lb, "traits")
	if blockTruthy(traitsBlock) {

		if len(data["traits"].([]string)) == 0 {
			data["traits"] = bareValues(traitsBlock.Entries)
		}
	}
}

func mapMilitary(data map[string]any, milBlock *paradox.Block, isNavy bool) error {
	if !blockTruthy(milBlock) {
		return nil
	}
	mil := milBlock.Entries
	keys := []string{"skill", "attack_skill", "defense_skill"}
	if isNavy {
		keys = append(keys, "maneuvering_skill", "coordination_skill")
	} else {
		keys = append(keys, "planning_skill", "logistics_skill")
	}
	for _, k := range keys {
		z, err := pyInt(scalarOr(mil, k, intScalar(1)))
		if err != nil {
			return err
		}
		data[k] = z
	}

	traitsBlock := blockOf(mil, "traits")
	if blockTruthy(traitsBlock) {
		data["traits"] = bareValues(traitsBlock.Entries)
	}

	visibleBlock := blockOf(mil, "visible")
	if blockTruthy(visibleBlock) {
		data["military_visible"] = importsvc.BlockToText(visibleBlock, 0)
	}
	return nil
}

func mapScientist(data map[string]any, sciBlock *paradox.Block) error {
	if !blockTruthy(sciBlock) {
		return nil
	}
	sci := sciBlock.Entries
	traitsBlock := blockOf(sci, "traits")
	if blockTruthy(traitsBlock) {
		data["traits"] = bareValues(traitsBlock.Entries)
	}

	skillsBlock := blockOf(sci, "skills")
	if blockTruthy(skillsBlock) {
		specializations := []any{}
		for _, entry := range skillsBlock.Entries {
			if entry.Key != nil && *entry.Key != "" && entry.Block == nil {
				level, err := pyInt(entry.Scalar)
				if err != nil {
					return err
				}
				specializations = append(specializations, map[string]any{
					"spec":  *entry.Key,
					"level": level,
				})
			}
		}
		data["scientist_data"] = map[string]any{
			"specializations": specializations,
			"allow_multiple":  len(specializations) > 1,
		}
	}

	visibleBlock := blockOf(sci, "visible")
	if blockTruthy(visibleBlock) {
		data["scientist_visible"] = importsvc.BlockToText(visibleBlock, 0)
	}
	return nil
}

func mapOperative(data map[string]any, opBlock *paradox.Block) {
	if !blockTruthy(opBlock) {
		return
	}
	op := opBlock.Entries
	skill := scalarOr(op, "skill_level", intScalar(1))
	level := big.NewInt(1)
	if truthy(skill) {
		if z, err := pyInt(skill); err == nil {
			level = z
		}
	}
	data["operative_skill_level"] = level

	traitsBlock := blockOf(op, "traits")
	if blockTruthy(traitsBlock) {
		data["operative_traits"] = bareValues(traitsBlock.Entries)
	}
	natBlock := blockOf(op, "nationalities")
	if blockTruthy(natBlock) {
		data["operative_nationalities"] = bareValues(natBlock.Entries)
	}
	availableBlock := blockOf(op, "available")
	if blockTruthy(availableBlock) {
		data["operative_available"] = importsvc.BlockToText(availableBlock, 0)
	}
	bg := scalarOr(op, "bg", strScalar(""))
	if truthy(bg) {
		data["operative_bg"] = pyStr(bg)
	}
}

var advisorKnownFields = map[string]bool{
	"slot": true, "cost": true, "idea_token": true, "traits": true,
	"available": true, "visible": true, "removal_cost": true, "modifier": true,
	"on_add": true, "on_remove": true, "allowed": true,
	"ai_will_do": true, "can_be_fired": true, "ledger": true,
	"always_show_on_actions_tooltip": true,
}

func mapAdvisor(data map[string]any, advBlock *paradox.Block, isDual bool) error {
	if !blockTruthy(advBlock) {
		return nil
	}
	adv := advBlock.Entries
	data["advisor_slot"] = pyStr(scalarOr(adv, "slot", strScalar("political_advisor")))
	costVal := scalarOr(adv, "cost", intScalar(150))
	if truthy(costVal) {
		z, err := pyInt(costVal)
		if err != nil {
			return err
		}
		data["advisor_cost"] = z
	} else {
		data["advisor_cost"] = big.NewInt(150)
	}

	ideaToken := scalarOr(adv, "idea_token", strScalar(""))
	if truthy(ideaToken) {
		data["advisor_idea_token"] = pyStr(ideaToken)
	}

	traitsBlock := blockOf(adv, "traits")
	if blockTruthy(traitsBlock) {
		traitList := bareValues(traitsBlock.Entries)
		if isDual {
			data["advisor_traits"] = traitList
		} else {
			data["traits"] = traitList
		}
	}

	if avail := blockOf(adv, "available"); blockTruthy(avail) {
		data["advisor_available"] = importsvc.BlockToText(avail, 0)
	}
	if visible := blockOf(adv, "visible"); blockTruthy(visible) {
		data["advisor_visible"] = importsvc.BlockToText(visible, 0)
	}
	if removalCost, found := scalarFound(adv, "removal_cost"); found {
		z, err := pyInt(removalCost)
		if err != nil {
			return err
		}
		data["advisor_removal_cost"] = z
	}
	if modifier := blockOf(adv, "modifier"); blockTruthy(modifier) {
		data["advisor_modifier"] = importsvc.BlockToText(modifier, 0)
	}
	if onAdd := blockOf(adv, "on_add"); blockTruthy(onAdd) {
		data["advisor_on_add"] = importsvc.BlockToText(onAdd, 0)
	}
	if onRemove := blockOf(adv, "on_remove"); blockTruthy(onRemove) {
		data["advisor_on_remove"] = importsvc.BlockToText(onRemove, 0)
	}

	if asoat, found := scalarFound(adv, "always_show_on_actions_tooltip"); found {
		data["always_show_on_actions_tooltip"] = yesEq(asoat)
	}

	rawExtras := data["raw_extras"].(map[string]any)
	if allowedBlock := blockOf(adv, "allowed"); blockTruthy(allowedBlock) {
		rawExtras["advisor_allowed"] = importsvc.BlockToText(allowedBlock, 0)
	}

	for _, entry := range adv {
		if entry.Key != nil && *entry.Key != "" && !advisorKnownFields[*entry.Key] {
			prefixKey := "advisor_" + *entry.Key
			if entry.Block != nil {
				accumulateRawExtra(rawExtras, prefixKey, importsvc.BlockToText(entry.Block, 0))
			} else {
				accumulateRawExtra(rawExtras, prefixKey, sv(entry.Scalar))
			}
		}
	}
	return nil
}

func ImportReplacementCountryLeadersFromEvents(text string) ([]map[string]any, *importsvc.ImportSummary) {
	summary := importsvc.NewImportSummary("character")
	ast, _, err := paradox.Parse(text)
	if err != nil {
		summary.Warnings = append(summary.Warnings, importsvc.ParseSkipWarning(summary, ""))
		return []map[string]any{}, summary
	}
	expandAtConstants(ast.Entries)

	results := []map[string]any{}
	for _, entry := range ast.Entries {
		if entry.Key == nil || *entry.Key != "country_event" || entry.Block == nil {
			continue
		}
		immediate := blockOf(entry.Block.Entries, "immediate")
		if !blockTruthy(immediate) {
			continue
		}
		for _, scope := range immediate.Entries {
			if scope.Key == nil || *scope.Key == "" || scope.Block == nil {
				continue
			}
			countryTag := pyUpper(pyStrip(*scope.Key))
			scopeBody := scope.Block.Entries
			hasRetire := false
			for _, e := range scopeBody {
				if e.Key != nil && *e.Key == "retire_country_leader" &&
					e.Block == nil && pyLower(pyStrip(pyStr(e.Scalar))) == "yes" {
					hasRetire = true
					break
				}
			}
			if !hasRetire {

				continue
			}
			for _, clEntry := range findAll(scopeBody, "create_country_leader") {
				if clEntry.Block == nil {
					continue
				}
				clBody := clEntry.Block.Entries
				name := pyStrip(pyStr(scalarOr(clBody, "name", strScalar(""))))
				if name == "" {
					continue
				}
				ideology := pyStr(scalarOr(clBody, "ideology", strScalar("neutrality")))
				expire := pyStr(scalarOr(clBody, "expire", strScalar("")))
				picture := pyStr(scalarOr(clBody, "picture", strScalar("")))
				traits := []string{}
				if tb := blockOf(clBody, "traits"); blockTruthy(tb) {
					traits = bareValues(tb.Entries)
				}
				results = append(results, map[string]any{
					"tag":                 name,
					"name":                name,
					"name_en":             name,
					"country_tag":         countryTag,
					"role_type":           "country_leader",
					"is_replacement":      true,
					"replaces_vanilla_id": true,
					"ideology":            ideology,
					"expire":              expire,
					"traits":              traits,
					"advisor_traits":      []string{},
					"has_advisor_role":    false,
					"skill":               big.NewInt(1), "attack_skill": big.NewInt(1),
					"defense_skill": big.NewInt(1), "planning_skill": big.NewInt(1),
					"logistics_skill": big.NewInt(1), "maneuvering_skill": big.NewInt(1),
					"coordination_skill": big.NewInt(1),
					"advisor_slot":       "", "advisor_cost": big.NewInt(150),
					"scientist_data": map[string]any{},
					"raw_extras":     map[string]any{"_origin": "trait_overlay_event"},
					"large_portrait": picture,
					"small_portrait": "",
				})
				summary.AddPreserved("create_country_leader")
			}
		}
	}
	summary.Total = len(results)
	return results, summary
}

func IsTraitOverlayEventFile(filename string) bool {
	base := filename
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "\\"); i >= 0 {
		base = base[i+1:]
	}
	return strings.HasSuffix(pyLower(base), "_trait_overlay.txt")
}
