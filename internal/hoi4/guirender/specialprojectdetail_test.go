package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func outputStatsText(items []map[string]any) string {
	out := []string{}
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s %s", strings.TrimSpace(strings.ReplaceAll(texticonName.ReplaceAllString(item["label"].(string), ""), "£", "")), item["value"]))
	}
	return strings.Join(out, "|")
}

func TestRealGameEscortCarrierPage(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.SpecialProjectDetailPayload("ENG", "1936.1.1.12", "sp_naval_escort_carrier")
	if out["label"] != "Escort Carrier" || !strings.HasPrefix(out["description"].(string), "To safeguard our supply lines") {
		t.Errorf("label / description = %v / %v", out["label"], out["description"])
	}
	if got := strings.Join(out["outputs"].([]string), "|"); got != "Gain 100.00% discount for 1 design template(s). Can be used for: Escort Carrier" {
		t.Errorf("outputs = %s", got)
	}
	modules := out["modules"].([]map[string]any)
	if len(modules) != 1 {
		t.Fatalf("modules = %v", modules)
	}
	if got := outputStatsText(modules[0]["stats"].([]map[string]any)); got != "Carrier Sub Detection: 7.0|Carrier Surface Detection: 5.0|Max Speed: -5.0%|Deck size: 1|Production Cost: 700.00" {
		t.Errorf("deck stats = %s", got)
	}
	equipment := out["equipment"].([]map[string]any)
	if len(equipment) != 1 || equipment[0]["name"] != "ship_hull_escort_carrier" {
		t.Fatalf("equipment = %v", equipment)
	}
	want := "Service Manpower: 1500|Surface Detection: 20.0|Sub Detection: 1.0|Surface Visibility: 30.0|Max Speed: 27.5 kn|Max Range: 3000 km|" +
		"Weather Penalty: 100%|HP: 175.0|Reliability: 70.0%|Fuel Usage: 66.00|Production Cost: 1375.00|Naval Dominance: 175"
	if got := outputStatsText(equipment[0]["stats"].([]map[string]any)); got != want {
		t.Errorf("carrier stats =\n %s\nwant\n %s", got, want)
	}
	if resources := equipment[0]["resources"].([]map[string]any); len(resources) != 1 || resources[0]["resource"] != "steel" || resources[0]["amount"] != 3.0 {
		t.Errorf("carrier resources = %v", resources)
	}
	if equipment[0]["icon"] == nil || modules[0]["icon"] == nil {
		t.Errorf("pictures = %v / %v", equipment[0]["icon"], modules[0]["icon"])
	}
}

func TestRealGameJellicoePage(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.SpecialProjectDetailPayload("ENG", "1936.1.1.12", "sp_naval_super_heavy_battleship")
	if out["label"] != "Jellicoe Class Battleship" || !strings.HasPrefix(out["description"].(string), "The naval arms race may have diversified") {
		t.Errorf("label / description = %v / %v", out["label"], out["description"])
	}
	if got := strings.Join(out["outputs"].([]string), "|"); got != "Gain 100.00% discount for 1 design template(s). Can be used for: Super Heavy Battleship" {
		t.Errorf("outputs = %s", got)
	}
	modules := out["modules"].([]map[string]any)
	if len(modules) != 1 {
		t.Fatalf("modules = %v", modules)
	}
	if got := outputStatsText(modules[0]["stats"].([]map[string]any)); !strings.HasPrefix(got, "Max Speed: -12.0%|Torpedo Damage Reduction: 0.2|") {
		t.Errorf("armour stats = %s", got)
	}
	resources := []string{}
	for _, item := range modules[0]["resources"].([]map[string]any) {
		resources = append(resources, fmt.Sprintf("%v %v", item["resource"], item["amount"]))
	}
	if strings.Join(resources, "|") != "steel 2|chromium 1" {
		t.Errorf("armour resources = %v", resources)
	}
}

func TestRealGameSubmarinePages(t *testing.T) {
	idx := realTopbarIndex(t, false)
	for _, want := range []struct{ id, stats string }{
		{"sp_naval_cruiser_submarine", "Service Manpower: 200|Surface Detection: 20.0|Surface Visibility: 1.0|Sub Visibility: 20.0|Max Speed: 16.8 kn|Max Range: 5000 km|Weather Penalty: 100%|HP: 30.0|Reliability: 75.0%|Fuel Usage: 8.00|Production Cost: 539.00|Naval Dominance: 10"},
		{"sp_naval_midget_submarine", "Service Manpower: 100|Surface Detection: 20.0|Surface Visibility: 1.0|Sub Visibility: 15.0|Max Speed: 8.3 kn|Max Range: 500 km|Weather Penalty: 100%|HP: 5.0|Reliability: 60.0%|Fuel Usage: 1.00|Production Cost: 150.00|Naval Dominance: 7"},
	} {
		out := idx.SpecialProjectDetailPayload("ENG", "1936.1.1.12", want.id)
		equipment, _ := out["equipment"].([]map[string]any)
		if len(equipment) == 0 {
			t.Errorf("%s: no equipment (%v)", want.id, out["found"])
			continue
		}
		if got := outputStatsText(equipment[0]["stats"].([]map[string]any)); got != want.stats {
			t.Errorf("%s stats =\n %s\nwant\n %s", want.id, got, want.stats)
		}
	}
}
