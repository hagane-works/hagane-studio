package guirender

import (
	"strings"
	"testing"
)

const bookmarkFile = `
bookmarks = {
	bookmark = {
		name = "GATHERING_STORM_NAME"
		date = 1936.1.1.12
		picture = "GFX_select_date_1936"
		default_country = "GER"
		default = yes
		filters = { new_content continent }

		"GER"={
			history = "GER_GATHERING_STORM_DESC"
			ideology = fascism
			version = base_game
			ideas = { GER_mefo_bills_1936_fake_ns }
			focuses = { GER_rhineland }
		}
		"GER"={
			history = "GER_GATHERING_STORM_DESC"
			ideology = fascism
			version = waking_the_tiger
			ideas = { GER_something_else }
		}
		"FRA"={
			history = "FRA_GATHERING_STORM_DESC"
			ideology = democratic
			ideas = { FRA_victors_of_wwi }
			focuses = { }
		}
	}
	bookmark = {
		name = "BLITZKRIEG_NAME"
		date = 1939.8.14.12
		default_country = "GER"
		"GER"={ history = "GER_BLITZKRIEG_DESC" ideology = fascism }
	}
}
`

const ideologyFile = `
ideologies = {
	democratic = { types = { conservatism = {} liberalism = {} } }
	fascism = { types = { nazism = {} } }
	neutrality = { types = { despotism = {} } }
}
`

const tagFile = `
GER = "countries/Germany.txt"
FRA = "countries/France.txt"
SIA = "countries/Siam.txt"
D01 = "countries/Dynamic.txt"
`

const gerHistory = `
capital = 64
set_politics = {
	ruling_party = fascism
	last_election = "1933.3.5"
	elections_allowed = no
}
set_popularities = { fascism = 40 democratic = 30 communism = 30 }
`

const fraHistory = `
if = {
	limit = { has_dlc = "La Resistance" }
	set_politics = { ruling_party = democratic elections_allowed = yes }
	set_popularities = { democratic = 64 fascism = 16 }
}
`

const siaHistory = `set_politics = { ruling_party = neutrality elections_allowed = no }`

const gerCharacters = `
characters = {
	GER_ernst_thalmann = {
		name = GER_ernst_thalmann
		portraits = { civilian = { large = "GFX_portrait_GER_ernst_thalmann" } }
		country_leader = { ideology = marxism }
	}
	GER_adolf_hitler = {
		name = GER_adolf_hitler
		portraits = { civilian = { large = "GFX_portrait_GER_adolf_hitler" } }
		country_leader = { ideology = nazism }
	}
}
`

const ideaFile = `
ideas = {
	country = {
		GER_mefo_bills_1936_fake_ns = {
			name = GER_mefo_bills_modifier
			picture = ger_mefo_bills
		}
		FRA_victors_of_wwi = {
			removal_cost = -1
			picture = generic_victors_of_ww1
		}
	}
}
`

const countryLoc = "l_english:\n GER_fascism: \"German Reich\"\n FRA_democratic: \"France\"\n SIA_neutrality: \"Siam\"\n" +
	" fascism: \"Fascist\"\n democratic: \"Democratic\"\n GATHERING_STORM_NAME: \"Gathering Storm\"\n" +
	" GER_GATHERING_STORM_DESC: \"A new Germany has risen.\"\n GER_adolf_hitler: \"Adolf Hitler\"\n"

const focusTreeFile = `
focus_tree = {
	id = german_focus
	country = {
		factor = 0
		modifier = { add = 10 tag = GER }
	}
	focus = { id = GER_rhineland icon = GFX_focus_rhineland x = 1 y = 0 cost = 10 }
}
focus_tree = {
	id = german_dlc_focus
	country = {
		factor = 0
		modifier = { add = 1000 tag = GER has_dlc = "Waking the Tiger" }
	}
}
focus_tree = {
	id = italian_focus
	country = {
		factor = 0
		modifier = { add = 10 OR = { tag = ITA tag = RSI } }
	}
}
`

func countryIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/the_gathering_storm.txt": []byte(bookmarkFile),
		"common/ideologies/00_ideologies.txt":      []byte(ideologyFile),
		"common/country_tags/00_countries.txt":     []byte(tagFile),
		"common/ideas/ger.txt":                     []byte(ideaFile),
		"common/characters/GER.txt":                []byte(gerCharacters),
		"common/national_focus/germany.txt":        []byte(focusTreeFile),
		"history/countries/GER - Germany.txt":      []byte(gerHistory),
		"history/countries/FRA - France.txt":       []byte(fraHistory),
		"history/countries/SIA - Siam.txt":         []byte(siaHistory),
		"localisation/english/countries.yml":       []byte(countryLoc),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestBookmarksKeepScenarioOrderAndOneEntryPerCountry(t *testing.T) {
	payload := countryIndex(t).BookmarksPayload()
	marks, _ := payload["bookmarks"].([]map[string]any)
	if len(marks) != 2 {
		t.Fatalf("bookmarks = %d, want both scenarios", len(marks))
	}

	if marks[0]["date"] != "1936.1.1.12" {
		t.Errorf("first scenario = %v", marks[0]["date"])
	}
	if marks[0]["label"] != "Gathering Storm" {
		t.Errorf("label = %v, want the localised name", marks[0]["label"])
	}
	countries, _ := marks[0]["countries"].([]map[string]any)
	if len(countries) != 2 {
		t.Fatalf("countries = %d, want GER once and FRA", len(countries))
	}
	if countries[0]["tag"] != "GER" || countries[1]["tag"] != "FRA" {
		t.Fatalf("countries = %v", countries)
	}

	ideas, _ := countries[0]["ideas"].([]map[string]any)
	if len(ideas) != 1 || ideas[0]["token"] != "GER_mefo_bills_1936_fake_ns" {
		t.Errorf("ideas = %v, want the base-game entry only", ideas)
	}
	if countries[0]["history"] != "A new Germany has risen." {
		t.Errorf("history = %v", countries[0]["history"])
	}
}

func TestCountryCardReadsStartingPolitics(t *testing.T) {
	payload := countryIndex(t).BookmarksPayload()
	marks, _ := payload["bookmarks"].([]map[string]any)
	countries, _ := marks[0]["countries"].([]map[string]any)
	ger := countries[0]
	if ger["name"] != "German Reich" {
		t.Errorf("name = %v, want the ideology's own name", ger["name"])
	}
	if ger["ruling_party_label"] != "Fascist" || ger["ruling_party_share"] != 40 {
		t.Errorf("government = %v / %v", ger["ruling_party_label"], ger["ruling_party_share"])
	}
	if ger["elections_allowed"] != false {
		t.Errorf("elections = %v, want a declared no", ger["elections_allowed"])
	}

	fra := countries[1]
	if fra["name"] != "France" || fra["ruling_party_share"] != 64 {
		t.Errorf("france = %v", fra)
	}

	if ger["name_key"] != "GER_fascism" || fra["name_key"] != "FRA_democratic" {
		t.Errorf("name keys = %v / %v", ger["name_key"], fra["name_key"])
	}

	if got := countryIndex(t).CountryNameKey("SIA", "fascism"); got != "SIA" {
		t.Errorf("fallback name key = %q, want SIA", got)
	}
}

