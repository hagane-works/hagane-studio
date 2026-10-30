package exportengine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func guiOrphanTestSnapshot(t *testing.T, edits string) (Snapshot, BlobResolver) {
	t.Helper()
	var sourceZip bytes.Buffer
	zw := zip.NewWriter(&sourceZip)
	w, _ := zw.Create("interface/demo.gui")
	_, _ = w.Write([]byte("guiTypes = { containerWindowType = { name = Top instantTextBoxType = { name = Label text = OLD position = { x = 1 y = 2 } } } }"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"gui_overlays":       {json.RawMessage(`{"id":1,"file":"interface/demo.gui","source":"imported","edits":` + edits + `}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/source.zip"}`)},
	}}
	resolve := func(key string, max int64) ([]byte, error) {
		if key == "gui/source.zip" {
			return sourceZip.Bytes(), nil
		}
		return nil, fmt.Errorf("unexpected key %s", key)
	}
	return snapshot, resolve
}

func TestNativeGUIOverlaysNotesOrphanEditAndAppliesTheRest(t *testing.T) {

	snapshot, resolve := guiOrphanTestSnapshot(t, `[
		{"target":"Top/Label","op":"set_layout","payload":{"x":20,"y":30}},
		{"target":"Top/Gone","op":"set_text","payload":{"text":"Ghost"}}
	]`)
	sink := &imageSink{}
	entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
	if err != nil {
		t.Fatalf("orphan edit must not fail the export: %v", err)
	}
	var gui string
	for _, entry := range entries {
		if entry.Path == "Demo/interface/demo.gui" {
			gui = string(entry.Bytes)
		}
	}
	if !strings.Contains(gui, "x = 20") || !strings.Contains(gui, "y = 30") {
		t.Fatalf("live edit must still apply:\n%s", gui)
	}
	if strings.Contains(gui, "Ghost") {
		t.Fatalf("orphan edit must not leak into the file:\n%s", gui)
	}
	if len(sink.warnings) != 1 {
		t.Fatalf("exactly one note expected, got %#v", sink.warnings)
	}
	note := sink.warnings[0]
	for _, want := range []string{
		"Interface edits in",
		`"interface/demo.gui"`,
		"set_text on Top/Gone",
		"no longer in the imported interface file",
		"The other interface edits were exported normally",
	} {
		if !strings.Contains(note, want) {
			t.Fatalf("note missing %q: %q", want, note)
		}
	}
}

func TestNativeGUIOverlaysAggregatesManyOrphansIntoOneNote(t *testing.T) {

	snapshot, resolve := guiOrphanTestSnapshot(t, `[
		{"target":"Top/A","op":"set_text","payload":{"text":"a"}},
		{"target":"Top/B","op":"set_layout","payload":{"x":1}},
		{"target":"Top/C","op":"set_orientation","payload":{"value":"CENTER"}},
		{"target":"Top/D","op":"add_node","payload":{"node":{"type":"iconType","name":"n"}}},
		{"target":"Top/E","op":"delete_node","payload":{}}
	]`)
	sink := &imageSink{}
	if _, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false); err != nil {
		t.Fatal(err)
	}
	if len(sink.warnings) != 1 {
		t.Fatalf("orphans must aggregate into one note per file, got %#v", sink.warnings)
	}
	note := sink.warnings[0]
	for _, want := range []string{"Interface edits in", "left out (5)", "set_text on Top/A", "and 2 more", "no longer in the imported interface file"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note missing %q: %q", want, note)
		}
	}
}

func TestNativeGUIOverlaysNoOrphanNoNote(t *testing.T) {
	snapshot, resolve := guiOrphanTestSnapshot(t, `[
		{"target":"Top/Label","op":"set_layout","payload":{"x":20,"y":30}},
		{"target":"Top/Label","op":"set_orientation","payload":{"value":"CENTER"}}
	]`)
	sink := &imageSink{}
	if _, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false); err != nil {
		t.Fatal(err)
	}
	if len(sink.warnings) != 0 {
		t.Fatalf("no orphan ⇒ no note, got %#v", sink.warnings)
	}
}

func TestBuildNativeShadowSurfacesGUIOrphanNote(t *testing.T) {
	snapshot, resolve := guiOrphanTestSnapshot(t, `[{"target":"Top/Gone","op":"set_layout","payload":{"x":1}}]`)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, resolve)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, warning := range artifact.Warnings {
		if strings.Contains(warning, "Interface edits in") && strings.Contains(warning, "set_layout on Top/Gone") {
			found = true
		}
	}
	if !found {
		t.Fatalf("orphan note must reach artifact.Warnings; got %#v", artifact.Warnings)
	}
}
