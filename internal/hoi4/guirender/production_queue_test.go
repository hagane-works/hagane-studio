package guirender

import (
	"math"
	"testing"
)

func TestQueueOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	queue := idx.ProductionPayload("GER", "1936.1.1.12")["queue"].(map[string]any)
	if queue["military_used"] != 20 || queue["military_total"] != 28 || queue["naval_used"] != 10 || queue["naval_total"] != 10 {
		t.Errorf("factories %v/%v, dockyards %v/%v", queue["military_used"], queue["military_total"], queue["naval_used"], queue["naval_total"])
	}
	categories := queue["categories"].(map[string]int)
	want := map[string]int{"infantry": 15, "armor": 2, "air": 3, "capital_ships": 4, "screen_ships": 2, "other_ships": 4}
	for group, count := range want {
		if categories[group] != count {
			t.Errorf("%s uses %d factories, want %d", group, categories[group], count)
		}
	}
	modifiers := queue["modifiers"].(map[string]any)
	if math.Round(modifiers["production_output"].(float64)*100) != 17 || modifiers["efficiency_cap"] != 0.5 {
		t.Errorf("modifiers %v", modifiers)
	}
	lines := queue["lines"].([]map[string]any)
	if len(lines) != 21 {
		t.Fatalf("%d lines", len(lines))
	}
	for at, name := range map[int]string{0: "Karabiner 98k", 5: "Bf 109 D", 8: "Type II", 14: "Deutschland", 16: "Scharnhorst", 20: "Zerstörer 1934A"} {
		if lines[at]["name"] != name {
			t.Errorf("line %d is %v, want %s", at+1, lines[at]["name"], name)
		}
	}
	if lines[14]["general_name"] != "Panzerschiff" {
		t.Errorf("line 15 is a %v, want the Panzerschiff, not the old Deutschland class", lines[14]["general_name"])
	}
	for _, at := range []int{18, 19, 20} {
		if lines[at]["factories"] != 0 || lines[at]["speed"] != "§RNone§!" {
			t.Errorf("line %d: %v factories, %v", at+1, lines[at]["factories"], lines[at]["speed"])
		}
	}
	if lines[0]["factories"] != 10 || lines[0]["general_name"] != "Infantry Equipment I" {
		t.Errorf("line 1 %v", lines[0])
	}
}

func TestQueueHandsOutFactoriesInOrder(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/units/infantry.txt":           []byte(testSubUnits),
		"common/units/equipment/infantry.txt": []byte(testEquipment),
		"common/resources/00_resources.txt":   []byte(testResources),
		"history/countries/TAG - Tagland.txt": []byte(testLogisticsHistory),
		"history/units/TAG_1936.txt": []byte(testLogisticsOOB + `
instant_effect = {
	add_equipment_production = { equipment = { type = infantry_equipment_1 creator = TAG } requested_factories = 3 efficiency = 50 }
	add_equipment_production = { equipment = { type = infantry_equipment_1 creator = TAG } requested_factories = 4 efficiency = 50 }
}
`),
		"history/states/1-Home.txt": []byte("state = { id = 1 provinces = { 10 } history = { owner = TAG buildings = { arms_factory = 5 } } }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	queue := idx.ProductionPayload("TAG", "1936.1.1")["queue"].(map[string]any)
	lines := queue["lines"].([]map[string]any)
	if len(lines) != 2 || lines[0]["factories"] != 3 || lines[1]["factories"] != 2 || lines[1]["requested"] != 4 {
		t.Fatalf("lines %v", lines)
	}
	if queue["military_used"] != 5 || queue["military_total"] != 5 {
		t.Errorf("used %v of %v", queue["military_used"], queue["military_total"])
	}
}

func TestProducingTextPerDayOrPerMonth(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.producingText(52.88); got != "52.88/day" {
		t.Errorf("%q", got)
	}
	if got := idx.producingText(0.777); got != "5.44/week" {
		t.Errorf("%q", got)
	}
	if got := idx.producingText(0.107); got != "3.21/month" {
		t.Errorf("%q", got)
	}
}
