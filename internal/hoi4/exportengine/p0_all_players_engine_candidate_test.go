package exportengine

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/techsurface"
	"haganeworks/internal/hoi4/techsurfaceprofiles"
)

func TestP0FinalAllPlayersTechnologyEquipmentZIP(t *testing.T) {
	baselinePath := os.Getenv("P0A_TECH_GUI_BASELINE")
	if baselinePath == "" {
		t.Skip("set P0A_TECH_GUI_BASELINE to the audited original countrytechtreeview.gui")
	}
	baseline, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read original GUI baseline: %v", err)
	}
	if got, want := techsurface.Hash(baseline), techsurfaceprofiles.CountryTechTreeGUISHA256; got != want {
		t.Fatalf("original GUI baseline hash=%s, want %s", got, want)
	}

	resolved, err := techsurfaceprofiles.Resolve(techsurfaceprofiles.NewEnvironment(
		techsurfaceprofiles.AuditedSourceFingerprint(),
		techsurfaceprofiles.DLCState{ByBloodAlone: true, ManTheGuns: true},
	))
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := techsurface.ProfileByID(resolved.ID)
	if !ok {
		t.Fatalf("production technology profile %s is unavailable", resolved.ID)
	}

	fixtures := p0aBaseCandidate().Branches
	manifest := techsurface.Manifest{
		SchemaVersion: techsurface.SchemaVersion, Profile: profile.ID,
		BaselineSHA256: profile.BaselineSHA256, CompatibilityMode: "all_players",
	}
	rows := make([]json.RawMessage, 0, len(fixtures)*2)
	for branchIndex, fixture := range fixtures {
		surface := fixture.Surface
		if surface == "civilian" {
			surface = techsurfaceprofiles.EntryIndustryEquipment
		}
		folders := []string{fixture.Folder}
		switch fixture.Folder {
		case "air_techs_folder":
			folders = append(folders, "bba_air_techs_folder")
		case "naval_folder":
			folders = append(folders, "mtgnavalfolder")
		}
		placements := make([]techsurface.BranchPlacement, 0, len(folders))
		for _, folderID := range folders {
			folder, exists := profile.Folders[folderID]
			if !exists {
				t.Fatalf("production profile missing compatible folder %s", folderID)
			}
			placements = append(placements, techsurface.BranchPlacement{Folder: folderID, Slot: p0OnlyProfileSlot(t, folder)})
		}
		manifest.Branches = append(manifest.Branches, techsurface.Branch{
			ID:      strings.ReplaceAll(strings.ToLower(fixture.Label), " ", "_") + "_branch",
			Surface: surface, Folder: placements[0].Folder, Slot: placements[0].Slot,
			RootTechnologyID: fixture.RootID,
			TechnologyIDs:    []string{fixture.RootID, fixture.ChildID},
			Placements:       placements,
		})

		for techIndex, technology := range []struct {
			ID, NameEN, NameZH string
		}{
			{fixture.RootID, fixture.RootNameEN, fixture.RootNameZH},
			{fixture.ChildID, fixture.ChildNameEN, fixture.ChildNameZH},
		} {
			primary := profile.Folders[placements[0].Folder]
			primaryCell, exists := primary.TimelineCellAt(techIndex)
			if !exists {
				t.Fatalf("production folder %s missing timeline cell %d", primary.ID, techIndex)
			}
			folderPlacements := make([]map[string]any, 0, len(placements))
			for _, placement := range placements {
				folder := profile.Folders[placement.Folder]
				cell, exists := folder.TimelineCellAt(techIndex)
				if !exists {
					t.Fatalf("compatible folder %s missing timeline cell %d", folder.ID, techIndex)
				}
				folderPlacements = append(folderPlacements, map[string]any{"folder": folder.ID, "x": cell.X, "y": cell.Y})
			}
			row := map[string]any{
				"id": branchIndex*2 + techIndex + 1, "tech_id": technology.ID,
				"tech_line": primary.TechLine, "source": "created", "folder": primary.ID,
				"start_year": primaryCell.StartYear, "folder_placements": folderPlacements,
				"research_cost": 0.1, "ai_will_do": "factor = 0", "categories": []any{},
				"modifiers": map[string]any{"__modifier": "research_speed_factor = 0.001"},
				"name_en":   technology.NameEN, "name_zh": technology.NameZH,
				"desc_en": technology.NameEN + " all-player engine acceptance node.",
				"desc_zh": technology.NameZH + " 全玩家兼容实机验收节点。",
			}
			if techIndex == 0 {
				row["path"] = []any{map[string]any{"leads_to_tech": fixture.ChildID, "research_cost_coeff": 1}}
			} else {
				row["path"] = []any{}
			}
			if branchIndex == 0 && techIndex == 0 {
				row["enable_equipments"] = []string{p0FinalBaseEquipmentID}
			}
			rows = append(rows, mustJSON(t, row))
		}
	}

	project := mustJSON(t, map[string]any{
		"id": 606, "name": "Hagane P0 Final All Players Technology Equipment", "country_tag": "TST",
		"version": "1.0", "supported_version": "1.19.*", "tags": []any{"Utilities"},
		"export_languages": []any{"en", "zh"}, "tech_surface_manifest": manifest,
	})
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 606, Project: project,
		Tables: map[string][]json.RawMessage{
			"technologies": rows,
			"equipments": {mustJSON(t, map[string]any{
				"id": 607, "equipment_id": p0FinalBaseEquipmentID,
				"name_en": "P0 Acceptance Rifle", "name_zh": "P0 验收步枪",
				"desc_en": "Unlocked by P0 Base Infantry I.", "desc_zh": "由 P0 基础步兵 I 解锁。",
				"source": "created", "country_tag": "TST", "year": 1936,
				"archetype": "infantry_equipment", "parent": "infantry_equipment_0",
				"is_archetype": false, "is_buildable": true,
				"soft_attack": 1.0, "defense": 1.0, "build_cost_ic": 0.1,
			})},
			"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"p0-all-players-gui.zip"}`)},
		},
	}
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	archive := p0aGUISnapshotZIP(t, baseline)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, _ int64) ([]byte, error) {
		if key != "p0-all-players-gui.zip" {
			t.Fatalf("unexpected GUI source key %q", key)
		}
		return archive, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	root := artifactRoot(artifact)
	byPath := make(map[string][]byte, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry.Bytes
	}
	for path, folders := range map[string][]string{
		root + "/common/technologies/TST_air.txt":   {"air_techs_folder", "bba_air_techs_folder"},
		root + "/common/technologies/TST_naval.txt": {"naval_folder", "mtgnavalfolder"},
	} {
		for _, folder := range folders {
			if !bytes.Contains(byPath[path], []byte("name = "+folder)) {
				t.Fatalf("all-player technology %s missing %s placement:\n%s", path, folder, byPath[path])
			}
		}
	}
	gui := byPath[root+"/"+techsurface.BaselinePath]
	for _, gridbox := range []string{"p0a_base_air_1_tree", "p0a_base_naval_1_tree"} {
		if count := bytes.Count(gui, []byte(`name = "`+gridbox+`"`)); count != 2 {
			t.Fatalf("all-player GUI gridbox %s count=%d, want 2", gridbox, count)
		}
	}
	p0FinalWriteCandidate(t, "P0_FINAL_ALL_PLAYERS_TECH_EQUIPMENT_ZIP_OUT", "P0_FINAL_ALL_PLAYERS_TECH_EQUIPMENT.zip", snapshot.Fingerprint, artifact.Entries)
}

func p0OnlyProfileSlot(t *testing.T, folder techsurface.FolderProfile) string {
	t.Helper()
	if len(folder.Slots) != 1 {
		t.Fatalf("production folder %s exposes %d P0 slots, want 1", folder.ID, len(folder.Slots))
	}
	for id := range folder.Slots {
		return id
	}
	return ""
}
