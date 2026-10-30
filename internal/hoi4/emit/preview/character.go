package preview

import "strings"

var advisorEmittedRawKeys = map[string]bool{
	"advisor_slot": true, "advisor_cost": true, "advisor_idea_token": true,
	"advisor_traits": true, "advisor_available": true, "advisor_visible": true,
	"advisor_removal_cost": true, "advisor_modifier": true,
	"advisor_on_add": true, "advisor_on_remove": true, "advisor_allowed": true,
}

func joinStrList(items []any, sep string) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, it.(string))
	}
	return strings.Join(parts, sep)
}

func joinPyStrList(items []any, sep string) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, pyStr(it))
	}
	return strings.Join(parts, sep)
}

func emitUnitLeaderVisible(char *OD) []string {
	return emitVisibleBlock(getStr(char, "military_visible"), 3)
}

func GenerateCharacterScript(
	char *OD, countryTag string,
	hasPortraitImage, hasAdvisorImage bool,
	portraitPathOverride, advisorPathOverride string,
) string {
	tag := pyUpper(countryTag)
	slug := ExtractSlug(getStr(char, "tag"), tag)
	var lines []string

	lines = append(lines, "characters = {")
	lines = append(lines, "\t"+getStr(char, "tag")+" = {")
	lines = append(lines, "\t\tname = "+CharacterLocKey(char))

	if pyTruthy(attr(char, "gender")) {
		lines = append(lines, "\t\tgender = "+pyStr(attr(char, "gender")))
	}

	if v, ok := attr(char, "can_be_captured").(bool); ok && !v {
		lines = append(lines, "\t\tcan_be_captured = no")
	}

	portraitPath := portraitPathOverride
	if portraitPath == "" {
		portraitPath = ExportPortraitPath(tag, slug)
	}
	advisorPath := advisorPathOverride
	if advisorPath == "" {
		advisorPath = ExportAdvisorIconPath(tag, slug)
	}
	if portraitPathOverride != "" {
		hasPortraitImage = true
	}
	if advisorPathOverride != "" {
		hasAdvisorImage = true
	}

	roleType := getStr(char, "role_type")
	roleUsesArmyLarge := roleType == "general" || roleType == "field_marshal" || roleType == "scientist"
	roleUsesNavyLarge := roleType == "admiral"

	emitCivilianLarge := hasPortraitImage
	emitArmyLarge := roleUsesArmyLarge && hasPortraitImage
	emitNavyLarge := roleUsesNavyLarge && hasPortraitImage
	emitArmySmall := hasAdvisorImage

	if emitCivilianLarge || emitArmyLarge || emitNavyLarge || emitArmySmall {
		lines = append(lines, "\t\tportraits = {")
		if emitCivilianLarge {
			lines = append(lines, "\t\t\tcivilian = {")
			lines = append(lines, "\t\t\t\tlarge = \""+portraitPath+"\"")
			lines = append(lines, "\t\t\t}")
		}
		if emitArmyLarge || emitArmySmall {
			lines = append(lines, "\t\t\tarmy = {")
			if emitArmyLarge {
				lines = append(lines, "\t\t\t\tlarge = \""+portraitPath+"\"")
			}
			if emitArmySmall {
				lines = append(lines, "\t\t\t\tsmall = \""+advisorPath+"\"")
			}
			lines = append(lines, "\t\t\t}")
		}
		if emitNavyLarge {
			lines = append(lines, "\t\t\tnavy = {")
			lines = append(lines, "\t\t\t\tlarge = \""+portraitPath+"\"")
			lines = append(lines, "\t\t\t}")
		}
		lines = append(lines, "\t\t}")
	}

	traitsStr := ""
	if pyTruthy(attr(char, "traits")) {
		traitsStr = joinStrList(getList(char, "traits"), " ")
	}

	switch roleType {
	case "general", "field_marshal":
		kw := "corps_commander"
		if roleType == "field_marshal" {
			kw = "field_marshal"
		}
		lines = append(lines, "\t\t"+kw+" = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskill = "+pyStr(attr(char, "skill")))
		lines = append(lines, "\t\t\tattack_skill = "+pyStr(attr(char, "attack_skill")))
		lines = append(lines, "\t\t\tdefense_skill = "+pyStr(attr(char, "defense_skill")))
		lines = append(lines, "\t\t\tplanning_skill = "+pyStr(attr(char, "planning_skill")))
		lines = append(lines, "\t\t\tlogistics_skill = "+pyStr(attr(char, "logistics_skill")))
		lines = append(lines, "\t\t\tlegacy_id = -1")
		lines = append(lines, emitUnitLeaderVisible(char)...)
		lines = append(lines, "\t\t}")

	case "admiral":
		lines = append(lines, "\t\tnavy_leader = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskill = "+pyStr(attr(char, "skill")))
		lines = append(lines, "\t\t\tattack_skill = "+pyStr(attr(char, "attack_skill")))
		lines = append(lines, "\t\t\tdefense_skill = "+pyStr(attr(char, "defense_skill")))
		lines = append(lines, "\t\t\tmaneuvering_skill = "+pyStr(attr(char, "maneuvering_skill")))
		lines = append(lines, "\t\t\tcoordination_skill = "+pyStr(attr(char, "coordination_skill")))
		lines = append(lines, "\t\t\tlegacy_id = -1")
		lines = append(lines, emitUnitLeaderVisible(char)...)
		lines = append(lines, "\t\t}")

	case "scientist":
		sci := getOD(char, "scientist_data")
		if sci == nil {
			sci = NewOD()
		}
		specs := asList(sci.GetDef("specializations", []any{}))
		lines = append(lines, "\t\tscientist = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskills = {")
		if len(specs) > 0 {
			for _, entryAny := range specs {
				entry := asOD(entryAny)
				spec := entry.GetDef("spec", "specialization_land")
				level := entry.GetDef("level", bigFromInt(2))
				lines = append(lines, "\t\t\t\t"+pyStr(spec)+" = "+pyStr(level))
			}
		} else {
			lines = append(lines, "\t\t\t\tspecialization_land = 2")
		}
		lines = append(lines, "\t\t\t}")
		lines = append(lines, emitVisibleBlock(getStr(char, "scientist_visible"), 3)...)
		lines = append(lines, "\t\t}")

	case "operative":
		lines = append(lines, "\t\toperative = {")
		skillLevel := attr(char, "operative_skill_level")
		if !pyTruthy(skillLevel) {
			skillLevel = bigFromInt(1)
		}
		if pyTruthy(skillLevel) && !pyEqInt(skillLevel, 1) {
			z, _ := pyIntFromVal(skillLevel)
			lines = append(lines, "\t\t\tskill_level = "+z.String())
		}
		opTraits := getList(char, "operative_traits")
		if len(opTraits) > 0 {
			lines = append(lines, "\t\t\ttraits = { "+joinPyStrList(opTraits, " ")+" }")
		}
		nats := getList(char, "operative_nationalities")
		if len(nats) > 0 {
			lines = append(lines, "\t\t\tnationalities = { "+joinPyStrList(nats, " ")+" }")
		}
		opAvailable := pyStrip(orEmptyStr(attr(char, "operative_available")))
		if opAvailable != "" {
			lines = append(lines, "\t\t\tavailable = {")
			lines = append(lines, indentBlock(opAvailable, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		bg := pyStrip(orEmptyStr(attr(char, "operative_bg")))
		if bg != "" {
			lines = append(lines, "\t\t\tbg = "+bg)
		}
		lines = append(lines, "\t\t}")
	}

	ideology := ""
	if pyTruthy(attr(char, "ideology")) {
		ideology = pyStr(attr(char, "ideology"))
	}
	shouldEmitLeader := ideology != "" && roleType == "country_leader"
	if shouldEmitLeader {
		expireVal := ""
		if pyTruthy(attr(char, "expire")) {
			expireVal = pyStr(attr(char, "expire"))
		}
		lines = append(lines, "\t\tcountry_leader = {")
		lines = append(lines, "\t\t\tideology = "+ideology)
		if traitsStr != "" && roleType == "country_leader" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		if expireVal != "" {
			lines = append(lines, "\t\t\texpire = \""+expireVal+"\"")
		}
		lines = append(lines, "\t\t\tid = -1")
		lines = append(lines, "\t\t}")
	}

	rawExtrasAll := getOD(char, "raw_extras")
	if rawExtrasAll == nil || !pyTruthy(attr(char, "raw_extras")) {
		if rawExtrasAll == nil {
			rawExtrasAll = NewOD()
		}
	}
	emitAdvisor := roleType == "advisor" || pyTruthy(attr(char, "has_advisor_role"))
	if emitAdvisor {
		slot := attr(char, "advisor_slot")
		if !pyTruthy(slot) {
			slot = "political_advisor"
		}
		cost := attr(char, "advisor_cost")
		if !pyTruthy(cost) {
			cost = bigFromInt(150)
		}
		advTraitsStr := ""
		if roleType == "advisor" {
			advTraitsStr = traitsStr
			if advTraitsStr == "" && pyTruthy(attr(char, "advisor_traits")) {
				advTraitsStr = joinStrList(getList(char, "advisor_traits"), " ")
			}
		} else if pyTruthy(attr(char, "advisor_traits")) {
			advTraitsStr = joinStrList(getList(char, "advisor_traits"), " ")
		}

		lines = append(lines, "\t\tadvisor = {")
		lines = append(lines, "\t\t\tslot = "+pyStr(slot))
		ideaToken := attr(char, "advisor_idea_token")
		if !pyTruthy(ideaToken) {
			ideaToken = getStr(char, "tag") + "_advisor"
		}
		lines = append(lines, "\t\t\tidea_token = "+pyStr(ideaToken))
		advisorAllowedRaw := rawExtrasAll.GetDef("advisor_allowed", nil)
		if advisorAllowedRaw != nil && pyTruthy(advisorAllowedRaw) {
			lines = append(lines, "\t\t\tallowed = {")
			lines = append(lines, indentBlock(pyStr(advisorAllowedRaw), 4)...)
			lines = append(lines, "\t\t\t}")
		} else {
			lines = append(lines, "\t\t\tallowed = {")
			lines = append(lines, "\t\t\t\toriginal_tag = "+tag)
			lines = append(lines, "\t\t\t}")
		}
		if pyTruthy(attr(char, "advisor_available")) {
			lines = append(lines, "\t\t\tavailable = {")
			lines = append(lines, indentBlock(pyStr(attr(char, "advisor_available")), 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if pyTruthy(attr(char, "advisor_visible")) {
			lines = append(lines, "\t\t\tvisible = {")
			lines = append(lines, indentBlock(pyStr(attr(char, "advisor_visible")), 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if advTraitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+advTraitsStr+" }")
		}
		removalCost := attr(char, "advisor_removal_cost")
		if removalCost != nil && valGEZero(removalCost) {
			lines = append(lines, "\t\t\tremoval_cost = "+pyStr(removalCost))
		}
		if v, ok := attr(char, "always_show_on_actions_tooltip").(bool); ok && v {
			lines = append(lines, "\t\t\talways_show_on_actions_tooltip = yes")
		}
		if pyTruthy(attr(char, "advisor_modifier")) {
			lines = append(lines, "\t\t\tmodifier = {")
			lines = append(lines, indentBlock(pyStr(attr(char, "advisor_modifier")), 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if pyTruthy(attr(char, "advisor_on_add")) {
			lines = append(lines, "\t\t\ton_add = {")
			lines = append(lines, indentBlock(pyStr(attr(char, "advisor_on_add")), 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if pyTruthy(attr(char, "advisor_on_remove")) {
			lines = append(lines, "\t\t\ton_remove = {")
			lines = append(lines, indentBlock(pyStr(attr(char, "advisor_on_remove")), 4)...)
			lines = append(lines, "\t\t\t}")
		}
		for _, key := range rawExtrasAll.Keys() {
			if !strings.HasPrefix(key, "advisor_") || advisorEmittedRawKeys[key] {
				continue
			}
			if key == "advisor_always_show_on_actions_tooltip" &&
				attr(char, "always_show_on_actions_tooltip") != nil {
				continue
			}
			subKey := key[len("advisor_"):]
			value, _ := rawExtrasAll.Get(key)
			emitRawExtras(&lines, subKey, value, "\t\t\t")
		}
		lines = append(lines, "\t\t\tcost = "+pyStr(cost))
		lines = append(lines, "\t\t}")
	}

	for _, key := range rawExtrasAll.Keys() {
		if strings.HasPrefix(key, "advisor_") {
			continue
		}
		if key == "_origin_has_instance" || key == "additional_instances" {
			continue
		}
		if key == "can_be_captured" {
			continue
		}
		value, _ := rawExtrasAll.Get(key)
		emitRawExtras(&lines, key, value, "\t\t")
	}

	lines = append(lines, "\t}")
	lines = append(lines, "}")

	if pyTruthy(rawExtrasAll.GetDef("_origin_has_instance", nil)) {
		charsOpen, charOpen := lines[0], lines[1]
		charClose, charsClose := lines[len(lines)-2], lines[len(lines)-1]
		body := lines[2 : len(lines)-2]
		wrapped := []string{charsOpen, charOpen, "\t\tinstance = {"}
		for _, bl := range body {
			if pyStrip(bl) != "" {
				wrapped = append(wrapped, "\t"+bl)
			} else {
				wrapped = append(wrapped, bl)
			}
		}
		wrapped = append(wrapped, "\t\t}")
		for _, instAny := range asList(rawExtrasAll.GetDef("additional_instances", nil)) {
			wrapped = append(wrapped, "\t\tinstance = {")
			for _, il := range pySplitlines(pyStr(instAny)) {
				if pyStrip(il) != "" {
					wrapped = append(wrapped, "\t\t\t"+pyRStrip(il))
				}
			}
			wrapped = append(wrapped, "\t\t}")
		}
		wrapped = append(wrapped, charClose, charsClose)
		lines = wrapped
	}

	return strings.Join(lines, "\n")
}
