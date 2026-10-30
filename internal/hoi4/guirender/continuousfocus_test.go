package guirender

import (
	"strings"
	"testing"
)

func TestContinuousFocusPanel(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/continuous_focus/generic.txt": []byte(`continuous_focus_palette = {
			id = generic_focus
			country = { factor = 1 }
			default = yes
			position = { x = 50 y = 1000 }
			focus = { id = continuous_naval_production icon = GFX_goal_continuous_naval_production available = { } daily_cost = 1 }
			focus = { id = continuous_boost_freedom available = { hidden_trigger = { has_dlc = "Together for Victory" } is_subject = yes } }
			focus = { id = SWI_militias available = { hidden_trigger = { has_dlc = "By Blood Alone" original_tag = SWI } } }
			focus = { id = ETH_lions available = { hidden_trigger = { original_tag = ETH } has_completed_focus = ETH_never_done OR = { is_government_in_exile = yes has_country_leader_with_trait = x } } }
			focus = { id = continuous_unknowable available = { has_country_leader_with_trait = x } }
		}`),
		"common/national_focus/germany.txt": []byte(`focus_tree = { id = german_focus
			country = { factor = 0 modifier = { add = 10 tag = GER } }
			continuous_focus_position = { x = 500 y = 1300 }
			focus = { id = GER_rearm x = 0 y = 0 } }`),
		"common/national_focus/generic.txt": []byte(`focus_tree = { id = generic_focus country = { factor = 1 } default = yes
			focus = { id = industrial_effort x = 0 y = 0 } }`),
		"localisation/english/cf_l_english.yml": []byte("l_english:\n continuous_naval_production:0 \"Naval Production\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := func(panel map[string]any) string {
		var out []string
		for _, focus := range panel["focuses"].([]map[string]any) {
			out = append(out, focus["id"].(string))
		}
		return strings.Join(out, ",")
	}

	ger := idx.ContinuousFocusPanel("GER", "1936.1.1", "german_focus")
	if ger == nil {
		t.Fatal("GER has no panel")
	}
	if got := ger["position"].(*focusPosition); *got != (focusPosition{X: 500, Y: 1300}) {
		t.Fatalf("GER panel at %+v, want the tree's continuous_focus_position", *got)
	}
	if got := ids(ger); got != "continuous_naval_production,continuous_unknowable" {
		t.Fatalf("GER focuses = %s", got)
	}
	first := ger["focuses"].([]map[string]any)[0]
	if first["name"] != "Naval Production" || first["daily_cost"] != 1.0 || first["icon"] != "GFX_goal_continuous_naval_production" {
		t.Fatalf("first focus = %+v", first)
	}

	swi := idx.ContinuousFocusPanel("SWI", "1936.1.1", "generic_focus")
	if got := swi["position"].(*focusPosition); *got != (focusPosition{X: 50, Y: 1000}) {
		t.Fatalf("SWI panel at %+v, want the palette's position", *got)
	}
	if got := ids(swi); got != "continuous_naval_production,SWI_militias,continuous_unknowable" {
		t.Fatalf("SWI focuses = %s", got)
	}
}
