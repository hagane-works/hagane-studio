package clausewitz

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type spanJSON struct {
	StartOffset int `json:"start_offset"`
	EndOffset   int `json:"end_offset"`
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_column"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_column"`
}

func (s spanJSON) toSpan() SourceSpan {
	return SourceSpan{s.StartOffset, s.EndOffset, s.StartLine, s.StartColumn, s.EndLine, s.EndColumn}
}

type candJSON struct {
	EffectName      string    `json:"effect_name"`
	TargetType      string    `json:"target_type"`
	TargetID        string    `json:"target_id"`
	SourceSpan      spanJSON  `json:"source_span"`
	ParentPath      []string  `json:"parent_path"`
	AssignmentSpan  *spanJSON `json:"assignment_span"`
	BooleanAncestry []string  `json:"boolean_ancestry"`
}

func (c candJSON) toCandidate() ReferenceCandidate {
	out := ReferenceCandidate{
		EffectName: c.EffectName, TargetType: c.TargetType, TargetID: c.TargetID,
		SourceSpan: c.SourceSpan.toSpan(), ParentPath: c.ParentPath,
		BooleanAncestry: c.BooleanAncestry,
	}
	if c.AssignmentSpan != nil {
		s := c.AssignmentSpan.toSpan()
		out.AssignmentSpan = &s
	}
	return out
}

type catalogJSON struct {
	ProjectIDs       map[string][]string `json:"project_ids"`
	VanillaIDs       map[string][]string `json:"vanilla_ids"`
	BaseModIDs       map[string][]string `json:"base_mod_ids"`
	CompleteTypes    []string            `json:"complete_types"`
	ExpectsBaseMod   bool                `json:"expects_base_mod"`
	BaseIndexedTypes []string            `json:"base_indexed_types"`
	TypoSuggestions  [][]any             `json:"typo_suggestions"`
	FatalKinds       [][]string          `json:"fatal_reference_kinds"`
}

func (c catalogJSON) toCatalog(t *testing.T) CrossRefCatalog {
	t.Helper()
	conv := func(m map[string][]string) map[string]map[string]bool {
		out := map[string]map[string]bool{}
		for k, ids := range m {
			set := map[string]bool{}
			for _, id := range ids {
				set[id] = true
			}
			out[k] = set
		}
		return out
	}
	set := func(xs []string) map[string]bool {
		out := map[string]bool{}
		for _, x := range xs {
			out[x] = true
		}
		return out
	}
	typo := map[[2]string]string{}
	for _, pair := range c.TypoSuggestions {
		key, ok := pair[0].([]any)
		if !ok || len(key) != 2 {
			t.Fatalf("bad typo_suggestions entry %v", pair)
		}
		typo[[2]string{key[0].(string), key[1].(string)}] = pair[1].(string)
	}
	fatal := map[[2]string]bool{}
	for _, k := range c.FatalKinds {
		fatal[[2]string{k[0], k[1]}] = true
	}
	return CrossRefCatalog{
		ProjectIDs: conv(c.ProjectIDs), VanillaIDs: conv(c.VanillaIDs),
		BaseModIDs: conv(c.BaseModIDs), CompleteTypes: set(c.CompleteTypes),
		ExpectsBaseMod: c.ExpectsBaseMod, BaseIndexedTypes: set(c.BaseIndexedTypes),
		TypoSuggestions: typo, FatalReferenceKind: fatal,
	}
}

func TestCrossRefParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "crossref_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Reports []struct {
			Name       string          `json:"name"`
			Text       string          `json:"text"`
			Catalog    catalogJSON     `json:"catalog"`
			Candidates []candJSON      `json:"candidates"`
			Report     json.RawMessage `json:"report"`
		} `json:"reports"`
		Plans []struct {
			Label            string          `json:"label"`
			Candidate        candJSON        `json:"candidate"`
			ReplacementID    *string         `json:"replacement_id"`
			DeleteAssignment bool            `json:"delete_assignment"`
			Plan             json.RawMessage `json:"plan"`
			Applied          *string         `json:"applied"`
		} `json:"plans"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for _, rc := range fx.Reports {
		cands := make([]ReferenceCandidate, 0, len(rc.Candidates))
		for _, c := range rc.Candidates {
			cands = append(cands, c.toCandidate())
		}
		rep, err := AnalyzeCrossReferences(rc.Text, cands, rc.Catalog.toCatalog(t))
		if err != nil {
			t.Errorf("report %s: %v", rc.Name, err)
			continue
		}
		got := jsonNorm(t, rep.ToMap())
		var want any
		if err := json.Unmarshal(rc.Report, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("report %s mismatch\n got %s\nwant %s", rc.Name, gb, rc.Report)
		}
	}

	text := fx.Reports[0].Text
	for _, pc := range fx.Plans {
		finding := CrossRefFinding{Candidate: pc.Candidate.toCandidate()}
		plan, err := PlanCrossRefRepair(text, finding, pc.ReplacementID, pc.DeleteAssignment)
		if err != nil {
			t.Errorf("plan %s: %v", pc.Label, err)
			continue
		}
		got := jsonNorm(t, plan.ToMap())
		var want any
		if err := json.Unmarshal(pc.Plan, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("plan %s mismatch\n got %s\nwant %s", pc.Label, gb, pc.Plan)
			continue
		}
		if pc.Applied != nil {
			applied, err := ApplyCrossRefRepair(text, plan)
			if err != nil {
				t.Errorf("apply %s: %v", pc.Label, err)
			} else if applied != *pc.Applied {
				t.Errorf("apply %s\n got %q\nwant %q", pc.Label, applied, *pc.Applied)
			}
		} else if plan.Applicable {
			t.Errorf("plan %s: Go 认为可用但 Python 没给 applied", pc.Label)
		}
	}
}

func TestCrossRefRepairGuards(t *testing.T) {
	text := "add_ideas = GER_x\n"
	span := Span([]rune(text), 12, 17)
	c := ReferenceCandidate{EffectName: "add_ideas", TargetType: "idea",
		TargetID: "GER_x", SourceSpan: span}
	f := CrossRefFinding{Candidate: c}
	repl := "GER_y"

	if _, err := PlanCrossRefRepair(text, f, &repl, true); err == nil {
		t.Error("同时给替换和删除应当拒绝")
	}
	if _, err := PlanCrossRefRepair(text, f, nil, false); err == nil {
		t.Error("两个意图都不给应当拒绝")
	}
	plan, err := PlanCrossRefRepair(text, f, &repl, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyCrossRefRepair("完全不同的源码", plan); err == nil {
		t.Error("源码已变应当拒绝应用")
	}
	out, err := ApplyCrossRefRepair(text, plan)
	if err != nil || out != "add_ideas = GER_y\n" {
		t.Errorf("apply = %q err=%v", out, err)
	}
}

func TestCrossRefRejectsMisalignedSpan(t *testing.T) {
	text := "add_ideas = GER_x\n"
	bad := ReferenceCandidate{EffectName: "add_ideas", TargetType: "idea",
		TargetID: "GER_x", SourceSpan: Span([]rune(text), 11, 16)}
	if _, err := AnalyzeCrossReferences(text, []ReferenceCandidate{bad}, CrossRefCatalog{}); err == nil {
		t.Error("span 与 target_id 不符时应当报错")
	}
}
