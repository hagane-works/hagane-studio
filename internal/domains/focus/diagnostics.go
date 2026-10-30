package focus

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/savediagnostics"
	"haganeworks/internal/project"
)

type SharedLayoutNode struct {
	FocusID            string
	RelativePositionID string
	X, Y               int
	Prerequisites      [][]string
	Offset             []map[string]any
}

func focusNodeAsLayoutNode(n FocusNode) LayoutNode {
	source := n.Source
	if source == "" {
		source = "project"
	}
	return LayoutNode{FocusID: n.FocusID, X: n.X, Y: n.Y, RelativePositionID: n.RelativePositionID, Source: source}
}

func focusLayoutSnapshot(nodes []FocusNode) []LayoutNode {
	out := make([]LayoutNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, focusNodeAsLayoutNode(n))
	}
	return out
}

func focusImportLayoutBaseline(nodes []FocusNode) []LayoutNode {
	var out []LayoutNode
	for _, n := range nodes {
		snap := n.sourceSnapshot
		if snap == nil {
			continue
		}
		if _, ok := snap["focus_id"]; !ok {
			continue
		}
		if _, ok := snap["x"]; !ok {
			continue
		}
		if _, ok := snap["y"]; !ok {
			continue
		}
		rel, _ := snap["relative_position_id"].(string)
		fid, _ := snap["focus_id"].(string)
		out = append(out, LayoutNode{
			FocusID: fid, X: coerceInt(snap["x"]), Y: coerceInt(snap["y"]),
			RelativePositionID: rel, Source: "import_snapshot",
		})
	}
	return out
}

func SharedLayoutProjections(tree FocusTree, nodes []SharedLayoutNode) []SharedProjectionPolicy {
	references := []map[string]any{}
	for _, item := range tree.SharedFocusRefs {
		if item == nil {
			continue
		}
		fid, _ := item["focus_id"].(string)
		if strings.TrimSpace(fid) != "" {
			references = append(references, item)
		}
	}
	if len(references) == 0 {
		return nil
	}
	byID := map[string]SharedLayoutNode{}
	for _, n := range nodes {
		if _, seen := byID[n.FocusID]; !seen {
			byID[n.FocusID] = n
		}
	}
	included := map[string]bool{}
	var pending []string
	for _, ref := range references {
		fid := strings.TrimSpace(ref["focus_id"].(string))
		if !included[fid] {
			included[fid] = true
			pending = append(pending, fid)
		}
	}
	for len(pending) > 0 {
		fid := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		node, ok := byID[fid]
		if !ok {
			continue
		}
		deps := map[string]bool{strings.TrimSpace(node.RelativePositionID): true}
		for _, group := range node.Prerequisites {
			for _, dep := range group {
				deps[strings.TrimSpace(dep)] = true
			}
		}
		for dep := range deps {
			if dep == "" {
				continue
			}
			if _, exists := byID[dep]; exists && !included[dep] {
				included[dep] = true
				pending = append(pending, dep)
			}
		}
	}
	refByID := func(fid string) map[string]any {
		for _, ref := range references {
			if strings.TrimSpace(ref["focus_id"].(string)) == fid {
				return ref
			}
		}
		return map[string]any{}
	}
	ids := make([]string, 0, len(included))
	for fid := range included {
		ids = append(ids, fid)
	}
	sort.Strings(ids)
	out := []SharedProjectionPolicy{}
	for _, fid := range ids {
		node, ok := byID[fid]
		if !ok {
			continue
		}
		dx, dy := FocusTreeConditionalOffset(node.Offset, tree.TreeID)
		ref := refByID(fid)
		condition, _ := ref["condition"].(string)
		out = append(out, SharedProjectionPolicy{
			FocusID: fid, BaseX: node.X + dx, BaseY: node.Y + dy,
			RelativePositionID: node.RelativePositionID,
			OffsetX:            coerceInt(ref["offset_x"]), OffsetY: coerceInt(ref["offset_y"]),
			Source: "shared_pool", Condition: condition,
		})
	}
	return out
}

