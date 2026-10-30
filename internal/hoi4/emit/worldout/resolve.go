package worldout

import (
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

var placeholderDisplayNames = map[string]bool{
	"new character":  true,
	"new focus":      true,
	"new idea":       true,
	"new technology": true,
}

var machineTokenIDRe = regexp.MustCompile(`[_.\p{Nd}]`)

func pyCasefold(s string) string { return strings.ToLower(s) }

func isPlaceholderDisplayName(text, idValue string) bool {
	t := pyStrip(text)
	if t == "" {
		return true
	}
	if placeholderDisplayNames[pyCasefold(t)] {
		return true
	}
	if idValue != "" {
		ident := pyStrip(idValue)
		if ident != "" {
			if t == ident {
				return true
			}
			if pyCasefold(t) == pyCasefold(ident) && machineTokenIDRe.MatchString(ident) {
				return true
			}
		}
	}
	return false
}

func resolveDisplayName(entity *OMap, langCode, legEn, legZh string) string {
	locs := eMap(entity, "display_name_locs")
	if locs != nil {
		if v, ok := locs.Get(langCode); ok {
			if s, isStr := v.(string); isStr {
				stripped := pyStrip(s)
				if stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = eStrOrEmpty(entity, legEn)
	} else if langCode == "zh" {
		legacy = eStrOrEmpty(entity, legZh)
	}
	return pyStrip(legacy)
}

func resolveJSONLocalisation(entity *OMap, langCode, mapField, legEn, legZh string) string {
	locs := eMap(entity, mapField)
	if locs != nil {
		if v, ok := locs.Get(langCode); ok {
			if s, isStr := v.(string); isStr {
				if stripped := pyStrip(s); stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = eStrOrEmpty(entity, legEn)
	} else if langCode == "zh" {
		legacy = eStrOrEmpty(entity, legZh)
	}
	return pyStrip(legacy)
}

func resolveMeaningfulDisplayName(
	entity *OMap, langCode, idValue string,
	legEn, legZh, literalAttr string,
) (value string, gate bool) {
	resolved := resolveDisplayName(entity, langCode, legEn, legZh)
	if !isPlaceholderDisplayName(resolved, idValue) {
		return resolved, false
	}
	if langCode != "en" {
		return "", false
	}
	if literalAttr != "" {
		literal := pyStrip(pyStrOrEmpty(eGet(entity, literalAttr)))
		if !isPlaceholderDisplayName(literal, idValue) {
			return literal, false
		}
	}
	locs := eMap(entity, "display_name_locs")
	var candidates []string
	if locs != nil {
		candidates = append(candidates, locs.Keys()...)
	}
	hasZh := false
	for _, k := range candidates {
		if k == "zh" {
			hasZh = true
		}
	}
	if !hasZh {
		candidates = append(candidates, "zh")
	}
	for _, other := range candidates {
		if other == "en" {
			continue
		}
		val := resolveDisplayName(entity, other, legEn, legZh)
		if !isPlaceholderDisplayName(val, idValue) {
			return "", true
		}
	}
	return "", false
}

type bookmarkNamePlan struct {
	ByLang   map[string]string
	Findings []exportmsg.Message
}

func planBookmarkNames(entity *OMap, langs []string, bookmarkID, nameKey string) bookmarkNamePlan {
	plan := bookmarkNamePlan{ByLang: map[string]string{}}
	var candidates []string
	seen := map[string]bool{}
	add := func(lang string) {
		lang = lowerStrip(lang)
		if lang == "" || seen[lang] {
			return
		}
		seen[lang] = true
		candidates = append(candidates, lang)
	}
	add("en")
	for _, lang := range langs {
		add(lang)
	}
	if locs := eMap(entity, "display_name_locs"); locs != nil {
		for _, lang := range locs.Keys() {
			add(lang)
		}
	}
	add("zh")

	fallback, fallbackLang := "", ""
	for _, lang := range candidates {
		value := resolveDisplayName(entity, lang, "name_en", "name_zh")
		if !isPlaceholderDisplayName(value, bookmarkID) {
			fallback, fallbackLang = value, lang
			break
		}
	}

	for _, lang := range langs {
		value := resolveDisplayName(entity, lang, "name_en", "name_zh")
		if !isPlaceholderDisplayName(value, bookmarkID) {
			plan.ByLang[lang] = value
			continue
		}
		if fallback != "" {
			plan.ByLang[lang] = fallback
			plan.Findings = append(plan.Findings, exportmsg.Message{Key: "bookmark.name_reused", Args: exportmsg.Args{
				"bookmark": bookmarkID, "lang": lang, "fallback": fallbackLang, "key": nameKey,
			}})
			continue
		}
		plan.Findings = append(plan.Findings, exportmsg.Message{Key: "bookmark.name_missing", Args: exportmsg.Args{
			"bookmark": bookmarkID, "lang": lang, "key": nameKey,
		}})
	}
	return plan
}

func resolveDescription(entity *OMap, langCode string) string {
	locs := eMap(entity, "description_locs")
	if locs != nil {
		if v, ok := locs.Get(langCode); ok {
			if s, isStr := v.(string); isStr {
				stripped := pyStrip(s)
				if stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = eStrOrEmpty(entity, "desc_en")
	} else if langCode == "zh" {
		legacy = eStrOrEmpty(entity, "desc_zh")
	}
	return pyStrip(legacy)
}

func resolveDescriptionAnyLang(entity *OMap) string {
	if en := resolveDescription(entity, "en"); en != "" {
		return en
	}
	locs := eMap(entity, "description_locs")
	var candidates []string
	if locs != nil {
		candidates = append(candidates, locs.Keys()...)
	}
	hasZh := false
	for _, k := range candidates {
		if k == "zh" {
			hasZh = true
		}
	}
	if !hasZh {
		candidates = append(candidates, "zh")
	}
	for _, other := range candidates {
		if other == "en" {
			continue
		}
		if val := resolveDescription(entity, other); val != "" {
			return val
		}
	}
	return ""
}

func resolveCountryExportName(country *OMap, langCode string) string {
	tag := toUpper(pyStrip(eStrOrEmpty(country, "tag")))
	resolved, gate := resolveMeaningfulDisplayName(
		country, langCode, tag, "name_en", "name_zh", "name")
	if gate {
		return ""
	}
	value := pyStrip(resolved)
	return value
}
