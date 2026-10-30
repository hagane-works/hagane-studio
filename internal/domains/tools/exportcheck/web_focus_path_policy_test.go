package exportcheck

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type parityFixture struct {
	NormalisePath []struct {
		Input  string  `json:"input"`
		Output *string `json:"output"`
	} `json:"normalise_path"`
	BuildPrimaryRegistry struct {
		Records  []parityRegistryRecord `json:"records"`
		ByTreeID map[string]string      `json:"by_tree_id"`
		ByPath   map[string]string      `json:"by_path"`
		Findings []any                  `json:"findings"`
	} `json:"build_primary_registry"`
	Decide struct {
		RegistryRecords []parityRegistryRecord `json:"registry_records"`
		Cases           []struct {
			Tree      map[string]any `json:"tree"`
			Namespace string         `json:"namespace"`
			Decision  struct {
				TreeID         string  `json:"tree_id"`
				OriginKind     string  `json:"origin_kind"`
				SourceFile     *string `json:"source_file"`
				EmitMode       string  `json:"emit_mode"`
				TargetPath     *string `json:"target_path"`
				TargetLayer    string  `json:"target_layer"`
				DecisionSource string  `json:"decision_source"`
				Findings       []any   `json:"findings"`
			} `json:"decision"`
		} `json:"cases"`
	} `json:"decide"`
	ConditionalOffset struct {
		Node  map[string]any `json:"node"`
		Cases []struct {
			TreeID string `json:"tree_id"`
			Result []int  `json:"result"`
		} `json:"cases"`
	} `json:"conditional_offset"`
	Layout []struct {
		Name        string             `json:"name"`
		Nodes       []parityLayoutNode `json:"nodes"`
		Mode        string             `json:"mode"`
		Projections []struct {
			FocusID            string `json:"focus_id"`
			BaseX              int    `json:"base_x"`
			BaseY              int    `json:"base_y"`
			RelativePositionID string `json:"relative_position_id"`
			OffsetX            int    `json:"offset_x"`
			OffsetY            int    `json:"offset_y"`
			Source             string `json:"source"`
			Condition          string `json:"condition"`
		} `json:"projections"`
		Before []parityLayoutNode `json:"before"`
		Result struct {
			Positions map[string][]int `json:"positions"`
			Findings  []any            `json:"findings"`
			Blocked   bool             `json:"blocked"`
		} `json:"result"`
	} `json:"layout"`
}

type parityRegistryRecord struct {
	TreeID        string `json:"tree_id"`
	CanonicalPath string `json:"canonical_path"`
	TreeKind      string `json:"tree_kind"`
	TargetLayer   string `json:"target_layer"`
	ContentHash   string `json:"content_hash"`
}

type parityLayoutNode struct {
	FocusID            string `json:"focus_id"`
	X                  int    `json:"x"`
	Y                  int    `json:"y"`
	RelativePositionID string `json:"relative_position_id"`
	Source             string `json:"source"`
}

func loadParityFixture(t *testing.T) parityFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/focus_policy_parity.json")
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx parityFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("fixture unreadable: %v", err)
	}
	return fx
}

func toRecords(rs []parityRegistryRecord) []RegistryRecord {
	out := make([]RegistryRecord, 0, len(rs))
	for _, r := range rs {
		out = append(out, RegistryRecord{
			TreeID: r.TreeID, CanonicalPath: r.CanonicalPath,
			TreeKind: r.TreeKind, TargetLayer: r.TargetLayer, ContentHash: r.ContentHash,
		})
	}
	return out
}

func findingsAsAny(t *testing.T, findings []FocusPolicyFinding) []any {
	t.Helper()
	if findings == nil {
		findings = []FocusPolicyFinding{}
	}
	raw, err := json.Marshal(findings)
	if err != nil {
		t.Fatalf("marshal findings: %v", err)
	}
	var out []any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal findings: %v", err)
	}
	return out
}

func normaliseAny(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestFocusPolicyParityNormalisePath(t *testing.T) {
	fx := loadParityFixture(t)
	if len(fx.NormalisePath) == 0 {
		t.Fatal("empty normalise_path fixture")
	}
	for _, c := range fx.NormalisePath {
		got, ok := normaliseFocusPath(c.Input)
		if c.Output == nil {
			if ok {
				t.Errorf("normalise(%q) = %q, python = None", c.Input, got)
			}
		} else if !ok || got != *c.Output {
			t.Errorf("normalise(%q) = %q/%v, python = %q", c.Input, got, ok, *c.Output)
		}
	}
}

func TestFocusPolicyParityBuildPrimaryRegistry(t *testing.T) {
	fx := loadParityFixture(t)
	c := fx.BuildPrimaryRegistry
	index := BuildPrimaryRegistry(RegistrySnapshot{Records: toRecords(c.Records)}, "")
	gotByTreeID := map[string]string{}
	for k, v := range index.ByTreeID {
		gotByTreeID[k] = v.CanonicalPath
	}
	if !reflect.DeepEqual(gotByTreeID, c.ByTreeID) {
		t.Errorf("by_tree_id: got %v, python %v", gotByTreeID, c.ByTreeID)
	}
	gotByPath := map[string]string{}
	for k, v := range index.ByPath {
		gotByPath[k] = v.TreeID
	}
	if !reflect.DeepEqual(gotByPath, c.ByPath) {
		t.Errorf("by_path: got %v, python %v", gotByPath, c.ByPath)
	}
	got := findingsAsAny(t, index.Findings)
	want := normaliseAny(t, c.Findings)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findings drift:\n got: %v\nwant: %v", got, want)
	}
}

