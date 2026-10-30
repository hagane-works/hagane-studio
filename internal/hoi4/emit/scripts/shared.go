package emitscripts

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

func FmtHoi4Float(value float64) string {
	if math.IsNaN(value) {
		return "nan"
	}
	if math.IsInf(value, 1) {
		return "inf"
	}
	if math.IsInf(value, -1) {
		return "-inf"
	}
	if math.Trunc(value) == value {

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

func fmtMioScalar(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "yes"
		}
		return "no"
	case float64:
		return FmtHoi4Float(t)
	}
	return PyStr(v)
}

func EmitMioValue(lines *[]string, key string, value any, indent string) {
	switch t := value.(type) {
	case bool:
		yn := "no"
		if t {
			yn = "yes"
		}
		*lines = append(*lines, indent+key+" = "+yn)
	case *Dict:
		*lines = append(*lines, indent+key+" = {")
		for _, subK := range t.Keys() {
			EmitMioValue(lines, subK, t.Get(subK), indent+"\t")
		}
		*lines = append(*lines, indent+"}")
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			parts = append(parts, fmtMioScalar(item))
		}
		*lines = append(*lines, indent+key+" = { "+strings.Join(parts, " ")+" }")
	default:
		*lines = append(*lines, indent+key+" = "+fmtMioScalar(value))
	}
}

func EmitRawScriptBlock(lines *[]string, key string, raw any, indent string) {
	if !pyTruthy(raw) {
		return
	}
	s := PyStr(raw)
	if pyStrip(s) == "" {
		return
	}
	inner := pyStrip(s)
	if strings.HasPrefix(inner, "{") && strings.HasSuffix(inner, "}") {
		*lines = append(*lines, indent+key+" = "+inner)
		return
	}
	*lines = append(*lines, indent+key+" = {")
	for _, rawLine := range splitNL(inner) {
		st := pyRStrip(rawLine)
		if st != "" {
			*lines = append(*lines, indent+"\t"+st)
		}
	}
	*lines = append(*lines, indent+"}")
}

var mioTraitScriptFields = map[string]bool{
	"available": true, "visible": true, "on_complete": true, "ai_will_do": true,
}

func EmitRawExtrasBlockOrScalar(lines *[]string, key string, valRaw any, indent string) {
	if l, ok := valRaw.([]any); ok {
		for _, item := range l {
			EmitRawExtrasBlockOrScalar(lines, key, item, indent)
		}
		return
	}
	var raw string
	switch t := valRaw.(type) {
	case float64:
		raw = FmtHoi4Float(t)
	case nil:
		raw = ""
	default:
		raw = PyStr(valRaw)
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
		if pyStrip(vl) != "" {
			body = append(body, pyStrip(vl))
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
