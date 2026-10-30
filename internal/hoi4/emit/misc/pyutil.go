package emitmisc

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string  { return strings.TrimFunc(s, pyIsSpace) }
func pyLStrip(s string) string { return strings.TrimLeftFunc(s, pyIsSpace) }
func pyRStrip(s string) string { return strings.TrimRightFunc(s, pyIsSpace) }

func pyStripChars(s, chars string) string {
	return strings.Trim(s, chars)
}

func pyUpper(s string) string { return strings.ToUpper(s) }
func pyLower(s string) string { return strings.ToLower(s) }

func pyCasefold(s string) string { return strings.ToLower(s) }

func pyFloatRepr(f float64) string {
	if math.IsNaN(f) {
		return "nan"
	}
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	neg := strings.HasPrefix(sci, "-")
	if neg {
		sci = sci[1:]
	}
	ePos := strings.IndexByte(sci, 'e')
	mant := sci[:ePos]
	exp, _ := strconv.Atoi(sci[ePos+1:])
	digits := strings.Replace(mant, ".", "", 1)

	var out string
	switch {
	case exp < -4 || exp >= 16:
		out = mant + "e" + pySciExp(exp)
	case exp >= len(digits)-1:
		out = digits + strings.Repeat("0", exp-(len(digits)-1)) + ".0"
	case exp >= 0:
		out = digits[:exp+1] + "." + digits[exp+1:]
	default:
		out = "0." + strings.Repeat("0", -exp-1) + digits
	}
	if neg {
		out = "-" + out
	}
	return out
}

func pySciExp(exp int) string {
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	s := strconv.Itoa(exp)
	if len(s) < 2 {
		s = "0" + s
	}
	return sign + s
}

func pyIntTrunc(f float64) int64 { return int64(math.Trunc(f)) }

func isIntegralFloat(f float64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return false
	}
	return f == math.Trunc(f)
}

func pyRound4Repr(f float64) string {
	s := strconv.FormatFloat(f, 'f', 4, 64)
	r, _ := strconv.ParseFloat(s, 64)
	return pyFloatRepr(r)
}

func fmtFixedTrim(v float64, prec int) string {
	s := strconv.FormatFloat(v, 'f', prec, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func FmtHOI4Float(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return pyFloatRepr(v)
	}
	if isIntegralFloat(v) {

		return strconv.FormatInt(pyIntTrunc(v), 10)
	}
	text := fmtFixedTrim(v, 12)
	if text == "" {
		return "0"
	}
	return text
}

type gateStr struct{}

var PerLocaleGate any = gateStr{}

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
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case *big.Int:
		return t.String()
	case float64:
		return pyFloatRepr(t)
	case gateStr:
		return ""
	}
	return ""
}

func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case *big.Int:
		return t.Sign() != 0
	case float64:
		return t != 0
	case []any:
		return len(t) > 0
	case *OD:
		return t != nil && len(t.keys) > 0
	case *NS:
		return t != nil
	case gateStr:
		return true
	}
	return true
}

func orAny(a, b any) any {
	if Truthy(a) {
		return a
	}
	return b
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case *big.Int:
		return t.Int64()
	case float64:
		return pyIntTrunc(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case string:
		f, err := strconv.ParseFloat(pyStrip(t), 64)
		if err != nil {
			return 0
		}
		return pyIntTrunc(f)
	}
	return 0
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case *big.Int:
		f, _ := new(big.Float).SetInt(t).Float64()
		return f, true
	case float64:
		return t, true
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	case string:
		f, err := strconv.ParseFloat(pyStrip(t), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

type OD struct {
	keys []string
	m    map[string]any
}

func NewOD() *OD { return &OD{m: map[string]any{}} }

func (d *OD) Set(k string, v any) *OD {
	if _, ok := d.m[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.m[k] = v
	return d
}

func (d *OD) Get(k string) (any, bool) {
	if d == nil {
		return nil, false
	}
	v, ok := d.m[k]
	return v, ok
}

func (d *OD) GetDef(k string, def any) any {
	if v, ok := d.Get(k); ok {
		return v
	}
	return def
}

func (d *OD) Keys() []string {
	if d == nil {
		return nil
	}
	return d.keys
}

func (d *OD) MapCopy() map[string]any {
	if d == nil {
		return nil
	}
	out := make(map[string]any, len(d.m))
	for _, key := range d.keys {
		out[key] = d.m[key]
	}
	return out
}

func (d *OD) Len() int {
	if d == nil {
		return 0
	}
	return len(d.keys)
}

func (d *OD) Del(k string) any {
	if d == nil {
		return nil
	}
	v, ok := d.m[k]
	if !ok {
		return nil
	}
	delete(d.m, k)
	for i, kk := range d.keys {
		if kk == k {
			d.keys = append(d.keys[:i:i], d.keys[i+1:]...)
			break
		}
	}
	return v
}

func (d *OD) Copy() *OD {
	out := NewOD()
	if d != nil {
		for _, k := range d.keys {
			out.Set(k, d.m[k])
		}
	}
	return out
}

type NS struct {
	m map[string]any
}

func NewNS() *NS { return &NS{m: map[string]any{}} }

func (n *NS) Set(field string, v any) *NS {
	n.m[field] = v
	return n
}

func (n *NS) Attr(field string, def any) any {
	if n == nil {
		return def
	}
	if v, ok := n.m[field]; ok {
		return v
	}
	return def
}

func (n *NS) A(field string) any { return n.Attr(field, nil) }

func (n *NS) Str(field string) string {
	v := n.Attr(field, nil)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return PyStr(v)
}

func (n *NS) StrOr(field, fallback string) string {
	s := n.Str(field)
	if s == "" {
		return fallback
	}
	return s
}

func (n *NS) Int(field string, def int64) int64 {
	v := n.Attr(field, nil)
	if v == nil {
		return def
	}
	return asInt64(v)
}

func (n *NS) Bool(field string, def bool) bool {
	v := n.Attr(field, nil)
	if v == nil {
		return def
	}
	return Truthy(v)
}

func (n *NS) List(field string) []any {
	v := n.Attr(field, nil)
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}

func (n *NS) Dict(field string) *OD {
	v := n.Attr(field, nil)
	if d, ok := v.(*OD); ok {
		return d
	}
	return nil
}

func asOD(v any) (*OD, bool) {
	d, ok := v.(*OD)
	return d, ok
}

func asNS(v any) (*NS, bool) {
	n, ok := v.(*NS)
	return n, ok
}
