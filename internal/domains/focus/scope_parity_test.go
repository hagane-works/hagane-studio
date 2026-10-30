package focus

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

type focusScopeParityCase struct {
	Name               string           `json:"name"`
	ProjectCountryTags []string         `json:"project_country_tags"`
	TreeKind           string           `json:"tree_kind"`
	ScopeKind          string           `json:"scope_kind"`
	CountryTag         *string          `json:"country_tag"`
	SharedCountryTags  []string         `json:"shared_country_tags"`
	Blocked            bool             `json:"blocked"`
	Findings           []map[string]any `json:"findings"`
	Tree               struct {
		TreeKind          string   `json:"tree_kind"`
		TreeID            string   `json:"tree_id"`
		ScopeKind         string   `json:"scope_kind"`
		CountryTag        string   `json:"country_tag"`
		SharedCountryTags []string `json:"shared_country_tags"`
	} `json:"tree"`
}

func TestFocusScopeCustodyMatchesPython(t *testing.T) {
	raw, err := os.ReadFile("testdata/focus_scope_parity.json")
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fixture struct {
		Cases []focusScopeParityCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("[FAIL] fixture has no cases — the gate would pass vacuously")
	}

	seenCodes := map[string]bool{}
	for _, tc := range fixture.Cases {
		got := MaterializeFocusBundle(FocusScopeCandidate{
			TreeKind:          tc.Tree.TreeKind,
			TreeID:            tc.Tree.TreeID,
			ScopeKind:         tc.Tree.ScopeKind,
			CountryTag:        tc.Tree.CountryTag,
			SharedCountryTags: tc.Tree.SharedCountryTags,
		}, tc.ProjectCountryTags, nil, "focus_tree:create")

		if got.TreeKind != tc.TreeKind || got.ScopeKind != tc.ScopeKind {
			t.Errorf("%s: kind/scope = %q/%q, want %q/%q",
				tc.Name, got.TreeKind, got.ScopeKind, tc.TreeKind, tc.ScopeKind)
		}
		wantTag := ""
		if tc.CountryTag != nil {
			wantTag = *tc.CountryTag
		}
		if got.CountryTag != wantTag {
			t.Errorf("%s: country_tag = %q, want %q", tc.Name, got.CountryTag, wantTag)
		}
		if got.Blocked() != tc.Blocked {
			t.Errorf("%s: blocked = %v, want %v", tc.Name, got.Blocked(), tc.Blocked)
		}
		if len(got.Findings) != len(tc.Findings) {
			t.Errorf("%s: %d findings, want %d", tc.Name, len(got.Findings), len(tc.Findings))
			continue
		}
		for i, want := range tc.Findings {
			gotJSON, _ := json.Marshal(got.Findings[i])
			var gotMap map[string]any
			_ = json.Unmarshal(gotJSON, &gotMap)
			seenCodes[got.Findings[i].Code] = true
			if !reflect.DeepEqual(gotMap, want) {
				gotPretty, _ := json.MarshalIndent(gotMap, "", " ")
				wantPretty, _ := json.MarshalIndent(want, "", " ")
				t.Errorf("%s finding[%d] drifted from the Python policy\n got %s\nwant %s",
					tc.Name, i, gotPretty, wantPretty)
			}
		}
	}

	var missing []string
	for _, code := range FocusFindingCodes {
		if !seenCodes[code] {
			missing = append(missing, code)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("[FAIL] fixture never exercises %v — add a case to "+
			"focus_scope_parity.json, otherwise those codes' copy is unchecked", missing)
	}
}
