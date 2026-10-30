package guirender

import "testing"

const oob1936 = `
division_template = {
	name = "Juntuán"
	division_names_group = CHI_INF_01
	regiments = {
		infantry = { x = 0 y = 0 }
		infantry = { x = 0 y = 1 }
		infantry = { x = 1 y = 0 }
	}
	priority = 0
}
division_template = {
	name = "Panzer-Division"
	regiments = {
		light_armor = { x = 0 y = 0 }
		light_armor = { x = 0 y = 1 }
		motorized = { x = 1 y = 0 }
	}
	support = {
		engineer = { x = 0 y = 0 }
		artillery = { x = 0 y = 1 }
	}
	priority = 2
}
units = { division = { name = "1. Division" } }
`

const oob1939 = `
division_template = {
	name = "Late Division"
	regiments = { infantry = { x = 0 y = 0 } }
}
`

const oobHistory = `
capital = 64
set_politics = { ruling_party = fascism elections_allowed = no }
IF = {
	limit = { has_dlc = "No Compromise, No Surrender" }
	set_oob = "TAG_1936_sea"
	else = { set_oob = "TAG_1936_legacy" }
}
1939.1.1 = {
	set_oob = "TAG_1939"
}
`

func oobIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/units/TAG_1936_sea.txt":      []byte(oob1936),
		"history/units/TAG_1936_legacy.txt":   []byte(oob1939),
		"history/units/TAG_1939.txt":          []byte(oob1939),
		"history/countries/TAG - Tagland.txt": []byte(oobHistory),
		"localisation/english/units.yml":      []byte("l_english:\n infantry: \"Infantry\"\n light_armor: \"Light Tank\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestDivisionTemplatesCountRegimentsByKind(t *testing.T) {
	payload := oobIndex(t).DivisionsPayload("TAG", "1936.1.1")
	templates, _ := payload["templates"].([]map[string]any)
	if len(templates) != 2 {
		t.Fatalf("templates = %d, want the two the 1936 order declares", len(templates))
	}
	if templates[0]["name"] != "Juntuán" || templates[0]["width"] != 3 {
		t.Errorf("first template = %v", templates[0])
	}
	regiments, _ := templates[1]["regiments"].([]map[string]any)

	if len(regiments) != 2 || regiments[0]["type"] != "light_armor" || regiments[0]["count"] != 2 {
		t.Fatalf("panzer regiments = %v", regiments)
	}
	if regiments[0]["label"] != "Light Tank" {
		t.Errorf("sub-unit name = %v, want the localised one", regiments[0]["label"])
	}
	support, _ := templates[1]["support"].([]map[string]any)
	if len(support) != 2 {
		t.Errorf("support = %v", support)
	}
	if templates[1]["priority"] != 2 {
		t.Errorf("priority = %v", templates[1]["priority"])
	}

	if templates[0]["priority"] != 0 {
		t.Errorf("Juntuán priority = %v, want the 0 it declares", templates[0]["priority"])
	}
}

func TestTemplateWithoutPriorityIsNormal(t *testing.T) {
	payload := oobIndex(t).DivisionsPayload("TAG", "1939.8.14")
	templates, _ := payload["templates"].([]map[string]any)
	for _, template := range templates {
		if template["name"] == "Late Division" {
			if template["priority"] != 1 {
				t.Errorf("Late Division priority = %v, want 1", template["priority"])
			}
			return
		}
	}
	t.Fatalf("templates = %v, want Late Division", templates)
}

func TestBattleOrderFollowsScenarioAndInstalledContent(t *testing.T) {
	idx := oobIndex(t)

	at1936 := idx.OOBOf("TAG", "1936.1.1")
	if !contains(at1936, "TAG_1936_sea") || contains(at1936, "TAG_1936_legacy") {
		t.Errorf("1936 order = %v", at1936)
	}
	if contains(at1936, "TAG_1939") {
		t.Errorf("1936 picked up the later scenario's order: %v", at1936)
	}
	names := []string{}
	for _, template := range idx.DivisionTemplatesOf("TAG", "1939.8.14") {
		names = append(names, template.Name)
	}
	if !contains(names, "Late Division") || !contains(names, "Juntuán") {
		t.Errorf("1939 templates = %v, want both orders' worth", names)
	}
}

func TestMissingBattleOrderIsNotAnError(t *testing.T) {
	payload := oobIndex(t).DivisionsPayload("ZZZ", "1936.1.1")
	templates, _ := payload["templates"].([]map[string]any)
	if len(templates) != 0 {
		t.Errorf("a country with no history returned %v", templates)
	}
}