func parityTreeFromMap(m map[string]any) FocusPolicyTree {
	str := func(k string) string {
		v, _ := m[k].(string)
		return v
	}
	return FocusPolicyTree{
		TreeID:     str("tree_id"),
		TreeKind:   str("tree_kind"),
		OriginKind: str("origin_kind"),
		SourceFile: str("source_file"),
	}
}

func TestFocusPolicyParityDecide(t *testing.T) {
	fx := loadParityFixture(t)
	index := BuildPrimaryRegistry(RegistrySnapshot{Records: toRecords(fx.Decide.RegistryRecords)}, "")
	if len(fx.Decide.Cases) == 0 {
		t.Fatal("empty decide fixture")
	}
	for _, c := range fx.Decide.Cases {
		d := DecideFocusFilePolicy(
			parityTreeFromMap(c.Tree), index, c.Namespace, "focus_file_policy:persist",
		)
		want := c.Decision
		optional := func(v string) *string {
			if v == "" {
				return nil
			}
			return &v
		}
		if d.TreeID != want.TreeID || d.OriginKind != want.OriginKind ||
			d.EmitMode != want.EmitMode || d.TargetLayer != want.TargetLayer ||
			d.DecisionSource != want.DecisionSource ||
			!reflect.DeepEqual(optional(d.SourceFile), want.SourceFile) ||
			!reflect.DeepEqual(optional(d.TargetPath), want.TargetPath) {
			t.Errorf("decide(%v) scalar drift:\n got: %+v\nwant: %+v", c.Tree, d, want)
		}
		got := findingsAsAny(t, d.Findings)
		wantFindings := normaliseAny(t, want.Findings)
		if !reflect.DeepEqual(got, wantFindings) {
			t.Errorf("decide(%v) findings drift:\n got: %v\nwant: %v", c.Tree, got, wantFindings)
		}
	}
}

func TestFocusPolicyParityConditionalOffset(t *testing.T) {
	fx := loadParityFixture(t)
	rawOffsets, _ := fx.ConditionalOffset.Node["offset"].([]any)
	offsets := make([]map[string]any, 0, len(rawOffsets))
	for _, item := range rawOffsets {
		m, _ := item.(map[string]any)
		offsets = append(offsets, m)
	}
	for _, c := range fx.ConditionalOffset.Cases {
		dx, dy := FocusTreeConditionalOffset(offsets, c.TreeID)
		if dx != c.Result[0] || dy != c.Result[1] {
			t.Errorf("offset(%q) = (%d,%d), python = %v", c.TreeID, dx, dy, c.Result)
		}
	}
}

func toLayoutNodes(ns []parityLayoutNode) []LayoutNode {
	out := make([]LayoutNode, 0, len(ns))
	for _, n := range ns {
		src := n.Source
		if src == "" {
			src = "project"
		}
		out = append(out, LayoutNode{
			FocusID: n.FocusID, X: n.X, Y: n.Y,
			RelativePositionID: n.RelativePositionID, Source: src,
		})
	}
	return out
}

func TestFocusPolicyParityLayout(t *testing.T) {
	fx := loadParityFixture(t)
	if len(fx.Layout) == 0 {
		t.Fatal("empty layout fixture")
	}
	for _, c := range fx.Layout {
		projections := make([]SharedProjectionPolicy, 0, len(c.Projections))
		for _, p := range c.Projections {
			projections = append(projections, SharedProjectionPolicy{
				FocusID: p.FocusID, BaseX: p.BaseX, BaseY: p.BaseY,
				RelativePositionID: p.RelativePositionID,
				OffsetX:            p.OffsetX, OffsetY: p.OffsetY,
				Source: p.Source, Condition: p.Condition,
			})
		}
		result := ValidateCandidateLayout(
			toLayoutNodes(c.Nodes), c.Mode, projections,
			toLayoutNodes(c.Before), c.Before != nil, "focus_layout:parity",
		)
		gotPositions := map[string][]int{}
		for id, pos := range result.Positions {
			gotPositions[id] = []int{pos[0], pos[1]}
		}
		if !reflect.DeepEqual(gotPositions, c.Result.Positions) {
			t.Errorf("[%s] positions: got %v, python %v", c.Name, gotPositions, c.Result.Positions)
		}
		got := findingsAsAny(t, result.Findings)
		want := normaliseAny(t, c.Result.Findings)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("[%s] findings drift:\n got: %v\nwant: %v", c.Name, got, want)
		}
		if result.Blocked() != c.Result.Blocked {
			t.Errorf("[%s] blocked = %v, python = %v", c.Name, result.Blocked(), c.Result.Blocked)
		}
	}
}
