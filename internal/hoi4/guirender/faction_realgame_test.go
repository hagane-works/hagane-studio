package guirender

import (
	"strings"
	"testing"
)

func TestFactionOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	got := idx.FactionPayload("GER", "1936.1.1.12")
	if got["in_faction"] != true || got["name"] != "Axis" || got["manifest"] != "Conquest and Domination" {
		t.Fatalf("header: %v / %v / %v", got["in_faction"], got["name"], got["manifest"])
	}
	if got["icon"] != "GFX_faction_logo_axis" {
		t.Errorf("icon %v", got["icon"])
	}
	goals := got["goals"].(map[string]any)
	want := map[string]string{"short_term": "Unite the German States", "medium_term": "Establish Fortress Europa", "long_term": "Secure Strategic Resources"}
	for term, name := range want {
		goal, _ := goals[term].(map[string]any)
		if goal == nil || goal["name"] != name {
			t.Errorf("%s goal %v, want %q", term, goals[term], name)
		}
	}
	members := got["members"].([]map[string]any)
	if len(members) != 1 || members[0]["tag"] != "GER" || members[0]["share"] != 100 {
		t.Errorf("members %v", members)
	}
	if got["rank"] != 4 || !strings.Contains(got["rank_text"].(string), "Global Rank:") {
		t.Errorf("rank %v %q, want 4 (the recording)", got["rank"], got["rank_text"])
	}

	intel := got["intel"].(map[string]any)
	if intel["spymaster_button"] != "Become Spymaster £faction_initiative_texticon£ 1" || intel["vacant"] != "No intel advisor assigned" {
		t.Errorf("intel %q / %q", intel["spymaster_button"], intel["vacant"])
	}
	branches := intel["branches"].([]map[string]any)
	if len(branches) != 3 || branches[0]["header"] != "Cryptology" || branches[2]["header"] != "Operations" ||
		branches[1]["button"] != "Unlock £faction_initiative_texticon£ 1" || branches[0]["can_take"] != false || intel["spymaster_can_take"] != false {
		t.Errorf("branches %v", branches)
	}
	if intel["portrait"] != vacantPortrait {
		t.Errorf("portrait %v", intel["portrait"])
	}
	t.Logf("factions on the first day: %v", idx.factionLeaders("1936.1.1.12"))
	if none := idx.FactionPayload("POL", "1936.1.1.12"); none["in_faction"] != false {
		t.Errorf("Poland in a faction in 1936: %v", none)
	}
}

func TestConstructionOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	got := idx.ConstructionPayload("GER", "1936.1.1.12")
	if got["civilian"] != 35 || got["military"] != 28 {
		t.Errorf("factories %v civilian / %v military, want 35 / 28", got["civilian"], got["military"])
	}
	if speed := got["speed"].(float64); speed < 0.0499 || speed > 0.0501 {
		t.Errorf("speed %v, want 0.05", speed)
	}
	if got["consumer"] != 15 || got["projects"] != 0 || got["used"] != 15 {
		t.Errorf("consumer %v projects %v used %v", got["consumer"], got["projects"], got["used"])
	}
	rows := got["rows"].([]map[string]any)
	if len(rows) != 1 || rows[0]["kind"] != "consumer_goods" {
		t.Errorf("rows %v", rows)
	}
}
