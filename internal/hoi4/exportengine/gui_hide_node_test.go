package exportengine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

const hideNodeSource = `guiTypes = {
	containerWindowType = {
		name = "panel"
		size = { width = 600 height = 400 }
		buttonType = { name = "ok" position = { x = 20 y = 30 } quadTextureSprite = "GFX_button_123x34" }
		iconType = { name = "flag" spriteType = "GFX_flag_small" hide = no }
		iconType = { name = "kept" spriteType = "GFX_kept" }
	}
}`

func TestHideNodePreviewMarksTheElementInPlace(t *testing.T) {
	out, err := ApplyGUIPreviewEdits([]byte(hideNodeSource), []map[string]any{
		{"target": "panel/ok", "op": "hide_node", "payload": map[string]any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		`name = "ok" position = { x = 20 y = 30 } quadTextureSprite = "GFX_button_123x34" gameview_hidden = yes`,
		`name = "kept" spriteType = "GFX_kept" }`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
}

func TestHideNodeExportParksTheElementOffScreen(t *testing.T) {
	ast, _, err := paradox.Parse(hideNodeSource)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"panel/ok", "panel/flag"} {
		node, _, _ := resolveGUINode(ast, target)
		if node == nil {
			t.Fatalf("%s not found", target)
		}
		hideGUINode(node)
	}
	text := strings.Join(strings.Fields(importsvc.ToScript(ast, 0)), " ")
	for _, want := range []string{
		`name = "ok" position = { x = -10000 y = -10000 } quadTextureSprite = "GFX_button_123x34" hide = yes`,
		`name = "flag" spriteType = "GFX_flag_small" hide = yes position = { x = -10000 y = -10000 }`,
		`name = "kept" spriteType = "GFX_kept" }`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "gameview_hidden") || strings.Contains(text, `"yes"`) {
		t.Errorf("GameView's mark or a quoted yes reached the export:\n%s", text)
	}
}

func TestNativeGUIOverlaysExportsAHiddenElementOffScreen(t *testing.T) {
	snapshot, resolve := guiOrphanTestSnapshot(t, `[{"target":"Top/Label","op":"hide_node","payload":{}}]`)
	entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, &imageSink{}, false)
	if err != nil {
		t.Fatal(err)
	}
	var gui string
	for _, entry := range entries {
		if entry.Path == "Demo/interface/demo.gui" {
			gui = strings.Join(strings.Fields(string(entry.Bytes)), " ")
		}
	}
	want := `name = Label text = OLD position = { x = -10000 y = -10000 } hide = yes`
	if !strings.Contains(gui, want) {
		t.Fatalf("missing %s in the exported file:\n%s", want, gui)
	}
	if strings.Contains(gui, "gameview_hidden") {
		t.Fatalf("GameView's mark reached the mod:\n%s", gui)
	}
}

func TestSwapSpriteLeavesTheGameSwappedMenuBackgroundAlone(t *testing.T) {
	const menu = `guiTypes = {
	containerWindowType = {
		name = "frontend_background"
		background = { name = "Background" quadTextureSprite = "GFX_frontend_bg" }
	}
	containerWindowType = {
		name = "panel"
		background = { name = "Background" quadTextureSprite = "GFX_tiled_window" }
	}
}`
	edits := []map[string]any{
		{"target": "frontend_background", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_gv_frontend_background_1"}},
		{"target": "panel", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_gv_panel_1"}},
	}
	out, err := ApplyGUIPreviewEdits([]byte(menu), edits)
	if err != nil {
		t.Fatal(err)
	}
	preview := strings.Join(strings.Fields(string(out)), " ")
	if !strings.Contains(preview, `quadTextureSprite = "GFX_frontend_bg"`) || strings.Contains(preview, "GFX_gv_frontend_background_1") {
		t.Errorf("preview swapped the menu background:\n%s", out)
	}
	if !strings.Contains(preview, `quadTextureSprite = "GFX_gv_panel_1"`) {
		t.Errorf("preview left the panel alone:\n%s", out)
	}

	var sourceZip bytes.Buffer
	zw := zip.NewWriter(&sourceZip)
	w, _ := zw.Create("interface/frontendmainview.gui")
	_, _ = w.Write([]byte(menu))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(edits)
	snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
		"gui_overlays":       {json.RawMessage(`{"id":1,"file":"interface/frontendmainview.gui","source":"vanilla","edits":` + string(encoded) + `}`)},
		"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/source.zip"}`)},
	}}
	resolve := func(key string, max int64) ([]byte, error) {
		if key == "gui/source.zip" {
			return sourceZip.Bytes(), nil
		}
		return nil, fmt.Errorf("unexpected key %s", key)
	}
	sink := &imageSink{}
	entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
	if err != nil {
		t.Fatal(err)
	}
	var gui string
	for _, entry := range entries {
		if entry.Path == "Demo/interface/frontendmainview.gui" {
			gui = strings.Join(strings.Fields(string(entry.Bytes)), " ")
		}
	}
	if !strings.Contains(gui, `quadTextureSprite = "GFX_frontend_bg"`) || strings.Contains(gui, "GFX_gv_frontend_background_1") || !strings.Contains(gui, "GFX_gv_panel_1") {
		t.Errorf("exported:\n%s", gui)
	}
	if len(sink.warnings) != 1 || !strings.Contains(sink.warnings[0], "swap_sprite on frontend_background") {
		t.Errorf("notes = %#v", sink.warnings)
	}
}

func TestNativeGUIOverlaysExportsUnclosedFilesAndSkipsUnreadableOnes(t *testing.T) {
	export := func(text string) (string, []string) {
		var sourceZip bytes.Buffer
		zw := zip.NewWriter(&sourceZip)
		w, _ := zw.Create("interface/demo.gui")
		_, _ = w.Write([]byte(text))
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		snapshot := Snapshot{ProjectID: 42, Project: []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`), Tables: map[string][]json.RawMessage{
			"gui_overlays":       {json.RawMessage(`{"id":1,"file":"interface/demo.gui","source":"imported","edits":[{"target":"Top/Label","op":"set_layout","payload":{"x":20}}]}`)},
			"project_obs_assets": {json.RawMessage(`{"asset_type":"gui_source","gfx_name":"snapshot.zip","obs_key":"gui/source.zip"}`)},
		}}
		resolve := func(key string, max int64) ([]byte, error) {
			if key == "gui/source.zip" {
				return sourceZip.Bytes(), nil
			}
			return nil, fmt.Errorf("unexpected key %s", key)
		}
		sink := &imageSink{}
		entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
		if err != nil {
			t.Fatalf("the export failed: %v", err)
		}
		for _, entry := range entries {
			if entry.Path == "Demo/interface/demo.gui" {
				return string(entry.Bytes), sink.warnings
			}
		}
		return "", sink.warnings
	}
	gui, notes := export("guiTypes = { containerWindowType = { name = Top instantTextBoxType = { name = Label position = { x = 1 y = 2 } } }")
	if !strings.Contains(gui, "x = 20") || strings.Count(gui, "{") != strings.Count(gui, "}") || len(notes) != 0 {
		t.Errorf("unclosed file: notes %v\n%s", notes, gui)
	}
	gui, notes = export("guiTypes = { containerWindowType = { name = Top } } }")
	if gui != "" || len(notes) != 1 || !strings.Contains(notes[0], "could not be read back") {
		t.Errorf("unreadable file: notes %v\n%s", notes, gui)
	}
}
