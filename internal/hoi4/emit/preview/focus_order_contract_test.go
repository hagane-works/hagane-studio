package preview

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc"
)

func TestFocusOrderContractPreview(t *testing.T) {
	for _, kind := range []string{"primary", "shared_pool"} {
		t.Run(kind, func(t *testing.T) {
			tree := BuildFocusTree(&importsvc.TreeBundle{
				TreeData: map[string]any{"tree_id": "contract_tree", "country_tag": "GER"}, TreeKind: kind,
				NodesData: []map[string]any{
					{"focus_id": "ORDER_child", "relative_position_id": "ORDER_anchor", "x": -3, "y": 5, "completion_reward": "add_stability = 0.1"},
					{"focus_id": "ORDER_independent", "x": 9, "y": 1},
					{"focus_id": "ORDER_anchor", "x": 6, "y": 2},
				},
			})

			tree.Nodes[0], tree.Nodes[2] = tree.Nodes[2], tree.Nodes[0]
			script := GenerateFocusTreeScript(tree, nil)
			matches := regexp.MustCompile(`(?m)^\s*id = (ORDER_\w+)\s*$`).FindAllStringSubmatch(script, -1)
			var order []string
			for _, match := range matches {
				order = append(order, match[1])
			}
			want := []string{"ORDER_independent", "ORDER_anchor", "ORDER_child"}
			if !reflect.DeepEqual(order, want) {
				t.Fatalf("emitter dependency order=%v, want %v", order, want)
			}
			for _, field := range []string{"relative_position_id = ORDER_anchor", "x = -3", "y = 5", "add_stability = 0.1"} {
				if !strings.Contains(script, field) {
					t.Errorf("author field missing: %s", field)
				}
			}
		})
	}
}

func TestFocusOrderContractPreviewFallback(t *testing.T) {
	for _, kind := range []string{"primary", "shared_pool"} {
		t.Run(kind, func(t *testing.T) {
			tree := BuildFocusTree(&importsvc.TreeBundle{
				TreeData: map[string]any{"tree_id": "contract_tree", "country_tag": "GER"}, TreeKind: kind,
				NodesData: []map[string]any{
					{"focus_id": "ORDER_cycle_a", "relative_position_id": "ORDER_cycle_b"},
					{"focus_id": "ORDER_leaf", "relative_position_id": "ORDER_root"},
					{"focus_id": "ORDER_cycle_b", "relative_position_id": "ORDER_cycle_a"},
					{"focus_id": "ORDER_root"},
				},
			})
			script := GenerateFocusTreeScript(tree, nil)
			matches := regexp.MustCompile(`(?m)^\s*id = (ORDER_\w+)\s*$`).FindAllStringSubmatch(script, -1)
			var order []string
			for _, match := range matches {
				order = append(order, match[1])
			}
			want := []string{"ORDER_cycle_a", "ORDER_cycle_b", "ORDER_root", "ORDER_leaf"}
			if !reflect.DeepEqual(order, want) {
				t.Fatalf("fallback order=%v, want %v", order, want)
			}
			for _, ref := range []string{"relative_position_id = ORDER_cycle_a", "relative_position_id = ORDER_cycle_b", "relative_position_id = ORDER_root"} {
				if !strings.Contains(script, ref) {
					t.Errorf("reference removed: %s", ref)
				}
			}
		})
	}
}
