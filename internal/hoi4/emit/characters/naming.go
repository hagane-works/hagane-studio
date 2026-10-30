package characters

import "strings"

const assetComponentMax = 80

var windowsReservedComponents = func() map[string]bool {
	m := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}
	for i := 1; i < 10; i++ {
		m["COM"+string(rune('0'+i))] = true
		m["LPT"+string(rune('0'+i))] = true
	}
	return m
}()

func windowsInvalidComponentRune(r rune) bool {
	if r <= 0x1f {
		return true
	}
	switch r {
	case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
		return true
	}
	return false
}

func componentStemUpper(component string) string {
	stem := component
	if i := strings.Index(component, "."); i >= 0 {
		stem = component[:i]
	}
	return pyUpper(stem)
}

func isSafeComponent(component string, maxLength int) bool {
	if component == "" || component == "." || component == ".." {
		return false
	}
	if strings.Contains(component, "..") {
		return false
	}
	for _, r := range component {
		if windowsInvalidComponentRune(r) {
			return false
		}
	}
	if strings.HasSuffix(component, " ") || strings.HasSuffix(component, ".") {
		return false
	}
	if windowsReservedComponents[componentStemUpper(component)] {
		return false
	}
	if len([]rune(component)) > maxLength {
		return false
	}
	return true
}

func collapseRun(s string, pred func(rune) bool, minRun int) string {
	var b strings.Builder
	rs := []rune(s)
	i := 0
	for i < len(rs) {
		if pred(rs[i]) {
			j := i
			for j < len(rs) && pred(rs[j]) {
				j++
			}
			if j-i >= minRun {
				b.WriteRune('_')
			} else {
				for k := i; k < j; k++ {
					b.WriteRune(rs[k])
				}
			}
			i = j
			continue
		}
		b.WriteRune(rs[i])
		i++
	}
	return b.String()
}

func stripChars(s, cutset string) string {
	return strings.Trim(s, cutset)
}

func assetComponent(value string, maxLength int) string {
	raw := pyStrip(value)
	digest := sha256Hex10(raw)

	if isSafeComponent(raw, maxLength) {
		return raw
	}
	if raw == "" {
		return "unnamed"
	}

	component := raw
	var b strings.Builder
	for _, r := range component {
		if windowsInvalidComponentRune(r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	component = b.String()
	component = collapseRun(component, func(r rune) bool { return r == '.' }, 2)
	component = collapseRun(component, func(r rune) bool { return r == '_' }, 1)
	component = stripChars(component, " ._")
	if component == "" {
		component = "asset_" + digest
	}
	if windowsReservedComponents[componentStemUpper(component)] {
		component = "asset_" + component + "_" + digest
	}
	if len([]rune(component)) > maxLength {
		rs := []rune(component)
		head := strings.TrimRight(string(rs[:maxLength-11]), " ._")
		component = head + "_" + digest
	}
	if isSafeComponent(component, maxLength) {
		return component
	}
	return "asset_" + digest
}

func assetCountryTag(value string) string {
	return pyUpper(assetComponent(value, 16))
}

func slugify(text string) string {
	t := pyStrip(pyLower(text))
	t = nfkdSubset(t)
	var b strings.Builder
	for _, r := range t {
		if r == '_' || r == '-' || pyIsSpace(r) || isPyWordRune(r) {
			b.WriteRune(r)
		}
	}
	t = b.String()
	t = collapseRun(t, func(r rune) bool { return pyIsSpace(r) || r == '-' }, 1)
	t = collapseRun(t, func(r rune) bool { return r == '_' }, 1)
	t = strings.Trim(t, "_")
	if t == "" {
		return "unnamed"
	}
	return t
}

func Slugify(text string) string { return slugify(text) }

func MakeCharacterID(countryTag, nameEn string) string {
	return pyUpper(countryTag) + "_" + slugify(nameEn)
}

func isPyWordRune(r rune) bool {
	if r == '_' {
		return true
	}

	return isPyAlpha(r) || isPyNumber(r)
}

func isPyAlpha(r rune) bool  { return isLetterRune(r) }
func isPyNumber(r rune) bool { return isNumberRune(r) }

func extractSlug(characterID, countryTag string) string {
	prefix := pyUpper(countryTag) + "_"
	if strings.HasPrefix(pyUpper(characterID), prefix) {
		rs := []rune(characterID)
		plen := len([]rune(prefix))
		if plen <= len(rs) {
			return assetComponent(string(rs[plen:]), assetComponentMax)
		}
	}
	return assetComponent(slugify(characterID), assetComponentMax)
}

func portraitFilename(countryTag, slug string) string {
	return "portrait_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax) + ".png"
}

func advisorIconFilename(countryTag, slug string) string {
	return "advisor_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax) + ".png"
}

func portraitSpriteName(countryTag, slug string) string {
	return "GFX_portrait_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax)
}

func advisorSpriteName(countryTag, slug string) string {
	return "GFX_advisor_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax)
}

func exportPortraitPath(countryTag, slug string) string {
	return "gfx/leaders/" + pyUpper(countryTag) + "/" + portraitFilename(countryTag, slug)
}

func exportAdvisorIconPath(countryTag, slug string) string {
	return "gfx/interface/ideas/" + advisorIconFilename(countryTag, slug)
}
