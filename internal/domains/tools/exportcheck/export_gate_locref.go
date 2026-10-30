package exportcheck

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var locAssignmentRe = regexp.MustCompile(
	`(?i)(?:custom_effect_tooltip|custom_trigger_tooltip|tooltip|` +
		`localization_key|localisation_key)` +
		`\s*=\s*(?:"([^"]+)"|([A-Za-z0-9_.-]+))`)

var locKeyRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

var locNonKeys = map[string]bool{
	"yes": true, "no": true, "true": true, "false": true, "none": true,
}

func isLocIdentEdge(r byte) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-'
}

type ScriptLocSource struct {
	EntityType string
	EntityID   string
	Field      string
	Value      any
}

func stripClausewitzComment(line string) string {
	quoted, escaped := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && quoted {
			escaped = true
			continue
		}
		if c == '"' {
			quoted = !quoted
			continue
		}
		if c == '#' && !quoted {
			return line[:i]
		}
	}
	return line
}

func leafStrings(value any, out *[]string) {
	switch v := value.(type) {
	case string:
		*out = append(*out, v)
	case map[string]any:

		for _, child := range v {
			leafStrings(child, out)
		}
	case []any:
		for _, child := range v {
			leafStrings(child, out)
		}
	}
}

func ExtractLocalisationReferences(value any) map[string]bool {
	found := map[string]bool{}
	var texts []string
	leafStrings(value, &texts)
	for _, text := range texts {
		lines := strings.Split(text, "\n")
		for i, l := range lines {
			lines[i] = stripClausewitzComment(strings.TrimRight(l, "\r"))
		}
		cleaned := strings.Join(lines, "\n")
		for _, loc := range locAssignmentRe.FindAllStringSubmatchIndex(cleaned, -1) {

			if loc[0] > 0 && isLocIdentEdge(cleaned[loc[0]-1]) {
				continue
			}
			key := ""
			if loc[2] >= 0 {
				key = cleaned[loc[2]:loc[3]]
			} else if loc[4] >= 0 {
				key = cleaned[loc[4]:loc[5]]
			}
			key = strings.TrimSpace(key)
			if key == "" || !locKeyRe.MatchString(key) ||
				locNonKeys[strings.ToLower(key)] || isAllDigits(key) {
				continue
			}
			found[key] = true
		}
	}
	return found
}

func ScanMissingLocalisationReferences(sources []ScriptLocSource,
	definedKeys map[string]bool) []map[string]any {

	type identity struct{ etype, eid, key string }
	missing := map[identity]map[string]any{}
	var order []identity

	for _, src := range sources {
		keys := ExtractLocalisationReferences(src.Value)

		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, key := range sorted {
			if definedKeys[key] {
				continue
			}
			id := identity{src.EntityType, src.EntityID, key}
			if _, exists := missing[id]; exists {
				continue
			}
			order = append(order, id)
			missing[id] = map[string]any{
				"level": "warning", "code": "localisation_reference_not_in_project",
				"entity_ref": map[string]any{
					"type": src.EntityType, "id": src.EntityID,
					"loc_key": key, "field": src.Field,
				},
				"detail": map[string]any{
					"id": src.EntityID, "key": key, "field": src.Field,
				},
				"severity": "warning",
				"message": fmt.Sprintf(
					"%s '%s' references localisation key '%s' in %s, but this project has "+
						"no definition for it. It may still come from vanilla or the "+
						"selected base mod.",
					src.EntityType, src.EntityID, key, src.Field),
				"fix_hint": fmt.Sprintf(
					"Add '%s' to the project's localisation, or confirm that the key is "+
						"supplied by vanilla/the base mod before ignoring this warning.",
					key),
			}
		}
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].etype != order[j].etype {
			return order[i].etype < order[j].etype
		}
		if order[i].eid != order[j].eid {
			return order[i].eid < order[j].eid
		}
		return order[i].key < order[j].key
	})
	out := make([]map[string]any, 0, len(order))
	for _, id := range order {
		out = append(out, missing[id])
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
