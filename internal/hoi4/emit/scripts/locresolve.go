package emitscripts

import "regexp"

var placeholderDisplayNames = map[string]bool{
	"new character":  true,
	"new focus":      true,
	"new idea":       true,
	"new technology": true,
}

var machineTokenID = regexp.MustCompile(`[_.\p{Nd}]`)

func IsPlaceholderDisplayName(text, idValue string) bool {
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
			if pyCasefold(t) == pyCasefold(ident) && machineTokenID.MatchString(ident) {
				return true
			}
		}
	}
	return false
}

func resolveDisplayName(c *Character, langCode string) string {
	if c.DisplayNameLocs != nil {
		if val, ok := c.DisplayNameLocs.Get(langCode).(string); ok {
			if s := pyStrip(val); s != "" {
				return s
			}
		}
	}
	legacy := ""
	switch langCode {
	case "en":
		legacy = c.NameEn
	case "zh":
		legacy = c.NameZh
	}
	return pyStrip(legacy)
}

func ResolveMeaningfulDisplayName(c *Character, langCode, idValue string) (string, bool) {
	resolved := resolveDisplayName(c, langCode)
	if !IsPlaceholderDisplayName(resolved, idValue) {
		return resolved, resolved != ""
	}
	if langCode != "en" {
		return "", false
	}
	literal := pyStrip(c.Name)
	if !IsPlaceholderDisplayName(literal, idValue) {
		return literal, literal != ""
	}
	var candidates []string
	if c.DisplayNameLocs != nil {
		candidates = append(candidates, c.DisplayNameLocs.Keys()...)
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
		val := resolveDisplayName(c, other)
		if !IsPlaceholderDisplayName(val, idValue) {

			return "", true
		}
	}
	return "", false
}

func resolveDescription(c *Character, langCode string) string {
	if c.DescriptionLocs != nil {
		if val, ok := c.DescriptionLocs.Get(langCode).(string); ok {
			if s := pyStrip(val); s != "" {
				return s
			}
		}
	}
	legacy := ""
	switch langCode {
	case "en":
		legacy = c.DescEn
	case "zh":
		legacy = c.DescZh
	}
	return pyStrip(legacy)
}

func ResolveDescriptionAnyLang(c *Character) string {
	if en := resolveDescription(c, "en"); en != "" {
		return en
	}
	var candidates []string
	if c.DescriptionLocs != nil {
		candidates = append(candidates, c.DescriptionLocs.Keys()...)
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
		if val := resolveDescription(c, other); val != "" {
			return val
		}
	}
	return ""
}
