package locout

var placeholderDisplayNames = map[string]bool{
	"new character":  true,
	"new focus":      true,
	"new idea":       true,
	"new technology": true,
}

func PlaceholderDisplayNamesSorted() []string {
	return []string{"new character", "new focus", "new idea", "new technology"}
}

type Text struct {
	S    string
	Gate bool
}

func (t Text) Truthy() bool { return t.Gate || t.S != "" }

func orText(a, b Text) Text {
	if a.Truthy() {
		return a
	}
	return b
}

const perLocaleOmissionPrefix = "# HAGANE_PER_LOCALE_EMPTY "

type legacyAttrs struct{ en, zh string }

var legacyName = legacyAttrs{"name_en", "name_zh"}
var legacyTitle = legacyAttrs{"title_en", "title_zh"}
var legacyDesc = legacyAttrs{"desc_en", "desc_zh"}

func resolveJSONLocalisation(entity *Obj, langCode, mapField string, legacy legacyAttrs) string {
	locs := entity.Attr(mapField)
	if locs.Kind == KObj {
		if val, ok := locs.Obj.Get(langCode); ok && val.Kind == KStr {
			if stripped := pyStrip(val.Str); stripped != "" {
				return stripped
			}
		}
	}
	legacyVal := ""
	switch langCode {
	case "en":
		legacyVal = strOrEmpty(entity.Attr(legacy.en))
	case "zh":
		legacyVal = strOrEmpty(entity.Attr(legacy.zh))
	}
	return pyStrip(legacyVal)
}

func firstJSONLocalisation(entity *Obj, mapField string, legacy legacyAttrs) string {
	if val := resolveJSONLocalisation(entity, "en", mapField, legacy); val != "" {
		return val
	}
	locs := entity.Attr(mapField)
	if locs.Kind == KObj {
		for _, key := range locs.Obj.Keys {
			if val, ok := locs.Obj.Get(key); ok && val.Kind == KStr {
				if stripped := pyStrip(val.Str); stripped != "" {
					return stripped
				}
			}
		}
	}
	return resolveJSONLocalisation(entity, "zh", mapField, legacy)
}

func resolveDisplayName(entity *Obj, langCode string, legacy legacyAttrs) string {
	locs := entity.Attr("display_name_locs")
	if locs.Kind == KObj {
		if val, ok := locs.Obj.Get(langCode); ok && val.Kind == KStr {
			if stripped := pyStrip(val.Str); stripped != "" {
				return stripped
			}
		}
	}
	legacyVal := ""
	switch langCode {
	case "en":
		legacyVal = strOrEmpty(entity.Attr(legacy.en))
	case "zh":
		legacyVal = strOrEmpty(entity.Attr(legacy.zh))
	}
	return pyStrip(legacyVal)
}

func IsPlaceholderDisplayName(text string, idValue Value) bool {
	t := pyStrip(text)
	if t == "" {
		return true
	}
	if placeholderDisplayNames[pyCaseFold(t)] {
		return true
	}
	if PyTruthy(idValue) {
		ident := pyStrip(PyStr(idValue))
		if ident != "" {
			if t == ident {
				return true
			}
			if pyCaseFold(t) == pyCaseFold(ident) && machineTokenID(ident) {
				return true
			}
		}
	}
	return false
}

func resolveMeaningfulDisplayName(
	entity *Obj, langCode string, idValue Value,
	legacy legacyAttrs, literalAttr string, perLocale bool,
) Text {
	resolved := resolveDisplayName(entity, langCode, legacy)
	if !IsPlaceholderDisplayName(resolved, idValue) {
		return Text{S: resolved}
	}
	if langCode != "en" {
		return Text{}
	}
	if literalAttr != "" {

		literal := pyStrip(strOrPyStr(entity.Attr(literalAttr)))
		if !IsPlaceholderDisplayName(literal, idValue) {
			return Text{S: literal}
		}
	}
	locs := entity.Attr("display_name_locs")
	candidates := []string{}
	if locs.Kind == KObj {
		candidates = append(candidates, locs.Obj.Keys...)
	}
	hasZh := false
	for _, k := range candidates {
		if k == "zh" {
			hasZh = true
			break
		}
	}
	if !hasZh {
		candidates = append(candidates, "zh")
	}
	for _, other := range candidates {
		if other == "en" {
			continue
		}
		val := resolveDisplayName(entity, other, legacy)
		if !IsPlaceholderDisplayName(val, idValue) {
			if perLocale {
				return Text{Gate: true}
			}
			return Text{S: val}
		}
	}
	return Text{}
}

func strOrPyStr(v Value) string {
	if !PyTruthy(v) {
		return ""
	}
	return PyStr(v)
}

func HasMeaningfulDisplayName(entity *Obj, idValue Value) bool {
	return resolveMeaningfulDisplayName(
		entity, "en", idValue, legacyName, "name", true,
	).Truthy()
}

func resolveDescription(entity *Obj, langCode string, legacy legacyAttrs) string {
	locs := entity.Attr("description_locs")
	if locs.Kind == KObj {
		if val, ok := locs.Obj.Get(langCode); ok && val.Kind == KStr {
			if stripped := pyStrip(val.Str); stripped != "" {
				return stripped
			}
		}
	}
	legacyVal := ""
	switch langCode {
	case "en":
		legacyVal = strOrEmpty(entity.Attr(legacy.en))
	case "zh":
		legacyVal = strOrEmpty(entity.Attr(legacy.zh))
	}
	return pyStrip(legacyVal)
}

func resolveDescriptionAnyLang(entity *Obj, legacy legacyAttrs) string {
	if en := resolveDescription(entity, "en", legacy); en != "" {
		return en
	}
	locs := entity.Attr("description_locs")
	candidates := []string{}
	if locs.Kind == KObj {
		candidates = append(candidates, locs.Obj.Keys...)
	}
	hasZh := false
	for _, k := range candidates {
		if k == "zh" {
			hasZh = true
			break
		}
	}
	if !hasZh {
		candidates = append(candidates, "zh")
	}
	for _, other := range candidates {
		if other == "en" {
			continue
		}
		if val := resolveDescription(entity, other, legacy); val != "" {
			return val
		}
	}
	return ""
}

func resolveEffectTooltip(entity *Obj, langCode string) string {
	locs := entity.Attr("effect_tooltip_locs")
	if locs.Kind == KObj {
		if val, ok := locs.Obj.Get(langCode); ok && val.Kind == KStr {
			if stripped := pyStrip(val.Str); stripped != "" {
				return stripped
			}
		}
	}
	return ""
}

func HasMeaningfulDisplayNameLiteral(entity *Obj, idValue Value, literalAttr string) bool {
	return resolveMeaningfulDisplayName(
		entity, "en", idValue, legacyName, literalAttr, true,
	).Truthy()
}
