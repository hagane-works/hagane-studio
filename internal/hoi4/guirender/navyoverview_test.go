package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func TestRealGameNavyOverview(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.NavyOverviewPayload("GER", "1936.1.1.12")
	rows := []string{}
	for _, row := range out["task_forces"].([]map[string]any) {
		rows = append(rows, fmt.Sprintf("%v/%v/%v/%v/%v", row["name"], row["region"], row["size"], row["class"], row["pride"]))
	}
	want := "Hochseeflotte/North-West Germany/12/heavy_cruiser/true|" +
		"I. U-Boots Flottille Weddigen/Jutland Peninsula/8/submarine/false|" +
		"II. U-Boots-Ausbildungs-Gruppe/North-West Germany/6/submarine/false|" +
		"Marineschule Kiel/Jutland Peninsula/2/light_cruiser/false|" +
		"Ostseeflotte/Western Poland/10/battleship/false"
	if got := strings.Join(rows, "|"); got != want {
		t.Errorf("rows =\n %s\nwant\n %s", got, want)
	}
	pills := []string{}
	for _, pill := range out["pills"].([]map[string]any) {
		pills = append(pills, fmt.Sprintf("%v %v", pill["count"], pill["class"]))
	}
	if got := strings.Join(pills, "|"); got != "2 battleship|2 heavy_cruiser|8 light_cruiser|12 destroyer|14 submarine" {
		t.Errorf("pills = %s", got)
	}
	texts := out["texts"].(map[string]any)
	if texts["fleets"] != "5 Fleets" || texts["theatres"] != "Active in 3 Strategic area(s)" {
		t.Errorf("texts = %v", texts)
	}
}
