package exportengine

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

func VanillaYesNoKeysOf(raw []byte, known func(string) bool, keys map[string]bool) bool {
	block, _, err := paradox.Parse(strings.TrimPrefix(string(raw), "\uFEFF"))
	if err != nil || block == nil {
		return false
	}
	walkBlock(block, func(e *paradox.Entry) {
		if e.Key == nil || e.Block != nil {
			return
		}
		if rawScriptedPredicate(*e.Key, e.Scalar.StrValue(), known) {
			keys[*e.Key] = true
		}
	})
	return true
}

func rawScriptedPredicate(key, scalar string, known func(string) bool) bool {
	if key == "" {
		return false
	}
	if lower := strings.ToLower(scalar); lower != "yes" && lower != "no" {
		return false
	}
	if known(key) {
		return false
	}
	if scriptWrapKeys[key] {
		return false
	}
	if _, isCrossRef := crossRefSpecs[key]; isCrossRef {
		return false
	}
	return !strings.Contains(key, ".") && !pyIsUpper(key)
}
