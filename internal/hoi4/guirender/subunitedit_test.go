package guirender

import (
	"fmt"
	"testing"
)

func TestRealGameSubUnitEditPayload(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.SubUnitEditPayload("Infantry")
	if out["found"] != true || out["name"] != "Infantry" || out["file"] != "common/units/infantry.txt" {
		t.Fatalf("payload = %v", out)
	}
	stats := out["stats"].(map[string]any)
	got := fmt.Sprint(stats["combat_width"], stats["max_strength"], stats["max_organisation"], stats["default_morale"], stats["manpower"],
		stats["training_time"], stats["suppression"], stats["weight"], stats["supply_consumption"])
	if want := "2 25 60 0.3 1000 90 1.5 0.5 0.06"; got != want {
		t.Errorf("stats = %s, want %s", got, want)
	}
	need := out["need"].([]map[string]any)
	if len(need) != 1 || need[0]["equipment"] != "infantry_equipment" || need[0]["amount"] != 100.0 || need[0]["name"] != "Infantry Equipment" {
		t.Errorf("need = %v", need)
	}
	if missing := idx.SubUnitEditPayload("no_such_battalion"); missing["found"] != false {
		t.Errorf("missing = %v", missing)
	}
}
