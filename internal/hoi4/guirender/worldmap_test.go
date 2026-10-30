package guirender

import (
	"testing"
)

const bom = string(rune(0xFEFF))

func worldMapIndex(t *testing.T) *Index {
	t.Helper()
	game, err := NewIndex(map[string][]byte{
		"map/definition.csv": []byte("0;0;0;0;land;false;unknown;0\n" +
			"1;10;20;30;land;false;plains;1\n" +
			"2;40;50;60;land;true;forest;1\n" +
			"3;70;80;90;sea;true;ocean;0\n" +
			"4;100;110;120;lake;false;lakes;1\n"),
		"map/continent.txt":               []byte(`continents = { europe }`),
		"map/strategicregions/1-Land.txt": []byte(`strategic_region = { id = 1 name = "STRATEGICREGION_1" provinces = { 1 2 4 } }`),
		"map/strategicregions/2-Sea.txt":  []byte(`strategic_region = { id = 2 name = "STRATEGICREGION_2" provinces = { 3 } }`),
		"history/states/1-Inland.txt": []byte(`state = { id = 1 name = "STATE_1" provinces = { 1 4 }
			history = { owner = AAA 1939.1.1 = { owner = BBB } } }`),
		"history/states/2-Coast.txt":                     []byte(`state = { id = 2 name = "STATE_2" provinces = { 2 } history = { owner = BBB } }`),
		"common/country_tags/00.txt":                     []byte(`AAA = "countries/Alpha.txt"` + "\n" + `BBB = "countries/Beta.txt"`),
		"common/countries/Alpha.txt":                     []byte(`graphical_culture = western_european_gfx color = { 106  119  89 } # { 1 2 3 }`),
		"common/countries/Beta.txt":                      []byte(`color = rgb { 200 10 10 }`),
		"localisation/english/state_names_l_english.yml": []byte(bom + "l_english:\n STATE_1:0 \"Inland\"\n STATE_2:0 \"Coast\"\n STRATEGICREGION_1:0 \"Heartland\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return game
}

func TestWorldMapListsEveryProvinceWithItsStateAndRegion(t *testing.T) {
	payload := worldMapIndex(t).WorldMap("1936.1.1")
	if len(payload.Provinces) != 4 {
		t.Fatalf("provinces: %v", payload.Provinces)
	}
	want := map[int][]any{

		1: {1, uint32(10<<16 | 20<<8 | 30), 0, 0, "plains", 1, 1, 1},
		2: {2, uint32(40<<16 | 50<<8 | 60), 0, 1, "forest", 1, 2, 1},
		3: {3, uint32(70<<16 | 80<<8 | 90), 1, 1, "ocean", 0, 0, 2},
		4: {4, uint32(100<<16 | 110<<8 | 120), 2, 0, "lakes", 1, 1, 1},
	}
	for _, row := range payload.Provinces {
		id := row[0].(int)
		for i, value := range want[id] {
			if row[i] != value {
				t.Errorf("province %d field %d: got %v, want %v", id, i, row[i], value)
			}
		}
	}
}

func TestWorldMapNamesStatesAndRegionsAndKnowsOwnersAtTheDate(t *testing.T) {
	game := worldMapIndex(t)
	at1936 := game.WorldMap("1936.1.1")
	states := map[int][]any{}
	for _, row := range at1936.States {
		states[row[0].(int)] = row
	}
	if states[1][1] != "Inland" || states[1][2] != "AAA" || states[2][1] != "Coast" || states[2][2] != "BBB" {
		t.Errorf("states 1936: %v", at1936.States)
	}
	if got := game.WorldMap("1939.1.1").States[0][2]; got != "BBB" {
		t.Errorf("state 1 owner in 1939: %v", got)
	}
	if len(at1936.Regions) != 2 || at1936.Regions[0][1] != "Heartland" || at1936.Regions[1][1] != "STRATEGICREGION_2" {
		t.Errorf("regions: %v", at1936.Regions)
	}
}

func TestWorldMapColoursCountriesAsTheirFilesSay(t *testing.T) {
	colors := worldMapIndex(t).WorldMap("1936.1.1").Colors
	if colors["AAA"] != [3]uint8{106, 119, 89} || colors["BBB"] != [3]uint8{200, 10, 10} {
		t.Errorf("colors: %v", colors)
	}
}

func TestCountryColorReadsHSVAndFractions(t *testing.T) {
	root, _ := parseScript([]byte(`color = hsv { 0 1 1 }`))
	if got, ok := countryColor(root.Entries); !ok || got != [3]uint8{255, 0, 0} {
		t.Errorf("hsv red: %v %v", got, ok)
	}
	root, _ = parseScript([]byte(`color = rgb { 0.5 0.0 1.0 }`))
	if got, ok := countryColor(root.Entries); !ok || got != [3]uint8{128, 0, 255} {
		t.Errorf("fractions: %v %v", got, ok)
	}
}
