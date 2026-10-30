package gfx

import (
	"strings"
	"unicode/utf8"
)

var cp1252High = [32]rune{
	0x20AC, -1, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, -1, 0x017D, -1,
	-1, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, -1, 0x017E, 0x0178,
}

func DecodeScriptText(raw []byte) (string, bool) {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		if utf8.Valid(raw[3:]) {
			return string(raw[3:]), true
		}
	} else if utf8.Valid(raw) {
		return string(raw), true
	}
	var sb strings.Builder
	sb.Grow(len(raw))
	for _, b := range raw {
		switch {
		case b < 0x80:
			sb.WriteByte(b)
		case b < 0xA0:
			r := cp1252High[b-0x80]
			if r < 0 {
				return "", false
			}
			sb.WriteRune(r)
		default:
			sb.WriteRune(rune(b))
		}
	}
	return sb.String(), true
}
