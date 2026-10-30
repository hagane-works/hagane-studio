package preview

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

const assetComponentMax = 80

var windowsReserved = func() map[string]bool {
	m := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}
	for i := 1; i < 10; i++ {
		m["COM"+string(rune('0'+i))] = true
		m["LPT"+string(rune('0'+i))] = true
	}
	return m
}()

func winInvalid(r rune) bool {
	if r <= 0x1f {
		return true
	}
	switch r {
	case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
		return true
	}
	return false
}

func containsWinInvalid(s string) bool {
	for _, r := range s {
		if winInvalid(r) {
			return true
		}
	}
	return false
}

func pyWordChar(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) ||
		unicode.In(r, unicode.Nl, unicode.No)
}

func Slugify(text string) string {
	s := pyStrip(pyLower(text))
	s = pyNFKD(s)

	var b strings.Builder
	for _, r := range s {
		if pyWordChar(r) || pyIsSpaceRe(r) || r == '-' {
			b.WriteRune(r)
		}
	}
	s = b.String()

	s = reSubRuns(s, func(r rune) bool { return pyIsSpaceRe(r) || r == '-' }, "_")

	s = reSubRuns(s, func(r rune) bool { return r == '_' }, "_")
	s = pyStripSet(s, "_")
	if s == "" {
		return "unnamed"
	}
	return s
}

func pyIsSpaceRe(r rune) bool { return pyIsSpace(r) }

func reSubRuns(s string, match func(rune) bool, repl string) string {
	var b strings.Builder
	inRun := false
	for _, r := range s {
		if match(r) {
			if !inRun {
				b.WriteString(repl)
				inRun = true
			}
		} else {
			b.WriteRune(r)
			inRun = false
		}
	}
	return b.String()
}

func sha256Hex10(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:10]
}

func assetStemUpper(component string) string {
	stem := component
	if i := strings.Index(component, "."); i >= 0 {
		stem = component[:i]
	}
	return pyUpper(stem)
}

func assetIsSafe(component string, maxLength int) bool {
	if component == "" || component == "." || component == ".." {
		return false
	}
	if strings.Contains(component, "..") {
		return false
	}
	if containsWinInvalid(component) {
		return false
	}
	if strings.HasSuffix(component, " ") || strings.HasSuffix(component, ".") {
		return false
	}
	if windowsReserved[assetStemUpper(component)] {
		return false
	}
	if len([]rune(component)) > maxLength {
		return false
	}
	return true
}

func assetComponent(value string, maxLength int) string {
	raw := pyStrip(value)
	digest := sha256Hex10(raw)

	if assetIsSafe(raw, maxLength) {
		return raw
	}
	if raw == "" {
		return "unnamed"
	}
	component := pyNFKC(raw)

	var b strings.Builder
	for _, r := range component {
		if winInvalid(r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	component = b.String()

	component = subDotRuns(component)

	component = reSubRuns(component, func(r rune) bool { return r == '_' }, "_")
	component = pyStripSet(component, " ._")
	if component == "" {
		component = "asset_" + digest
	}
	if windowsReserved[assetStemUpper(component)] {
		component = "asset_" + component + "_" + digest
	}
	if len([]rune(component)) > maxLength {
		rs := []rune(component)
		head := pyRStripSet(string(rs[:maxLength-11]), " ._")
		component = head + "_" + digest
	}
	if assetIsSafe(component, maxLength) {
		return component
	}
	return "asset_" + digest
}

func pyRStripSet(s, chars string) string {
	return strings.TrimRightFunc(s, func(r rune) bool { return strings.ContainsRune(chars, r) })
}

func subDotRuns(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); {
		if rs[i] == '.' {
			j := i
			for j < len(rs) && rs[j] == '.' {
				j++
			}
			if j-i >= 2 {
				b.WriteRune('_')
			} else {
				b.WriteRune('.')
			}
			i = j
		} else {
			b.WriteRune(rs[i])
			i++
		}
	}
	return b.String()
}

func assetCountryTag(value string) string {
	return pyUpper(assetComponent(value, 16))
}

func ExtractSlug(characterID, countryTag string) string {
	prefix := pyUpper(countryTag) + "_"
	if strings.HasPrefix(pyUpper(characterID), prefix) {
		rs := []rune(characterID)
		n := len([]rune(prefix))
		if n > len(rs) {
			n = len(rs)
		}
		return assetComponent(string(rs[n:]), assetComponentMax)
	}
	return assetComponent(Slugify(characterID), assetComponentMax)
}

func portraitFilename(countryTag, slug string) string {
	return "portrait_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax) + ".png"
}

func advisorIconFilename(countryTag, slug string) string {
	return "advisor_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax) + ".png"
}

func ExportPortraitPath(countryTag, slug string) string {
	return "gfx/leaders/" + pyUpper(countryTag) + "/" + portraitFilename(countryTag, slug)
}

func ExportAdvisorIconPath(countryTag, slug string) string {
	return "gfx/interface/ideas/" + advisorIconFilename(countryTag, slug)
}

func DecisionIconSpriteName(decisionID string) string {
	return "GFX_decision_" + Slugify(decisionID)
}

func DecisionCategoryIconSpriteName(categoryID string) string {
	return "GFX_decision_category_" + Slugify(categoryID)
}
