package loc

import (
	"strings"
	"unicode"
)

func pyIsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

func pyIsDigit(r rune) bool { return unicode.IsDigit(r) }

func pyIsWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

func isKeyChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-'
}

func isPyLineBoundary(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, '\u2028', '\u2029':
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

func pyStrip(s string) string { return strings.TrimFunc(s, pyIsSpace) }
