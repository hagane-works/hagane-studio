package guirender

import "testing"

const testSubUnits = `
sub_units = {
	infantry = {
		abbreviation = "INF"
		sprite = infantry
		group = infantry
		combat_width = 2
		max_strength = 25
		max_organisation = 60
		default_morale = 0.3
		manpower = 1000
		training_time = 90
		weight = 0.5
		supply_consumption = 0.06
		categories = { category_front_line category_all_infantry }
		need = { infantry_equipment = 100 }
	}
	light_armor = {
		abbreviation = "LAR"
		sprite = light_armor
		group = armor
		combat_width = 2
		max_strength = 25
		max_organisation = 10
		default_morale = 0.3
		manpower = 500
		training_time = 120
		weight = 2
		supply_consumption = 0.3
		need = { light_tank_chassis = 50 }
	}
	engineer = {
		abbreviation = "ENG"
		sprite = infantry
		map_icon_category = infantry
		group = support
		regimental = no
		affects_speed = no
		combat_width = 0
		max_strength = 2
		max_organisation = 20
		default_morale = 0.3
		manpower = 300
		training_time = 120
		weight = 0.1
		supply_consumption = 0.02
		need = { support_equipment = 10 }
	}
	marines = {
		abbreviation = "MAR"
		sprite = marine
		group = infantry
		combat_width = 2
		max_strength = 25
		max_organisation = 60
		manpower = 1000
		categories = { category_front_line category_special_forces }
		need = { infantry_equipment = 100 }
	}
}
`

const testEquipment = `
equipments = {
	infantry_equipment = {
		year = 1936
		is_archetype = yes
		maximum_speed = 4
		defense = 20
		breakthrough = 2
		hardness = 0
		soft_attack = 3
		hard_attack = 0.5
		ap_attack = 1
		build_cost_ic = 0.43
		resources = { steel = 2 }
	}
	infantry_equipment_1 = {
		year = 1936
		archetype = infantry_equipment
		defense = 22
		soft_attack = 6
		build_cost_ic = 0.5
	}
	infantry_equipment_2 = {
		year = 1939
		archetype = infantry_equipment
		parent = infantry_equipment_1
		soft_attack = 7
	}
	light_tank_chassis = {
		year = 1936
		is_archetype = yes
		maximum_speed = 8
		hardness = 0.8
		armor_value = 10
		soft_attack = 4
		build_cost_ic = 4
		resources = { steel = 4 tungsten = 1 }
	}
	support_equipment = {
		year = 1936
		is_archetype = yes
		build_cost_ic = 4
		resources = { steel = 1 }
	}
}
`

const testDesignerOOB = `
division_template = {
	name = "Infanterie-Division"
	regiments = {
		infantry = { x = 0 y = 0 }
		infantry = { x = 0 y = 1 }
		light_armor = { x = 1 y = 0 }
	}
	support = {
		engineer = { x = 0 y = 0 }
	}
}
division_template = {
	name = "Marine-Brigade"
	regiments = { marines = { x = 0 y = 0 } }
}
`

const testDesignerHistory = `
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
set_oob = "TAG_1936"
set_research_slots = 3
set_technology = {
	infantry_weapons = 1
	tech_trucks = 1
	marines = 0
}
1939.1.1 = {
	set_technology = { infantry_weapons1 = 1 }
}
`

const testSubUnitIcons = `spriteTypes = {
	spriteType = { name = "GFX_unit_infantry_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_infantry_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_unit_engineer_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_engineer_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_unit_light_armor_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_light_tank_icon.dds" noOfFrames = 2 }
	spriteType = { name = "GFX_unit_marine_icon_medium" textureFile = "gfx/interface/counters/divisions_large/unit_marine_icon.dds" noOfFrames = 2 }
}`

func designerIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/units/infantry.txt":           []byte(testSubUnits),
		"common/units/equipment/infantry.txt": []byte(testEquipment),
		"history/units/TAG_1936.txt":          []byte(testDesignerOOB),
		"history/countries/TAG - Tagland.txt": []byte(testDesignerHistory),
		"interface/subuniticons.gfx":          []byte(testSubUnitIcons),
		"localisation/english/units.yml": []byte(
			"l_english:\n infantry: \"Infantry\"\n light_armor: \"Light Tank\"\n" +
				" infantry_equipment_1: \"Infantry Equipment I\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestDesignerFillsBothGridsOneCellPerBattalion(t *testing.T) {
	payload := designerIndex(t).DesignerPayload("TAG", "1936.1.1", "")
	if payload["name"] != "Infanterie-Division" {
		t.Fatalf("template = %v, want the first the battle order declares", payload["name"])
	}
	regiments, _ := payload["regiments"].([]map[string]any)

	if len(regiments) != 3 {
		t.Fatalf("regiment cells = %d, want one per battalion", len(regiments))
	}
	if regiments[0]["icon"] != "GFX_unit_infantry_icon_medium" {
		t.Errorf("cell icon = %v, want the battalion's own counter", regiments[0]["icon"])
	}
	if regiments[0]["label"] != "Infantry" || regiments[0]["abbreviation"] != "INF" {
		t.Errorf("cell = %v", regiments[0])
	}
	support, _ := payload["support"].([]map[string]any)
	if len(support) != 1 {
		t.Fatalf("support cells = %v", support)
	}

	if support[0]["icon"] != "GFX_unit_engineer_icon_medium" {
		t.Errorf("engineer icon = %v, want GFX_unit_engineer_icon_medium, not the infantry helmet", support[0]["icon"])
	}
}

