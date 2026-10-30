package trees

import "testing"

func TestTechnologiesJSONPreservesOrderedFolderPlacements(t *testing.T) {
	got := TechnologiesJSON(`technologies = {
		lossless_tech = {
			research_cost = 1
			start_year = 1936
			folder = { name = infantry_folder position = { x = 1.5 y = 3.25 } }
			folder = { name = light_air_techs_folder position = { x = -2.75 y = 4.5 } }
		}
	}`, "infantry", "infantry_folder")
	techs, ok := got["techs"].([]any)
	if !ok || len(techs) != 1 {
		t.Fatalf("technologies JSON = %#v", got["techs"])
	}
	tech, ok := techs[0].(map[string]any)
	if !ok {
		t.Fatalf("technology JSON = %#v", techs[0])
	}
	placements, ok := tech["folder_placements"].([]any)
	if !ok || len(placements) != 2 {
		t.Fatalf("folder_placements = %#v", tech["folder_placements"])
	}
	first, firstOK := placements[0].(map[string]any)
	second, secondOK := placements[1].(map[string]any)
	coordinate := func(value any, want string) bool {
		encoded, ok := value.(map[string]any)
		return ok && encoded["t"] == "float" && encoded["v"] == want
	}
	if !firstOK || !secondOK || first["folder"] != "infantry_folder" ||
		!coordinate(first["x"], "1.5") || !coordinate(first["y"], "3.25") ||
		second["folder"] != "light_air_techs_folder" || !coordinate(second["x"], "-2.75") ||
		!coordinate(second["y"], "4.5") {
		t.Fatalf("folder_placements = %#v", placements)
	}
}
