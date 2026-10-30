package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func rosterIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nrecruit_character = TAG_late\nrecruit_character = TAG_first\nrecruit_character = TAG_hidden\nrecruit_character = TAG_marshal\nrecruit_character = TAG_weak\n"),
		"common/characters/TAG.txt": []byte(`characters = {
			TAG_first = { name = TAG_first corps_commander = { traits = { brilliant_strategist } skill = 4 attack_skill = 4 defense_skill = 3 planning_skill = 4 logistics_skill = 2 } }
			TAG_hidden = { name = TAG_hidden corps_commander = { visible = { has_completed_focus = TAG_later } skill = 4 } }
			TAG_weak = { name = TAG_weak corps_commander = { skill = 2 } }
			TAG_marshal = { name = TAG_marshal field_marshal = { skill = 3 } }
			TAG_late = { name = TAG_late corps_commander = { skill = 4 } }
		}`),
		"common/unit_leader/00_traits.txt": []byte(`leader_traits = {
			brilliant_strategist = { type = land trait_type = personality_trait attack_skill = 1 planning_skill = 1 }
			panzer_expert = { type = corps_commander trait_type = assignable_trait any_parent = { panzer_leader } mutually_exclusive = cavalry_expert gui_row = 1 gui_column = 2 }
		}`),
		"localisation/english/l.yml": []byte("l_english:\n cc_leader:0 \"General\"\n fm_leader:0 \"Field Marshal\"\n TAG_first:0 \"First\"\n"),
		"common/defines/00_defines.lua": []byte("NDefines.NPolitics.ARMY_LEADER_COST = 5\nNDefines.NPolitics.ARMY_LEADER_MAX_COST = 80\n" +
			"NDefines.NMilitary.CORPS_COMMANDER_DIVISIONS_CAP = 24\nNDefines.NMilitary.FIELD_MARSHAL_DIVISIONS_CAP = 24\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestCommanderRosterOrderAndVisibility(t *testing.T) {
	payload := rosterIndex(t).CommanderRosterPayload("TAG", "1936.1.1.12", "army")
	tokens := []string{}
	for _, item := range payload["items"].([]map[string]any) {
		tokens = append(tokens, item["token"].(string))
	}
	if got := strings.Join(tokens, ","); got != "TAG_first,TAG_late,TAG_marshal,TAG_weak" {
		t.Errorf("roster = %s", got)
	}

	if payload["new_leader_cost"] != 25 {
		t.Errorf("new leader cost = %v", payload["new_leader_cost"])
	}
}

func TestCommanderRosterRow(t *testing.T) {
	items := rosterIndex(t).CommanderRosterPayload("TAG", "1936.1.1.12", "army")["items"].([]map[string]any)
	first := items[0]
	skills := first["skills"].(map[string]int)
	if fmt.Sprint(skills["attack_skill"], skills["defense_skill"], skills["planning_skill"], skills["logistics_skill"]) != "5 3 5 2" {
		t.Errorf("skills = %v", skills)
	}
	if first["name"] != "First" || first["rank_label"] != "General" || first["capacity"] != 24 {
		t.Errorf("row = %v", first)
	}
	traits := first["traits"].([]map[string]any)
	if len(traits) != 1 || traits[0]["icon"] != "GFX_trait_brilliant_strategist" {
		t.Errorf("traits = %v", traits)
	}
	if items[2]["rank_label"] != "Field Marshal" {
		t.Errorf("marshal = %v", items[2])
	}
}

func TestLeaderTraitIndex(t *testing.T) {
	trait := rosterIndex(t).leaderTraitFor("panzer_expert")
	if trait == nil || !trait.HasGui || trait.GuiRow != 1 || trait.GuiColumn != 2 ||
		fmt.Sprint(trait.AnyParent, trait.Exclusive, trait.Types) != "[panzer_leader] [cavalry_expert] [corps_commander]" {
		t.Errorf("panzer_expert = %+v", trait)
	}
}

func TestRealGameCommanderRosterMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.CommanderRosterPayload("GER", "1936.1.1.12", "army")
	items := payload["items"].([]map[string]any)
	names := []string{}
	for _, item := range items[:6] {
		names = append(names, item["name"].(string))
	}
	if got := strings.Join(names, ", "); got != "Gerd von Rundstedt, Günther von Kluge, Erich von Manstein, Fedor von Bock, Maximilian von Weichs, Erwin von Witzleben" {
		t.Errorf("roster opens %s", got)
	}
	skills := items[0]["skills"].(map[string]int)
	if fmt.Sprint(skills["attack_skill"], skills["defense_skill"], skills["planning_skill"], skills["logistics_skill"]) != "5 3 5 2" {
		t.Errorf("Rundstedt's skills = %v", skills)
	}
	if items[0]["rank_label"] != "General" || items[0]["capacity"] != 24 || payload["new_leader_cost"] != 80 {
		t.Errorf("Rundstedt %v %v, new leader cost %v", items[0]["rank_label"], items[0]["capacity"], payload["new_leader_cost"])
	}
	for _, trait := range items[0]["traits"].([]map[string]any) {
		if _, declared := idx.spriteRecordFor(trait["icon"].(string)); !declared {
			t.Errorf("trait icon %v not declared", trait["icon"])
		}
	}
}

func TestRealGameNavyRoster(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.CommanderRosterPayload("GER", "1936.1.1.12", "navy")
	items := payload["items"].([]map[string]any)
	if len(items) == 0 || items[0]["name"] != "Karl Dönitz" || items[0]["rank_label"] != "Admiral" || items[0]["capacity"] != 10 {
		t.Fatalf("navy roster opens %v", items[:min(1, len(items))])
	}
	skills := items[0]["skills"].(map[string]int)
	if skills["skill"] != 5 || skills["maneuvering_skill"] == 0 || skills["coordination_skill"] == 0 {
		t.Errorf("Dönitz's skills = %v", skills)
	}
	if payload["new_leader_cost"] != 5*len(items) {
		t.Errorf("new admiral costs %v for %d admirals", payload["new_leader_cost"], len(items))
	}
}
