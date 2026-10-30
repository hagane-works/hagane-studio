package guirender

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const constructionBuildings = `
buildings = {
	infrastructure = {
		base_cost = 6000
		icon_frame = 3
		level_cap = {
			state_max = 5
		}
	}

	arms_factory = {
		base_cost = 7200
		base_cost_conversion = 4000
		icon_frame = 2
		level_cap = {
			shares_slots = yes
			state_max = 20
		}
	}

	naval_base = {
		base_cost = 5000
		per_level_extra_cost = 1000
		icon_frame = 6
		spawn_point = naval_base_spawn
		tags = { naval_buildings_tag }
		level_cap = {
			province_max = 10
		}
	}

	naval_facility = {
		base_cost = 10000
		icon_frame = 18
		specialization = { specialization_naval }
		tags = { facility }
		level_cap = {
			province_max = 1
			state_max = 1
			group_by = special_project_facility
		}
	}

	fuel_silo = {
		base_cost = 5000
		icon_frame = 14
		level_cap = {
			shares_slots = yes
		}
	}

	dam = {
		base_cost = 20000
		icon_frame = 26
		is_buildable = no
		spawn_point = dam_spawn
		level_cap = {
			province_max = 1
		}
	}
}

spawn_points = {
	naval_base_spawn = {
		type = province
		max = 1
	}
	dam_spawn = {
		type = province
		max = 1
	}
}
`

const constructionLandmarks = `
@landmark_repair_speed_factor = 0.15

buildings = {
	landmark_big_ben = {
		base_cost = 20000
		icon_frame = 22
		is_buildable = no
		repair_speed_factor = @landmark_repair_speed_factor
		level_cap = {
			province_max = 1
		}
	}
}
`

const constructionBrandenburg = `
state = {
	id = 64
	manpower = 7265029
	provinces = { 6521 }
	history = {
		owner = GER
		buildings = {
			infrastructure = 4
			arms_factory = 5
			6521 = {
				naval_base = 2
			}
		}
		1939.1.1 = {
			buildings = {
				arms_factory = 10
			}
		}
	}
}
`

const constructionDanzig = `
state = {
	id = 65
	manpower = 1000000
	provinces = { 100 }
	history = {
		owner = POL
		buildings = { infrastructure = 2 }
		1939.1.1 = { owner = GER }
	}
}
`

func constructionIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/buildings/00_buildings.txt":            []byte(constructionBuildings),
		"common/buildings/01_landmark_buildings.txt":   []byte(constructionLandmarks),
		"history/states/64-Brandenburg.txt":            []byte(constructionBrandenburg),
		"history/states/65-Danzig.txt":                 []byte(constructionDanzig),
		"localisation/english/buildings_l_english.yml": []byte("l_english:\n infrastructure: \"Infrastructure\"\n arms_factory: \"Military Factory\"\n naval_base: \"Naval Base\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func buildingsByName(t *testing.T, payload map[string]any) map[string]map[string]any {
	t.Helper()
	items, _ := payload["buildings"].([]map[string]any)
	out := map[string]map[string]any{}
	for _, item := range items {
		name, _ := item["name"].(string)
		out[name] = item
	}
	return out
}

func TestConstructionListsBuildingsAndNotSpawnPoints(t *testing.T) {
	payload := constructionIndex(t).BuildingsPayload("GER", "1936.1.1")
	if payload["tag"] != "GER" {
		t.Errorf("tag = %v", payload["tag"])
	}
	rows := buildingsByName(t, payload)
	for _, wanted := range []string{"infrastructure", "arms_factory", "naval_base", "naval_facility", "fuel_silo"} {
		if rows[wanted] == nil {
			t.Errorf("%s missing from the construction list", wanted)
		}
	}

	for _, unwanted := range []string{"naval_base_spawn", "dam_spawn"} {
		if rows[unwanted] != nil {
			t.Errorf("%s is a spawn point, not a building", unwanted)
		}
	}
}

func TestUnbuildableBuildingsStayOffTheConstructionScreen(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("GER", "1936.1.1"))

	if rows["dam"] != nil || rows["landmark_big_ben"] != nil {
		t.Errorf("an unbuildable building reached the screen: %v %v", rows["dam"], rows["landmark_big_ben"])
	}
	if len(rows) != 5 {
		t.Errorf("listed %d buildings, want the five buildable ones", len(rows))
	}
}

