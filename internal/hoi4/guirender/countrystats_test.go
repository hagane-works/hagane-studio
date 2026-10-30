package guirender

import (
	"strings"
	"testing"
)

const statLaws = `
ideas = {
	mobilization_laws = {
		volunteer_only = { picture = vol default = yes modifier = { conscription = 0.015 } }
		limited_conscription = { picture = lim modifier = { conscription = 0.025 } }
	}
}
`

const brandenburg = `
state = {
	id = 64
	manpower = 7000000
	provinces = { 6521 }
	history = {
		owner = GER
		buildings = {
			infrastructure = 4
			arms_factory = 5
			industrial_complex = 4
			dockyard = 1
		}
		1939.1.1 = {
			buildings = {
				arms_factory = 10
				industrial_complex = 7
			}
		}
	}
}
`

const danzig = `
state = {
	id = 65
	manpower = 1000000
	provinces = { 100 }
	history = {
		owner = POL
		buildings = { industrial_complex = 2 }
		1939.1.1 = { owner = GER }
	}
}
`

const statHistory = `
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
set_stability = 0.7
set_war_support = 0.3
set_convoys = 200
set_research_slots = 4
add_ideas = { limited_conscription }
`

func statsIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/ideas/00_laws.txt":            []byte(statLaws),
		"common/idea_tags/00_idea.txt":        []byte(lawTags),
		"history/states/64-Brandenburg.txt":   []byte(brandenburg),
		"history/states/65-Danzig.txt":        []byte(danzig),
		"history/countries/GER - Germany.txt": []byte(statHistory),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestCountryStatsCountWhatItOwnsAtThatDate(t *testing.T) {
	idx := statsIndex(t)
	at1936 := idx.CountryStatsPayload("GER", "1936.1.1")

	if at1936["civilian_factories"] != 4 || at1936["military_factories"] != 5 || at1936["dockyards"] != 1 {
		t.Errorf("1936 industry = %v/%v/%v", at1936["civilian_factories"], at1936["military_factories"], at1936["dockyards"])
	}
	if at1936["population"] != 7000000 {
		t.Errorf("1936 population = %v, want Brandenburg's alone", at1936["population"])
	}

	at1939 := idx.CountryStatsPayload("GER", "1939.8.14")

	if at1939["civilian_factories"] != 9 || at1939["military_factories"] != 10 {
		t.Errorf("1939 industry = %v/%v, want the later levels plus Danzig",
			at1939["civilian_factories"], at1939["military_factories"])
	}
	if at1939["population"] != 8000000 {
		t.Errorf("1939 population = %v", at1939["population"])
	}
}

func TestManpowerIsTheShareTheLawAllows(t *testing.T) {
	stats := statsIndex(t).CountryStatsPayload("GER", "1936.1.1")

	if stats["conscription"] != 0.025 || stats["manpower"] != 175000 {
		t.Errorf("manpower = %v at %v", stats["manpower"], stats["conscription"])
	}
}

func TestCountryStatsReadHistoryDirectly(t *testing.T) {
	stats := statsIndex(t).CountryStatsPayload("GER", "1936.1.1")
	if stats["stability"] != 0.7 || stats["war_support"] != 0.3 {
		t.Errorf("stability/war support = %v/%v", stats["stability"], stats["war_support"])
	}
	if stats["convoys"] != 200 || stats["research_slots"] != 4 {
		t.Errorf("convoys/slots = %v/%v", stats["convoys"], stats["research_slots"])
	}
}

func TestPoliticsLinesNameTheIdeologyAndTheNextVote(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"localisation/english/politics_l_english.yml": []byte("l_english:\n" +
			" fascism:0 \"Fascist\"\n democratic:0 \"Democratic\"\n" +
			" POLITICS_NOELECTIONS:0 \"No Elections\"\n" +
			" POLITICS_NEXT_ELECTION:0 \"Next Election $DATE$\"\n" +
			" Nov:0 \"Nov\"\n"),
		"history/countries/GER - Germany.txt": []byte(
			`set_politics = { ruling_party = fascism last_election = "1933.3.5" election_frequency = 48 elections_allowed = no }`),
		"history/countries/USA - USA.txt": []byte(
			`set_politics = { ruling_party = democratic last_election = "1932.11.8" election_frequency = 48 elections_allowed = yes }`),
		"history/countries/XXX - Nowhere.txt": []byte(`set_politics = { ruling_party = oligarchy }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	ger := idx.politicsLines("GER", "1936.1.1")
	if ger["ideology"] != "Fascist" || ger["elections"] != "No Elections" {
		t.Errorf("GER = %v", ger)
	}
	usa := idx.politicsLines("USA", "1936.1.1")
	if usa["elections"] != "Next Election 8 Nov, 1936" {
		t.Errorf("USA 1936 = %v", usa)
	}

	if got := idx.politicsLines("USA", "1939.8.14")["next_election"]; got != "1940.11.8" {
		t.Errorf("USA 1939 next election = %v", got)
	}

	xxx := idx.politicsLines("XXX", "")
	if xxx["ideology"] != "oligarchy" || xxx["elections"] != nil {
		t.Errorf("XXX = %v", xxx)
	}

	if ger["elections_key"] != "POLITICS_NOELECTIONS" || usa["elections_key"] != "POLITICS_NEXT_ELECTION" {
		t.Errorf("elections_key: GER %v, USA %v", ger["elections_key"], usa["elections_key"])
	}
	if _, has := xxx["elections_key"]; has {
		t.Errorf("no election line, no key: %v", xxx)
	}
}

func TestPopularityPieFollowsTheIdeologyFile(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/ideologies/00_ideologies.txt": []byte(`ideologies = {
	democratic = { types = { liberalism = {} } color = { 0 0 255 } }
	communism = { color = { 255 0 0 } }
	fascism = { color = { 150 75 0 } }
	neutrality = { color = { 124 124 124 } }
}`),
		"history/countries/GER - Germany.txt": []byte(
			`set_popularities = { fascism = 40 neutrality = 15 democratic = 35 communism = 10 }`),
		"history/countries/XXX - Nowhere.txt":        []byte(`set_popularities = { neutrality = 100 communism = 0 }`),
		"localisation/english/parties_l_english.yml": []byte("l_english:\n GER_democratic_party:0 \"Zentrum\"\n neutrality:0 \"Non-Aligned\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	pie := idx.popularityPie("GER")
	order := []string{}
	for _, slice := range pie {
		order = append(order, slice["party"].(string))
	}
	if strings.Join(order, ",") != "democratic,communism,fascism,neutrality" {
		t.Errorf("order = %v", order)
	}
	if got := pie[2]["color"].([]int); got[0] != 150 || got[1] != 75 || got[2] != 0 {
		t.Errorf("fascism colour = %v", got)
	}
	if got := idx.popularityPie("XXX"); len(got) != 1 || got[0]["party"] != "neutrality" {
		t.Errorf("zero shares should be left out: %v", got)
	}

	if pie[0]["label"] != "Zentrum" {
		t.Errorf("democratic party label = %v, want Zentrum", pie[0]["label"])
	}
	if got := idx.popularityPie("XXX"); got[0]["label"] != "Non-Aligned" {
		t.Errorf("a country with no party names = %v, want the family's name", got[0]["label"])
	}

	if pie[0]["label_key"] != "GER_democratic_party" {
		t.Errorf("named party key = %v", pie[0]["label_key"])
	}
	if got := idx.popularityPie("XXX"); got[0]["label_key"] != "XXX_neutrality_party" {
		t.Errorf("unnamed party key = %v, want XXX_neutrality_party", got[0]["label_key"])
	}
}
