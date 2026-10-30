package emitmisc

import (
	"strings"
	"unicode"
)

func letterRune(r rune) bool { return unicode.IsLetter(r) }
func digitRune(r rune) bool  { return unicode.IsDigit(r) || unicode.IsNumber(r) }

func isPyWordChar(r rune) bool {
	return r == '_' || isLetter(r) || isDigit(r)
}

func isLetter(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || r > 127 && letterRune(r)
}

func isDigit(r rune) bool {
	return ('0' <= r && r <= '9') || (r > 127 && digitRune(r))
}

func Slugify(text string) string {
	s := pyStrip(pyLower(text))

	var b strings.Builder
	for _, r := range s {
		if isPyWordChar(r) || pyIsSpaceRe(r) || r == '-' {
			b.WriteRune(r)
		}
	}
	s = b.String()

	var out strings.Builder
	inRun := false
	for _, r := range s {
		if pyIsSpaceRe(r) || r == '-' {
			if !inRun {
				out.WriteRune('_')
				inRun = true
			}
			continue
		}
		inRun = false
		out.WriteRune(r)
	}
	s = out.String()

	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if s == "" {
		return "unnamed"
	}
	return s
}

func pyIsSpaceRe(r rune) bool { return pyIsSpace(r) }

func DecisionIconSpriteName(decisionID string) string {
	return "GFX_decision_" + Slugify(decisionID)
}

func ExportDecisionIconPath(decisionID string) string {
	return "gfx/interface/decisions/decision_" + Slugify(decisionID) + ".dds"
}

func DecisionCategoryIconSpriteName(categoryID string) string {
	return "GFX_decision_category_" + Slugify(categoryID)
}

func ExportDecisionCategoryIconPath(categoryID string) string {
	return "gfx/interface/decisions/decision_category_" + Slugify(categoryID) + ".dds"
}

func FocusIconSpriteName(focusID string) string {
	return "GFX_focus_" + Slugify(focusID)
}

func ExportFocusIconPath(focusID string) string {
	return "gfx/interface/goals/focus_" + Slugify(focusID) + ".dds"
}

func TechIconSpriteName(techID string) string {
	return "GFX_" + Slugify(techID) + "_medium"
}

func ExportTechIconPath(techID string) string {
	return "gfx/interface/technologies/" + Slugify(techID) + ".dds"
}

func LibraryIconZipPath(gfxName string) string {
	base := gfxName
	if strings.HasPrefix(gfxName, "GFX_") {
		base = gfxName[4:]
	}
	return "gfx/interface/goals/" + base + ".dds"
}
