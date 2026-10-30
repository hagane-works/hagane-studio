package workbench

import (
	"fmt"
	"sort"
	"strings"

	"haganeworks/internal/reftables"
)

func validateStartDate(raw string) (*string, error) {
	return validateHOI4Date(raw, "start_date")
}

func validateEndDate(raw string) (*string, error) {
	return validateHOI4Date(raw, "end_date")
}

func validateHOI4Date(raw, field string) (*string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, nil
	}
	if !dateRE.MatchString(v) {
		return nil, fmt.Errorf("%s must use HOI4 Y.M.D or Y.M.D.H format", field)
	}
	return &v, nil
}

const (
	ScenarioHistoryPolicyInheritBaseHistory       = "inherit_base_history"
	ScenarioHistoryPolicyProjectOwnsStartingWorld = "project_owns_starting_world"
)

func validateScenarioHistoryPolicy(raw string) error {
	if raw == ScenarioHistoryPolicyInheritBaseHistory || raw == ScenarioHistoryPolicyProjectOwnsStartingWorld {
		return nil
	}
	return fmt.Errorf("scenario_history_policy must be one of %q or %q", ScenarioHistoryPolicyInheritBaseHistory, ScenarioHistoryPolicyProjectOwnsStartingWorld)
}

func normalizeStageStateIDs(raw []int64) ([]int64, error) {
	ids := append([]int64(nil), raw...)
	for _, id := range ids {
		if id < 1 {
			return nil, fmt.Errorf("stage_state_ids entries must be HOI4 state ids >= 1 · got %d", id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := ids[:0]
	for _, id := range ids {
		if len(out) == 0 || out[len(out)-1] != id {
			out = append(out, id)
		}
	}
	if len(out) > 5000 {
		return nil, fmt.Errorf("stage_state_ids too large (%d unique ids · max 5000)", len(out))
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeExportLanguages(raw []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, entry := range raw {
		code := strings.ToLower(strings.TrimSpace(entry))
		if code == "" {
			continue
		}
		if !languages[code] {
			keys := []string{"de", "en", "es", "fr", "ja", "ko", "pl", "pt", "ru", "zh"}
			return nil, fmt.Errorf("Unknown HOI4 locale code %q · expected one of %v", entry, keys)
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	return out, nil
}

func normalizeReplacePaths(raw []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, entry := range raw {
		if strings.ContainsAny(entry, "\r\n\x00") {
			continue
		}
		v := strings.TrimSpace(strings.ReplaceAll(entry, "\\", "/"))
		for strings.Contains(v, "//") {
			v = strings.ReplaceAll(v, "//", "/")
		}
		for strings.HasPrefix(v, "./") {
			v = strings.TrimPrefix(v, "./")
		}
		v = strings.TrimRight(v, "/")
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func validateRegistry(registry map[string]any, deps []string) error {
	seen := map[string]bool{}
	roots := map[string]bool{}
	if xs, ok := registry["root_definitions"].([]any); ok {
		for _, x := range xs {
			m := x.(map[string]any)
			id, _ := m["id"].(string)
			body, _ := m["body"].(string)
			reason := validateRoot(id, body)
			if seen[id] {
				reason = "duplicate ROOT id"
			}
			seen[id] = true
			if reason != "" {
				return fmt.Errorf("Invalid custom ROOT ideology definition(s): %s: %s", id, reason)
			}
			roots[id] = true
		}
	}
	if len(deps) > 0 && len(roots) > 0 {
		return fmt.Errorf("Custom ROOT ideologies are currently standalone-only. A submod must inherit its base mod's ROOT groups or use an explicit Raw Files full replacement after reviewing replace_path safety.")
	}
	if len(deps) == 0 {
		vanilla := reftables.Ideologies()
		bad := []string{}
		seenSub := map[string]bool{}
		if xs, ok := registry["sub_ideologies"].([]any); ok {
			for _, x := range xs {
				m := x.(map[string]any)
				id, _ := m["id"].(string)
				if vanilla.IsSub(id) || seenSub[id] {
					continue
				}
				seenSub[id] = true
				group, _ := m["group"].(string)
				if !vanilla.IsGroup(group) && !roots[group] {
					bad = append(bad, fmt.Sprintf("%s (parent=%s)", id, map[bool]string{true: "missing", false: group}[group == ""]))
				}
			}
		}
		if len(bad) > 0 {
			groups := append([]string(nil), vanilla.Groups...)
			sort.Strings(groups)
			return fmt.Errorf("Standalone custom ideologies require a parent ROOT group from %v: %s", groups, strings.Join(bad, ", "))
		}
	}
	return nil
}
