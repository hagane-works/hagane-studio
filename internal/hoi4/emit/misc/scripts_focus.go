package emitmisc

import (
	"sort"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/focusorder"
)

func indentBlock(text string, baseTabs int) []string {
	prefix := strings.Repeat("\t", baseTabs)
	var result []string
	for _, rawLine := range strings.Split(text, "\n") {
		stripped := pyStrip(rawLine)
		if stripped == "" {
			continue
		}
		leadingTabs := 0
		for _, r := range rawLine {
			if r == '\t' {
				leadingTabs++
			} else {
				break
			}
		}
		result = append(result, prefix+strings.Repeat("\t", leadingTabs)+stripped)
	}
	return result
}

func indentInitialPositionTrigger(text, prefix string) []string {
	parts := splitInitialPositionLines(text)
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

func splitInitialPositionLines(text string) []string {
	var out []string
	runes := []rune(text)
	start := 0
	for i := 0; i < len(runes); {
		r := runes[i]
		isBreak := r == '\n' || r == '\r' || r == '\v' || r == '\f' ||
			r == 0x1c || r == 0x1d || r == 0x1e || r == 0x85 ||
			r == 0x2028 || r == 0x2029
		if !isBreak {
			i++
			continue
		}
		out = append(out, string(runes[start:i]))
		if r == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			i++
		}
		i++
		start = i
	}
	if start < len(runes) {
		out = append(out, string(runes[start:]))
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
	position, ok := asOD(value)
	if !ok || position.Len() == 0 {
		legacyFocus := pyStrip(PyStr(legacyValue))
		if legacyFocus == "" || numericLegacyInitialPosition(legacyFocus) {
			return lines
		}
		return append(lines,
			"", "\tinitial_show_position = {",
			"\t\tfocus = "+legacyFocus, "\t}")
	}
	kind := PyStr(position.GetDef("kind", ""))
	body := []string{}
	switch kind {
	case "focus":
		focusID := pyStrip(PyStr(position.GetDef("focus_id", "")))
		if focusID == "" {
			return lines
		}
		body = append(body, "\t\tfocus = "+focusID)
	case "coordinates":
		body = append(body,
			"\t\tx = "+PyStr(position.GetDef("x", 0)),
			"\t\ty = "+PyStr(position.GetDef("y", 0)),
		)
		offsets, _ := position.GetDef("offsets", []any{}).([]any)
		for _, raw := range offsets {
			offset, ok := asOD(raw)
			if !ok {
				continue
			}
			body = append(body,
				"\t\toffset = {",
				"\t\t\tx = "+PyStr(offset.GetDef("x", 0)),
				"\t\t\ty = "+PyStr(offset.GetDef("y", 0)),
			)
			trigger := PyStr(offset.GetDef("trigger", ""))
			if pyStrip(trigger) != "" {
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

func serializeConditionAST(ast any, indent int) []string {
	var children []any
	if d, ok := asOD(ast); ok {
		children, _ = d.GetDef("children", []any{}).([]any)
	} else if l, ok := ast.([]any); ok {
		children = l
	}
	tabs := strings.Repeat("\t", indent)
	var out []string
	for _, childAny := range children {
		child, ok := asOD(childAny)
		if !ok {
			continue
		}
		op := PyStr(child.GetDef("op", ""))
		if op == "" {
			continue
		}
		if PyStr(child.GetDef("kind", nil)) == "block" && hasKind(child) {
			out = append(out, tabs+op+" = {")
			out = append(out, serializeConditionAST(child.GetDef("children", []any{}), indent+1)...)
			out = append(out, tabs+"}")
		} else {
			valStr := PyStr(child.GetDef("value", ""))
			if valStr == "" || strings.Contains(valStr, " ") ||
				strings.Contains(valStr, "/") || strings.Contains(valStr, "\\") {
				valStr = `"` + valStr + `"`
			}
			out = append(out, tabs+op+" = "+valStr)
		}
	}
	return out
}

func hasKind(*OD) bool { return true }

func gateIcon(name string, validIcons map[string]bool) string {
	if name == "" {
		return name
	}
	if validIcons == nil {
		return name
	}
	if validIcons[name] {
		return name
	}
	return ""
}

func yn(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func generateFocusNodeScript(node *NS, keyword string, validIcons map[string]bool) []string {
	var lines []string
	lines = append(lines, "\t"+keyword+" = {")
	lines = append(lines, "\t\tid = "+node.Str("focus_id"))

	iconEmit := gateIcon(nodeIconStr(node), validIcons)
	if iconEmit == "" {
		iconEmit = "GFX_goal_unknown"
	}

	dynamicIconEntries := node.List("dynamic_icon")
	var dynLines []string
	yesGfx := map[string]bool{}
	for _, entryAny := range dynamicIconEntries {
		entry, ok := asOD(entryAny)
		if !ok {
			continue
		}
		iconKey := gateIcon(pyStrip(PyStr(entry.GetDef("icon", ""))), validIcons)
		if iconKey == "" {
			continue
		}
		trigText := pyStrip(PyStr(entry.GetDef("trigger", "")))
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

	textIconVal := gateIcon(pyStrip(node.Str("text_icon")), validIcons)
	if textIconVal != "" {
		lines = append(lines, "\t\ttext_icon = "+textIconVal)
	}

	lines = append(lines, "\t\tx = "+PyStr(node.A("x")))
	lines = append(lines, "\t\ty = "+PyStr(node.A("y")))

	if Truthy(node.A("relative_position_id")) {
		lines = append(lines, "\t\trelative_position_id = "+node.Str("relative_position_id"))
	}

	for _, entryAny := range node.List("offset") {
		entry, ok := asOD(entryAny)
		if !ok {
			continue
		}
		ex := entry.GetDef("x", int64(0))
		ey := entry.GetDef("y", int64(0))
		trig := pyStrip(PyStr(entry.GetDef("trigger", "")))
		lines = append(lines, "\t\toffset = {")
		lines = append(lines, "\t\t\tx = "+PyStr(ex))
		lines = append(lines, "\t\t\ty = "+PyStr(ey))
		if trig != "" {
			lines = append(lines, "\t\t\ttrigger = {")
			lines = append(lines, indentBlock(trig, 4)...)
			lines = append(lines, "\t\t\t}")
		}
		lines = append(lines, "\t\t}")
	}

	costVal := ""
	switch c := node.A("cost").(type) {
	case float64:
		if isIntegralFloat(c) {
			costVal = PyStr(pyIntTrunc(c))
		} else {
			costVal = pyFloatRepr(c)
		}
	default:
		costVal = PyStr(node.A("cost"))
	}
	lines = append(lines, "\t\tcost = "+costVal)

	for _, groupAny := range node.List("prerequisites") {
		group, _ := groupAny.([]any)
		if len(group) == 1 {
			lines = append(lines, "\t\tprerequisite = { focus = "+PyStr(group[0])+" }")
		} else if len(group) > 1 {
			parts := make([]string, 0, len(group))
			for _, f := range group {
				parts = append(parts, "focus = "+PyStr(f))
			}
			lines = append(lines, "\t\tprerequisite = { "+strings.Join(parts, " ")+" }")
		}
	}

	me := node.List("mutually_exclusive")
	if len(me) > 0 {
		parts := make([]string, 0, len(me))
		for _, f := range me {
			parts = append(parts, "focus = "+PyStr(f))
		}
		lines = append(lines, "\t\tmutually_exclusive = { "+strings.Join(parts, " ")+" }")
	}

	abAst := node.Dict("allow_branch_ast")
	if abAst.Len() > 0 {
		if children, ok := abAst.GetDef("children", nil).([]any); ok && len(children) > 0 {
			lines = append(lines, "\t\tallow_branch = {")
			lines = append(lines, serializeConditionAST(abAst, 3)...)
			lines = append(lines, "\t\t}")
		}
	}

	for _, f := range []string{"available", "bypass", "cancel"} {
		text := node.Str(f)
		if text != "" && pyStrip(text) != "" {
			lines = append(lines, "\t\t"+f+" = {")
			lines = append(lines, indentBlock(text, 3)...)
			lines = append(lines, "\t\t}")
		}
	}

	rawExtras := node.Dict("raw_extras").Copy()
	rawExtras.Del("_origin_keyword")
	rawExtras.Del("allow_branch")
	bypassEffectAny := rawExtras.Del("bypass_effect")
	if s, ok := bypassEffectAny.(string); ok && pyStrip(s) != "" {
		lines = append(lines, "\t\tbypass_effect = {")
		lines = append(lines, indentBlock(s, 3)...)
		lines = append(lines, "\t\t}")
	}
	rawExtras.Del("cancel_if_invalid")
	rawExtras.Del("continue_if_invalid")
	rawExtras.Del("available_if_capitulated")
	cancelIfInvalid := node.Attr("cancel_if_invalid", true)
	civ := true
	if cancelIfInvalid != nil {
		civ = Truthy(cancelIfInvalid)
	}
	lines = append(lines, "\t\tcancel_if_invalid = "+yn(civ))
	lines = append(lines, "\t\tcontinue_if_invalid = "+yn(node.Bool("continue_if_invalid", false)))
	lines = append(lines, "\t\tavailable_if_capitulated = "+yn(node.Bool("available_if_capitulated", false)))

	rawExtras.Del("will_lead_to_war_with")
	for _, tagAny := range node.List("will_lead_to_war_with") {
		tag := pyStrip(PyStr(tagAny))
		if tag != "" {
			lines = append(lines, "\t\twill_lead_to_war_with = "+tag)
		}
	}

	filters := node.List("search_filters")
	if len(filters) > 0 {
		parts := make([]string, 0, len(filters))
		for _, f := range filters {
			parts = append(parts, PyStr(f))
		}
		lines = append(lines, "\t\tsearch_filters = { "+strings.Join(parts, " ")+" }")
	}

	if se := node.Str("select_effect"); se != "" && pyStrip(se) != "" {
		lines = append(lines, "\t\tselect_effect = {")
		lines = append(lines, indentBlock(se, 3)...)
		lines = append(lines, "\t\t}")
	}

	aiText := node.StrOr("ai_will_do", "factor = 1")
	lines = append(lines, "\t\tai_will_do = {")
	lines = append(lines, indentBlock(aiText, 3)...)
	lines = append(lines, "\t\t}")

	etLocs := node.Dict("effect_tooltip_locs")
	hasEffectTooltip := false
	if etLocs != nil {
		for _, k := range etLocs.Keys() {
			if v, _ := etLocs.Get(k); v != nil {
				if s, ok := v.(string); ok && pyStrip(s) != "" {
					hasEffectTooltip = true
					break
				}
			}
		}
	}
	lines = append(lines, "\t\tcompletion_reward = {")
	if cr := node.Str("completion_reward"); cr != "" && pyStrip(cr) != "" {
		lines = append(lines, indentBlock(cr, 3)...)
	}
	if hasEffectTooltip {
		lines = append(lines, "\t\t\tcustom_effect_tooltip = "+node.Str("focus_id")+"_et_tt")
	}
	lines = append(lines, "\t\t}")

	for _, key := range rawExtras.Keys() {
		v, _ := rawExtras.Get(key)
		EmitRawExtrasBlockOrScalar(&lines, key, v, "\t\t")
	}

	lines = append(lines, "\t}")
	return lines
}

func nodeIconStr(node *NS) string { return node.Str("icon") }

func sortNodesByID(nodes []*NS) []*NS {
	out := make([]*NS, len(nodes))
	copy(out, nodes)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Int("id", 0) < out[j].Int("id", 0)
	})
	return out
}

func nsList(l []any) []*NS {
	out := make([]*NS, 0, len(l))
	for _, v := range l {
		if n, ok := asNS(v); ok {
			out = append(out, n)
		}
	}
	return out
}

func generateSharedPoolScript(tree *NS, validIcons map[string]bool) string {
	var lines []string
	nodes := orderFocusDefinitions(nsList(tree.List("nodes")))
	for i, node := range nodes {
		if i > 0 {
			lines = append(lines, "")
		}
		originKw := "shared_focus"
		if re := node.Dict("raw_extras"); re != nil {
			originKw = PyStr(re.GetDef("_origin_keyword", "shared_focus"))
		}
		block := generateFocusNodeScript(node, originKw, validIcons)
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

type FocusTreeSelection struct {
	Tag string

	EmitBlock bool

	Note exportmsg.Message
}

func PlanFocusTreeSelection(scopeKind, countryTag, fallbackTag string, sharedTags []string) FocusTreeSelection {
	tag := pyUpper(pyStrip(countryTag))
	fallback := pyUpper(pyStrip(fallbackTag))
	switch scopeKind {
	case "own", "":
		if tag != "" {
			return FocusTreeSelection{Tag: tag, EmitBlock: true}
		}
		if fallback != "" {
			return FocusTreeSelection{
				Tag: fallback, EmitBlock: true,
				Note: exportmsg.Message{Key: "focus.tree_assigned", Args: exportmsg.Args{"tag": fallback}},
			}
		}
		return FocusTreeSelection{Note: exportmsg.Message{Key: "focus.tree_no_country"}}
	case "explicit", "global_except":
		if len(sharedTags) > 0 {
			return FocusTreeSelection{EmitBlock: true}
		}
		return FocusTreeSelection{Note: exportmsg.Message{Key: "focus.tree_empty_list"}}
	case "global":

		return FocusTreeSelection{EmitBlock: true}
	case "shared_pool":

		return FocusTreeSelection{}
	default:
		return FocusTreeSelection{Note: exportmsg.Message{Key: "focus.tree_unknown_scope", Args: exportmsg.Args{"scope": scopeKind}}}
	}
}

func generatePrimaryTreeScript(tree *NS, validIcons map[string]bool) string {
	return generatePrimaryTreeScriptWithFallback(tree, validIcons, "")
}

func generatePrimaryTreeScriptWithFallback(tree *NS, validIcons map[string]bool, fallbackTag string) string {
	scopeKind := tree.StrOr("scope_kind", "own")
	plan := PlanFocusTreeSelection(
		scopeKind, tree.Str("country_tag"), fallbackTag,
		pyUpperList(tree.List("shared_country_tags")),
	)
	tag := plan.Tag
	var sharedTags []string
	for _, t := range tree.List("shared_country_tags") {
		sharedTags = append(sharedTags, pyUpper(PyStr(t)))
	}
	var lines []string

	lines = append(lines, "focus_tree = {")
	lines = append(lines, "\tid = "+tree.Str("tree_id"))
	lines = append(lines, "")

	if plan.EmitBlock && (scopeKind == "own" || scopeKind == "") && tag != "" {
		lines = append(lines, "\tcountry = {")
		lines = append(lines, "\t\tfactor = 0")
		lines = append(lines, "")
		lines = append(lines, "\t\tmodifier = {")
		lines = append(lines, "\t\t\tadd = 100")
		lines = append(lines, "\t\t\ttag = "+tag)
		lines = append(lines, "\t\t}")
		lines = append(lines, "\t}")
	} else if scopeKind == "global" && plan.EmitBlock {

		lines = append(lines, "\tcountry = {")
		lines = append(lines, "\t\tfactor = 100")
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
		lines, tree.A("initial_show_position"), tree.A("initial_show_focus"))

	lines = append(lines, "")
	lines = append(lines, "\tcontinuous_focus_position = { x = "+PyStr(tree.A("continuous_focus_position_x"))+
		" y = "+PyStr(tree.A("continuous_focus_position_y"))+" }")
	lines = append(lines, "")
	def := "no"
	if tree.Bool("is_default", false) {
		def = "yes"
	}
	lines = append(lines, "\tdefault = "+def)

	refs := tree.List("shared_focus_refs")
	if len(refs) > 0 {
		lines = append(lines, "")
		for _, refAny := range refs {
			focusID := ""
			if ref, ok := asOD(refAny); ok {
				focusID = PyStr(ref.GetDef("focus_id", ""))
			} else {
				focusID = PyStr(refAny)
			}
			if focusID != "" {
				lines = append(lines, "\tshared_focus = "+focusID)
			}
		}
	}

	nodes := orderFocusDefinitions(nsList(tree.List("nodes")))
	for _, node := range nodes {
		lines = append(lines, "")
		lines = append(lines, generateFocusNodeScript(node, "focus", validIcons)...)
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func orderFocusDefinitions(nodes []*NS) []*NS {
	baseline := sortNodesByID(nodes)
	input := make([]focusorder.Node, len(baseline))
	for i, node := range baseline {
		input[i] = focusorder.Node{FocusID: node.Str("focus_id"), RelativePositionID: node.Str("relative_position_id")}
	}
	plan := focusorder.Plan(input)
	out := make([]*NS, len(baseline))
	for i, index := range plan.Order {
		out[i] = baseline[index]
	}
	return out
}

func GenerateFocusTreeScript(tree *NS, validIcons map[string]bool) string {
	return GenerateFocusTreeScriptWithFallback(tree, validIcons, "")
}

func GenerateFocusTreeScriptWithFallback(tree *NS, validIcons map[string]bool, fallbackTag string) string {

	if tree.Str("tree_kind") == "shared_pool" {
		return generateSharedPoolScript(tree, validIcons)
	}
	return generatePrimaryTreeScriptWithFallback(tree, validIcons, fallbackTag)
}

func pyUpperList(items []any) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := pyUpper(pyStrip(PyStr(item))); s != "" {
			out = append(out, s)
		}
	}
	return out
}
