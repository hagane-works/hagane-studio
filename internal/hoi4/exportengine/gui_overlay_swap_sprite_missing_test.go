package exportengine

import (
	"strings"
	"testing"
)

func TestNativeGUIOverlaysNotesSwapSpriteWithoutUploadedOverride(t *testing.T) {
	snapshot, resolve := guiOrphanTestSnapshot(t, `[
		{"target":"Top/Label","op":"set_layout","payload":{"x":20,"y":30}},
		{"target":"Top/Label","op":"swap_sprite","payload":{"override_file":"missing_button.dds"}}
	]`)
	sink := &imageSink{}
	entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
	if err != nil {
		t.Fatalf("missing override must not fail the export: %v", err)
	}
	var gui string
	for _, entry := range entries {
		if entry.Path == "Demo/interface/demo.gui" {
			gui = string(entry.Bytes)
		}
		if strings.HasSuffix(entry.Path, ".dds") && strings.Contains(entry.Path, "missing_button") {
			t.Fatalf("no sprite must be emitted for a missing override: %s", entry.Path)
		}
	}
	if !strings.Contains(gui, "x = 20") {
		t.Fatalf("sibling edit must still apply:\n%s", gui)
	}
	if strings.Contains(gui, "GFX_mysub_") {
		t.Fatalf("sprite must not be rewritten when the override is missing:\n%s", gui)
	}
	if len(sink.warnings) != 1 {
		t.Fatalf("exactly one note expected, got %#v", sink.warnings)
	}
	note := sink.warnings[0]
	for _, want := range []string{
		"Picture swap in",
		`"interface/demo.gui"`,
		"swap_sprite on Top/Label",
		`"missing_button.dds"`,
		"no uploaded file was found",
		"The other interface edits were exported normally",
	} {
		if !strings.Contains(note, want) {
			t.Fatalf("note missing %q: %q", want, note)
		}
	}
}

func TestNativeGUIOverlaysExportsExistingSpriteReferenceWithoutOverride(t *testing.T) {
	snapshot, resolve := guiOrphanTestSnapshot(t, `[
		{"target":"Top/Label","op":"swap_sprite","payload":{"gfx_ref":"GFX_existing_button"}}
	]`)
	sink := &imageSink{}
	entries, err := nativeGUIOverlays(snapshot, "Demo", resolve, sink, false)
	if err != nil {
		t.Fatalf("existing GFX reference swap: %v", err)
	}
	var gui string
	for _, entry := range entries {
		if entry.Path == "Demo/interface/demo.gui" {
			gui = string(entry.Bytes)
		}
	}

	if !strings.Contains(gui, `spriteType = "GFX_existing_button"`) || strings.Contains(gui, "	sprite = ") {
		t.Fatalf("existing sprite reference was not exported:\n%s", gui)
	}
	if len(sink.warnings) != 0 {
		t.Fatalf("existing GFX reference must not need an uploaded override: %#v", sink.warnings)
	}
}
