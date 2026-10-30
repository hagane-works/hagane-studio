package characters

import (
	"math/big"
	"strings"
)

type ScriptOpts struct {
	HasPortraitImage     bool
	HasAdvisorImage      bool
	PortraitPathOverride string
	AdvisorPathOverride  string
}

func indentBlock(text string, baseTabs int) []string {
	prefix := strings.Repeat("\t", baseTabs)
	var out []string
	for _, rawLine := range pySplitlines(text) {
		stripped := pyStrip(rawLine)
		if stripped == "" {
			continue
		}
		leading := len(rawLine) - len(strings.TrimLeft(rawLine, "\t"))
		out = append(out, prefix+strings.Repeat("\t", leading)+stripped)
	}
	return out
}

func emitVisibleBlock(body string, keywordTabs int) []string {
	text := pyStrip(body)
	if text == "" {
		return nil
	}
	kw := strings.Repeat("\t", keywordTabs)
	out := []string{kw + "visible = {"}
	out = append(out, indentBlock(text, keywordTabs+1)...)
	out = append(out, kw+"}")
	return out
}

var advisorEmittedRawKeys = map[string]bool{
	"advisor_slot": true, "advisor_cost": true, "advisor_idea_token": true,
	"advisor_traits": true, "advisor_available": true, "advisor_visible": true,
	"advisor_removal_cost": true, "advisor_modifier": true,
	"advisor_on_add": true, "advisor_on_remove": true, "advisor_allowed": true,
}

func bigStr(z *big.Int) string {
	if z == nil {
		return "0"
	}
	return z.String()
}

