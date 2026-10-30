package emitscripts

import (
	"math"
	"math/big"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
)

type Dict struct {
	keys   []string
	values map[string]any
}

func NewDict() *Dict { return &Dict{values: map[string]any{}} }

func (d *Dict) Set(k string, v any) {
	if _, ok := d.values[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.values[k] = v
}

func (d *Dict) Get(k string) any {
	if d == nil {
		return nil
	}
	return d.values[k]
}

func (d *Dict) Has(k string) bool {
	if d == nil {
		return false
	}
	_, ok := d.values[k]
	return ok
}

func (d *Dict) Keys() []string {
	if d == nil {
		return nil
	}
	return d.keys
}

func (d *Dict) Len() int {
	if d == nil {
		return 0
	}
	return len(d.keys)
}

func (d *Dict) Truthy() bool { return d.Len() > 0 }

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }

func pyRStrip(s string) string { return strings.TrimRightFunc(s, pyIsSpace) }

func pyStripChar(s string, c rune) string {
	return strings.TrimFunc(s, func(r rune) bool { return r == c })
}

func pyLower(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		switch r {
		case 0x0130:
			b.WriteString("i̇")
		case 0x03A3:
			if sigmaIsFinal(rs, i) {
				b.WriteRune(0x03C2)
			} else {
				b.WriteRune(0x03C3)
			}
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r)
}

func sigmaIsFinal(rs []rune, i int) bool {
	caseIgnorable := func(r rune) bool {
		return unicode.In(r, unicode.Mn, unicode.Me) || r == '\'' || r == 0x2019
	}
	before := false
	for j := i - 1; j >= 0; j-- {
		if caseIgnorable(rs[j]) {
			continue
		}
		before = isCased(rs[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(rs); j++ {
		if caseIgnorable(rs[j]) {
			continue
		}
		return !isCased(rs[j])
	}
	return true
}

var upperSpecial = map[rune]string{
	0x00DF: "SS", 0xFB00: "FF", 0xFB01: "FI", 0xFB02: "FL", 0xFB03: "FFI",
	0xFB04: "FFL", 0xFB05: "ST", 0xFB06: "ST", 0x0149: "ʼN", 0x01F0: "J̌",
	0x0390: "Ϊ́", 0x03B0: "Ϋ́", 0x0587: "ԵՒ", 0xFB13: "ՄՆ",
	0xFB14: "ՄԵ", 0xFB15: "ՄԻ", 0xFB16: "ՎՆ", 0xFB17: "ՄԽ",
	0x1E96: "H̱", 0x1E97: "T̈", 0x1E98: "W̊", 0x1E99: "Y̊", 0x1E9A: "Aʾ",
}

func pyUpper(s string) string {
	var b strings.Builder
	for _, r := range s {
		if exp, ok := upperSpecial[r]; ok {
			b.WriteString(exp)
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

func pyCasefold(s string) string {
	return strings.ReplaceAll(pyLower(s), "ß", "ss")
}

func PyStr(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case string:
		return t
	case bool:
		if t {
			return "True"
		}
		return "False"
	case *big.Int:
		return t.String()
	case float64:
		return importsvc.PyFloatReprStr(t)
	case []any:
		return pyReprList(t)
	case *Dict:
		return pyReprDict(t)
	}
	return ""
}

func pyReprStr(s string) string {
	quote := byte('\'')
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case rune(quote):
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				const hex = "0123456789abcdef"
				b.WriteString(`\x`)
				b.WriteByte(hex[(r>>4)&0xf])
				b.WriteByte(hex[r&0xf])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte(quote)
	return b.String()
}

func pyRepr(v any) string {
	if s, ok := v.(string); ok {
		return pyReprStr(s)
	}
	return PyStr(v)
}

func pyReprList(l []any) string {
	parts := make([]string, 0, len(l))
	for _, it := range l {
		parts = append(parts, pyRepr(it))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func pyReprDict(d *Dict) string {
	parts := make([]string, 0, d.Len())
	for _, k := range d.Keys() {
		parts = append(parts, pyReprStr(k)+": "+pyRepr(d.Get(k)))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func pyTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case *big.Int:
		return t.Sign() != 0
	case float64:
		return t != 0
	case []any:
		return len(t) > 0
	case *Dict:
		return t.Truthy()
	}
	return true
}

func pyIntStr(v any) string {
	switch t := v.(type) {
	case *big.Int:
		return t.String()
	case bool:
		if t {
			return "1"
		}
		return "0"
	case float64:
		if math.IsInf(t, 0) || math.IsNaN(t) {
			return PyStr(t)
		}
		z, _ := new(big.Float).SetFloat64(t).Int(nil)
		return z.String()
	}
	return PyStr(v)
}

func pyFloatOf(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case *big.Int:
		f, _ := new(big.Float).SetInt(t).Float64()
		return f
	case bool:
		if t {
			return 1
		}
		return 0
	}
	return 0
}

func pySplitlines(s string) []string {
	isBoundary := func(r rune) bool {
		switch r {
		case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
			return true
		}
		return false
	}
	var out []string
	rs := []rune(s)
	start := 0
	i := 0
	for i < len(rs) {
		if isBoundary(rs[i]) {
			out = append(out, string(rs[start:i]))
			if rs[i] == '\r' && i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			i++
			start = i
		} else {
			i++
		}
	}
	if start < len(rs) {
		out = append(out, string(rs[start:]))
	}
	return out
}

func splitNL(s string) []string { return strings.Split(s, "\n") }
