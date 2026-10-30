package emitscripts

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

var windowsReservedComponents = func() map[string]bool {
	m := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}
	for i := 1; i < 10; i++ {
		m["COM"+string(rune('0'+i))] = true
		m["LPT"+string(rune('0'+i))] = true
	}
	return m
}()

var windowsInvalidComponent = regexp.MustCompile("[\x00-\x1f<>:\"/\\\\|?*]")
var multiDotRe = regexp.MustCompile(`\.{2,}`)
var multiUnderscoreRe = regexp.MustCompile(`_+`)

const assetComponentMax = 80

func componentIsSafe(component string, maxLength int) bool {
	if component == "" || component == "." || component == ".." {
		return false
	}
	if strings.Contains(component, "..") {
		return false
	}
	if windowsInvalidComponent.MatchString(component) {
		return false
	}
	if strings.HasSuffix(component, " ") || strings.HasSuffix(component, ".") {
		return false
	}
	stem := pyUpper(strings.SplitN(component, ".", 2)[0])
	if windowsReservedComponents[stem] {
		return false
	}
	if len([]rune(component)) > maxLength {
		return false
	}
	return true
}

func assetComponent(value string, maxLength int) string {
	raw := pyStrip(value)
	sum := sha256.Sum256([]byte(raw))
	digest := hex.EncodeToString(sum[:])[:10]

	if componentIsSafe(raw, maxLength) {
		return raw
	}
	if raw == "" {
		return "unnamed"
	}
	component := raw
	component = windowsInvalidComponent.ReplaceAllString(component, "_")
	component = multiDotRe.ReplaceAllString(component, "_")
	component = multiUnderscoreRe.ReplaceAllString(component, "_")
	component = strings.Trim(component, " ._")
	if component == "" {
		component = "asset_" + digest
	}
	if windowsReservedComponents[pyUpper(strings.SplitN(component, ".", 2)[0])] {
		component = "asset_" + component + "_" + digest
	}
	if rc := []rune(component); len(rc) > maxLength {
		head := strings.TrimRight(string(rc[:maxLength-11]), " ._")
		component = head + "_" + digest
	}
	if componentIsSafe(component, maxLength) {
		return component
	}
	return "asset_" + digest
}

func assetCountryTag(value string) string {
	return pyUpper(assetComponent(value, 16))
}

func pyWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
}

func Slugify(text string) string {
	t := pyStrip(pyLower(text))

	var b1 strings.Builder
	for _, r := range t {
		if pyWordRune(r) || pyIsSpace(r) || r == '-' {
			b1.WriteRune(r)
		}
	}
	t = b1.String()

	var b2 strings.Builder
	inRun := false
	for _, r := range t {
		if pyIsSpace(r) || r == '-' {
			if !inRun {
				b2.WriteByte('_')
				inRun = true
			}
		} else {
			b2.WriteRune(r)
			inRun = false
		}
	}
	t = multiUnderscoreRe.ReplaceAllString(b2.String(), "_")
	t = strings.Trim(t, "_")
	if t == "" {
		return "unnamed"
	}
	return t
}

func PortraitFilename(countryTag, slug string) string {
	return "portrait_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax) + ".png"
}

func AdvisorIconFilename(countryTag, slug string) string {
	return "advisor_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax) + ".png"
}

func PortraitSpriteName(countryTag, slug string) string {
	return "GFX_portrait_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax)
}

func AdvisorSpriteName(countryTag, slug string) string {
	return "GFX_advisor_" + assetCountryTag(countryTag) + "_" + assetComponent(slug, assetComponentMax)
}

func IntelAgencyLogoSpriteName(agencyID string) string {
	return "GFX_intelligence_agency_logo_" + Slugify(agencyID)
}

func ExtractSlug(characterID, countryTag string) string {
	prefix := pyUpper(countryTag) + "_"
	upperID := pyUpper(characterID)
	if strings.HasPrefix(upperID, prefix) {

		rc := []rune(characterID)
		n := len([]rune(prefix))
		if n <= len(rc) {
			return assetComponent(string(rc[n:]), assetComponentMax)
		}
	}
	return assetComponent(Slugify(characterID), assetComponentMax)
}
