package guirender

import "testing"

const testOfficers = `
characters = {
GER_ludwig_beck = {
	name = GER_ludwig_beck
	portraits = { army = { small = GFX_idea_ludwig_beck } }
	advisor = {
		slot = army_chief
		idea_token = ludwig_beck
		traits = { army_chief_organizational_2 }
		cost = 100
	}
}
GER_erich_raeder = {
	name = GER_erich_raeder
	portraits = { navy = { small = GFX_portrait_raeder } }
	advisor = {
		slot = navy_chief
		idea_token = raeder
		traits = { navy_chief_decisive_battle_2 }
		cost = 150
	}
}
GER_someone_else = {
	name = GER_someone_else
	advisor = { slot = political_advisor  idea_token = someone }
}
}
`

func officerIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/characters/GER.txt":           []byte(testOfficers),
		"history/countries/GER - Germany.txt": []byte("capital = 1\nset_politics = { ruling_party = fascism }\n"),
		"localisation/english/advisors.yml": []byte(
			"l_english:\n GER_ludwig_beck: \"$ludwig_beck$\"\n ludwig_beck: \"Ludwig Beck\"\n" +
				" raeder: \"Erich Raeder\"\n" +
				" army_chief_organizational_2: \"Army Organization (Expert)\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestOfficerCorpsGroupsByTheSlotTheAdvisorNames(t *testing.T) {
	payload, ok := officerIndex(t).OfficerCorpsPayload("GER", "1936.1.1")
	if !ok {
		t.Fatal("no payload")
	}
	slots, _ := payload["slots"].(map[string][]map[string]any)
	if len(slots["army_chief"]) != 1 || len(slots["navy_chief"]) != 1 {
		t.Fatalf("slots = %v, want one chief each", slots)
	}

	if _, listed := slots["political_advisor"]; listed {
		t.Errorf("a political advisor is in the officer corps: %v", slots["political_advisor"])
	}
	chief := slots["army_chief"][0]
	if chief["cost"] != 100 {
		t.Errorf("cost = %v, want what the advisor block says", chief["cost"])
	}
	traits, _ := chief["traits"].([]map[string]any)
	if len(traits) != 1 || traits[0]["label"] != "Army Organization (Expert)" {
		t.Errorf("traits = %v", traits)
	}
}

func TestAdvisorNameFollowsTheLocalisationHop(t *testing.T) {
	payload, _ := officerIndex(t).OfficerCorpsPayload("GER", "1936.1.1")
	slots, _ := payload["slots"].(map[string][]map[string]any)

	if slots["army_chief"][0]["name"] != "Ludwig Beck" {
		t.Errorf("name = %v, want the key the reference points at", slots["army_chief"][0]["name"])
	}

	if slots["navy_chief"][0]["name"] != "Erich Raeder" {
		t.Errorf("name = %v, want the idea token's own name", slots["navy_chief"][0]["name"])
	}
}

