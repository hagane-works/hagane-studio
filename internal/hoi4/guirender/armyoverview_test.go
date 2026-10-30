package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func TestRealGameDivisionOverview(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.ArmyOverviewPayload("GER", "1936.1.1.12")
	texts := out["texts"].(map[string]any)
	if texts["divisions"] != "30 Divisions" || texts["battalions"] != "315 Battalions" || out["special_forces_cap"] != 24 || texts["theatres"] != "Active in 0 Theater(s)" {
		t.Errorf("header = %v %v %v %v", texts["divisions"], texts["battalions"], out["special_forces_cap"], texts["theatres"])
	}
	pills := []string{}
	for _, pill := range out["pills"].([]map[string]any) {
		pills = append(pills, fmt.Sprintf("%v %v", pill["count"], pill["icon"]))
	}
	if got := strings.Join(pills, "|"); got != "24 GFX_unit_infantry_icon_medium|3 GFX_unit_light_armor_icon_medium|1 GFX_unit_motorized_icon_medium|1 GFX_unit_cavalry_icon_medium|1 GFX_unit_mountaineers_icon_medium" {
		t.Errorf("pills = %s", got)
	}
	names := []string{}
	for _, row := range out["divisions"].([]map[string]any)[:6] {
		names = append(names, fmt.Sprintf("%v L%v", row["name"], row["level"]))
	}
	if got := strings.Join(names, "|"); got != "1. Infanterie-Division L2|1. Panzer-Division L2|10. Infanterie-Division L2|11. Infanterie-Division L2|12. Infanterie-Division L2|13. Infanterie-Division L2" {
		t.Errorf("first rows = %s", got)
	}

	officers := 0
	for _, row := range out["divisions"].([]map[string]any) {
		if location, _ := row["location"].(int); location <= 0 {
			t.Errorf("%v has no province: %v", row["name"], row["location"])
		}
		if officer, _ := row["officer"].(string); officer != "" {
			officers++
		}
	}
	if officers == 0 {
		t.Error("no division's commander was read")
	}
}
