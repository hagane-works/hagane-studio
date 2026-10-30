package guirender

import (
	"fmt"
	"testing"
)

func TestIndustrialOrgTraitsCarryTheirValues(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tag.txt": []byte("capital = 1\n"),
		"common/military_industrial_organization/organizations/TAG.txt": []byte(`
			generic_works = {
				allowed = { always = no }
				trait = { token = root position = { x = 1 y = 0 } equipment_bonus = { soft_attack = 0.05 reliability = 0.02 } }
				trait = { token = child position = { x = 1 y = 2 } relative_position_id = root production_bonus = { production_cost_factor = -0.05 } }
			}
			TAG_works = {
				include = generic_works
				allowed = { original_tag = TAG }
				override_trait = { token = root equipment_bonus = { hard_attack = 0.1 } }
			}
		`),
	})
	if err != nil {
		t.Fatal(err)
	}
	byToken := map[string]map[string]any{}
	for _, trait := range idx.IndustrialOrgPayload("TAG", "1936.1.1.12", "TAG_works")["traits"].([]map[string]any) {
		byToken[trait["token"].(string)] = trait
	}
	if got := fmt.Sprint(byToken["root"]["values"]); got != "map[equipment_bonus:[map[key:hard_attack value:0.1]]]" {
		t.Errorf("root values = %s", got)
	}
	if got := fmt.Sprint(byToken["child"]["values"], " ", byToken["child"]["relative_to"]); got != "map[production_bonus:[map[key:production_cost_factor value:-0.05]]] root" {
		t.Errorf("child = %s", got)
	}
	if _, ok := byToken["root"]["relative_to"]; ok {
		t.Error("root counts from nothing")
	}
}
