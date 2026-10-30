package mapgen

import (
	"path"
	"strconv"
	"strings"
	"testing"
)

const gameTutorial = `tutorial = {
    window = "tutorial_screen_2"
    text = "tut_2_text_1"
    state = 550 #Eritrea
    highlight_states_trigger = { 550 }
    direction_pointer = 5091
    move_camera_to = {
        textbox = "obj_1"
        state = 550 #Eritrea
    }
    highlight_provinces = { 11837 4014 11735 }
    offensive_order = {
        textbox = "obj_1"
        target = 5010 #Ethiopian Capital
    }
    declare_war = {
        textbox = "obj_1"
        target = "ETH"
    }
}
`

func TestWorldTutorialPointsAtTheWorld(t *testing.T) {
	got := WorldTutorial(gameTutorial, 1, 42, "NWB", 300)
	for _, want := range []string{
		"state = 1 #Eritrea", "highlight_states_trigger = { 1 }", "direction_pointer = 42",
		"highlight_provinces = { 42 }", `target = "NWB"`, "target = 300 #Ethiopian Capital", `window = "tutorial_screen_2"`, `textbox = "obj_1"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	for _, gone := range []string{"550", "5091", "11837", "5010", "ETH"} {
		if strings.Contains(got, gone) {
			t.Errorf("%s left in\n%s", gone, got)
		}
	}
}

func TestWorldFilesStartAWorld(t *testing.T) {
	_, world := twoIslands(t)
	tags := []string{"GER", "NWB"}
	owner, capital := world.Owners(tags)
	if len(owner) != len(world.All.States) || capital["GER"] == 0 || capital["NWB"] == 0 {
		t.Fatalf("owners %v, capitals %v for %d states", owner, capital, len(world.All.States))
	}
	x := map[int]float64{}
	for _, pr := range world.All.Provinces {
		x[pr.ID] = pr.X
	}
	westmostNWB := 1e9
	for id, tag := range owner {
		if tag == "NWB" {
			westmostNWB = min(westmostNWB, x[world.All.States[id-1].Capital])
		}
	}
	for id, tag := range owner {
		if tag == "GER" && x[world.All.States[id-1].Capital] > westmostNWB {
			t.Errorf("state %d of GER lies east of NWB's states", id)
		}
		if owner[capital[tag]] != tag {
			t.Errorf("%s's capital %d is %s's", tag, capital[tag], owner[capital[tag]])
		}
	}

	files := world.WorldFiles(WorldMod{
		Tags: tags, Owner: owner, Capital: capital, Seed: 3,
		LandWeather: "weather={}", SeaWeather: "weather={}",
		Tutorial: gameTutorial, TagLines: []string{`GER = "countries/Germany.txt"`}, DynamicTags: "D01 = \"countries/D01.txt\"\n",
		Histories: []string{"NWB"}, Languages: []string{"english", "simp_chinese"},
	})
	regions := 0
	for name := range files {
		if strings.HasPrefix(name, "map/strategicregions/") {
			regions++
		}
	}
	if regions != len(world.Regions) {
		t.Errorf("%d region files for %d regions", regions, len(world.Regions))
	}
	enemyCapital := world.All.States[capital["NWB"]-1].Capital
	tutorial := files["tutorial/tutorial.txt"]
	for _, want := range []string{"state = 1 #Eritrea", `target = "NWB"`, "highlight_provinces = { " + strconv.Itoa(world.All.States[0].Capital) + " }", "target = " + strconv.Itoa(enemyCapital)} {
		if !strings.Contains(tutorial, want) {
			t.Errorf("tutorial lacks %q", want)
		}
	}
	if _, ok := files["history/countries/GER - New World.txt"]; ok {
		t.Error("GER got a start though it keeps its own")
	}
	nwbHistory := files["history/countries/NWB - New World.txt"]
	if !strings.Contains(nwbHistory, "capital = "+strconv.Itoa(capital["NWB"])) {
		t.Errorf("NWB's start: %q", nwbHistory)
	}
	if strings.Contains(nwbHistory, "create_country_leader") || strings.Contains(nwbHistory, "GFX_portrait_Europe_Generic_1") {
		t.Errorf("NWB's raw start still carries a second, legacy leader: %q", nwbHistory)
	}
	bookmark := files["common/bookmarks/new_world.txt"]
	if !strings.Contains(bookmark, `"GER" = {`) || !strings.Contains(bookmark, `"NWB" = {`) || !strings.Contains(bookmark, `default_country = "GER"`) {
		t.Errorf("bookmark: %s", bookmark)
	}
	if files["common/country_tags/00_countries.txt"] != "GER = \"countries/Germany.txt\"\n" {
		t.Errorf("tag lines: %q", files["common/country_tags/00_countries.txt"])
	}
	if strings.HasSuffix(files["map/buildings.txt"], "\n") {
		t.Error("buildings.txt ends in a newline (the game reads the empty line as a broken entry)")
	}
	english, chinese := files["localisation/english/new_world_l_english.yml"], files["localisation/simp_chinese/new_world_l_simp_chinese.yml"]
	if !strings.Contains(english, `STRATEGICREGION_1:0 "Land Region 1"`) || !strings.Contains(chinese, `STRATEGICREGION_1:0 "陆地战略区 1"`) {
		t.Errorf("region names:\n%s\n%s", english, chinese)
	}
	if strings.Contains(english, "STATE_1:") {
		t.Error("state names written though the states carry their own")
	}

	written := make([]string, 0, len(files))
	for name := range files {
		written = append(written, name)
	}
	markers := UnloadMarkers(written)
	for _, folder := range UnloadedFolders {
		has := false
		for _, name := range append(written, keys(markers)...) {
			if path.Dir(name) == folder {
				has = true
			}
		}
		if !has {
			t.Errorf("nothing unloads %s", folder)
		}
	}
	for _, folder := range []string{"history/states", "events", "common/decisions/categories"} {
		if _, ok := markers[folder+"/zz_new_world_unloaded.txt"]; !ok {
			t.Errorf("no marker in %s", folder)
		}
	}
	if _, ok := markers["map/strategicregions/zz_new_world_unloaded.txt"]; ok {
		t.Error("a marker in a folder the world writes into")
	}
	if !Unloads("common/decisions/categories/x.txt") || Unloads("common/decisions/categories/sub/x.txt") || Unloads("map/definition.csv") {
		t.Error("Unloads matches more or less than the folders themselves")
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
