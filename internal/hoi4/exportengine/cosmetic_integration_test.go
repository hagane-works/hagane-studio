package exportengine

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/cosmetic"
)

func cosmeticZIPSnapshot() Snapshot {
	return Snapshot{Schema: SnapshotSchema, ProjectID: 4242,
		Project: json.RawMessage(`{"id":4242,"name":"CosmeticZip","country_tag":"ROM","export_languages":["en","zh"]}`),
		Tables:  map[string][]json.RawMessage{}}
}

func cosmeticZIPRow(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func cosmeticFinalZIP(t *testing.T, snapshot Snapshot, includeImported bool) map[string][]byte {
	t.Helper()
	fingerprint, err := SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Fingerprint = fingerprint
	artifact, err := BuildNativeShadow(snapshot, nil, includeImported)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "cosmetic-final.zip")
	if _, err := BuildWithMetadata(filename, fingerprint, artifact.Entries, DefaultArtifactBudget(), artifact.Timeline); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	files := map[string][]byte{}
	for _, entry := range archive.File {
		if _, exists := files[entry.Name]; exists {
			t.Fatalf("duplicate final ZIP entry %s", entry.Name)
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		files[entry.Name] = body
	}
	if len(files) == 0 {
		t.Fatal("empty final ZIP")
	}
	return files
}

var cosmeticZIPLocLine = regexp.MustCompile(`^[ \t]+([A-Za-z0-9_.-]+):[0-9]*[ \t]+"(.*)"[ \t]*$`)

func cosmeticZIPNames(t *testing.T, files map[string][]byte) map[string]map[string][]string {
	t.Helper()
	names := map[string]map[string][]string{}
	for path, body := range files {
		if !strings.Contains(path, "/localisation/") || !strings.HasSuffix(path, ".yml") {
			continue
		}
		if !bytes.HasPrefix(body, []byte("\ufeff")) {
			t.Fatalf("localisation has no BOM in final ZIP: %s", path)
		}
		locale := ""
		for _, line := range strings.Split(strings.TrimPrefix(string(body), "\ufeff"), "\n") {
			line = strings.TrimSuffix(line, "\r")
			if strings.HasPrefix(line, "l_") && strings.HasSuffix(line, ":") {
				locale = strings.TrimSuffix(line, ":")
				continue
			}
			match := cosmeticZIPLocLine.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			if locale == "" {
				t.Fatalf("entry precedes locale header in %s", path)
			}
			if names[locale] == nil {
				names[locale] = map[string][]string{}
			}
			names[locale][match[1]] = append(names[locale][match[1]], match[2])
		}
	}
	return names
}

func cosmeticZIPRequireName(t *testing.T, names map[string]map[string][]string, locale, tag, suffix, want string) {
	t.Helper()
	key := tag
	if suffix != "" {
		key += "_" + suffix
	}
	values := names[locale][key]
	if len(values) != 1 || values[0] != want {
		t.Fatalf("final ZIP %s/%s = %#v, want exactly one %q", locale, key, values, want)
	}
}

var cosmeticZIPForms = []string{"", "ADJ", "DEF", "democratic", "democratic_ADJ", "democratic_DEF", "fascism", "fascism_ADJ", "fascism_DEF", "neutrality", "neutrality_ADJ", "neutrality_DEF", "communism", "communism_ADJ", "communism_DEF"}

func TestCosmeticFinalZIPNamesAllSourcesAndLanguages(t *testing.T) {
	snapshot := cosmeticZIPSnapshot()
	snapshot.Tables["history_files"] = []json.RawMessage{json.RawMessage(`{"id":1,"country_tag":"ROM","set_cosmetic_tag":"ROM_shared","source":"created"}`)}
	snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{json.RawMessage(`{"id":11,"tag":"ROM_shared","source":"imported"}`)}
	for _, row := range []map[string]any{
		{"cosmetic_tag_id": 11, "locale": "l_english", "suffix": "ADJ", "value": "Royal", "source": "created"},
		{"cosmetic_tag_id": 11, "locale": "l_english", "suffix": "fascism", "value": "Imperial Kingdom", "source": "created"},
		{"cosmetic_tag_id": 11, "locale": "l_simp_chinese", "suffix": "", "value": ` 新王国 "帝国" \ $TOKEN$ `, "source": "created"},
		{"cosmetic_tag_id": 11, "locale": "l_simp_chinese", "suffix": "ADJ", "value": "新王", "source": "created"},
	} {
		snapshot.Tables["project_cosmetic_tag_names"] = append(snapshot.Tables["project_cosmetic_tag_names"], cosmeticZIPRow(t, row))
	}
	for _, row := range []map[string]any{
		{"id": 1, "locale": "l_english", "key": "ROM_shared", "value": "Imported Kingdom", "version": 0, "source_file": "localisation/english/original_l_english.yml", "source": "imported"},
		{"id": 2, "locale": "l_english", "key": "ROM_shared_DEF", "value": "", "source_file": "localisation/english/original_l_english.yml", "source": "imported"},
		{"id": 3, "locale": "l_english", "key": "ROM_shared_fascism_ADJ", "value": "Old imported adjective", "source_file": "localisation/english/original_l_english.yml", "source": "imported"},
		{"id": 4, "locale": "l_simp_chinese", "key": "ROM_shared", "value": "旧名字", "source_file": "localisation/simp_chinese/original_l_simp_chinese.yml", "source": "imported"},
	} {
		snapshot.Tables["project_loc_entries"] = append(snapshot.Tables["project_loc_entries"], cosmeticZIPRow(t, row))
	}
	names := cosmeticZIPNames(t, cosmeticFinalZIP(t, snapshot, true))
	english := map[string]string{"": "Imported Kingdom", "DEF": "", "ADJ": "Royal", "fascism": "Imperial Kingdom", "fascism_DEF": "Imperial Kingdom"}
	for _, suffix := range cosmeticZIPForms {
		want, exists := english[suffix]
		if !exists {
			if strings.HasSuffix(suffix, "_ADJ") {
				want = "Royal"
			} else {
				want = "ROM Shared"
			}
		}
		cosmeticZIPRequireName(t, names, "l_english", "ROM_shared", suffix, want)
		wantChinese := ` 新王国 \"帝国\" \\ $TOKEN$ `
		if suffix == "ADJ" || strings.HasSuffix(suffix, "_ADJ") {
			wantChinese = "新王"
		}
		cosmeticZIPRequireName(t, names, "l_simp_chinese", "ROM_shared", suffix, wantChinese)
	}
}

func TestCosmeticFinalZIPRegistryOnlyAndFieldProvenance(t *testing.T) {
	snapshot := cosmeticZIPSnapshot()
	snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{
		json.RawMessage(`{"id":1,"tag":"authored_over_import","source":"imported","imported_color":[91,92,93],"color_ui":[7,8,9]}`),
		json.RawMessage(`{"id":2,"tag":"pure_import","source":"imported"}`),
		json.RawMessage(`{"id":3,"tag":"created_unused","source":"created"}`),
	}
	snapshot.Tables["project_cosmetic_tag_names"] = []json.RawMessage{
		json.RawMessage(`{"cosmetic_tag_id":1,"locale":"l_english","suffix":"","value":"Authored survives","source":"created"}`),
		json.RawMessage(`{"cosmetic_tag_id":1,"locale":"l_simp_chinese","suffix":"","value":"导入应排除","source":"imported"}`),
	}
	snapshot.Tables["project_loc_entries"] = []json.RawMessage{json.RawMessage(`{"id":1,"locale":"l_simp_chinese","key":"authored_over_import","value":"导入应排除","source_file":"localisation/simp_chinese/original_l_simp_chinese.yml","source":"imported"}`)}
	for _, includeImported := range []bool{false, true} {
		files := cosmeticFinalZIP(t, snapshot, includeImported)
		names := cosmeticZIPNames(t, files)
		for _, suffix := range cosmeticZIPForms {
			cosmeticZIPRequireName(t, names, "l_english", "authored_over_import", suffix, "Authored survives")
			cosmeticZIPRequireName(t, names, "l_english", "created_unused", suffix, "Created Unused")
		}
		wantChinese := "Authored Over Import"
		if includeImported {
			wantChinese = "导入应排除"
		}
		cosmeticZIPRequireName(t, names, "l_simp_chinese", "authored_over_import", "", wantChinese)
		if !includeImported {
			for _, body := range files {
				if bytes.Contains(body, []byte("pure_import")) || bytes.Contains(body, []byte("91 92 93")) {
					t.Fatal("imported-only object or imported color leaked into created-only ZIP")
				}
			}
		} else {
			cosmeticZIPRequireName(t, names, "l_english", "pure_import", "", "Pure Import")
		}
		if !strings.Contains(string(files["CosmeticZip/common/countries/cosmetic.txt"]), "color_ui = rgb { 7 8 9 }") {
			t.Fatal("authored UI color on imported parent was filtered out")
		}
	}
}

func TestCosmeticFinalZIPStableOwnerAndIndependentColors(t *testing.T) {
	for _, authored := range []string{"map", "ui", "neither"} {
		for _, includeImported := range []bool{false, true} {
			snapshot := cosmeticZIPSnapshot()
			snapshot.Tables["countries"] = []json.RawMessage{json.RawMessage(`{"id":1,"tag":"ROM","color":[10,20,30],"color_ui":[11,21,31]}`), json.RawMessage(`{"id":2,"tag":"GER","color":[70,80,90],"color_ui":[71,81,91]}`)}

			snapshot.Tables["history_files"] = []json.RawMessage{json.RawMessage(`{"id":20,"country_tag":"GER","set_cosmetic_tag":"shared_tag","source":"created"}`), json.RawMessage(`{"id":10,"country_tag":"ROM","set_cosmetic_tag":"shared_tag","source":"imported"}`)}
			parent := map[string]any{"id": 1, "tag": "shared_tag", "source": "imported", "imported_color": []int{40, 50, 60}, "imported_color_ui": []int{41, 51, 61}}
			if authored == "map" {
				parent["color"] = []int{1, 2, 3}
			}
			if authored == "ui" {
				parent["color_ui"] = []int{4, 5, 6}
			}
			snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{cosmeticZIPRow(t, parent)}
			snapshot.Tables["project_cosmetic_tag_names"] = []json.RawMessage{json.RawMessage(`{"cosmetic_tag_id":1,"locale":"l_english","suffix":"","value":"Shared authored name","source":"created"}`)}
			files := cosmeticFinalZIP(t, snapshot, includeImported)
			mapColor, uiColor := "10 20 30", "11 21 31"
			if includeImported {
				mapColor, uiColor = "40 50 60", "41 51 61"
			}
			if authored == "map" {
				mapColor = "1 2 3"
			}
			if authored == "ui" {
				uiColor = "4 5 6"
			}
			want := "shared_tag = {\n\tcolor = rgb { " + mapColor + " }\n\tcolor_ui = rgb { " + uiColor + " }\n}"
			if body := string(files["CosmeticZip/common/countries/cosmetic.txt"]); !strings.Contains(body, want) {
				t.Fatalf("authored=%s include=%v expected %q, final bytes=%q", authored, includeImported, want, body)
			}
		}
	}
}

func TestCosmeticFinalZIPRawSamePathWins(t *testing.T) {
	snapshot := cosmeticZIPSnapshot()
	snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{json.RawMessage(`{"id":1,"tag":"raw_target","source":"created","color":[1,2,3]}`)}
	locPath := "localisation/english/replace/zz_CosmeticZip_cosmetic_l_english.yml"
	colorPath := "common/countries/cosmetic.txt"
	locBytes := "\ufeffl_english:\n raw_target: \"Raw author final name\"\n"
	colorBytes := "raw_target = { color = rgb { 201 202 203 } color_ui = rgb { 101 102 103 } }\n"
	for _, raw := range []struct {
		id         int
		path, body string
	}{{1, locPath, locBytes}, {2, colorPath, colorBytes}} {
		snapshot.Tables["project_raw_files"] = append(snapshot.Tables["project_raw_files"], cosmeticZIPRow(t, map[string]any{"id": raw.id, "dest_path": raw.path, "content": raw.body, "raw_kind": "verbatim_override", "source": "created"}))
	}
	files := cosmeticFinalZIP(t, snapshot, false)
	if got := string(files["CosmeticZip/"+locPath]); got != locBytes {
		t.Fatalf("raw loc replacement not byte exact: %q", got)
	}
	if got := string(files["CosmeticZip/"+colorPath]); got != colorBytes {
		t.Fatalf("raw colors replacement not byte exact: %q", got)
	}
	log := string(files["CosmeticZip/_RAW_OVERRIDE_FILES.log"])
	if !strings.Contains(log, locPath) || !strings.Contains(log, colorPath) {
		t.Fatal("final raw wins are missing the explicit override registry")
	}
	cosmeticZIPRequireName(t, cosmeticZIPNames(t, files), "l_english", "raw_target", "", "Raw author final name")
}

func TestCosmeticFinalZIPEngineColorFilename(t *testing.T) {
	snapshot := cosmeticZIPSnapshot()
	snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{json.RawMessage(`{"id":1,"tag":"ROM_greater_kingdom","source":"created","color":[156,72,220],"color_ui":[240,180,45]}`)}
	files := cosmeticFinalZIP(t, snapshot, true)

	body, ok := files["CosmeticZip/common/countries/cosmetic.txt"]
	if !ok {
		t.Fatal("engine cosmetic color file missing: common/countries/cosmetic.txt")
	}
	base, err := cosmetic.VanillaColorFile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(body, base) {
		t.Fatal("new project tag erased untouched vanilla cosmetic colors")
	}
	for _, want := range []string{"ROM_greater_kingdom = {", "color = rgb { 156 72 220 }", "color_ui = rgb { 240 180 45 }"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("engine color file missing %q", want)
		}
	}
	if _, exists := files["CosmeticZip/common/countries/zz_CosmeticZip_cosmetic.txt"]; exists {
		t.Fatal("unloaded cosmetic sibling must not be emitted")
	}
}

func TestCosmeticFinalZIPIneligiblePreservedNamesNeverMakeHoles(t *testing.T) {
	for _, input := range []struct {
		source, path string
		include      bool
	}{{"imported", "localisation/english/old_l_english.yml", false}, {"imported", "../../escape.yml", true}} {
		snapshot := cosmeticZIPSnapshot()
		snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{json.RawMessage(`{"id":1,"tag":"fallback_tag","source":"created"}`)}
		snapshot.Tables["project_cosmetic_tag_names"] = []json.RawMessage{json.RawMessage(`{"cosmetic_tag_id":1,"locale":"l_english","suffix":"","value":"ineligible projection","source":"imported"}`)}
		snapshot.Tables["project_loc_entries"] = []json.RawMessage{cosmeticZIPRow(t, map[string]any{"id": 1, "locale": "l_english", "key": "fallback_tag", "value": "ineligible original", "source": input.source, "source_file": input.path})}
		names := cosmeticZIPNames(t, cosmeticFinalZIP(t, snapshot, input.include))
		for _, suffix := range cosmeticZIPForms {
			cosmeticZIPRequireName(t, names, "l_english", "fallback_tag", suffix, "Fallback Tag")
		}
	}
}
