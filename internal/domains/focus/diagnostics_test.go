package focus

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"
)

func TestFocusDiagnosticsStayLinearOnAPackedTree(t *testing.T) {
	const rows, cols = 30, 40
	tree := FocusTree{ID: 9, TreeID: "qa_packed"}
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			id := int64(len(tree.Nodes) + 1)
			tree.Nodes = append(tree.Nodes, FocusNode{
				ID: id, FocusTreeID: tree.ID, FocusID: fmt.Sprintf("qa_%02d_%02d", y, x), X: x, Y: y,
			})
		}
	}
	attachFocusDiagnostics(&tree, nil)
	body, err := json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}

	if len(body) > 3<<20 {
		t.Fatalf("tree response is %d bytes", len(body))
	}

	middle := tree.Nodes[5*cols+7]
	var found bool
	for _, d := range *middle.Diagnostics {
		if d.Code != "focus_layout_adjacent_same_row" {
			continue
		}
		found = true
		pairs, _ := d.Evidence["pairs"].([]any)
		if len(pairs) != 2 {
			t.Errorf("qa_05_07 carries %d pairs, want its own 2: %v", len(pairs), pairs)
		}
		affected := append([]string(nil), d.Affected...)
		sort.Strings(affected)
		want := []string{"qa_05_06", "qa_05_07", "qa_05_08"}
		if fmt.Sprint(affected) != fmt.Sprint(want) {
			t.Errorf("qa_05_07 affected %v, want %v", affected, want)
		}
	}
	if !found {
		t.Fatal("qa_05_07 has no adjacent-row warning")
	}
	if got := len(*tree.Diagnostics); got != rows*cols {
		t.Errorf("tree diagnostics %d, want one per focus (%d)", got, rows*cols)
	}
}

func TestFocusNodeEvidenceKeepsShortListsWhole(t *testing.T) {
	long := make([]string, 40)
	for i := range long {
		long[i] = fmt.Sprintf("qa_%02d", i)
	}
	evidence, affected := focusNodeEvidence(map[string]any{"focus_ids": long, "translation": []int{1, 0}}, "qa_03")
	if fmt.Sprint(evidence["focus_ids"]) != "[qa_03]" || fmt.Sprint(affected) != "[qa_03]" {
		t.Errorf("shift evidence %v affected %v", evidence["focus_ids"], affected)
	}
	if fmt.Sprint(evidence["translation"]) != "[1 0]" {
		t.Errorf("translation lost: %v", evidence["translation"])
	}
	evidence, affected = focusNodeEvidence(map[string]any{"focus_ids": []string{"qa_a", "qa_b"}, "position": []int{2, 3}}, "qa_a")
	if fmt.Sprint(evidence["focus_ids"]) != "[qa_a qa_b]" || len(affected) != 3 {
		t.Errorf("same-cell evidence %v affected %v", evidence["focus_ids"], affected)
	}
}