func TestBattalionIconFallsBackToItsSpriteOnlyWithoutACounterOfItsOwn(t *testing.T) {
	idx := designerIndex(t)
	regiments, _ := idx.DesignerPayload("TAG", "1936.1.1", "Marine-Brigade")["regiments"].([]map[string]any)

	if len(regiments) != 1 || regiments[0]["icon"] != "GFX_unit_marine_icon_medium" {
		t.Errorf("marines cell = %v, want the sprite family's counter", regiments)
	}
}

func TestDivisionStatsFollowTheEnginesOwnRules(t *testing.T) {
	payload := designerIndex(t).DesignerPayload("TAG", "1936.1.1", "")
	stats, _ := payload["stats"].(DivisionStats)

	if stats["combat_width"] != 6 {
		t.Errorf("combat width = %v, want 2+2+2 (the engineer is 0)", stats["combat_width"])
	}
	if stats["max_strength"] != 77 {
		t.Errorf("HP = %v, want 25+25+25+2", stats["max_strength"])
	}
	if stats["manpower"] != 2800 {
		t.Errorf("manpower = %v, want 1000+1000+500+300", stats["manpower"])
	}

	if got := stats["max_organisation"]; got < 37.4 || got > 37.6 {
		t.Errorf("org = %v, want (60+60+10+20)/4", got)
	}

	if stats["speed"] != 4 {
		t.Errorf("speed = %v, want the infantry's 4", stats["speed"])
	}

	if got := stats["hardness"]; got < 0.25 || got > 0.27 {
		t.Errorf("hardness = %v, want 0.8*25/77", got)
	}

	if stats["training_time"] != 120 {
		t.Errorf("training = %v", stats["training_time"])
	}

	if stats["soft_attack"] != 6+6+4 {
		t.Errorf("soft attack = %v, want 6 per infantry battalion + the tank's 4", stats["soft_attack"])
	}

	if got := stats["ap_attack"]; got > 4.1 {
		t.Errorf("piercing = %v, want it blended rather than summed", got)
	}

	if stats["recovery_rate"] != 0.3 {
		t.Errorf("recovery rate = %v, want the average default_morale", stats["recovery_rate"])
	}
}

