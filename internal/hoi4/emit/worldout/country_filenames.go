package worldout

import "strings"

// HistoryCountryFilename names the history/countries file a country's history
// is written to. A history imported from a mod keeps the file name it had, so
// a mod that replaces a base-game file (`GRE - Greece.txt`) still replaces it;
// any other history goes to `TAG - z<mod>.txt`, which loads after the base
// game's file for the same tag.
func HistoryCountryFilename(tag, sourceFile, modDirName string) string {
	source := strings.ReplaceAll(strings.TrimSpace(sourceFile), "\\", "/")
	if dir, base, ok := strings.Cut(source, "history/countries/"); ok && (dir == "" || strings.HasSuffix(dir, "/")) {
		base = strings.TrimSpace(base)
		if !strings.Contains(base, "/") && !stateFnIllegalRe.MatchString(base) &&
			strings.HasSuffix(strings.ToLower(base), ".txt") && strings.Trim(base, ". ") != "" &&
			strings.EqualFold(historyFilenameTag(base), tag) {
			return base
		}
	}
	return tag + " - z" + modDirName + ".txt"
}

func historyFilenameTag(base string) string {
	tag, _, ok := strings.Cut(base, "-")
	if !ok {
		return ""
	}
	return strings.TrimSpace(tag)
}

// CountryDefinitionFilename names the common/countries file a country's
// definition is written to: the file its mod's country_tags named, when that is
// a plain file in common/countries, else `TAG.txt`.
func CountryDefinitionFilename(tag, sourceFile string) string {
	source := strings.ReplaceAll(strings.TrimSpace(sourceFile), "\\", "/")
	if len(source) > len("common/countries/") && strings.EqualFold(source[:len("common/countries/")], "common/countries/") {
		base := source[len("common/countries/"):]
		lower := strings.ToLower(base)
		if !strings.Contains(base, "/") && !stateFnIllegalRe.MatchString(base) && strings.HasSuffix(lower, ".txt") &&
			strings.Trim(base, ". ") != "" && lower != "colors.txt" && lower != "cosmetic.txt" {
			return base
		}
	}
	return tag + ".txt"
}
