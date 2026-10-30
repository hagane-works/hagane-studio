package trees

import "testing"

func TestImportTechnologiesPreservesFractionalFolderPosition(t *testing.T) {
	rows, _, _, err := ImportTechnologies(`technologies = {
	jet_engines = {
		folder = { name = air_techs_folder position = { x = -8.5 y = 3.25 } }
		folder = { name = light_air_techs_folder position = { x = 2 y = 7 } }
	}
}`, "air_techs", "air_techs_folder")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if got, ok := rows[0]["position_x"].(float64); !ok || got != -8.5 {
		t.Fatalf("position_x = %#v (%T)", rows[0]["position_x"], rows[0]["position_x"])
	}
	if got, ok := rows[0]["position_y"].(float64); !ok || got != 3.25 {
		t.Fatalf("position_y = %#v (%T)", rows[0]["position_y"], rows[0]["position_y"])
	}
	placements, ok := rows[0]["folder_placements"].([]map[string]any)
	if !ok || len(placements) != 2 || placements[1]["folder"] != "light_air_techs_folder" {
		t.Fatalf("folder_placements = %#v", rows[0]["folder_placements"])
	}
}

func TestImportTechnologiesEmptyFolderUsesFileFallback(t *testing.T) {
	rows, _, _, err := ImportTechnologies(`technologies = { hidden = { folder = {} } }`, "infantry", "infantry_folder")
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if rows[0]["folder"] != "infantry_folder" {
		t.Fatalf("folder = %#v", rows[0]["folder"])
	}
}
