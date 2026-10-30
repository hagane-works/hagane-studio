package timeline

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseDateAndNormalise(t *testing.T) {
	cases := []struct {
		input string
		want  string
		valid bool
	}{
		{input: "1964.5.1", want: "1964.5.1.12", valid: true},
		{input: " 183.1.1.0 ", want: "183.1.1.0", valid: true},
		{input: "1964.5.1.24", valid: false},
		{input: "1964.13.1", valid: false},
		{input: "1964.5", valid: false},
		{input: "", valid: false},
	}
	for _, tc := range cases {
		got, ok := ParseDate(tc.input)
		if ok != tc.valid {
			t.Fatalf("ParseDate(%q) valid=%v want %v", tc.input, ok, tc.valid)
		}
		if ok && got.Canonical() != tc.want {
			t.Fatalf("ParseDate(%q) canonical=%q want %q", tc.input, got.Canonical(), tc.want)
		}
	}
}

func TestBuildCustomTimelineEmitsMinimalDefinesAndBookmarkReplacePath(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		ModDirectory:     "iron_curtain",
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1.12"}, Emitted: true,
		}},
		RequestedReplacePaths: []string{"common/ideas"},
	})
	if plan.EffectiveStartDate != "1964.5.1.12" {
		t.Fatalf("start=%q", plan.EffectiveStartDate)
	}
	if plan.EffectiveEndDate != "2064.1.1.12" || plan.EndDateSource != "compatibility_fallback" {
		t.Fatalf("end=%q source=%q", plan.EffectiveEndDate, plan.EndDateSource)
	}
	if !plan.EmitsBookmark || !plan.EmitDefines || !plan.Defines.EmitsEndDate {
		t.Fatalf("bookmark=%v defines=%v end=%v", plan.EmitsBookmark, plan.EmitDefines, plan.Defines.EmitsEndDate)
	}
	if plan.Defines.Path != "common/defines/iron_curtain_defines.lua" {
		t.Fatalf("defines path=%q", plan.Defines.Path)
	}
	wantContent := "NDefines.NGame.START_DATE = \"1964.5.1.12\"\nNDefines.NGame.END_DATE = \"2064.1.1\"\n"
	if plan.Defines.Content != wantContent {
		t.Fatalf("defines content:\n%s", plan.Defines.Content)
	}
	if !reflect.DeepEqual(plan.EffectiveReplacePaths, []string{"common/ideas", "common/bookmarks"}) {
		t.Fatalf("replace paths=%#v", plan.EffectiveReplacePaths)
	}
	if !hasFinding(plan, "timeline.end_date_defaulted") || !hasFinding(plan, "timeline.tension_scale_not_reviewed") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestNoBookmarkNeverAddsBookmarkReplacePath(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate:      "1964.5.1",
		RequestedReplacePaths: []string{"common/ideas", "COMMON/IDEAS"},
		InheritedReplacePaths: []string{"events"},
		ModDirectory:          "timeline_fixture",
	})
	if plan.EmitsBookmark {
		t.Fatal("no emitted bookmark must not be reported as emitted")
	}
	if !plan.EmitDefines {
		t.Fatal("a valid custom project start still needs a timeline define")
	}
	if containsFold(plan.EffectiveReplacePaths, BookmarksPath) {
		t.Fatalf("no bookmark must not add %q: %#v", BookmarksPath, plan.EffectiveReplacePaths)
	}
	if !reflect.DeepEqual(plan.EffectiveReplacePaths, []string{"common/ideas", "events"}) {
		t.Fatalf("replace paths=%#v", plan.EffectiveReplacePaths)
	}
	if !hasFinding(plan, "timeline.custom_start_without_bookmark") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestRawBookmarkSamePathWinsAndDrivesStartDate(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "campaign", Path: "common/bookmarks/campaign.txt", DateTokens: []string{"1936.1.1"}, Emitted: true,
		}},
		RawBookmarks: []BookmarkEvidence{{
			ID: "raw-campaign", Path: "common/bookmarks/campaign.txt", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
	})
	if plan.EffectiveStartDate != "1964.5.1.12" {
		t.Fatalf("raw winner start=%q", plan.EffectiveStartDate)
	}
	if len(plan.BookmarkSources) != 2 || plan.BookmarkSources[0].Winner || !plan.BookmarkSources[1].Winner {
		t.Fatalf("sources=%#v", plan.BookmarkSources)
	}
	if plan.BookmarkSources[0].ShadowedBy != "raw-campaign" {
		t.Fatalf("shadowed by=%q", plan.BookmarkSources[0].ShadowedBy)
	}
	if !hasFinding(plan, "timeline.bookmark_path_shadow_date_mismatch") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestExplicitEndDateWinsOverCompatibilityFallback(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		RequestedEndDate: "1976.12.31",
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
	})
	if plan.EndDateSource != "explicit" || plan.EffectiveEndDate != "1976.12.31.12" {
		t.Fatalf("end=%q source=%q", plan.EffectiveEndDate, plan.EndDateSource)
	}
	if strings.Contains(plan.Defines.Content, "2064") || !strings.Contains(plan.Defines.Content, `END_DATE = "1976.12.31.12"`) {
		t.Fatalf("defines content=%q", plan.Defines.Content)
	}
	if hasFinding(plan, "timeline.end_date_defaulted") {
		t.Fatalf("explicit end must not default: %#v", plan.Findings)
	}
}

