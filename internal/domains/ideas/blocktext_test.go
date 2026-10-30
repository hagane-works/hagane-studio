package ideas

import (
	"errors"
	"strings"
	"testing"

	"haganeworks/internal/workbench"
)

func TestBlockTextPolicyNormalizesUnquotedLiteralNewline(t *testing.T) {
	got, issues, err := canonicalBlockText("add_stability = 0.1\\nset_country_flag = ready")
	if err != nil {
		t.Fatal(err)
	}
	if got != "add_stability = 0.1\nset_country_flag = ready" {
		t.Fatalf("unexpected canonical block: %q", got)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "后续内容不生效") {
		t.Fatalf("missing creator-facing issue: %#v", issues)
	}
}

func TestBlockTextPolicyPreservesComments(t *testing.T) {
	src := "add_stability = 0.1 # 文档示例 \\n 不应改写\nset_country_flag = ready"
	result := applyBlockTextPolicy(src)
	if result.Value != src || result.Normalized != 0 || result.Rejected || len(result.Issues) != 0 {
		t.Fatalf("comment text changed: %#v", result)
	}
}

func TestBlockTextPolicyRejectsQuotedAndEscapedAmbiguity(t *testing.T) {
	for _, src := range []string{`custom_effect_tooltip = "a\nb"`, `path = foo\\\nbar`} {
		got, issues, err := canonicalBlockText(src)
		if !errors.Is(err, errAmbiguousBlockText) {
			t.Fatalf("%q: expected reject, got %v", src, err)
		}
		if got != src || len(issues) != 1 {
			t.Fatalf("%q: rejected input must remain unchanged: %q %#v", src, got, issues)
		}
	}
}

func TestCanonicalBlockTextFieldsNamesRepairLocation(t *testing.T) {
	fields, issues, err := canonicalBlockTextFields(map[string]string{
		"allowed": "tag = GER",
		"on_add":  "set_country_flag = a\\nadd_stability = 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fields["on_add"] != "set_country_flag = a\nadd_stability = 1" {
		t.Fatalf("field was not normalized: %#v", fields)
	}
	if len(issues) != 1 || !strings.HasPrefix(issues[0].Repair, "on_add:") {
		t.Fatalf("repair does not name the field: %#v", issues)
	}
}

func TestCanonicalIdeaCreateAndPatchCoverAllScriptText(t *testing.T) {
	in := createInput{
		Allowed: "original_tag = GER\\nhas_war = no", AllowedCivilWar: "always = yes\\nhas_government = fascism",
		Available: "has_war = no\\nis_ai = no", DoEffect: "add_stability = 0.1\\nadd_war_support = 0.1",
		OnAdd: "set_country_flag = demo\\nadd_political_power = 25", NameZh: `精神\n名称`,
	}
	if err := canonicalIdeaCreate(&in); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"allowed": in.Allowed, "allowed_civil_war": in.AllowedCivilWar,
		"available": in.Available, "do_effect": in.DoEffect, "on_add": in.OnAdd,
	} {
		if strings.Contains(value, `\n`) || !strings.Contains(value, "\n") {
			t.Errorf("%s was not canonicalised: %q", name, value)
		}
	}
	if in.NameZh != `精神\n名称` {
		t.Fatal("idea display text changed")
	}

	value := "clr_country_flag = demo\\nadd_stability = -0.1"
	patch := patchInput{OnRemove: workbench.Optional[string]{Set: true, Value: &value}}
	if err := canonicalIdeaPatch(&patch); err != nil {
		t.Fatal(err)
	}
	if *patch.OnRemove.Value != "clr_country_flag = demo\nadd_stability = -0.1" {
		t.Fatalf("idea patch not canonicalised: %q", *patch.OnRemove.Value)
	}
	if patch.OnAdd.Set {
		t.Fatal("an absent field became set")
	}
}
