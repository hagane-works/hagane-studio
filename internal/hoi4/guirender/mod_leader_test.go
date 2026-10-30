package guirender

import "testing"

func TestModLeaderBeatsBaseGameLeaderOfTheSameParty(t *testing.T) {
	vanilla, err := NewIndex(map[string][]byte{
		"common/ideologies/00_ideologies.txt": []byte(
			"ideologies = { neutrality = { types = { despotism = {} centrism = {} } } " +
				"democratic = { types = { conservatism = {} } } }"),
		"history/countries/KOR - Korea.txt": []byte(
			"capital = 1\nset_politics = { ruling_party = neutrality }\n" +
				"recruit_character = KOR_old_leader\nrecruit_character = KOR_democrat\n"),
		"common/characters/KOR.txt": []byte(`characters = {
	KOR_old_leader = { name = "Old Leader" country_leader = { ideology = centrism } }
	KOR_democrat = { name = "Democrat" country_leader = { ideology = conservatism } }
}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := NewIndexWithFallback(map[string][]byte{
		"history/countries/KOR - zmymod.txt": []byte("recruit_character = KOR_rm\n"),
		"common/characters/KOR_mymod.txt": []byte(`characters = {
	KOR_rm = { name = "RM" country_leader = { ideology = despotism } }
}`),
	}, vanilla)
	if err != nil {
		t.Fatal(err)
	}

	people, ok := mod.CharactersPayloadAt("KOR", "", "1937.1.1")
	if !ok {
		t.Fatal("no characters for KOR")
	}
	if leader, _ := people["leader"].(map[string]any); leader["token"] != "KOR_rm" {
		t.Errorf("leader = %v, want the mod's KOR_rm", leader["token"])
	}

	if name := mod.countryLeaderName("KOR", "1937.1.1"); name != "RM" {
		t.Errorf("diplomacy leader name = %q, want RM", name)
	}

	base, _ := vanilla.CharactersPayloadAt("KOR", "", "1937.1.1")
	if leader, _ := base["leader"].(map[string]any); leader["token"] != "KOR_old_leader" {
		t.Errorf("base-game leader = %v, want KOR_old_leader", leader["token"])
	}

	democratic, _ := mod.CharactersPayloadAt("KOR", "democratic", "1937.1.1")
	if leader, _ := democratic["leader"].(map[string]any); leader["token"] != "KOR_democrat" {
		t.Errorf("democratic leader = %v, want KOR_democrat", leader["token"])
	}
}
