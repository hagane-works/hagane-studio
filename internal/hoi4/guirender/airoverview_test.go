package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func TestRealGameAirOverview(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.AirOverviewPayload("GER", "1936.1.1.12")
	rows := []string{}
	for _, row := range out["wings"].([]map[string]any) {
		rows = append(rows, fmt.Sprintf("%v/%v/%v", row["name"], row["base"], row["count"]))
	}
	want := []string{
		"Küstenfliegergruppe 106/Holstein/72",
		"KG 1 'Hindenburg'/Brandenburg/80",
		"Kampfgeschwader 253/Niederschlesien/60",
		"KG 2 'Holzhammer'/Niederschlesien/60",
		"Kampfgeschwader 152/Westfalen/80",
		"Kampfgeschwader 155/Westfalen/80",
		"Kampfgeschwader 154/Oberbayern/80",
		"Jagdgeschwader 232/Königsberg/40",
		"Jagdgeschwader 132/Brandenburg/80",
		"Jagdgeschwader 134/Westfalen/60",
		"JG 1 'Oesau'/Westfalen/60",
		"Jagdgeschwader 136/Oberbayern/40",
		"Sturzkampfgeschwader 162/Niederschlesien/80",
		"Kampfgeschwader 153/Brandenburg/40",
		"LG 'Greifswald'/Brandenburg/80",
	}
	if got := strings.Join(rows, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("rows =\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
	pills := []string{}
	for _, pill := range out["pills"].([]map[string]any) {
		pills = append(pills, fmt.Sprintf("%v %v", pill["role"], pill["count"]))
	}
	if got := strings.Join(pills, "|"); got != "cas 80|fighter 280|naval_bomber 72|tactical_bomber 480|air_transport 80" {
		t.Errorf("pills = %s", got)
	}
	if texts := out["texts"].(map[string]any); texts["wings"] != "15 Air Wings" || texts["theatres"] != "Active in 0 Strategic area(s)" {
		t.Errorf("texts = %v", texts)
	}
}
