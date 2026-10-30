package locout

var langCodeOrder = []string{
	"en", "zh", "ru", "fr", "de", "es", "ja", "ko", "pl", "pt",
}

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

const LocReplaceSubdir = "replace"

func HOI4LangSubdir(langCode string) string {
	return LangCodeToHOI4[langCode] + "/" + LocReplaceSubdir
}

func HOI4LangSuffix(langCode string) string { return langHeader(langCode) }

func EnsureEnFirst(langs Value) []string {
	if !PyTruthy(langs) || langs.Kind != KArr {
		return []string{"en"}
	}
	out := []string{"en"}
	seen := map[string]bool{"en": true}
	for _, code := range langs.Arr {
		if code.Kind != KStr {
			continue
		}
		c := pyLower(pyStrip(code.Str))
		if c == "" || seen[c] {
			continue
		}
		if _, known := LangCodeToHOI4[c]; !known {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func EnsureEnFirstStrings(langs []string) []string {
	if len(langs) == 0 {
		return []string{"en"}
	}
	values := make([]Value, 0, len(langs))
	for _, lang := range langs {
		values = append(values, Str(lang))
	}
	return EnsureEnFirst(Value{Kind: KArr, Arr: values})
}

type LangTriple struct {
	Code   string
	Subdir string
	Suffix string
}

func BuildExportLangTriples(langs Value) []LangTriple {
	codes := EnsureEnFirst(langs)
	out := make([]LangTriple, 0, len(codes))
	for _, c := range codes {
		out = append(out, LangTriple{c, HOI4LangSubdir(c), HOI4LangSuffix(c)})
	}
	return out
}
