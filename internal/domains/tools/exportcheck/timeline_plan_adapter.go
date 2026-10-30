package exportcheck

import (
	"haganeworks/internal/hoi4/exportplan/history"
	"haganeworks/internal/hoi4/exportplan/timeline"
	"strings"
)

func BuildSharedTimelinePlanForExport(
	project ExportGateProject, bookmarks []ExportGateBookmark,
	rawFiles []FocusGateRawFile, requestedReplacePaths []string,
) timeline.Plan {
	structured := make([]timeline.BookmarkEvidence, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		id := strings.TrimSpace(bookmark.BookmarkID)
		if id == "" {
			continue
		}
		source := strings.TrimSpace(bookmark.Source)
		if source == "" {
			source = "project_structured"
		}
		structured = append(structured, timeline.BookmarkEvidence{
			ID: id, Path: timeline.BookmarksPath + "/" + id + ".txt",
			Source: source, DateTokens: []string{bookmark.Date}, Emitted: true,
		})
	}

	modDirectory := modArtifactName(project)
	definesPath := timeline.DefinesPathForModDirectory(modDirectory)
	rawBookmarks := make([]timeline.BookmarkEvidence, 0)
	rawDefines := make([]timeline.RawDefinesEvidence, 0)
	for _, row := range rawFiles {
		path := SafeRawPath(row.DestPath)
		if path == "" || row.IsBinary {
			continue
		}
		source := strings.TrimSpace(row.Source)
		if source == "" {
			source = "project_preserved_raw"
		}
		if strings.HasPrefix(strings.ToLower(path), timeline.BookmarksPath+"/") &&
			strings.HasSuffix(strings.ToLower(path), ".txt") {
			tokens, _ := ExtractRawBookmarkDateTokens(row.Content)
			id := strings.TrimSpace(anyToString(row.ID))
			if id == "" {
				id = path
			}
			rawBookmarks = append(rawBookmarks, timeline.BookmarkEvidence{
				ID: id, Path: path, Source: source, SourceHash: row.ContentSHA256,
				DateTokens: tokens, Emitted: true,
			})
		}
		lowerPath := strings.ToLower(path)
		if strings.HasPrefix(lowerPath, "common/defines/") && strings.HasSuffix(lowerPath, ".lua") {
			id := strings.TrimSpace(anyToString(row.ID))
			if id == "" {
				id = path
			}
			rawDefines = append(rawDefines, timeline.RawDefinesEvidence{
				ID: id, Path: path, Source: source, SourceHash: row.ContentSHA256,
				Content: row.Content, Emitted: true,
			})
		}
	}

	return timeline.Build(timeline.Input{
		ProjectStartDate:      project.StartDate,
		RequestedEndDate:      project.EndDate,
		HasDependencies:       len(project.Dependencies) > 0 || project.BasemodTemplateID != nil,
		StructuredBookmarks:   structured,
		RawBookmarks:          rawBookmarks,
		RawDefines:            rawDefines,
		RequestedReplacePaths: append([]string(nil), requestedReplacePaths...),
		ModDirectory:          modDirectory,
		DefinesPath:           definesPath,
	})
}

func timelineProjectionFindings(projection timeline.ExportIssuesProjection) []map[string]any {
	findings := make([]map[string]any, 0, len(projection.Problems))
	for _, problem := range projection.Problems {

		switch problem.Code {
		case "timeline.end_date_defaulted", "timeline.base_timeline_source_unregistered", "timeline.requested_end_date_invalid",
			"timeline.raw_defines_end_date_missing_or_invalid", "timeline.raw_defines_end_date_conflict":
		default:
			continue
		}
		severity := problem.Severity
		if severity == "attention" {
			severity = "warning"
		}
		findings = append(findings, gateFinding(
			problem.Code, severity,
			findingSourceMap("timeline_plan", projection.TimelinePlanSHA256, "", "project"),
			map[string]any{
				"timeline_plan_sha256": projection.TimelinePlanSHA256,
				"headline":             problem.Headline,
				"consequence":          problem.Consequence,
				"recommended_action":   problem.RecommendedAction,
			},
			"timeline.review_campaign_setup",
		))
	}
	return findings
}

func BuildScenarioHistoryReadOnlyPlan(
	project ExportGateProject, timelinePlan timeline.Plan,
) history.ScenarioHistoryPlan {
	mode := history.InheritBaseWorld
	if project.ScenarioHistoryPolicy == ScenarioHistoryPolicyProjectOwnsStartingWorld {
		mode = history.UseMyScenarioWorld
	}
	standalone := len(project.Dependencies) == 0 && project.BasemodTemplateID == nil
	return history.BuildScenarioHistoryPlan(history.AnalysisInput{
		WorldMode:          mode,
		EffectiveStartDate: timelinePlan.EffectiveStartDate,
		Provider: history.ProviderEvidence{
			Kind: history.ProviderVanilla, Standalone: standalone,
		},
	})
}

func scenarioHistoryProjectionFindings(plan history.ScenarioHistoryPlan) []map[string]any {
	findings := make([]map[string]any, 0, len(plan.Findings))
	for _, finding := range plan.Findings {
		severity := finding.Severity
		if severity == "attention" {
			severity = "warning"
		}
		findings = append(findings, gateFinding(
			finding.Code, severity,
			findingSourceMap("scenario_history_preview", finding.Tag, finding.Path, "read_only"),
			map[string]any{
				"headline": finding.Title, "consequence": finding.Reason,
				"recommended_action":   finding.Suggestion,
				"effective_start_date": plan.EffectiveStartDate,
				"risks_found":          len(plan.Preview.Risks),
			},
			"scenario.review_starting_world",
		))
	}
	return findings
}
