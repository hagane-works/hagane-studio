package guirender

import (
	"encoding/json"
	"testing"
)

func techItems(t *testing.T, idx *Index, folder string) (map[string]map[string]any, map[string]any) {
	t.Helper()
	body, err := json.Marshal(idx.TechTreePayload("GER", "1936.1.1", folder))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	items := map[string]map[string]any{}
	for _, raw := range payload["technologies"].([]any) {
		item := raw.(map[string]any)
		items[item["name"].(string)] = item
	}
	return items, payload
}

func TestRealGameResearchCards(t *testing.T) {
	idx := realTopbarIndex(t, false)
	cards := map[string]map[string]string{

		"infantry_folder": {"infantry_weapons": "Gewehr 98", "tech_trucks": "ALZ 13"},
		"mtgnavalfolder": {
			"basic_ship_hull_heavy": "Basic Heavy Ship",

			"early_ship_hull_carrier": "Deck Conversions",
		},
		"nsb_armour_folder": {
			"basic_light_tank_chassis": "1934 Light",
			"gwtank_chassis":           "Inter-War Tank Development",

			"amphibious_drive": "Amphibious Drive",
		},
		"mtgnavalsupportfolder": {"sp_improved_repair_ship": "Improved Repair Ship"},
	}
	for folder, want := range cards {
		items, _ := techItems(t, idx, folder)
		for name, label := range want {
			item := items[name]
			if item == nil {
				t.Errorf("%s: %s missing", folder, name)
				continue
			}
			if item["card"] != true || item["card_label"] != label {
				t.Errorf("%s: %s card=%v label=%q, want a card named %q", folder, name, item["card"], item["card_label"], label)
			}
		}
	}

	infantry, payload := techItems(t, idx, "infantry_folder")
	if key := infantry["infantry_weapons"]["card_label_key"]; key != "GER_infantry_equipment_0_short" {
		t.Errorf("infantry_weapons card_label_key = %v", key)
	}
	armour, _ := techItems(t, idx, "nsb_armour_folder")
	if key := armour["basic_light_tank_chassis"]["card_label_key"]; key != "basic_light_tank_chassis_short" {
		t.Errorf("basic_light_tank_chassis card_label_key = %v", key)
	}
	if payload["base_tech_cost"] != float64(110) {
		t.Errorf("base_tech_cost = %v, want 110", payload["base_tech_cost"])
	}

	for folder, name := range map[string]string{
		"support_folder":        "sp_armored_motorized_military_police_tech",
		"mtgnavalsupportfolder": "mtg_landing_craft",
	} {
		items, _ := techItems(t, idx, folder)
		if item := items[name]; item == nil || item["card"] == true {
			t.Errorf("%s: %s = %v, want a small square", folder, name, item)
		}
	}
}

func TestRealGameResearchSpecialProjects(t *testing.T) {
	idx := realTopbarIndex(t, false)
	items, payload := techItems(t, idx, "support_folder")

	if items["tech_recon"]["state"] != "researched" {
		t.Fatalf("tech_recon = %v", items["tech_recon"]["state"])
	}
	sp := items["sp_helicopter_artillery_observers_tech"]
	if sp["special_project"] != true || sp["state"] != "unavailable" {
		t.Errorf("sp_helicopter_artillery_observers_tech = %v", sp)
	}

	offset, _ := payload["special_project_offset"].(map[string]any)
	if offset["x"] != float64(-10) || offset["y"] != float64(-10) {
		t.Errorf("special_project_offset = %v, want -10,-10", payload["special_project_offset"])
	}
}
