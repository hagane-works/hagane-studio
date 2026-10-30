package preview

import "strings"

type DecisionCategory struct {
	Ent *OD
	ID  int
}

type Decision struct {
	Ent        *OD
	ID         int
	CategoryID int
}

func generateCategoryScriptLines(cat *OD, hasIconImage bool) []string {
	var lines []string
	lines = append(lines, getStr(cat, "category_id")+" = {")
	if pyTruthy(attr(cat, "icon")) {
		customSprite := DecisionCategoryIconSpriteName(getStr(cat, "category_id"))
		isCustomUpload := getStr(cat, "icon") == customSprite
		if !(isCustomUpload && !hasIconImage) {
			lines = append(lines, "\ticon = "+pyStr(attr(cat, "icon")))
		}
	}
	if pyTruthy(attr(cat, "picture")) {
		lines = append(lines, "\tpicture = "+pyStr(attr(cat, "picture")))
	}
	if pyTruthy(attr(cat, "priority")) {
		lines = append(lines, "\tpriority = "+pyStr(attr(cat, "priority")))
	}
	onMapArea := pyStrip(orEmptyStr(attr(cat, "on_map_area")))
	if onMapArea != "" {
		lines = append(lines, "\ton_map_area = {")
		lines = append(lines, indentBlock(onMapArea, 2)...)
		lines = append(lines, "\t}")
	}
	if pyTruthy(attr(cat, "allowed")) && pyStrip(getStr(cat, "allowed")) != "" {
		lines = append(lines, "\tallowed = {")
		lines = append(lines, indentBlock(getStr(cat, "allowed"), 2)...)
		lines = append(lines, "\t}")
	}
	if pyTruthy(attr(cat, "visible")) && pyStrip(getStr(cat, "visible")) != "" {
		lines = append(lines, "\tvisible = {")
		lines = append(lines, indentBlock(getStr(cat, "visible"), 2)...)
		lines = append(lines, "\t}")
	} else {
		sk := orEmptyStr(attr(cat, "scope_kind"))
		if sk == "" {
			sk = "own"
		}
		if pyLower(sk) == "own" {
			gate, ok := synthCountryGate(
				orEmptyStr(attr(cat, "scope_kind")),
				orEmptyStr(attr(cat, "country_tag")),
				asList(attr(cat, "shared_country_tags")))
			if ok {
				lines = append(lines, "\tvisible = { "+gate+" }")
			}
		}
	}
	if rawExtras := getOD(cat, "raw_extras"); rawExtras != nil {
		for _, key := range rawExtras.Keys() {
			value, _ := rawExtras.Get(key)
			emitRawExtras(&lines, key, value, "\t")
		}
	}
	lines = append(lines, "}")
	return lines
}

func emitTargetedModifierBlocks(tmod any, indent string) []string {
	var blocks []*OD
	if lst, ok := tmod.([]any); ok {
		for _, b := range lst {
			if od := asOD(b); od != nil && od.Len() > 0 {
				blocks = append(blocks, od)
			}
		}
	} else if od := asOD(tmod); od != nil && od.Len() > 0 {
		blocks = []*OD{od}
	} else {
		return nil
	}
	var out []string
	for _, tm := range blocks {
		tagAny := tm.GetDef("tag", nil)
		type pair struct {
			k string
			v any
		}
		var pairs []pair
		if inner := asOD(tm.GetDef("modifier", nil)); inner != nil {
			for _, k := range inner.Keys() {
				v, _ := inner.Get(k)
				pairs = append(pairs, pair{k, v})
			}
		}
		for _, k := range tm.Keys() {
			if k == "tag" || k == "modifier" {
				continue
			}
			v, _ := tm.Get(k)
			pairs = append(pairs, pair{k, v})
		}
		tagEmpty := tagAny == nil || pyStrip(pyStr(tagAny)) == ""
		if tagEmpty && len(pairs) == 0 {
			continue
		}
		out = append(out, indent+"targeted_modifier = {")
		if tagAny != nil && pyStrip(pyStr(tagAny)) != "" {
			out = append(out, indent+"\ttag = "+pyStrip(pyStr(tagAny)))
		}
		for _, p := range pairs {
			if s, ok := p.v.(string); ok && (strings.Contains(s, "\n") || strings.Contains(s, "{") || strings.Contains(s, "=")) {
				out = append(out, indent+"\t"+p.k+" = {")
				out = append(out, indentBlock(s, strings.Count(indent, "\t")+2)...)
				out = append(out, indent+"\t}")
			} else {
				out = append(out, indent+"\t"+p.k+" = "+fmtDecisionScalar(p.v))
			}
		}
		out = append(out, indent+"}")
	}
	return out
}

