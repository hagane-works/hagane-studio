package exportcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExportCollectParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "export_collect_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name  string `json:"name"`
			Input struct {
				Project struct {
					ID                any      `json:"id"`
					Name              string   `json:"name"`
					CountryTag        string   `json:"country_tag"`
					StartDate         *string  `json:"start_date"`
					ExportLanguages   []string `json:"export_languages"`
					Dependencies      []string `json:"dependencies"`
					BasemodTemplateID any      `json:"basemod_template_id"`
				} `json:"project"`
				Bookmarks []struct {
					BookmarkID          string           `json:"bookmark_id"`
					Date                *string          `json:"date"`
					Source              string           `json:"source"`
					DefaultFlag         bool             `json:"default_flag"`
					DefaultCountry      string           `json:"default_country"`
					SelectableCountries []map[string]any `json:"selectable_countries"`
					Filters             []string         `json:"filters"`
				} `json:"bookmarks"`
				RawFiles []FocusGateRawFile `json:"raw_files"`
			} `json:"input"`
			TimelinePlan struct {
				ProjectStartDate                   *string         `json:"project_start_date"`
				ProjectParsedDate                  *string         `json:"project_parsed_date"`
				EarliestEmittedDate                *string         `json:"earliest_emitted_date"`
				EmitsBookmark                      bool            `json:"emits_bookmark"`
				ShouldEmitDefines                  bool            `json:"should_emit_defines"`
				DefinesStartDate                   *string         `json:"defines_start_date"`
				NeedsBookmarkReplacePath           bool            `json:"needs_bookmark_replace_path"`
				DescriptorReplacePaths             []string        `json:"descriptor_replace_paths"`
				DescriptorReplacePathsWithExisting []string        `json:"descriptor_replace_paths_with_existing"`
				Findings                           json.RawMessage `json:"findings"`
			} `json:"timeline_plan"`
			QualityFindings   json.RawMessage `json:"quality_findings"`
			CollectedFindings json.RawMessage `json:"collected_findings"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("fixture 为空")
	}

	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	for _, c := range fx.Cases {
		project := ExportGateProject{
			ID: c.Input.Project.ID, Name: c.Input.Project.Name,
			CountryTag:        c.Input.Project.CountryTag,
			StartDate:         deref(c.Input.Project.StartDate),
			ExportLanguages:   c.Input.Project.ExportLanguages,
			Dependencies:      c.Input.Project.Dependencies,
			BasemodTemplateID: c.Input.Project.BasemodTemplateID,
		}
		bookmarks := []ExportGateBookmark{}
		for _, b := range c.Input.Bookmarks {
			bookmarks = append(bookmarks, ExportGateBookmark{
				BookmarkID: b.BookmarkID, Date: deref(b.Date), Source: b.Source,
				DefaultFlag: b.DefaultFlag, DefaultCountry: b.DefaultCountry,
				SelectableCountries: b.SelectableCountries, Filters: b.Filters,
			})
		}
		raws := c.Input.RawFiles

		plan := BuildTimelinePlanForExport(project, bookmarks, raws)
		check := func(label string, got, want any) {
			if !reflect.DeepEqual(got, want) {
				t.Errorf("case %s %s = %v want %v", c.Name, label, got, want)
			}
		}
		check("emits_bookmark", plan.EmitsBookmark, c.TimelinePlan.EmitsBookmark)
		check("should_emit_defines", plan.ShouldEmitDefines, c.TimelinePlan.ShouldEmitDefines)
		check("defines_start_date", plan.DefinesStartDate, deref(c.TimelinePlan.DefinesStartDate))
		check("needs_bookmark_replace_path", plan.NeedsBookmarkReplacePath,
			c.TimelinePlan.NeedsBookmarkReplacePath)
		check("earliest_emitted_date", plan.EarliestEmittedDate,
			deref(c.TimelinePlan.EarliestEmittedDate))
		check("project_parsed_date", plan.ProjectParsedDate,
			deref(c.TimelinePlan.ProjectParsedDate))

		wantPaths := c.TimelinePlan.DescriptorReplacePaths
		if wantPaths == nil {
			wantPaths = []string{}
		}
		check("descriptor_replace_paths", plan.DescriptorReplacePaths(nil), wantPaths)
		check("descriptor_replace_paths(existing)",
			plan.DescriptorReplacePaths([]string{"common/ideas", "common/bookmarks"}),
			c.TimelinePlan.DescriptorReplacePathsWithExisting)

		planFindings := []map[string]any{}
		for _, f := range plan.Findings {
			planFindings = append(planFindings, f.ToMap())
		}
		compareJSON(t, c.Name+" plan.findings", planFindings, c.TimelinePlan.Findings)

		quality, rawFindings := BuildBookmarkQualityForExport(bookmarks, raws, "")
		qualityFindings := []map[string]any{}
		for _, f := range quality.Findings {
			qualityFindings = append(qualityFindings, f.ToMap())
		}
		qualityFindings = append(qualityFindings, rawFindings...)
		compareJSON(t, c.Name+" quality.findings", qualityFindings, c.QualityFindings)

		collected, _, err := CollectP0ExportFindings(CollectP0ExportFindingsInput{
			Project: project, Bookmarks: bookmarks, RawFiles: raws,
		})
		if err != nil {
			t.Fatalf("case %s: %v", c.Name, err)
		}
		compareJSON(t, c.Name+" collected", collected, c.CollectedFindings)
	}
}

func compareJSON(t *testing.T, label string, got any, want json.RawMessage) {
	t.Helper()
	gotNorm := jsonNorm(t, got)
	var wantNorm any
	if err := json.Unmarshal(want, &wantNorm); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotNorm, wantNorm) {
		gb, _ := json.MarshalIndent(gotNorm, "", " ")
		t.Errorf("%s\n got %s\nwant %s", label, gb, want)
	}
}
