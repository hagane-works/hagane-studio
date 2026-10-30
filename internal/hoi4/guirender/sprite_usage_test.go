package guirender

import "testing"

func TestSpriteUsageIncludesBindingsLocatorsAndProjectTooltipNotPixelGuesses(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"interface/demo.gui":                      []byte(`guiTypes={containerWindowType={name=demo buttonType={name=action quadTextureSprite=GFX_action frame=2 pdx_tooltip=ACTION_TT} background={name=plate spriteType=GFX_plate}}}`),
		"localisation/english/demo_l_english.yml": []byte("l_english:\n ACTION_TT:0 \"Deploy available units\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(map[string][]byte{
		"localisation/english/project_l_english.yml": []byte("l_english:\n ACTION_TT:0 \"Deploy the ruler's guards\"\n"),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	all := idx.SpriteUsagePayload([]string{"GFX_action", "GFX_plate", "GFX_unknown"})
	use := all["GFX_action"].(map[string]any)
	refs := use["references"].([]map[string]any)
	if len(refs) != 1 || refs[0]["name_path"] != "demo/action" || refs[0]["gui_frame"] != 2 || use["exact_runtime_condition_available"] != false {
		t.Fatalf("lost exact source/state evidence: %+v", use)
	}
	text := refs[0]["texts"].(map[string]any)["ACTION_TT"].(map[string]any)
	if text["text"] != "Deploy the ruler's guards" || text["resolved"] != true {
		t.Fatalf("wrong layer: %+v", text)
	}
	if len(all["GFX_plate"].(map[string]any)["references"].([]map[string]any)) != 1 {
		t.Fatal("panel background omitted")
	}
	if all["GFX_unknown"].(map[string]any)["usage_known"] != false {
		t.Fatal("invented use for unknown art")
	}

	top, err := NewIndexWithFallback(map[string][]byte{"interface/replacement.gui": []byte(`guiTypes={containerWindowType={name=demo iconType={name=new spriteType=GFX_new}}}`)}, idx)
	if err != nil {
		t.Fatal(err)
	}
	if top.SpriteUsagePayload([]string{"GFX_action"})["GFX_action"].(map[string]any)["usage_known"] != false {
		t.Fatal("shadowed GUI reference leaked")
	}
}

func TestSpriteUsageAlertFramesAreZeroBasedAndKeepAmbiguousMappingHonest(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"interface/alerts.gfx":                      []byte(`spriteTypes={spriteType={name=GFX_global_alert_icons texturefile="gfx/interface/alerts.dds" noOfFrames=70}}`),
		"localisation/english/alerts_l_english.yml": []byte("l_english:\n alert_select_focus_instant:0 \"Choose a national focus\"\n alert_select_focus_delayed:0 \"No focus is selected\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	use := idx.SpriteUsagePayload([]string{"GFX_global_alert_icons"})["GFX_global_alert_icons"].(map[string]any)
	frames := use["frame_usages"].([]map[string]any)
	if len(frames) != 70 {
		t.Fatalf("frames=%d", len(frames))
	}
	seen := map[int]bool{}
	for _, row := range frames {
		frame := row["frame"].(int)
		if frame < 0 || frame >= 70 || seen[frame] || row["game_frame"] != frame+1 || row["exact_runtime_condition_available"] != false {
			t.Fatalf("bad frame metadata: %+v", row)
		}
		seen[frame] = true
		if frame == 1 && (row["key"] != "alert_select_focus" || row["title"] != "Choose a national focus") {
			t.Fatalf("question mark is not a generic unknown action: %+v", row)
		}
		if row["alert_type"] == 68 && row["confidence"] != "inferred_alert_name" {
			t.Fatal("inference presented as verified name")
		}
	}
}
