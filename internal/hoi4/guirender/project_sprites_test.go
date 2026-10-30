package guirender

import "testing"

func TestProjectSpritesCountAsDrawn(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"interface/goals.gfx": []byte(`spriteTypes = { spriteType = { name = "GFX_focus_TST_rearm" texturefile = "gfx/interface/goals/TST_rearm.dds" } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	before := idx.SpritesPayload([]string{"GFX_focus_TST_rearm", "GFX_decision_TST_road"})
	if before["GFX_focus_TST_rearm"].(map[string]any)["resolved"] != false {
		t.Fatalf("resolved before the project has a picture: %v", before)
	}
	idx.SetProjectSprites(map[string][2]int{"GFX_focus_TST_rearm": {100, 88}, "GFX_decision_TST_road": {33, 32}})
	after := idx.SpritesPayload([]string{"GFX_focus_TST_rearm", "GFX_decision_TST_road"})
	for name, size := range map[string][2]int{"GFX_focus_TST_rearm": {100, 88}, "GFX_decision_TST_road": {33, 32}} {
		sprite := after[name].(map[string]any)
		if sprite["resolved"] != true || sprite["missing_file"] != false || sprite["width"] != size[0] || sprite["height"] != size[1] {
			t.Errorf("%s: %v", name, sprite)
		}
		if _, has := sprite["version"]; has {
			t.Errorf("%s: a version before any was set: %v", name, sprite)
		}
	}

	idx.SetProjectSprites(map[string][2]int{
		"GFX_focus_TST_rearm": {100, 88}, "gfx/leaders/TST/portrait_TST_ada.png": {156, 210},
	})
	for _, version := range []string{"a1b2c3d4e5f6", "0f0e0d0c0b0a"} {
		idx.SetProjectSpriteVersions(map[string]string{
			"GFX_focus_TST_rearm": version, "gfx/leaders/TST/portrait_TST_ada.png": version,
		})
		got := idx.SpritesPayload([]string{"GFX_focus_TST_rearm", "gfx/leaders/TST/portrait_TST_ada.png"})
		for name, sprite := range got {
			if sprite.(map[string]any)["version"] != version {
				t.Errorf("%s version = %v, want %s", name, sprite.(map[string]any)["version"], version)
			}
		}
	}
}

func TestProjectSpriteWinsOverLowerTextureAtTheSamePath(t *testing.T) {
	below, err := NewIndex(map[string][]byte{
		"gfx/interface/goals/TST_rearm.png": pngOf(t, 12, 9),
	})
	if err != nil {
		t.Fatal(err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"interface/goals.gfx": []byte(`spriteTypes = { spriteType = { name = "GFX_focus_TST_rearm" texturefile = "gfx/interface/goals/TST_rearm.dds" } }`),
	}, below)
	if err != nil {
		t.Fatal(err)
	}
	top.SetProjectSprites(map[string][2]int{"GFX_focus_TST_rearm": {100, 88}})
	payload := top.SpritesPayload([]string{"GFX_focus_TST_rearm"})["GFX_focus_TST_rearm"].(map[string]any)
	if payload["width"] != 100 || payload["height"] != 88 || payload["vanilla_url"] != nil {
		t.Fatalf("project sprite lost to lower texture: %v", payload)
	}
}

func TestSpritePNGReadsTheEffectiveFallbackTexture(t *testing.T) {
	below, err := NewIndex(map[string][]byte{
		"interface/imported.gfx":     []byte(`spriteTypes = { spriteType = { name = GFX_imported texturefile = "gfx/interface/imported.png" } }`),
		"gfx/interface/imported.png": pngOf(t, 13, 11),
	})
	if err != nil {
		t.Fatal(err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"events/project.txt": []byte("add_namespace = project"),
	}, below)
	if err != nil {
		t.Fatal(err)
	}
	body, width, height, ok := top.SpritePNG("GFX_imported")
	if !ok || len(body) == 0 || width != 13 || height != 11 {
		t.Fatalf("fallback sprite = %d bytes, %dx%d, %v", len(body), width, height, ok)
	}
	if member, ok := top.SpriteTextureMember("GFX_imported"); !ok || member != "gfx/interface/imported.png" {
		t.Fatalf("effective texture member = %q, %v", member, ok)
	}
}

func TestSpriteTextureMemberDoesNotResurrectAShadowedImportedImage(t *testing.T) {
	imported, err := NewIndex(map[string][]byte{
		"interface/imported.gfx":     []byte(`spriteTypes = { spriteType = { name = GFX_imported texturefile = "gfx/interface/imported.png" } }`),
		"gfx/interface/imported.png": pngOf(t, 13, 11),
	})
	if err != nil {
		t.Fatal(err)
	}
	masked, err := imported.WithoutFiles([]string{"GFX/interface/IMPORTED.PNG"})
	if err != nil {
		t.Fatal(err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"interface/project.gfx": []byte(`spriteTypes = { spriteType = { name = GFX_imported texturefile = "gfx/interface/imported.png" } }`),
	}, masked)
	if err != nil {
		t.Fatal(err)
	}
	if member, ok := top.SpriteTextureMember("GFX_imported"); ok || member != "" {
		t.Fatalf("shadowed imported image resurfaced as %q", member)
	}
	if _, _, _, ok := top.SpritePNG("GFX_imported"); ok {
		t.Fatal("shadowed imported image was decoded from a lower layer")
	}
}

func TestSpriteTextureMemberFindsAnEffectiveLooseFallbackTexture(t *testing.T) {
	imported, err := NewIndex(map[string][]byte{
		"gfx/interface/goals/goal_imported.png": pngOf(t, 13, 11),
	})
	if err != nil {
		t.Fatal(err)
	}
	top, err := NewIndexWithFallback(map[string][]byte{
		"events/project.txt": []byte("add_namespace = project"),
	}, imported)
	if err != nil {
		t.Fatal(err)
	}
	if member, ok := top.SpriteTextureMember("GFX_goal_imported"); !ok || member != "gfx/interface/goals/goal_imported.png" {
		t.Fatalf("effective loose texture member = %q, %v", member, ok)
	}

	masked, err := imported.WithoutFiles([]string{"gfx/interface/goals/goal_imported.png"})
	if err != nil {
		t.Fatal(err)
	}
	maskedTop, err := NewIndexWithFallback(map[string][]byte{}, masked)
	if err != nil {
		t.Fatal(err)
	}
	if member, ok := maskedTop.SpriteTextureMember("GFX_goal_imported"); ok || member != "" {
		t.Fatalf("shadowed loose imported image resurfaced as %q", member)
	}
}
