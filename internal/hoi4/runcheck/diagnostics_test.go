package runcheck

import "testing"

func TestCollectPlayerActionableEventAndFocusDiagnostics(t *testing.T) {
	findings := Collect(Snapshot{
		Events: []Event{
			{ID: 7, NamespaceID: 3, EventID: "other.12", EventNumber: 12, Namespace: "demo"},
			{ID: 8, NamespaceID: 3, EventID: "demo.14", EventNumber: 13, Namespace: "demo"},
		},
		FocusNodes: []FocusNode{
			{ID: 21, TreeID: 5, FocusID: "A", Prerequisites: [][]string{{"B"}}, RelativePosition: "missing_anchor"},
			{ID: 22, TreeID: 5, FocusID: "B", Prerequisites: [][]string{{"A", "missing_prereq"}}, MutuallyExclusive: []string{"missing_mutex"}},
			{ID: 23, TreeID: 6, FocusID: "B"},
		},
	})
	want := map[string]bool{
		"event_namespace_mismatch":          false,
		"event_number_mismatch":             false,
		"focus_relative_position_not_found": false,
		"focus_prereq_not_found":            false,
		"focus_mutex_not_found":             false,
		"focus_prereq_cycle":                false,
		"focus_id_duplicate":                false,
	}
	for _, finding := range findings {
		if _, ok := want[finding.Code]; ok {
			want[finding.Code] = true
		}
		if finding.Severity != "warning" {
			t.Fatalf("supplemental finding %s must stay advisory, got %q", finding.Code, finding.Severity)
		}
		if finding.Message != "" || finding.FixHint != "" {
			t.Fatalf("Go findings must not ship player copy: %#v", finding)
		}
	}
	for code, seen := range want {
		if !seen {
			t.Errorf("missing finding %s: %#v", code, findings)
		}
	}
}

func TestCollectCleanControlsStayQuiet(t *testing.T) {
	findings := Collect(Snapshot{
		Events: []Event{{ID: 1, NamespaceID: 2, EventID: "demo.7", EventNumber: 7, Namespace: "demo"}},
		FocusNodes: []FocusNode{
			{ID: 10, TreeID: 4, FocusID: "ROOT"},
			{ID: 11, TreeID: 4, FocusID: "CHILD", Prerequisites: [][]string{{"ROOT"}}, RelativePosition: "ROOT"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("valid event/focus snapshot produced findings: %#v", findings)
	}
}
