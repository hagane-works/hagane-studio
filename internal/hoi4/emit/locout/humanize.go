package locout

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func pyIsAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func HumanizeMIOToken(token string) string {
	s := pyStrip(token)

	if i := strings.Index(s, "_"); i >= 0 {
		head := s[:i]
		rest := s[i+1:]
		hl := utf8.RuneCountInString(head)
		if rest != "" && hl >= 2 && hl <= 4 && pyIsAlpha(head) {
			s = rest
		}
	}
	for _, noise := range []string{
		"mio_initial_trait_", "mio_trait_", "initial_trait_", "mio_", "generic_",
	} {
		s = strings.ReplaceAll(s, noise, "")
	}
	s = pyStrip(strings.ReplaceAll(s, "_", " "))
	if s != "" {
		return pyTitle(s)
	}
	return pyStrip(token)
}

func EffectiveIntelligenceAgencyNames(agency *Obj) []Value {
	authored := agency.Attr("names")
	if PyTruthy(authored) {
		switch authored.Kind {
		case KArr:
			return append([]Value{}, authored.Arr...)
		case KStr:
			out := make([]Value, 0, len(authored.Str))
			for _, r := range authored.Str {
				out = append(out, Str(string(r)))
			}
			return out
		}

		return nil
	}
	agencyID := agency.Attr("agency_id")
	if PyTruthy(agencyID) {
		return []Value{Str(PyStr(agencyID))}
	}
	return []Value{}
}
