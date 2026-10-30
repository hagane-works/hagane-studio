package exportengine

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/paradox"
)

func previewZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestReadGUIPreviewSnapshotRejectsTraversalAndBoundsRetainedImages(t *testing.T) {
	if _, err := ReadGUIPreviewSnapshot(previewZip(t, map[string][]byte{"../escape.gui": []byte("x")})); err == nil {
		t.Fatal("zip traversal accepted")
	}
	raw := bytes.Repeat([]byte{0x7f}, 2<<20)
	files, err := ReadGUIPreviewSnapshot(previewZip(t, map[string][]byte{
		"gfx/interface/large.dds": raw,
		"interface/demo.gui":      []byte("guiTypes = {}"),
		"ignored/blob.bin":        raw,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(files["gfx/interface/large.dds"]); got != int(guiPreviewImagePrefix) {
		t.Fatalf("retained image bytes=%d", got)
	}
	if _, ok := files["ignored/blob.bin"]; ok {
		t.Fatal("unrelated blob retained")
	}
}

func TestReadGUIPreviewSnapshotRejectsOversizedScript(t *testing.T) {
	body := bytes.Repeat([]byte("x"), int(guiPreviewScriptLimit)+1)
	if _, err := ReadGUIPreviewSnapshot(previewZip(t, map[string][]byte{"interface/huge.gui": body})); err == nil {
		t.Fatal("oversized script accepted")
	}
}

func TestReadGUIPreviewSnapshotKeepsGameData(t *testing.T) {
	files, err := ReadGUIPreviewSnapshot(previewZip(t, map[string][]byte{
		"events/mod.txt":             []byte("add_namespace = mod"),
		"gfx/entities/tank.asset":    []byte("entity = {}"),
		"map/definition.csv":         []byte("0;0;0;0;land"),
		"history/countries/huge.txt": bytes.Repeat([]byte("x"), int(guiPreviewScriptLimit)+1),
		"interface/demo.gui":         []byte("guiTypes = {}"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"events/mod.txt", "map/definition.csv", "interface/demo.gui"} {
		if _, ok := files[name]; !ok {
			t.Errorf("%s was not kept", name)
		}
	}
	for _, name := range []string{"history/countries/huge.txt", "gfx/entities/tank.asset"} {
		if _, ok := files[name]; ok {
			t.Errorf("%s was kept", name)
		}
	}
}

func TestReadGUIPreviewSnapshotKeepsWebFontsWhole(t *testing.T) {
	ttf := bytes.Repeat([]byte{0x54}, int(guiPreviewImagePrefix)+17)
	otf := bytes.Repeat([]byte{0x4f}, int(guiPreviewImagePrefix)+23)
	files, err := ReadGUIPreviewSnapshot(previewZip(t, map[string][]byte{
		"gfx/fonts/custom.ttf": ttf,
		"gfx/fonts/custom.otf": otf,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files["gfx/fonts/custom.ttf"], ttf) {
		t.Fatal("TTF was omitted or reduced to an image header")
	}
	if !bytes.Equal(files["gfx/fonts/custom.otf"], otf) {
		t.Fatal("OTF was omitted or reduced to an image header")
	}
}

func TestApplyGUIPreviewEditsUsesExporterNamePathSemantics(t *testing.T) {
	source := []byte(`guiTypes={containerWindowType={name=Root position={x=1 y=2} instantTextBoxType={name=Label text=OLD}}}`)
	edited, err := ApplyGUIPreviewEdits(source, []map[string]any{
		{"target": "Root/Label", "op": "set_text", "payload": map[string]any{"text": "NEW"}},
		{"target": "Root/Label", "op": "set_layout", "payload": map[string]any{"x": 9, "y": 8}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ast, _, err := paradox.Parse(string(edited))
	if err != nil {
		t.Fatal(err)
	}
	node, _, _ := resolveGUINode(ast, "Root/Label")
	if node == nil || !strings.Contains(string(edited), `text = "NEW"`) || !strings.Contains(string(edited), "x = 9") {
		t.Fatalf("overlay not applied:\n%s", edited)
	}
}

func TestApplyGUIPreviewEditsPreservesExplicitEmptyValuesAndClampsInsert(t *testing.T) {
	source := []byte(`guiTypes={containerWindowType={name=Root instantTextBoxType={name=Label text=OLD orientation=LEFT}}}`)
	edited, err := ApplyGUIPreviewEdits(source, []map[string]any{
		{"target": "Root/Label", "op": "set_text", "payload": map[string]any{"text": ""}},
		{"target": "Root/Label", "op": "set_orientation", "payload": map[string]any{"value": ""}},
		{"target": "Root", "op": "add_node", "payload": map[string]any{
			"index": -10, "node": map[string]any{"type": "buttonType"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(edited)
	if !strings.Contains(text, `text = ""`) || !strings.Contains(text, `orientation = ""`) {
		t.Fatalf("explicit empty values lost:\n%s", text)
	}
	if strings.Index(text, "buttonType") > strings.Index(text, "instantTextBoxType") {
		t.Fatalf("negative insert index was not clamped to zero:\n%s", text)
	}
}

func TestGUIOverlayPreviewLocalisationsMatchesExportedKeyRules(t *testing.T) {
	got := GUIOverlayPreviewLocalisations([]map[string]any{
		{"op": "set_text", "payload": map[string]any{
			"key": "NEW_LABEL", "is_new": true,
			"lang_texts": map[string]any{"en": "Hello", "zh": "你好"},
		}},
		{"op": "set_text", "payload": map[string]any{
			"key": "EXISTING_LABEL", "is_new": false, "text": "Override",
		}},
		{"op": "set_text", "payload": map[string]any{
			"key": "REFERENCE_ONLY", "is_new": false,
		}},
	})
	if got["GUISUB_NEW_LABEL"] != "Hello" || got["EXISTING_LABEL"] != "Override" {
		t.Fatalf("candidate localisation projection=%#v", got)
	}
	if _, exists := got["REFERENCE_ONLY"]; exists {
		t.Fatalf("key-only existing reference was treated as generated: %#v", got)
	}
}

func TestGUIOverlayPreviewLocalisationsInTheReadLanguage(t *testing.T) {
	edits := []map[string]any{
		{"op": "set_text", "payload": map[string]any{
			"key": "BOTH", "is_new": true,
			"lang_texts": map[string]any{"en": "Hello", "zh": "你好"},
		}},
		{"op": "set_text", "payload": map[string]any{
			"key": "ZH_ONLY", "is_new": true,
			"lang_texts": map[string]any{"zh": "只写了中文"},
		}},
	}
	zh := GUIOverlayPreviewLocalisationsIn(edits, "zh")
	if zh["GUISUB_BOTH"] != "你好" || zh["GUISUB_ZH_ONLY"] != "只写了中文" {
		t.Fatalf("zh projection=%#v", zh)
	}
	en := GUIOverlayPreviewLocalisationsIn(edits, "")
	if en["GUISUB_BOTH"] != "Hello" || en["GUISUB_ZH_ONLY"] != "只写了中文" {
		t.Fatalf("en projection=%#v", en)
	}
}

func TestNormalizeGUISourcePath(t *testing.T) {
	for _, bad := range []string{"", "../secret.gui", "/absolute.gui", `..\\secret.gui`, "demo.txt"} {
		if _, err := NormalizeGUISourcePath(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if got, err := NormalizeGUISourcePath(`folder\\demo.gui`); err != nil || got != "interface/folder/demo.gui" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}
