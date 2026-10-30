package preview

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/importsvc"
)

func bigFromInt(n int) *big.Int { return big.NewInt(int64(n)) }

func indentBlock(text string, baseTabs int) []string {
	prefix := strings.Repeat("\t", baseTabs)
	var result []string
	for _, rawLine := range pySplitlines(text) {
		stripped := pyStrip(rawLine)
		if stripped == "" {
			continue
		}
		leadingTabs := len(rawLine) - len(pyLStripTabs(rawLine))
		result = append(result, prefix+strings.Repeat("\t", leadingTabs)+stripped)
	}
	return result
}

func fmtHoi4Float(value float64) string {
	if value != value || math.IsInf(value, 0) {
		return importsvc.PyFloatReprStr(value)
	}
	if value == math.Trunc(value) {
		z, _ := new(big.Float).SetFloat64(value).Int(nil)
		return z.String()
	}
	text := strconv.FormatFloat(value, 'f', 12, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "0"
	}
	return text
}

func emitRawExtras(lines *[]string, key string, valRaw any, indent string) {
	if lst, ok := valRaw.([]any); ok {
		for _, item := range lst {
			emitRawExtras(lines, key, item, indent)
		}
		return
	}
	if od, ok := valRaw.(*OD); ok {
		if od.Len() == 0 {
			*lines = append(*lines, indent+key+" = { }")
			return
		}
		*lines = append(*lines, indent+key+" = {")
		for _, sub := range od.Keys() {
			v, _ := od.Get(sub)
			emitRawExtras(lines, sub, v, indent+"\t")
		}
		*lines = append(*lines, indent+"}")
		return
	}
	var raw string
	switch t := valRaw.(type) {
	case float64:
		raw = fmtHoi4Float(t)
	case nil:
		raw = ""
	default:
		raw = pyStr(valRaw)
	}
	isBlock := strings.ContainsAny(raw, "\n{=><")
	if !isBlock {
		val := pyStrip(raw)
		if val == "" {
			return
		}
		*lines = append(*lines, indent+key+" = "+val)
		return
	}
	var body []string
	for _, vl := range pySplitlines(raw) {
		if s := pyStrip(vl); s != "" {
			body = append(body, s)
		}
	}
	if len(body) == 0 {
		*lines = append(*lines, indent+key+" = { }")
		return
	}
	*lines = append(*lines, indent+key+" = {")
	for _, vl := range body {
		*lines = append(*lines, indent+"\t"+vl)
	}
	*lines = append(*lines, indent+"}")
}

func formatModifierStat(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "yes"
		}
		return "no"
	case float64:
		return fmtHoi4Float(t)
	case *big.Int:
		return t.String()
	}
	s := pyStr(v)
	if strings.ContainsAny(s, " \t/\\\"") {
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	return s
}

func fmtDecisionScalar(value any) string {
	switch t := value.(type) {
	case bool:
		if t {
			return "yes"
		}
		return "no"
	case float64:
		s := strconv.FormatFloat(t, 'f', 6, 64)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		if s == "" {
			return "0"
		}
		return s
	}
	return pyStrip(pyStr(value))
}

func serializeConditionAST(ast any, indent int) []string {
	var children []any
	if od, ok := ast.(*OD); ok {
		children, _ = od.GetDef("children", []any{}).([]any)
	} else if lst, ok := ast.([]any); ok {
		children = lst
	}
	tabs := strings.Repeat("\t", indent)
	var out []string
	for _, childAny := range children {
		child, ok := childAny.(*OD)
		if !ok {
			continue
		}
		op := orEmptyStr(child.GetDef("op", ""))
		if op == "" {
			continue
		}
		if kind, _ := child.GetDef("kind", nil).(string); kind == "block" {
			out = append(out, tabs+op+" = {")
			out = append(out, serializeConditionAST(child.GetDef("children", []any{}), indent+1)...)
			out = append(out, tabs+"}")
		} else {
			valStr := pyStr(child.GetDef("value", ""))
			if valStr == "" || strings.ContainsAny(valStr, " /\\") {
				valStr = "\"" + valStr + "\""
			}
			out = append(out, tabs+op+" = "+valStr)
		}
	}
	return out
}

func synthCountryGate(scopeKind string, countryTag string, sharedTags []any) (string, bool) {
	sk := pyLower(scopeKind)
	if sk == "" {
		sk = "own"
	}
	var tags []string
	for _, t := range sharedTags {
		s := orEmptyStr(t)
		if s != "" {
			tags = append(tags, pyUpper(s))
		}
	}
	switch sk {
	case "own":
		owner := pyUpper(countryTag)
		if owner != "" {
			return "original_tag = " + owner, true
		}
		return "", false
	case "explicit":
		members := tags
		if len(members) == 0 && countryTag != "" {
			members = []string{pyUpper(countryTag)}
		}
		if len(members) == 0 {
			return "", false
		}
		parts := make([]string, 0, len(members))
		for _, t := range members {
			parts = append(parts, "original_tag = "+t)
		}
		return "OR = { " + strings.Join(parts, " ") + " }", true
	case "global_except":
		if len(tags) == 0 {
			return "", false
		}
		parts := make([]string, 0, len(tags))
		for _, t := range tags {
			parts = append(parts, "original_tag = "+t)
		}
		return "NOT = { OR = { " + strings.Join(parts, " ") + " } }", true
	}
	return "", false
}

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
