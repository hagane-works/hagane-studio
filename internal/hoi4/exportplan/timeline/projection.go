package timeline

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

const (
	FingerprintSchemaV1 = "hoi4.timeline-fingerprint.v1"
	ArtifactSchemaV1    = "hoi4.timeline-artifact.v1"
)

type FingerprintFields struct {
	Schema                    string   `json:"schema"`
	TimelinePlanSHA256        string   `json:"timeline_plan_sha256"`
	EffectiveStartDate        string   `json:"effective_start_date,omitempty"`
	EffectiveEndDate          string   `json:"effective_end_date,omitempty"`
	EndDateSource             string   `json:"end_date_source,omitempty"`
	EmitsBookmark             bool     `json:"emits_bookmark"`
	EmitDefines               bool     `json:"emit_defines"`
	DefinesPath               string   `json:"defines_path"`
	DefinesContentSHA256      string   `json:"defines_content_sha256,omitempty"`
	EffectiveReplacePaths     []string `json:"effective_replace_paths"`
	RawDefinesWinnerPath      string   `json:"raw_defines_winner_path,omitempty"`
	RawDefinesWinnerHash      string   `json:"raw_defines_winner_hash,omitempty"`
	RawDefinesStartWinnerPath string   `json:"raw_defines_start_winner_path,omitempty"`
	RawDefinesStartWinnerHash string   `json:"raw_defines_start_winner_hash,omitempty"`
	RawDefinesEndWinnerPath   string   `json:"raw_defines_end_winner_path,omitempty"`
	RawDefinesEndWinnerHash   string   `json:"raw_defines_end_winner_hash,omitempty"`
}

type ArtifactMetadata struct {
	Schema             string            `json:"schema"`
	TimelinePlanSHA256 string            `json:"timeline_plan_sha256"`
	Fingerprint        FingerprintFields `json:"fingerprint"`
}

type PlayerProblem struct {
	Code              string `json:"code"`
	Severity          string `json:"severity"`
	Headline          string `json:"headline"`
	Consequence       string `json:"consequence"`
	RecommendedAction string `json:"recommended_action"`
}

type ExportIssuesProjection struct {
	TimelinePlanSHA256 string          `json:"timeline_plan_sha256"`
	Problems           []PlayerProblem `json:"problems"`
}

func (p Plan) FingerprintFields() FingerprintFields {
	fields := FingerprintFields{
		Schema:                FingerprintSchemaV1,
		TimelinePlanSHA256:    p.PlanSHA256,
		EffectiveStartDate:    p.EffectiveStartDate,
		EffectiveEndDate:      p.EffectiveEndDate,
		EndDateSource:         p.EndDateSource,
		EmitsBookmark:         p.EmitsBookmark,
		EmitDefines:           p.EmitDefines,
		DefinesPath:           p.Defines.Path,
		EffectiveReplacePaths: append([]string(nil), p.EffectiveReplacePaths...),
	}
	if p.EmitDefines {
		sum := sha256.Sum256([]byte(p.Defines.Content))
		fields.DefinesContentSHA256 = hex.EncodeToString(sum[:])
	}
	if p.RawDefinesWinner != nil {
		fields.RawDefinesWinnerPath = p.RawDefinesWinner.Path
		fields.RawDefinesWinnerHash = p.RawDefinesWinner.SourceHash
		fields.RawDefinesStartWinnerPath = p.RawDefinesWinner.StartPath
		fields.RawDefinesStartWinnerHash = p.RawDefinesWinner.StartSourceHash
		fields.RawDefinesEndWinnerPath = p.RawDefinesWinner.EndPath
		fields.RawDefinesEndWinnerHash = p.RawDefinesWinner.EndSourceHash
	}
	return fields
}

func (p Plan) ArtifactMetadata() ArtifactMetadata {
	return ArtifactMetadata{
		Schema:             ArtifactSchemaV1,
		TimelinePlanSHA256: p.PlanSHA256,
		Fingerprint:        p.FingerprintFields(),
	}
}

