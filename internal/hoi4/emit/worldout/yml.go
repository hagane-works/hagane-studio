package worldout

import (
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
	"haganeworks/internal/hoi4/exportmsg"
)

var locKeyAllowedRe = regexp.MustCompile(`^[A-Za-z0-9_.'-]+$`)

func toLower(s string) string { return strings.ToLower(s) }
func toUpper(s string) string { return strings.ToUpper(s) }

func isLegalLocKey(key string) bool {
	return key != "" && locKeyAllowedRe.MatchString(key)
}

func sanitizeLocKey(key string) string {
	if key == "" {
		return ""
	}
	folded := nfkdStrip(key)
	var b strings.Builder
	for _, r := range folded {
		if locKeyAllowedRe.MatchString(string(r)) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "_"
	}
	return out
}

func ymlEscapeValue(text string) string {
	return locout.YmlEscapeValue(text)
}

const perLocaleOmissionPrefix = "# HAGANE_PER_LOCALE_EMPTY "

func ymlLine(key, text string, gate bool) string {
	safeKey := key
	if !isLegalLocKey(key) {
		safeKey = sanitizeLocKey(key)
	}
	if gate {
		return " " + perLocaleOmissionPrefix + safeKey
	}
	return " " + safeKey + ": \"" + ymlEscapeValue(text) + "\""
}

func extractYmlKeys(ymlText string) map[string]bool {
	keys := map[string]bool{}
	for _, rawLine := range pySplitlines(ymlText) {
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

func extractPerLocaleOmissions(ymlText string) map[string]bool {
	omissions := map[string]bool{}
	for _, rawLine := range pySplitlines(ymlText) {
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

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func assertYmlSymmetricMulti(langs []string, locByLang map[string]string, label string) []exportmsg.Message {
	warnings := []exportmsg.Message{}
	enText, ok := locByLang["en"]
	if !ok {
		return warnings
	}
	enKeys := extractYmlKeys(enText)
	enOmissions := extractPerLocaleOmissions(enText)
	for _, lang := range langs {
		if lang == "en" {
			continue
		}
		text := locByLang[lang]
		otherKeys := extractYmlKeys(text)
		otherOmissions := extractPerLocaleOmissions(text)
		for _, k := range sortedKeys(enKeys) {
			if otherKeys[k] {
				continue
			}
			if otherOmissions[k] {
				continue
			}
			warnings = append(warnings, ymlSymmetryFinding(label, k, "en", lang))
		}
		for _, k := range sortedKeys(otherKeys) {
			if enKeys[k] {
				continue
			}
			if enOmissions[k] {
				continue
			}
			warnings = append(warnings, ymlSymmetryFinding(label, k, lang, "en"))
		}
	}
	return warnings
}

func ymlSymmetryFinding(label, key, written, missing string) exportmsg.Message {
	return exportmsg.Message{Key: "yml.missing_language", Args: exportmsg.Args{
		"label": label, "key": key, "written": written, "missing": missing,
	}}
}
