package exportengine

import (
	"strings"
	"testing"
)

func TestSwapSpritePointsTheElementsOwnPictureKey(t *testing.T) {
	source := []byte(`guiTypes = {
	containerWindowType = {
		name = "panel"
		size = { width = 600 height = 400 }
		background = { name = "Background" quadTextureSprite = "GFX_tiled_window_1b_border" }
		buttonType = { name = "ok" quadTextureSprite = "GFX_button_123x34" }
		iconType = { name = "flag" spriteType = "GFX_flag_small" }
		containerWindowType = { name = "inner" }
	}
}`)
	edits := []map[string]any{
		{"target": "panel", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_gv_panel"}},
		{"target": "panel/ok", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_gv_ok"}},
		{"target": "panel/flag", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_gv_flag"}},
		{"target": "panel/inner", "op": "swap_sprite", "payload": map[string]any{"gfx_ref": "GFX_gv_inner"}},
	}
	out, err := ApplyGUIPreviewEdits(source, edits)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		`quadTextureSprite = "GFX_gv_panel"`,
		`quadTextureSprite = "GFX_gv_ok"`,
		`spriteType = "GFX_gv_flag"`,
		`name = "inner" background = { name = "Background" quadTextureSprite = "GFX_gv_inner" }`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
	for _, gone := range []string{"GFX_tiled_window_1b_border", "GFX_button_123x34", "GFX_flag_small", " sprite = "} {
		if strings.Contains(text, gone) {
			t.Errorf("%q still there:\n%s", gone, out)
		}
	}
}
