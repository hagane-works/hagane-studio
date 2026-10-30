package worldout

var langCodeToHoi4 = map[string]string{
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

const locReplaceSubdir = "replace"

type LangTriple struct {
	Code   string
	Subdir string
	Suffix string
}

func ensureEnFirst(langs []string) []string {
	if len(langs) == 0 {
		return []string{"en"}
	}
	out := []string{"en"}
	seen := map[string]bool{"en": true}
	for _, code := range langs {
		c := lowerStrip(code)
		if c == "" || seen[c] {
			continue
		}
		if _, ok := langCodeToHoi4[c]; !ok {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func lowerStrip(s string) string {
	return toLower(pyStrip(s))
}

func buildExportLangTriples(langs []string) []LangTriple {
	var out []LangTriple
	for _, code := range ensureEnFirst(langs) {
		hoi4 := langCodeToHoi4[code]
		out = append(out, LangTriple{
			Code:   code,
			Subdir: hoi4 + "/" + locReplaceSubdir,
			Suffix: "l_" + hoi4,
		})
	}
	return out
}
