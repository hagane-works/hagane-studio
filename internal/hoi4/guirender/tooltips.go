package guirender

import (
	"regexp"
	"strings"
)

var locRefRE = regexp.MustCompile(`\$([^$|\s]+)\$`)

func (idx *Index) LocaliseMarkup(key string) (string, bool) {
	value, ok := idx.locValue(key)
	if !ok {
		return "", false
	}
	return idx.resolveLocRefs(idx.resolveScripted(value, "", ""), 0, map[string]bool{key: true}), true
}

func (idx *Index) resolveLocRefs(value string, depth int, seen map[string]bool) string {
	if depth >= 5 || !strings.Contains(value, "$") {
		return value
	}
	return locRefRE.ReplaceAllStringFunc(value, func(match string) string {
		name := match[1 : len(match)-1]
		if seen[name] {
			return match
		}
		inner, ok := idx.locValue(name)
		if !ok {
			return match
		}
		seen[name] = true
		resolved := idx.resolveLocRefs(inner, depth+1, seen)
		delete(seen, name)
		return resolved
	})
}

func (idx *Index) tooltipText(key string) map[string]any {
	if value, ok := idx.LocaliseMarkup(key); ok {
		return map[string]any{"display": value, "kind": "loc"}
	}
	return map[string]any{"display": key, "kind": "key"}
}

func tooltipKeys(node map[string]any) []string {
	raw, _ := node["raw"].(map[string]any)
	if raw == nil {
		return nil
	}
	out := []string{}
	for _, field := range []string{"pdx_tooltip", "pdx_tooltip_delayed"} {
		if key := strings.Trim(strings.TrimSpace(toString(raw[field])), "\""); key != "" {
			out = append(out, key)
		}
	}
	return out
}
