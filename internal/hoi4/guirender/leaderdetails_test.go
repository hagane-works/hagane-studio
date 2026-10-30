package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func detailsIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nrecruit_character = TAG_general\n"),
		"common/characters/TAG.txt": []byte(`characters = {
			TAG_general = {
				name = TAG_general
				advisor = { slot = high_command idea_token = TAG_general_adv traits = { army_regrouping_2 } }
				corps_commander = { traits = { old_guard trait_cautious panzer_leader } skill = 4 attack_skill = 1 }
			}
		}`),
		"common/unit_leader/00_traits.txt": []byte(`leader_traits = {
			old_guard = { type = land trait_type = personality_trait non_shared_modifier = { experience_gain_factor = -0.25 } modifier = { max_dig_in = 1 } }
			trait_cautious = { type = land trait_type = personality_trait leader_default_proximity_offset = 1 modifier = { planning_speed = -0.20 } attack_skill = 1 }
			panzer_leader = { type = corps_commander gui_row = 1 }
			cavalry_leader = { type = corps_commander gui_row = 3 }
			combined_arms_expert = { type = corps_commander trait_type = assignable_trait gui_row = 2 any_parent = { cavalry_leader panzer_leader } mutually_exclusive = { panzer_expert } }
			fast_planner = { type = { field_marshal } trait_type = assignable_trait gui_row = 1 }
			adaptable = { type = land trait_type = assignable_terrain_trait gui_row = 3 parent = { traits = { desert_fox } num_parents_needed = 1 } }
			sea_wolf = { type = navy trait_type = assignable_trait gui_row = 0 gui_column = 0 }
		}`),
		"common/country_leader/00_traits.txt": []byte(`leader_traits = {
			army_regrouping_2 = { sprite = 5 army_morale_factor = 0.08 command_cap_increase = 10 command_power = 1 }
		}`),
		"localisation/english/l.yml": []byte("l_english:\n" +
			" cc_leader:0 \"General\"\n TAG_general:0 \"The General\"\n army_regrouping_2:0 \"Army Regrouping\\n(Expert)\"\n" +
			" MODIFIER_MAX_DIG_IN:0 \"Max Entrenchment\"\n MODIFIER_XP_GAIN_FACTOR:0 \"Leader Experience Gain\"\n" +
			" MODIFIER_PLANNING_SPEED:0 \"Planning Speed\"\n MODIFIER_ARMY_MORALE_FACTOR:0 \"Division Recovery Rate\"\n" +
			" trait_bonus_attack:0 \"Attack: $VAL|+=0$\"\n" +
			" trait_bonus_leader_default_proximity_offset:0 \"Starting Proximity to frontline: $NAME$\"\n" +
			" LEADER_PROXIMITY_MEDIUM_TITLE:0 \"§HMedium§!\"\n" +
			" combined_arms_expert:0 \"Combined Arms Expert\"\n combined_arms_expert_short:0 \"Comb. Arms Expert\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestLeaderDetailsLeftColumn(t *testing.T) {
	payload := detailsIndex(t).LeaderDetailsPayload("TAG", "1936.1.1.12", "TAG_general")
	advisor := payload["advisor"].(map[string]any)
	if advisor["label"] != "Army Regrouping(Expert)" || fmt.Sprint(advisor["effects"]) != "[Division Recovery Rate: +8.0%]" || advisor["sprite"] != 5 {
		t.Errorf("advisor = %v", advisor)
	}
	lines := []string{}
	for _, item := range payload["background"].([]map[string]any) {
		lines = append(lines, item["label"].(string)+" "+fmt.Sprint(item["effects"]))
	}
	want := "trait_cautious [Attack: +1 Planning Speed: -20.0% Starting Proximity to frontline: Medium]|" +
		"old_guard [Max Entrenchment: +1.0 Leader Experience Gain: -25.0%]"
	if got := strings.Join(lines, "|"); got != want {
		t.Errorf("background = %s", got)
	}
	traits := payload["traits"].([]map[string]any)
	if len(traits) != 1 || traits[0]["token"] != "panzer_leader" {
		t.Errorf("traits = %v", traits)
	}

	if payload["pick_slots"] != 2 {
		t.Errorf("pick slots = %v", payload["pick_slots"])
	}
}

