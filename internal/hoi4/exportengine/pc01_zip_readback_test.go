package exportengine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/techsurface"
)

func TestPC01FinalZIPReadbackChecksCriticalContent(t *testing.T) {
	zipPath := buildPC01ZIPReadbackFixture(t, "")
	entries, err := readPC01ZIPEntries(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePC01ZIPCriticalContent(entries); err != nil {
		t.Fatalf("PC-01 final ZIP content gate rejected the valid artifact: %v", err)
	}
}

func TestPC01FinalZIPReadbackRejectsMissingCriticalContent(t *testing.T) {
	const omitted = "PC01/common/ideologies/00_ideologies.txt"
	zipPath := buildPC01ZIPReadbackFixture(t, omitted)
	entries, err := readPC01ZIPEntries(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	err = validatePC01ZIPCriticalContent(entries)
	if err == nil {
		t.Fatalf("PC-01 final ZIP content gate accepted an artifact missing %s", omitted)
	}
	if !strings.Contains(err.Error(), omitted) {
		t.Fatalf("PC-01 negative control failed for the wrong reason: %v", err)
	}
}

func buildPC01ZIPReadbackFixture(t *testing.T, omitPath string) string {
	t.Helper()
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
	restore := techsurface.RegisterProfileForTest(p0aTechProfile(candidate, baseline))
	defer restore()
	archive := p0aGUISnapshotZIP(t, baseline)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, _ int64) ([]byte, error) {
		if key != "p0a-tech-gui-source.zip" {
			return nil, fmt.Errorf("unexpected blob key %q", key)
		}
		return archive, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	entries := make([]Entry, 0, len(artifact.Entries))
	for _, entry := range artifact.Entries {
		if entry.Path == omitPath {
			continue
		}
		entries = append(entries, entry)
	}
	if omitPath != "" && len(entries) == len(artifact.Entries) {
		t.Fatalf("negative-control path %s was not emitted", omitPath)
	}
	zipPath := filepath.Join(t.TempDir(), "pc01-final-readback.zip")
	if _, err := Build(zipPath, snapshot.Fingerprint, entries, DefaultArtifactBudget()); err != nil {
		t.Fatal(err)
	}
	return zipPath
}

func readPC01ZIPEntries(path string) (map[string][]byte, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open PC-01 final ZIP: %w", err)
	}
	defer reader.Close()
	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		if _, duplicate := entries[file.Name]; duplicate {
			return nil, fmt.Errorf("PC-01 final ZIP contains duplicate path %s", file.Name)
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open PC-01 ZIP entry %s: %w", file.Name, err)
		}
		body, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read PC-01 ZIP entry %s: %w", file.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close PC-01 ZIP entry %s: %w", file.Name, closeErr)
		}
		entries[file.Name] = body
	}
	return entries, nil
}

func validatePC01ZIPCriticalContent(entries map[string][]byte) error {
	required := map[string][]string{
		"PC01/interface/countrytechtreeview.gui": {
			`name = "p0a_base_infantry_1_tree"`,
		},
		"PC01/common/technologies/P01_p0a_base_infantry.txt": {
			"p0a_base_infantry_1 = {", "name = infantry_folder", "enable_equipments = {", "P01_pc01_rifle",
		},
		"PC01/common/units/equipment/zz_P01_equipment.txt": {
			"P01_pc01_rifle",
		},
		"PC01/common/ideologies/00_ideologies.txt": {
			"pc01_civic_coalition = {", "pc01_civic_reformism = {",
		},
		"PC01/history/countries/P01 - zPC01.txt": {
			"ruling_party = pc01_civic_coalition", "pc01_civic_coalition = 60",
		},
		"PC01/common/characters/P01_P01.txt": {
			"P01_pc01_civic_leader", "ideology = pc01_civic_reformism",
		},
		"PC01/localisation/english/replace/P01_technology_l_english.yml": {
			"p0a_base_infantry_1", "P0 Base Infantry I", "p0a_base_infantry_1_desc",
		},
		"PC01/localisation/english/replace/P01_equipment_l_english.yml": {
			`P01_pc01_rifle_desc: "Unlocked by the PC-01 technology."`,
		},
		"PC01/localisation/english/replace/P01_characters_l_english.yml": {
			"P01_pc01_civic_leader", "PC-01 Civic Leader",
		},
		"PC01.mod": {
			`replace_path="common/ideologies"`,
		},
	}
	paths := make([]string, 0, len(required))
	for path := range required {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := requirePC01ZIPContent(entries, path, required[path]...); err != nil {
			return err
		}
	}
	return requirePC01ZIPContentBySuffix(entries, "_ideology_l_english.yml", "pc01_civic_coalition", "PC-01 Civic Coalition", "pc01_civic_reformism", "PC-01 Civic Reformism")
}

func requirePC01ZIPContent(entries map[string][]byte, path string, needles ...string) error {
	body, ok := entries[path]
	if !ok {
		return fmt.Errorf("PC-01 final ZIP missing critical file %s", path)
	}
	if strings.Contains(path, "/localisation/") && !bytes.HasPrefix(body, []byte{0xef, 0xbb, 0xbf}) {
		return fmt.Errorf("PC-01 final ZIP localisation %s is not UTF-8 BOM", path)
	}
	for _, needle := range needles {
		if !bytes.Contains(body, []byte(needle)) {
			return fmt.Errorf("PC-01 final ZIP file %s missing critical content %q", path, needle)
		}
	}
	return nil
}

func requirePC01ZIPContentBySuffix(entries map[string][]byte, suffix string, needles ...string) error {
	paths := make([]string, 0, 1)
	for path := range entries {
		if strings.HasSuffix(path, suffix) {
			paths = append(paths, path)
		}
	}
	if len(paths) != 1 {
		sort.Strings(paths)
		return fmt.Errorf("PC-01 final ZIP expected one critical file ending in %s, found %v", suffix, paths)
	}
	return requirePC01ZIPContent(entries, paths[0], needles...)
}
