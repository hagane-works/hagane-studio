package guirender

import (
	"strings"
	"testing"
)

func decisionStartIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/country_tags/00_countries.txt": []byte("TAG = \"countries/Tag.txt\"\nHOL = \"countries/Holland.txt\"\nSPR = \"countries/Spain.txt\"\n"),
		"common/ideologies/00_ideologies.txt": []byte(`ideologies = {
			democratic = { types = { liberalism = {} } }
			fascism = { types = { nazism = {} } }
			communism = { types = { marxism = {} } }
		}`),
		"history/countries/TAG - Tag.txt": []byte(`capital = 1
			set_politics = { ruling_party = fascism }
			set_popularities = { fascism = 60 communism = 10 democratic = 30 }
			activate_mission = TAG_bills_due
		`),
		"history/countries/HOL - Holland.txt": []byte("capital = 2\nset_politics = { ruling_party = democratic }\n"),
		"history/states/1-Home.txt":           []byte("state = { id = 1 provinces = { 1 } history = { owner = TAG } }\n"),
		"history/states/2-Holland.txt":        []byte("state = { id = 2 provinces = { 2 } history = { owner = HOL } }\n"),
		"common/decisions/categories/cats.txt": []byte(`
			crisis = { priority = 100 visible = { has_war = yes } }
			propaganda = { }
			political = { }
			bills = { priority = 12 allowed = { original_tag = TAG } }
			gateway = {
				priority = 10
				allowed = { original_tag = TAG }
				visible_when_empty = yes
				visible = { HOL = { exists = yes is_subject = no } }
			}
		`),
		"common/decisions/decisions.txt": []byte(`
			crisis = { strikes = { } }
			propaganda = {
				war_propaganda = { visible = { has_war = yes } }
				radio_propaganda = {
					visible = {
						if = { limit = { tag = SPR } NOT = { has_idea = SPR_violence } }
					}
				}
			}
			political = {
				ban_communism = { visible = { NOT = { has_government = communism } communism > 0 } }
				ban_democrats = { visible = { democratic > 0.5 } }
			}
			bills = {
				TAG_bills_due = { allowed = { always = no } days_mission_timeout = 180 }
				TAG_cancel_bills = { }
				TAG_idle_mission = { days_mission_timeout = 30 activation = { always = no } }
			}
			gateway = {
				demand_trade = { targets = { HOL } target_trigger = { FROM = { tag = HOL } } }
				demand_anything = { target_trigger = { FROM = { exists = yes } } }
			}
		`),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func shownList(payload map[string]any) []string {
	out := []string{}
	categories, _ := payload["categories"].([]map[string]any)
	for _, category := range categories {
		line := category["name"].(string) + ":"
		if category["shown"] != true {
			line = "-" + line
		}
		rows := []string{}
		for _, row := range category["decisions"].([]map[string]any) {
			name := row["name"].(string)
			if row["shown"] != true {
				name = "-" + name
			}
			rows = append(rows, name)
		}
		out = append(out, line+" "+strings.Join(rows, ","))
	}
	return out
}