func generateSingleDecisionScriptLines(d *OD, hasIconImage bool) []string {
	var lines []string
	lines = append(lines, "\t"+getStr(d, "decision_id")+" = {")
	if pyTruthy(attr(d, "icon")) {
		customSprite := DecisionIconSpriteName(getStr(d, "decision_id"))
		isCustomUpload := getStr(d, "icon") == customSprite
		if !(isCustomUpload && !hasIconImage) {
			lines = append(lines, "\t\ticon = "+pyStr(attr(d, "icon")))
		}
	}
	allowedText := orEmptyStr(attr(d, "allowed"))
	if pyStrip(allowedText) != "" {
		lines = append(lines, "\t\tallowed = {")
		lines = append(lines, indentBlock(allowedText, 3)...)
		lines = append(lines, "\t\t}")
	} else {
		source := orEmptyStr(attr(d, "source"))
		if source == "" {
			source = "created"
		}
		if source == "created" {
			gate, ok := synthCountryGate(
				orEmptyStr(attr(d, "scope_kind")),
				orEmptyStr(attr(d, "country_tag")),
				asList(attr(d, "shared_country_tags")))
			if ok {
				lines = append(lines, "\t\tallowed = { "+gate+" }")
			}
		}
	}
	for _, field := range []string{"available", "visible"} {
		text := attr(d, field)
		if pyTruthy(text) && pyStrip(pyStr(text)) != "" {
			lines = append(lines, "\t\t"+field+" = {")
			lines = append(lines, indentBlock(pyStr(text), 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	if pyTruthy(attr(d, "cost")) {
		lines = append(lines, "\t\tcost = "+pyStr(attr(d, "cost")))
	}
	if pyTruthy(attr(d, "fire_only_once")) {
		lines = append(lines, "\t\tfire_only_once = yes")
	}
	if pyTruthy(attr(d, "cancel_if_not_visible")) {
		lines = append(lines, "\t\tcancel_if_not_visible = yes")
	}
	if pyTruthy(attr(d, "days_remove")) {
		lines = append(lines, "\t\tdays_remove = "+pyStr(attr(d, "days_remove")))
	}
	if pyTruthy(attr(d, "days_re_enable")) {
		lines = append(lines, "\t\tdays_re_enable = "+pyStr(attr(d, "days_re_enable")))
	}
	if pyTruthy(attr(d, "days_mission_timeout")) {
		lines = append(lines, "\t\tdays_mission_timeout = "+pyStr(attr(d, "days_mission_timeout")))
	}
	if pyTruthy(attr(d, "selectable_mission")) {
		lines = append(lines, "\t\tselectable_mission = yes")
	}
	if pyTruthy(attr(d, "is_good")) {
		lines = append(lines, "\t\tis_good = yes")
	}
	if pyTruthy(attr(d, "modifier")) && pyStrip(getStr(d, "modifier")) != "" {
		lines = append(lines, "\t\tmodifier = {")
		lines = append(lines, indentBlock(getStr(d, "modifier"), 3)...)
		lines = append(lines, "\t\t}")
	}
	activationText := attr(d, "activation")
	if pyTruthy(activationText) && pyStrip(pyStr(activationText)) != "" {
		lines = append(lines, "\t\tactivation = {")
		lines = append(lines, indentBlock(pyStr(activationText), 3)...)
		lines = append(lines, "\t\t}")
	}
	timeoutText := attr(d, "timeout_effect")
	if pyTruthy(timeoutText) && pyStrip(pyStr(timeoutText)) != "" {
		lines = append(lines, "\t\ttimeout_effect = {")
		lines = append(lines, indentBlock(pyStr(timeoutText), 3)...)
		lines = append(lines, "\t\t}")
	}
	etLocs := getOD(d, "effect_tooltip_locs")
	hasEffectTooltip := false
	if etLocs != nil {
		for _, k := range etLocs.Keys() {
			v, _ := etLocs.Get(k)
			if s, ok := v.(string); ok && pyStrip(s) != "" {
				hasEffectTooltip = true
				break
			}
		}
	}
	effectTooltipKey := getStr(d, "decision_id") + "_et_tt"
	for _, field := range []string{"complete_effect", "remove_effect", "cancel_trigger", "cancel_effect"} {
		text := attr(d, field)
		if field == "complete_effect" {
			nonEmpty := pyTruthy(text) && pyStrip(pyStr(text)) != ""
			if nonEmpty || hasEffectTooltip {
				lines = append(lines, "\t\t"+field+" = {")
				if nonEmpty {
					lines = append(lines, indentBlock(pyStr(text), 3)...)
				}
				if hasEffectTooltip {
					lines = append(lines, "\t\t\tcustom_effect_tooltip = "+effectTooltipKey)
				}
				lines = append(lines, "\t\t}")
			}
			continue
		}
		if pyTruthy(text) && pyStrip(pyStr(text)) != "" {
			lines = append(lines, "\t\t"+field+" = {")
			lines = append(lines, indentBlock(pyStr(text), 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	aiText := attr(d, "ai_will_do")
	if pyTruthy(aiText) && pyStrip(pyStr(aiText)) != "" {
		lines = append(lines, "\t\tai_will_do = {")
		lines = append(lines, indentBlock(pyStr(aiText), 3)...)
		lines = append(lines, "\t\t}")
	}
	if pyTruthy(attr(d, "priority")) {
		lines = append(lines, "\t\tpriority = "+pyStr(attr(d, "priority")))
	}
	stateTarget := pyStrip(orEmptyStr(attr(d, "state_target")))
	if stateTarget != "" {
		lines = append(lines, "\t\tstate_target = "+stateTarget)
	}
	targets := attr(d, "targets")
	if targetsList, ok := targets.([]any); ok {
		var parts []string
		for _, t := range targetsList {
			s := pyStrip(pyStr(t))
			if s != "" {
				parts = append(parts, s)
			}
		}
		toks := strings.Join(parts, " ")
		if toks != "" {
			lines = append(lines, "\t\ttargets = { "+toks+" }")
		}
	} else if s, ok := targets.(string); ok && pyStrip(s) != "" {
		lines = append(lines, "\t\ttargets = { "+pyStrip(s)+" }")
	}
	targetArray := attr(d, "target_array")
	if taList, ok := targetArray.([]any); ok {
		for _, arr := range taList {
			arrS := pyStrip(pyStr(arr))
			if arrS != "" {
				lines = append(lines, "\t\ttarget_array = "+arrS)
			}
		}
	} else if s, ok := targetArray.(string); ok && pyStrip(s) != "" {
		lines = append(lines, "\t\ttarget_array = "+pyStrip(s))
	}
	for _, field := range []string{"target_root_trigger", "target_trigger"} {
		text := orEmptyStr(attr(d, field))
		if text != "" && pyStrip(text) != "" {
			lines = append(lines, "\t\t"+field+" = {")
			lines = append(lines, indentBlock(text, 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	lines = append(lines, emitTargetedModifierBlocks(attr(d, "targeted_modifier"), "\t\t")...)
	customCostTrigger := orEmptyStr(attr(d, "custom_cost_trigger"))
	if customCostTrigger != "" && pyStrip(customCostTrigger) != "" {
		lines = append(lines, "\t\tcustom_cost_trigger = {")
		lines = append(lines, indentBlock(customCostTrigger, 3)...)
		lines = append(lines, "\t\t}")
	}
	customCostText := pyStrip(orEmptyStr(attr(d, "custom_cost_text")))
	if customCostText != "" {
		lines = append(lines, "\t\tcustom_cost_text = "+customCostText)
	}
	onMapMode := pyStrip(orEmptyStr(attr(d, "on_map_mode")))
	if onMapMode != "" {
		lines = append(lines, "\t\ton_map_mode = "+onMapMode)
	}
	highlightStates := attr(d, "highlight_states")
	if hsList, ok := highlightStates.([]any); ok {
		var hsIDs []string
		for _, s := range hsList {
			id := pyStrip(pyStr(s))
			if id != "" {
				hsIDs = append(hsIDs, id)
			}
		}
		if len(hsIDs) > 0 {
			lines = append(lines, "\t\thighlight_states = {")
			lines = append(lines, "\t\t\thighlight_state_targets = {")
			for _, sid := range hsIDs {
				lines = append(lines, "\t\t\t\tstate = "+sid)
			}
			lines = append(lines, "\t\t\t}")
			lines = append(lines, "\t\t}")
		}
	}
	for _, field := range []string{"war_with_on_remove"} {
		val := pyStrip(orEmptyStr(attr(d, field)))
		if val != "" {
			lines = append(lines, "\t\t"+field+" = "+val)
		}
	}
	if rawExtras := getOD(d, "raw_extras"); rawExtras != nil {
		for _, key := range rawExtras.Keys() {
			value, _ := rawExtras.Get(key)
			emitRawExtras(&lines, key, value, "\t\t")
		}
	}
	lines = append(lines, "\t}")
	return lines
}

func GenerateDecisionCategoriesScript(categories []*DecisionCategory, iconGates map[int]bool) string {
	var lines []string
	for _, cat := range categories {
		hasIcon := true
		if iconGates != nil {
			if v, ok := iconGates[cat.ID]; ok {
				hasIcon = v
			}
		}
		lines = append(lines, generateCategoryScriptLines(cat.Ent, hasIcon)...)
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func GenerateDecisionsPreview(categories []*DecisionCategory, decisions []*Decision, iconGates map[int]bool) string {
	var lines []string
	catMap := map[int][]*Decision{}
	for _, d := range decisions {
		catMap[d.CategoryID] = append(catMap[d.CategoryID], d)
	}
	for _, cat := range categories {
		catDecisions := catMap[cat.ID]
		lines = append(lines, getStr(cat.Ent, "category_id")+" = {")
		lines = append(lines, "")
		for _, d := range catDecisions {
			hasIcon := true
			if iconGates != nil {
				if v, ok := iconGates[d.ID]; ok {
					hasIcon = v
				}
			}
			lines = append(lines, generateSingleDecisionScriptLines(d.Ent, hasIcon)...)
			lines = append(lines, "")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
