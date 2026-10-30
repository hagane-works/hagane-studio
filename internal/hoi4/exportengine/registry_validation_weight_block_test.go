package exportengine

import (
	"strings"
	"testing"
)

func TestRegistryValidationFocusTreeOwnershipWeightIsNotModifierKeys(t *testing.T) {
	useCWTools(t)

	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/national_focus/demo.txt", "focus_tree = {\n id = demo_focus\n country = {\n  factor = 0\n\n  modifier = {\n   add = 10\n   tag = GER\n  }\n }\n default = no\n}\n"),
	), english)
	got := registryFindingCodes(report)
	if strings.Contains(got, "modifier.unknown") {
		t.Fatalf("focus-tree ownership weight reported as unknown modifiers:\n%s", got)
	}
}

func TestRegistryValidationWeightBlockTriggersAreNotModifierKeys(t *testing.T) {
	useCWTools(t)

	for _, tc := range []struct {
		name string
		rel  string
		text string
	}{
		{

			name: "ai_will_do factor parent",
			rel:  "common/national_focus/demo.txt",
			text: "focus_tree = {\n id = demo\n focus = {\n  id = demo_focus\n  ai_will_do = {\n   factor = 1\n   modifier = {\n    factor = 0\n    original_tag = GER\n    has_government = fascism\n    has_country_flag = demo_flag\n   }\n  }\n }\n}\n",
		},
		{

			name: "base parent",
			rel:  "common/decisions/demo.txt",
			text: "demo_category = {\n demo_decision = {\n  ai_will_do = {\n   base = 10\n   modifier = {\n    add = -10\n    has_war = yes\n    date > 1939.1.1\n   }\n  }\n }\n}\n",
		},
		{

			name: "random_list arm",
			rel:  "common/decisions/random.txt",
			text: "demo_category = {\n demo_decision = {\n  complete_effect = {\n   random_list = {\n    25 = {\n     modifier = {\n      factor = 0\n      has_government = communism\n     }\n     set_country_flag = demo_students\n    }\n   }\n  }\n }\n}\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := registryFindingCodes(ScanRegistryValidation(registryArtifact(registryEntry(tc.rel, tc.text)), english))
			if strings.Contains(got, "modifier.unknown") {
				t.Fatalf("weight-block trigger reported as unknown modifier:\n%s", got)
			}
		})
	}
}

func TestRegistryValidationRealModifierTypoStillReportedNextToWeightBlocks(t *testing.T) {
	useCWTools(t)
	report := ScanRegistryValidation(registryArtifact(

		registryEntry("common/ideas/demo.txt", "ideas = {\n country = {\n  demo_spirit = {\n   picture = generic_spirit\n   modifier = {\n    politcal_power_factor = 0.1\n   }\n   ai_will_do = {\n    factor = 1\n    modifier = {\n     factor = 0\n     original_tag = GER\n    }\n   }\n  }\n }\n}\n"),
	), english)
	got := registryFindingCodes(report)
	if !strings.Contains(got, "modifier.unknown:politcal_power_factor") {
		t.Fatalf("misspelled modifier key was silently accepted:\n%s", got)
	}

	for _, quiet := range []string{"modifier.unknown:factor", "modifier.unknown:original_tag"} {
		if strings.Contains(got, quiet) {
			t.Fatalf("weight operand %s reported alongside the real typo:\n%s", quiet, got)
		}
	}
}

func TestRegistryValidationDynamicModifierDefinitionStillChecked(t *testing.T) {
	useCWTools(t)
	got := registryFindingCodes(ScanRegistryValidation(registryArtifact(
		registryEntry("common/dynamic_modifiers/demo.txt", "demo_dynamic = {\n icon = GFX_x\n political_power_factor = 0.1\n not_a_real_modifier_key = 1\n}\n"),
	), english))
	if !strings.Contains(got, "modifier.unknown:not_a_real_modifier_key") {
		t.Fatalf("dynamic modifier definition escaped the catalogue check:\n%s", got)
	}
	if strings.Contains(got, "modifier.unknown:political_power_factor") {
		t.Fatalf("valid modifier rejected:\n%s", got)
	}
}

func TestRegistryValidationCustomModifierTooltipIsNotAModifierKey(t *testing.T) {
	useCWTools(t)
	report := ScanRegistryValidation(registryArtifact(
		registryEntry("common/ideas/demo.txt", "ideas = {\n country = {\n  demo_spirit = {\n   modifier = {\n    hidden_modifier = {\n     political_power_gain = 0.1\n    }\n    custom_modifier_tooltip = demo_spirit_tt\n    political_power_factor = 0.1\n   }\n  }\n }\n}\n"),
	), english)
	got := registryFindingCodes(report)
	if strings.Contains(got, "modifier.unknown:custom_modifier_tooltip") {
		t.Fatalf("documented tooltip key reported as an unknown modifier:\n%s", got)
	}

	report = ScanRegistryValidation(registryArtifact(
		registryEntry("common/ideas/demo.txt", "ideas = {\n country = {\n  demo_spirit = {\n   modifier = {\n    custom_modifier_tooltip = demo_spirit_tt\n    politcal_power_factor = 0.1\n   }\n  }\n }\n}\n"),
	), english)
	if got = registryFindingCodes(report); !strings.Contains(got, "modifier.unknown:politcal_power_factor") {
		t.Fatalf("typo beside an exempt tooltip key was silently accepted:\n%s", got)
	}
}
