package worldmap

import (
	"testing"

	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
)

func TestMapCheckNamesWhatTheGameQuitsOn(t *testing.T) {
	_, base, _ := provinceEditWorld(t)
	regions := map[int][]int{10: {1}, 20: {2, 3, 4}, 30: {5}}
	payload := func(stateOf map[int]int) guirender.WorldMapPayload {
		p := guirender.WorldMapPayload{States: [][]any{{1, "West", "AAA"}, {2, "East", "BBB"}, {3, "Lake", "AAA"}}}
		for id := 1; id <= 5; id++ {
			p.Provinces = append(p.Provinces, []any{id, id, 0, 0, "", 0, stateOf[id], 0})
		}
		return p
	}
	whole := mapCheck(base, regions, payload(map[int]int{1: 1, 5: 2}))
	if whole["ok"] != true || len(whole["states_without_land"].([]map[string]any)) != 0 || whole["fix"] != nil {
		t.Fatalf("a whole map: %+v", whole)
	}
	broken := mapCheck(base, regions, payload(map[int]int{1: 1, 5: 1, 3: 3}))
	if broken["ok"] != false || broken["fix"] == nil {
		t.Fatalf("broken map: %+v", broken)
	}
	landless := broken["states_without_land"].([]map[string]any)
	if len(landless) != 1 || landless[0]["state"] != 3 || landless[0]["name"] != "Lake" {
		t.Fatalf("states without land: %+v", landless)
	}
	across := broken["states_across_regions"].([]map[string]any)
	if len(across) != 1 || across[0]["state"] != 1 {
		t.Fatalf("states across regions: %+v", across)
	}
	codes := map[string]bool{}
	for _, f := range broken["findings"].([]mapgen.Finding) {
		codes[f.Code] = true
	}
	if !codes["state_no_land"] || !codes["state_two_regions"] {
		t.Fatalf("findings %+v", broken["findings"])
	}
}