func TestExplicitEndDateIsHonouredBeforeVanillaEnd(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1939.9.1", RequestedEndDate: "1950.1.1",
		StructuredBookmarks: []BookmarkEvidence{{ID: "1939", DateTokens: []string{"1939.9.1"}, Emitted: true}},
	})
	if plan.EndDateSource != "explicit" || plan.EffectiveEndDate != "1950.1.1.12" || !plan.Defines.EmitsEndDate {
		t.Fatalf("explicit end was not emitted: %#v", plan)
	}
	if !strings.Contains(plan.Defines.Content, `END_DATE = "1950.1.1.12"`) {
		t.Fatalf("defines content=%q", plan.Defines.Content)
	}
}

func TestExplicitEndDateWithoutProjectStartEmitsEndOnly(t *testing.T) {
	plan := Build(Input{RequestedEndDate: "1956.12.31"})
	if plan.EffectiveStartDate != "" || plan.EffectiveEndDate != "1956.12.31.12" || plan.EndDateSource != "explicit" {
		t.Fatalf("plan=%#v", plan)
	}
	if strings.Contains(plan.Defines.Content, "START_DATE") || !strings.Contains(plan.Defines.Content, `END_DATE = "1956.12.31.12"`) {
		t.Fatalf("defines content=%q", plan.Defines.Content)
	}
}

func TestLatestSelectableBookmarkDrivesEndDateCoverage(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1936.1.1",
		StructuredBookmarks: []BookmarkEvidence{
			{ID: "1936", DateTokens: []string{"1936.1.1"}, Emitted: true},
			{ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true},
		},
	})
	if plan.EffectiveStartDate != "1936.1.1.12" || plan.EffectiveEndDate != "2064.1.1.12" {
		t.Fatalf("timeline coverage=%#v", plan)
	}
	if !plan.EmitDefines || !plan.Defines.EmitsEndDate || !strings.Contains(plan.Defines.Content, `END_DATE = "2064.1.1"`) {
		t.Fatalf("late bookmark is not playable: %#v", plan.Defines)
	}
}

func TestEndDatePolicyBoundaries(t *testing.T) {
	cases := []struct {
		name        string
		start       string
		wantEnd     string
		wantSource  string
		wantEmitEnd bool
	}{
		{name: "vanilla", start: "1936.1.1", wantEnd: "1949.1.1.12", wantSource: "vanilla", wantEmitEnd: false},
		{name: "before vanilla end", start: "1939.9.1", wantEnd: "1949.1.1.12", wantSource: "vanilla", wantEmitEnd: false},
		{name: "on vanilla end", start: "1949.1.1", wantEnd: "2049.1.1.12", wantSource: "compatibility_fallback", wantEmitEnd: true},
		{name: "after vanilla end", start: "1950.1.1", wantEnd: "2050.1.1.12", wantSource: "compatibility_fallback", wantEmitEnd: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := Build(Input{
				ProjectStartDate: tc.start,
				StructuredBookmarks: []BookmarkEvidence{{
					ID: "campaign", DateTokens: []string{tc.start}, Emitted: true,
				}},
			})
			if plan.EffectiveEndDate != tc.wantEnd || plan.EndDateSource != tc.wantSource || plan.Defines.EmitsEndDate != tc.wantEmitEnd {
				t.Fatalf("end=%q source=%q emits_end=%v", plan.EffectiveEndDate, plan.EndDateSource, plan.Defines.EmitsEndDate)
			}
		})
	}
}

