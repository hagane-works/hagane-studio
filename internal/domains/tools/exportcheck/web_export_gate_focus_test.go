package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type focusGateFixtureNode struct {
	FocusID            string            `json:"focus_id"`
	X                  int               `json:"x"`
	Y                  int               `json:"y"`
	RelativePositionID string            `json:"relative_position_id"`
	Source             string            `json:"source"`
	CompletionReward   string            `json:"completion_reward"`
	NameEn             string            `json:"name_en"`
	NameZh             string            `json:"name_zh"`
	DisplayNameLocs    map[string]string `json:"display_name_locs"`
	Prerequisites      [][]string        `json:"prerequisites"`
	Offset             []map[string]any  `json:"offset"`
	SourceSnapshot     map[string]any    `json:"source_snapshot"`
}

type focusGateFixtureTree struct {
	TreeID              string                 `json:"tree_id"`
	TreeKind            string                 `json:"tree_kind"`
	CountryTag          *string                `json:"country_tag"`
	IsDefault           bool                   `json:"is_default"`
	SharedFocusRefs     []map[string]any       `json:"shared_focus_refs"`
	SourceFile          string                 `json:"source_file"`
	InitialShowPosition map[string]any         `json:"initial_show_position"`
	InitialShowFocus    *string                `json:"initial_show_focus"`
	ScopeKind           string                 `json:"scope_kind"`
	SharedCountryTags   []string               `json:"shared_country_tags"`
	Source              string                 `json:"source"`
	Nodes               []focusGateFixtureNode `json:"nodes"`
}

func TestFocusGateParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "focus_gate_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Project struct {
					CountryTag        string   `json:"country_tag"`
					ExportLanguages   []string `json:"export_languages"`
					Dependencies      []string `json:"dependencies"`
					BasemodTemplateID any      `json:"basemod_template_id"`
				} `json:"project"`
				Trees        []focusGateFixtureTree `json:"trees"`
				Countries    []string               `json:"countries"`
				RawFiles     []FocusGateRawFile     `json:"raw_files"`
				FilePolicies []struct {
					ID             int64    `json:"id"`
					SourceFile     string   `json:"source_file"`
					OriginKind     string   `json:"origin_kind"`
					TreeIDs        []string `json:"tree_ids"`
					MatchedTreeIDs []string `json:"matched_tree_ids"`
					EmitMode       string   `json:"emit_mode"`
					TargetPath     string   `json:"target_path"`
					TargetLayer    string   `json:"target_layer"`
					DecisionSource string   `json:"decision_source"`
				} `json:"file_policies"`
				ReplacePaths []string `json:"replace_paths"`
			} `json:"input"`
			Findings json.RawMessage `json:"findings"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	for _, c := range fx.Cases {
		in := FocusFindingsInput{
			Project: FocusGateProject{
				CountryTag:        c.Input.Project.CountryTag,
				ExportLanguages:   c.Input.Project.ExportLanguages,
				Dependencies:      c.Input.Project.Dependencies,
				BasemodTemplateID: c.Input.Project.BasemodTemplateID,
			},
			Countries:    c.Input.Countries,
			RawFiles:     c.Input.RawFiles,
			ReplacePaths: c.Input.ReplacePaths,
		}
		for _, ft := range c.Input.Trees {
			gt := FocusTree{
				TreeID: ft.TreeID, TreeKind: ft.TreeKind, CountryTag: ft.CountryTag,
				IsDefault: ft.IsDefault, SharedFocusRefs: ft.SharedFocusRefs,
				SourceFile: ft.SourceFile, InitialShowPosition: ft.InitialShowPosition,
				ScopeKind: ft.ScopeKind, SharedCountryTags: ft.SharedCountryTags,
				Source: ft.Source,
			}
			if ft.InitialShowFocus != nil {
				gt.InitialShowFocus = *ft.InitialShowFocus
			}
			for _, fn := range ft.Nodes {
				gt.Nodes = append(gt.Nodes, newFocusNodeForGate(fn))
			}
			in.Trees = append(in.Trees, gt)
		}
		for _, fp := range c.Input.FilePolicies {
			in.FilePolicies = append(in.FilePolicies, FocusGatePolicyRow{
				ID: fp.ID,
				Policy: FocusFilePolicy{
					SourceFile: fp.SourceFile, OriginKind: fp.OriginKind,
					TreeIDs: fp.TreeIDs, MatchedTreeIDs: fp.MatchedTreeIDs,
					EmitMode: fp.EmitMode, TargetPath: fp.TargetPath,
					TargetLayer: fp.TargetLayer, DecisionSource: fp.DecisionSource,
				},
			})
		}

		got, err := FocusFindings(in)
		if err != nil {
			t.Fatalf("case %s: %v", c.Name, err)
		}
		gotNorm := jsonNorm(t, got)
		var want any
		if err := json.Unmarshal(c.Findings, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotNorm, want) {
			gb, _ := json.MarshalIndent(gotNorm, "", " ")
			t.Errorf("case %s\n got %s\nwant %s", c.Name, gb, c.Findings)
		}
	}
}

func newFocusNodeForGate(fn focusGateFixtureNode) FocusNode {
	source := fn.Source
	if source == "" {
		source = "created"
	}
	return FocusNode{
		FocusID: fn.FocusID, X: fn.X, Y: fn.Y,
		RelativePositionID: fn.RelativePositionID, Source: source,
		CompletionReward: fn.CompletionReward, NameEn: fn.NameEn, NameZh: fn.NameZh,
		DisplayNameLocs: fn.DisplayNameLocs, Prerequisites: fn.Prerequisites,
		Offset: fn.Offset, sourceSnapshot: fn.SourceSnapshot,
	}
}
