package focusorder

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
)

func TestPlanOrderAndFallback(t *testing.T) {
	tests := []struct {
		name        string
		nodes       []Node
		want        []int
		diagnostics []Diagnostic
	}{
		{name: "empty", want: []int{}, diagnostics: []Diagnostic{}},
		{name: "stable candidates not every unrelated pair", nodes: []Node{{"A", "C"}, {"B", ""}, {"C", ""}}, want: []int{1, 2, 0}, diagnostics: []Diagnostic{}},
		{name: "valid existing order", nodes: []Node{{"root", ""}, {"a", "root"}, {"b", "root"}, {"end", "a"}}, want: []int{0, 1, 2, 3}, diagnostics: []Diagnostic{}},
		{name: "reverse chain", nodes: []Node{{"c", "b"}, {"b", "a"}, {"a", ""}}, want: []int{2, 1, 0}, diagnostics: []Diagnostic{}},
		{name: "missing anchor with incoming dependents", nodes: []Node{{"child", "orphan"}, {"orphan", "external"}, {"late", "root"}, {"root", ""}}, want: []int{0, 1, 3, 2}, diagnostics: []Diagnostic{{Code: "focus_layout_missing_anchor", NodeIndices: []int{1}, AffectedIndices: []int{0, 1}, FocusID: "orphan", RelativePositionID: "external"}}},
		{name: "cycle interleaved with healthy region", nodes: []Node{{"a", "b"}, {"leaf", "root"}, {"tail", "a"}, {"root", ""}, {"b", "a"}}, want: []int{0, 2, 3, 1, 4}, diagnostics: []Diagnostic{{Code: "focus_layout_relative_cycle", NodeIndices: []int{0, 4}, AffectedIndices: []int{0, 2, 4}}}},
		{name: "duplicate identity and incoming reference", nodes: []Node{{"child", "dup"}, {"late", "root"}, {"dup", ""}, {"root", ""}, {"dup", ""}}, want: []int{0, 2, 3, 1, 4}, diagnostics: []Diagnostic{{Code: "focus_layout_duplicate_focus_id", NodeIndices: []int{2, 4}, AffectedIndices: []int{0, 2, 4}, FocusID: "dup"}}},
		{name: "self cycle", nodes: []Node{{"self", "self"}}, want: []int{0}, diagnostics: []Diagnostic{{Code: "focus_layout_relative_cycle", NodeIndices: []int{0}, AffectedIndices: []int{0}}}},
		{name: "empty identity preserves whole region", nodes: []Node{{"", "anchor"}, {"leaf", "root"}, {"anchor", ""}, {"root", ""}}, want: []int{0, 2, 3, 1}, diagnostics: []Diagnostic{{Code: "focus_layout_missing_focus_id", NodeIndices: []int{0}, AffectedIndices: []int{0, 2}}}},
		{name: "multiple independent cycles", nodes: []Node{{"a", "b"}, {"x", "y"}, {"b", "a"}, {"y", "x"}}, want: []int{0, 1, 2, 3}, diagnostics: []Diagnostic{{Code: "focus_layout_relative_cycle", NodeIndices: []int{0, 2}, AffectedIndices: []int{0, 2}}, {Code: "focus_layout_relative_cycle", NodeIndices: []int{1, 3}, AffectedIndices: []int{1, 3}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := append([]Node(nil), tt.nodes...)
			got := Plan(tt.nodes)
			if !reflect.DeepEqual(got.Order, tt.want) || !reflect.DeepEqual(got.Diagnostics, tt.diagnostics) {
				t.Fatalf("Plan=%+v, want order=%v diagnostics=%+v", got, tt.want, tt.diagnostics)
			}
			if !reflect.DeepEqual(tt.nodes, original) {
				t.Fatal("Plan mutated input")
			}
			assertPlanProperties(t, tt.nodes, got)
		})
	}
}

func assertPlanProperties(t *testing.T, nodes []Node, got Result) {
	t.Helper()
	if len(got.Order) != len(nodes) {
		t.Fatalf("lost nodes: %d inputs, %v", len(nodes), got.Order)
	}
	seen := make(map[int]bool, len(nodes))
	reordered := make([]Node, len(nodes))
	for out, in := range got.Order {
		if in < 0 || in >= len(nodes) || seen[in] {
			t.Fatalf("not a permutation: %v", got.Order)
		}
		seen[in] = true
		reordered[out] = nodes[in]
	}
	if again := Plan(nodes); !reflect.DeepEqual(got, again) {
		t.Fatal("Plan is not deterministic")
	}
	idempotent := Plan(reordered)
	for out, in := range idempotent.Order {
		if out != in {
			t.Fatalf("not idempotent: first=%v second=%v", got.Order, idempotent.Order)
		}
	}
	positions := make(map[int]int, len(nodes))
	for out, in := range got.Order {
		positions[in] = out
	}
	for _, d := range got.Diagnostics {
		for i := 1; i < len(d.AffectedIndices); i++ {
			if positions[d.AffectedIndices[i-1]] >= positions[d.AffectedIndices[i]] {
				t.Fatalf("fallback region reordered: diagnostic=%+v order=%v", d, got.Order)
			}
		}
	}
}

func TestPlanPermutedDAGs(t *testing.T) {
	rng := rand.New(rand.NewSource(73))
	for trial := 0; trial < 200; trial++ {
		count := 1 + rng.Intn(80)
		original := make([]Node, count)
		for i := range original {
			original[i].FocusID = fmt.Sprintf("N%d", i)
			if i > 0 && rng.Intn(3) != 0 {
				original[i].RelativePositionID = original[rng.Intn(i)].FocusID
			}
		}
		nodes := make([]Node, count)
		for i, p := range rng.Perm(count) {
			nodes[i] = original[p]
		}
		got := Plan(nodes)
		if len(got.Diagnostics) != 0 {
			t.Fatalf("valid DAG diagnosed: %+v", got.Diagnostics)
		}
		positions := make(map[string]int)
		for out, in := range got.Order {
			positions[nodes[in].FocusID] = out
		}
		for _, node := range nodes {
			if node.RelativePositionID != "" && positions[node.RelativePositionID] >= positions[node.FocusID] {
				t.Fatalf("dependent before anchor: %+v, order=%v", node, got.Order)
			}
		}
		assertPlanProperties(t, nodes, got)
	}
}

func TestPlanLongChain(t *testing.T) {
	const count = 20000
	nodes := make([]Node, count)
	for i := range nodes {
		nodes[i].FocusID = fmt.Sprintf("N%d", i)
		if i+1 < count {
			nodes[i].RelativePositionID = fmt.Sprintf("N%d", i+1)
		}
	}
	got := Plan(nodes)
	if len(got.Diagnostics) != 0 || len(got.Order) != count {
		t.Fatalf("invalid chain result: %d nodes %v", len(got.Order), got.Diagnostics)
	}
	for out, in := range got.Order {
		if in != count-out-1 {
			t.Fatalf("wrong long chain order at %d: %d", out, in)
		}
	}
}

func TestPlanDuplicateMissingAnchorsShareReadOnlyComponent(t *testing.T) {
	for _, count := range []int{16, 256, 2048} {
		t.Run(fmt.Sprintf("nodes_%d", count), func(t *testing.T) {
			nodes := duplicateMissingAnchorNodes(count)
			got := Plan(nodes)
			if len(got.Diagnostics) != count+1 {
				t.Fatalf("diagnostics=%d want=%d", len(got.Diagnostics), count+1)
			}
			storage := map[*int]int{}
			for _, diagnostic := range got.Diagnostics {
				if len(diagnostic.AffectedIndices) != count {
					t.Fatalf("component membership lost: %+v", diagnostic)
				}
				storage[&diagnostic.AffectedIndices[0]] = len(diagnostic.AffectedIndices)
				for i, member := range diagnostic.AffectedIndices {
					if i != member {
						t.Fatalf("membership at %d=%d", i, member)
					}
				}
			}
			if len(storage) != 1 {
				t.Fatalf("component members copied per finding: %d backing arrays for %d nodes", len(storage), count)
			}
			for i, index := range got.Order {
				if i != index {
					t.Fatalf("invalid component no longer preserves order: %v", got.Order)
				}
			}

			again := Plan(nodes)
			if &again.Diagnostics[0].AffectedIndices[0] == &got.Diagnostics[0].AffectedIndices[0] {
				t.Fatal("component storage escaped across calls")
			}
		})
	}
}

func duplicateMissingAnchorNodes(count int) []Node {
	nodes := make([]Node, count)
	for i := range nodes {
		nodes[i] = Node{FocusID: "duplicate", RelativePositionID: "absent"}
	}
	return nodes
}

func BenchmarkPlanDuplicateMissingAnchors(b *testing.B) {
	for _, count := range []int{128, 512, 2048} {
		b.Run(fmt.Sprintf("nodes_%d", count), func(b *testing.B) {
			nodes := duplicateMissingAnchorNodes(count)
			b.ReportAllocs()
			b.ResetTimer()
			var result Result
			for i := 0; i < b.N; i++ {
				result = Plan(nodes)
			}
			runtime.KeepAlive(result)
		})
	}
}

func FuzzPlanPreservesContentAndOrderContract(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 2, 2, 1, 3, 0})
	f.Add([]byte{1, 3, 1, 0, 3, 1, 4, 2})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			data = data[:256]
		}
		nodes := make([]Node, len(data)/2)
		for i := range nodes {
			if data[2*i] != 0 {
				nodes[i].FocusID = fmt.Sprintf("N%d", data[2*i]%32)
			}
			if data[2*i+1] != 0 {
				nodes[i].RelativePositionID = fmt.Sprintf("N%d", data[2*i+1]%32)
			}
		}
		original := append([]Node{}, nodes...)
		got := Plan(nodes)
		if !reflect.DeepEqual(original, nodes) {
			t.Fatal("input was mutated")
		}
		assertPlanProperties(t, nodes, got)
	})
}