func TestRawDefinesWinnerSuppressesGeneratedDefinesAndReportsStartEndConflicts(t *testing.T) {
	path := "common/defines/iron_curtain_defines.lua"
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		RequestedEndDate: "1976.12.31",
		DefinesPath:      path,
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
		RawDefines: []RawDefinesEvidence{{
			ID: "raw-defines", Path: path, Emitted: true,
			Content: "-- NDefines.NGame.START_DATE = \"1964.5.1\"\n" +
				"NDefines.NGame.START_DATE = \"1939.1.1\"\n" +
				"NDefines.NGame.END_DATE = \"1949.1.1\"\n",
		}},
	})
	if plan.EmitDefines || plan.Defines.Content != "" || plan.RawDefinesWinner == nil {
		t.Fatalf("raw same-path winner was not preserved: %#v", plan)
	}
	if plan.RawDefinesWinner.ParsedStartDate != "1939.1.1.12" {
		t.Fatalf("raw start=%q", plan.RawDefinesWinner.ParsedStartDate)
	}
	if !hasFinding(plan, "timeline.raw_defines_start_date_conflict") ||
		!hasFinding(plan, "timeline.raw_defines_end_date_conflict") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestRawDefinesExplicitEndConflictIsVisible(t *testing.T) {
	path := "common/defines/fixture_defines.lua"
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		RequestedEndDate: "1976.12.31",
		DefinesPath:      path,
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
		RawDefines: []RawDefinesEvidence{{
			Path: path, Emitted: true, StartDate: "1964.5.1", EndDate: "1980.1.1",
		}},
	})
	if !hasFinding(plan, "timeline.raw_defines_end_date_conflict") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestLaterRawDefinesParticipatesInFinalFilenameOrder(t *testing.T) {
	generated := "common/defines/fixture_defines.lua"
	plan := Build(Input{
		ProjectStartDate: "1964.5.1", RequestedEndDate: "1976.12.31", DefinesPath: generated,
		StructuredBookmarks: []BookmarkEvidence{{ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true}},
		RawDefines: []RawDefinesEvidence{{
			ID: "late-raw", Path: "common/defines/zz_custom.lua", Emitted: true,
			Content: `NDefines.NGame.START_DATE = "1939.1.1"`,
		}},
	})
	if !plan.EmitDefines || plan.RawDefinesWinner == nil || !plan.RawDefinesWinner.LoadsAfterGenerated {
		t.Fatalf("late raw evidence missing: %#v", plan)
	}
	if !hasFinding(plan, "timeline.raw_defines_start_date_conflict") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestLaterRawDefinesInvalidEndDateIsVisible(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1", DefinesPath: "common/defines/fixture_defines.lua",
		StructuredBookmarks: []BookmarkEvidence{{ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true}},
		RawDefines: []RawDefinesEvidence{{
			ID: "late-raw", Path: "common/defines/zz_custom.lua", Emitted: true,
			Content: `NDefines.NGame.END_DATE = "not-a-date"`,
		}},
	})
	if !hasFinding(plan, "timeline.raw_defines_end_date_missing_or_invalid") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestRawDefinesChangeVanillaTimelineWhenNoGeneratedDefinesExist(t *testing.T) {
	plan := Build(Input{
		DefinesPath: "common/defines/hagane_defines.lua",
		RawDefines: []RawDefinesEvidence{{
			ID: "late-start", Path: "common/defines/zz_custom.lua", Emitted: true,
			Content: `NDefines.NGame.START_DATE = "1964.5.1"`,
		}},
	})
	if plan.EmitDefines {
		t.Fatal("vanilla plan should not manufacture generated defines")
	}
	if !hasFinding(plan, "timeline.raw_defines_start_date_conflict") ||
		!hasFinding(plan, "timeline.raw_defines_end_date_missing_or_invalid") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestSubmodLateBookmarkDoesNotAssumeVanillaEndDate(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1", HasDependencies: true,
		StructuredBookmarks: []BookmarkEvidence{{ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true}},
	})
	if plan.Defines.EmitsEndDate || plan.EffectiveEndDate != "" || plan.EndDateSource != "unregistered" {
		t.Fatalf("submod inherited an unverified vanilla end: %#v", plan)
	}
	if !hasFinding(plan, "timeline.base_timeline_source_unregistered") {
		t.Fatalf("findings=%#v", plan.Findings)
	}
}

