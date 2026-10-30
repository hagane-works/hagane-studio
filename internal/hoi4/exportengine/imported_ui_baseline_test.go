package exportengine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importassets"
)

func importedUIBaselineZIP(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for name, body := range files {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func importedUIBaselineSnapshot() Snapshot {
	return Snapshot{
		Schema:    SnapshotSchema,
		ProjectID: 42,
		Project:   json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"project_obs_assets": {
				json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"private/gui.zip"}`),
			},
		},
	}
}

func importedUIBaselineEntry(artifact NativeArtifact, name string) (Entry, bool) {
	for _, entry := range artifact.Entries {
		if entry.Path == name {
			return entry, true
		}
	}
	return Entry{}, false
}

func TestIsImportedUIBaselinePath(t *testing.T) {
	for _, name := range []string{
		"interface/custom.gui",
		"interface/nested/custom.GFX",
		"gfx/interface/custom.dds",
		"gfx/fonts/custom.fnt",
		"gfx/fonts/custom.ttf",
		"gfx/fonts/custom.otf",
		"gfx/models/custom.asset",
		"common/scripted_guis/custom.txt",
		"common/scripted_guis/nested/custom.txt",
		"common/alerts.txt",
	} {
		if !IsImportedUIBaselinePath(name) {
			t.Errorf("UI baseline rejected %q", name)
		}
	}
	for _, name := range []string{
		importassets.GUISourceGameDataMarker,
		"interface/readme.txt",
		"gfx/interface/readme.txt",
		"common/ideas/mod.txt",
		"history/countries/GER.txt",
		"events/mod.txt",
		"localisation/english/mod_l_english.yml",
		"../interface/escape.gui",
	} {
		if IsImportedUIBaselinePath(name) {
			t.Errorf("UI baseline accepted %q", name)
		}
	}
}

func TestImportedUIBaselineFullOnlyAndRawWins(t *testing.T) {
	files := map[string][]byte{
		"interface/custom.gui":               []byte("baseline gui"),
		"interface/custom.gfx":               []byte("baseline gfx"),
		"gfx/interface/custom.dds":           []byte("baseline dds"),
		"gfx/fonts/custom.fnt":               []byte("baseline fnt"),
		"gfx/fonts/custom.ttf":               []byte("baseline ttf"),
		"common/scripted_guis/custom.txt":    []byte("baseline scripted gui"),
		"common/alerts.txt":                  []byte("baseline alerts"),
		importassets.GUISourceGameDataMarker: []byte("1\n"),
		"common/ideas/not_ui.txt":            []byte("excluded common"),
		"history/countries/GER.txt":          []byte("excluded history"),
		"events/not_ui.txt":                  []byte("excluded event"),
		"localisation/english/not_ui_l.yml":  []byte("excluded loc"),
		"interface/not_ui.txt":               []byte("excluded interface"),
		"gfx/interface/not_ui.txt":           []byte("excluded gfx"),
	}
	archive := importedUIBaselineZIP(t, files)
	snapshot := importedUIBaselineSnapshot()
	resolver := func(string, int64) ([]byte, error) { return archive, nil }

	full, err := buildNativeShadowWithBlobs(
		snapshot, nil, true, resolver, nativeBuildOptions{skipAdvisory: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"interface/custom.gui",
		"interface/custom.gfx",
		"gfx/interface/custom.dds",
		"gfx/fonts/custom.fnt",
		"gfx/fonts/custom.ttf",
		"common/scripted_guis/custom.txt",
		"common/alerts.txt",
	} {
		entry, ok := importedUIBaselineEntry(full, "Demo/"+name)
		if !ok {
			t.Errorf("full export omitted %q", name)
			continue
		}
		if entry.SourceEntity != importedUIBaselineSource || entry.Origin != "imported" {
			t.Errorf("%s provenance = %#v", name, entry)
		}
	}
	for _, name := range []string{
		importassets.GUISourceGameDataMarker,
		"common/ideas/not_ui.txt",
		"history/countries/GER.txt",
		"events/not_ui.txt",
		"localisation/english/not_ui_l.yml",
		"interface/not_ui.txt",
		"gfx/interface/not_ui.txt",
	} {
		if _, ok := importedUIBaselineEntry(full, "Demo/"+name); ok {
			t.Errorf("full export leaked non-UI snapshot member %q", name)
		}
	}

	delta, err := buildNativeShadowWithBlobs(
		snapshot, nil, false, resolver, nativeBuildOptions{skipAdvisory: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := importedUIBaselineEntry(delta, "Demo/interface/custom.gui"); ok {
		t.Fatal("delta export included the imported UI baseline")
	}

	snapshot.Tables["project_raw_files"] = []json.RawMessage{
		json.RawMessage(`{"id":9,"dest_path":"interface/custom.gui","content":"raw winner","source":"created","raw_kind":"verbatim_override","is_binary":false}`),
	}
	withRaw, err := buildNativeShadowWithBlobs(
		snapshot, nil, true, resolver, nativeBuildOptions{skipAdvisory: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	winner, ok := importedUIBaselineEntry(withRaw, "Demo/interface/custom.gui")
	if !ok || string(winner.Bytes) != "raw winner" || winner.Origin != "raw_override" {
		t.Fatalf("raw same-path file did not win: %#v, present=%v", winner, ok)
	}

	index, err := BuildNativeFileIndex(importedUIBaselineSnapshot(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range index.Paths {
		if name == "Demo/interface/custom.gui" {
			t.Fatal("manifest-only resolver fabricated an unknown snapshot member")
		}
	}
}

func TestImportedUIBaselineTechnologyAssetsAreRebuiltWithoutPathCollisions(t *testing.T) {
	const (
		texturePath = "Demo/gfx/interface/technologies/ger_radio.dds"
		gfxPath     = "Demo/interface/GER_technologies.gfx"
	)
	archive := importedUIBaselineZIP(t, map[string][]byte{
		"gfx/interface/technologies/ger_radio.dds": []byte("stale imported technology texture"),
		"interface/GER_technologies.gfx":           []byte("stale imported technology declaration"),
	})
	snapshot := importedUIBaselineSnapshot()
	snapshot.Tables["technologies"] = []json.RawMessage{
		json.RawMessage(`{"id":7,"tech_id":"GER_radio","tech_line":"electronics","source":"imported","name_en":"Radio"}`),
	}
	snapshot.Tables["entity_obs_paths"] = []json.RawMessage{
		json.RawMessage(`{"id":8,"project_id":42,"entity_type":"tech_icon","entity_id":7,"obs_key":"tech/radio.png"}`),
	}
	picture := tolerancePNG(t)
	artifact, err := buildNativeShadowWithBlobs(
		snapshot, nil, true,
		func(key string, _ int64) ([]byte, error) {
			switch key {
			case "private/gui.zip":
				return archive, nil
			case "tech/radio.png":
				return picture, nil
			default:
				return nil, errors.New("unexpected object key: " + key)
			}
		},
		nativeBuildOptions{skipAdvisory: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	winners := map[string]Entry{}
	for _, entry := range artifact.Entries {
		if entry.Path == texturePath || entry.Path == gfxPath {
			counts[entry.Path]++
			winners[entry.Path] = entry
		}
	}
	for _, name := range []string{texturePath, gfxPath} {
		if counts[name] != 1 {
			t.Fatalf("artifact contains %d copies of %s, want exactly one", counts[name], name)
		}
		if winners[name].SourceEntity != "technology_icons" || winners[name].Origin != "generated" {
			t.Fatalf("%s winner = %#v, want rebuilt technology asset", name, winners[name])
		}
	}
	if !bytes.HasPrefix(winners[texturePath].Bytes, []byte("DDS ")) {
		t.Fatalf("technology texture did not replace imported baseline: %q", winners[texturePath].Bytes)
	}
	if gfx := string(winners[gfxPath].Bytes); !strings.Contains(gfx, "GFX_ger_radio_medium") ||
		!strings.Contains(gfx, "technologies/ger_radio.dds") {
		t.Fatalf("rebuilt technology declaration did not win:\n%s", gfx)
	}
	if _, err := Build(
		filepath.Join(t.TempDir(), "full.zip"), "baseline-technology-overlap",
		artifact.Entries, DefaultArtifactBudget(),
	); err != nil {
		t.Fatalf("complete artifact build rejected baseline/technology overlap: %v", err)
	}
}

func TestImportedUIBaselineFailuresAreWarnings(t *testing.T) {
	tests := []struct {
		name    string
		resolve BlobResolver
	}{
		{name: "no resolver"},
		{name: "read failure", resolve: func(string, int64) ([]byte, error) {
			return nil, errors.New("private store unavailable")
		}},
		{name: "corrupt archive", resolve: func(string, int64) ([]byte, error) {
			return []byte("not a zip"), nil
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact, err := buildNativeShadowWithBlobs(
				importedUIBaselineSnapshot(), nil, true, test.resolve,
				nativeBuildOptions{skipAdvisory: true},
			)
			if err != nil {
				t.Fatalf("baseline failure blocked the export: %v", err)
			}
			if !strings.Contains(strings.Join(artifact.Warnings, "\n"), "The imported mod's own interface files were left out") {
				t.Fatalf("baseline failure was silent: %#v", artifact.Warnings)
			}
		})
	}
}
