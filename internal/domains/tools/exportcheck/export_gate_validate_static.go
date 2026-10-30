package exportcheck

import (
	"fmt"
	"sort"
	"strings"
)

type ValidateState struct {
	StateID   int64
	Name      string
	Owner     string
	Provinces []int64
}

type ValidateHistoryFile struct {
	CountryTag string
	Capital    int64

	SetPoliticsRulingParty string
	SetPopularities        map[string]int
	SetPopularitiesError   string
}

func validateCapitalNotOwned(states []ValidateState,
	historyFiles []ValidateHistoryFile) []map[string]any {

	issues := []map[string]any{}
	if len(states) == 0 {
		return issues
	}
	ownerByState := map[int64]string{}
	ownedTags := map[string]bool{}
	for _, st := range states {
		ownerByState[st.StateID] = st.Owner
		if st.Owner != "" {
			ownedTags[st.Owner] = true
		}
	}
	for _, hf := range historyFiles {
		cap := hf.Capital
		if cap == 0 {
			continue
		}
		owner, known := ownerByState[cap]
		if !known {
			continue
		}
		if !ownedTags[hf.CountryTag] {
			continue
		}
		if owner == hf.CountryTag {
			continue
		}
		issues = append(issues, map[string]any{
			"level": "warning", "code": "capital_not_owned",
			"entity_ref": map[string]any{
				"type": "history_file", "id": hf.CountryTag, "state_id": cap,
			},
			"severity": "warning",
			"fix_hint": fmt.Sprintf(
				"Set state %d's owner to %s, or set %s's capital to a state it actually owns.",
				cap, hf.CountryTag, hf.CountryTag),
			"message": fmt.Sprintf(
				"Country %s: capital state %d is owned by '%s', not %s — HOI4 will "+
					"center the bookmark camera wrong / log 'capital not owned'. Set the "+
					"capital state's owner to %s or pick a capital it owns.",
				hf.CountryTag, cap, owner, hf.CountryTag, hf.CountryTag),
		})
	}
	return issues
}

func validateProvinceDoubleMapping(states []ValidateState) []map[string]any {
	issues := []map[string]any{}
	if len(states) == 0 {
		return issues
	}
	nameByState := map[int64]string{}
	for _, st := range states {
		nameByState[st.StateID] = st.Name
	}
	provinceStates := map[int64][]int64{}
	var provinceOrder []int64
	for _, st := range states {
		for _, pid := range st.Provinces {
			if _, seen := provinceStates[pid]; !seen {
				provinceOrder = append(provinceOrder, pid)
			}
			provinceStates[pid] = append(provinceStates[pid], st.StateID)
		}
	}
	for _, pid := range provinceOrder {

		set := map[int64]bool{}
		for _, sid := range provinceStates[pid] {
			set[sid] = true
		}
		uniq := make([]int64, 0, len(set))
		for sid := range set {
			uniq = append(uniq, sid)
		}
		sort.Slice(uniq, func(i, j int) bool { return uniq[i] < uniq[j] })
		if len(uniq) < 2 {
			continue
		}
		parts := make([]string, 0, len(uniq))
		for _, s := range uniq {
			n := nameByState[s]
			if n == "" {
				n = "unnamed"
			}
			parts = append(parts, fmt.Sprintf("%d (%s)", s, n))
		}
		issues = append(issues, map[string]any{
			"level": "error", "code": "province_double_mapping",
			"entity_ref": map[string]any{
				"type": "state", "id": uniq[0],
				"province_id": pid, "state_ids": uniq,
			},
			"severity": "blocker",
			"fix_hint": fmt.Sprintf(
				"Remove province %d from every state except one (it currently appears in %d states).",
				pid, len(uniq)),
			"message": fmt.Sprintf(
				"Province %d is assigned to multiple states: %s — HOI4 will crash on load "+
					"('province in two states'). Remove it from all but one state.",
				pid, strings.Join(parts, ", ")),
		})
	}
	return issues
}

func validateTechCircularPrereq(techs []ValidateTech) []map[string]any {
	issues := []map[string]any{}
	byID := map[string]ValidateTech{}
	for _, t := range techs {
		byID[t.TechID] = t
	}
	type frame struct {
		id      string
		visited map[string]bool
	}
	for _, tech := range techs {
		stack := []frame{{tech.TechID, map[string]bool{tech.TechID: true}}}
		cycleFound := false
		for len(stack) > 0 {
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			cur, ok := byID[f.id]
			if !ok {
				continue
			}
			for _, p := range cur.Prerequisites {
				if f.visited[p] {
					issues = append(issues, map[string]any{
						"level": "warning", "code": "tech_circular_prereq",
						"entity_ref": map[string]any{"type": "technology", "id": tech.ID},
						"detail":     map[string]any{"tech_id": tech.TechID},
						"message": fmt.Sprintf(
							"Tech '%s': circular prerequisite dependency detected", tech.TechID),
					})
					cycleFound = true
					break
				}
				if _, known := byID[p]; known {
					next := make(map[string]bool, len(f.visited)+1)
					for k := range f.visited {
						next[k] = true
					}
					next[p] = true
					stack = append(stack, frame{p, next})
				}
			}
			if cycleFound {
				break
			}
		}
		if cycleFound {
			break
		}
	}
	return issues
}

func appendSuccessLine(issues []map[string]any) []map[string]any {
	for _, i := range issues {
		if lvl, _ := i["level"].(string); lvl == "error" || lvl == "warning" {
			return issues
		}
	}
	return append(issues, map[string]any{
		"level": "success", "message": "All checks passed — ready to export",
	})
}
