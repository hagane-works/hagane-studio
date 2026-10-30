package runcheck

import (
	"sort"

	"haganeworks/internal/hoi4/politicsrefs"
)

func CollectPolitics(snapshot politicsrefs.Snapshot) ([]Finding, error) {
	impacts, err := politicsrefs.AnalyzeCustomRootImpacts(snapshot)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(impacts))
	for _, impact := range impacts {
		severity := "warning"
		if impact.Severity == politicsrefs.SeverityFatalBlocker {
			severity = "blocker"
		}
		findings = append(findings, Finding{
			Code: impact.Code,
			EntityRef: map[string]any{
				"type": impact.Entity.Kind,
				"id":   impact.Entity.ID,
				"path": impact.Entity.Path,
			},
			Severity: severity,
			Detail: map[string]any{
				"custom_root_id":       impact.CustomRootID,
				"expected_root_id":     impact.ExpectedRootID,
				"reference_key":        impact.ReferenceKey,
				"automatic_rewrite":    false,
				"requires_user_review": true,
			},
			Source:      map[string]any{"kind": "go_run_check", "domain": "politics"},
			RepairRoute: "politics.references",
		})
	}
	eligibleByRoot := map[string]bool{}
	for _, ideology := range snapshot.Ideologies {
		if ideology.Kind == politicsrefs.KindSubtype && !ideology.RandomDisabled {
			eligibleByRoot[ideology.RootID] = true
		}
	}
	for _, ideology := range snapshot.Ideologies {
		if ideology.Kind != politicsrefs.KindRoot || ideology.Source != politicsrefs.SourceProjectManaged || eligibleByRoot[ideology.ScriptID] {
			continue
		}
		findings = append(findings, Finding{
			Code: "politics_custom_root_no_random_subtype",
			EntityRef: map[string]any{
				"type": "ideology",
				"id":   ideology.ScriptID,
			},
			Severity: "blocker",
			Detail: map[string]any{
				"custom_root_id":       ideology.ScriptID,
				"automatic_rewrite":    false,
				"requires_user_review": true,
			},
			Source:      map[string]any{"kind": "go_run_check", "domain": "politics"},
			RepairRoute: "politics.ideologies",
		})
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == "blocker"
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return entityID(findings[i]) < entityID(findings[j])
	})
	return findings, nil
}
