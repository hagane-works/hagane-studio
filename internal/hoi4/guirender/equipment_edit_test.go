package guirender

import "testing"

func TestRealGameEquipmentPayload(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.EquipmentPayload("GER", "infantry_equipment_1")
	if out["found"] != true || out["file"] != "common/units/equipment/infantry.txt" || out["archetype"] != "infantry_equipment" || out["year"] != 1936 {
		t.Fatalf("payload = %v", out)
	}
	if out["country_label"] != "Karabiner 98k" || out["country_key"] != "GER_infantry_equipment_1" || out["label"] != "Infantry Equipment I" {
		t.Errorf("names = %v / %v / %v", out["country_label"], out["country_key"], out["label"])
	}
	own := out["numbers"].(map[string]float64)
	if own["defense"] != 22 || own["soft_attack"] != 6 || own["build_cost_ic"] != 0.5 {
		t.Errorf("own numbers = %v", own)
	}
	if _, listed := own["year"]; listed {
		t.Error("the year is not a stat")
	}
	inherited := out["inherited_resources"].(map[string]float64)
	if inherited["steel"] <= 0 {
		t.Errorf("the archetype's resources = %v", inherited)
	}
	if file, _, line, ok := idx.EquipmentSource("infantry_equipment_1"); !ok || file != "common/units/equipment/infantry.txt" || line <= 0 {
		t.Errorf("source = %s:%d %v", file, line, ok)
	}
	if missing := idx.EquipmentPayload("", "no_such_equipment"); missing["found"] != false {
		t.Errorf("missing = %v", missing)
	}
}
