package characters

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

func fmtHoi4Float(v float64) string {
	if math.IsNaN(v) {
		return "nan"
	}
	if math.IsInf(v, 1) {
		return "inf"
	}
	if math.IsInf(v, -1) {
		return "-inf"
	}
	if v == math.Trunc(v) {
		z, _ := new(big.Float).SetFloat64(v).Int(nil)
		return z.String()
	}
	text := strconv.FormatFloat(v, 'f', 12, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "0"
	}
	return text
}

func rawExtraValueStr(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case float64:
		return fmtHoi4Float(v)
	case bool:
		if v {
			return "True"
		}
		return "False"
	case string:
		return v
	default:
		if z, ok := bigOf(v); ok {
			return z.String()
		}
		return fmt.Sprint(v)
	}
}

func emitRawExtrasBlockOrScalar(lines *[]string, key string, val any, indent string) {
	if lst, ok := val.([]any); ok {
		for _, item := range lst {
			emitRawExtrasBlockOrScalar(lines, key, item, indent)
		}
		return
	}
	if lst, ok := val.([]string); ok {
		for _, item := range lst {
			emitRawExtrasBlockOrScalar(lines, key, item, indent)
		}
		return
	}
	raw := rawExtraValueStr(val)
	isBlock := strings.ContainsAny(raw, "\n{=><")
	if !isBlock {
		v := pyStrip(raw)
		if v == "" {
			return
		}
		*lines = append(*lines, indent+key+" = "+v)
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