func TestCombatStatsDoNotScaleWithEquipmentCount(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/units/infantry.txt": []byte(`
sub_units = {
	colonial = {
		sprite = infantry
		combat_width = 2  max_strength = 25  max_organisation = 50
		manpower = 1000  training_time = 80  suppression = 1.5
		weight = 0.5  supply_consumption = 0.06
		need = { infantry_equipment = 100 }
	}
	para = {
		sprite = marine
		combat_width = 2  max_strength = 22  max_organisation = 70
		manpower = 1000  training_time = 150  suppression = 1
		weight = 0.5  supply_consumption = 0.05
		soft_attack = 0.05
		breakthrough = 0.10
		maximum_speed = 0.05
		need = { infantry_equipment = 130 }
	}
}`),
		"common/units/equipment/infantry.txt": []byte(`
equipments = {
	infantry_equipment = { year = 1936 is_archetype = yes maximum_speed = 4 reliability = 0.9 }
	infantry_equipment_2 = {
		year = 1939  archetype = infantry_equipment
		soft_attack = 9  hard_attack = 1.5  defense = 28  breakthrough = 4  ap_attack = 5
	}
}`),
		"history/units/TAG_1939.txt": []byte(`
division_template = { name = "Schutztruppe" regiments = {
	colonial = { x = 0 y = 0 } colonial = { x = 0 y = 1 } colonial = { x = 0 y = 2 }
	colonial = { x = 1 y = 0 } colonial = { x = 1 y = 1 } } }
division_template = { name = "Fallschirmjager" regiments = {
	para = { x = 0 y = 0 } para = { x = 0 y = 1 } para = { x = 0 y = 2 }
	para = { x = 1 y = 0 } para = { x = 1 y = 1 } } }`),
		"history/countries/TAG - Tagland.txt": []byte(`
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
set_oob = "TAG_1939"`),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	colonial, _ := idx.DesignerPayload("TAG", "1939.8.14", "Schutztruppe")["stats"].(DivisionStats)
	para, _ := idx.DesignerPayload("TAG", "1939.8.14", "Fallschirmjager")["stats"].(DivisionStats)

	if colonial["soft_attack"] != 45 {
		t.Errorf("colonial soft attack = %v, want 45.0 as the game shows", colonial["soft_attack"])
	}
	if got := para["soft_attack"]; got < 47.2 || got > 47.3 {
		t.Errorf("paratrooper soft attack = %v, want 47.2 (5 x 9 x 1.05)", got)
	}
	if colonial["breakthrough"] != 20 || para["breakthrough"] != 22 {
		t.Errorf("breakthrough = %v / %v, want 20.0 and 22.0", colonial["breakthrough"], para["breakthrough"])
	}

	if colonial["defense"] != 140 || para["defense"] != 140 {
		t.Errorf("defense = %v / %v, want 140.0 for both", colonial["defense"], para["defense"])
	}
	if colonial["speed"] != 4 || para["speed"] != 4.2 {
		t.Errorf("speed = %v / %v, want 4.0 and 4.2 km/h", colonial["speed"], para["speed"])
	}

	checks := []struct {
		name string
		got  float64
		want float64
	}{
		{"colonial HP", colonial["max_strength"], 125},
		{"colonial org", colonial["max_organisation"], 50},
		{"colonial suppression", colonial["suppression"], 7.5},
		{"colonial supply", colonial["supply_use"], 0.3},
		{"colonial width", colonial["combat_width"], 10},
		{"colonial manpower", colonial["manpower"], 5000},
		{"colonial training", colonial["training_time"], 80},
		{"colonial piercing", colonial["ap_attack"], 5},
		{"para HP", para["max_strength"], 110},
		{"para supply", para["supply_use"], 0.25},
		{"para suppression", para["suppression"], 5},
		{"para training", para["training_time"], 150},
		{"para hard attack", para["hard_attack"], 7.5},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	bill, _ := idx.DesignerPayload("TAG", "1939.8.14", "Fallschirmjager")["equipment"].([]map[string]any)
	if len(bill) != 1 || bill[0]["amount"] != float64(650) {
		t.Errorf("equipment bill = %v, want 650 infantry equipment", bill)
	}
}

func TestDesignerPicksTheYearsEquipmentModel(t *testing.T) {
	idx := designerIndex(t)
	at1936 := idx.DesignerPayload("TAG", "1936.1.1", "")
	at1939 := idx.DesignerPayload("TAG", "1939.1.1", "")
	early, _ := at1936["equipment"].([]map[string]any)
	late, _ := at1939["equipment"].([]map[string]any)
	if len(early) == 0 || early[0]["model"] != "infantry_equipment_1" {
		t.Fatalf("1936 model = %v", early)
	}
	if late[0]["model"] != "infantry_equipment_2" {
		t.Errorf("1939 model = %v, want the later variant", late[0]["model"])
	}

	stats, _ := late[0]["stats"].(map[string]float64)
	if stats["defense"] != 20 || stats["soft_attack"] != 7 {
		t.Errorf("defense, soft attack = %v, %v, want the archetype's 20 and its own 7", stats["defense"], stats["soft_attack"])
	}
	if early[0]["amount"] != float64(200) {
		t.Errorf("equipment bill = %v, want 100 per infantry battalion", early[0]["amount"])
	}
}

func TestSpecialForcesComeFromTheBattalionsOwnCategory(t *testing.T) {
	idx := designerIndex(t)
	if idx.DesignerPayload("TAG", "1936.1.1", "")["is_special"] != false {
		t.Error("a line division should not read as special forces")
	}
	if idx.DesignerPayload("TAG", "1936.1.1", "Marine-Brigade")["is_special"] != true {
		t.Error("marines carry category_special_forces")
	}
}

func TestStartingTechnologiesComeFromHistoryWithTheirDate(t *testing.T) {
	idx := designerIndex(t)
	early := idx.ResearchedBy("TAG", "1936.1.1")
	if !contains(early, "infantry_weapons") || !contains(early, "tech_trucks") {
		t.Errorf("1936 technologies = %v", early)
	}

	if contains(early, "marines") {
		t.Errorf("a technology set to 0 counted as researched: %v", early)
	}

	if late := idx.ResearchedBy("TAG", "1939.1.1"); !contains(late, "infantry_weapons1") {
		t.Errorf("1939 technologies = %v", late)
	}
	if contains(early, "infantry_weapons1") {
		t.Errorf("1936 picked up the 1939 block: %v", early)
	}
}
