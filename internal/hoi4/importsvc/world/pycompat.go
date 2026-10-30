package world

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

type PyRaise struct{ Type string }

func (e *PyRaise) Error() string { return e.Type }

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

func pyRStripChar(s string, c rune) string {
	return strings.TrimRightFunc(s, func(r rune) bool { return r == c })
}

var upperSpecial = map[rune]string{
	0x00DF: "SS",
	0xFB00: "FF",
	0xFB01: "FI",
	0xFB02: "FL",
	0xFB03: "FFI",
	0xFB04: "FFL",
	0xFB05: "ST",
	0xFB06: "ST",
	0x0149: "ʼN",
	0x01F0: "J̌",
	0x0390: "Ϊ́",
	0x03B0: "Ϋ́",
	0x0587: "ԵՒ",
	0xFB13: "ՄՆ",
	0xFB14: "ՄԵ",
	0xFB15: "ՄԻ",
	0xFB16: "ՎՆ",
	0xFB17: "ՄԽ",
	0x1E96: "H̱",
	0x1E97: "T̈",
	0x1E98: "W̊",
	0x1E99: "Y̊",
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

func unicodeDigitValue(r rune) int {
	if !unicode.IsDigit(r) {
		return -1
	}
	v := 0
	for rr := r; rr > 0 && v < 10 && unicode.IsDigit(rr-1); rr-- {
		v++
	}
	return v
}

func transformDecimalAndSpaceToASCII(s string) (string, bool) {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 128:
			b.WriteRune(r)
		case pyIsSpace(r):
			b.WriteByte(' ')
		default:
			if d := unicodeDigitValue(r); d >= 0 {
				b.WriteByte(byte('0' + d))
			} else {
				return "", false
			}
		}
	}
	return b.String(), true
}

func trimASCIISpace(s string) string {
	return strings.Trim(s, " \t\n\r\v\f")
}

func validPyFloatBody(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	i, n := 0, len(s)
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	rest := strings.ToLower(s[i:])
	if rest == "inf" || rest == "infinity" || rest == "nan" {
		return strings.ReplaceAll(s, "_", ""), true
	}
	isDigit := func(j int) bool { return j < n && s[j] >= '0' && s[j] <= '9' }
	digits := func(j int) (int, bool) {
		if !isDigit(j) {
			return j, false
		}
		j++
		for j < n {
			if isDigit(j) {
				j++
			} else if s[j] == '_' && j+1 < n && isDigit(j+1) {
				j += 2
			} else {
				break
			}
		}
		return j, true
	}
	var sawDigits bool
	i, sawDigits = digits(i)
	if i < n && s[i] == '.' {
		i++
		var frac bool
		i, frac = digits(i)
		sawDigits = sawDigits || frac
	}
	if !sawDigits {
		return "", false
	}
	if i < n && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < n && (s[i] == '+' || s[i] == '-') {
			i++
		}
		var expd bool
		i, expd = digits(i)
		if !expd {
			return "", false
		}
	}
	if i != n {
		return "", false
	}
	return strings.ReplaceAll(s, "_", ""), true
}

func pyFloatFromStr(s string) (float64, bool) {
	t, ok := transformDecimalAndSpaceToASCII(s)
	if !ok {
		return 0, false
	}
	t = trimASCIISpace(t)
	body, ok := validPyFloatBody(t)
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(body, 64)
	if err != nil {
		if ne, isNum := err.(*strconv.NumError); !isNum || ne.Err != strconv.ErrRange {
			return 0, false
		}
	}
	return f, true
}

func pyIntFromStr(s string) (*big.Int, bool) {
	t, ok := transformDecimalAndSpaceToASCII(s)
	if !ok {
		return nil, false
	}
	t = trimASCIISpace(t)
	if t == "" {
		return nil, false
	}
	i, n := 0, len(t)
	if t[i] == '+' || t[i] == '-' {
		i++
	}
	if i >= n {
		return nil, false
	}
	isDigit := func(j int) bool { return j < n && t[j] >= '0' && t[j] <= '9' }
	if !isDigit(i) {
		return nil, false
	}
	for j := i; j < n; j++ {
		if t[j] >= '0' && t[j] <= '9' {
			continue
		}
		if t[j] == '_' && j > i && isDigit(j-1) && isDigit(j+1) {
			continue
		}
		return nil, false
	}
	z, ok := new(big.Int).SetString(strings.ReplaceAll(t, "_", ""), 10)
	return z, ok
}

func pyIntFromFloat(f float64) (*big.Int, bool) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return nil, false
	}
	z, _ := new(big.Float).SetFloat64(f).Int(nil)
	return z, true
}

func pyFloatFromBigInt(z *big.Int) (float64, bool) {
	f, _ := new(big.Float).SetInt(z).Float64()
	if math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func pyIntAny(v any) (*big.Int, *PyRaise) {
	switch t := v.(type) {
	case *big.Int:
		return t, nil
	case bool:
		if t {
			return big.NewInt(1), nil
		}
		return big.NewInt(0), nil
	case float64:
		if math.IsInf(t, 0) {
			return nil, &PyRaise{Type: "OverflowError"}
		}
		if math.IsNaN(t) {
			return nil, &PyRaise{Type: "ValueError"}
		}
		z, _ := pyIntFromFloat(t)
		return z, nil
	case string:
		if z, ok := pyIntFromStr(t); ok {
			return z, nil
		}
		return nil, &PyRaise{Type: "ValueError"}
	default:
		return nil, &PyRaise{Type: "TypeError"}
	}
}

func pyFloatAny(v any) (float64, *PyRaise) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case bool:
		if t {
			return 1, nil
		}
		return 0, nil
	case *big.Int:
		if f, ok := pyFloatFromBigInt(t); ok {
			return f, nil
		}
		return 0, &PyRaise{Type: "OverflowError"}
	case string:
		if f, ok := pyFloatFromStr(t); ok {
			return f, nil
		}
		return 0, &PyRaise{Type: "ValueError"}
	default:
		return 0, &PyRaise{Type: "TypeError"}
	}
}

func coerceInt(v any, def any) (any, *PyRaise) {
	z, exc := pyIntAny(v)
	if exc == nil {
		return z, nil
	}
	if exc.Type == "OverflowError" {
		return nil, exc
	}
	return def, nil
}

func coerceFloat(v any, def any) (any, *PyRaise) {
	f, exc := pyFloatAny(v)
	if exc == nil {
		return f, nil
	}
	if exc.Type == "OverflowError" {
		return nil, exc
	}
	return def, nil
}
