package guirender

import "testing"

const testStateFile = `
state = {
	id = 64
	name = "STATE_64"
	manpower = 3000000
	state_category = megalopolis
	resources = { steel = 12  aluminium = 4 }

	history = {
		owner = TAG
		add_core_of = TAG
		victory_points = { 11384 50 }
		victory_points = { 11385 5 }
		buildings = {
			infrastructure = 4
			arms_factory = 2
			11384 = { naval_base = 3 }
		}
		1939.1.1 = {
			buildings = { arms_factory = 5 }
		}
	}

	provinces = { 11384 11385 3499 }
}
`

const testStateBuildings = `
buildings = {
	infrastructure = { base_cost = 3000  icon_frame = 3  level_cap = { state_max = 5 } }
	arms_factory = {
		base_cost = 7200  icon_frame = 2
		level_cap = { shares_slots = yes }
	}
	naval_base = {
		base_cost = 6000  icon_frame = 4
		level_cap = { province_max = 10 }
	}
}
`

func stateIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/states/64-Brandenburg.txt":   []byte(testStateFile),
		"common/buildings/00_buildings.txt":   []byte(testStateBuildings),
		"common/resources/00_resources.txt":   []byte(testResources),
		"history/countries/TAG - Tagland.txt": []byte("capital = 64\n"),
		"localisation/english/states.yml":     []byte("l_english:\n STATE_64: \"Brandenburg\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestStateOpensOnTheCapitalAndReadsItsOwnFile(t *testing.T) {
	payload := stateIndex(t).StatesOfPayload("TAG", "1936.1.1", 0)
	detail, _ := payload["state"].(map[string]any)
	if detail == nil {
		t.Fatal("no state")
	}
	if detail["name"] != "Brandenburg" || detail["id"] != 64 {
		t.Errorf("state = %v/%v", detail["name"], detail["id"])
	}

	if detail["victory_points"] != 55 {
		t.Errorf("victory points = %v, want 50+5", detail["victory_points"])
	}
	if cores, _ := detail["cores"].([]string); len(cores) != 1 || cores[0] != "TAG" {
		t.Errorf("cores = %v", detail["cores"])
	}
	if detail["provinces"] != 3 {
		t.Errorf("provinces = %v", detail["provinces"])
	}
}

func TestStateBuildingsSplitByTheirOwnLevelCap(t *testing.T) {
	payload := stateIndex(t).StatesOfPayload("TAG", "1936.1.1", 0)
	detail, _ := payload["state"].(map[string]any)
	shelf := func(key string) map[string]int {
		out := map[string]int{}
		items, _ := detail[key].([]map[string]any)
		for _, item := range items {
			out[item["building"].(string)] = item["level"].(int)
		}
		return out
	}

	if shelf("buildings")["infrastructure"] != 4 {
		t.Errorf("state shelf = %v", shelf("buildings"))
	}
	if shelf("shared")["arms_factory"] != 2 {
		t.Errorf("shared shelf = %v", shelf("shared"))
	}
	province, _ := detail["province_buildings"].([]map[string]any)
	if len(province) != 1 || province[0]["province"] != 11384 || province[0]["level"] != 3 {
		t.Errorf("province shelf = %v", province)
	}
}

func TestALaterBlockReplacesTheLevelRatherThanAddingToIt(t *testing.T) {

	payload := stateIndex(t).StatesOfPayload("TAG", "1939.1.1", 64)
	detail, _ := payload["state"].(map[string]any)
	items, _ := detail["shared"].([]map[string]any)
	if len(items) != 1 || items[0]["level"] != 5 {
		t.Errorf("1939 shared shelf = %v, want the level the later block sets", items)
	}
}