func TestBookmarkCarriesItsDescriptionKey(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/storm.txt":           []byte(`bookmarks = { bookmark = { name = STORM desc = STORM_DESC date = 1936.1.1.12 default_country = "GER" "GER" = { ideology = fascism } } }`),
		"localisation/english/b_l_english.yml": []byte("l_english:\n STORM:0 \"Storm\"\n STORM_DESC:0 \"Clouds gather.\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	marks, _ := idx.BookmarksPayload()["bookmarks"].([]map[string]any)
	if len(marks) != 1 || marks[0]["desc"] != "Clouds gather." || marks[0]["desc_key"] != "STORM_DESC" {
		t.Errorf("bookmark = %v", marks)
	}
}

func TestMinorListSkipsTagsWithNoStartingPolitics(t *testing.T) {
	payload := countryIndex(t).BookmarksPayload()
	minors, _ := payload["minors"].([]map[string]any)
	tags := map[string]bool{}
	for _, minor := range minors {
		tags[minor["tag"].(string)] = true
	}
	if !tags["SIA"] {
		t.Error("a playable minor is missing from the list")
	}
	if tags["D01"] {
		t.Error("D01 has no history file: it is a released-puppet placeholder, not a country")
	}
	if tags["GER"] || tags["FRA"] {
		t.Error("bookmark countries must not repeat in the minor list")
	}

	for _, minor := range minors {
		if minor["tag"] == "SIA" && minor["name"] != "Siam" {
			t.Errorf("SIA name = %v, want the ruling party's name", minor["name"])
		}
	}
}

func TestMinorListIsTheCountriesOwningLand(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/the_gathering_storm.txt": []byte(bookmarkFile),
		"common/ideologies/00_ideologies.txt":      []byte(ideologyFile),
		"common/country_tags/00_countries.txt":     []byte(tagFile),
		"common/country_tags/zz_new.txt":           []byte(`ZQX = "countries/Zeta.txt"` + "\n" + `ZQY = "countries/Zeta.txt"`),
		"history/countries/GER - Germany.txt":      []byte(gerHistory),
		"history/countries/FRA - France.txt":       []byte(fraHistory),
		"history/countries/SIA - Siam.txt":         []byte(siaHistory),
		"history/countries/ZQX - Zeta.txt":         []byte(siaHistory),
		"history/countries/ZQY - Zeta.txt":         []byte(siaHistory),
		"history/states/1-Siam.txt":                []byte(`state = { id = 1 name = "STATE_1" provinces = { 1 } history = { owner = SIA } }`),
		"history/states/2-Later.txt":               []byte(`state = { id = 2 name = "STATE_2" provinces = { 2 } history = { owner = SIA 1939.1.1 = { owner = ZQY } } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	minors, _ := idx.BookmarksPayload()["minors"].([]map[string]any)
	tags := map[string]bool{}
	for _, minor := range minors {
		tags[minor["tag"].(string)] = true
	}
	if !tags["SIA"] || tags["ZQX"] || tags["ZQY"] {
		t.Errorf("minors %v: want SIA (owns a state), not ZQX (no land) nor ZQY (land only from 1939)", tags)
	}
}

func TestCountryWithNobodyStillHasItsPoliticsScreen(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/the_gathering_storm.txt": []byte(bookmarkFile),
		"common/ideologies/00_ideologies.txt":      []byte(ideologyFile),
		"common/country_tags/00_countries.txt":     []byte(tagFile),
		"common/country_tags/zz_new.txt":           []byte(`ZQX = "countries/Zeta.txt"`),
		"history/countries/GER - Germany.txt":      []byte(gerHistory),
		"history/countries/ZQX - Zeta.txt":         []byte(siaHistory),
	})
	if err != nil {
		t.Fatal(err)
	}
	people, ok := idx.CharactersPayloadAt("ZQX", "", "1936.1.1")
	if !ok {
		t.Fatal("ZQX is a country: want its (empty) politics payload")
	}
	if leader, _ := people["leader"].(map[string]any); leader != nil {
		t.Errorf("leader = %v, want none", leader)
	}
	if advisors, _ := people["advisors"].([]map[string]any); advisors == nil || len(advisors) != 0 {
		t.Errorf("advisors = %#v, want an empty list", people["advisors"])
	}
	if _, ok := idx.OfficerCorpsPayload("ZQX", "1936.1.1"); !ok {
		t.Error("ZQX is a country: want its (empty) officer corps")
	}
	if _, ok := idx.CharactersPayloadAt("QQQ", "", "1936.1.1"); ok {
		t.Error("QQQ is no country: want not found")
	}
	if _, ok := idx.OfficerCorpsPayload("QQQ", "1936.1.1"); ok {
		t.Error("QQQ is no country: want not found")
	}
}

func TestLeaderFollowsTheRulingParty(t *testing.T) {
	idx := countryIndex(t)
	people, ok := idx.CharactersPayloadFor("GER", "")
	if !ok {
		t.Fatal("no characters for GER")
	}
	leader, _ := people["leader"].(map[string]any)

	if leader["name"] != "Adolf Hitler" {
		t.Errorf("leader = %v, want the fascist one", leader["name"])
	}
	communist, _ := idx.CharactersPayloadFor("GER", "communism")
	if leader2, _ := communist["leader"].(map[string]any); leader2["token"] != "GER_ernst_thalmann" {
		t.Errorf("communist leader = %v", leader2)
	}
	if idx.PartyOfIdeology("despotism") != "neutrality" {
		t.Errorf("despotism maps to %q", idx.PartyOfIdeology("despotism"))
	}

	if idx.PartyOfIdeology("technocracy") != "technocracy" {
		t.Error("an unknown ideology should map to itself")
	}
}

func TestFocusTreeForTagTakesTheHighestScoringTree(t *testing.T) {
	idx := countryIndex(t)
	if got := idx.FocusTreeForTag("GER"); got != "german_dlc_focus" {
		t.Errorf("GER tree = %q, want the add=1000 variant", got)
	}

	if got := idx.FocusTreeForTag("ITA"); got != "italian_focus" {
		t.Errorf("ITA tree = %q", got)
	}
	if got := idx.FocusTreeForTag("SIA"); got != "" {
		t.Errorf("SIA has no tree, got %q", got)
	}
}

func TestSpiritIconsComeFromTheIdeasPicture(t *testing.T) {
	payload := countryIndex(t).BookmarksPayload()
	marks, _ := payload["bookmarks"].([]map[string]any)
	countries, _ := marks[0]["countries"].([]map[string]any)
	ideas, _ := countries[0]["ideas"].([]map[string]any)

	if ideas[0]["icon"] != nil {
		t.Errorf("icon = %v, but this fixture declares no sprites", ideas[0]["icon"])
	}
	if got := countryIndex(t).ideaPicture("FRA_victors_of_wwi"); got != "generic_victors_of_ww1" {
		t.Errorf("picture = %q", got)
	}
}

const continentFile = `
continents = {
	europe
	north_america
	asia
}
`

const provinceDefinitions = `0;0;0;0;land;false;unknown;0
6521;230;81;119;land;false;plains;1
9999;10;20;30;land;false;plains;3
`

const berlinState = `
state = {
	id = 64
	name = "STATE_64"
	provinces = { 6521 3522 }
	history = { owner = GER }
}
`

const tokyoState = `
state = {
	id = 282
	provinces = { 9999 }
	history = { owner = JAP }
}
`

func continentIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/the_gathering_storm.txt": []byte(bookmarkFile),
		"common/country_tags/00_countries.txt":     []byte(tagFile),
		"history/countries/GER - Germany.txt":      []byte("capital = 64\n" + gerHistory),
		"history/countries/SIA - Siam.txt":         []byte("capital = 282\n" + siaHistory),
		"history/countries/FRA - France.txt":       []byte(fraHistory),
		"history/states/64-Brandenburg.txt":        []byte(berlinState),
		"history/states/282-Tokyo.txt":             []byte(tokyoState),
		"map/continent.txt":                        []byte(continentFile),
		"map/definition.csv":                       []byte(provinceDefinitions),
		"localisation/english/countries.yml":       []byte(countryLoc),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestContinentWalksCapitalToProvinceToContinent(t *testing.T) {
	idx := continentIndex(t)
	if got := idx.ContinentOf("GER"); got != "europe" {
		t.Errorf("GER continent = %q, want europe", got)
	}
	if got := idx.ContinentOf("SIA"); got != "asia" {
		t.Errorf("SIA continent = %q, want asia", got)
	}

	if got := idx.ContinentOf("FRA"); got != "" {
		t.Errorf("FRA continent = %q, want empty", got)
	}
	if got := idx.ContinentOf("ZZZ"); got != "" {
		t.Errorf("unknown tag returned %q", got)
	}
}

func TestBookmarkOffersTheFiltersItDeclares(t *testing.T) {
	payload := continentIndex(t).BookmarksPayload()
	marks, _ := payload["bookmarks"].([]map[string]any)
	filters, _ := marks[0]["filters"].([]string)
	if len(filters) != 2 || filters[0] != "new_content" || filters[1] != "continent" {
		t.Fatalf("filters = %v", filters)
	}
	continents, _ := marks[0]["continents"].([]map[string]any)
	if len(continents) != 3 || continents[0]["name"] != "europe" {
		t.Errorf("continent buttons = %v, want them in declaration order", continents)
	}

	if _, present := marks[1]["continents"]; present {
		t.Error("a scenario without a continent filter must not carry the buttons")
	}
}

const dynamicModifierFile = `
CHI_nine_power_treaty_sea = {
	icon = GFX_idea_CHI_nine_power_treaty_sea
	consumer_goods_factor = 0.1
}
`

const startStateHistory = `
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
add_ideas = { bitter_loser }
add_dynamic_modifier = { modifier = CHI_nine_power_treaty_sea }
IF = {
	limit = { has_dlc = "No Step Back" }
	add_ideas = { dlc_spirit }
	else = { add_ideas = { no_dlc_spirit } }
}
if = {
	limit = { has_country_flag = something_we_cannot_know }
	add_ideas = { conditional_spirit }
}
1939.1.1 = {
	add_ideas = {
		war_economy
		GER_hired_advisor_idea
	}
}
`

const startStateIdeas = `
ideas = {
	country = {
		bitter_loser = { picture = ger_bitter_loser }
		dlc_spirit = { picture = dlc_spirit_pic }
		no_dlc_spirit = { picture = no_dlc_pic }
		conditional_spirit = { picture = conditional_pic }
	}
	economy = {
		war_economy = { picture = war_economy_pic }
	}
	political_advisor = {
		GER_hired_advisor_idea = { picture = advisor_pic }
	}
}
`

const startStateCharacters = `
characters = {
	GER_someone = {
		name = GER_someone
		portraits = { civilian = { small = "GFX_portrait_GER_someone_small" } }
		advisor = { slot = political_advisor idea_token = GER_hired_advisor_idea }
	}
	GER_other = {
		name = GER_other
		portraits = { civilian = { small = "GFX_portrait_GER_other_small" } }
		advisor = { slot = political_advisor idea_token = GER_other_idea }
	}
}
`

func startStateIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/countries/GER - Germany.txt":     []byte(startStateHistory),
		"common/ideas/00_ideas.txt":               []byte(startStateIdeas),
		"common/dynamic_modifiers/00_dynamic.txt": []byte(dynamicModifierFile),
		"common/characters/GER.txt":               []byte(startStateCharacters),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestStartingIdeasFollowTheScenarioDate(t *testing.T) {
	idx := startStateIndex(t)
	at1936 := idx.StartingIdeas("GER", "1936.1.1")
	if contains(at1936, "war_economy") {
		t.Errorf("1936 picked up the 1939 block: %v", at1936)
	}
	if !contains(at1936, "bitter_loser") {
		t.Errorf("1936 is missing the top-level ideas: %v", at1936)
	}
	at1939 := idx.StartingIdeas("GER", "1939.8.14")
	if !contains(at1939, "war_economy") || !contains(at1939, "bitter_loser") {
		t.Errorf("1939 should hold both blocks: %v", at1939)
	}
}

func TestStartingIdeasDecideOnlyWhatTheyCan(t *testing.T) {
	ideas := startStateIndex(t).StartingIdeas("GER", "1936.1.1")

	if !contains(ideas, "dlc_spirit") || contains(ideas, "no_dlc_spirit") {
		t.Errorf("DLC conditional resolved wrong: %v", ideas)
	}

	if contains(ideas, "conditional_spirit") {
		t.Errorf("an undecidable condition was taken anyway: %v", ideas)
	}
}

func TestStartStateSplitsSpiritsFromHiredAdvisors(t *testing.T) {
	idx := startStateIndex(t)
	payload, ok := idx.CharactersPayloadAt("GER", "", "1939.8.14")
	if !ok {
		t.Fatal("no payload")
	}
	spirits, _ := payload["spirits"].([]map[string]any)
	tokens := []string{}
	for _, spirit := range spirits {
		tokens = append(tokens, spirit["token"].(string))
	}
	if !contains(tokens, "bitter_loser") || !contains(tokens, "dlc_spirit") {
		t.Errorf("spirits = %v", tokens)
	}

	if !contains(tokens, "CHI_nine_power_treaty_sea") {
		t.Errorf("dynamic modifier missing from the spirit row: %v", tokens)
	}

	if len(tokens) == 0 || tokens[len(tokens)-1] != "CHI_nine_power_treaty_sea" {
		t.Errorf("spirit order = %v, want the dynamic modifier after the ideas", tokens)
	}

	if contains(tokens, "war_economy") || contains(tokens, "GER_hired_advisor_idea") {
		t.Errorf("spirit row took a law or an advisor: %v", tokens)
	}
	advisors, _ := payload["advisors"].([]map[string]any)
	hired, idle := 0, 0
	for _, advisor := range advisors {
		if advisor["hired"] == true {
			hired++
		} else {
			idle++
		}
	}
	if hired != 1 || idle != 1 {
		t.Errorf("hired/idle = %d/%d, want the one in office marked", hired, idle)
	}
}

const sharedCharacterPool = `
characters = {
	GXC_li_zongren = {
		name = GXC_li_zongren
		portraits = { civilian = { large = "GFX_portrait_GXC_li_zongren" } }
		country_leader = { ideology = despotism }
	}
	ENG_stanley_baldwin = {
		name = ENG_stanley_baldwin
		portraits = { civilian = { large = "GFX_portrait_ENG_baldwin" } }
		country_leader = { ideology = liberalism }
	}
	ENG_neville_chamberlain = {
		name = ENG_neville_chamberlain
		portraits = { civilian = { large = "GFX_portrait_ENG_chamberlain" } }
		country_leader = { ideology = liberalism }
	}
	CHI_understudy = {
		name = CHI_understudy
		portraits = { civilian = { large = "GFX_portrait_CHI_understudy" } }
		country_leader = { ideology = despotism }
	}
}
`

const engHistory = `
capital = 126
set_politics = { ruling_party = democratic elections_allowed = yes }
recruit_character = ENG_stanley_baldwin
1939.1.1 = {
	recruit_character = ENG_neville_chamberlain
	promote_character = ENG_neville_chamberlain
}
`

const gxcHistory = `
capital = 595
set_politics = { ruling_party = neutrality elections_allowed = no }
recruit_character = GXC_li_zongren
`

const chiBranchHistory = `
capital = 608
set_politics = { ruling_party = neutrality elections_allowed = no }
IF = {
	limit = { has_dlc = "No Compromise, No Surrender" }
	recruit_character = CHI_chiang
	else = { recruit_character = CHI_understudy }
}
`

func poolIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{

		"common/characters/Warlords_China.txt": []byte(sharedCharacterPool),
		"common/ideologies/00_ideologies.txt":  []byte(ideologyFile),
		"history/countries/ENG - Britain.txt":  []byte(engHistory),
		"history/countries/GXC - Guangxi.txt":  []byte(gxcHistory),
		"history/countries/CHI - China.txt":    []byte(chiBranchHistory),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestCharactersComeFromRecruitsNotTheFileName(t *testing.T) {
	idx := poolIndex(t)

	people, ok := idx.CharactersPayloadAt("GXC", "", "1936.1.1")
	if !ok {
		t.Fatal("GXC has no characters; the shared pool was dropped")
	}
	leader, _ := people["leader"].(map[string]any)
	if leader["token"] != "GXC_li_zongren" {
		t.Errorf("GXC leader = %v", leader)
	}
}

func TestPromotedLeaderWinsAtItsDate(t *testing.T) {
	idx := poolIndex(t)
	at1936, _ := idx.CharactersPayloadAt("ENG", "", "1936.1.1")
	leader36, _ := at1936["leader"].(map[string]any)
	if leader36["token"] != "ENG_stanley_baldwin" {
		t.Errorf("1936 leader = %v, want Baldwin", leader36)
	}
	at1939, _ := idx.CharactersPayloadAt("ENG", "", "1939.8.14")
	leader39, _ := at1939["leader"].(map[string]any)

	if leader39["token"] != "ENG_neville_chamberlain" {
		t.Errorf("1939 leader = %v, want Chamberlain", leader39)
	}
}

func TestRecruitsIgnoreTheBranchThatDoesNotApply(t *testing.T) {
	idx := poolIndex(t)
	recruits := idx.RecruitedCharacters("CHI", "1936.1.1")
	if contains(recruits, "CHI_understudy") {
		t.Errorf("took the else branch although the DLC is loaded: %v", recruits)
	}
	if !contains(recruits, "CHI_chiang") {
		t.Errorf("missed the branch that applies: %v", recruits)
	}
}

const lawIdeas = `
ideas = {
	mobilization_laws = {
		volunteer_only = { picture = volunteer_only_pic default = yes }
		limited_conscription = { picture = limited_conscription_pic }
	}
	trade_laws = {
		export_focus = { picture = export_focus_pic default = yes }
	}
	industrial_concern = {
		ig_faben = { picture = ig_faben_pic }
		krupp = { picture = krupp_pic }
	}
}
`

const lawTags = `
idea_categories = {
	government = {
		slot = mobilization_laws
		slot = trade_laws
		character_slot = political_advisor
		ledger = civilian
	}
	research_production = {
		slot = industrial_concern
		slot = industrial_concern
		ledger = civilian
	}
}
`

const lawHistory = `
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
add_ideas = { limited_conscription ig_faben }
recruit_character = LAW_someone
`

const lawCharacters = `
characters = {
	LAW_someone = {
		name = LAW_someone
		portraits = { civilian = { small = "GFX_portrait_LAW_someone" } }
		country_leader = { ideology = nazism }
	}
}
`

func lawIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/ideas/00_laws.txt":            []byte(lawIdeas),
		"common/idea_tags/00_idea.txt":        []byte(lawTags),
		"common/ideologies/00_ideologies.txt": []byte(ideologyFile),
		"common/characters/LAW.txt":           []byte(lawCharacters),
		"history/countries/LAW - Lawland.txt": []byte(lawHistory),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestSlotIdeasCarryLawsAndTheirDefaults(t *testing.T) {
	payload, ok := lawIndex(t).CharactersPayloadAt("LAW", "", "1936.1.1")
	if !ok {
		t.Fatal("no payload")
	}
	slots, _ := payload["slot_ideas"].(map[string][]map[string]any)
	if len(slots["mobilization_laws"]) != 1 || slots["mobilization_laws"][0]["token"] != "limited_conscription" {
		t.Errorf("conscription slot = %v, want the country's own law", slots["mobilization_laws"])
	}

	if len(slots["trade_laws"]) != 1 || slots["trade_laws"][0]["token"] != "export_focus" {
		t.Errorf("trade slot = %v, want the default", slots["trade_laws"])
	}
	if len(slots["industrial_concern"]) != 1 || slots["industrial_concern"][0]["token"] != "ig_faben" {
		t.Errorf("concern slot = %v", slots["industrial_concern"])
	}

	spirits, _ := payload["spirits"].([]map[string]any)
	if len(spirits) != 0 {
		t.Errorf("spirits = %v, want the laws kept off that row", spirits)
	}
}

func TestStartingIdeasReplaceTheLawInTheirSlot(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/idea_tags/00_idea.txt": []byte(`idea_categories = {
			government = { slot = mobilization_laws slot = economy character_slot = political_advisor cost = 150 }
		}`),
		"common/ideas/laws.txt": []byte(`ideas = {
			mobilization_laws = { limited = { } extensive = { } }
			economy = { partial = { } war = { } }
			country = { spirit = { } }
		}`),
		"history/countries/TAG - Tag.txt": []byte(`capital = 1
			1939.1.1 = { add_ideas = { extensive war } }
			add_ideas = { spirit limited partial }
		`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(idx.StartingIdeas("TAG", "1936.1.1"), ","); got != "spirit,limited,partial" {
		t.Errorf("1936 = %s", got)
	}
	if got := strings.Join(idx.StartingIdeas("TAG", "1939.8.14"), ","); got != "spirit,extensive,war" {
		t.Errorf("1939 = %s, want the 1939 laws in place of the 1936 ones", got)
	}
}
