package staticcatalog

import (
	_ "embed"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

//go:embed data/modifier_zh.yml
var bundledModifierYML []byte

type ModifierTranslation struct {
	NameZH string `json:"name_zh"`
	DescZH string `json:"desc_zh"`
}

var modifierLine = regexp.MustCompile(`^\s+(MODIFIER_[A-Z0-9_]+):\d+\s+"(.*)"\s*$`)

func ParseModifierTranslations(raw []byte) map[string]ModifierTranslation {
	text := strings.TrimPrefix(string(raw), "\ufeff")
	lines := strings.Split(text, "\n")
	rawValues := make(map[string]string)
	order := make([]string, 0)
	for _, line := range lines {
		match := modifierLine.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if match == nil {
			continue
		}
		if _, exists := rawValues[match[1]]; exists {
			continue
		}
		rawValues[match[1]] = match[2]
		order = append(order, match[1])
	}

	out := make(map[string]ModifierTranslation)
	for _, fullKey := range order {
		baseKey := fullKey
		isDescription := false
		if strings.HasSuffix(fullKey, "_DESC") {
			sibling := strings.TrimSuffix(fullKey, "_DESC")
			if sibling != "MODIFIER_" {
				if _, exists := rawValues[sibling]; exists {
					baseKey = sibling
					isDescription = true
				}
			}
		}
		modifierID := strings.ToLower(strings.TrimPrefix(baseKey, "MODIFIER_"))
		if modifierID == "" {
			continue
		}
		entry := out[modifierID]
		if isDescription {
			if entry.DescZH == "" {
				entry.DescZH = rawValues[fullKey]
			}
		} else if entry.NameZH == "" {
			entry.NameZH = rawValues[fullKey]
		}
		if entry.NameZH != "" || entry.DescZH != "" {
			out[modifierID] = entry
		}
	}
	return out
}

func LoadModifierTranslations(libraryRoot string) map[string]ModifierTranslation {
	if root := strings.TrimSpace(libraryRoot); root != "" {
		override := filepath.Join(root, "基础代码", "代码提词器", "modifiers", "r_modifiers_l_english.yml")
		if raw, err := os.ReadFile(override); err == nil {
			return ParseModifierTranslations(raw)
		}
	}
	return ParseModifierTranslations(bundledModifierYML)
}

var (
	defaultModifiersOnce sync.Once
	defaultModifiers     map[string]ModifierTranslation
)

func DefaultModifierTranslations() map[string]ModifierTranslation {
	defaultModifiersOnce.Do(func() {
		defaultModifiers = LoadModifierTranslations(os.Getenv("VSMOD_LIBRARY_ROOT"))
	})
	return defaultModifiers
}