func diagnosticStrings(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func focusSaveDiagnostics(tree FocusTree, nodes []FocusNode, shared []SharedLayoutNode) []SaveDiagnostic {
	projections := SharedLayoutProjections(tree, shared)
	baseline := focusImportLayoutBaseline(nodes)
	mode := "created_independent"
	if strings.TrimSpace(tree.SourceFile) != "" {
		mode = "replacement"
	}
	result := ValidateCandidateLayout(focusLayoutSnapshot(nodes), mode, projections, baseline,
		len(baseline) > 0, fmt.Sprintf("focus_layout:%s", tree.TreeID))
	byFocus := map[string][]int64{}
	for _, n := range nodes {
		byFocus[n.FocusID] = append(byFocus[n.FocusID], n.ID)
	}
	out := []SaveDiagnostic{}
	for _, f := range result.Findings {
		affected := diagnosticStrings(f.Evidence["focus_ids"])
		if id, ok := f.Evidence["focus_id"].(string); ok {
			affected = append(affected, id)
		}
		if cycle := diagnosticStrings(f.Evidence["cycle"]); len(cycle) > 0 {
			affected = append(affected, cycle...)
		}
		if pairs, ok := f.Evidence["pairs"].([]any); ok {
			for _, pair := range pairs {
				affected = append(affected, diagnosticStrings(pair)...)
			}
		}
		seen := map[int64]bool{}
		for _, focusID := range affected {
			for _, id := range byFocus[focusID] {
				if seen[id] {
					continue
				}
				seen[id] = true
				evidence, mine := focusNodeEvidence(f.Evidence, focusID)
				out = append(out, SaveDiagnostic{
					Key:  fmt.Sprintf("%s:%d:node:%d", f.Code, tree.ID, id),
					Code: f.Code, Severity: "warning",
					EntityRef: savediagnostics.EntityRef{Type: "focus_node", ID: strconv.FormatInt(id, 10)},
					Evidence:  evidence, RepairRoute: f.RepairRoute, Affected: mine, Source: f.Source,
				})
			}
		}
		if len(seen) == 0 {
			out = append(out, SaveDiagnostic{
				Key: fmt.Sprintf("%s:tree:%d", f.Code, tree.ID), Code: f.Code, Severity: "warning",
				EntityRef: savediagnostics.EntityRef{Type: "focus_tree", ID: strconv.FormatInt(tree.ID, 10)},
				Evidence:  f.Evidence, RepairRoute: f.RepairRoute, Affected: affected, Source: f.Source,
			})
		}
	}
	return savediagnostics.Current(out)
}

const focusEvidenceInlineIDs = 12

func focusNodeEvidence(evidence map[string]any, focusID string) (map[string]any, []string) {
	out := make(map[string]any, len(evidence))
	for key, value := range evidence {
		out[key] = value
	}
	affected := []string{focusID}
	if pairs, ok := evidence["pairs"].([]any); ok {
		mine := []any{}
		for _, pair := range pairs {
			ids := diagnosticStrings(pair)
			for _, id := range ids {
				if id == focusID {
					mine = append(mine, pair)
					affected = append(affected, ids...)
					break
				}
			}
		}
		out["pairs"] = mine
	}
	if ids := diagnosticStrings(evidence["focus_ids"]); len(ids) > focusEvidenceInlineIDs {
		out["focus_ids"] = []string{focusID}
	} else {
		affected = append(affected, ids...)
	}
	if id, ok := evidence["focus_id"].(string); ok {
		affected = append(affected, id)
	}
	affected = append(affected, diagnosticStrings(evidence["cycle"])...)
	return out, affected
}

func focusRowDiagnostics(all []SaveDiagnostic, n FocusNode) *[]SaveDiagnostic {
	out := []SaveDiagnostic{}
	nodeID, treeID := strconv.FormatInt(n.ID, 10), strconv.FormatInt(n.FocusTreeID, 10)
	for _, d := range all {
		if (d.EntityRef.Type == "focus_node" && d.EntityRef.ID == nodeID) ||
			(d.EntityRef.Type == "focus_tree" && d.EntityRef.ID == treeID) {
			out = append(out, d)
		}
	}
	return savediagnostics.Evaluated(out)
}

func currentTreeDiagnostics(diagnostics []SaveDiagnostic) *[]SaveDiagnostic {
	return savediagnostics.Evaluated(savediagnostics.Current(diagnostics))
}

func attachFocusDiagnostics(tree *FocusTree, shared []SharedLayoutNode) {
	all := focusSaveDiagnostics(*tree, tree.Nodes, shared)
	tree.Diagnostics = savediagnostics.Evaluated(all)
	for i := range tree.Nodes {
		tree.Nodes[i].Diagnostics = focusRowDiagnostics(all, tree.Nodes[i])
	}
}

func layoutNodeFromRow(row project.Row) FocusNode {
	n := FocusNode{
		ID: project.Int64(row["id"]), FocusTreeID: project.Int64(row["focus_tree_id"]),
		FocusID: colStr(row, "focus_id", ""), X: colInt(row, "x", 0), Y: colInt(row, "y", 0),
		RelativePositionID: colStr(row, "relative_position_id", ""), Source: colStr(row, "source", "created"),
	}
	decodeInto(row["source_snapshot"], &n.sourceSnapshot)
	return n
}

func treeDiagnostics(d treeData, treeRow FocusTree) []SaveDiagnostic {
	nodes := []FocusNode{}
	for _, row := range d.nodesOf(treeRow.ID) {
		nodes = append(nodes, layoutNodeFromRow(row))
	}
	return focusSaveDiagnostics(treeRow, nodes, d.sharedPoolNodes())
}
