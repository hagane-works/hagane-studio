package exportengine

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"slices"
	"sort"
	"testing"
)

func TestBuildNativeFileIndexMatchesMaterializedNativePathsWithoutZIP(t *testing.T) {
	var source bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0x7f
	}
	if err := png.Encode(&source, img); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 42,
		Project: json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"technologies":     {json.RawMessage(`{"id":7,"tech_id":"GER_radio","tech_line":"electronics","source":"created","name_en":"Radio"}`)},
			"entity_obs_paths": {json.RawMessage(`{"id":8,"project_id":42,"entity_type":"tech_icon","entity_id":7,"obs_key":"tech/radio.png"}`)},
		},
	}
	materialized, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) {
		return source.Bytes(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := make([]string, 0, len(materialized.Entries)+1)
	for _, entry := range materialized.Entries {
		want = append(want, entry.Path)
	}
	want = append(want, artifactManifestPath(materialized.Entries))
	sort.Strings(want)

	index, err := BuildNativeFileIndex(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(index.Paths)
	if !slices.Equal(index.Paths, want) {
		t.Fatalf("file index drift:\n got=%v\nwant=%v", index.Paths, want)
	}
	if !slices.Contains(index.Paths, "Demo/gfx/interface/technologies/ger_radio.dds") {
		t.Fatalf("private asset destination omitted: %v", index.Paths)
	}
	if !slices.Contains(index.Paths, "Demo/"+ArtifactManifestBasename) {
		t.Fatalf("embedded manifest path omitted: %v", index.Paths)
	}
}

func TestBuildNativeFileIndexDerivesGUIPathsWithoutReadingSnapshotBlob(t *testing.T) {
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 42,
		Project: json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en","zh"]}`),
		Tables: map[string][]json.RawMessage{
			"gui_overlays": {json.RawMessage(`{"file":"interface/demo.gui","source":"import","edits":[{"target":"root","op":"swap_sprite","payload":{"override_file":"demo.png"}},{"target":"root","op":"set_text","payload":{"key":"DEMO_LABEL","text":"Demo"}}]}`)},
			"project_obs_assets": {
				json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"private/gui.zip"}`),
				json.RawMessage(`{"asset_type":"gui_sprite_override","gfx_name":"demo.png","obs_key":"private/demo.png"}`),
			},
		},
	}
	index, err := BuildNativeFileIndex(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Demo/interface/demo.gui",
		"Demo/interface/demo_sprites.gfx",
		"Demo/gfx/interface/demo.dds",
		"Demo/localisation/english/replace/demo_gui_submod_l_english.yml",
		"Demo/localisation/simp_chinese/replace/demo_gui_submod_l_simp_chinese.yml",
	} {
		if !slices.Contains(index.Paths, want) {
			t.Errorf("missing GUI manifest path %q: %v", want, index.Paths)
		}
	}
}

func TestFileIndexFromLegacyManifestDerivesEmbeddedManifestPath(t *testing.T) {
	index := FileIndexFromManifest(Manifest{Entries: []ManifestEntry{
		{Path: "Demo.mod"}, {Path: "Demo/descriptor.mod"},
	}}, "Demo.zip")
	if !slices.Contains(index.Paths, "Demo/"+ArtifactManifestBasename) {
		t.Fatalf("legacy manifest lost embedded path: %v", index.Paths)
	}
}
