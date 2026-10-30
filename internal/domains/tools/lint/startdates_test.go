package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStartDatesParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "startdates_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name      string  `json:"name"`
			StartDate *string `json:"start_date"`
			Bookmarks []struct {
				ID   string `json:"id"`
				Date string `json:"date"`
			} `json:"bookmarks"`
			RawFiles []struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			} `json:"raw_files"`
			Report json.RawMessage `json:"report"`
		} `json:"cases"`
		Math []struct {
			Value  string `json:"value"`
			Custom bool   `json:"custom"`
			Bump   bool   `json:"bump"`
		} `json:"math"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	for _, m := range fx.Math {
		if got := IsCustomStart(m.Value); got != m.Custom {
			t.Errorf("IsCustomStart(%q)=%v want %v", m.Value, got, m.Custom)
		}
		if got := NeedsEndDateBump(m.Value); got != m.Bump {
			t.Errorf("NeedsEndDateBump(%q)=%v want %v", m.Value, got, m.Bump)
		}
	}
	for _, c := range fx.Cases {
		var structured []StructuredBookmarkInput
		for _, b := range c.Bookmarks {
			structured = append(structured, StructuredBookmarkInput{
				BookmarkID: b.ID, Date: b.Date,
				Path:    "common/bookmarks/" + b.ID + ".txt",
				Source:  "created",
				Emitted: b.ID != "",
			})
		}
		var raws []PreservedRawBookmarkInput
		for _, r := range c.RawFiles {
			raws = append(raws, PreservedRawBookmarkInput{
				Path: r.Path, Content: r.Content, Source: "created", Emitted: true,
			})
		}
		start := ""
		if c.StartDate != nil {
			start = *c.StartDate
		}
		plan := BuildTimelineExportPlan(start, structured, raws)
		findings := LintStartDates(plan)
		var startOut any
		if c.StartDate != nil {
			startOut = *c.StartDate
		}
		got := LintFindingsReport(findings, map[string]any{
			"start_date":   startOut,
			"custom_start": IsCustomStart(start),
		})
		gotTree := jsonNorm(t, got)
		var wantTree any
		if err := json.Unmarshal(c.Report, &wantTree); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotTree, wantTree) {
			gb, _ := json.Marshal(gotTree)
			wb, _ := json.Marshal(wantTree)
			t.Errorf("case %s mismatch\n got %.900s\nwant %.900s", c.Name, gb, wb)
		}
	}
}
