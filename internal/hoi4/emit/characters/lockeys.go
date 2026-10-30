package characters

import "strings"

func isLegalLocKeyRune(r rune) bool {
	switch {
	case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		return true
	case r == '_' || r == '.' || r == '\'' || r == '-':
		return true
	}
	return false
}

func isLegalLocKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if !isLegalLocKeyRune(r) {
			return false
		}
	}
	return true
}

func sanitizeLocKey(key string) string {
	if key == "" {
		return ""
	}
	folded := nfkdSubset(key)
	var kept strings.Builder
	for _, r := range folded {
		if isCombiningMark(r) {
			continue
		}
		kept.WriteRune(r)
	}
	var out strings.Builder
	for _, r := range kept.String() {
		if isLegalLocKeyRune(r) {
			out.WriteRune(r)
		} else {
			out.WriteRune('_')
		}
	}
	if out.Len() == 0 {
		return "_"
	}
	return out.String()
}

func characterLocKey(c *Character) string {
	tag := pyStrip(c.Tag)
	if isLegalLocKey(tag) {
		return tag
	}
	name := pyStrip(c.Name)
	if isLegalLocKey(name) {
		return name
	}
	return sanitizeLocKey(tag)
}

var placeholderDisplayNames = map[string]bool{
	"new character":  true,
	"new focus":      true,
	"new idea":       true,
	"new technology": true,
}

func machineTokenID(ident string) bool {
	for _, r := range ident {
		if r == '_' || r == '.' || (r >= '0' && r <= '9') || (r > 0x7f && isDigitRune(r)) {
			return true
		}
	}
	return false
}

func isPlaceholderDisplayName(text, idValue string) bool {
	t := pyStrip(text)
	if t == "" {
		return true
	}
	if placeholderDisplayNames[pyLower(t)] {
		return true
	}
	if idValue != "" {
		ident := pyStrip(idValue)
		if ident != "" {
			if t == ident {
				return true
			}
			if pyLower(t) == pyLower(ident) && machineTokenID(ident) {
				return true
			}
		}
	}
	return false
}

func resolveDisplayName(c *Character, lang string) string {
	if v, ok := kvLookup(c.DisplayNameLocs, lang); ok {
		if s, isStr := v.(string); isStr {
			if stripped := pyStrip(s); stripped != "" {
				return stripped
			}
		}
	}
	switch lang {
	case "en":
		return pyStrip(c.NameEn)
	case "zh":
		return pyStrip(c.NameZh)
	}
	return ""
}

func resolveMeaningfulDisplayNameENTruthy(c *Character, idValue string) bool {
	resolved := resolveDisplayName(c, "en")
	if !isPlaceholderDisplayName(resolved, idValue) {
		return true
	}
	literal := pyStrip(c.Name)
	if !isPlaceholderDisplayName(literal, idValue) {
		return true
	}
	candidates := make([]string, 0, len(c.DisplayNameLocs)+1)
	hasZh := false
	for _, kv := range c.DisplayNameLocs {
		candidates = append(candidates, kv.Key)
		if kv.Key == "zh" {
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
		if !isPlaceholderDisplayName(resolveDisplayName(c, other), idValue) {
			return true
		}
	}
	return false
}

func resolveDescription(c *Character, lang string) string {
	if v, ok := kvLookup(c.DescriptionLocs, lang); ok {
		if s, isStr := v.(string); isStr {
			if stripped := pyStrip(s); stripped != "" {
				return stripped
			}
		}
	}
	switch lang {
	case "en":
		return pyStrip(c.DescEn)
	case "zh":
		return pyStrip(c.DescZh)
	}
	return ""
}

func resolveDescriptionAnyLang(c *Character) string {
	if en := resolveDescription(c, "en"); en != "" {
		return en
	}
	candidates := make([]string, 0, len(c.DescriptionLocs)+1)
	hasZh := false
	for _, kv := range c.DescriptionLocs {
		candidates = append(candidates, kv.Key)
		if kv.Key == "zh" {
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

func isDigitRune(r rune) bool {

	return r >= '0' && r <= '9' || (r > 0x7f && isNumberRune(r) && isDecimalLike(r))
}

func isDecimalLike(r rune) bool {
	return r >= 0x0660
}
