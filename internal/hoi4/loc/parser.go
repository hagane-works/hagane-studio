package loc

import (
	"fmt"
	"sort"
	"strings"
)

var suffixFromLang = map[string]string{
	"l_english":      "en",
	"l_simp_chinese": "zh",
	"l_russian":      "ru",
	"l_french":       "fr",
	"l_german":       "de",
	"l_spanish":      "es",
	"l_japanese":     "ja",
	"l_korean":       "ko",
	"l_polish":       "pl",
	"l_braz_por":     "pt",
}

func LanguageToSuffix(lang string) (string, bool) {
	s, ok := suffixFromLang[lang]
	return s, ok
}

func ParseLocText(text string) (language string, entries map[string]string, diagnostics []string) {
	text = strings.TrimPrefix(text, "\ufeff")

	entries = map[string]string{}
	diagnostics = []string{}
	preHeaderDrops := 0
	malformedDrops := 0
	lenientRecoveries := 0

	for _, line := range pySplitLines(text) {
		stripped := pyStrip(line)
		if stripped == "" || stripped[0] == '#' {
			continue
		}
		runes := []rune(line)

		if language == "" {
			if lang, ok := matchLang(runes); ok {
				language = lang
				continue
			}

			if _, _, ok := matchEntryStrict(runes); ok {
				preHeaderDrops++
			} else if _, _, ok := matchEntryLenient(runes); ok {
				preHeaderDrops++
			}
			continue
		}

		if key, val, ok := matchEntryStrict(runes); ok {
			entries[key] = DecodeLocEscapes(val)
			continue
		}
		if key, val, ok := matchEntryLenient(runes); ok {
			entries[key] = DecodeLocEscapes(val)
			lenientRecoveries++
		} else if strings.ContainsRune(stripped, ':') && strings.ContainsRune(stripped, '"') {
			malformedDrops++
		}
	}

	if language == "" {
		diagnostics = append(diagnostics, fmt.Sprintf(
			"no l_* language header found; %d entry-like line(s) dropped",
			preHeaderDrops))
	} else if preHeaderDrops > 0 {
		diagnostics = append(diagnostics, fmt.Sprintf(
			"%d entry-like line(s) before the %s header were dropped (not misfiled)",
			preHeaderDrops, language))
	}
	if lenientRecoveries > 0 {
		diagnostics = append(diagnostics, fmt.Sprintf(
			"%d malformed entry line(s) recovered via lenient parse (inner quote / trailing junk)",
			lenientRecoveries))
	}
	if malformedDrops > 0 {
		diagnostics = append(diagnostics, fmt.Sprintf(
			"%d unrecoverable entry-like line(s) skipped", malformedDrops))
	}

	return language, entries, diagnostics
}

func DecodeLocEscapes(val string) string {
	var b strings.Builder
	b.Grow(len(val))
	n := len(val)
	for i := 0; i < n; {
		c := val[i]
		if c == '\\' && i+1 < n {
			var decoded byte
			switch val[i+1] {
			case 'n':
				decoded = '\n'
			case 't':
				decoded = '\t'
			case '"':
				decoded = '"'
			case '\\':
				decoded = '\\'
			default:
				decoded = 0
			}
			if decoded != 0 {
				b.WriteByte(decoded)
				i += 2
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func matchLang(line []rune) (string, bool) {
	n := len(line)
	i := 0
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	if i+1 >= n || line[i] != 'l' || line[i+1] != '_' {
		return "", false
	}
	start := i
	i += 2
	j := i
	for j < n && pyIsWord(line[j]) {
		j++
	}
	if j == i {
		return "", false
	}
	lang := string(line[start:j])
	i = j
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	if i >= n || line[i] != ':' {
		return "", false
	}
	i++
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	if i != n {
		return "", false
	}
	return lang, true
}

func matchEntryPrefix(line []rune) (key string, after int, ok bool) {
	n := len(line)
	i := 0
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	j := i
	for j < n && isKeyChar(line[j]) {
		j++
	}
	if j == i {
		return "", 0, false
	}
	key = string(line[i:j])
	i = j
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	if i >= n || line[i] != ':' {
		return "", 0, false
	}
	i++
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	for i < n && pyIsDigit(line[i]) {
		i++
	}
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	if i >= n || line[i] != '"' {
		return "", 0, false
	}
	return key, i + 1, true
}

func entryTailOK(line []rune, i int) bool {
	n := len(line)
	for i < n && pyIsSpace(line[i]) {
		i++
	}
	return i == n || line[i] == '#'
}

func matchEntryStrict(line []rune) (key, val string, ok bool) {
	key, i, ok := matchEntryPrefix(line)
	if !ok {
		return "", "", false
	}
	n := len(line)
	start := i
	for {
		if i >= n {
			return "", "", false
		}
		c := line[i]
		if c == '\\' {
			if i+1 >= n {
				return "", "", false
			}
			i += 2
			continue
		}
		if c == '"' {
			break
		}
		i++
	}
	if !entryTailOK(line, i+1) {
		return "", "", false
	}
	return key, string(line[start:i]), true
}

func matchEntryLenient(line []rune) (key, val string, ok bool) {
	key, i, ok := matchEntryPrefix(line)
	if !ok {
		return "", "", false
	}
	for p := len(line) - 1; p >= i; p-- {
		if line[p] == '"' && entryTailOK(line, p+1) {
			return key, string(line[i:p]), true
		}
	}
	return "", "", false
}

type LocFile struct {
	Path string
	Text string
}

func BuildLocMap(files []LocFile) map[string]map[string]string {
	out := map[string]map[string]string{"en": {}, "zh": {}}
	ordered := make([]LocFile, len(files))
	copy(ordered, files)
	sort.SliceStable(ordered, func(a, b int) bool {
		return ordered[a].Path < ordered[b].Path
	})
	for _, f := range ordered {
		lang, entries, _ := ParseLocText(f.Text)
		suffix, ok := LanguageToSuffix(lang)
		if !ok || len(entries) == 0 {
			continue
		}
		m := out[suffix]
		if m == nil {
			m = map[string]string{}
			out[suffix] = m
		}
		for k, v := range entries {
			m[k] = v
		}
	}
	return out
}
