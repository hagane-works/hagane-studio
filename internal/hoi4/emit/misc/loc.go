package emitmisc

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/emit/locout"
)

var LangCodeToHOI4 = map[string]string{
	"en": "english",
	"zh": "simp_chinese",
	"ru": "russian",
	"fr": "french",
	"de": "german",
	"es": "spanish",
	"ja": "japanese",
	"ko": "korean",
	"pl": "polish",
	"pt": "braz_por",
}

func langHeader(code string) string { return "l_" + LangCodeToHOI4[code] }

const locReplaceSubdir = "replace"

func hoi4LangSubdir(code string) string { return LangCodeToHOI4[code] + "/" + locReplaceSubdir }
func hoi4LangSuffix(code string) string { return langHeader(code) }

type LangTriple struct {
	Code   string
	Subdir string
	Suffix string
}

func EnsureEnFirst(langs []string) []string {
	if len(langs) == 0 {
		return []string{"en", "zh"}
	}
	out := []string{"en"}
	seen := map[string]bool{"en": true}
	for _, code := range langs {
		c := pyLower(pyStrip(code))
		if c == "" || seen[c] {
			continue
		}
		if _, ok := LangCodeToHOI4[c]; !ok {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func BuildExportLangTriples(langs []string) []LangTriple {
	var out []LangTriple
	for _, code := range EnsureEnFirst(langs) {
		out = append(out, LangTriple{code, hoi4LangSubdir(code), hoi4LangSuffix(code)})
	}
	return out
}

func isLegalLocKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
			r == '_' || r == '.' || r == '\'' || r == '-') {
			return false
		}
	}
	return true
}