func (p Plan) ExportIssuesProjection() ExportIssuesProjection {
	problems := make([]PlayerProblem, 0, len(p.Findings))
	for _, finding := range p.Findings {
		problems = append(problems, playerProblemForFinding(finding))
	}
	sort.SliceStable(problems, func(i, j int) bool {
		if problems[i].Severity != problems[j].Severity {
			return playerSeverityRank(problems[i].Severity) < playerSeverityRank(problems[j].Severity)
		}
		return problems[i].Code < problems[j].Code
	})
	return ExportIssuesProjection{
		TimelinePlanSHA256: p.PlanSHA256,
		Problems:           problems,
	}
}

func playerProblemForFinding(finding Finding) PlayerProblem {
	problem := PlayerProblem{Code: finding.Code, Severity: finding.Severity}
	switch finding.Code {
	case "timeline.end_date_defaulted":
		problem.Headline = "Campaign end date is temporary"
		problem.Consequence = "The campaign can start, but its intended length has not been set."
		problem.RecommendedAction = "Set the campaign end date."
	case "timeline.base_timeline_source_unregistered":
		problem.Headline = "Base Mod campaign end is not registered"
		problem.Consequence = "The program cannot safely decide when this campaign should end."
		problem.RecommendedAction = "Check the Base Mod timeline and set this scenario's end date."
	case "timeline.requested_end_date_invalid":
		problem.Headline = "Campaign end date needs attention"
		problem.Consequence = "The chosen end date is not after the campaign starts."
		problem.RecommendedAction = "Choose an end date after the campaign start date."
	case "timeline.custom_start_without_bookmark":
		problem.Headline = "No starting scenario for this campaign date"
		problem.Consequence = "Players may not have a campaign entry they can select for this date."
		problem.RecommendedAction = "Create a starting scenario for the campaign date."
	case "timeline.project_earliest_bookmark_mismatch":
		problem.Headline = "Campaign dates do not match"
		problem.Consequence = "The project date and the earliest selectable scenario lead to different starting worlds."
		problem.RecommendedAction = "Align the campaign start date and starting scenario date."
	case "timeline.structured_bookmark_date_unknown", "timeline.raw_bookmark_date_unknown":
		problem.Headline = "Starting scenario date cannot be read"
		problem.Consequence = "The campaign cannot be checked against the selected starting scenario."
		problem.RecommendedAction = "Repair the starting scenario date."
	case "timeline.bookmark_path_shadow_date_mismatch":
		problem.Headline = "Two starting scenarios compete for the same slot"
		problem.Consequence = "The final campaign may use a different date from the one shown in project content."
		problem.RecommendedAction = "Keep one starting scenario for this campaign slot."
	case "timeline.raw_defines_start_date_conflict":
		problem.Headline = "Campaign rules use a different start date"
		problem.Consequence = "The selected scenario and campaign clock may begin on different dates."
		problem.RecommendedAction = "Align the campaign rules with the starting scenario."
	case "timeline.raw_defines_end_date_missing_or_invalid":
		problem.Headline = "Campaign rules have no playable end date"
		problem.Consequence = "The game can finish as soon as the campaign loads."
		problem.RecommendedAction = "Set an end date after the campaign start date."
	case "timeline.raw_defines_end_date_conflict":
		problem.Headline = "Campaign rules use a different end date"
		problem.Consequence = "The campaign may finish on a different date from the one set for the scenario."
		problem.RecommendedAction = "Align the campaign rules with the campaign end date."
	case "timeline.project_start_date_unknown":
		problem.Headline = "Campaign start date cannot be read"
		problem.Consequence = "The program cannot prepare the campaign clock or verify the starting scenario."
		problem.RecommendedAction = "Set a valid campaign start date."
	case "timeline.tension_scale_not_reviewed":
		problem.Headline = "World tension pacing has not been reviewed"
		problem.Consequence = "A non-standard campaign date can change the expected pace of world tension."
		problem.RecommendedAction = "Review world tension pacing before playtesting."
	default:
		problem.Headline = "Campaign setup needs attention"
		problem.Consequence = "Part of the campaign setup could lead to a different world from the one you planned."
		problem.RecommendedAction = "Review the campaign settings and starting scenario."
	}
	return problem
}

func playerSeverityRank(severity string) int {
	switch severity {
	case "attention":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}
