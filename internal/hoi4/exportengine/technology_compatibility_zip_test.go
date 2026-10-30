package exportengine

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/techsurface"
)

func TestTechnologyAllPlayersCompatibilitySurvivesFinalZIP(t *testing.T) {
	const baseline = `guiTypes = {
	containerWindowType = {
		name = "countrytechtreeview"
		containerWindowType = { name = "air_techs_folder" }
		containerWindowType = { name = "bba_air_techs_folder" }
	}
}
`
	const profileID = "test_export_all_players_air"
	restore := techsurface.RegisterProfileForTest(techsurface.Profile{
		ID:             profileID,
		BaselineSHA256: techsurface.Hash([]byte(baseline)),
		Folders: map[string]techsurface.FolderProfile{
			"air_techs_folder": {
				ID: "air_techs_folder", Surface: "air", TechLine: "p0_compat_air",
				GridboxFormat: "UP", SlotWidth: 60, SlotHeight: 70,
				Slots:    map[string]techsurface.Slot{"base": {ID: "base", X: 10, Y: 20}},
				Timeline: []techsurface.TimelineCell{{StartYear: 1933, X: 0, Y: 2}},
			},
			"bba_air_techs_folder": {
				ID: "bba_air_techs_folder", Surface: "air", TechLine: "p0_compat_air",
				GridboxFormat: "UP", SlotWidth: 70, SlotHeight: 70,
				Slots:    map[string]techsurface.Slot{"bba": {ID: "bba", X: 30, Y: 40}},
				Timeline: []techsurface.TimelineCell{{StartYear: 1933, X: 0, Y: 2}},
			},
		},
	})
	defer restore()

	manifest := techsurface.Manifest{
		SchemaVersion:     techsurface.SchemaVersion,
		Profile:           profileID,
		BaselineSHA256:    techsurface.Hash([]byte(baseline)),
		CompatibilityMode: "all_players",
		Branches: []techsurface.Branch{{
			ID: "air", Surface: "air", Folder: "air_techs_folder", Slot: "base",
			RootTechnologyID: "TST_compat_air", TechnologyIDs: []string{"TST_compat_air"},
			Placements: []techsurface.BranchPlacement{
				{Folder: "air_techs_folder", Slot: "base"},
				{Folder: "bba_air_techs_folder", Slot: "bba"},
			},
		}},
	}
	projectRaw, err := json.Marshal(map[string]any{
		"id": 601, "name": "P0 Technology Compatibility", "country_tag": "TST",
		"version": "1.0", "supported_version": "1.19.*", "tags": []any{"Utilities"},
		"export_languages": []any{"en"}, "tech_surface_manifest": manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	technologyRaw, err := json.Marshal(map[string]any{
		"id": 1, "tech_id": "TST_compat_air", "tech_line": "p0_compat_air",
		"source": "created", "folder": "air_techs_folder", "start_year": 1933,
		"folder_placements": []any{
			map[string]any{"folder": "air_techs_folder", "x": 0, "y": 2},
			map[string]any{"folder": "bba_air_techs_folder", "x": 0, "y": 2},
		},
		"research_cost": 0.1, "ai_will_do": "factor = 0", "categories": []any{},
		"modifiers": map[string]any{"__modifier": "research_speed_factor = 0.001"},
		"name_en":   "P0 Compatible Air", "desc_en": "Visible with or without the aircraft designer DLC.",
		"path": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 601, Project: projectRaw,
		Tables: map[string][]json.RawMessage{
			"technologies":       {technologyRaw},
			"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"p0-compat-gui.zip"}`)},
		},
	}
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, maxBytes int64) ([]byte, error) {
		return p0aGUISnapshotZIP(t, []byte(baseline)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "p0-technology-all-players.zip")
	if _, err := Build(destination, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget()); err != nil {
		t.Fatal(err)
	}
	entries := p0aZIPEntries(t, destination)
	root := artifactRoot(artifact)
	technology := string(entries[root+"/common/technologies/TST_p0_compat_air.txt"])
	for _, folder := range []string{"air_techs_folder", "bba_air_techs_folder"} {
		if !strings.Contains(technology, "name = "+folder) {
			t.Fatalf("final ZIP technology source missing %s placement:\n%s", folder, technology)
		}
	}
	gui := entries[root+"/"+techsurface.BaselinePath]
	if count := bytes.Count(gui, []byte(`name = "TST_compat_air_tree"`)); count != 2 {
		t.Fatalf("final ZIP compatible gridbox count=%d, want 2:\n%s", count, gui)
	}
}