func sanitizeLocKey(key string) string {
	if key == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range key {
		if isLegalLocKey(string(r)) {
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

const perLocaleOmissionPrefix = "# HAGANE_PER_LOCALE_EMPTY "

func YmlEscapeValue(text string) string {
	return locout.YmlEscapeValue(text)
}

func ymlLine(key string, text any) string {
	safeKey := key
	if !isLegalLocKey(key) {
		safeKey = sanitizeLocKey(key)
	}
	if _, isGate := text.(gateStr); isGate {
		return " " + perLocaleOmissionPrefix + safeKey
	}
	s, _ := text.(string)
	return " " + safeKey + `: "` + YmlEscapeValue(s) + `"`
}

var placeholderDisplayNames = map[string]bool{
	"new character":  true,
	"new focus":      true,
	"new idea":       true,
	"new technology": true,
}

func machineTokenID(s string) bool {
	for _, r := range s {
		if r == '_' || r == '.' || digitRune(r) {
			return true
		}
	}
	return false
}

func resolveDisplayName(entity *NS, langCode string, legacyEn, legacyZh string) string {
	if locs, ok := asOD(entity.A("display_name_locs")); ok {
		if v, has := locs.Get(langCode); has {
			if s, isStr := v.(string); isStr {
				if stripped := pyStrip(s); stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = entity.Str(legacyEn)
	} else if langCode == "zh" {
		legacy = entity.Str(legacyZh)
	}
	return pyStrip(legacy)
}

func resolveJSONLocalisationNS(entity *NS, langCode, mapField, legacyEn, legacyZh string) string {
	if locs, ok := asOD(entity.A(mapField)); ok {
		if v, has := locs.Get(langCode); has {
			if s, isStr := v.(string); isStr {
				if stripped := pyStrip(s); stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = entity.Str(legacyEn)
	} else if langCode == "zh" {
		legacy = entity.Str(legacyZh)
	}
	return pyStrip(legacy)
}

func resolveJSONLocalisationOD(entity *OD, langCode, mapField, legacyEn, legacyZh string) string {
	if locs, ok := asOD(entity.GetDef(mapField, nil)); ok {
		if v, has := locs.Get(langCode); has {
			if s, isStr := v.(string); isStr {
				if stripped := pyStrip(s); stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = PyStr(orAny(entity.GetDef(legacyEn, nil), ""))
	} else if langCode == "zh" {
		legacy = PyStr(orAny(entity.GetDef(legacyZh, nil), ""))
	}
	return pyStrip(legacy)
}

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
			if pyCasefold(t) == pyCasefold(ident) && machineTokenID(ident) {
				return true
			}
		}
	}
	return false
}

func resolveMeaningfulDisplayName(entity *NS, langCode, idValue string,
	legacyEn, legacyZh, literalAttr string) any {
	resolved := resolveDisplayName(entity, langCode, legacyEn, legacyZh)
	if !isPlaceholderDisplayName(resolved, idValue) {
		return resolved
	}
	if langCode != "en" {
		return ""
	}
	if literalAttr != "" {
		literal := pyStrip(entity.Str(literalAttr))
		if !isPlaceholderDisplayName(literal, idValue) {
			return literal
		}
	}
	var candidates []string
	if locs, ok := asOD(entity.A("display_name_locs")); ok {
		candidates = append(candidates, locs.Keys()...)
	}
	hasZh := false
	for _, c := range candidates {
		if c == "zh" {
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
		val := resolveDisplayName(entity, other, legacyEn, legacyZh)
		if !isPlaceholderDisplayName(val, idValue) {
			return PerLocaleGate
		}
	}
	return ""
}

func rmdn(entity *NS, langCode, idValue string) any {
	return resolveMeaningfulDisplayName(entity, langCode, idValue, "name_en", "name_zh", "name")
}

func rmdnNoLiteral(entity *NS, langCode, idValue string) any {
	return resolveMeaningfulDisplayName(entity, langCode, idValue, "name_en", "name_zh", "")
}

func resolveDescription(entity *NS, langCode string) string {
	if locs, ok := asOD(entity.A("description_locs")); ok {
		if v, has := locs.Get(langCode); has {
			if s, isStr := v.(string); isStr {
				if stripped := pyStrip(s); stripped != "" {
					return stripped
				}
			}
		}
	}
	legacy := ""
	if langCode == "en" {
		legacy = entity.Str("desc_en")
	} else if langCode == "zh" {
		legacy = entity.Str("desc_zh")
	}
	return pyStrip(legacy)
}

func resolveEffectTooltip(entity *NS, langCode string) string {
	if locs, ok := asOD(entity.A("effect_tooltip_locs")); ok {
		if v, has := locs.Get(langCode); has {
			if s, isStr := v.(string); isStr {
				if stripped := pyStrip(s); stripped != "" {
					return stripped
				}
			}
		}
	}
	return ""
}

func extractYmlKeys(ymlText string) map[string]bool {
	keys := map[string]bool{}
	for _, rawLine := range strings.Split(ymlText, "\n") {
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
	for _, rawLine := range strings.Split(ymlText, "\n") {
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

type locByLang struct {
	keys []string
	m    map[string]string
}

func newLocByLang() *locByLang { return &locByLang{m: map[string]string{}} }

func (l *locByLang) set(k, v string) {
	if _, ok := l.m[k]; !ok {
		l.keys = append(l.keys, k)
	}
	l.m[k] = v
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func assertYmlSymmetricMulti(loc *locByLang, label string) []string {
	var warnings []string
	enText, hasEn := loc.m["en"]
	if !hasEn {
		return warnings
	}
	enKeys := extractYmlKeys(enText)
	enOmissions := extractPerLocaleOmissions(enText)
	for _, lang := range loc.keys {
		if lang == "en" {
			continue
		}
		text := loc.m[lang]
		otherKeys := extractYmlKeys(text)
		otherOmissions := extractPerLocaleOmissions(text)
		for _, k := range sortedKeys(enKeys) {
			if otherKeys[k] {
				continue
			}
			if otherOmissions[k] {
				continue
			}
			warnings = append(warnings, label+" · "+k+" · en emit · "+lang+" missing")
		}
		for _, k := range sortedKeys(otherKeys) {
			if enKeys[k] {
				continue
			}
			if enOmissions[k] {
				continue
			}
			warnings = append(warnings, label+" · "+k+" · "+lang+" emit · en missing")
		}
	}
	return warnings
}

func GenerateFocusLocalisation(trees []*NS, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, tree := range trees {
		for _, nodeAny := range tree.List("nodes") {
			node, _ := asNS(nodeAny)
			focusID := node.Str("focus_id")
			nameEn := rmdn(node, "en", focusID)
			if Truthy(nameEn) {
				nameLang := nameEn
				if lang != "en" {
					nameLang = rmdn(node, lang, focusID)
				}
				nameVal := nameEn
				if lang != "en" {
					nameVal = orAny(nameLang, nameEn)
				}
				lines = append(lines, ymlLine(focusID, nameVal))
			}
			descEn := resolveDescription(node, "en")
			descLang := descEn
			if lang != "en" {
				descLang = resolveDescription(node, lang)
			}
			if descEn != "" || descLang != "" {
				descVal := descEn
				if lang != "en" {
					descVal = orStr(descLang, descEn)
				}
				lines = append(lines, ymlLine(focusID+"_desc", descVal))
			}
			ttEn := resolveEffectTooltip(node, "en")
			ttLang := ttEn
			if lang != "en" {
				ttLang = resolveEffectTooltip(node, lang)
			}
			if ttEn != "" || ttLang != "" {
				ttVal := ttEn
				if lang != "en" {
					ttVal = orStr(ttLang, ttEn)
				}
				lines = append(lines, ymlLine(focusID+"_et_tt", ttVal))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateTechLocalisation(techs []*NS, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, tech := range techs {
		techID := tech.Str("tech_id")
		nameEn := rmdn(tech, "en", techID)
		if Truthy(nameEn) {
			nameLang := nameEn
			if lang != "en" {
				nameLang = rmdn(tech, lang, techID)
			}
			nameVal := nameEn
			if lang != "en" {
				nameVal = orAny(nameLang, nameEn)
			}
			lines = append(lines, ymlLine(techID, nameVal))
		}
		descEn := resolveDescription(tech, "en")
		descLang := descEn
		if lang != "en" {
			descLang = resolveDescription(tech, lang)
		}
		if descEn != "" || descLang != "" {
			descVal := descEn
			if lang != "en" {
				descVal = orStr(descLang, descEn)
			}
			lines = append(lines, ymlLine(techID+"_desc", descVal))
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateDecisionsLocalisation(categories, decisions []*NS, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, cat := range categories {
		catID := cat.Str("category_id")
		catNameEn := rmdnNoLiteral(cat, "en", catID)
		if Truthy(catNameEn) {
			catNameLang := catNameEn
			if lang != "en" {
				catNameLang = rmdnNoLiteral(cat, lang, catID)
			}
			val := catNameEn
			if lang != "en" {
				val = orAny(catNameLang, catNameEn)
			}
			lines = append(lines, ymlLine(catID, val))
		}
		catDescEn := resolveDescription(cat, "en")
		catDescLang := catDescEn
		if lang != "en" {
			catDescLang = resolveDescription(cat, lang)
		}
		if catDescEn != "" || catDescLang != "" {
			val := catDescEn
			if lang != "en" {
				val = orStr(catDescLang, catDescEn)
			}
			lines = append(lines, ymlLine(catID+"_desc", val))
		}
	}
	for _, dec := range decisions {
		decID := dec.Str("decision_id")
		decNameEn := rmdnNoLiteral(dec, "en", decID)
		if Truthy(decNameEn) {
			decNameLang := decNameEn
			if lang != "en" {
				decNameLang = rmdnNoLiteral(dec, lang, decID)
			}
			val := decNameEn
			if lang != "en" {
				val = orAny(decNameLang, decNameEn)
			}
			lines = append(lines, ymlLine(decID, val))
		}
		decDescEn := resolveDescription(dec, "en")
		decDescLang := decDescEn
		if lang != "en" {
			decDescLang = resolveDescription(dec, lang)
		}
		if decDescEn != "" || decDescLang != "" {
			val := decDescEn
			if lang != "en" {
				val = orStr(decDescLang, decDescEn)
			}
			lines = append(lines, ymlLine(decID+"_desc", val))
		}
		decTtEn := resolveEffectTooltip(dec, "en")
		decTtLang := decTtEn
		if lang != "en" {
			decTtLang = resolveEffectTooltip(dec, lang)
		}
		if decTtEn != "" || decTtLang != "" {
			val := decTtEn
			if lang != "en" {
				val = orStr(decTtLang, decTtEn)
			}
			lines = append(lines, ymlLine(decID+"_et_tt", val))
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateEquipmentLocalisation(equipment []*NS, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, eq := range equipment {
		eqID := eq.Str("equipment_id")
		nameEn := rmdn(eq, "en", eqID)
		if Truthy(nameEn) {
			var nameLang any = nameEn
			if lang != "en" {
				nameLang = resolveDisplayName(eq, lang, "name_en", "name_zh")
			}
			nameVal := nameEn
			if lang != "en" {
				nameVal = orAny(nameLang, nameEn)
			}
			lines = append(lines, ymlLine(eqID, nameVal))
		}
		descEn := resolveDescription(eq, "en")
		descLang := descEn
		if lang != "en" {
			descLang = resolveDescription(eq, lang)
		}
		if descEn != "" || descLang != "" {
			descVal := descEn
			if lang != "en" {
				descVal = orStr(descLang, descEn)
			}
			lines = append(lines, ymlLine(eqID+"_desc", descVal))
		}
	}
	return strings.Join(lines, "\n")
}

func GenerateSubUnitLocalisation(subUnits []*NS, lang string) string {
	lines := []string{langHeader(lang) + ":"}
	for _, su := range subUnits {
		suID := su.Str("sub_unit_id")
		nameEn := rmdn(su, "en", suID)
		if Truthy(nameEn) {
			var nameLang any = nameEn
			if lang != "en" {
				nameLang = resolveDisplayName(su, lang, "name_en", "name_zh")
			}
			nameVal := nameEn
			if lang != "en" {
				nameVal = orAny(nameLang, nameEn)
			}
			lines = append(lines, ymlLine(suID, nameVal))
		}
		descEn := resolveDescription(su, "en")
		descLang := descEn
		if lang != "en" {
			descLang = resolveDescription(su, lang)
		}
		if descEn != "" || descLang != "" {
			descVal := descEn
			if lang != "en" {
				descVal = orStr(descLang, descEn)
			}
			lines = append(lines, ymlLine(suID+"_desc", descVal))
		}
	}
	return strings.Join(lines, "\n")
}
