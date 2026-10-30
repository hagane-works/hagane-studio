package guirender

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestOfficerStatsSumTheStartingModifiers(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nadd_ideas = tag_spirit\n" +
			"set_variable = { TAG_planning = 0.25 }\nadd_dynamic_modifier = { modifier = TAG_army }\n" +
			"set_technology = { transport = 1 }\nset_grand_doctrine = mobile\n"),
		"common/ideas/tag.txt":                      []byte("ideas = { country = { tag_spirit = { modifier = { supply_consumption_factor = 0.15 } } } }"),
		"common/dynamic_modifiers/tag.txt":          []byte("TAG_army = { icon = GFX_idea_x planning_speed = TAG_planning }"),
		"common/technologies/tech.txt":              []byte("technologies = { transport = { naval_invasion_division_cap = 2 research_cost = 1 } }"),
		"common/doctrines/grand_doctrines/land.txt": []byte("mobile = { folder = land planning_speed = 0.20 army_speed_factor = 0.10 xp_cost = 100 }"),
		"common/defines/00_defines.lua": []byte("NDefines.NNavy.BASE_NAVAL_INVASION_DIVISION_CAP = 4\n" +
			"NDefines.NMilitary.BASE_NIGHT_ATTACK_PENALTY = -0.5\nNDefines.NMilitary.SUPPLY_GRACE = 72\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	stats := idx.OfficerStatsPayload("TAG", "1936.1.1")["stats"].(map[string]any)
	got := map[string]string{}
	for _, branch := range []string{"army", "navy"} {
		for _, item := range stats[branch].([]map[string]any) {
			got[item["modifier"].(string)] = fmt.Sprint(item["text"], " ", item["colour"])
		}
	}
	for modifier, want := range map[string]string{
		"planning_speed":              "145% G",
		"army_speed_factor":           "10% G",
		"supply_consumption_factor":   "115% R",
		"naval_invasion_division_cap": "6 G",
		"land_night_attack":           "-50% Y",
		"no_supply_grace":             "72 H Y",
	} {
		if got[modifier] != want {
			t.Errorf("%s = %q, want %q", modifier, got[modifier], want)
		}
	}
}

func TestRealGameOfficerStatsMatchTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	stats := idx.OfficerStatsPayload("GER", "1936.1.1.12")["stats"].(map[string]any)
	want := map[string]string{
		"army":           "-50% Y|10% G|145% G|30% Y|100% Y|5 Y|100% Y|100% Y|2% Y|115% R|100% Y|100% Y|108 H G|-20% Y",
		"navy":           "0% Y|50% Y|0% Y|100% Y|100% Y|115% G|6 G|2 G|50 G|100% Y",
		"air":            "100% Y|100% Y|100% Y|120% G|100% Y|100% Y|100% Y|100% Y|100% Y|100% Y|115% G|100% Y|125% G|100% Y",
		"special_forces": strings.Repeat("100% Y|", 8) + "100% Y",
	}
	failed := false
	for branch, row := range want {
		got := []string{}
		for _, item := range stats[branch].([]map[string]any) {
			got = append(got, fmt.Sprint(item["text"], " ", item["colour"]))
		}
		if strings.Join(got, "|") != row {
			failed = true
			t.Errorf("%s\n got %s\nwant %s", branch, strings.Join(got, "|"), row)
		}
	}
	if failed {

		for _, source := range idx.startingModifierSources("GER", "1936.1.1.12") {
			keys := []string{}
			for key := range source.Values {
				for _, stats := range officerStats {
					for _, s := range stats {
						if s.Modifier == key {
							keys = append(keys, fmt.Sprintf("%s=%v", key, source.Values[key]))
						}
					}
				}
			}
			sort.Strings(keys)
			if len(keys) > 0 {
				t.Logf("%s: %s", source.Name, strings.Join(keys, " "))
			}
		}
	}
}
