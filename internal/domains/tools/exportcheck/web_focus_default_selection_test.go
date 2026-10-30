package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFocusDefaultSelectionParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "focus_defaults_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Emissions              []FocusEmission   `json:"emissions"`
				VanillaGenericPath     *string           `json:"vanilla_generic_path"`
				ReplacePaths           []string          `json:"replace_paths"`
				SelectedArtifactByPath map[string]string `json:"selected_artifact_by_path"`
				Source                 string            `json:"source"`
			} `json:"input"`
			Expected struct {
				EffectiveTrees         []EffectiveFocusTree `json:"effective_trees"`
				DefaultTreeIDs         []string             `json:"default_tree_ids"`
				ReplacePaths           []string             `json:"replace_paths"`
				SelectedArtifacts      [][]string           `json:"selected_artifacts"`
				VanillaGenericInjected bool                 `json:"vanilla_generic_injected"`
				FocusDirectoryReplaced bool                 `json:"focus_directory_replaced"`
				Blocked                bool                 `json:"blocked"`
				Findings               json.RawMessage      `json:"findings"`
			} `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	for _, c := range fx.Cases {
		in := FocusDefaultSelectionInput{
			Emissions:              c.Input.Emissions,
			ReplacePaths:           c.Input.ReplacePaths,
			SelectedArtifactByPath: c.Input.SelectedArtifactByPath,
			Source:                 c.Input.Source,
		}
		if c.Input.VanillaGenericPath != nil {
			in.VanillaGenericPath = *c.Input.VanillaGenericPath
		}
		got := SelectEffectiveFocusDefaults(in)

		wantTrees := c.Expected.EffectiveTrees
		if wantTrees == nil {
			wantTrees = []EffectiveFocusTree{}
		}
		if !reflect.DeepEqual(got.EffectiveTrees, wantTrees) {
			gb, _ := json.Marshal(got.EffectiveTrees)
			wb, _ := json.Marshal(wantTrees)
			t.Errorf("case %s effective_trees\n got %s\nwant %s", c.Name, gb, wb)
		}

		gotDefaults := []string{}
		for _, tree := range got.DefaultTrees {
			gotDefaults = append(gotDefaults, tree.TreeID)
		}
		wantDefaults := c.Expected.DefaultTreeIDs
		if wantDefaults == nil {
			wantDefaults = []string{}
		}
		if !reflect.DeepEqual(gotDefaults, wantDefaults) {
			t.Errorf("case %s default_tree_ids\n got %v\nwant %v",
				c.Name, gotDefaults, wantDefaults)
		}

		wantReplace := c.Expected.ReplacePaths
		if wantReplace == nil {
			wantReplace = []string{}
		}
		if !reflect.DeepEqual(got.ReplacePaths, wantReplace) {
			t.Errorf("case %s replace_paths\n got %v\nwant %v",
				c.Name, got.ReplacePaths, wantReplace)
		}

		gotSelected := [][]string{}
		for _, pair := range got.SelectedArtifacts {
			gotSelected = append(gotSelected, []string{pair.PathKey, pair.ArtifactID})
		}
		wantSelected := c.Expected.SelectedArtifacts
		if wantSelected == nil {
			wantSelected = [][]string{}
		}
		if !reflect.DeepEqual(gotSelected, wantSelected) {
			t.Errorf("case %s selected_artifacts\n got %v\nwant %v",
				c.Name, gotSelected, wantSelected)
		}

		if got.VanillaGenericInjected != c.Expected.VanillaGenericInjected {
			t.Errorf("case %s vanilla_generic_injected = %v want %v",
				c.Name, got.VanillaGenericInjected, c.Expected.VanillaGenericInjected)
		}
		if got.FocusDirectoryReplaced != c.Expected.FocusDirectoryReplaced {
			t.Errorf("case %s focus_directory_replaced = %v want %v",
				c.Name, got.FocusDirectoryReplaced, c.Expected.FocusDirectoryReplaced)
		}
		if got.Blocked() != c.Expected.Blocked {
			t.Errorf("case %s blocked = %v want %v",
				c.Name, got.Blocked(), c.Expected.Blocked)
		}

		gotFindings := jsonNorm(t, got.Findings)
		var wantFindings any
		if err := json.Unmarshal(c.Expected.Findings, &wantFindings); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotFindings, wantFindings) {
			gb, _ := json.Marshal(gotFindings)
			t.Errorf("case %s findings\n got %s\nwant %s", c.Name, gb, c.Expected.Findings)
		}
	}
}

func TestNormaliseModPathDotQuirk(t *testing.T) {
	if got, ok := normaliseModPath("."); !ok || got != "." {
		t.Errorf(`normaliseModPath(".") = %q,%v want ".",true`, got, ok)
	}
	if _, ok := normaliseModPath(".."); ok {
		t.Error(`".." 必须被拒`)
	}
	if _, ok := normaliseModPath("a/../b"); ok {
		t.Error(`"a/../b" 必须被拒`)
	}
}
