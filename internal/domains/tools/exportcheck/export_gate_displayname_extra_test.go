package exportcheck

import (
	"strings"
)

func DoctrineLocKey(name string) string {
	key := strings.TrimSpace(name)
	if key == "" || strings.Contains(key, " ") {
		return ""
	}
	return key
}

var displayNameKindSpecs = []struct {
	Kind            string
	Singular        string
	Plural          string
	LiteralAttr     string
	RowSourceScoped bool
}{

	{"focus", "focus", "focuses", "name", false},
	{"technology", "technology", "technologies", "name", true},
	{"idea", "idea", "ideas", "name", true},
	{"decision", "decision", "decisions", "", true},

	{"equipment", "equipment", "equipment", "name", true},

	{"custom_trait", "custom trait", "custom traits", "", false},
	{"dynamic_modifier", "dynamic modifier", "dynamic modifiers", "", true},
	{"autonomy_state", "autonomy state", "autonomy states", "", true},
	{"ui_panel", "UI panel", "UI panels", "", true},
	{"bookmark", "bookmark", "bookmarks", "", true},
	{"mio", "military industrial organization",
		"military industrial organizations", "", true},

	{"decision_category", "decision category", "decision categories", "", false},
	{"doctrine", "doctrine entry", "doctrine entries", "", true},
}

func DisplayNameKindSpec(kind string) (singular, plural, literalAttr string,
	scoped bool, ok bool) {
	for _, s := range displayNameKindSpecs {
		if s.Kind == kind {
			return s.Singular, s.Plural, s.LiteralAttr, s.RowSourceScoped, true
		}
	}
	return "", "", "", false, false
}
