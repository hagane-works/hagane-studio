package loc

import (
	"errors"
	"strings"
	"unicode/utf8"
)

func DecodeScriptText(raw []byte) (string, string, bool) {
	body := raw
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		body = body[3:]
	}
	if utf8.Valid(body) {
		return string(body), "utf-8-sig", true
	}
	s, err := decodeCP1252(raw)
	if err != nil {
		return "", "", false
	}
	return s, "cp1252", true
}

var cp1252High = [32]rune{
	0x20AC, 0, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0, 0x017D, 0,
	0, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0, 0x017E, 0x0178,
}

var errCP1252Undefined = errors.New("cp1252: undefined byte")

func decodeCP1252(raw []byte) (string, error) {
	var b strings.Builder
	b.Grow(len(raw))
	for _, c := range raw {
		switch {
		case c < 0x80:
			b.WriteByte(c)
		case c < 0xA0:
			r := cp1252High[c-0x80]
			if r == 0 {
				return "", errCP1252Undefined
			}
			b.WriteRune(r)
		default:
			b.WriteRune(rune(c))
		}
	}
	return b.String(), nil
}
