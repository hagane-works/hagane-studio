package emitmisc

import (
	"strings"

	"haganeworks/internal/hoi4/importsvc"
)

func generateCategoryScript(cat *NS, hasIconImage bool) []string {
	var lines []string
	lines = append(lines, cat.Str("category_id")+" = {")
	if Truthy(cat.A("icon")) {
		customSprite := DecisionCategoryIconSpriteName(cat.Str("category_id"))
		isCustomUpload := cat.Str("icon") == customSprite
		if !(isCustomUpload && !hasIconImage) {
			lines = append(lines, "\ticon = "+cat.Str("icon"))
		}
	}
	if Truthy(cat.A("picture")) {
		lines = append(lines, "\tpicture = "+cat.Str("picture"))
	}
	if Truthy(cat.A("priority")) {
		lines = append(lines, "\tpriority = "+PyStr(cat.A("priority")))
	}
	onMapArea := pyStrip(cat.Str("on_map_area"))
	if onMapArea != "" {
		lines = append(lines, "\ton_map_area = {")
		lines = append(lines, indentBlock(onMapArea, 2)...)
		lines = append(lines, "\t}")
	}
	allowed := cat.Str("allowed")
	if allowed != "" && pyStrip(allowed) != "" {
		lines = append(lines, "\tallowed = {")
		lines = append(lines, indentBlock(allowed, 2)...)
		lines = append(lines, "\t}")
	}
	visible := cat.Str("visible")
	if visible != "" && pyStrip(visible) != "" {
		lines = append(lines, "\tvisible = {")
		lines = append(lines, indentBlock(visible, 2)...)
		lines = append(lines, "\t}")
	} else if pyLower(cat.StrOr("scope_kind", "own")) == "own" {
		gate, ok := importsvc.SynthCountryGate(
			cat.Str("scope_kind"),
			cat.Str("country_tag"),
			strList(cat.List("shared_country_tags")),
		)
		if ok && gate != "" {
			lines = append(lines, "\tvisible = { "+gate+" }")
		}
	}
	rawExtras := cat.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		EmitRawExtrasBlockOrScalar(&lines, key, value, "\t")
	}
	lines = append(lines, "}")
	return lines
}

func strList(l []any) []string {
	out := make([]string, 0, len(l))
	for _, v := range l {
		out = append(out, PyStr(v))
	}
	return out
}

func fmtDecisionScalar(value any) string {
	switch t := value.(type) {
	case bool:
		return yn(t)
	case float64:
		s := fmtFixedTrim(t, 6)
		if s == "" {
			return "0"
		}
		return s
	}
	return pyStrip(PyStr(value))
}