func TestSubmodEarlyBookmarkStillReportsUnregisteredBaseTimeline(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1939.9.1", HasDependencies: true,
		StructuredBookmarks: []BookmarkEvidence{{ID: "1939", DateTokens: []string{"1939.9.1"}, Emitted: true}},
	})
	if plan.EffectiveEndDate != "" || plan.EndDateSource != "unregistered" ||
		!hasFinding(plan, "timeline.base_timeline_source_unregistered") {
		t.Fatalf("submod timeline claimed an unregistered vanilla end: %#v", plan)
	}
}

func TestRawDefinesFindingsKeepPerFieldOwners(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1", RequestedEndDate: "1976.12.31",
		DefinesPath:         "common/defines/hagane_defines.lua",
		StructuredBookmarks: []BookmarkEvidence{{ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true}},
		RawDefines: []RawDefinesEvidence{
			{ID: "start-owner", Path: "common/defines/y.lua", SourceHash: "start-hash", Emitted: true, StartDate: "1939.1.1"},
			{ID: "end-owner", Path: "common/defines/z.lua", SourceHash: "end-hash", Emitted: true, EndDate: "1976.12.31"},
		},
	})
	if plan.RawDefinesWinner == nil || plan.RawDefinesWinner.StartPath != "common/defines/y.lua" ||
		plan.RawDefinesWinner.EndPath != "common/defines/z.lua" {
		t.Fatalf("winner=%#v", plan.RawDefinesWinner)
	}
	fingerprint := plan.FingerprintFields()
	if fingerprint.RawDefinesStartWinnerPath != "common/defines/y.lua" ||
		fingerprint.RawDefinesStartWinnerHash != "start-hash" ||
		fingerprint.RawDefinesEndWinnerPath != "common/defines/z.lua" ||
		fingerprint.RawDefinesEndWinnerHash != "end-hash" {
		t.Fatalf("fingerprint=%#v", fingerprint)
	}
	for _, finding := range plan.Findings {
		if finding.Code == "timeline.raw_defines_start_date_conflict" && finding.Path != "common/defines/y.lua" {
			t.Fatalf("start finding points at wrong owner: %#v", finding)
		}
	}
}

func TestPlanHashIsStableAndSemantic(t *testing.T) {
	input := Input{
		ProjectStartDate: "1964.5.1",
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
	}
	first := Build(input)
	second := Build(input)
	if first.PlanSHA256 == "" || first.PlanSHA256 != second.PlanSHA256 {
		t.Fatalf("hashes differ: %q %q", first.PlanSHA256, second.PlanSHA256)
	}
	input.RequestedEndDate = "1976.12.31"
	changed := Build(input)
	if changed.PlanSHA256 == first.PlanSHA256 {
		t.Fatal("a behavior-affecting requested end date must change the plan hash")
	}
}

func TestExtractDefineDatesUsesLastUncommentedAssignment(t *testing.T) {
	start, end := ExtractDefineDates("-- NDefines.NGame.START_DATE = \"1936.1.1\"\n" +
		"NDefines.NGame.START_DATE = '1964.5.1'\n" +
		"NDefines.NGame.START_DATE = \"1965.1.1\"\n" +
		"NDefines.NGame.END_DATE = \"1976.12.31\"\n")
	if start != "1965.1.1" || end != "1976.12.31" {
		t.Fatalf("start=%q end=%q", start, end)
	}
}

func hasFinding(plan Plan, code string) bool {
	for _, finding := range plan.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}