func TestLeaderDetailsTree(t *testing.T) {
	payload := detailsIndex(t).LeaderDetailsPayload("TAG", "1936.1.1.12", "TAG_general")
	cells := []string{}
	for _, tree := range []string{"regular", "terrain"} {
		for _, item := range payload[tree].([]map[string]any) {
			cells = append(cells, fmt.Sprintf("%s %v,%v %s %v %v %v", tree, item["column"], item["row"], item["label"], item["parents"], item["exclusive"], item["held"]))
		}
	}
	want := strings.Join([]string{
		"regular 0,1 panzer_leader [] [] true",
		"regular 0,3 cavalry_leader [] [] false",
		"regular 1,2 Comb. Arms Expert [cavalry_leader panzer_leader] [panzer_expert] false",
		"regular 2,1 fast_planner [] [] false",
		"terrain 1,3 adaptable [desert_fox] [] false",
	}, "\n")
	if got := strings.Join(cells, "\n"); got != want {
		t.Errorf("tree =\n%s\nwant\n%s", got, want)
	}
}

func TestRealGameLeaderDetailsMatchTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.LeaderDetailsPayload("GER", "1936.1.1.12", "GER_gerd_von_rundstedt")
	advisor, _ := payload["advisor"].(map[string]any)
	if advisor["label"] != "Army Regrouping(Expert)" || fmt.Sprint(advisor["effects"]) != "[Division Recovery Rate: +8.0%]" {
		t.Errorf("advisor = %v", advisor)
	}
	lines := []string{}
	for _, item := range payload["background"].([]map[string]any) {
		lines = append(lines, item["label"].(string)+": "+strings.Join(item["effects"].([]string), " / "))
	}
	want := strings.Join([]string{
		"Old Guard: Max Entrenchment: +1.0 / Leader Experience Gain: -25.0%",
		"Brilliant Strategist: Attack: +1 / Planning: +1",
		"Cautious: Planning Speed: -20.0% / Chance to get wounded in combat: -50.0% / Starting Proximity to frontline: Medium",
	}, "\n")
	if got := strings.Join(lines, "\n"); got != want {
		t.Errorf("background =\n%s", got)
	}
	if payload["pick_slots"] != 2 || len(payload["traits"].([]map[string]any)) != 0 {
		t.Errorf("pick slots %v, traits %v", payload["pick_slots"], payload["traits"])
	}
	columns := map[string][]string{}
	for _, tree := range []string{"regular", "terrain"} {
		for _, item := range payload[tree].([]map[string]any) {
			key := fmt.Sprintf("%s %v", tree, item["column"])
			row := item["row"].(int)
			for len(columns[key]) <= row {
				columns[key] = append(columns[key], "-")
			}
			columns[key][row] = item["label"].(string)
		}
	}
	for key, want := range map[string]string{
		"regular 0": "Organizer, Panzer Leader, -, Cavalry Leader, Engineer, -, Infantry Leader, -, Invader, -, Commando",
		"regular 1": "-, Panzer Expert, Comb. Arms Expert, Cavalry Expert, Fortress Buster, Scavenger, Infantry Expert, Ambusher, Amphibious, Naval Liaison, Skirmisher, Paratroopers",
		"regular 2": "Logistics Wizard, Fast Planner, Thorough Planner, Unyielding Defender, Aggr. Assaulter, Offensive Doctrine, Defensive Doctrine, Organization First, Charismatic",
		"terrain 0": "Desert Fox, Swamp Fox, Hill Fighter, Mountaineer, Urban Assault Specialist, Ranger, Jungle Rat, Winter Specialist",
		"terrain 1": "-, -, -, Adaptable, -, -, -, Winter Expert",
	} {
		column := columns[key]
		if len(column) > len(strings.Split(want, ", ")) {
			column = column[:len(strings.Split(want, ", "))]
		}
		if got := strings.Join(column, ", "); got != want {
			t.Errorf("%s = %s", key, got)
		}
	}
	for _, name := range []string{"GFX_trait_organizer", "GFX_trait_adaptable"} {
		if _, declared := idx.spriteRecordFor(name); !declared {
			t.Errorf("%s not declared", name)
		}
	}
}