func GenerateCharacterScript(char *Character, countryTag string, opts ScriptOpts) string {
	tag := pyUpper(countryTag)
	slug := extractSlug(char.Tag, tag)
	var lines []string

	lines = append(lines, "characters = {")
	lines = append(lines, "\t"+char.Tag+" = {")
	lines = append(lines, "\t\tname = "+characterLocKey(char))

	if char.Gender != "" {
		lines = append(lines, "\t\tgender = "+char.Gender)
	}

	if char.CanBeCaptured != nil && !*char.CanBeCaptured {
		lines = append(lines, "\t\tcan_be_captured = no")
	}

	hasPortraitImage := opts.HasPortraitImage
	hasAdvisorImage := opts.HasAdvisorImage
	portraitPath := opts.PortraitPathOverride
	if portraitPath == "" {
		portraitPath = exportPortraitPath(tag, slug)
	}
	advisorPath := opts.AdvisorPathOverride
	if advisorPath == "" {
		advisorPath = exportAdvisorIconPath(tag, slug)
	}
	if opts.PortraitPathOverride != "" {
		hasPortraitImage = true
	}
	if opts.AdvisorPathOverride != "" {
		hasAdvisorImage = true
	}

	roleUsesArmyLarge := char.RoleType == "general" || char.RoleType == "field_marshal" || char.RoleType == "scientist"
	roleUsesNavyLarge := char.RoleType == "admiral"

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

	traitsStr := strings.Join(char.Traits, " ")

	switch char.RoleType {
	case "general":
		lines = append(lines, "\t\tcorps_commander = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskill = "+bigStr(char.Skill))
		lines = append(lines, "\t\t\tattack_skill = "+bigStr(char.AttackSkill))
		lines = append(lines, "\t\t\tdefense_skill = "+bigStr(char.DefenseSkill))
		lines = append(lines, "\t\t\tplanning_skill = "+bigStr(char.PlanningSkill))
		lines = append(lines, "\t\t\tlogistics_skill = "+bigStr(char.LogisticsSkill))
		lines = append(lines, "\t\t\tlegacy_id = -1")
		lines = append(lines, emitVisibleBlock(char.MilitaryVisible, 3)...)
		lines = append(lines, "\t\t}")

	case "field_marshal":
		lines = append(lines, "\t\tfield_marshal = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskill = "+bigStr(char.Skill))
		lines = append(lines, "\t\t\tattack_skill = "+bigStr(char.AttackSkill))
		lines = append(lines, "\t\t\tdefense_skill = "+bigStr(char.DefenseSkill))
		lines = append(lines, "\t\t\tplanning_skill = "+bigStr(char.PlanningSkill))
		lines = append(lines, "\t\t\tlogistics_skill = "+bigStr(char.LogisticsSkill))
		lines = append(lines, "\t\t\tlegacy_id = -1")
		lines = append(lines, emitVisibleBlock(char.MilitaryVisible, 3)...)
		lines = append(lines, "\t\t}")

	case "admiral":
		lines = append(lines, "\t\tnavy_leader = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskill = "+bigStr(char.Skill))
		lines = append(lines, "\t\t\tattack_skill = "+bigStr(char.AttackSkill))
		lines = append(lines, "\t\t\tdefense_skill = "+bigStr(char.DefenseSkill))
		lines = append(lines, "\t\t\tmaneuvering_skill = "+bigStr(char.ManeuveringSkill))
		lines = append(lines, "\t\t\tcoordination_skill = "+bigStr(char.CoordinationSkill))
		lines = append(lines, "\t\t\tlegacy_id = -1")
		lines = append(lines, emitVisibleBlock(char.MilitaryVisible, 3)...)
		lines = append(lines, "\t\t}")

	case "scientist":
		lines = append(lines, "\t\tscientist = {")
		if traitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		lines = append(lines, "\t\t\tskills = {")
		if len(char.ScientistSpecs) > 0 {
			for _, entry := range char.ScientistSpecs {
				spec := entry.Spec
				if spec == "" {
					spec = "specialization_land"
				}
				level := "2"
				if entry.Level != nil {
					level = entry.Level.String()
				}
				lines = append(lines, "\t\t\t\t"+spec+" = "+level)
			}
		} else {
			lines = append(lines, "\t\t\t\tspecialization_land = 2")
		}
		lines = append(lines, "\t\t\t}")
		lines = append(lines, emitVisibleBlock(char.ScientistVisible, 3)...)
		lines = append(lines, "\t\t}")

	case "operative":
		lines = append(lines, "\t\toperative = {")
		skillLevel := char.OperativeSkillLevel
		if skillLevel == nil || skillLevel.Sign() == 0 {
			skillLevel = big.NewInt(1)
		}
		if skillLevel.Cmp(big.NewInt(1)) != 0 {
			lines = append(lines, "\t\t\tskill_level = "+skillLevel.String())
		}
		if len(char.OperativeTraits) > 0 {
			lines = append(lines, "\t\t\ttraits = { "+strings.Join(char.OperativeTraits, " ")+" }")
		}
		if len(char.OperativeNationalities) > 0 {
			lines = append(lines, "\t\t\tnationalities = { "+strings.Join(char.OperativeNationalities, " ")+" }")
		}
		if opAvailable := pyStrip(char.OperativeAvailable); opAvailable != "" {
			lines = append(lines, "\t\t\tavailable = {")
			lines = append(lines, indentBlock(opAvailable, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if bg := pyStrip(char.OperativeBg); bg != "" {
			lines = append(lines, "\t\t\tbg = "+bg)
		}
		lines = append(lines, "\t\t}")
	}

	ideology := char.Ideology
	shouldEmitLeader := ideology != "" && char.RoleType == "country_leader"
	if shouldEmitLeader {
		expireVal := char.Expire
		lines = append(lines, "\t\tcountry_leader = {")
		lines = append(lines, "\t\t\tideology = "+ideology)
		if traitsStr != "" && char.RoleType == "country_leader" {
			lines = append(lines, "\t\t\ttraits = { "+traitsStr+" }")
		}
		if expireVal != "" {
			lines = append(lines, "\t\t\texpire = \""+expireVal+"\"")
		}
		lines = append(lines, "\t\t\tid = -1")
		lines = append(lines, "\t\t}")
	}

	emitAdvisor := char.RoleType == "advisor" || char.HasAdvisorRole
	if emitAdvisor {
		slot := char.AdvisorSlot
		if slot == "" {
			slot = "political_advisor"
		}
		cost := char.AdvisorCost
		if cost == nil || cost.Sign() == 0 {
			cost = big.NewInt(150)
		}
		var advTraitsStr string
		if char.RoleType == "advisor" {
			advTraitsStr = traitsStr
			if advTraitsStr == "" {
				advTraitsStr = strings.Join(char.AdvisorTraits, " ")
			}
		} else {
			advTraitsStr = strings.Join(char.AdvisorTraits, " ")
		}

		lines = append(lines, "\t\tadvisor = {")
		lines = append(lines, "\t\t\tslot = "+slot)
		ideaToken := char.AdvisorIdeaToken
		if ideaToken == "" {
			ideaToken = char.Tag + "_advisor"
		}
		lines = append(lines, "\t\t\tidea_token = "+ideaToken)
		advisorAllowedRaw, _ := kvLookup(char.RawExtras, "advisor_allowed")
		allowedStr, _ := advisorAllowedRaw.(string)
		if pyTruthy(advisorAllowedRaw) && allowedStr != "" {
			lines = append(lines, "\t\t\tallowed = {")
			lines = append(lines, indentBlock(allowedStr, 4)...)
			lines = append(lines, "\t\t\t}")
		} else {
			lines = append(lines, "\t\t\tallowed = {")
			lines = append(lines, "\t\t\t\toriginal_tag = "+tag)
			lines = append(lines, "\t\t\t}")
		}
		if char.AdvisorAvailable != "" {
			lines = append(lines, "\t\t\tavailable = {")
			lines = append(lines, indentBlock(char.AdvisorAvailable, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if char.AdvisorVisible != "" {
			lines = append(lines, "\t\t\tvisible = {")
			lines = append(lines, indentBlock(char.AdvisorVisible, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if advTraitsStr != "" {
			lines = append(lines, "\t\t\ttraits = { "+advTraitsStr+" }")
		}
		if char.AdvisorRemovalCost != nil && char.AdvisorRemovalCost.Sign() >= 0 {
			lines = append(lines, "\t\t\tremoval_cost = "+char.AdvisorRemovalCost.String())
		}
		if char.AlwaysShowOnActionsTooltip != nil && *char.AlwaysShowOnActionsTooltip {
			lines = append(lines, "\t\t\talways_show_on_actions_tooltip = yes")
		}
		if char.AdvisorModifier != "" {
			lines = append(lines, "\t\t\tmodifier = {")
			lines = append(lines, indentBlock(char.AdvisorModifier, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if char.AdvisorOnAdd != "" {
			lines = append(lines, "\t\t\ton_add = {")
			lines = append(lines, indentBlock(char.AdvisorOnAdd, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		if char.AdvisorOnRemove != "" {
			lines = append(lines, "\t\t\ton_remove = {")
			lines = append(lines, indentBlock(char.AdvisorOnRemove, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		for _, kv := range char.RawExtras {
			if !strings.HasPrefix(kv.Key, "advisor_") || advisorEmittedRawKeys[kv.Key] {
				continue
			}
			if kv.Key == "advisor_always_show_on_actions_tooltip" && char.AlwaysShowOnActionsTooltip != nil {
				continue
			}
			subKey := kv.Key[len("advisor_"):]
			emitRawExtrasBlockOrScalar(&lines, subKey, kv.Val, "\t\t\t")
		}
		lines = append(lines, "\t\t\tcost = "+cost.String())
		lines = append(lines, "\t\t}")
	}

	for _, kv := range char.RawExtras {
		if strings.HasPrefix(kv.Key, "advisor_") {
			continue
		}
		if kv.Key == "_origin_has_instance" || kv.Key == "additional_instances" {
			continue
		}
		if kv.Key == "can_be_captured" {
			continue
		}
		emitRawExtrasBlockOrScalar(&lines, kv.Key, kv.Val, "\t\t")
	}

	lines = append(lines, "\t}")
	lines = append(lines, "}")

	if origin, ok := kvLookup(char.RawExtras, "_origin_has_instance"); ok && pyTruthy(origin) {
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
		if extra, ok := kvLookup(char.RawExtras, "additional_instances"); ok && pyTruthy(extra) {
			if lst, isList := extra.([]any); isList {
				for _, instText := range lst {
					wrapped = append(wrapped, "\t\tinstance = {")
					for _, il := range pySplitlines(rawExtraValueStr(instText)) {
						if pyStrip(il) != "" {
							wrapped = append(wrapped, "\t\t\t"+pyRstrip(il))
						}
					}
					wrapped = append(wrapped, "\t\t}")
				}
			}
		}
		wrapped = append(wrapped, charClose, charsClose)
		lines = wrapped
	}

	return strings.Join(lines, "\n")
}
