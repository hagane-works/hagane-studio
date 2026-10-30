package guirender

import (
	"math"
	"strings"
	"testing"
)

func designerValue(t *testing.T, out map[string]any, column, stat string) float64 {
	t.Helper()
	for _, row := range out["columns"].(map[string]any)[column].([]map[string]any) {
		if row["stat"] == stat {
			return row["value"].(float64)
		}
	}
	t.Fatalf("no %s in %s", stat, column)
	return 0
}

func TestRealGameTankDesignerMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.EquipmentDesignerPayload("GER", "1936.11.1.12", "medium_tank_chassis_0")

	if out["name"] != "Mittlerer Traktor" || out["battalion"] != "medium_armor" || out["blueprint"] != "equipment_designer_medium_tank_chassis_ger" {
		t.Fatalf("design = %v / %v / %v", out["name"], out["battalion"], out["blueprint"])
	}
	if out["kind"] != "tank" || out["experience_icon"] != "army_experience" {
		t.Errorf("kind = %v / %v", out["kind"], out["experience_icon"])
	}
	for _, want := range []struct {
		column, stat string
		value        float64
	}{
		{"base", "maximum_speed", 5.175},
		{"base", "reliability", 0.75},
		{"base", "supply_consumption", 0.25},
		{"combat", "hardness", 0.85},
		{"combat", "armor_value", 20},
		{"combat", "breakthrough", 11.5},
		{"combat", "defense", 2},
		{"misc", "fuel_consumption", 2},
		{"misc", "suppression", 2.5},
	} {
		if got := designerValue(t, out, want.column, want.stat); math.Abs(got-want.value) > 1e-6 {
			t.Errorf("%s = %v, want %v", want.stat, got, want.value)
		}
	}
	if cost := out["cost"].(float64); math.Abs(cost-4.6) > 1e-6 {
		t.Errorf("cost = %v, want 4.60", cost)
	}

	slots := out["slots"].([]map[string]any)
	if len(slots) != 9 || slots[0]["slot"] != "turret_type_slot" || slots[1]["module"] != "" || slots[4]["module"] != "tank_gasoline_engine" {
		t.Errorf("slots = %v", slots)
	}
	if upgrades := out["upgrades"].([]map[string]any); len(upgrades) != 2 {
		t.Errorf("upgrades = %v", upgrades)
	}

	found := false
	for _, item := range idx.ProductionPayload("GER", "1936.1.1.12")["equipment"].([]map[string]any) {
		if item["name"] == "medium_tank_chassis_0" {
			found = item["designer"] == "tank"
		}
		if item["name"] == "infantry_equipment_1" && item["designer"] != nil {
			t.Error("rifles have no designer")
		}
	}
	if !found {
		t.Error("the medium tank row does not open the tank designer")
	}

	panzer := idx.EquipmentDesignerPayload("GER", "1936.1.1.12", "Panzer I Ausf. A")
	if panzer["type"] != "light_tank_chassis_1" || panzer["battalion"] != "light_armor" {
		t.Errorf("Panzer I = %v / %v", panzer["type"], panzer["battalion"])
	}
}

func TestRealGamePlaneDesignerMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.EquipmentDesignerPayload("GER", "1936.11.1.12", "medium_plane_airframe_1")
	if out["name"] != "He 111" || out["kind"] != "plane" || out["blueprint"] != "equipment_designer_medium_plane_airframe_1_ger" {
		t.Fatalf("design = %v / %v / %v", out["name"], out["kind"], out["blueprint"])
	}
	if out["role"] != "Add main armament to give role" || out["experience_icon"] != "air_experience" {
		t.Errorf("role = %v, experience = %v", out["role"], out["experience_icon"])
	}

	if out["icon"] != "GFX_GER_tactical_bomber1_medium" || out["version"] != "B" {
		t.Errorf("icon = %v, version = %v", out["icon"], out["version"])
	}
	if texts := out["texts"].(map[string]any); texts["title"] != "Plane Designer" {
		t.Errorf("title = %v", texts["title"])
	}
	for _, want := range []struct {
		column, stat string
		value        float64
	}{
		{"base", "maximum_speed", 325},
		{"base", "air_range", 900},
		{"base", "supply_consumption", 0.28},
		{"base", "weight", 8},
		{"base", "thrust", 19},
		{"combat", "air_defence", 18},
		{"combat", "air_attack", 0},
		{"combat", "air_agility", 35.5},
		{"combat", "air_superiority", 1},
		{"combat", "naval_strike_attack", 1},
		{"combat", "naval_strike_targetting", 1},
		{"combat", "air_ground_attack", 0},
		{"combat", "air_bombing", 0},
		{"misc", "reliability", 0.8},
		{"misc", "fuel_consumption", 0.32},
		{"misc", "surface_detection", 0},
		{"misc", "night_penalty", 0},
		{"misc", "mines_sweeping", 0},
	} {
		if got := designerValue(t, out, want.column, want.stat); math.Abs(got-want.value) > 1e-6 {
			t.Errorf("%s = %v, want %v", want.stat, got, want.value)
		}
	}
	if cost := out["cost"].(float64); math.Abs(cost-32) > 1e-6 {
		t.Errorf("cost = %v, want 32.00", cost)
	}
	if resources := out["resources"].(map[string]float64); resources["aluminium"] != 3 || resources["rubber"] != 1 {
		t.Errorf("resources = %v", resources)
	}

	fixed, custom := []map[string]any{}, []map[string]any{}
	for _, slot := range out["slots"].([]map[string]any) {
		if slot["row"] == "fixed" {
			fixed = append(fixed, slot)
		} else {
			custom = append(custom, slot)
		}
	}
	if len(fixed) != 4 || fixed[0]["slot"] != "fixed_main_weapon_slot" || fixed[0]["module"] != "" || fixed[3]["index"] != 3 {
		t.Errorf("top row = %v", fixed)
	}
	if len(custom) != 5 || custom[0]["module"] != "engine_1_2x" || custom[0]["index"] != 0 {
		t.Errorf("bottom row = %v", custom)
	}
	if upgrades := out["upgrades"].([]map[string]any); len(upgrades) != 0 {
		t.Errorf("upgrades = %v", upgrades)
	}

	found := false
	costs := map[string]float64{"small_plane_airframe_1": 17, "cv_small_plane_airframe_1": 20.5, "large_plane_airframe_0": 64, "medium_plane_airframe_1": 32}
	for _, item := range idx.ProductionPayload("GER", "1936.1.1.12")["equipment"].([]map[string]any) {
		name, _ := item["name"].(string)
		if name == "medium_plane_airframe_1" {
			found = item["designer"] == "plane"
		}
		if want, ok := costs[name]; ok && math.Abs(item["cost"].(float64)-want) > 1e-6 {
			t.Errorf("%s costs %v, want %v", name, item["cost"], want)
		}
	}
	if !found {
		t.Error("the medium airframe row does not open the plane designer")
	}

	heinkel := idx.EquipmentDesignerPayload("GER", "1936.1.1.12", "He 111")
	if heinkel["type"] != "medium_plane_airframe_1" || heinkel["role"] != "Tactical Bomber" {
		t.Errorf("He 111 = %v / %v", heinkel["type"], heinkel["role"])
	}
}

func TestRealGameShipDesignerMatchesTheRecording(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.EquipmentDesignerPayload("GER", "1936.11.1.12", "ship_hull_heavy_2")
	if out["name"] != "Basic Heavy Ship" || out["kind"] != "ship" || out["version"] != "A" || out["blueprint"] != "equipment_designer_ship_hull_heavy" {
		t.Fatalf("design = %v / %v / %v / %v", out["name"], out["kind"], out["version"], out["blueprint"])
	}
	if out["experience_icon"] != "navy_experience" || out["names"] != "None" || out["cost_decimals"] != 0 {
		t.Errorf("experience = %v, names = %v, decimals = %v", out["experience_icon"], out["names"], out["cost_decimals"])
	}
	if texts := out["texts"].(map[string]any); texts["title"] != "Ship Designer" {
		t.Errorf("title = %v", texts["title"])
	}
	for _, want := range []struct {
		column, stat string
		value        float64
	}{
		{"base", "naval_speed", 32.4},
		{"base", "naval_range", 3500},
		{"base", "max_organisation", 50},
		{"base", "max_strength", 388.5},
		{"base", "reliability", 0.85},
		{"base", "supply_consumption", 0.56},
		{"base", "manpower", 4300},
		{"combat", "hg_attack", 0},
		{"combat", "armor_value", 30},
		{"combat", "anti_air_attack", 0},
		{"misc", "fuel_consumption", 88},
		{"misc", "surface_visibility", 20},
		{"misc", "surface_detection", 20},
		{"misc", "sub_visibility", 0},
		{"misc", "sub_detection", 1},
		{"misc", "naval_weather_penalty_factor", 1},
		{"misc", "hit_profile", 2000 / 36.2},
	} {
		if got := designerValue(t, out, want.column, want.stat); math.Abs(got-want.value) > 1e-6 {
			t.Errorf("%s = %v, want %v", want.stat, got, want.value)
		}
	}
	if cost := out["cost"].(float64); math.Abs(cost-3825) > 1e-6 {
		t.Errorf("cost = %v, want 3825", cost)
	}
	if resources := out["resources"].(map[string]float64); resources["steel"] != 4 || resources["chromium"] != 1 {
		t.Errorf("resources = %v", resources)
	}

	if order := out["resource_order"].([]string); len(order) != 2 || order[0] != "steel" || order[1] != "chromium" {
		t.Errorf("resource order = %v", order)
	}

	fixed, custom := map[string]map[string]any{}, 0
	for _, slot := range out["slots"].([]map[string]any) {
		if slot["row"] == "fixed" {
			fixed[slot["slot"].(string)] = slot
		} else {
			custom++
		}
	}
	if len(fixed) != 7 || custom != 4 {
		t.Errorf("fixed %d, custom %d", len(fixed), custom)
	}
	if fixed["fixed_ship_engine_slot"]["module"] != "heavy_ship_engine_2" || fixed["fixed_ship_armor_slot"]["module"] != "ship_armor_bb_1" || fixed["fixed_ship_battery_slot"]["module"] != "" {
		t.Errorf("fixed row = %v", fixed)
	}

	for name, want := range map[string]float64{
		"ship_hull_heavy_2": 589, "ship_hull_pre_dreadnought": 285, "ship_hull_cruiser_panzerschiff": 349,
		"ship_hull_cruiser_2": 198, "ship_hull_light_2": 46, "ship_hull_submarine_2": 12,
	} {
		design := idx.EquipmentDesignerPayload("GER", "1936.11.1.12", name)
		if got := math.Floor(design["dominance"].(float64)); got != want {
			t.Errorf("%s dominance = %v, want %v", name, design["dominance"], want)
		}
	}
	found := false
	for _, item := range idx.ProductionPayload("GER", "1936.1.1.12")["equipment"].([]map[string]any) {
		if item["name"] == "ship_hull_heavy_2" {
			found = item["designer"] == "ship" && math.Abs(item["cost"].(float64)-3825) < 1e-6
		}
	}
	if !found {
		t.Error("the Basic Heavy Ship row does not open the ship designer at 3825")
	}
}

