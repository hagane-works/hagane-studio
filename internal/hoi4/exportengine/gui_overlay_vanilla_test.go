package exportengine

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNativeGUIOverlaysExportABaseGameScreenFromItsPinnedScaffold(t *testing.T) {
	scaffold := []byte("guiTypes = { containerWindowType = { name = topbar iconType = { name = flag position = { x = 1 y = 2 } } } }")
	hash := sha256Hex(scaffold)
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"gui_overlays": {json.RawMessage(`{"id":1,"file":"interface/topbar.gui","source":"vanilla","scaffold_hash":"` + hash +
			`","edits":[{"target":"topbar/flag","op":"set_layout","payload":{"x":40}}]}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_vanilla_scaffold","gfx_name":"` + hash + `","obs_key":"p/x/gui-scaffolds/` + hash + `.gui"}`)},
	}}
	stored := scaffold
	resolve := func(key string, _ int64) ([]byte, error) {
		if key == "p/x/gui-scaffolds/"+hash+".gui" {
			return stored, nil
		}
		return nil, fmt.Errorf("unexpected key %s", key)
	}

	sink := &imageSink{}
	entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != "Demo/interface/topbar.gui" || !strings.Contains(string(entries[0].Bytes), "x = 40") {
		t.Fatalf("base-game overlay not exported: %#v notes=%v", entries, sink.warnings)
	}

	index, err := nativeGUIOverlayFileIndex(snapshot, "Demo", map[string]string{"gui_vanilla_scaffold\x00" + hash: "k"}, false)
	if err != nil || len(index) == 0 || index[0].Path != "Demo/interface/topbar.gui" {
		t.Fatalf("file index %#v err=%v", index, err)
	}

	stored = []byte("guiTypes = { containerWindowType = { name = topbar } }")
	sink = &imageSink{}
	entries, err = nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 || !strings.Contains(strings.Join(sink.warnings, "\n"), "interface/topbar.gui") {
		t.Fatalf("a mismatched scaffold was used: %#v notes=%v", entries, sink.warnings)
	}
}

func TestCollectBlobRequestsIncludesPinnedGUIScaffolds(t *testing.T) {
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42}`), Tables: map[string][]json.RawMessage{
		"gui_overlays":       {json.RawMessage(`{"id":1,"file":"interface/topbar.gui","source":"vanilla","edits":[]}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_vanilla_scaffold","gfx_name":"abc","obs_key":"p/x/gui-scaffolds/abc.gui"}`)},
	}}
	for _, request := range CollectBlobRequests(snapshot, false) {
		if request.ObjectKey == "p/x/gui-scaffolds/abc.gui" {
			return
		}
	}
	t.Fatal("pinned GUI scaffold is not requested")
}
