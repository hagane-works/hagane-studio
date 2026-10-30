package locout

import (
	"sort"
	"strings"
)

func YmlEscapeValue(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text) + 8)
	n := len(text)
	for i := 0; i < n; {
		c := text[i]
		if c == '\\' {
			if i+1 < n && isLocEscapeTail(text[i+1]) {
				b.WriteByte('\\')
				b.WriteByte(text[i+1])
				i += 2
				continue
			}
			b.WriteString(`\\`)
			i++
			continue
		}
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\r':
			b.WriteString(`\n`)
			if i+1 < n && text[i+1] == '\n' {
				i++
			}
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
		i++
	}
	return b.String()
}

func isLocEscapeTail(c byte) bool {
	return c == 'n' || c == 't' || c == '"' || c == '\\'
}

func YmlLine(key string, text Text) string {
	safeKey := key
	if !IsLegalLocKey(key) {
		safeKey = SanitizeLocKey(key)
	}
	if text.Gate {
		return " " + perLocaleOmissionPrefix + safeKey
	}
	return " " + safeKey + ": \"" + YmlEscapeValue(text.S) + "\""
}

func ExtractYmlKeys(ymlText string) map[string]bool {
	keys := map[string]bool{}
	for _, rawLine := range pySplitLines(ymlText) {
		line := pyStrip(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			continue
		}
		head := line
		if i := strings.Index(line, " "); i >= 0 {
			head = line[:i]
		}
		if !strings.Contains(head, ":") {
			continue
		}
		key := pyStrip(head[:strings.Index(head, ":")])
		if key != "" {
			keys[key] = true
		}
	}
	return keys
}

func ExtractPerLocaleOmissions(ymlText string) map[string]bool {
	omissions := map[string]bool{}
	for _, rawLine := range pySplitLines(ymlText) {
		line := pyStrip(rawLine)
		if strings.HasPrefix(line, perLocaleOmissionPrefix) {
			key := pyStrip(line[len(perLocaleOmissionPrefix):])
			if key != "" {
				omissions[key] = true
			}
		}
	}
	return omissions
}

func sortedDiff(a, b map[string]bool) []string {
	out := []string{}
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func AssertYmlSymmetric(enText, zhText, label string) []string {
	enKeys := ExtractYmlKeys(enText)
	zhKeys := ExtractYmlKeys(zhText)
	warnings := []string{}
	for _, k := range sortedDiff(enKeys, zhKeys) {
		warnings = append(warnings, label+" · "+k+" · en emit · zh missing")
	}
	for _, k := range sortedDiff(zhKeys, enKeys) {
		warnings = append(warnings, label+" · "+k+" · zh emit · en missing")
	}
	return warnings
}

type LangText struct {
	Lang string
	Text string
}

func AssertYmlSymmetricMulti(locByLang []LangText, label string, perLocale bool) []string {
	warnings := []string{}
	var enText string
	hasEn := false
	for _, lt := range locByLang {
		if lt.Lang == "en" {
			enText = lt.Text
			hasEn = true
			break
		}
	}
	if !hasEn {
		return warnings
	}
	enKeys := ExtractYmlKeys(enText)
	enOmissions := ExtractPerLocaleOmissions(enText)
	for _, lt := range locByLang {
		if lt.Lang == "en" {
			continue
		}
		otherKeys := ExtractYmlKeys(lt.Text)
		otherOmissions := ExtractPerLocaleOmissions(lt.Text)
		for _, k := range sortedDiff(enKeys, otherKeys) {
			if perLocale && otherOmissions[k] {
				continue
			}
			warnings = append(warnings, label+" · "+k+" · en emit · "+lt.Lang+" missing")
		}
		for _, k := range sortedDiff(otherKeys, enKeys) {
			if perLocale && enOmissions[k] {
				continue
			}
			warnings = append(warnings, label+" · "+k+" · "+lt.Lang+" emit · en missing")
		}
	}
	return warnings
}