func TestEveryBuildingLandsInOneOfTheFourGrids(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("GER", "1936.1.1"))
	grids := map[string][]string{}
	for name, row := range rows {
		group, ok := row["group"].(string)
		if !ok {
			t.Fatalf("%s has no group: %v", name, row)
		}
		grids[group] = append(grids[group], name)
	}
	for group, want := range map[string]string{

		"state": "infrastructure",

		"shared": "arms_factory fuel_silo",

		"province": "naval_base",

		"facility": "naval_facility",
	} {
		sort.Strings(grids[group])
		if strings.Join(grids[group], " ") != want {
			t.Errorf("the %s grid holds %v, want %q", group, grids[group], want)
		}
	}
	if len(grids) != 4 {
		t.Errorf("buildings were sorted into %d grids: %v", len(grids), grids)
	}
}

func TestFacilitiesAreProvincialButGridThemselves(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("GER", "1936.1.1"))
	if rows["naval_facility"]["is_province"] != true || rows["naval_facility"]["group"] != "facility" {
		t.Errorf("naval_facility = %v, want a provincial building in the facility grid", rows["naval_facility"])
	}

	if rows["naval_facility"]["max_level"] != 1 {
		t.Errorf("naval_facility max_level = %v", rows["naval_facility"]["max_level"])
	}

	if rows["naval_base"]["group"] != "province" {
		t.Errorf("an unrelated tag moved naval_base to the %v grid", rows["naval_base"]["group"])
	}
}

func TestProvinceBuildingsAreTheOnesWithAProvinceCap(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("GER", "1936.1.1"))

	if rows["naval_base"]["is_province"] != true || rows["naval_base"]["max_level"] != 10 {
		t.Errorf("naval_base = %v, want the provincial cap", rows["naval_base"])
	}
	if rows["infrastructure"]["is_province"] != false || rows["infrastructure"]["max_level"] != 5 {
		t.Errorf("infrastructure = %v, want the state cap", rows["infrastructure"])
	}

	if rows["fuel_silo"]["is_province"] != false || rows["fuel_silo"]["max_level"] != 0 {
		t.Errorf("fuel_silo = %v, want an uncapped state building", rows["fuel_silo"])
	}
}

func TestBuildingIconsAreFramesOfOneStrip(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("GER", "1936.1.1"))
	if len(rows) == 0 {
		t.Fatal("no buildings to check")
	}

	for name, row := range rows {
		icon, ok := row["icon"].(string)
		if !ok || icon != buildingIconSprite {
			t.Errorf("%s draws from %v, want the shared strip on every row", name, row["icon"])
		}
	}

	if rows["infrastructure"]["icon_frame"] != 3 || rows["arms_factory"]["icon_frame"] != 2 {
		t.Errorf("icon frames = %v / %v", rows["infrastructure"]["icon_frame"], rows["arms_factory"]["icon_frame"])
	}
	if rows["infrastructure"]["base_cost"] != 6000 || rows["arms_factory"]["base_cost"] != 7200 {
		t.Errorf("base costs = %v / %v", rows["infrastructure"]["base_cost"], rows["arms_factory"]["base_cost"])
	}
	if rows["arms_factory"]["label"] != "Military Factory" {
		t.Errorf("label = %v, want the localised name", rows["arms_factory"]["label"])
	}

	if _, ok := rows["fuel_silo"]["label"]; ok {
		t.Errorf("fuel_silo invented a label: %v", rows["fuel_silo"]["label"])
	}
}

func TestBuiltCountsWhatTheCountryOwnsAtThatDate(t *testing.T) {
	idx := constructionIndex(t)

	at1936 := buildingsByName(t, idx.BuildingsPayload("GER", "1936.1.1"))

	if at1936["infrastructure"]["built"] != 4 || at1936["arms_factory"]["built"] != 5 {
		t.Errorf("1936 built = %v / %v", at1936["infrastructure"]["built"], at1936["arms_factory"]["built"])
	}

	at1939 := buildingsByName(t, idx.BuildingsPayload("GER", "1939.8.14"))

	if at1939["arms_factory"]["built"] != 10 {
		t.Errorf("1939 arms_factory = %v, want the later level alone", at1939["arms_factory"]["built"])
	}
	if at1939["infrastructure"]["built"] != 6 {
		t.Errorf("1939 infrastructure = %v, want Brandenburg plus Danzig", at1939["infrastructure"]["built"])
	}

	polish := buildingsByName(t, idx.BuildingsPayload("POL", "1936.1.1"))
	if polish["infrastructure"]["built"] != 2 {
		t.Errorf("POL 1936 infrastructure = %v", polish["infrastructure"]["built"])
	}
}

func TestBuiltAgreesWithTheTopbar(t *testing.T) {
	idx := constructionIndex(t)
	for _, date := range []string{"1936.1.1", "1939.8.14"} {
		rows := buildingsByName(t, idx.BuildingsPayload("GER", date))
		stats := idx.CountryStatsPayload("GER", date)
		if rows["arms_factory"]["built"] != stats["military_factories"] {
			t.Errorf("%s: construction says %v military factories, topbar says %v",
				date, rows["arms_factory"]["built"], stats["military_factories"])
		}
	}
}

