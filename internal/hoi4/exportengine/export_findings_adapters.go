package exportengine

import (
	"encoding/json"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/exportplan/timeline"
)

func ExportFindingsFromTimelinePlan(plan timeline.Plan, acknowledged bool) []ExportFinding {
	problems := map[string]timeline.PlayerProblem{}
	for _, problem := range plan.ExportIssuesProjection().Problems {
		problems[problem.Code] = problem
	}
	out := make([]ExportFinding, 0, len(plan.Findings))
	for _, item := range plan.Findings {
		path, line := findingLocation(item.Path, 0)
		identifier := item.SourceID
		if identifier == "" {
			identifier = item.Code
		}
		problem := problems[item.Code]
		out = append(out, ExportFinding{
			Code: item.Code, Severity: item.Severity,
			Source: findingSource(item.SourceKind, identifier),
			Path:   path, Line: line, Message: problem.Headline,
			PlayerImpact: problem.Consequence, EditRoute: problem.RecommendedAction,
			Acknowledged:   acknowledged,
			SourceEvidence: map[string]any{"expected": item.Expected, "actual": item.Actual},
		})
	}
	return out
}

func findingSource(kind, identifier string) json.RawMessage {
	body, _ := json.Marshal(map[string]any{"kind": kind, "identifier": identifier})
	return body
}

func findingLocation(path string, line int) (*string, *int) {
	var pathOut *string
	var lineOut *int
	if path != "" {
		copy := path
		pathOut = &copy
	}
	if line > 0 {
		copy := line
		lineOut = &copy
	}
	return pathOut, lineOut
}

func ExportFindingsFromFinalSyntax(report *FinalArtifactSyntaxReport, acknowledged bool) []ExportFinding {
	if report == nil {
		return nil
	}
	out := make([]ExportFinding, 0, len(report.Findings))
	for _, item := range report.Findings {
		path, line := findingLocation(item.Path, item.Line)
		out = append(out, ExportFinding{
			Code: item.Code, Severity: item.Severity,
			Source: findingSource("final_artifact_syntax", item.Code),
			Path:   path, Line: line, Message: item.Detail,
			PlayerImpact: item.PlayerImpact, EditRoute: item.SuggestedEdit,
			Acknowledged:   acknowledged,
			SourceEvidence: map[string]any{"snippet": item.Snippet},
		})
	}
	return out
}

func ExportFindingsFromReferenceIntegrity(report ReferenceIntegrityReport, acknowledged bool) []ExportFinding {
	out := make([]ExportFinding, 0, len(report.Findings))
	for _, item := range report.Findings {
		path, line := findingLocation(item.File, item.Line)
		identifier := item.Target
		if identifier == "" {
			identifier = item.Code
		}
		severity := item.Severity
		if severity == "" {
			severity = "review"
		}
		out = append(out, ExportFinding{
			Code: item.Code, Severity: severity,
			Source: findingSource("reference_integrity", identifier),
			Path:   path, Line: line, Message: item.Summary,
			PlayerImpact: item.Consequence, EditRoute: item.EditAt,
			Acknowledged:   acknowledged,
			SourceEvidence: map[string]any{"target": item.Target},
		})
	}
	return out
}

func ExportFindingsFromRegistryValidation(report RegistryReport, acknowledged bool) []ExportFinding {
	out := make([]ExportFinding, 0, len(report.Findings))
	for _, item := range report.Findings {
		path, line := findingLocation(item.Path, item.Line)
		identifier := item.Token
		if identifier == "" {
			identifier = item.Code
		}
		out = append(out, ExportFinding{
			Code:   item.Code,
			Source: findingSource("registry_validation", identifier),
			Path:   path, Line: line, Message: item.Text,
			PlayerImpact: "", EditRoute: "",
			Acknowledged:   acknowledged,
			SourceEvidence: map[string]any{"status": item.Status, "token": item.Token},
		})
	}
	return out
}

func ExportFindingsFromIdeaPictureDrops(drops []IdeaPictureDrop, acknowledged bool, msg exportmsg.Printer) []ExportFinding {
	out := make([]ExportFinding, 0, len(drops))
	for _, drop := range drops {
		identifier := drop.IdeaID
		if identifier == "" {
			identifier = drop.Picture
		}
		evidence := map[string]any{"idea_id": drop.IdeaID, "picture": drop.Picture, "sprite_name": drop.Stripped}
		if drop.RowID > 0 {
			evidence["idea_row_id"] = drop.RowID
		}
		if len(drop.Suggestions) > 0 {
			evidence["suggestions"] = drop.Suggestions
		}
		path, line := findingLocation("", 0)
		out = append(out, ExportFinding{
			Code: IdeaPictureDroppedCode, Severity: "warning",
			Source: findingSource("idea_picture_resolve", identifier),
			Path:   path, Line: line, Message: drop.Message(msg),
			PlayerImpact: drop.PlayerImpact(msg), EditRoute: drop.EditRoute(msg),
			Acknowledged:   acknowledged,
			SourceEvidence: evidence,
		})
	}
	return out
}

func ExportFindingsFromLiteralNewlines(report LiteralNewlineReport, acknowledged bool) []ExportFinding {
	out := make([]ExportFinding, 0, len(report.Hits))
	for _, item := range report.Hits {
		path, line := findingLocation(item.Path, item.Line)
		out = append(out, ExportFinding{
			Code: "script.literal_newline", Severity: "warning",
			Source: findingSource("literal_newline_lint", item.Path),
			Path:   path, Line: line,
			Message:        report.msg.Text("newline.finding.message", exportmsg.Args{"column": item.Column, "excerpt": item.Excerpt}),
			PlayerImpact:   report.msg.Text("newline.finding.impact", nil),
			EditRoute:      report.msg.Text("newline.finding.edit", nil),
			Acknowledged:   acknowledged,
			SourceEvidence: map[string]any{"column": item.Column, "excerpt": item.Excerpt},
		})
	}
	return out
}
