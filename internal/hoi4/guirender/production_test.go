package guirender

import "testing"

const testProductionEquipment = `
equipments = {
	infantry_equipment = {
		is_archetype = yes
		picture = archetype_infantry_equipment
		type = infantry
		interface_category = interface_category_land
		build_cost_ic = 0.43
		resources = { steel = 2 }
	}
	infantry_equipment_1 = {
		archetype = infantry_equipment
		year = 1936
		build_cost_ic = 0.5
	}
	ship_hull_heavy = {
		is_archetype = yes
		type = capital_ship
		interface_category = interface_category_capital_ships
		build_cost_ic = 3000
		resources = { steel = 3 chromium = 1 }
	}
	ship_hull_heavy_2 = { archetype = ship_hull_heavy  year = 1936 }
	ship_hull_cruiser = {
		is_archetype = yes
		type = cruiser
		interface_category = interface_category_screen_ships
		build_cost_ic = 2800
		resources = { steel = 3 }
	}
	ship_hull_cruiser_panzerschiff = { archetype = ship_hull_cruiser  year = 1929 }
}
`

const testProductionTech = `
technologies = {
	infantry_weapons1 = { enable_equipments = { infantry_equipment_1 } }
	basic_ship_hull_heavy = { enable_equipments = { ship_hull_heavy_2 } }
	panzerschiffe = { enable_equipments = { ship_hull_cruiser_panzerschiff } }
}
`

const testProductionHistory = `
capital = 1
set_technology = { infantry_weapons1 = 1 basic_ship_hull_heavy = 1 panzerschiffe = 1 }
`

const testProductionSprites = `
spriteTypes = {
	spriteType = {
		name = "GFX_archetype_infantry_equipment_medium"
		texturefile = "gfx/interface/archetypes/archetype_infantry_equipment.dds"
	}
	spriteType = {
		name = "GFX_basic_ship_hull_heavy_medium"
		texturefile = "gfx/interface/ships/basic_ship_hull_heavy.dds"
	}
	spriteType = {
		name = "GFX_ship_hull_cruiser_panzerschiff_medium"
		texturefile = "gfx/interface/ships/panzerschiff.dds"
	}
}
`

const testResources = `
resources = {
	oil = { icon_frame = 1 }
	steel = { icon_frame = 5 }
	chromium = { icon_frame = 6 }
}
`

func productionIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/units/equipment/all.txt":      []byte(testProductionEquipment),
		"common/technologies/all.txt":         []byte(testProductionTech),
		"common/resources/00_resources.txt":   []byte(testResources),
		"history/countries/TAG - Tagland.txt": []byte(testProductionHistory),
		"interface/equipment.gfx":             []byte(testProductionSprites),
		"localisation/english/equip.yml": []byte(
			"l_english:\n infantry_equipment_1: \"Gewehr 98\"\n" +
				" ship_hull_heavy_2: \"Basic Heavy Ship\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestProductionGroupsByTheScreensOwnCategory(t *testing.T) {
	payload := productionIndex(t).ProductionPayload("TAG", "1936.1.1")
	items, _ := payload["equipment"].([]map[string]any)
	if len(items) != 3 {
		t.Fatalf("items = %d, want the three models the country has researched", len(items))
	}

	want := []struct{ name, group string }{
		{"infantry_equipment_1", "infantry"},
		{"ship_hull_heavy_2", "capital_ships"},
		{"ship_hull_cruiser_panzerschiff", "screen_ships"},
	}
	for at, expect := range want {
		if items[at]["name"] != expect.name || items[at]["group"] != expect.group {
			t.Errorf("item %d = %v/%v, want %v/%v",
				at, items[at]["name"], items[at]["group"], expect.name, expect.group)
		}
	}

	for _, item := range items {
		if item["name"] == "infantry_equipment" {
			t.Errorf("the archetype itself is listed: %v", item)
		}
	}
}

func TestProductionIconFallsBackFromPictureToTechName(t *testing.T) {
	payload := productionIndex(t).ProductionPayload("TAG", "1936.1.1")
	items, _ := payload["equipment"].([]map[string]any)
	byName := map[string]map[string]any{}
	for _, item := range items {
		byName[item["name"].(string)] = item
	}

	if got := byName["infantry_equipment_1"]["icon"]; got != "GFX_archetype_infantry_equipment_medium" {
		t.Errorf("land icon = %v, want the one its own `picture` names", got)
	}

	if got := byName["ship_hull_heavy_2"]["icon"]; got != "GFX_basic_ship_hull_heavy_medium" {
		t.Errorf("hull icon = %v, want the enabling technology's sprite", got)
	}

	if got := byName["ship_hull_cruiser_panzerschiff"]["icon"]; got != "GFX_ship_hull_cruiser_panzerschiff_medium" {
		t.Errorf("named hull icon = %v, want the model's own sprite", got)
	}

	if got := productionIndex(t).equipmentIcon("TAG", "nope", "nope", "", ""); got != "" {
		t.Errorf("unknown model icon = %q, want empty", got)
	}
}

func TestProductionCarriesTheResourceStripFrames(t *testing.T) {
	payload := productionIndex(t).ProductionPayload("TAG", "1936.1.1")
	frames, _ := payload["resource_frames"].(map[string]int)
	if frames["steel"] != 5 || frames["chromium"] != 6 || frames["oil"] != 1 {
		t.Fatalf("frames = %v, want the numbers common/resources gives", frames)
	}
	items, _ := payload["equipment"].([]map[string]any)
	for _, item := range items {
		if item["name"] != "ship_hull_heavy_2" {
			continue
		}

		resources, _ := item["resources"].(map[string]float64)
		if resources["steel"] != 3 || resources["chromium"] != 1 {
			t.Errorf("hull resources = %v", resources)
		}
	}
}