func TestDecisionsPayloadAtShowsWhatTheStartShows(t *testing.T) {
	got := shownList(decisionStartIndex(t).DecisionsPayloadAt("TAG", "1936.1.1"))
	want := []string{

		"bills: TAG_cancel_bills,TAG_bills_due,-TAG_idle_mission",
		"gateway: demand_trade,-demand_anything",
		"propaganda: radio_propaganda,-war_propaganda",
		"political: ban_communism,-ban_democrats",

		"-crisis: -strikes",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("payload =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestDecisionsPayloadAtMarksMissions(t *testing.T) {
	payload := decisionStartIndex(t).DecisionsPayloadAt("TAG", "1936.1.1")
	if row := decisionNamed(payload, "bills", "TAG_bills_due"); row == nil || row["mission"] != true {
		t.Errorf("the running mission is not marked: %v", row)
	}
	if row := decisionNamed(payload, "bills", "TAG_cancel_bills"); row == nil || row["mission"] != nil {
		t.Errorf("an ordinary decision is marked a mission: %v", row)
	}
}

func TestRealGameDecisionsShownAtStart(t *testing.T) {
	idx := realTopbarIndex(t, false)
	shown := []string{}
	for _, line := range shownList(idx.DecisionsPayloadAt("GER", "1936.1.1.12")) {
		if strings.HasPrefix(line, "-") {
			continue
		}
		rows := []string{}
		for _, row := range strings.Split(strings.SplitN(line, " ", 2)[1], ",") {
			if row != "" && !strings.HasPrefix(row, "-") {
				rows = append(rows, row)
			}
		}
		shown = append(shown, strings.SplitN(line, ":", 2)[0]+": "+strings.Join(rows, ","))
	}
	want := []string{
		"GER_mefo_bills_category: GER_cancel_mefos,GER_mefo_bills_mission",
		"HOL_gateway_to_europe_ger_category: HOL_demand_increased_dutch_trade_with_germany",
		"propaganda_efforts: war_propaganda_radio_industry,war_propaganda_film_industry",
		"political_actions: improved_worker_conditions,anti_democratic_raids,anti_communist_raids," +
			"ban_democratic_party,ban_communist_party,institute_press_censorship_fascist_state",
	}
	if strings.Join(shown, "\n") != strings.Join(want, "\n") {
		t.Errorf("GER 1936 shown =\n%s\nwant\n%s", strings.Join(shown, "\n"), strings.Join(want, "\n"))
	}

	payload := idx.DecisionsPayloadAt("GER", "1936.1.1.12")
	if row := decisionNamed(payload, "propaganda_efforts", "war_propaganda_film_industry"); row == nil ||
		row["label"] != "Utilize Domestic Film Industry" {
		t.Errorf("film decision = %v, want the recording's Utilize Domestic Film Industry", row)
	}
}

func TestCharacterScopeAsksWhoIsHired(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nadd_ideas = { TAG_hired_idea }\n"),
		"common/characters/TAG.txt": []byte(`characters = {
			TAG_hired = { name = TAG_hired advisor = { slot = political_advisor idea_token = TAG_hired_idea } }
			TAG_free = { name = TAG_free advisor = { slot = political_advisor idea_token = TAG_free_idea } }
		}`),
		"common/ideas/advisors.txt": []byte("ideas = { political_advisor = { TAG_hired_idea = { } TAG_free_idea = { } } }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	state := idx.decisionStartOf("TAG", "1936.1.1")
	cases := []struct {
		trigger       string
		holds, decide bool
	}{
		{"TAG_hired = { is_hired_as_advisor = yes }", true, true},
		{"TAG_free = { is_hired_as_advisor = no }", true, true},
		{"TAG_free = { is_hired_as_advisor = yes }", false, true},

		{"TAG_free = { is_country_leader = yes }", false, false},
	}
	for _, c := range cases {
		root, ok := parseScript([]byte(c.trigger))
		if !ok {
			t.Fatalf("parse %q", c.trigger)
		}
		holds, decided := state.eval(root.Entries, false)
		if holds != c.holds || decided != c.decide {
			t.Errorf("%s = %v/%v, want %v/%v", c.trigger, holds, decided, c.holds, c.decide)
		}
	}
}

func TestDecisionsPayloadAtLeavesOutOtherCountriesHiddenOnes(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/country_tags/00_countries.txt": []byte("TAG = \"countries/Tag.txt\"\nHOL = \"countries/Holland.txt\"\n"),
		"history/countries/TAG - Tag.txt":      []byte("capital = 1\n"),
		"common/decisions/categories/cats.txt": []byte(`
			political = { }
			TAG_own = { visible = { has_war = yes } }
			HOL_secret = { visible = { has_war = yes } }
			HOL_open = { }
		`),
		"common/decisions/decisions.txt": []byte(`
			political = {
				HOL_shown = { }
				HOL_hidden = { visible = { has_war = yes } }
				TAG_hidden = { visible = { has_war = yes } }
				generic_hidden = { visible = { has_war = yes } }
				NORDIC_hidden = { visible = { has_war = yes } }
			}
			TAG_own = { TAG_war_plan = { } }
			HOL_secret = { HOL_war_plan = { } }
			HOL_open = { HOL_open_hidden = { visible = { has_war = yes } } HOL_open_shown = { } }
		`),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(shownList(idx.DecisionsPayloadAt("TAG", "1936.1.1")), "\n")
	want := strings.Join([]string{
		"political: HOL_shown,-TAG_hidden,-generic_hidden,-NORDIC_hidden",
		"HOL_open: HOL_open_shown",
		"-TAG_own: -TAG_war_plan",
	}, "\n")
	if got != want {
		t.Errorf("TAG's screen =\n%s\nwant\n%s", got, want)
	}
}

func TestRealGameItalyDecisionsAreItalys(t *testing.T) {
	idx := realTopbarIndex(t, false)

	tags := map[string]bool{}
	for _, tag := range idx.ListCountryTags() {
		tags[tag] = true
	}
	foreign := func(name string) bool {
		return len(name) > 3 && name[3] == '_' && name[:3] != "ITA" && tags[name[:3]]
	}
	shown := 0
	for _, line := range shownList(idx.DecisionsPayloadAt("ITA", "1936.1.1.12")) {
		tab, rows, _ := strings.Cut(line, ": ")
		for _, row := range append([]string{strings.TrimSuffix(tab, ":")}, strings.Split(rows, ",")...) {
			if row == "" {
				continue
			}
			if !strings.HasPrefix(row, "-") {
				shown++
				continue
			}
			if foreign(strings.TrimPrefix(row, "-")) {
				t.Errorf("Italy's screen lists %s", row)
			}
		}
	}
	if shown < 8 {
		t.Errorf("only %d shown tabs and rows on Italy's 1936 screen", shown)
	}
}
