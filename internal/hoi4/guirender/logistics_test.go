package guirender

import "testing"

const testLogisticsOOB = `
division_template = {
	name = "Infanterie-Division"
	regiments = {
		infantry = { x = 0 y = 0 }
		infantry = { x = 0 y = 1 }
	}
	support = {
		engineer = { x = 0 y = 0 }
	}
}
units = {
	division = { name = "1. Div" division_template = "Infanterie-Division" start_experience_factor = 0.3 }
	division = { name = "2. Div" division_template = "Infanterie-Division" }
	fleet = {
		name = "Flotte"
		task_force = {
			name = "TF"
			ship = { name = "Emden" definition = light_cruiser equipment = { light_tank_chassis = { amount = 3 owner = TAG } } }
		}
	}
}
air_wings = {
	100 = {
		infantry_equipment_1 = { owner = "TAG" amount = 7 }
	}
}
`

const testLogisticsHistory = `
capital = 1
set_oob = "TAG_1936"
`

func logisticsIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/units/infantry.txt":           []byte(testSubUnits),
		"common/units/equipment/infantry.txt": []byte(testEquipment),
		"common/resources/00_resources.txt":   []byte(testResources),
		"history/units/TAG_1936.txt":          []byte(testLogisticsOOB),
		"history/countries/TAG - Tagland.txt": []byte(testLogisticsHistory),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestNeedIsTheOrderOfBattle(t *testing.T) {
	need := logisticsIndex(t).equipmentNeed("TAG", "1936.1.1")

	if need["infantry_equipment"] != 400 {
		t.Errorf("infantry_equipment = %v, want 2 divisions x 2 battalions x 100", need["infantry_equipment"])
	}

	if need["infantry_equipment_1"] != 7 {
		t.Errorf("the wing's own model = %v, want 7", need["infantry_equipment_1"])
	}
	if need["light_tank_chassis"] != 3 {
		t.Errorf("the ship's equipment = %v, want 3", need["light_tank_chassis"])
	}
}

func TestLogisticsRowsCarryTheirResourceBill(t *testing.T) {
	payload := logisticsIndex(t).LogisticsPayload("TAG", "1936.1.1")
	rows, _ := payload["equipment"].([]map[string]any)
	if len(rows) == 0 {
		t.Fatal("no rows")
	}

	needs := map[string]int{}
	for _, row := range rows {
		if row["archetype"] != "infantry_equipment" {
			continue
		}
		model, _ := row["model"].(string)
		amount, _ := row["needs"].(int)
		needs[model] = amount
		if resources, _ := row["resources"].(map[string]float64); len(resources) == 0 {
			t.Errorf("row has no resource bill: %v", row)
		}
	}
	if len(needs) != 1 {
		t.Fatalf("infantry rows = %v, want one row per model", needs)
	}
	if needs["infantry_equipment_1"] != 407 {
		t.Errorf("infantry row = %v, want the divisions' 400 plus the wing's 7", needs)
	}
}

func TestProductionLinesAreReadAtAnyDepth(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tagland.txt": []byte(testLogisticsHistory),
		"history/units/TAG_1936.txt": []byte(testLogisticsOOB + `
instant_effect = {
	add_equipment_production = {
		equipment = { type = infantry_equipment_1 creator = TAG }
		requested_factories = 10
		progress = 0.1
		efficiency = 50
		industrial_manufacturer = mio:TAG_rifles
	}
	if = {
		limit = { has_dlc = "By Blood Alone" }
		add_equipment_production = {
			equipment = { type = light_tank_chassis version_name = "Panzer II" creator = "TAG" }
			requested_factories = 2
		}
	}
}
`),
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := idx.ProductionLinesOf("TAG", "1936.1.1")
	if len(lines) != 2 {
		t.Fatalf("lines %+v", lines)
	}
	first := lines[0]
	if first.Equipment != "infantry_equipment_1" || first.Factories != 10 || first.Progress != 0.1 || first.Efficiency != 50 || first.Manufacturer != "TAG_rifles" || first.Creator != "TAG" {
		t.Errorf("first line %+v", first)
	}
	if lines[1].Version != "Panzer II" || lines[1].Factories != 2 {
		t.Errorf("second line %+v", lines[1])
	}
}
