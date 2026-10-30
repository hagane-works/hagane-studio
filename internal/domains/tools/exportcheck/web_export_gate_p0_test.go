package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExportGateP0Parity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "exportgate_p0_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	type rowJSON struct {
		ID      int64    `json:"id"`
		TreeIDs []string `json:"tree_ids"`
	}
	var fx struct {
		Policy []struct {
			Name       string             `json:"name"`
			Rows       []rowJSON          `json:"rows"`
			ByTreeID   map[string]int64   `json:"by_tree_id"`
			Duplicates map[string][]int64 `json:"duplicates"`
		} `json:"policy"`
		Cohorts []struct {
			Name     string    `json:"name"`
			Selected []string  `json:"selected"`
			All      []string  `json:"all"`
			Rows     []rowJSON `json:"rows"`
			Expanded []string  `json:"expanded"`
		} `json:"cohorts"`
		Findings []struct {
			Name    string          `json:"name"`
			Finding json.RawMessage `json:"finding"`
			Issue   json.RawMessage `json:"issue"`
		} `json:"findings"`
		SafePaths []struct {
			Input  string `json:"input"`
			Output string `json:"output"`
		} `json:"safe_paths"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	toRows := func(rs []rowJSON) []FocusPolicyRow {
		out := make([]FocusPolicyRow, 0, len(rs))
		for _, r := range rs {
			out = append(out, FocusPolicyRow{ID: r.ID, TreeIDs: r.TreeIDs})
		}
		return out
	}

	for _, c := range fx.Policy {
		gotBy := map[string]int64{}
		for k, v := range PolicyRowsByTreeID(toRows(c.Rows)) {
			gotBy[k] = v.ID
		}
		want := c.ByTreeID
		if want == nil {
			want = map[string]int64{}
		}
		if !reflect.DeepEqual(gotBy, want) {
			t.Errorf("policy %s by_tree_id\n got %v\nwant %v", c.Name, gotBy, want)
		}
		gotDup := map[string][]int64{}
		for k, rows := range DuplicatePolicyTreeIDs(toRows(c.Rows)) {
			ids := make([]int64, 0, len(rows))
			for _, r := range rows {
				ids = append(ids, r.ID)
			}
			gotDup[k] = ids
		}
		wantDup := c.Duplicates
		if wantDup == nil {
			wantDup = map[string][]int64{}
		}
		if !reflect.DeepEqual(gotDup, wantDup) {
			t.Errorf("policy %s duplicates\n got %v\nwant %v", c.Name, gotDup, wantDup)
		}
	}

	for _, c := range fx.Cohorts {
		got := ExpandFocusExportCohorts(c.Selected, c.All, toRows(c.Rows))
		want := c.Expanded
		if want == nil {
			want = []string{}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("cohort %s\n got %v\nwant %v", c.Name, got, want)
		}
	}

	for _, c := range fx.Findings {
		var finding map[string]any
		if err := json.Unmarshal(c.Finding, &finding); err != nil {
			t.Fatal(err)
		}
		got := jsonNorm(t, P0FindingToExportIssue(finding))
		var want any
		if err := json.Unmarshal(c.Issue, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("finding %s\n got %s\nwant %s", c.Name, gb, c.Issue)
		}
	}

	for _, c := range fx.SafePaths {
		if got := SafeRawPath(c.Input); got != c.Output {
			t.Errorf("SafeRawPath(%q) = %q want %q", c.Input, got, c.Output)
		}
	}
}

func TestCohortExpansionKeepsFileIntact(t *testing.T) {
	rows := []FocusPolicyRow{{ID: 1, TreeIDs: []string{"a", "b", "c"}}}
	got := ExpandFocusExportCohorts([]string{"a"}, []string{"a", "b", "c", "d"}, rows)
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("选中同伙里一棵应当拉齐整组,得到 %v", got)
	}

	chained := []FocusPolicyRow{
		{ID: 1, TreeIDs: []string{"a", "b"}},
		{ID: 2, TreeIDs: []string{"b", "c"}},
	}
	got2 := ExpandFocusExportCohorts([]string{"a"}, []string{"a", "b", "c"}, chained)
	if !reflect.DeepEqual(got2, []string{"a", "b"}) {
		t.Errorf("同伙不该传递闭包,得到 %v", got2)
	}
}

func TestPolicyConflictLowestIDWins(t *testing.T) {
	rows := []FocusPolicyRow{{ID: 9, TreeIDs: []string{"x"}},
		{ID: 3, TreeIDs: []string{"x"}}, {ID: 5, TreeIDs: []string{"x"}}}
	if got := PolicyRowsByTreeID(rows)["x"].ID; got != 3 {
		t.Errorf("应当是 id=3 赢,得到 %d", got)
	}
}