func TestProvinceBuildingsAreCounted(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("GER", "1936.1.1"))
	if rows["naval_base"]["built"] != 2 {
		t.Errorf("naval_base built = %v, want the 2 the province sub-block records",
			rows["naval_base"]["built"])
	}
}

func TestAMissingCountryIsNotAnError(t *testing.T) {
	rows := buildingsByName(t, constructionIndex(t).BuildingsPayload("ZZZ", "1936.1.1"))

	if len(rows) != 5 || rows["infrastructure"]["built"] != 0 {
		t.Errorf("unknown tag = %v", rows)
	}
}

func TestRealGameConstructionCatalogue(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("HOI4_GAME_ROOT"))
	if root == "" {
		t.Skip("HOI4_GAME_ROOT unset; set it to a Hearts of Iron IV install to run the real-corpus gate")
	}
	files := map[string][]byte{}
	load := func(sub string, extension string) {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(sub)))
		if err != nil {
			t.Fatalf("read %s: %v", sub, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), extension) {
				continue
			}
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sub), entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[sub+"/"+entry.Name()] = body
		}
	}
	load("common/buildings", ".txt")
	load("history/states", ".txt")
	load("localisation/english", ".yml")
	load("interface", ".gfx")

	idx, err := NewIndex(files)
	if err != nil {
		t.Fatal(err)
	}
	payload := idx.BuildingsPayload("GER", "1936.1.1")
	items, _ := payload["buildings"].([]map[string]any)
	if len(items) < 20 {
		t.Fatalf("the real catalogue came back with %d buildings", len(items))
	}

	strip, ok := idx.spriteRecordFor(buildingIconSprite)
	if !ok {
		t.Fatalf("%s is not declared anywhere in interface/*.gfx", buildingIconSprite)
	}
	filled := map[string]int{}
	for _, item := range items {
		frame, _ := item["icon_frame"].(int)
		if frame < 1 || frame > strip.Frames {
			t.Errorf("%v has icon_frame %d, outside the strip's %d frames", item["name"], frame, strip.Frames)
		}
		if item["icon"] != buildingIconSprite {
			t.Errorf("%v draws from %v", item["name"], item["icon"])
		}
		group, _ := item["group"].(string)
		switch group {
		case "state", "shared", "province", "facility":
			filled[group]++
		default:
			t.Errorf("%v landed in grid %q", item["name"], group)
		}
	}

	for _, group := range []string{"state", "shared", "province", "facility"} {
		if filled[group] == 0 {
			t.Errorf("the %s grid came back empty against the real files", group)
		}
	}
	t.Logf("real catalogue: %d buildings across %v", len(items), filled)

	stats := idx.CountryStatsPayload("GER", "1936.1.1")
	for _, item := range items {
		if item["name"] == "industrial_complex" && item["built"] != stats["civilian_factories"] {
			t.Errorf("construction says %v civilian factories, topbar says %v", item["built"], stats["civilian_factories"])
		}
	}
}

func TestCatalogueHidesWhatTheCountryCannotBuildYet(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\nset_technology = { fuel_silos = 1 }\n"),
		"common/technologies/tech.txt": []byte("technologies = {\n" +
			" fuel_silos = { research_cost = 1 enable_building = { building = fuel_silo level = 1 } }\n" +
			" nuclear_reactor = { research_cost = 1 enable_building = { building = nuclear_reactor level = 1 } }\n}\n"),
		"common/buildings/00_buildings.txt": []byte("buildings = {\n" +
			" fuel_silo = { icon_frame = 14 hide_if_missing_tech = yes level_cap = { shares_slots = yes } }\n" +
			" nuclear_reactor = { icon_frame = 13 hide_if_missing_tech = yes level_cap = { shares_slots = yes } }\n" +
			" land_facility = { icon_frame = 19 only_display_if_exists = yes tags = { facility } level_cap = { province_max = 1 } }\n" +
			" air_facility = { icon_frame = 17 only_display_if_exists = yes tags = { facility } level_cap = { province_max = 1 } }\n}\n"),
		"history/states/1-Home.txt": []byte("state = { id = 1 provinces = { 10 } history = { owner = TAG buildings = { 10 = { land_facility = 1 } } } }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	rows := buildingsByName(t, idx.BuildingsPayload("TAG", "1936.1.1"))
	for _, name := range []string{"fuel_silo", "land_facility", "air_facility"} {
		if rows[name] == nil {
			t.Errorf("%s left out", name)
		}
	}
	for _, name := range []string{"nuclear_reactor"} {
		if rows[name] != nil {
			t.Errorf("%s listed: %v", name, rows[name])
		}
	}
}
