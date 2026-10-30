package guirender

import (
	"encoding/json"
	"testing"
)

const modBrandenburg = `
state = {
	id = 64
	manpower = 9000000
	provinces = { 6521 }
	resources = { steel = 7 }
	history = {
		owner = GER
		buildings = { arms_factory = 2 industrial_complex = 3 dockyard = 1 synthetic_refinery = 2 }
	}
}
`

const topbarDefines = `NDefines = {
NCountry = {
	DEFAULT_STABILITY = 0.5,
	DEFAULT_WAR_SUPPORT = 0.5,
	STARTING_FUEL_RATIO = 0.25,
	BASE_FUEL_GAIN_PER_OIL = 2,
	BASE_FUEL_GAIN = 2.0,
	BASE_FUEL_CAPACITY = 50000,
},
NPolitics = {
	BASE_POLITICAL_POWER_INCREASE = 2,
},
}
`

func layeredStates(t *testing.T, top map[string][]byte) *Index {
	t.Helper()
	base, err := NewIndex(map[string][]byte{
		"common/ideas/00_laws.txt":            []byte(statLaws),
		"common/idea_tags/00_idea.txt":        []byte(lawTags),
		"history/states/64-Brandenburg.txt":   []byte(brandenburg),
		"history/states/65-Danzig.txt":        []byte(danzig),
		"history/countries/GER - Germany.txt": []byte(statHistory),
		"common/defines/00_defines.lua":       []byte(topbarDefines),
	})
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(top, base)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestAStateCountsOnceAcrossLayers(t *testing.T) {

	same := layeredStates(t, map[string][]byte{
		"history/states/64-Brandenburg.txt": []byte(brandenburg),
		"history/states/65-Danzig.txt":      []byte(danzig),
	})
	stats := same.CountryStatsPayload("GER", "1936.1.1")
	if stats["civilian_factories"] != 4 || stats["military_factories"] != 5 || stats["population"] != 7000000 {
		t.Errorf("duplicated layers: %v/%v pop %v, want 4/5 and 7,000,000",
			stats["civilian_factories"], stats["military_factories"], stats["population"])
	}
	if rows, _ := same.StatesOfPayload("GER", "1936.1.1", 0)["states"].([]map[string]any); len(rows) != 1 {
		t.Errorf("state list has %d rows for one state id", len(rows))
	}
	if built := same.builtBuildings("GER", "1936.1.1"); built["arms_factory"] != 5 {
		t.Errorf("construction counts %d arms factories, want 5", built["arms_factory"])
	}

	modded := layeredStates(t, map[string][]byte{"history/states/64-Brandenburg.txt": []byte(modBrandenburg)})
	stats = modded.CountryStatsPayload("GER", "1936.1.1")
	if stats["civilian_factories"] != 3 || stats["military_factories"] != 2 || stats["population"] != 9000000 {
		t.Errorf("mod copy: %v/%v pop %v, want the mod's 3/2 and 9,000,000",
			stats["civilian_factories"], stats["military_factories"], stats["population"])
	}
	if resources := modded.resourcesByOwner("1936.1.1")["GER"]; resources["steel"] != 7 {
		t.Errorf("trade counts %v steel, want the mod's 7", resources["steel"])
	}

	if at1939 := modded.CountryStatsPayload("GER", "1939.8.14"); at1939["population"] != 10000000 {
		t.Errorf("1939 population = %v, want 9,000,000 + Danzig's 1,000,000", at1939["population"])
	}
}

const datedHistory = `
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
set_stability = 0.7
set_war_support = 0.3
set_convoys = 200
set_research_slots = 4
set_fuel_ratio = 0.8
add_command_power = 20
SLO = { set_stability = 0.1 add_political_power = 500 }
1939.1.1 = {
	set_war_support = 0.7
	add_political_power = 1000
	add_command_power = 80
	set_research_slots = 5
	army_experience = 10
	add_nuclear_bombs = 2
	complete_national_focus = GER_doctrine
	add_named_threat = { threat = 40 name = 1939_bookmark_threat }
}
`

const doctrineFocus = `
focus_tree = {
	id = german_focus
	focus = {
		id = GER_doctrine
		x = 0 y = 0 cost = 10
		completion_reward = {
			add_command_power = 25
			army_experience = 15
			add_war_support = 0.05
			IF = { limit = { has_dlc = "No Step Back" } add_command_power = 99 }
			every_owned_state = { add_political_power = 7 }
		}
	}
}
`

func datedIndex(t *testing.T, extra map[string][]byte) *Index {
	t.Helper()
	files := map[string][]byte{
		"history/countries/GER - Germany.txt": []byte(datedHistory),
		"history/countries/ITA - Italy.txt": []byte("capital = 2\n" +
			"1939.1.1 = { add_named_threat = { threat = 2 name = ITA_albania_war_goal } }\n"),
		"history/countries/HOL - Holland.txt": []byte("capital = 3\nset_politics = { ruling_party = democratic }\n"),
		"common/national_focus/germany.txt":   []byte(doctrineFocus),
		"common/defines/00_defines.lua":       []byte(topbarDefines),
	}
	for name, body := range extra {
		files[name] = body
	}
	idx, err := NewIndex(files)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestHistoryRunsToTheScenarioDate(t *testing.T) {
	idx := datedIndex(t, nil)
	at1936 := idx.CountryStatsPayload("GER", "1936.1.1")

	if at1936["command_power"] != 20.0 || at1936["political_power"] != 2.0 || at1936["research_slots"] != 4 ||
		at1936["army_experience"] != 0 || at1936["nukes"] != 0 || !near(at1936["war_support"], 0.3) {
		t.Errorf("1936 = cp %v pp %v slots %v army %v nukes %v ws %v", at1936["command_power"],
			at1936["political_power"], at1936["research_slots"], at1936["army_experience"],
			at1936["nukes"], at1936["war_support"])
	}

	if !near(at1936["stability"], 0.7) {
		t.Errorf("stability = %v: a scoped block leaked in", at1936["stability"])
	}

	at1939 := idx.CountryStatsPayload("GER", "1939.8.14.12")

	base1939, _ := idx.countryStartAt("GER", "1939.8.14.12")
	if at1939["command_power"] != 125.0 || base1939.PoliticalPower != 1000.0 || at1939["political_power"] != 1002.0 {
		t.Errorf("1939 cp/pp = %v/%v (topbar %v), want 125/1000 (1002)", at1939["command_power"], base1939.PoliticalPower, at1939["political_power"])
	}

	if at1939["research_slots"] != 5 || at1939["convoys"] != 200 {
		t.Errorf("1939 slots/convoys = %v/%v, want 5/200", at1939["research_slots"], at1939["convoys"])
	}

	if !near(at1939["war_support"], 0.75) || at1939["army_experience"] != 25 || at1939["nukes"] != 2 {
		t.Errorf("1939 ws/army/nukes = %v/%v/%v, want 0.75/25/2",
			at1939["war_support"], at1939["army_experience"], at1939["nukes"])
	}
	if slots := idx.researchSlotsAt("GER", "1939.8.14"); slots != 5 {
		t.Errorf("the tech screen reads %d slots in 1939", slots)
	}
}

func TestHistoryDefaults(t *testing.T) {
	idx := datedIndex(t, nil)
	holland := idx.CountryStatsPayload("HOL", "1936.1.1")
	if holland["stability"] != 0.5 || holland["war_support"] != 0.5 {
		t.Errorf("stability/war support = %v/%v, want DEFAULT_STABILITY/DEFAULT_WAR_SUPPORT 0.5",
			holland["stability"], holland["war_support"])
	}
	if fuel, _ := holland["fuel"].(map[string]any); fuel["ratio"] != 0.25 {
		t.Errorf("fuel ratio = %v, want STARTING_FUEL_RATIO 0.25", fuel["ratio"])
	}
	if fuel, _ := idx.CountryStatsPayload("GER", "1936.1.1")["fuel"].(map[string]any); fuel["ratio"] != 0.8 {
		t.Errorf("GER fuel ratio = %v, want its set_fuel_ratio 0.8", fuel["ratio"])
	}
}

func TestDefinesOverrideTheDefaults(t *testing.T) {
	idx := datedIndex(t, map[string][]byte{
		"common/defines/00_defines.lua": []byte("NDefines = {\nNCountry = {\n\tDEFAULT_STABILITY = 0.5,\t-- note\n" +
			"\tSTARTING_FUEL_RATIO = 0.25,\n\tGAME_SPEED = { 1, 2 },\n\tBASE_FUEL_CAPACITY = 50000,\n},\n}\n"),
		"common/defines/zz_mod.lua": []byte("NDefines.NCountry.DEFAULT_STABILITY = 0.6\n" +
			"NDefines.NCountry.STARTING_FUEL_RATIO = 0.5\n"),
	})
	holland := idx.CountryStatsPayload("HOL", "1936.1.1")
	if holland["stability"] != 0.6 {
		t.Errorf("stability = %v, want the override's 0.6", holland["stability"])
	}
	if fuel, _ := holland["fuel"].(map[string]any); fuel["ratio"] != 0.5 {
		t.Errorf("fuel ratio = %v, want the override's 0.5", fuel["ratio"])
	}
	if value := idx.defineNumber("NCountry.BASE_FUEL_CAPACITY"); value != 50000 {
		t.Errorf("a define the override leaves alone reads %v, want 00_defines.lua's 50000", value)
	}
}

func TestWorldTensionIsHistoryThreatByDate(t *testing.T) {
	idx := datedIndex(t, nil)
	if tension := idx.CountryStatsPayload("GER", "1936.1.1.12")["world_tension"]; tension != 0.0 {
		t.Errorf("1936 tension = %v", tension)
	}
	if tension := idx.CountryStatsPayload("HOL", "1939.8.14.12")["world_tension"]; tension != 42.0 {
		t.Errorf("1939 tension = %v, want 40 + 2 from every country's history", tension)
	}
}

func TestFuelEstimate(t *testing.T) {
	idx := layeredStates(t, map[string][]byte{
		"history/states/64-Brandenburg.txt": []byte(modBrandenburg),
		"common/buildings/00_buildings.txt": []byte(`buildings = {
			infrastructure = { base_cost = 6000 country_modifiers = { modifiers = { max_fuel_building = 1.5 } } }
			synthetic_refinery = { base_cost = 14500 country_modifiers = { modifiers = { fuel_gain_from_states = 2.0 } } }
		}`),
		"history/countries/GER - Germany.txt": []byte(statHistory + "set_fuel_ratio = 0.5\n"),
	})
	fuel, _ := idx.CountryStatsPayload("GER", "1936.1.1")["fuel"].(map[string]any)

	if fuel["capacity"] != 50000.0 || fuel["days"] != 174 || fuel["gaining"] != true || fuel["ratio"] != 0.5 {
		t.Errorf("fuel = %v", fuel)
	}
}

func TestCountryStatsContractFields(t *testing.T) {
	stats := statsIndex(t).CountryStatsPayload("GER", "1936.1.1")
	if stats["factories_total"] != 10 || stats["supply_ratio"] != 1.0 {
		t.Errorf("total %v supply %v", stats["factories_total"], stats["supply_ratio"])
	}
	for _, key := range []string{"fuel", "world_tension", "decisions_takeable", "decisions_takeable_count",
		"command_power", "army_experience", "navy_experience", "air_experience", "nukes", "research_slots"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("country-stats has no %q", key)
		}
	}

	if raw, err := json.Marshal(stats["faction"]); err != nil || string(raw) != "null" {
		t.Errorf("faction = %s, want an explicit null", raw)
	}
}
