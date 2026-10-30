package emitmisc

import (
	"strings"
	"testing"
)

func TestGenerateTechLineScriptPreservesOrderedFolderPlacements(t *testing.T) {
	tech := NewNS().
		Set("tech_id", "lossless_tech").
		Set("research_cost", 1.0).
		Set("start_year", 1936).
		Set("folder", "infantry_folder").
		Set("position_x", 99.0).
		Set("position_y", 99.0).
		Set("folder_placements", []any{
			NewOD().Set("folder", "infantry_folder").Set("x", 1.5).Set("y", 3.25),
			NewOD().Set("folder", "light_air_techs_folder").Set("x", -2.75).Set("y", 4.5),
		})

	script := GenerateTechLineScript([]*NS{tech}, "infantry", nil, nil)
	first := "name = infantry_folder\n\t\t\tposition = { x = 1.5 y = 3.25 }"
	second := "name = light_air_techs_folder\n\t\t\tposition = { x = -2.75 y = 4.5 }"
	if !strings.Contains(script, first) || !strings.Contains(script, second) {
		t.Fatalf("folder placements were not emitted losslessly:\n%s", script)
	}
	if strings.Index(script, first) > strings.Index(script, second) {
		t.Fatalf("folder placements changed source order:\n%s", script)
	}
	if strings.Contains(script, "x = 99") {
		t.Fatalf("legacy placement leaked instead of ordered placements:\n%s", script)
	}
}
