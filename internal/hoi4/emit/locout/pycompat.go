package locout

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

func isPyLineBoundary(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, ' ', ' ':
		return true
	}
	return false
}

func pySplitLines(s string) []string {
	out := []string{}
	runes := []rune(s)
	n := len(runes)
	start, i := 0, 0
	for i < n {
		if isPyLineBoundary(runes[i]) {
			out = append(out, string(runes[start:i]))
			if runes[i] == '\r' && i+1 < n && runes[i+1] == '\n' {
				i += 2
			} else {
				i++
			}
			start = i
		} else {
			i++
		}
	}
	if start < n {
		out = append(out, string(runes[start:]))
	}
	return out
}

func pySplitWS(s string) []string {
	return strings.FieldsFunc(s, pyIsSpace)
}

func pyCaseFold(s string) string {
	var b strings.Builder
	for _, r := range s {
		if exp, ok := caseFoldTable[r]; ok {
			b.WriteString(exp)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isCasedRune(r rune) bool {
	lo, hi := 0, len(casedRanges)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		rg := casedRanges[mid]
		switch {
		case r < rg[0]:
			hi = mid - 1
		case r > rg[1]:
			lo = mid + 1
		default:
			return true
		}
	}
	return false
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
		before = isCasedRune(rs[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(rs); j++ {
		if caseIgnorable(rs[j]) {
			continue
		}
		return !isCasedRune(rs[j])
	}
	return true
}

func lowerOneInto(b *strings.Builder, rs []rune, i int) {
	r := rs[i]
	if r == 0x03A3 {
		if sigmaIsFinal(rs, i) {
			b.WriteRune(0x03C2)
		} else {
			b.WriteRune(0x03C3)
		}
		return
	}
	if exp, ok := lowerFullTable[r]; ok {
		b.WriteString(exp)
		return
	}
	b.WriteRune(r)
}

func pyLower(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i := range rs {
		lowerOneInto(&b, rs, i)
	}
	return b.String()
}

func pyTitle(s string) string {
	rs := []rune(s)
	var b strings.Builder
	previousIsCased := false
	for i, r := range rs {
		if previousIsCased {
			lowerOneInto(&b, rs, i)
		} else if exp, ok := titleFullTable[r]; ok {
			b.WriteString(exp)
		} else {
			b.WriteRune(r)
		}
		previousIsCased = isCasedRune(r)
	}
	return b.String()
}

const (
	hangulSBase = 0xAC00
	hangulLBase = 0x1100
	hangulVBase = 0x1161
	hangulTBase = 0x11A7
	hangulVCnt  = 21
	hangulTCnt  = 28
	hangulNCnt  = hangulVCnt * hangulTCnt
	hangulSCnt  = 11172
)

func nfkdStripInto(b *strings.Builder, r rune) {
	if r >= hangulSBase && r < hangulSBase+hangulSCnt {
		sIndex := r - hangulSBase
		b.WriteRune(hangulLBase + sIndex/hangulNCnt)
		b.WriteRune(hangulVBase + (sIndex%hangulNCnt)/hangulTCnt)
		if t := sIndex % hangulTCnt; t > 0 {
			b.WriteRune(hangulTBase + t)
		}
		return
	}
	if exp, ok := nfkdStripTable[r]; ok {
		b.WriteString(exp)
		return
	}
	b.WriteRune(r)
}

func pyNFKDStripCombining(s string) string {
	var b strings.Builder
	for _, r := range s {
		nfkdStripInto(&b, r)
	}
	return b.String()
}

func pyIsPrintable(r rune) bool {
	if r == ' ' {
		return true
	}
	if unicode.In(r, unicode.Cc, unicode.Cf, unicode.Cs, unicode.Co,
		unicode.Zl, unicode.Zp, unicode.Zs) {
		return false
	}
	return unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)
}

func pyStrRepr(s string) string {
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
		default:
			if pyIsPrintable(r) {
				b.WriteRune(r)
			} else if r < 0x100 {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else if r < 0x10000 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				fmt.Fprintf(&b, `\U%08x`, r)
			}
		}
	}
	b.WriteRune(quote)
	return b.String()
}
