package rawfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/clausewitz"
)

func TestPositionedCandidateParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "positioned_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Candidates []struct {
			Name       string          `json:"name"`
			Text       string          `json:"text"`
			Candidates json.RawMessage `json:"candidates"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, c := range fx.Candidates {
		cands := CollectPositionedReferenceCandidates(c.Text, nil)
		list := make([]map[string]any, 0, len(cands))
		for _, x := range cands {
			list = append(list, x.ToMap())
		}
		got := jsonNorm(t, list)
		var want any
		if err := json.Unmarshal(c.Candidates, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("candidates %s mismatch\n got %s\nwant %s", c.Name, gb, c.Candidates)
		}
	}
}

func TestPositionedSpansSelectExactToken(t *testing.T) {
	for _, text := range []string{
		"add_ideas = { GER_x GER_xy }",
		"add_ideas = { GER_x GER_x }",
		"completion_reward = {\n\tadd_ideas = GER_war_economy\n}",
		"add_ideas = GER_x # 中文注释\nremove_ideas = GER_y",
	} {
		runes := []rune(text)
		for _, c := range CollectPositionedReferenceCandidates(text, nil) {
			got := string(runes[c.SourceSpan.StartOffset:c.SourceSpan.EndOffset])
			if got != c.TargetID {
				t.Errorf("%q: span %d..%d 切出 %q,目标是 %q",
					text, c.SourceSpan.StartOffset, c.SourceSpan.EndOffset, got, c.TargetID)
			}
			if c.AssignmentSpan != nil {
				as := *c.AssignmentSpan
				if c.SourceSpan.StartOffset < as.StartOffset || c.SourceSpan.EndOffset > as.EndOffset {
					t.Errorf("%q: 目标 span 跑到了容器赋值 span 之外", text)
				}
			}
		}
	}
}

func TestPositionedCandidateFeedsRepair(t *testing.T) {
	text := "completion_reward = {\n\tadd_ideas = GER_war_economi\n}\n"
	cands := CollectPositionedReferenceCandidates(text, nil)
	if len(cands) != 1 {
		t.Fatalf("candidates=%d want 1", len(cands))
	}
	repl := "GER_war_economy"
	plan, err := clausewitz.PlanCrossRefRepair(text,
		clausewitz.CrossRefFinding{Candidate: cands[0]}, &repl, false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := clausewitz.ApplyCrossRefRepair(text, plan)
	if err != nil {
		t.Fatal(err)
	}
	want := "completion_reward = {\n\tadd_ideas = GER_war_economy\n}\n"
	if out != want {
		t.Errorf("apply\n got %q\nwant %q", out, want)
	}
}

func jsonNorm(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
