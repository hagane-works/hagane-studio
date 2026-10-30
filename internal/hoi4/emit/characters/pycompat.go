package characters

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }

func pyRstrip(s string) string { return strings.TrimRightFunc(s, pyIsSpace) }

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

func pySplitlines(s string) []string {
	isBreak := func(r rune) bool {
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
		if isBreak(rs[i]) {
			out = append(out, string(rs[start:i]))
			if rs[i] == '\r' && i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			i++
			start = i
			continue
		}
		i++
	}
	if start < len(rs) {
		out = append(out, string(rs[start:]))
	}
	return out
}

func sha256Hex10(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:10]
}

var nfkdLatin = map[rune]string{
	'À': "À", 'Á': "Á", 'Â': "Â", 'Ã': "Ã", 'Ä': "Ä", 'Å': "Å",
	'Ç': "Ç",
	'È': "È", 'É': "É", 'Ê': "Ê", 'Ë': "Ë",
	'Ì': "Ì", 'Í': "Í", 'Î': "Î", 'Ï': "Ï",
	'Ñ': "Ñ",
	'Ò': "Ò", 'Ó': "Ó", 'Ô': "Ô", 'Õ': "Õ", 'Ö': "Ö",
	'Ù': "Ù", 'Ú': "Ú", 'Û': "Û", 'Ü': "Ü",
	'Ý': "Ý",
	'à': "à", 'á': "á", 'â': "â", 'ã': "ã", 'ä': "ä", 'å': "å",
	'ç': "ç",
	'è': "è", 'é': "é", 'ê': "ê", 'ë': "ë",
	'ì': "ì", 'í': "í", 'î': "î", 'ï': "ï",
	'ñ': "ñ",
	'ò': "ò", 'ó': "ó", 'ô': "ô", 'õ': "õ", 'ö': "ö",
	'ù': "ù", 'ú': "ú", 'û': "û", 'ü': "ü",
	'ý': "ý", 'ÿ': "ÿ",
}

func nfkdSubset(s string) string {
	var b strings.Builder
	for _, r := range s {
		if d, ok := nfkdLatin[r]; ok {
			b.WriteString(d)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isCombiningMark(r rune) bool { return r >= 0x0300 && r <= 0x036F }

func pyTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case []string:
		return len(x) > 0
	default:
		if z, ok := bigOf(v); ok {
			return z.Sign() != 0
		}
		return true
	}
}
