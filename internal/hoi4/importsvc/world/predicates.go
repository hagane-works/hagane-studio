package world

import (
	"regexp"
	"strings"
	"unicode"
)

var tagShapeRE = regexp.MustCompile(`\A[A-Z][A-Z0-9]{1,4}\z`)

func IsTagShape(text string) bool {
	if text == "" {
		return false
	}
	return tagShapeRE.MatchString(text)
}

func DeriveCountryTagFromHistoryFilename(filename string) string {
	if filename == "" {
		return ""
	}
	base := filename
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "\\"); i >= 0 {
		base = base[i+1:]
	}
	stem := base
	if i := strings.Index(stem, " "); i >= 0 {
		stem = stem[:i]
	}
	if i := strings.Index(stem, "."); i >= 0 {
		stem = stem[:i]
	}
	stem = pyUpper(pyStrip(stem))
	if IsTagShape(stem) {
		return stem
	}
	return ""
}

func IsHistoryFile(filename string) bool {
	norm := pyLower(strings.ReplaceAll(filename, "\\", "/"))
	return strings.HasPrefix(norm, "history/countries/") && strings.HasSuffix(norm, ".txt")
}

func IsStateFile(filename string) bool {
	norm := pyLower(strings.ReplaceAll(filename, "\\", "/"))
	return strings.HasPrefix(norm, "history/states/") && strings.HasSuffix(norm, ".txt")
}

const pySpaceCls = `[\t\n\x0B\f\r \x1C-\x1F\x{0085}\x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}]`

var seMarkerPrefixRE = regexp.MustCompile(`(?i)add_namespace` + pySpaceCls + `*=` + pySpaceCls + `*"?`)

func isSeClassRune(r rune) bool {
	switch {
	case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
		return true
	case r == 0x017F, r == 0x212A:
		return true
	}
	return false
}

func isPyWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsNumber(r)
}

func seNSMarkerSearch(content string) bool {
	for _, loc := range seMarkerPrefixRE.FindAllStringIndex(content, -1) {
		rs := []rune(content[loc[1]:])
		runEnd := 0
		for runEnd < len(rs) && isSeClassRune(rs[runEnd]) {
			runEnd++
		}
		if runEnd < 3 || rs[runEnd-3] != '_' {
			continue
		}
		s, e := rs[runEnd-2], rs[runEnd-1]
		if !(s == 's' || s == 'S' || s == 0x017F) || !(e == 'e' || e == 'E') {
			continue
		}
		if runEnd == len(rs) || !isPyWordRune(rs[runEnd]) {
			return true
		}
	}
	return false
}

func IsSuperEventFile(filename string, content *string) bool {
	base := filename
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "\\"); i >= 0 {
		base = base[i+1:]
	}
	base = pyLower(base)
	if !strings.HasSuffix(base, "_se.txt") && !strings.HasSuffix(base, "_super_event.txt") {
		return false
	}
	if content == nil {
		return true
	}
	return seNSMarkerSearch(*content)
}