func TestRealGameDesignerDraft(t *testing.T) {
	idx := realTopbarIndex(t, false)
	draft := ParseDesignDraft("fixed_main_weapon_slot:medium_bomb_bay;special_type_slot_1:empty", "", true)
	out := idx.EquipmentDesignerDraftPayload("GER", "1936.11.1.12", "medium_plane_airframe_1", draft)
	if out["role"] != "Tactical Bomber" {
		t.Errorf("role = %v", out["role"])
	}
	if cost := out["cost"].(float64); math.Abs(cost-38) > 1e-6 {
		t.Errorf("cost = %v, want 38", cost)
	}
	if got := designerValue(t, out, "combat", "air_agility"); math.Abs(got-35) > 1e-6 {
		t.Errorf("agility = %v, want 35", got)
	}
	slots := out["slots"].([]map[string]any)
	if slots[0]["module"] != "medium_bomb_bay" {
		t.Errorf("main weapon = %v", slots[0]["module"])
	}
	found := false
	for _, option := range slots[0]["allowed"].([]map[string]any) {
		found = found || option["name"] == "medium_bomb_bay"
	}
	if !found {
		t.Errorf("the main weapon slot does not offer the bomb bay: %v", slots[0]["allowed"])
	}

	names := []string{}
	for _, design := range out["designs"].([]map[string]any) {
		names = append(names, design["name"].(string))
	}
	if !strings.Contains(strings.Join(names, "|"), "He 111") || out["variant"] != false {
		t.Errorf("designs = %v, variant = %v", names, out["variant"])
	}

	tank := idx.EquipmentDesignerPayload("GER", "1936.11.1.12", "medium_tank_chassis_0")
	if upgrades := tank["upgrades"].([]map[string]any); len(upgrades) == 0 || upgrades[0]["max"] == nil {
		t.Errorf("upgrades = %v", upgrades)
	}
}

func TestRealGameCatalogueListsDesigns(t *testing.T) {
	idx := realTopbarIndex(t, false)
	items := idx.ProductionPayload("GER", "1936.1.1.12")["equipment"].([]map[string]any)
	byName := map[string]map[string]any{}
	firstType := map[string]int{}
	lastDesign := map[string]int{}
	for at, item := range items {
		byName[item["name"].(string)] = item
		group, _ := item["group"].(string)
		if design, _ := item["design"].(bool); design {
			lastDesign[group] = at
		} else if _, seen := firstType[group]; !seen {
			firstType[group] = at
		}
	}
	if rifle := byName["infantry_equipment_1"]; rifle == nil || rifle["label"] != "Karabiner 98k" || rifle["general_name"] != "Infantry Equipment I" {
		t.Errorf("rifle row = %v", rifle)
	}

	for name, want := range map[string]bool{"infantry_equipment_0": true, "medium_plane_airframe_0": true, "large_plane_airframe_0": false, "infantry_equipment_1": false} {
		if row := byName[name]; row == nil || (row["obsolete"] == true) != want {
			t.Errorf("%s obsolete = %v, want %v", name, row["obsolete"], want)
		}
	}
	panzer := byName["Panzer II Ausf. A"]
	if panzer == nil || panzer["general_name"] != "Improved Light Tank" || panzer["type"] != "light_tank_chassis_2" || panzer["designer"] != "tank" {
		t.Errorf("Panzer II row = %v", panzer)
	}
	if dornier := byName["Do 22"]; dornier == nil || dornier["general_name"] != "Basic Naval Bomber" || dornier["designer"] != "plane" {
		t.Errorf("Do 22 row = %v", dornier)
	}
	if hipper := byName["Admiral Hipper"]; hipper == nil || hipper["designer"] != "ship" || hipper["cost"].(float64) <= 0 {
		t.Errorf("Admiral Hipper row = %v", hipper)
	}
	for group, at := range lastDesign {
		if first, ok := firstType[group]; ok && at > first {
			t.Errorf("%s: a design after the first type", group)
		}
	}
}
