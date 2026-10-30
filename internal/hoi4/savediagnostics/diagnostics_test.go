package savediagnostics_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/savediagnostics"
)

func finding(key string, affected []string, evidence map[string]any) savediagnostics.Diagnostic {
	return savediagnostics.Diagnostic{
		Key: key, Code: "bookmark.default_multiple", Severity: "warning",
		EntityRef: savediagnostics.EntityRef{Type: "project", ID: "17"},
		Evidence:  evidence, RepairRoute: "bookmark_editor", Affected: affected,
	}
}

func assertChange(t *testing.T, before, current []savediagnostics.Diagnostic, want savediagnostics.Change) savediagnostics.Diagnostic {
	t.Helper()
	got := savediagnostics.Compare(before, current)
	if len(got) != 1 || got[0].Change != want {
		t.Fatalf("Compare = %#v; want exactly one %s", got, want)
	}
	return got[0]
}

func TestEnvelopePreservesUnevaluatedVersusEvaluatedEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   savediagnostics.Envelope
		want string
	}{
		{"absent", savediagnostics.Envelope{}, `{}`},
		{"evaluated empty", savediagnostics.Envelope{Diagnostics: savediagnostics.Evaluated(nil)}, `{"diagnostics":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil || string(got) != tc.want {
				t.Fatalf("Marshal = %s, %v; want %s", got, err, tc.want)
			}
		})
	}
	if got := savediagnostics.Compare(nil, nil); got == nil || len(got) != 0 {
		t.Fatalf("evaluated empty comparison must be nonnil: %#v", got)
	}
}

func TestDefaultCardinalityTransitions(t *testing.T) {
	two := finding("bookmark.default_multiple/project/17", []string{"A", "B"}, map[string]any{"default_count": 2, "default_bookmarks": []string{"A", "B"}})
	three := finding(two.Key, []string{"A", "B", "C"}, map[string]any{"default_count": 3, "default_bookmarks": []string{"A", "B", "C"}})
	assertChange(t, []savediagnostics.Diagnostic{two}, []savediagnostics.Diagnostic{three}, savediagnostics.Worsened)
	assertChange(t, []savediagnostics.Diagnostic{three}, []savediagnostics.Diagnostic{two}, savediagnostics.Improved)

	assertChange(t, []savediagnostics.Diagnostic{two}, nil, savediagnostics.Resolved)
	assertChange(t, nil, []savediagnostics.Diagnostic{two}, savediagnostics.Introduced)
}

func TestSameNodeChangingCollisionPartnerIsChanged(t *testing.T) {
	before := finding("focus_layout_same_cell/tree/T/node/B", []string{"A", "B"}, map[string]any{"focus_ids": []string{"A", "B"}, "position": []int{0, 0}})
	current := finding(before.Key, []string{"B", "C"}, map[string]any{"focus_ids": []string{"B", "C"}, "position": []int{0, 0}})
	assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Changed)
}

func TestSetAndMapOrderDoesNotInventChanges(t *testing.T) {
	before := finding("collision/B", []string{"B", "A", "A"}, map[string]any{
		"focus_ids": []string{"B", "A", "A"}, "bookmark_ids": []string{"b", "a"},
		"default_bookmarks": []string{"B", "A"}, "paths": []string{"z", "a"},
		"interesting_countries": []string{"GER", "ENG"},
		"position":              []int{0, 1}, "nested": map[string]any{"y": 2, "x": 1},
	})
	current := finding(before.Key, []string{"A", "B"}, map[string]any{
		"nested":                map[string]any{"x": json.Number("1.0"), "y": json.Number("2e0")},
		"position":              []any{json.Number("0"), json.Number("1")},
		"interesting_countries": []string{"ENG", "GER"},
		"paths":                 []string{"a", "z"}, "default_bookmarks": []string{"A", "B"},
		"bookmark_ids": []string{"a", "b"}, "focus_ids": []string{"A", "B"},
	})
	got := assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Unchanged)
	if !reflect.DeepEqual(got.Affected, []string{"A", "B"}) {
		t.Fatalf("affected must be normalized set: %#v", got.Affected)
	}
}

func TestCoordinateCycleAndUnknownArraysKeepOrder(t *testing.T) {
	for _, field := range []string{"position", "translation", "cycle", "unknown_sequence"} {
		t.Run(field, func(t *testing.T) {
			before := finding("layout/B", []string{"B"}, map[string]any{field: []int{0, 1}})
			current := finding(before.Key, before.Affected, map[string]any{field: []int{1, 0}})
			assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Changed)
		})
	}

	before := finding("layout/B", []string{"B"}, map[string]any{"focus_ids": []any{[]int{0, 1}}})
	current := finding(before.Key, before.Affected, map[string]any{"focus_ids": []any{[]int{1, 0}}})
	assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Changed)
}

func TestNumericSetMembersNormalizeWithoutFloatRounding(t *testing.T) {
	before := finding("set", nil, map[string]any{"focus_ids": []any{json.Number("1"), json.Number("10"), json.Number("1e0"), json.Number("1.0")}})
	current := finding(before.Key, nil, map[string]any{"focus_ids": []any{10, 1}})
	assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Unchanged)
	before.Evidence = map[string]any{"count": json.Number("9007199254740992")}
	current.Evidence = map[string]any{"count": json.Number("9007199254740993")}
	assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Changed)
}

func TestCurrentAndResolvedDeduplicateInStableKeyOrder(t *testing.T) {
	active := finding("b", []string{"B"}, nil)
	active.Change = savediagnostics.Introduced
	resolved := active
	resolved.Change = savediagnostics.Resolved
	resolved.Evidence = map[string]any{"old": true}
	other := finding("a", []string{"A"}, nil)
	input := []savediagnostics.Diagnostic{resolved, active, other, active, resolved, other}
	got := savediagnostics.Current(input)
	if len(got) != 2 || got[0].Key != "a" || got[1].Key != "b" || got[1].Change != "" || len(got[1].Evidence) != 0 {
		t.Fatalf("Current mixed resolved or failed to dedup: %#v", got)
	}
	result := *savediagnostics.Evaluated(input)
	if len(result) != 2 || result[1].Change != savediagnostics.Introduced {
		t.Fatalf("mutation must prefer active over resolved: %#v", result)
	}
	for left, right := 0, len(input)-1; left < right; left, right = left+1, right-1 {
		input[left], input[right] = input[right], input[left]
	}
	if reversed := *savediagnostics.Evaluated(input); !reflect.DeepEqual(result, reversed) {
		t.Fatalf("arrival order changed dedup result: %#v vs %#v", result, reversed)
	}
	changes := savediagnostics.Compare(input, nil)
	if len(changes) != 2 || changes[0].Key != "a" || changes[1].Key != "b" || changes[0].Change != savediagnostics.Resolved || changes[1].Change != savediagnostics.Resolved {
		t.Fatalf("each old active key resolves once: %#v", changes)
	}
	if repeated := savediagnostics.Compare(changes, nil); len(repeated) != 0 {
		t.Fatalf("already resolved findings reappeared: %#v", repeated)
	}
}

func TestConflictingDuplicatePayloadChoiceIsDeterministic(t *testing.T) {
	first := finding("b", []string{"B"}, map[string]any{"value": "first"})
	second := finding("b", []string{"C"}, map[string]any{"value": "second"})
	one := savediagnostics.Current([]savediagnostics.Diagnostic{first, second})
	two := savediagnostics.Current([]savediagnostics.Diagnostic{second, first})
	if len(one) != 1 || !reflect.DeepEqual(one, two) {
		t.Fatalf("duplicate producer key should have stable representative: %#v vs %#v", one, two)
	}
}

func TestCurrentPayloadWinsOverBeforeAndKeyIsAuthoritative(t *testing.T) {
	before := finding("old", []string{"B"}, map[string]any{"problem": "old"})
	current := finding(before.Key, before.Affected, map[string]any{"problem": "current"})
	got := assertChange(t, []savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current}, savediagnostics.Changed)
	if got.Evidence["problem"] != "current" {
		t.Fatalf("current payload must win: %#v", got)
	}
	current.Key = "new"
	changes := savediagnostics.Compare([]savediagnostics.Diagnostic{before}, []savediagnostics.Diagnostic{current})
	if len(changes) != 2 || changes[0].Key != "new" || changes[0].Change != savediagnostics.Introduced || changes[1].Key != "old" || changes[1].Change != savediagnostics.Resolved {
		t.Fatalf("producer keys must determine identity: %#v", changes)
	}
}

func TestCompareDoesNotMutateSnapshotsOrShareJSONContainers(t *testing.T) {
	input := finding("b", []string{"B", "A"}, map[string]any{"nested": map[string]any{"value": "original"}, "focus_ids": []string{"B", "A"}})
	input.Source = map[string]any{"route": "editor"}
	input.Change = savediagnostics.Worsened
	got := savediagnostics.Compare(nil, []savediagnostics.Diagnostic{input})
	got[0].Affected[0] = "mutated"
	got[0].Evidence["nested"].(map[string]any)["value"] = "mutated"
	got[0].Source.(map[string]any)["route"] = "mutated"
	if !reflect.DeepEqual(input.Affected, []string{"B", "A"}) || input.Evidence["nested"].(map[string]any)["value"] != "original" || input.Source.(map[string]any)["route"] != "editor" || input.Change != savediagnostics.Worsened {
		t.Fatalf("comparison mutated source snapshot: %#v", input)
	}
}

func TestInvalidJSONEvidenceRemainsVisibleToEncoder(t *testing.T) {
	input := finding("b", nil, map[string]any{"invalid": make(chan int)})
	_, err := json.Marshal(savediagnostics.Envelope{Diagnostics: savediagnostics.Evaluated([]savediagnostics.Diagnostic{input})})
	if err == nil {
		t.Fatal("invalid evidence must not silently become empty diagnostics")
	}
}