func TestOfficerCorpsSeatsAndHolders(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/characters/GER.txt":           []byte(testOfficers),
		"history/countries/GER - Germany.txt": []byte("capital = 1\nset_politics = { ruling_party = fascism }\nadd_ideas = ludwig_beck\n"),
		"common/idea_tags/00_idea.txt": []byte(`idea_categories = {
			military_staff = { character_slot = army_chief character_slot = navy_chief character_slot = high_command character_slot = high_command }
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := idx.OfficerCorpsPayload("GER", "1936.1.1")
	seats := payload["seats"].(map[string]int)
	if seats["army_chief"] != 1 || seats["high_command"] != 2 || seats["theorist"] != 0 {
		t.Errorf("seats = %v", seats)
	}
	slots := payload["slots"].(map[string][]map[string]any)
	if slots["army_chief"][0]["hired"] != true || slots["navy_chief"][0]["hired"] != nil {
		t.Errorf("holders: beck %v, raeder %v", slots["army_chief"][0]["hired"], slots["navy_chief"][0]["hired"])
	}
}

func TestRealGameOfficerCorpsSeatsAreEmpty(t *testing.T) {
	payload, ok := realTopbarIndex(t, false).OfficerCorpsPayload("GER", "1936.1.1.12")
	if !ok {
		t.Fatal("no payload")
	}
	seats := payload["seats"].(map[string]int)
	if seats["high_command"] != 3 || seats["theorist"] != 1 || seats["army_chief"] != 1 || seats["navy_chief"] != 1 || seats["air_chief"] != 1 {
		t.Errorf("seats = %v", seats)
	}
	for slot, people := range payload["slots"].(map[string][]map[string]any) {
		for _, person := range people {
			if person["hired"] == true {
				t.Errorf("%s holds %s on the first day", person["name"], slot)
			}
		}
	}
}

func TestStartingGrandDoctrines(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/characters/GER.txt": []byte(testOfficers),
		"history/countries/GER - Germany.txt": []byte("capital = 1\nset_politics = { ruling_party = fascism }\n" +
			"set_grand_doctrine = mass_assault\nset_grand_doctrine = convoy_raiding\n1939.1.1 = { set_grand_doctrine = mobile_warfare }\n"),
		"common/doctrines/grand_doctrines/land.txt": []byte("mass_assault = { folder = land icon = GFX_doctrine_mass_assault_medium }\n" +
			"mobile_warfare = { folder = land icon = GFX_doctrine_mobile_warfare_medium }\n"),
		"common/doctrines/grand_doctrines/sea.txt": []byte("convoy_raiding = { folder = naval icon = GFX_doctrine_trade_interdiction_medium }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.StartingGrandDoctrines("GER", "1936.1.1"); got["land"] != "mass_assault" || got["naval"] != "convoy_raiding" || len(got) != 2 {
		t.Errorf("1936 = %v", got)
	}
	if got := idx.StartingGrandDoctrines("GER", "1939.1.1"); got["land"] != "mobile_warfare" {
		t.Errorf("1939 = %v", got)
	}
}

func TestRealGameOfficerCorpsDoctrines(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload, _ := idx.OfficerCorpsPayload("GER", "1936.1.1.12")
	doctrines := payload["doctrines"].(map[string]any)
	for folder, want := range map[string]string{"land": "new_mobile_warfare", "naval": "new_convoy_raiding", "air": "new_battlefield_support"} {
		item, _ := doctrines[folder].(map[string]any)
		if item["token"] != want {
			t.Errorf("%s ring = %v, want %s", folder, item, want)
			continue
		}
		if _, declared := idx.spriteRecordFor(item["icon"].(string)); !declared {
			t.Errorf("%s icon %v not declared", folder, item["icon"])
		}
	}
	if _, ok := doctrines["special_forces"]; ok {
		t.Errorf("special forces ring = %v", doctrines["special_forces"])
	}
}

func TestAdvisorNameInAnotherLanguage(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/characters/GER.txt":           []byte(testOfficers),
		"history/countries/GER - Germany.txt": []byte("capital = 1\nset_politics = { ruling_party = fascism }\n"),
		"localisation/english/advisors.yml": []byte(
			"l_english:\n GER_ludwig_beck: \"$ludwig_beck$\"\n ludwig_beck: \"路德维希·贝克\"\n raeder: \"雷德尔\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := idx.OfficerCorpsPayload("GER", "1936.1.1")
	slots, _ := payload["slots"].(map[string][]map[string]any)
	if slots["army_chief"][0]["name"] != "路德维希·贝克" {
		t.Errorf("name = %v, want the name the reference points at", slots["army_chief"][0]["name"])
	}
	if slots["navy_chief"][0]["name"] != "雷德尔" {
		t.Errorf("name = %v, want a name without any separator", slots["navy_chief"][0]["name"])
	}
}
