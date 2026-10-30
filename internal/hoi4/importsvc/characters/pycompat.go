package characters

import (
	"fmt"
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

func pyReprStr(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == quote || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case pyIsPrintable(r):
			b.WriteRune(r)
		case r < 0x100:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

func pyIsPrintable(r rune) bool { return unicode.IsPrint(r) }
