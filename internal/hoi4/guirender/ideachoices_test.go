package guirender

import (
	"sort"
	"strings"
	"testing"
)

func ideaChoicesIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nset_politics = { ruling_party = fascism }\nadd_ideas = { middle_law }\n"),
		"common/ideas/_laws.txt": []byte(`ideas = {
			mobilization_laws = {
				low_law = { cost = 150 level = 7 modifier = { conscription = 0.01 } }
				middle_law = { cost = 150 level = 5 modifier = { conscription = 0.025 } }
				near_law = { cost = 150 level = 4 modifier = { conscription = 0.05 } }
				other_country_law = { cost = 150 level = 3 allowed = { original_tag = SPR } }
			}
			trade_laws = {
				open_trade = { cost = 150 default = yes modifier = { min_export = 0.8 } }
				shut_trade = { cost = 150 modifier = { min_export = 0 } }
			}
		}`),
		"localisation/english/laws_l_english.yml": []byte("l_english:\n mobilization_laws:0 \"Conscription Law\"\n middle_law:0 \"Middle Law\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func choiceTokens(payload map[string]any) []string {
	out := []string{}
	for _, idea := range payload["ideas"].([]map[string]any) {
		out = append(out, idea["token"].(string))
	}
	return out
}

func TestIdeaChoicesListTheGroupAndMarkTheHeldLaw(t *testing.T) {
	payload := ideaChoicesIndex(t).IdeaChoicesPayload("TAG", "1936.1.1", "mobilization_laws")

	if got := strings.Join(choiceTokens(payload), ","); got != "low_law,middle_law,near_law" {
		t.Errorf("ideas = %s", got)
	}
	if payload["title"] != "Conscription Law" || payload["current"] != "middle_law" || payload["current_label"] != "Middle Law" {
		t.Errorf("title/current = %v / %v / %v", payload["title"], payload["current"], payload["current_label"])
	}
	ideas := payload["ideas"].([]map[string]any)

	if ideas[0]["cost"] != 300.0 || ideas[2]["cost"] != 150.0 || ideas[1]["cost"] != nil || ideas[1]["current"] != true {
		t.Errorf("costs = %v / %v / %v", ideas[0]["cost"], ideas[1]["cost"], ideas[2]["cost"])
	}
	if ideas[1]["stats"] != "£manpower_texticon §G+2.5%§!" {
		t.Errorf("stats = %v", ideas[1]["stats"])
	}
}

func TestIdeaChoicesFallBackToTheDefaultLaw(t *testing.T) {
	payload := ideaChoicesIndex(t).IdeaChoicesPayload("TAG", "1936.1.1", "trade_laws")
	if payload["current"] != "open_trade" {
		t.Errorf("current = %v, want the group's default", payload["current"])
	}

	ideas := payload["ideas"].([]map[string]any)
	if ideas[1]["cost"] != 150.0 || ideas[0]["stats"] != "§Y80%§!" {
		t.Errorf("shut_trade cost = %v, open_trade stats = %v", ideas[1]["cost"], ideas[0]["stats"])
	}
}

func TestRealGameIdeaChoicesMatchTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	cases := []struct{ group, current, tokens string }{
		{"mobilization_laws", "limited_conscription", "disarmed_nation,volunteer_only,limited_conscription," +
			"extensive_conscription,service_by_requirement,all_adults_serve,scraping_the_barrel"},
		{"trade_laws", "limited_exports", "free_trade,export_focus,limited_exports,closed_economy"},
		{"economy", "partial_economic_mobilisation", "civilian_economy,low_economic_mobilisation," +
			"partial_economic_mobilisation,war_economy,tot_economic_mobilisation"},
	}
	for _, c := range cases {
		payload := idx.IdeaChoicesPayload("GER", "1936.1.1.12", c.group)
		if payload["current"] != c.current || strings.Join(choiceTokens(payload), ",") != c.tokens {
			t.Errorf("%s: current %v, ideas %v", c.group, payload["current"], choiceTokens(payload))
		}
	}
	laws := idx.IdeaChoicesPayload("GER", "1936.1.1.12", "mobilization_laws")["ideas"].([]map[string]any)
	costs := []any{}
	for _, law := range laws[:5] {
		costs = append(costs, law["cost"])
	}

	if costs[0] != 300.0 || costs[1] != 150.0 || costs[2] != nil || costs[3] != 150.0 || costs[4] != 300.0 {
		t.Errorf("conscription costs = %v", costs)
	}
}

func TestRealGameAdvisorPickerMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload, ok := idx.CharactersPayloadAt("GER", "", "1936.1.1.12")
	if !ok {
		t.Fatal("no payload")
	}
	available, rest := []string{}, []string{}
	costs := map[string]any{}
	for _, advisor := range payload["advisors"].([]map[string]any) {
		if advisor["slot"] != "political_advisor" || advisor["shown"] != true {
			continue
		}
		name := advisor["name"].(string)
		costs[name] = advisor["cost"]
		if advisor["available"] == true {
			available = append(available, name)
		} else {
			rest = append(rest, name)
		}
	}
	sort.Strings(available)
	sort.Strings(rest)
	if got := strings.Join(available, ","); got != "Franz Seldte,Hanns Kerrl,Hjalmar Schacht,Reinhard Heydrich,Walther Funk" {
		t.Errorf("hireable = %s", got)
	}

	if len(rest) < 4 || strings.Join(rest[:4], ",") != "Adolf Friedrich of Mecklenburg,Alfred Hugenberg,Andreas Hermes,August Thalheimer" {
		t.Errorf("the rest = %v", rest)
	}

	for _, name := range append(available, rest...) {
		if name == "Fritz Todt" {
			t.Errorf("Fritz Todt is listed")
		}
	}
	if costs["Hjalmar Schacht"] != 75 || costs["Franz Seldte"] != 150 || costs["Konstantin von Neurath"] != 50 {
		t.Errorf("costs = %v / %v / %v", costs["Hjalmar Schacht"], costs["Franz Seldte"], costs["Konstantin von Neurath"])
	}
}

func TestRealGameConcernPickerMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	payload := idx.IdeaChoicesPayload("GER", "1936.1.1.12", "industrial_concern")
	if payload["layout"] != "grid" || payload["subtitle"] != "Select §YIndustrial Concern§!" {
		t.Errorf("layout / subtitle = %v / %v", payload["layout"], payload["subtitle"])
	}
	names, traits := []string{}, []string{}
	for _, idea := range payload["ideas"].([]map[string]any) {
		names = append(names, idea["label"].(string))
		trait, _ := idea["trait"].(string)
		traits = append(traits, trait)
	}
	want := "IG Farben,Krupp,Siemens,Vereinigte Stahlwerke,Deutsche Reichsbahn,Philipp Holzmann,RWE,Reichswerke"
	if strings.Join(names, ",") != want {
		t.Errorf("concerns = %v", names)
	}
	if traits[0] != "Refining Concern" || traits[7] != "Conglomerate" {
		t.Errorf("traits = %v", traits)
	}
}

func TestIdeaChoicesReadInThePageLanguage(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nadd_ideas = { middle_law }\n"),
		"common/ideas/_laws.txt": []byte(`ideas = {
			mobilization_laws = {
				low_law = { cost = 150 level = 7 modifier = { conscription = 0.01 } }
				middle_law = { cost = 150 level = 5 modifier = { conscription = 0.025 } }
			}
			industrial_concern = { concern_a = { cost = 150 } }
		}`),
		"localisation/english/laws_l_english.yml": []byte("l_english:\n mobilization_laws:0 \"Conscription Law\"\n" +
			" middle_law:0 \"Middle Law\"\n low_law:0 \"Low Law\"\n industrial_concern:0 \"Industrial Concern\"\n" +
			" MODIFIER_CONSCRIPTION_FACTOR:0 \"Recruitable Population\"\n"),
		"localisation/french/laws_l_french.yml": []byte("\ufeffl_french:\n mobilization_laws:0 \"Législation sur la conscription\"\n" +
			" middle_law:0 \"Service court\"\n low_law:0 \"Volontaires\"\n industrial_concern:0 \"Groupe industriel\"\n" +
			" MODIFIER_CONSCRIPTION_FACTOR:0 \"Population recrutable\"\n" +
			" IDEA_WINDOW_SWITCH:0 \"Remplacer §Y$IDEA$§! ?\"\n IDEA_WINDOW_ADD:0 \"Sélectionner un $GROUP|Y$\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	tooltipOf := func(payload map[string]any, token string) string {
		for _, idea := range payload["ideas"].([]map[string]any) {
			if idea["token"] == token {
				tip, _ := idea["tooltip"].(string)
				return tip
			}
		}
		return ""
	}

	fr := idx.InLanguage("fr", nil)
	law := fr.IdeaChoicesPayload("TAG", "1936.1.1", "mobilization_laws")
	if law["title"] != "Législation sur la conscription" || law["subtitle"] != "Remplacer §YService court§! ?" {
		t.Errorf("French window = %v / %v", law["title"], law["subtitle"])
	}
	if got := tooltipOf(law, "low_law"); got != "§HVolontaires§!\n\nPopulation recrutable: §G+1.0%§!" {
		t.Errorf("French law tooltip = %q", got)
	}
	if grid := fr.IdeaChoicesPayload("TAG", "1936.1.1", "industrial_concern"); grid["subtitle"] != "Sélectionner un §YGroupe industriel§!" {
		t.Errorf("French grid line = %v", grid["subtitle"])
	}

	en := idx.IdeaChoicesPayload("TAG", "1936.1.1", "mobilization_laws")
	if en["subtitle"] != "Replace §YMiddle Law?§!" {
		t.Errorf("English window line = %v", en["subtitle"])
	}
	if got := tooltipOf(en, "low_law"); !strings.Contains(got, "Recruitable Population: §G+1.0%§!") {
		t.Errorf("English law tooltip = %q", got)
	}
}
