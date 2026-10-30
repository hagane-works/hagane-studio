package preview

import (
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/importsvc"
)

type OD struct {
	keys []string
	m    map[string]any
}

func NewOD() *OD { return &OD{m: map[string]any{}} }

func (d *OD) Set(k string, v any) {
	if _, ok := d.m[k]; !ok {
		i := 0
		for i < len(d.keys) && d.keys[i] < k {
			i++
		}
		d.keys = append(d.keys, "")
		copy(d.keys[i+1:], d.keys[i:])
		d.keys[i] = k
	}
	d.m[k] = v
}

func (d *OD) Get(k string) (any, bool) { v, ok := d.m[k]; return v, ok }

func (d *OD) GetDef(k string, def any) any {
	if v, ok := d.m[k]; ok {
		return v
	}
	return def
}

func (d *OD) Has(k string) bool { _, ok := d.m[k]; return ok }
func (d *OD) Keys() []string    { return d.keys }
func (d *OD) Len() int          { return len(d.keys) }
func (d *OD) Del(k string) {
	if _, ok := d.m[k]; !ok {
		return
	}
	delete(d.m, k)
	for i, kk := range d.keys {
		if kk == k {
			d.keys = append(d.keys[:i], d.keys[i+1:]...)
			break
		}
	}
}

func (d *OD) Copy() *OD {
	out := NewOD()
	for _, k := range d.keys {
		out.Set(k, d.m[k])
	}
	return out
}

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string  { return strings.TrimFunc(s, pyIsSpace) }
func pyRStrip(s string) string { return strings.TrimRightFunc(s, pyIsSpace) }
func pyLStripTabs(s string) string {
	return strings.TrimLeft(s, "\t")
}

func pyStripSet(s, chars string) string {
	return strings.TrimFunc(s, func(r rune) bool { return strings.ContainsRune(chars, r) })
}

func pySplitlines(s string) []string {
	var out []string
	start := 0
	rs := []rune(s)
	i := 0
	for i < len(rs) {
		r := rs[i]
		isBreak := r == '\n' || r == '\r' || r == '\v' || r == '\f' ||
			r == 0x1c || r == 0x1d || r == 0x1e || r == 0x85 ||
			r == 0x2028 || r == 0x2029
		if isBreak {
			out = append(out, string(rs[start:i]))
			if r == '\r' && i+1 < len(rs) && rs[i+1] == '\n' {
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

var upperSpecial = map[rune]string{
	0x00DF: "SS", 0xFB00: "FF", 0xFB01: "FI", 0xFB02: "FL", 0xFB03: "FFI",
	0xFB04: "FFL", 0xFB05: "ST", 0xFB06: "ST", 0x0149: "ʼN",
	0x01F0: "J̌", 0x0390: "Ϊ́", 0x03B0: "Ϋ́",
	0x0587: "ԵՒ", 0xFB13: "ՄՆ", 0xFB14: "ՄԵ",
	0xFB15: "ՄԻ", 0xFB16: "ՎՆ", 0xFB17: "ՄԽ",
	0x1E96: "H̱", 0x1E97: "T̈", 0x1E98: "W̊", 0x1E99: "Y̊",
	0x1E9A: "Aʾ",
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

func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) ||
		unicode.In(r, unicode.Lm, unicode.Lo) && unicode.IsLetter(r) && (unicode.ToUpper(r) != r || unicode.ToLower(r) != r)
}

func sigmaIsFinal(rs []rune, i int) bool {

	before := false
	for j := i - 1; j >= 0; j-- {
		if unicode.In(rs[j], unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk) {
			continue
		}
		before = unicode.IsUpper(rs[j]) || unicode.IsLower(rs[j]) || unicode.IsTitle(rs[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(rs); j++ {
		if unicode.In(rs[j], unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk) {
			continue
		}
		return !(unicode.IsUpper(rs[j]) || unicode.IsLower(rs[j]) || unicode.IsTitle(rs[j]))
	}
	return true
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

func pyStr(v any) string {
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
	}
	panic("pyStr: unexpected type")
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
	case *OD:
		return t.Len() > 0
	}
	panic("pyTruthy: unexpected type")
}

func pyIntFromVal(v any) (*big.Int, bool) {
	switch t := v.(type) {
	case *big.Int:
		return t, true
	case bool:
		if t {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case float64:
		if t != t || t > 1e308 || t < -1e308 {
			return nil, false
		}
		z, _ := new(big.Float).SetFloat64(t).Int(nil)
		return z, true
	case string:
		s := pyStrip(t)
		neg := false
		if strings.HasPrefix(s, "+") {
			s = s[1:]
		} else if strings.HasPrefix(s, "-") {
			neg = true
			s = s[1:]
		}
		if s == "" {
			return nil, false
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return nil, false
			}
		}
		z, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return nil, false
		}
		if neg {
			z.Neg(z)
		}
		return z, true
	}
	return nil, false
}

func numCmp(a, b any) int {
	af, aIsF := a.(float64)
	bf, bIsF := b.(float64)
	az, aIsI := a.(*big.Int)
	bz, bIsI := b.(*big.Int)
	switch {
	case aIsI && bIsI:
		return az.Cmp(bz)
	case aIsF && bIsF:
		if af < bf {
			return -1
		} else if af > bf {
			return 1
		}
		return 0
	case aIsI && bIsF:
		x, _ := new(big.Float).SetInt(az).Float64()
		if x < bf {
			return -1
		} else if x > bf {
			return 1
		}
		return 0
	case aIsF && bIsI:
		return -numCmp(b, a)
	}
	panic("numCmp: unexpected types")
}

func pyFloatParse(s string) (float64, bool) {
	t := pyStrip(s)
	if t == "" || strings.ContainsAny(t, "_ \t") {
		return 0, false
	}
	f, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func valGEZero(v any) bool {
	switch t := v.(type) {
	case *big.Int:
		return t.Sign() >= 0
	case float64:
		return t >= 0
	case bool:
		return true
	}
	return false
}

func valGTZero(v any) bool {
	switch t := v.(type) {
	case *big.Int:
		return t.Sign() > 0
	case float64:
		return t > 0
	case bool:
		return t
	}
	return false
}

func pyEqInt(v any, n int64) bool {
	switch t := v.(type) {
	case *big.Int:
		return t.IsInt64() && t.Int64() == n
	case float64:
		return t == float64(n)
	case bool:
		if t {
			return n == 1
		}
		return n == 0
	}
	return false
}
