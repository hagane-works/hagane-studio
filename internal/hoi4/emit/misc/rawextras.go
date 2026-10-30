package emitmisc

import "strings"

func EmitRawExtrasBlockOrScalar(lines *[]string, key string, valRaw any, indent string) {
	if lst, ok := valRaw.([]any); ok {
		for _, item := range lst {
			EmitRawExtrasBlockOrScalar(lines, key, item, indent)
		}
		return
	}
	var raw string
	switch t := valRaw.(type) {
	case float64:
		raw = FmtHOI4Float(t)
	case nil:
		raw = ""
	default:
		raw = PyStr(valRaw)
	}
	isBlock := strings.Contains(raw, "\n") || strings.Contains(raw, "{") ||
		strings.Contains(raw, "=") || strings.Contains(raw, ">") ||
		strings.Contains(raw, "<")
	if !isBlock {
		val := pyStrip(raw)
		if val == "" {
			return
		}
		*lines = append(*lines, indent+key+" = "+val)
		return
	}
	var body []string
	for _, vl := range strings.Split(raw, "\n") {
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
