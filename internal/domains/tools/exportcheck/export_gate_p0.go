package exportcheck

import (
	"sort"
)

type FocusPolicyRow struct {
	ID      int64
	TreeIDs []string
}

func PolicyRowsByTreeID(rows []FocusPolicyRow) map[string]FocusPolicyRow {
	ordered := append([]FocusPolicyRow(nil), rows...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	byTreeID := map[string]FocusPolicyRow{}
	for _, row := range ordered {
		for _, treeID := range row.TreeIDs {
			if _, exists := byTreeID[treeID]; !exists {
				byTreeID[treeID] = row
			}
		}
	}
	return byTreeID
}

func DuplicatePolicyTreeIDs(rows []FocusPolicyRow) map[string][]FocusPolicyRow {
	memberships := map[string][]FocusPolicyRow{}
	var order []string
	for _, row := range rows {
		for _, treeID := range row.TreeIDs {
			if _, seen := memberships[treeID]; !seen {
				order = append(order, treeID)
			}
			memberships[treeID] = append(memberships[treeID], row)
		}
	}
	out := map[string][]FocusPolicyRow{}
	for _, id := range order {
		if len(memberships[id]) > 1 {
			out[id] = memberships[id]
		}
	}
	return out
}

func ExpandFocusExportCohorts(selectedTreeIDs, allTreeIDs []string,
	policyRows []FocusPolicyRow) []string {

	selected := map[string]bool{}
	for _, id := range selectedTreeIDs {
		selected[id] = true
	}
	required := map[string]bool{}
	for id := range selected {
		required[id] = true
	}
	for _, policy := range policyRows {
		members := map[string]bool{}
		hit := false
		for _, id := range policy.TreeIDs {
			members[id] = true
			if selected[id] {
				hit = true
			}
		}
		if hit {
			for id := range members {
				required[id] = true
			}
		}
	}
	expanded := []string{}
	inExpanded := map[string]bool{}
	for _, id := range allTreeIDs {
		if required[id] && !inExpanded[id] {
			expanded = append(expanded, id)
			inExpanded[id] = true
		}
	}
	for _, id := range selectedTreeIDs {
		if !inExpanded[id] {
			expanded = append(expanded, id)
			inExpanded[id] = true
		}
	}
	return expanded
}

func P0FindingToExportIssue(finding map[string]any) map[string]any {
	code := asString(finding["code"])
	if code == "" {
		code = "p0_export_gate"
	}
	repairRoute := asString(finding["repair_route"])

	sourceDict := map[string]any{}
	if src, ok := finding["source"].(map[string]any); ok {
		for k, v := range src {
			sourceDict[k] = v
		}
	} else {
		id := code
		if s := asString(finding["source"]); s != "" {
			id = s
		}
		sourceDict = map[string]any{"kind": "focus_policy", "identifier": id}
	}
	evidenceDict := map[string]any{}
	if ev, ok := finding["evidence"].(map[string]any); ok {
		for k, v := range ev {
			evidenceDict[k] = v
		}
	}

	var entityRef map[string]any
	if ent, ok := sourceDict["entity"].(map[string]any); ok {
		entityRef = ent
	} else {
		kind := "export_gate"
		if k := asString(sourceDict["kind"]); k != "" {
			kind = k
		}
		var id any = code
		if v := sourceDict["identifier"]; v != nil && asString(v) != "" {
			id = v
		} else if v := sourceDict["file"]; v != nil && asString(v) != "" {
			id = v
		}
		entityRef = map[string]any{"type": kind, "id": id}
	}

	rawSeverity := asString(finding["severity"])
	level := "info"
	switch rawSeverity {
	case "blocker":
		level = "error"
	case "warning":
		level = "warning"
	}
	severity := rawSeverity
	if _, present := finding["severity"]; !present {
		severity = "warning"
	}
	return map[string]any{
		"level": level, "code": code, "severity": severity,
		"entity_ref": entityRef, "source": sourceDict,
		"evidence": evidenceDict, "repair_route": repairRoute,
		"detail": evidenceDict, "message": code, "fix_hint": repairRoute,
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func SafeRawPath(value string) string {
	safe, errMsg := SanitizeDestPath(value)
	if errMsg != "" {
		return ""
	}
	return safe
}
