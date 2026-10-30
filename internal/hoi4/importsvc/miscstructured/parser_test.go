package miscstructured

import (
	"strings"
	"testing"
)

func TestStructuredParsers(t *testing.T) {
	mods, e := ModifierDefinitions(`my_mod={ color_type=good precision=1 category=all }`, `common/modifier_definitions/a.txt`)
	if e != nil || len(mods) != 1 || mods[0]["modifier_id"] != "my_mod" {
		t.Fatalf("modifier: %#v %v", mods, e)
	}
	plans, e := AIPlans(`test_plan={ name=TEST allowed={ always=yes } ai_strategy={ type=prepare_for_war id=GER value=10 } }`, `common/ai_strategy_plans/a.txt`)
	if e != nil || len(plans) != 1 {
		t.Fatalf("plan: %#v %v", plans, e)
	}
	templates, e := AITemplates(`test_role={ role=infantry target_a={ target_template=infantry } }`, `common/ai_templates/a.txt`)
	if e != nil || len(templates) != 1 {
		t.Fatalf("template: %#v %v", templates, e)
	}
	w, e := Wargoals(`wargoal_types={ test_goal={ war_name=TEST_WAR allowed={ always=yes } generator=1 } }`, `common/wargoals/a.txt`)
	if e != nil || len(w) != 1 || w[0]["wargoal_key"] != "test_goal" {
		t.Fatalf("wargoal: %#v %v", w, e)
	}
	if got := w[0]["allowed"]; got != "always = yes\n" {
		t.Fatalf("wargoal block-body contract changed: %#v", got)
	}
}

func TestAmbiguousFilesFailClosed(t *testing.T) {
	if _, e := ModifierDefinitions(`bad={ unknown=yes }`, "x"); e == nil {
		t.Fatal("unknown modifier field accepted")
	}
	if _, e := Wargoals(`test={}`, "x"); e == nil {
		t.Fatal("missing wrapper accepted")
	}
	if _, e := AITemplates(`x={ role=infantry }`, "common/ai_templates/x.txt"); e == nil {
		t.Fatal("target-less template accepted")
	}
}

func TestAIParsersKeepOuterBraceBlocks(t *testing.T) {
	plans, err := AIPlans(
		`test_plan={ allowed={ always=yes } }`,
		`common/ai_strategy_plans/a.txt`,
	)
	if err != nil || len(plans) != 1 {
		t.Fatalf("plan: %#v %v", plans, err)
	}
	allowed, ok := plans[0]["allowed"].([]string)
	if !ok || len(allowed) != 1 || allowed[0] != "{\n\talways = yes\n}" {
		t.Fatalf("plan allowed lost its outer block: %#v", plans[0]["allowed"])
	}

	templates, err := AITemplates(
		`test_role={ role=infantry available_for={ GER } target_a={ target_template=infantry } }`,
		`common/ai_templates/a.txt`,
	)
	if err != nil || len(templates) != 1 {
		t.Fatalf("template: %#v %v", templates, err)
	}
	if got, ok := templates[0]["available_for"].(string); !ok || got != "{\n\tGER\n}" {
		t.Fatalf("template available_for lost its outer block: %#v", templates[0]["available_for"])
	}
	targets, ok := templates[0]["target_templates"].([]map[string]string)
	if !ok || len(targets) != 1 || targets[0]["body"] != "{\n\ttarget_template = infantry\n}" {
		t.Fatalf("template target lost its outer block: %#v", templates[0]["target_templates"])
	}
}

func TestAITemplateFieldOrderUsesTheAPITargetMarkers(t *testing.T) {
	templates, err := AITemplates(`generic_suppression={ role=suppression upgrade_prio={ factor=1 } generic_suppression_template={ target_template=x } }`, "common/ai_templates/generic.txt")
	if err != nil || len(templates) != 1 {
		t.Fatalf("template: %#v %v", templates, err)
	}
	order, _ := templates[0]["field_order"].([]string)
	want := []string{"role", "upgrade_prio", "target:generic_suppression_template"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("field_order = %v, want %v", order, want)
	}
}
