package exportcheck

import (
	"context"
	"errors"
	"fmt"
	"haganeworks/internal/hoi4/politicsrefs"
	"haganeworks/internal/hoi4/runcheck"
	"sort"
	"strings"
)

const politicsSourceMissingCode = "politics_source_snapshot_missing"

const politicsSourceStaleCode = "politics_source_snapshot_stale"

func (s *Store) PoliticsLifecycleFindings(ctx context.Context, projectID int64) ([]runcheck.Finding, error) {
	project, found, err := s.GetMetadata(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrProjectMissing
	}
	snapshot, err := s.PoliticsReferences(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return CollectPoliticsLifecycleFindings(project, snapshot)
}

func CollectPoliticsLifecycleFindings(project Project, snapshot politicsrefs.Snapshot) ([]runcheck.Finding, error) {
	findings, err := runcheck.CollectPolitics(snapshot)
	if err != nil {
		return nil, err
	}
	if !hasInheritedProjectSubtype(snapshot) {
		return findings, nil
	}
	state := strings.ToLower(strings.TrimSpace(anyToString(project.IdeologySourceManifest["state"])))
	sourceKey := strings.TrimSpace(anyToString(project.IdeologySourceManifest["source_key"]))
	code := ""
	switch {
	case len(project.IdeologySourceManifest) == 0 || sourceKey == "":
		code = politicsSourceMissingCode
	case state != "ready":
		code = politicsSourceStaleCode
	}
	if code != "" {
		findings = append(findings, politicsSourceLifecycleFinding(code, state, sourceKey))
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == "blocker"
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return fmt.Sprint(findings[i].EntityRef["id"]) < fmt.Sprint(findings[j].EntityRef["id"])
	})
	return findings, nil
}

func hasInheritedProjectSubtype(snapshot politicsrefs.Snapshot) bool {
	roots := map[string]politicsrefs.SourceOwnership{}
	for _, ideology := range snapshot.Ideologies {
		if ideology.Kind == politicsrefs.KindRoot {
			roots[strings.TrimSpace(ideology.ScriptID)] = ideology.Source
		}
	}
	for _, ideology := range snapshot.Ideologies {
		if ideology.Kind == politicsrefs.KindSubtype && ideology.Source == politicsrefs.SourceProjectManaged &&
			roots[strings.TrimSpace(ideology.RootID)] != politicsrefs.SourceProjectManaged {
			return true
		}
	}
	return false
}

func politicsSourceLifecycleFinding(code, state, sourceKey string) runcheck.Finding {
	severity, automatic, reimport, route := "blocker", false, true, "politics.source.reimport"

	if code == politicsSourceStaleCode {
		severity, automatic, reimport, route = "warning", true, false, ""
	}
	return runcheck.Finding{
		Code: code, Severity: severity,
		EntityRef: map[string]any{"type": "project", "id": "politics"},
		Detail: map[string]any{
			"state": state, "source_key": sourceKey,
			"automatic_rewrite": automatic, "requires_reimport": reimport,
		},
		Source:      map[string]any{"kind": "go_run_check", "domain": "politics"},
		RepairRoute: route,
	}
}

func politicsExportIssues(project Project, snapshot politicsrefs.Snapshot) ([]map[string]any, error) {
	findings, err := CollectPoliticsLifecycleFindings(project, snapshot)
	if err != nil {
		return nil, err
	}
	issues := make([]map[string]any, 0, len(findings))
	for _, finding := range findings {
		message, hint, err := politicsFindingWording(finding)
		if err != nil {
			return nil, err
		}
		issues = append(issues, map[string]any{
			"level": severityLevel(finding.Severity), "severity": finding.Severity,
			"code": finding.Code, "entity_ref": finding.EntityRef,
			"detail": finding.Detail, "source": finding.Source,
			"repair_route": finding.RepairRoute, "message": message, "fix_hint": hint,
		})
	}
	return issues, nil
}

func severityLevel(severity string) string {
	if severity == "blocker" {
		return "error"
	}
	return "warning"
}

func politicsFindingWording(finding runcheck.Finding) (string, string, error) {
	custom := strings.TrimSpace(anyToString(finding.Detail["custom_root_id"]))
	expected := strings.TrimSpace(anyToString(finding.Detail["expected_root_id"]))
	switch finding.Code {
	case "politics_custom_root_no_random_subtype":
		return fmt.Sprintf("Custom political ROOT %q has no subtype available for random leader ideology selection.", custom),
			"Open Ideologies and leave at least one subtype under this ROOT eligible for random selection before exporting.", nil
	case "politics_custom_root_compatibility_blocker":
		return fmt.Sprintf("Custom political ROOT %q is incompatible with required military naming content that still expects %q.", custom, expected),
			"Open Political references and choose a compatible military name group; the program will not rewrite this automatically.", nil
	case "politics_custom_root_intentional_review":
		return fmt.Sprintf("Custom political ROOT %q is used beside an intentional has_government check for %q.", custom, expected),
			"Review the political condition and keep or edit it deliberately; the program will not rewrite it automatically.", nil
	case politicsSourceMissingCode:
		return "An inherited political subtype has no confirmed source snapshot, so its owner ROOT cannot be reproduced safely.",
			"Re-import the game or base Mod ideology source, then confirm the inherited subtype again.", nil
	case politicsSourceStaleCode:
		return "The inherited ideology files were imported again after these subtypes were confirmed.",
			"Nothing to do: the export applies the subtypes to the current ideology files again. If their ROOT is no longer there, it leaves them out and says so in the export notes.", nil
	default:
		return "", "", errors.New("unmapped politics diagnostic wording: " + finding.Code)
	}
}
