package guirender

import "testing"

func TestDiplomacyHeaderCarriesTheKeysItPrintsFrom(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/ideologies/00_ideologies.txt":  []byte(`ideologies = { neutrality = { types = { despotism = {} } } }`),
		"common/country_tags/00_countries.txt": []byte(`LED = "countries/Leaderland.txt"`),
		"history/countries/LED - Leaderland.txt": []byte(`capital = 1
set_politics = { ruling_party = neutrality }
recruit_character = LED_boss`),
		"common/characters/LED.txt": []byte(`characters = {
	LED_boss = { name = LED_boss_name country_leader = { ideology = despotism } }
}`),
		"localisation/english/l_l_english.yml": []byte("l_english:\n LED_neutrality:0 \"Leaderland\"\n LED_boss_name:0 \"The Boss\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	out := idx.DiplomacyPayload("LED", "1936.1.1.12")
	self := out["self"].(map[string]any)
	if self["name"] != "Leaderland" || self["name_key"] != "LED_neutrality" {
		t.Errorf("self = %v", self)
	}
	if out["leader"] != "The Boss" || out["leader_token"] != "LED_boss" {
		t.Errorf("leader = %v / %v", out["leader"], out["leader_token"])
	}
}

func TestRealGameDiplomacySeenFromAnotherCountry(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.DiplomacyPayload("POL", "1936.1.1.12")
	self := out["self"].(map[string]any)
	if self["party_name"] != "Sanacja" {
		t.Errorf("party name = %v", self["party_name"])
	}
	texts := out["texts"].(map[string]any)
	if texts["no_faction"] != "No faction" || texts["intel_ledger"] != "Intel Ledger" || texts["unknown_focus"] != "Unknown Focus" {
		t.Errorf("texts = %v", texts)
	}
	if texts["select_focus"] != "Select a National Focus" {
		t.Errorf("select_focus = %v, want the politics screen's own label", texts["select_focus"])
	}
	actions := out["actions"].([]map[string]any)
	if len(actions) != 18 || actions[0]["label"] != "Declare war" || actions[1]["id"] != "generate_wargoal" || actions[1]["label"] != "Justify War goal" {
		t.Fatalf("actions = %v", actions)
	}
	byID := map[string]map[string]any{}
	for _, action := range actions {
		byID[action["id"].(string)] = action
	}
	for id, want := range map[string][2]float64{
		"guarantee": {15, 0}, "improve_relations": {10, 0}, "send_attache": {100, 50}, "embargo": {100, 0},
	} {
		pp, _ := byID[id]["pp"].(float64)
		cp, _ := byID[id]["cp"].(float64)
		if pp != want[0] || cp != want[1] {
			t.Errorf("%s costs %v PP %v CP, want %v", id, pp, cp, want)
		}
	}
}
