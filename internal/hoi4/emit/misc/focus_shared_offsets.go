package emitmisc

import (
	"math"
	"math/big"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
)

var sharedOffsetTreeID = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func sharedOffsetInteger(raw any) (int64, bool) {
	switch value := raw.(type) {
	case nil:
		return 0, true
	case int:
		return int64(value), true
	case int64:
		return value, true
	case *big.Int:
		return value.Int64(), value.IsInt64()
	case float64:
		return int64(value), !math.IsNaN(value) && !math.IsInf(value, 0) && math.Trunc(value) == value && value >= -0x1p63 && value < 0x1p63
	default:
		return 0, false
	}
}

func CompileSharedReferenceOffsets(trees []*NS) ([]*NS, []exportmsg.Message) {
	definitions := map[string][]*NS{}
	treeCounts := map[string]int{}
	for _, tree := range trees {
		if tree.Str("tree_kind") == "shared_pool" {
			for _, raw := range tree.List("nodes") {
				if node, ok := raw.(*NS); ok {
					definitions[node.Str("focus_id")] = append(definitions[node.Str("focus_id")], node)
				}
			}
		} else {
			treeCounts[tree.Str("tree_id")]++
		}
	}
	additions := map[*NS][]any{}
	var warnings []exportmsg.Message
	for _, tree := range trees {
		if tree.Str("tree_kind") == "shared_pool" {
			continue
		}
		treeID := tree.Str("tree_id")
		seen := map[string]bool{}
		for _, raw := range tree.List("shared_focus_refs") {
			ref, ok := asOD(raw)
			if !ok {
				if id, scalar := raw.(string); scalar {
					seen[strings.TrimSpace(id)] = true
				}
				continue
			}
			id, _ := ref.GetDef("focus_id", "").(string)
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			x, xOK := sharedOffsetInteger(ref.GetDef("offset_x", int64(0)))
			y, yOK := sharedOffsetInteger(ref.GetDef("offset_y", int64(0)))
			if !xOK || !yOK {
				warnings = append(warnings, exportmsg.Message{Key: "focus.shared_offset_not_integer", Args: exportmsg.Args{"focus": id, "tree": treeID}})
				continue
			}
			if x == 0 && y == 0 {
				continue
			}
			if treeCounts[treeID] != 1 || !sharedOffsetTreeID.MatchString(treeID) || len(definitions[id]) != 1 {
				warnings = append(warnings, exportmsg.Message{Key: "focus.shared_offset_unresolved", Args: exportmsg.Args{"focus": id, "tree": treeID}})
				continue
			}
			node := definitions[id][0]
			additions[node] = append(additions[node], NewOD().Set("x", x).Set("y", y).Set("trigger", "has_focus_tree = "+treeID))
		}
	}
	if len(additions) == 0 {
		return trees, warnings
	}
	clone := func(n *NS) *NS {
		out := NewNS()
		for key, value := range n.m {
			out.Set(key, value)
		}
		return out
	}
	views := append([]*NS(nil), trees...)
	for i, tree := range trees {
		if tree.Str("tree_kind") != "shared_pool" {
			continue
		}
		nodes := append([]any(nil), tree.List("nodes")...)
		changed := false
		for j, raw := range nodes {
			node, ok := raw.(*NS)
			if !ok || len(additions[node]) == 0 {
				continue
			}
			view := clone(node)
			offsets := append([]any(nil), node.List("offset")...)
			view.Set("offset", append(offsets, additions[node]...))
			nodes[j], changed = view, true
		}
		if changed {
			views[i] = clone(tree).Set("nodes", nodes)
		}
	}
	return views, warnings
}
