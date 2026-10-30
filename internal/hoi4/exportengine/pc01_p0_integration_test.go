package exportengine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/techsurface"
)

func TestPC01TechnologyEquipmentAndProjectPoliticsShareOneArtifact(t *testing.T) {
	baseline := []byte(`guiTypes = {
	containerWindowType = {
		name = "countrytechtreeview"
		containerWindowType = {
			name = "infantry_folder"
			gridboxtype = {
				name = "infantry_weapons_tree"
				position = { x = 140 y = 325 }
				slotsize = { width = 70 height = 70 }
				format = "LEFT"
			}
		}
	}
}
`)
	candidate := p0aBaseCandidate()
	candidate.ModName = "PC01"
	candidate.Branches = candidate.Branches[:1]
	snapshot := p0aTechSnapshot(t, candidate, baseline)

	var project map[string]any
	if err := json.Unmarshal(snapshot.Project, &project); err != nil {
		t.Fatal(err)
	}
	project["name"] = "PC01"
	project["country_tag"] = "P01"
	project["ideology_registry"] = map[string]any{
		"groups": []string{"pc01_civic_coalition"},
		"root_definitions": []map[string]any{{
			"id": "pc01_civic_coalition", "body": "types = { }\ncolor = { 45 95 135 }",
			"name": map[string]string{"en": "PC-01 Civic Coalition"},
		}},
		"sub_ideologies": []map[string]any{{
			"id": "pc01_civic_reformism", "group": "pc01_civic_coalition",
			"name": map[string]string{"en": "PC-01 Civic Reformism"}, "random_disabled": false,
		}},
	}
	snapshot.Project = mustJSON(t, project)

	var rootTechnology map[string]any
	if err := json.Unmarshal(snapshot.Tables["technologies"][0], &rootTechnology); err != nil {
		t.Fatal(err)
	}
	rootTechnology["enable_equipments"] = []string{"P01_pc01_rifle"}
	snapshot.Tables["technologies"][0] = mustJSON(t, rootTechnology)
	snapshot.Tables["equipments"] = []json.RawMessage{json.RawMessage(`{
		"id":101,"equipment_id":"P01_pc01_rifle","name_en":"PC-01 Rifle",
		"desc_en":"Unlocked by the PC-01 technology.","source":"created"
	}`)}
	snapshot.Tables["countries"] = []json.RawMessage{json.RawMessage(`{
		"id":102,"tag":"P01","name_en":"PC-01 Country","color":[45,95,135],
		"graphical_culture":"western_european_gfx","graphical_culture_2d":"western_european_2d",
		"ideology":"pc01_civic_coalition","source":"created"
	}`)}
	snapshot.Tables["history_files"] = []json.RawMessage{json.RawMessage(`{
		"id":103,"country_tag":"P01","capital":1,"set_politics_ruling_party":"pc01_civic_coalition",
		"set_popularities":{"pc01_civic_coalition":60,"democratic":40},"source":"created"
	}`)}
	snapshot.Tables["characters"] = []json.RawMessage{json.RawMessage(`{
		"id":104,"tag":"P01_pc01_civic_leader","country_tag":"P01","name_en":"PC-01 Civic Leader",
		"role_type":"country_leader","ideology":"pc01_civic_reformism","traits":[],
		"advisor_traits":[],"raw_extras":{},"source":"created","replaces_vanilla_id":false
	}`)}
	var err error
	snapshot.Fingerprint, err = SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	profile := p0aTechProfile(candidate, baseline)
	restore := techsurface.RegisterProfileForTest(profile)
	defer restore()
	archive := p0aGUISnapshotZIP(t, baseline)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, _ int64) ([]byte, error) {
		if key != "p0a-tech-gui-source.zip" {
			t.Fatalf("unexpected blob key %q", key)
		}
		return archive, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string][]byte{}
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry.Bytes
	}
	assertPC01Contains(t, byPath, "PC01/interface/countrytechtreeview.gui", "p0a_base_infantry_1_tree")
	assertPC01Contains(t, byPath, "PC01/common/technologies/P01_p0a_base_infantry.txt", "enable_equipments = {")
	assertPC01Contains(t, byPath, "PC01/common/technologies/P01_p0a_base_infantry.txt", "P01_pc01_rifle")
	assertPC01Contains(t, byPath, "PC01/common/units/equipment/zz_P01_equipment.txt", "P01_pc01_rifle")
	assertPC01Contains(t, byPath, "PC01/localisation/english/replace/P01_equipment_l_english.yml", `P01_pc01_rifle_desc: "Unlocked by the PC-01 technology."`)
	assertPC01Contains(t, byPath, "PC01/common/ideologies/00_ideologies.txt", "pc01_civic_coalition")
	assertPC01Contains(t, byPath, "PC01/common/ideologies/00_ideologies.txt", "pc01_civic_reformism")
	assertPC01Contains(t, byPath, "PC01/history/countries/P01 - zPC01.txt", "ruling_party = pc01_civic_coalition")
	assertPC01Contains(t, byPath, "PC01/history/countries/P01 - zPC01.txt", "pc01_civic_coalition = 60")
	assertPC01Contains(t, byPath, "PC01/common/characters/P01_P01.txt", "ideology = pc01_civic_reformism")
	assertPC01Contains(t, byPath, "PC01.mod", `replace_path="common/ideologies"`)

	firstPath := filepath.Join(t.TempDir(), "pc01-first.zip")
	secondPath := filepath.Join(t.TempDir(), "pc01-second.zip")
	first, err := Build(firstPath, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(secondPath, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatal(err)
	}
	if first.ZIPSHA256 != second.ZIPSHA256 || first.ZIPBytes != second.ZIPBytes {
		t.Fatalf("PC-01 ZIP is not deterministic: first=%+v second=%+v", first, second)
	}
	reader, err := zip.OpenReader(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != len(artifact.Entries)+1 {
		t.Fatalf("PC-01 ZIP entries=%d want=%d", len(reader.File), len(artifact.Entries)+1)
	}
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatal(err)
	}
}

func assertPC01Contains(t *testing.T, entries map[string][]byte, path, needle string) {
	t.Helper()
	body, ok := entries[path]
	if !ok {
		paths := make([]string, 0, len(entries))
		for candidate := range entries {
			paths = append(paths, candidate)
		}
		t.Fatalf("PC-01 artifact missing %s; paths=%s", path, strings.Join(paths, ", "))
	}
	if !bytes.Contains(body, []byte(needle)) {
		t.Fatalf("PC-01 artifact %s missing %q:\n%s", path, needle, body)
	}
}
