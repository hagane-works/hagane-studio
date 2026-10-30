package guirender

import (
	"strings"
	"testing"
)

func TestRealGameBritainsSubjects(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.SubjectsPayload("ENG", "1936.1.1.12")
	rows := out["subjects"].([]map[string]any)
	got := []string{}
	for _, row := range rows {
		step := func(key string) string { return row[key].(map[string]any)["id"].(string) }
		got = append(got, row["tag"].(string)+" "+step("level")+" "+step("down")+" "+step("up"))
	}
	want := []string{
		"AST autonomy_dominion autonomy_colony free",
		"CAN autonomy_dominion autonomy_colony free",
		"NZL autonomy_dominion autonomy_colony free",
		"SAF autonomy_dominion autonomy_colony free",
		"OMA autonomy_colony autonomy_puppet autonomy_dominion",
		"RAJ autonomy_colony autonomy_puppet autonomy_dominion",
		"JOR autonomy_colony autonomy_puppet autonomy_dominion",
		"PAL autonomy_integrated_puppet annex autonomy_puppet",
		"MAL autonomy_integrated_puppet annex autonomy_puppet",
		"BRM autonomy_colony autonomy_puppet autonomy_dominion",
		"KUW autonomy_integrated_puppet annex autonomy_puppet",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("subjects:\n%s", strings.Join(got, "\n"))
	}
	names := []string{}
	for _, row := range rows {
		names = append(names, row["name"].(string))
	}

	wantNames := "Australia|Dominion of Canada|Dominion of New Zealand|South Africa|Sultanate of Muscat and Oman|British Raj|" +
		"Emirate of Transjordan|Mandatory Palestine|British Malaya|British Burma|Sheikhdom of Kuwait"
	if strings.Join(names, "|") != wantNames {
		t.Errorf("names = %s", strings.Join(names, "|"))
	}
	first := rows[0]
	if first["freedom"] != 0.2 || rows[1]["freedom"] != 0.4 || rows[8]["freedom"] != 0.0 {
		t.Errorf("freedom = %v %v %v", first["freedom"], rows[1]["freedom"], rows[8]["freedom"])
	}
	level := first["level"].(map[string]any)
	if level["label"] != "Dominion" || level["icon"] != "GFX_autonomy_dominion_icon" {
		t.Errorf("level = %v", level)
	}
	if icon := first["up"].(map[string]any)["icon"]; icon != "GFX_autonomy_free_icon" {
		t.Errorf("free icon = %v", icon)
	}
	if icon := rows[7]["down"].(map[string]any)["icon"]; icon != "GFX_autonomy_annex_icon" {
		t.Errorf("annex icon = %v", icon)
	}

	colony := out["levels"].(map[string]any)["autonomy_colony"].(map[string]any)
	if colony["min_freedom_level"] != 0.6 || colony["manpower_influence"] != 0.7 || !strings.Contains(colony["rule"].(string), "can_") ||
		colony["file"] != "common/autonomous_states/colony.txt" {
		t.Errorf("colony = %v", colony)
	}

	if none := idx.SubjectsPayload("GER", "1936.1.1.12")["subjects"].([]map[string]any); len(none) != 0 {
		t.Errorf("GER 1936 subjects = %v", none)
	}
	if later := idx.SubjectsPayload("GER", "1939.8.14.12")["subjects"].([]map[string]any); len(later) != 1 || later[0]["tag"] != "SLO" {
		t.Errorf("GER 1939 subjects = %v", later)
	}
}
