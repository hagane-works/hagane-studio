package preview

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/focusorder"
)

type FocusTree struct {
	Ent   *OD
	Nodes []*OD
}

func indentInitialPositionTrigger(text, prefix string) []string {
	parts := pySplitlines(text)
	out := make([]string, 0, len(parts))
	for _, line := range parts {
		if line == "" {
			out = append(out, strings.TrimRight(prefix, " \t\r\n"))
		} else {
			out = append(out, prefix+line)
		}
	}
	return out
}

func numericLegacyInitialPosition(value string) bool {
	stripped := pyStrip(value)
	if stripped == "" {
		return false
	}
	runes := []rune(stripped)
	if runes[0] == '+' || runes[0] == '-' {
		runes = runes[1:]
	}
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func appendInitialShowPosition(lines []string, value, legacyValue any) []string {
	position := asOD(value)
	if position == nil || position.Len() == 0 {
		legacyFocus := pyStrip(pyStr(legacyValue))
		if legacyFocus == "" || numericLegacyInitialPosition(legacyFocus) {
			return lines
		}
		return append(lines,
			"", "\tinitial_show_position = {",
			"\t\tfocus = "+legacyFocus, "\t}")
	}
	kind := pyStr(position.GetDef("kind", ""))
	body := []string{}
	switch kind {
	case "focus":
		focusID := strings.TrimSpace(pyStr(position.GetDef("focus_id", "")))
		if focusID == "" {
			return lines
		}
		body = append(body, "\t\tfocus = "+focusID)
	case "coordinates":
		body = append(body,
			"\t\tx = "+pyStr(position.GetDef("x", 0)),
			"\t\ty = "+pyStr(position.GetDef("y", 0)),
		)
		for _, raw := range asList(position.GetDef("offsets", []any{})) {
			offset := asOD(raw)
			if offset == nil {
				continue
			}
			body = append(body,
				"\t\toffset = {",
				"\t\t\tx = "+pyStr(offset.GetDef("x", 0)),
				"\t\t\ty = "+pyStr(offset.GetDef("y", 0)),
			)
			trigger := pyStr(offset.GetDef("trigger", ""))
			if strings.TrimSpace(trigger) != "" {
				body = append(body, "\t\t\ttrigger = {")
				body = append(body,
					indentInitialPositionTrigger(trigger, "\t\t\t\t")...)
				body = append(body, "\t\t\t}")
			}
			body = append(body, "\t\t}")
		}
	default:
		return lines
	}
	lines = append(lines, "", "\tinitial_show_position = {")
	lines = append(lines, body...)
	return append(lines, "\t}")
}

func generateFocusNodeScriptLines(node *OD, keyword string, validIcons map[string]bool) []string {
	var lines []string
	lines = append(lines, "\t"+keyword+" = {")
	lines = append(lines, "\t\tid = "+getStr(node, "focus_id"))

	iconEmit := gateIcon(getStr(node, "icon"), validIcons)
	if iconEmit == "" {
		iconEmit = "GFX_goal_unknown"
	}

	dynamicIconEntries := getList(node, "dynamic_icon")
	var dynLines []string
	yesGfx := map[string]bool{}
	for _, entryAny := range dynamicIconEntries {
		entry := asOD(entryAny)
		if entry == nil {
			continue
		}
		iconKey := gateIcon(pyStrip(orEmptyStr(entry.GetDef("icon", nil))), validIcons)
		if iconKey == "" {
			continue
		}
		trigText := pyStrip(orEmptyStr(entry.GetDef("trigger", nil)))
		if trigText == "" {
			if yesGfx[iconKey] {
				continue
			}
			dynLines = append(dynLines, "\t\t\t"+iconKey+" = yes")
			yesGfx[iconKey] = true
		} else {
			dynLines = append(dynLines, "\t\t\t"+iconKey+" = {")
			dynLines = append(dynLines, indentBlock(trigText, 4)...)
			dynLines = append(dynLines, "\t\t\t}")
		}
	}
	if len(dynLines) > 0 {
		if iconEmit != "" && !yesGfx[iconEmit] {
			dynLines = append(dynLines, "\t\t\t"+iconEmit+" = yes")
		}
		lines = append(lines, "\t\ticon = {")
		lines = append(lines, dynLines...)
		lines = append(lines, "\t\t}")
	} else {
		lines = append(lines, "\t\ticon = "+iconEmit)
	}

	textIconVal := gateIcon(pyStrip(orEmptyStr(attr(node, "text_icon"))), validIcons)
	if textIconVal != "" {
		lines = append(lines, "\t\ttext_icon = "+textIconVal)
	}

	lines = append(lines, "\t\tx = "+pyStr(attr(node, "x")))
	lines = append(lines, "\t\ty = "+pyStr(attr(node, "y")))

	if pyTruthy(attr(node, "relative_position_id")) {
		lines = append(lines, "\t\trelative_position_id = "+pyStr(attr(node, "relative_position_id")))
	}

	for _, entryAny := range getList(node, "offset") {
		entry := asOD(entryAny)
		if entry == nil {
			continue
		}
		ex := entry.GetDef("x", bigFromInt(0))
		ey := entry.GetDef("y", bigFromInt(0))
		trig := pyStrip(orEmptyStr(entry.GetDef("trigger", nil)))
		lines = append(lines, "\t\toffset = {")
		lines = append(lines, "\t\t\tx = "+pyStr(ex))
		lines = append(lines, "\t\t\ty = "+pyStr(ey))
		if trig != "" {
			lines = append(lines, "\t\t\ttrigger = {")
			lines = append(lines, indentBlock(trig, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		lines = append(lines, "\t\t}")
	}

	costVal := attr(node, "cost")
	costStr := pyStr(costVal)
	if f, ok := costVal.(float64); ok && f == math.Trunc(f) {
		z, _ := pyIntFromVal(f)
		costStr = z.String()
	}
	lines = append(lines, "\t\tcost = "+costStr)

	for _, groupAny := range getList(node, "prerequisites") {
		group := asList(groupAny)
		if len(group) == 1 {
			lines = append(lines, "\t\tprerequisite = { focus = "+pyStr(group[0])+" }")
		} else if len(group) > 1 {
			parts := make([]string, 0, len(group))
			for _, f := range group {
				parts = append(parts, "focus = "+pyStr(f))
			}
			lines = append(lines, "\t\tprerequisite = { "+strings.Join(parts, " ")+" }")
		}
	}

	me := getList(node, "mutually_exclusive")
	if len(me) > 0 {
		parts := make([]string, 0, len(me))
		for _, f := range me {
			parts = append(parts, "focus = "+pyStr(f))
		}
		lines = append(lines, "\t\tmutually_exclusive = { "+strings.Join(parts, " ")+" }")
	}

	abAst := getOD(node, "allow_branch_ast")
	if abAst != nil && abAst.Len() > 0 && pyTruthy(abAst.GetDef("children", nil)) {
		lines = append(lines, "\t\tallow_branch = {")
		lines = append(lines, serializeConditionAST(abAst, 3)...)
		lines = append(lines, "\t\t}")
	}

	for _, fld := range []string{"available", "bypass", "cancel"} {
		text := attr(node, fld)
		if pyTruthy(text) && pyStrip(pyStr(text)) != "" {
			lines = append(lines, "\t\t"+fld+" = {")
			lines = append(lines, indentBlock(pyStr(text), 3)...)
			lines = append(lines, "\t\t}")
		}
	}

	rawExtras := NewOD()
	if re := getOD(node, "raw_extras"); re != nil {
		rawExtras = re.Copy()
	}
	rawExtras.Del("_origin_keyword")
	rawExtras.Del("allow_branch")
	bypassEffectAny := rawExtras.GetDef("bypass_effect", "")
	rawExtras.Del("bypass_effect")
	if s, ok := bypassEffectAny.(string); ok && pyStrip(s) != "" {
		lines = append(lines, "\t\tbypass_effect = {")
		lines = append(lines, indentBlock(s, 3)...)
		lines = append(lines, "\t\t}")
	}
	rawExtras.Del("cancel_if_invalid")
	rawExtras.Del("continue_if_invalid")
	rawExtras.Del("available_if_capitulated")
	yn := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	civ := attr(node, "cancel_if_invalid")
	civBool := true
	if civ != nil {
		civBool = pyTruthy(civ)
	}
	lines = append(lines, "\t\tcancel_if_invalid = "+yn(civBool))
	lines = append(lines, "\t\tcontinue_if_invalid = "+yn(pyTruthy(attr(node, "continue_if_invalid"))))
	lines = append(lines, "\t\tavailable_if_capitulated = "+yn(pyTruthy(attr(node, "available_if_capitulated"))))

	rawExtras.Del("will_lead_to_war_with")
	for _, tagAny := range getList(node, "will_lead_to_war_with") {
		t := pyStrip(pyStr(tagAny))
		if t != "" {
			lines = append(lines, "\t\twill_lead_to_war_with = "+t)
		}
	}

	filters := getList(node, "search_filters")
	if len(filters) > 0 {
		lines = append(lines, "\t\tsearch_filters = { "+joinPyStrList(filters, " ")+" }")
	}

	if pyTruthy(attr(node, "select_effect")) && pyStrip(getStr(node, "select_effect")) != "" {
		lines = append(lines, "\t\tselect_effect = {")
		lines = append(lines, indentBlock(getStr(node, "select_effect"), 3)...)
		lines = append(lines, "\t\t}")
	}

	aiText := getStr(node, "ai_will_do")
	if !pyTruthy(attr(node, "ai_will_do")) {
		aiText = "factor = 1"
	}
	lines = append(lines, "\t\tai_will_do = {")
	lines = append(lines, indentBlock(aiText, 3)...)
	lines = append(lines, "\t\t}")

	etLocs := getOD(node, "effect_tooltip_locs")
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
	lines = append(lines, "\t\tcompletion_reward = {")
	if pyTruthy(attr(node, "completion_reward")) && pyStrip(getStr(node, "completion_reward")) != "" {
		lines = append(lines, indentBlock(getStr(node, "completion_reward"), 3)...)
	}
	if hasEffectTooltip {
		lines = append(lines, "\t\t\tcustom_effect_tooltip = "+getStr(node, "focus_id")+"_et_tt")
	}
	lines = append(lines, "\t\t}")

	for _, key := range rawExtras.Keys() {
		value, _ := rawExtras.Get(key)
		emitRawExtras(&lines, key, value, "\t\t")
	}

	lines = append(lines, "\t}")
	return lines
}

func GenerateFocusNodeScript(node *OD) string {
	lines := generateFocusNodeScriptLines(node, "focus", nil)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.ReplaceAll(line, "\t", "    "))
	}
	return strings.Join(out, "\n")
}

func sortedNodesByID(nodes []*OD) []*OD {
	out := append([]*OD(nil), nodes...)
	sort.SliceStable(out, func(i, j int) bool {
		return numCmp(attr(out[i], "id"), attr(out[j], "id")) < 0
	})
	return out
}

func generateSharedPoolScript(tree *FocusTree, validIcons map[string]bool) string {
	var lines []string
	nodes := orderFocusDefinitions(tree.Nodes)
	for i, node := range nodes {
		if i > 0 {
			lines = append(lines, "")
		}
		originKw := "shared_focus"
		if re := getOD(node, "raw_extras"); re != nil {
			if v, ok := re.Get("_origin_keyword"); ok {
				originKw = pyStr(v)
			}
		}
		block := generateFocusNodeScriptLines(node, originKw, validIcons)
		for _, line := range block {
			if strings.HasPrefix(line, "\t") {
				lines = append(lines, line[1:])
			} else {
				lines = append(lines, line)
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func generatePrimaryTreeScript(tree *FocusTree, validIcons map[string]bool) string {
	ent := tree.Ent
	tag := pyUpper(orEmptyStr(attr(ent, "country_tag")))
	scopeKind := getStr(ent, "scope_kind")
	if !pyTruthy(attr(ent, "scope_kind")) {
		scopeKind = "own"
	}
	var sharedTags []string
	for _, t := range asList(attr(ent, "shared_country_tags")) {
		sharedTags = append(sharedTags, pyUpper(pyStr(t)))
	}
	var lines []string

	lines = append(lines, "focus_tree = {")
	lines = append(lines, "\tid = "+getStr(ent, "tree_id"))
	lines = append(lines, "")

	if scopeKind == "own" && tag != "" {
		lines = append(lines, "\tcountry = {")
		lines = append(lines, "\t\tfactor = 0")
		lines = append(lines, "")
		lines = append(lines, "\t\tmodifier = {")
		lines = append(lines, "\t\t\tadd = 100")
		lines = append(lines, "\t\t\ttag = "+tag)
		lines = append(lines, "\t\t}")
		lines = append(lines, "\t}")
	} else if scopeKind == "explicit" && len(sharedTags) > 0 {
		lines = append(lines, "\tcountry = {")
		lines = append(lines, "\t\tfactor = 0")
		for _, st := range sharedTags {
			lines = append(lines, "")
			lines = append(lines, "\t\tmodifier = {")
			lines = append(lines, "\t\t\tadd = 100")
			lines = append(lines, "\t\t\ttag = "+st)
			lines = append(lines, "\t\t}")
		}
		lines = append(lines, "\t}")
	} else if scopeKind == "global_except" && len(sharedTags) > 0 {
		lines = append(lines, "\tcountry = {")
		lines = append(lines, "\t\tfactor = 100")
		for _, st := range sharedTags {
			lines = append(lines, "")
			lines = append(lines, "\t\tmodifier = {")
			lines = append(lines, "\t\t\tfactor = 0")
			lines = append(lines, "\t\t\ttag = "+st)
			lines = append(lines, "\t\t}")
		}
		lines = append(lines, "\t}")
	}

	lines = appendInitialShowPosition(
		lines, attr(ent, "initial_show_position"), attr(ent, "initial_show_focus"))

	lines = append(lines, "")
	lines = append(lines, "\tcontinuous_focus_position = { x = "+
		pyStr(attr(ent, "continuous_focus_position_x"))+" y = "+
		pyStr(attr(ent, "continuous_focus_position_y"))+" }")
	lines = append(lines, "")
	if pyTruthy(attr(ent, "is_default")) {
		lines = append(lines, "\tdefault = yes")
	} else {
		lines = append(lines, "\tdefault = no")
	}

	refs := asList(attr(ent, "shared_focus_refs"))
	if len(refs) > 0 {
		lines = append(lines, "")
		for _, refAny := range refs {
			focusID := ""
			if od := asOD(refAny); od != nil {
				focusID = orEmptyStr(od.GetDef("focus_id", ""))
			} else {
				focusID = pyStr(refAny)
			}
			if focusID != "" {
				lines = append(lines, "\tshared_focus = "+focusID)
			}
		}
	}

	for _, node := range orderFocusDefinitions(tree.Nodes) {
		lines = append(lines, "")
		lines = append(lines, generateFocusNodeScriptLines(node, "focus", validIcons)...)
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func orderFocusDefinitions(nodes []*OD) []*OD {
	baseline := sortedNodesByID(nodes)
	input := make([]focusorder.Node, len(baseline))
	for i, node := range baseline {
		input[i] = focusorder.Node{FocusID: getStr(node, "focus_id"), RelativePositionID: getStr(node, "relative_position_id")}
	}
	plan := focusorder.Plan(input)
	out := make([]*OD, len(baseline))
	for i, index := range plan.Order {
		out[i] = baseline[index]
	}
	return out
}

func GenerateFocusTreeScript(tree *FocusTree, validIcons map[string]bool) string {
	if getStr(tree.Ent, "tree_kind") == "shared_pool" {
		return generateSharedPoolScript(tree, validIcons)
	}
	return generatePrimaryTreeScript(tree, validIcons)
}
