package worldout

import (
	"strings"
	"testing"
)

func ideologyProject(t *testing.T, registryJSON string) *OMap {
	t.Helper()
	project := NewOMap()
	if err := project.UnmarshalJSON([]byte(`{"ideology_registry":` + registryJSON + `}`)); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	return project
}

func TestIdeologyExportPlanRequiresAllThreeSafetyConditions(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := ideologyProject(t, `{
		"groups":["democratic"],
		"sub_ideologies":[{"id":"demo_liberalism","group":"democratic","name":{"en":"Demo Liberalism"}}]
	}`)

	positive := BuildIdeologyExportPlan(tables, project, false, false)
	if !positive.Facts.RequiresReplacePath() || positive.FullText == "" {
		t.Fatalf("safe complete standalone plan not selected: %+v", positive.Facts)
	}
	for _, required := range []string{
		"\tdemocratic = {", "\tcommunism = {", "\tfascism = {",
		"\tneutrality = {", "\t\t\tdemo_liberalism = {",
	} {
		if !strings.Contains(positive.FullText, required) {
			t.Fatalf("complete file missing %q", required)
		}
	}

	for name, plan := range map[string]IdeologyExportPlan{
		"submod":       BuildIdeologyExportPlan(tables, project, true, false),
		"raw_override": BuildIdeologyExportPlan(tables, project, false, true),
		"no_full_file": BuildIdeologyExportPlan(&Tables{}, project, false, false),
	} {
		if plan.Facts.RequiresReplacePath() || plan.FullText != "" {
			t.Fatalf("negative control %s requested unsafe replacement: %+v", name, plan)
		}
	}
}

func TestIdeologyExportPlanCarriesValidatedCustomRootAndChild(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := ideologyProject(t, `{
		"groups":["new_order"],
		"root_definitions":[{
			"id":"new_order",
			"body":"types = { authored_sub = { can_be_randomly_selected = no } }\ncolor = { 1 2 3 }",
			"name":{"en":"New Order"}
		}],
		"sub_ideologies":[
			{"id":"authored_sub","group":"new_order","name":{"en":"Authored"}},
			{"id":"platform_sub","group":"new_order","name":{"en":"Platform"}}
		]
	}`)
	plan := BuildIdeologyExportPlan(tables, project, false, false)
	if !plan.Facts.RequiresReplacePath() {
		t.Fatalf("custom ROOT plan not complete: %+v", plan.Facts)
	}
	if strings.Count(plan.FullText, "\tnew_order = {") != 1 ||
		strings.Count(plan.FullText, "authored_sub = {") != 1 ||
		strings.Count(plan.FullText, "platform_sub = {") != 1 {
		t.Fatalf("custom ROOT/children not emitted exactly once:\n%s", plan.FullText)
	}
}

func TestIdeologyExportPlanRejectsMalformedCustomRoot(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := ideologyProject(t, `{
		"root_definitions":[{"id":"bad-root","body":"types = { }\ncolor = { 1 2 3 }"}]
	}`)
	plan := BuildIdeologyExportPlan(tables, project, false, false)
	if plan.Facts.RequiresReplacePath() || plan.FullText != "" {
		t.Fatalf("malformed ROOT must fail closed: %+v", plan)
	}
}
