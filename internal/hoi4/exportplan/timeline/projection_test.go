package timeline

import (
	"reflect"
	"strings"
	"testing"
)

func TestFingerprintFieldsAndArtifactMetadataSharePlanSHA(t *testing.T) {
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		RequestedEndDate: "1976.12.31",
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
		RequestedReplacePaths: []string{"common/ideas"},
	})
	fields := plan.FingerprintFields()
	metadata := plan.ArtifactMetadata()
	if fields.Schema != FingerprintSchemaV1 || fields.TimelinePlanSHA256 != plan.PlanSHA256 {
		t.Fatalf("fingerprint=%#v plan=%#v", fields, plan)
	}
	if fields.DefinesContentSHA256 == "" || fields.DefinesPath != plan.Defines.Path {
		t.Fatalf("defines fingerprint=%#v", fields)
	}
	if metadata.Schema != ArtifactSchemaV1 || metadata.TimelinePlanSHA256 != plan.PlanSHA256 ||
		metadata.Fingerprint.TimelinePlanSHA256 != plan.PlanSHA256 {
		t.Fatalf("metadata=%#v", metadata)
	}
	if !reflect.DeepEqual(metadata.Fingerprint, fields) {
		t.Fatalf("artifact fingerprint=%#v fields=%#v", metadata.Fingerprint, fields)
	}
	fields.EffectiveReplacePaths[0] = "mutated"
	if plan.EffectiveReplacePaths[0] == "mutated" {
		t.Fatal("fingerprint fields must not alias plan replace paths")
	}
}

func TestRawWinnerFingerprintDoesNotRequireCentralRecalculation(t *testing.T) {
	path := "common/defines/fixture_defines.lua"
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		DefinesPath:      path,
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
		RawDefines: []RawDefinesEvidence{{
			ID: "winner", Path: path, SourceHash: "abc123", Emitted: true,
			StartDate: "1964.5.1", EndDate: "1970.1.1",
		}},
	})
	fields := plan.FingerprintFields()
	if plan.EmitDefines || fields.RawDefinesWinnerPath != path || fields.RawDefinesWinnerHash != "abc123" {
		t.Fatalf("raw winner fingerprint=%#v plan=%#v", fields, plan)
	}
	if fields.DefinesContentSHA256 != "" {
		t.Fatalf("suppressed generated defines must not have content hash: %#v", fields)
	}
}

func TestExportIssuesProjectionUsesGameLanguageAndSamePlanSHA(t *testing.T) {
	path := "common/defines/fixture_defines.lua"
	plan := Build(Input{
		ProjectStartDate: "1964.5.1",
		DefinesPath:      path,
		StructuredBookmarks: []BookmarkEvidence{{
			ID: "1964", DateTokens: []string{"1964.5.1"}, Emitted: true,
		}},
		RawDefines: []RawDefinesEvidence{{
			Path: path, Emitted: true, StartDate: "1939.1.1", EndDate: "1949.1.1",
		}},
	})
	projection := plan.ExportIssuesProjection()
	if projection.TimelinePlanSHA256 != plan.PlanSHA256 || len(projection.Problems) == 0 {
		t.Fatalf("projection=%#v plan=%#v", projection, plan)
	}
	var startConflict *PlayerProblem
	for i := range projection.Problems {
		problem := &projection.Problems[i]
		if problem.Code == "timeline.raw_defines_start_date_conflict" {
			startConflict = problem
		}
		if strings.Contains(strings.ToLower(problem.Headline+problem.Consequence+problem.RecommendedAction), "ndefines") ||
			strings.Contains(strings.ToLower(problem.Headline+problem.Consequence+problem.RecommendedAction), "raw") {
			t.Fatalf("player problem leaks technical wording: %#v", problem)
		}
	}
	if startConflict == nil || startConflict.Headline != "Campaign rules use a different start date" {
		t.Fatalf("start conflict=%#v", startConflict)
	}
}

func TestExportIssuesProjectionFallbackStaysActionable(t *testing.T) {
	plan := Plan{
		PlanSHA256: "fixture",
		Findings:   []Finding{{Code: "timeline.future_contract", Severity: "attention"}},
	}
	projection := plan.ExportIssuesProjection()
	if len(projection.Problems) != 1 || projection.Problems[0].RecommendedAction == "" ||
		projection.Problems[0].Consequence == "" {
		t.Fatalf("projection=%#v", projection)
	}
}
