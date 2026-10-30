package exportcheck

import "testing"

func TestSharedTimelinePlanProjectsEndDateWithItsPlanSHA(t *testing.T) {
	project := ExportGateProject{
		ID: 7, Name: "Campaign", CountryTag: "TST", StartDate: "1964.5.1",
	}
	bookmarks := []ExportGateBookmark{{BookmarkID: "1964", Date: "1964.5.1"}}
	findings, plan, err := CollectP0ExportFindings(CollectP0ExportFindingsInput{
		Project: project, Bookmarks: bookmarks,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlanSHA256 == "" || plan.EndDateSource != "compatibility_fallback" {
		t.Fatalf("plan=%#v", plan)
	}
	for _, finding := range findings {
		if finding["code"] != "timeline.end_date_defaulted" {
			continue
		}
		evidence, _ := finding["evidence"].(map[string]any)
		if evidence["timeline_plan_sha256"] != plan.PlanSHA256 {
			t.Fatalf("finding=%#v plan=%q", finding, plan.PlanSHA256)
		}
		return
	}
	t.Fatalf("end-date finding missing: %#v", findings)
}

func TestScenarioHistoryPreviewFailsClosedWithoutPinnedCorpus(t *testing.T) {
	project := ExportGateProject{
		ID: 7, Name: "Campaign", CountryTag: "TST", StartDate: "1964.5.1",
		ScenarioHistoryPolicy: ScenarioHistoryPolicyProjectOwnsStartingWorld,
	}
	plan := BuildSharedTimelinePlanForExport(project,
		[]ExportGateBookmark{{BookmarkID: "1964", Date: "1964.5.1"}}, nil, nil)
	historyPlan := BuildScenarioHistoryReadOnlyPlan(project, plan)
	if len(historyPlan.Findings) != 1 || historyPlan.Findings[0].Code != "scenario.history_corpus_missing" {
		t.Fatalf("history findings=%#v", historyPlan.Findings)
	}
}
