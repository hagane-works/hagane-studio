package guirender

import "testing"

func TestCountryHistoryFilesOfOneLayerAddUp(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/GER - New World.txt": []byte(`capital = 1
set_politics = { ruling_party = neutrality last_election = "1932.1.1" election_frequency = 48 elections_allowed = no }
set_popularities = { democratic = 20 fascism = 20 communism = 20 neutrality = 40 }
recruit_character = GER_first
`),
		"history/countries/GER - zMod.txt": []byte("recruit_character = GER_otto_weber\n"),
		"history/countries/ENG - New World.txt": []byte(`capital = 5
set_popularities = { democratic = 60 neutrality = 40 }
set_politics = { ruling_party = democratic }
`),
		"history/countries/ENG - zMod.txt": []byte("set_politics = { ruling_party = fascism }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ger := idx.countryPoliticsFor("GER")
	if ger == nil || ger.Capital != 1 || ger.RulingParty != "neutrality" || ger.Popularities["neutrality"] != 40 || len(ger.Recruits) != 2 {
		t.Fatalf("GER: %+v", ger)
	}
	eng := idx.countryPoliticsFor("ENG")
	if eng == nil || eng.Capital != 5 || eng.RulingParty != "fascism" || eng.Popularities["democratic"] != 60 {
		t.Fatalf("ENG: %+v", eng)
	}
}

func TestCountryHistoryFilesAcrossLayers(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"history/countries/GER - Germany.txt": []byte(`capital = 64
set_politics = { ruling_party = fascism }
set_popularities = { fascism = 60 neutrality = 40 }
set_stability = 0.8
recruit_character = GER_adolf_hitler
add_named_threat = { threat = 5 }
`),
		"history/countries/ITA - Italy.txt": []byte("capital = 2\nset_politics = { ruling_party = fascism }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := NewIndexWithFallback(map[string][]byte{
		"history/countries/GER - zMod.txt":  []byte("recruit_character = GER_otto_weber\n"),
		"history/countries/ITA - Italy.txt": []byte("capital = 9\n"),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	ger := mod.countryPoliticsFor("ger")
	if ger == nil || ger.Capital != 64 || ger.RulingParty != "fascism" || ger.Popularities["fascism"] != 60 || len(ger.Recruits) != 2 || len(ger.Effects) == 0 {
		t.Fatalf("GER: %+v", ger)
	}
	if ita := mod.countryPoliticsFor("ITA"); ita == nil || ita.Capital != 9 || ita.RulingParty != "" {
		t.Fatalf("ITA replaced by the project's own file: %+v", ita)
	}
	if got := mod.WorldTension("1936.1.1"); got != 5 {
		t.Errorf("world tension %v, want the base game's 5", got)
	}
}

func TestCountryHistoryCopyBesideTheBaseGameCountsOnce(t *testing.T) {
	game := "capital = 64\nset_oob = \"GER_1936\"\nrecruit_character = GER_adolf_hitler\nadd_political_power = 50\n"
	base, err := NewIndex(map[string][]byte{"history/countries/GER - Germany.txt": []byte(game)})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := NewIndexWithFallback(map[string][]byte{
		"history/countries/GER - zMod.txt": []byte(game + "recruit_character = GER_otto_weber\n"),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	ger := mod.countryPoliticsFor("GER")
	if len(ger.OOBs) != 1 || len(ger.Recruits) != 2 || len(ger.Effects) != len(base.countryPoliticsFor("GER").Effects) {
		t.Errorf("OOBs %v, recruits %v, effects %d (base game %d)", ger.OOBs, ger.Recruits, len(ger.Effects), len(base.countryPoliticsFor("GER").Effects))
	}
}

func TestNarrowCountryHistoryFilesAreNotDropped(t *testing.T) {
	tests := []struct {
		name   string
		script string
		kept   func(*countryPolitics) bool
	}{
		{"ideas", "add_ideas = project_spirit\n", func(p *countryPolitics) bool { return len(p.Ideas) == 1 }},
		{"technologies", "set_technology = { infantry_weapons = 1 }\n", func(p *countryPolitics) bool { return len(p.Technologies) == 1 }},
		{"promotions", "promote_character = TAG_project_leader\n", func(p *countryPolitics) bool { return len(p.Promotions) == 1 }},
		{"effects", "set_stability = 0.8\n", func(p *countryPolitics) bool { return len(p.Effects) == 1 }},
		{"named threats", "add_named_threat = { threat = 5 name = project_threat }\n", func(p *countryPolitics) bool { return len(p.NamedThreats) == 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			idx, err := NewIndex(map[string][]byte{
				"history/countries/TAG - zProject.txt": []byte(test.script),
			})
			if err != nil {
				t.Fatal(err)
			}
			politics := idx.countryPoliticsFor("TAG")
			if politics == nil || !test.kept(politics) {
				t.Fatalf("narrow history was dropped or not parsed: %+v", politics)
			}
		})
	}
}

func TestStartingIdeasApplyDatedRemovalsAcrossHistoryLayers(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Base.txt": []byte(`
add_ideas = { imported_spirit stays }
add_dynamic_modifier = { modifier = imported_dynamic }
`),
	})
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(map[string][]byte{
		"history/countries/TAG - zProject.txt": []byte(`
1938.1.1 = {
	add_ideas = imported_spirit
	add_dynamic_modifier = { modifier = imported_dynamic }
}
1937.1.1 = {
	remove_ideas = { imported_spirit }
	remove_dynamic_modifier = { modifier = imported_dynamic }
}
`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}

	if got := idx.StartingIdeas("TAG", "1936.1.1"); !contains(got, "imported_spirit") || !contains(got, "imported_dynamic") || !contains(got, "stays") {
		t.Fatalf("1936 ideas = %v, want all imported start ideas", got)
	}
	if got := idx.StartingIdeas("TAG", "1937.1.1"); contains(got, "imported_spirit") || contains(got, "imported_dynamic") || !contains(got, "stays") {
		t.Fatalf("1937 ideas = %v, want both removals applied and unrelated idea retained", got)
	}
	if got := idx.StartingIdeas("TAG", "1938.1.1"); !contains(got, "imported_spirit") || !contains(got, "imported_dynamic") || !contains(got, "stays") {
		t.Fatalf("1938 ideas = %v, want the later dated additions restored", got)
	}
}