func emitTargetedModifierBlocks(tmod any, indent string) []string {
	var blocks []*OD
	if lst, ok := tmod.([]any); ok {
		for _, b := range lst {
			if d, isOD := asOD(b); isOD && d.Len() > 0 {
				blocks = append(blocks, d)
			}
		}
	} else if d, ok := asOD(tmod); ok && d.Len() > 0 {
		blocks = []*OD{d}
	} else {
		return nil
	}
	var out []string
	for _, tm := range blocks {
		tagAny, _ := tm.Get("tag")
		type kv struct {
			k string
			v any
		}
		var pairs []kv
		if inner, ok := asOD(tm.GetDef("modifier", nil)); ok {
			for _, k := range inner.Keys() {
				v, _ := inner.Get(k)
				pairs = append(pairs, kv{k, v})
			}
		}
		for _, k := range tm.Keys() {
			if k == "tag" || k == "modifier" {
				continue
			}
			v, _ := tm.Get(k)
			pairs = append(pairs, kv{k, v})
		}
		tagBlank := tagAny == nil || pyStrip(PyStr(tagAny)) == ""
		if tagBlank && len(pairs) == 0 {
			continue
		}
		out = append(out, indent+"targeted_modifier = {")
		if tagAny != nil && pyStrip(PyStr(tagAny)) != "" {
			out = append(out, indent+"\ttag = "+pyStrip(PyStr(tagAny)))
		}
		for _, p := range pairs {
			if s, ok := p.v.(string); ok && (strings.Contains(s, "\n") ||
				strings.Contains(s, "{") || strings.Contains(s, "=")) {
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

func generateSingleDecisionScript(d *NS, hasIconImage bool) []string {
	var lines []string
	lines = append(lines, "\t"+d.Str("decision_id")+" = {")
	if Truthy(d.A("icon")) {
		customSprite := DecisionIconSpriteName(d.Str("decision_id"))
		isCustomUpload := d.Str("icon") == customSprite
		if !(isCustomUpload && !hasIconImage) {
			lines = append(lines, "\t\ticon = "+d.Str("icon"))
		}
	}
	allowedText := d.Str("allowed")
	if pyStrip(allowedText) != "" {
		lines = append(lines, "\t\tallowed = {")
		lines = append(lines, indentBlock(allowedText, 3)...)
		lines = append(lines, "\t\t}")
	} else if d.StrOr("source", "created") == "created" {
		gate, ok := importsvc.SynthCountryGate(
			d.Str("scope_kind"),
			d.Str("country_tag"),
			strList(d.List("shared_country_tags")),
		)
		if ok && gate != "" {
			lines = append(lines, "\t\tallowed = { "+gate+" }")
		}
	}
	for _, f := range []string{"available", "visible"} {
		text := d.Str(f)
		if text != "" && pyStrip(text) != "" {
			lines = append(lines, "\t\t"+f+" = {")
			lines = append(lines, indentBlock(text, 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	if Truthy(d.A("cost")) {
		lines = append(lines, "\t\tcost = "+PyStr(d.A("cost")))
	}
	if Truthy(d.A("fire_only_once")) {
		lines = append(lines, "\t\tfire_only_once = yes")
	}
	if d.Bool("cancel_if_not_visible", false) {
		lines = append(lines, "\t\tcancel_if_not_visible = yes")
	}
	if Truthy(d.A("days_remove")) {
		lines = append(lines, "\t\tdays_remove = "+PyStr(d.A("days_remove")))
	}
	if Truthy(d.A("days_re_enable")) {
		lines = append(lines, "\t\tdays_re_enable = "+PyStr(d.A("days_re_enable")))
	}
	if Truthy(d.Attr("days_mission_timeout", int64(0))) {
		lines = append(lines, "\t\tdays_mission_timeout = "+PyStr(d.A("days_mission_timeout")))
	}
	if d.Bool("selectable_mission", false) {
		lines = append(lines, "\t\tselectable_mission = yes")
	}
	if d.Bool("is_good", false) {
		lines = append(lines, "\t\tis_good = yes")
	}
	modifier := d.Str("modifier")
	if modifier != "" && pyStrip(modifier) != "" {
		lines = append(lines, "\t\tmodifier = {")
		lines = append(lines, indentBlock(modifier, 3)...)
		lines = append(lines, "\t\t}")
	}
	activationText := d.Str("activation")
	if activationText != "" && pyStrip(activationText) != "" {
		lines = append(lines, "\t\tactivation = {")
		lines = append(lines, indentBlock(activationText, 3)...)
		lines = append(lines, "\t\t}")
	}
	timeoutText := d.Str("timeout_effect")
	if timeoutText != "" && pyStrip(timeoutText) != "" {
		lines = append(lines, "\t\ttimeout_effect = {")
		lines = append(lines, indentBlock(timeoutText, 3)...)
		lines = append(lines, "\t\t}")
	}
	etLocs := d.Dict("effect_tooltip_locs")
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
	effectTooltipKey := d.Str("decision_id") + "_et_tt"
	for _, f := range []string{"complete_effect", "remove_effect", "cancel_trigger", "cancel_effect"} {
		text := d.Str(f)
		if f == "complete_effect" {
			if (text != "" && pyStrip(text) != "") || hasEffectTooltip {
				lines = append(lines, "\t\t"+f+" = {")
				if text != "" && pyStrip(text) != "" {
					lines = append(lines, indentBlock(text, 3)...)
				}
				if hasEffectTooltip {
					lines = append(lines, "\t\t\tcustom_effect_tooltip = "+effectTooltipKey)
				}
				lines = append(lines, "\t\t}")
			}
			continue
		}
		if text != "" && pyStrip(text) != "" {
			lines = append(lines, "\t\t"+f+" = {")
			lines = append(lines, indentBlock(text, 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	aiText := d.Str("ai_will_do")
	if aiText != "" && pyStrip(aiText) != "" {
		lines = append(lines, "\t\tai_will_do = {")
		lines = append(lines, indentBlock(aiText, 3)...)
		lines = append(lines, "\t\t}")
	}
	if Truthy(d.Attr("priority", int64(0))) {
		lines = append(lines, "\t\tpriority = "+PyStr(d.A("priority")))
	}
	stateTarget := pyStrip(d.Str("state_target"))
	if stateTarget != "" {
		lines = append(lines, "\t\tstate_target = "+stateTarget)
	}
	targetsAny := d.Attr("targets", nil)
	if targets, ok := targetsAny.([]any); ok {
		var parts []string
		for _, t := range targets {
			if s := pyStrip(PyStr(t)); s != "" {
				parts = append(parts, s)
			}
		}
		toks := strings.Join(parts, " ")
		if toks != "" {
			lines = append(lines, "\t\ttargets = { "+toks+" }")
		}
	} else if s, ok := targetsAny.(string); ok && pyStrip(s) != "" {
		lines = append(lines, "\t\ttargets = { "+pyStrip(s)+" }")
	}
	targetArrayAny := d.Attr("target_array", nil)
	if targetArray, ok := targetArrayAny.([]any); ok {
		for _, arr := range targetArray {
			arrS := pyStrip(PyStr(arr))
			if arrS != "" {
				lines = append(lines, "\t\ttarget_array = "+arrS)
			}
		}
	} else if s, ok := targetArrayAny.(string); ok && pyStrip(s) != "" {
		lines = append(lines, "\t\ttarget_array = "+pyStrip(s))
	}
	for _, f := range []string{"target_root_trigger", "target_trigger"} {
		text := d.Str(f)
		if text != "" && pyStrip(text) != "" {
			lines = append(lines, "\t\t"+f+" = {")
			lines = append(lines, indentBlock(text, 3)...)
			lines = append(lines, "\t\t}")
		}
	}
	lines = append(lines, emitTargetedModifierBlocks(d.Attr("targeted_modifier", nil), "\t\t")...)
	customCostTrigger := d.Str("custom_cost_trigger")
	if customCostTrigger != "" && pyStrip(customCostTrigger) != "" {
		lines = append(lines, "\t\tcustom_cost_trigger = {")
		lines = append(lines, indentBlock(customCostTrigger, 3)...)
		lines = append(lines, "\t\t}")
	}
	customCostText := pyStrip(d.Str("custom_cost_text"))
	if customCostText != "" {
		lines = append(lines, "\t\tcustom_cost_text = "+customCostText)
	}
	onMapMode := pyStrip(d.Str("on_map_mode"))
	if onMapMode != "" {
		lines = append(lines, "\t\ton_map_mode = "+onMapMode)
	}
	if highlightStates, ok := d.Attr("highlight_states", nil).([]any); ok {
		var hsIDs []string
		for _, s := range highlightStates {
			if v := pyStrip(PyStr(s)); v != "" {
				hsIDs = append(hsIDs, v)
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
	for _, f := range []string{"war_with_on_remove"} {
		val := pyStrip(d.Str(f))
		if val != "" {
			lines = append(lines, "\t\t"+f+" = "+val)
		}
	}
	rawExtras := d.Dict("raw_extras")
	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		EmitRawExtrasBlockOrScalar(&lines, key, value, "\t\t")
	}
	lines = append(lines, "\t}")
	return lines
}

func GenerateDecisionCategoriesScript(categories []*NS, iconGates map[int64]bool) string {
	var lines []string
	for _, cat := range categories {
		hasIcon := true
		if iconGates != nil {
			g, ok := iconGates[cat.Int("id", 0)]
			if ok {
				hasIcon = g
			}
		}
		lines = append(lines, generateCategoryScript(cat, hasIcon)...)
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func GenerateDecisionsPreview(categories, decisions []*NS, iconGates map[int64]bool) string {
	var lines []string
	catMap := map[int64][]*NS{}
	for _, d := range decisions {
		cid := d.Int("category_id", 0)
		catMap[cid] = append(catMap[cid], d)
	}
	for _, cat := range categories {
		catDecisions := catMap[cat.Int("id", 0)]
		lines = append(lines, cat.Str("category_id")+" = {")
		lines = append(lines, "")
		for _, d := range catDecisions {
			hasIcon := true
			if iconGates != nil {
				g, ok := iconGates[d.Int("id", 0)]
				if ok {
					hasIcon = g
				}
			}
			lines = append(lines, generateSingleDecisionScript(d, hasIcon)...)
			lines = append(lines, "")
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
