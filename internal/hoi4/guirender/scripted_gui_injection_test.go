package guirender

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

type injectionParityCase struct {
	Name       string `json:"name"`
	GUIText    string `json:"gui_text"`
	EffectText string `json:"effect_text"`
}

func injectionParityCases() []injectionParityCase {
	return []injectionParityCase{
		{
			Name: "documented_dynamic_list_and_explicit_array",
			GUIText: `scripted_gui = { brics = {
				window_name = "brics_list_window"
				visible = { has_country_flag = open }
				dynamic_list = { name = "members" entry_container = "member_entry" limit = { has_idea = membership } }
			} }`,
			EffectText: `scripted_effects = {
				SOV_join_brics = { add_to_array = { array = members value = SOV } }
				unrelated = { set_country_flag = no }
			}`,
		},
		{
			Name: "hardcoded_panel_and_gate_order",
			GUIText: `scripted_gui = { parliament = {
				window_name = road56_parliament
				visible = { NOT = { has_country_flag = shut } tag = LUX has_country_flag = open }
				effects = { one = { set_country_flag = one } }
			} }`,
		},
		{
			Name: "plural_lists_scope_shorthand_and_idea_recipe",
			GUIText: `scripted_gui = { COMECON = {
				window_name = COMECON_GUI
				dynamic_lists = {
					members = { array = global.COMECON_MEMBERS entry_container = Member_Entry }
					eligible = { array = ROOT.COMECON_ELIGIBLE entry_container = Eligible_Entry }
				}
			} }`,
			EffectText: `scripted_effects = {
				refresh = { every_country = {
					limit = { has_idea = KOM_socintern_member NOT = { tag = ZZZ } }
					add_to_array = { global.COMECON_MEMBERS = THIS }
				} }
				KOM_join_bloc = { add_to_array = { PREV.unrelated = THIS } }
			}`,
		},
		{
			Name: "bare_blocks_name_heuristic_and_case_insensitive_fields",
			GUIText: `panel = {
				WINDOW_NAME = "bare_window"
				DYNAMIC_LIST = { ARRAY = ROOT.members ENTRY_CONTAINER = "row" }
				TRIGGER = { HAS_DLC = yes original_tag = GER }
			}`,
			EffectText: `GER_add_member = { hidden_effect = { ADD_TO_ARRAY = { ROOT.other = GER } } }`,
		},
		{
			Name: "non_window_wrapper_noise_is_ignored",
			GUIText: `scripted_gui = {
				noise = { dynamic_list = { name = bad entry_container = bad } }
				good = { window_name = "good" dynamic_lists = { grid = { array = ROOT.ok entry_container = row } } }
			}`,
			EffectText: `scripted_effect = { exact = { add_to_array = { array = ok value = X } } }`,
		},
	}
}

func canonicalInjectionJSON(value any) []byte {
	raw, _ := json.Marshal(value)
	var normalized any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(&normalized)
	out, _ := json.Marshal(normalized)
	return out
}

func TestAnalyzeInjectionPointsPythonDeepParity(t *testing.T) {
	cases := injectionParityCases()

	raw, err := os.ReadFile("testdata/scripted_gui_injection_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected []struct {
		Name   string           `json:"name"`
		Result []InjectionPoint `json:"result"`
	}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("decode Python result: %v\n%s", err, raw)
	}
	if len(expected) != len(cases) {
		t.Fatalf("Python returned %d fixtures, want %d", len(expected), len(cases))
	}
	for i, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if expected[i].Name != tc.Name {
				t.Fatalf("fixture name=%q want=%q", expected[i].Name, tc.Name)
			}
			got := AnalyzeInjectionPoints(tc.GUIText, tc.EffectText)
			if !bytes.Equal(canonicalInjectionJSON(got), canonicalInjectionJSON(expected[i].Result)) {
				t.Fatalf("Python contract drift\n got=%s\nwant=%s", canonicalInjectionJSON(got), canonicalInjectionJSON(expected[i].Result))
			}
		})
	}
}

func TestProjectInjectionPointsOrdersAndDeduplicatesWindows(t *testing.T) {
	guiFiles := []NamedInjectionSource{
		{Name: "z.txt", Text: `scripted_gui={z={window_name=z}}`},
		{Name: "a.txt", Text: `scripted_gui={first={window_name=a} duplicate={window_name=z}}`},
	}
	got := ProjectInjectionPoints(guiFiles, nil)
	if len(got) != 2 || got[0].WindowName != "a" || got[1].WindowName != "z" {
		t.Fatalf("ordering/dedupe=%#v", got)
	}
}

func TestAnalyzeInjectionPointsAlwaysReturnsJSONArrays(t *testing.T) {
	got := AnalyzeInjectionPoints(`scripted_gui={x={window_name=x}}`, "")
	if len(got) != 1 {
		t.Fatalf("points=%#v", got)
	}
	raw, _ := json.Marshal(got[0])
	for _, forbidden := range [][]byte{[]byte(`"gate":null`), []byte(`"member_add_effects":null`), []byte(`"dynamic_lists":null`), []byte(`"member_add_ideas":null`)} {
		if bytes.Contains(raw, forbidden) {
			t.Fatalf("nil collection leaked: %s", raw)
		}
	}
}
