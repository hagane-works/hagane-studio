package exportcheck

import (
	"haganeworks/internal/hoi4/clausewitz"
)

func (p RewardRepairPlan) ToDict() map[string]any {
	ops := p.Operations
	if ops == nil {
		ops = []map[string]any{}
	}
	var after any
	if p.PreviewAfter != nil {
		after = *p.PreviewAfter
	}
	return map[string]any{
		"repair_id":             p.RepairID,
		"finding_code":          p.FindingCode,
		"repair_route":          p.RepairRoute,
		"source":                p.Source,
		"operations":            ops,
		"preview_before":        p.PreviewBefore,
		"preview_after":         after,
		"transactional":         p.Transactional,
		"requires_confirmation": p.RequiresConfirmation,
		"applicable":            p.Applicable,
	}
}

func (a RewardAnalysis) Blocking() bool {
	for _, f := range a.Findings {
		if f.Severity == "blocker" {
			return true
		}
	}
	return false
}

func (a RewardAnalysis) ToDict() map[string]any {
	assignments := []map[string]any{}
	for _, x := range a.Assignments {
		assignments = append(assignments, x.ToMap())
	}
	findings := []clausewitz.AnalysisFinding{}
	findings = append(findings, a.Findings...)
	plans := []map[string]any{}
	for _, p := range a.RepairPlans {
		plans = append(plans, p.ToDict())
	}
	return map[string]any{
		"assignments":   assignments,
		"findings":      findings,
		"repair_plans":  plans,
		"source_sha256": a.SourceSHA256,
		"blocking":      a.Blocking(),
	}
}
