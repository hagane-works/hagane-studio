package mapgen

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWorldOfPartsKeepsTheModsStatesAndRegions(t *testing.T) {
	b := lakeStrip()
	states := []WorldPartState{{ID: 2, Provinces: []int{3, 4, 5}, Capital: 5}, {ID: 1, Provinces: []int{1, 2}}}
	regions := []Region{{ID: 7, Provinces: []int{3, 4, 5}, Weather: "weather={ }"}, {ID: 3, Provinces: []int{1, 2}}}
	world, err := WorldOfParts(b, states, regions)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.All.States) != 2 || world.All.States[0].Capital != 1 || world.All.States[1].Capital != 5 {
		t.Fatalf("states %+v", world.All.States)
	}
	if got := world.All.States[1].Provinces; len(got) != 2 || got[0] != 4 || got[1] != 5 {
		t.Fatalf("state 2 should keep its land and leave lake 3 to its region: %v", got)
	}
	if world.Regions[0].ID != 3 || world.Regions[1].ID != 7 || world.Regions[1].Weather != "weather={ }" {
		t.Fatalf("regions %+v", world.Regions)
	}
	if got := world.LoadRules(); len(got) != 0 {
		t.Fatalf("load rules: %+v", got)
	}
	files := world.WorldFiles(WorldMod{LandWeather: "weather={ land }", Names: map[string]map[string]string{
		"english": {"STRATEGICREGION_7": "Lake Shore", "VICTORY_POINTS_5": "Port Town"},
	}, Languages: []string{"english", "simp_chinese"}})
	if !strings.Contains(files["map/strategicregions/7-Region 7.txt"], "weather={ }") ||
		!strings.Contains(files["map/strategicregions/3-Region 3.txt"], "weather={ land }") {
		t.Fatalf("region weather: %q / %q", files["map/strategicregions/7-Region 7.txt"], files["map/strategicregions/3-Region 3.txt"])
	}
	for _, lang := range []string{"english", "simp_chinese"} {
		yml := files["localisation/"+lang+"/new_world_l_"+lang+".yml"]
		if !strings.Contains(yml, `STRATEGICREGION_7:0 "Lake Shore"`) || !strings.Contains(yml, `VICTORY_POINTS_5:0 "Port Town"`) {
			t.Fatalf("%s names:\n%s", lang, yml)
		}
	}
}

func TestWorldOfPartsRefusesWhatTheGameQuitsOn(t *testing.T) {
	b := lakeStrip()
	regions := []Region{{ID: 1, Provinces: []int{1, 2, 3, 4, 5}}}
	for name, c := range map[string]struct {
		states []WorldPartState
		code   string
		values map[string]any
	}{
		"gap":       {[]WorldPartState{{ID: 1, Provinces: []int{1, 2}}, {ID: 3, Provinces: []int{4, 5}}}, "mod_state_gap", map[string]any{"count": 2, "missing": 2}},
		"lake only": {[]WorldPartState{{ID: 1, Provinces: []int{1, 2, 4, 5}}, {ID: 2, Provinces: []int{3}}}, "mod_state_no_land", map[string]any{"states": "#2"}},
		"capital":   {[]WorldPartState{{ID: 1, Provinces: []int{1, 2}, Capital: 4}, {ID: 2, Provinces: []int{4, 5}}}, "mod_state_capital", map[string]any{"states": "#1"}},
	} {
		world, err := WorldOfParts(b, c.states, regions)
		refused, ok := err.(*AddError)
		if world != nil || !ok || refused.Code != c.code || !reflect.DeepEqual(refused.Values, c.values) || refused.Message == "" {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestWorldOfPartsOnTheBaseGame(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	read := func(name string) []byte {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(root, "map", name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	b, err := LoadBase(read("provinces.bmp"), read("heightmap.bmp"), read("terrain.bmp"), read("rivers.bmp"), read("definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	var states []WorldPartState
	for id, provinces := range statesIn(t, filepath.Join(root, "history", "states")) {
		states = append(states, WorldPartState{ID: id, Provinces: provinces})
	}
	var regions []Region
	for id, provinces := range regionsIn(t, filepath.Join(root, "map", "strategicregions")) {
		regions = append(regions, Region{ID: id, Provinces: provinces})
	}
	world, err := WorldOfParts(b, states, regions)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.All.States) != len(states) || len(world.Regions) != len(regions) {
		t.Fatalf("%d states, %d regions", len(world.All.States), len(world.Regions))
	}
	if got := world.LoadRules(); len(got) != 0 {
		t.Fatalf("the base game as parts: %+v", got)
	}
}
