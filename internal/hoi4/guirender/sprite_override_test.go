package guirender

import "testing"

func TestSpriteSourceAndOverride(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"interface/ideas.gfx": []byte(`spriteTypes = {
	spriteType = { name = "GFX_idea_slot_army_chief" texturefile = "gfx/interface/ideas/select_advisor.dds" }
	spriteType = { name = "GFX_idea_slot_navy_chief" texturefile = "gfx/interface/ideas/select_advisor.dds" }
}`),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"interface/mod.gfx": []byte(`spriteTypes = { spriteType = { name = "GFX_mod_icon" texturefile = "gfx/interface/mod_icon.dds" } }`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}

	file, body, ok := project.SpriteSource("GFX_idea_slot_navy_chief")
	if !ok || file != "interface/ideas.gfx" || len(body) == 0 {
		t.Fatalf("base game sprite source = %q, %d bytes, %v", file, len(body), ok)
	}
	if file, _, ok := project.SpriteSource("GFX_mod_icon"); !ok || file != "interface/mod.gfx" {
		t.Fatalf("mod sprite source = %q, %v", file, ok)
	}
	if _, _, ok := project.SpriteSource("GFX_nothing"); ok {
		t.Fatal("an undeclared sprite has no source")
	}

	route := "/api/go/projects/2/gui-render/sprite?name=GFX_idea_slot_army_chief&v=abc"
	project.SetSpriteOverrides(map[string]SpriteOverride{
		"GFX_idea_slot_army_chief": {Width: 80, Height: 90, URL: route},
		"GFX_mod_icon":             {Width: 10, Height: 10},
	})
	replaced := project.spritePayload("GFX_idea_slot_army_chief")
	if replaced["overridden"] != true || replaced["resolved"] != true || replaced["width"] != 80 || replaced["height"] != 90 {
		t.Fatalf("replaced sprite payload = %v", replaced)
	}

	if replaced["vanilla_url"] != route {
		t.Fatalf("vanilla_url = %v", replaced["vanilla_url"])
	}
	if _, has := project.spritePayload("GFX_mod_icon")["vanilla_url"]; has {
		t.Fatal("without a route, a replaced sprite has no store address at all")
	}
	if replaced["override_path"] != "gui_sprite_overrides/GFX_idea_slot_army_chief" {
		t.Fatalf("override_path = %v", replaced["override_path"])
	}
	if untouched := project.spritePayload("GFX_idea_slot_navy_chief"); untouched["overridden"] != false {
		t.Fatalf("an untouched sprite is not replaced: %v", untouched)
	}
}
