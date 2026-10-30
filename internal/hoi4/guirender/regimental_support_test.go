package guirender

import (
	"reflect"
	"testing"
)

const regimentalTestUnits = `sub_units = {
 field_guns = { group = support divisional = no regimental = yes manpower = 100 max_strength = 1 soft_attack = 2
   terrain = { forest = { attack = 0.1 } } }
 division_only = { group = support regimental = no manpower = 200 }
 locked_company = { group = support regimental = yes active = no }
}`

func regimentalIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/units/test.txt":               []byte(testSubUnits + regimentalTestUnits),
		"history/countries/TAG - Tagland.txt": []byte(`set_oob = TAG_1936`),
		"history/units/TAG_1936.txt": []byte(`division_template = {
 name = "Test" regiments = { infantry = { x = 0 y = 0 } infantry = { x = 0 y = 1 } infantry = { x = 0 y = 2 }
 infantry = { x = 1 y = 0 } infantry = { x = 1 y = 1 } infantry = { x = 1 y = 2 } }
 regimental_support = { field_guns = { x = 0 y = 0 } field_guns = { x = 1 y = 0 } }
} units = { division = { division_template = "Test" } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestRegimentalSupportParsesAndReturnsSeparateCells(t *testing.T) {
	idx := regimentalIndex(t)
	data := idx.DesignerPayload("TAG", "1936.1.1", "Test")
	cells := data["regimental_support"].([]map[string]any)
	if len(cells) != 2 || cells[0]["type"] != "field_guns" || cells[1]["x"] != 1 {
		t.Fatalf("regimental cells: %#v", cells)
	}
	if len(data["regiments"].([]map[string]any)) != 6 || len(data["support"].([]map[string]any)) != 0 {
		t.Fatal("regimental support leaked into another grid")
	}
	stats := data["stats"].(DivisionStats)
	if stats["manpower"] != 6200 {
		t.Fatalf("support omitted from manpower: %v", stats)
	}
	if idx.fieldedManpower("TAG", "1936.1.1") != 6200 {
		t.Fatal("deployed manpower omits support")
	}
}

func TestRegimentalSupportPreviewDoesNotAccumulateSavedCompanies(t *testing.T) {
	idx := regimentalIndex(t)
	data := idx.DesignerPayloadFor("TAG", "1936.1.1", "Test", "r:0:0:infantry,rs:0:0:field_guns")
	if len(data["regiments"].([]map[string]any)) != 1 || len(data["regimental_support"].([]map[string]any)) != 1 {
		t.Fatalf("preview did not replace all three grids: %#v", data)
	}
	if data["stats"].(DivisionStats)["manpower"] != 1100 {
		t.Fatal("preview stats omit support")
	}
	empty := idx.DesignerPayloadFor("TAG", "1936.1.1", "Test", "r:0:0:infantry")
	if len(empty["regimental_support"].([]map[string]any)) != 0 {
		t.Fatal("old support survived preview")
	}
}

func TestRegimentalSupportAvailableHonoursUnitFlagsAndTechnology(t *testing.T) {
	idx := regimentalIndex(t)
	if got := idx.RegimentalSupportAvailable("TAG", "1936.1.1"); !reflect.DeepEqual(got, []string{"field_guns"}) {
		t.Fatalf("regimental company pool: %v", got)
	}
	for _, row := range idx.designerAvailable("TAG", "1936.1.1", false) {
		if row["type"] == "division_only" && row["regimental"] != false {
			t.Fatal("picker lost regimental=no")
		}
	}
}
