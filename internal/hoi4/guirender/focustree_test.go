package guirender

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func focusPack(t *testing.T) map[string][]byte {
	t.Helper()
	manifest, err := json.Marshal(VanillaManifest{
		GameVersion: "1.19.3",
		Textures: map[string][]int{
			"gfx/interface/goals/goal_generic_army.dds":                 {94, 76},
			"gfx/interface/focusview/titlebar/focus_unavailable_bg.dds": {164, 50},
			"gfx/interface/focusview/focus_link_exclusive.dds":          {96, 32},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{
		"interface/focus.gfx": []byte(`spriteTypes={
			spriteType={ name="GFX_goal_army" texturefile="gfx/interface/goals/goal_generic_army.dds" }
			spriteType={ name="GFX_focus_unavailable" texturefile="gfx/interface/focusview/titlebar/focus_unavailable_bg.dds" }
			spriteType={ name="GFX_focus_link_exclusive" texturefile="gfx/interface/focusview/focus_link_exclusive.dds" noOfFrames=3 }
		}`),
		"localisation/english/focus_l_english.yml": []byte(
			"l_english:\n ROOT_FOCUS:0 \"Root Focus\"\n CHILD_FOCUS:0 \"Child Focus\"\n SHARED_FOCUS:0 \"Shared Focus\"\n DEEP_FOCUS:0 \"Deep Focus\"\n"),
		"common/national_focus/demo.txt": []byte(`
			focus_tree = {
				id = demo_focus
				focus = { id = ROOT_FOCUS icon = GFX_goal_army x = 5 y = 0 cost = 10 }
				focus = {
					id = CHILD_FOCUS icon = GFX_goal_army x = -2 y = 1
					relative_position_id = ROOT_FOCUS
					prerequisite = { focus = ROOT_FOCUS }
					mutually_exclusive = { focus = SHARED_FOCUS }
				}
				focus = {
					id = DEEP_FOCUS x = 1 y = 2 relative_position_id = CHILD_FOCUS
					prerequisite = { focus = CHILD_FOCUS focus = SHARED_FOCUS }
				}
				focus = { id = ORPHAN_FOCUS x = 7 y = 3 relative_position_id = NOT_IN_THIS_TREE }
				shared_focus = SHARED_FOCUS
			}
		`),
		"common/national_focus/shared.txt": []byte(`
			shared_focus = { id = SHARED_FOCUS icon = GFX_goal_army x = 9 y = 1 }
			shared_focus = { id = SHARED_CHILD icon = GFX_goal_army x = 0 y = 1
				relative_position_id = SHARED_FOCUS prerequisite = { focus = SHARED_FOCUS } }
			shared_focus = { id = SHARED_GRANDCHILD icon = GFX_goal_army x = 0 y = 1
				prerequisite = { focus = SHARED_CHILD } }
			shared_focus = { id = OTHER_COUNTRY_FOCUS icon = GFX_goal_army x = 40 y = 0 }
		`),
		VanillaManifestPath: manifest,
	}
}

func demoTree(t *testing.T) *FocusTree {
	t.Helper()
	idx, err := NewVanillaIndex(focusPack(t), "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	tree, ok := idx.FocusTreeByID("demo_focus")
	if !ok {
		t.Fatal("tree not indexed")
	}
	return tree
}

func focusByID(tree *FocusTree, id string) *Focus {
	for _, focus := range tree.Focuses {
		if focus.ID == id {
			return focus
		}
	}
	return nil
}

func TestFocusPositionsResolveThroughRelativeChains(t *testing.T) {
	tree := demoTree(t)

	if got := focusByID(tree, "CHILD_FOCUS"); got.X != 3 || got.Y != 1 {
		t.Fatalf("child at %d,%d", got.X, got.Y)
	}
	if got := focusByID(tree, "DEEP_FOCUS"); got.X != 4 || got.Y != 3 {
		t.Fatalf("deep at %d,%d", got.X, got.Y)
	}
}

func TestSharedFocusIsPulledInFromAnotherFile(t *testing.T) {
	tree := demoTree(t)
	shared := focusByID(tree, "SHARED_FOCUS")
	if shared == nil {
		t.Fatal("shared focus missing from the tree")
	}
	if shared.X != 9 || shared.Y != 1 {
		t.Fatalf("shared focus at %d,%d", shared.X, shared.Y)
	}
}

func TestSharedFocusPullsInItsWholeBranch(t *testing.T) {
	tree := demoTree(t)
	for _, id := range []string{"SHARED_CHILD", "SHARED_GRANDCHILD"} {
		if focusByID(tree, id) == nil {
			t.Errorf("%s: branch member missing", id)
		}
	}

	if focusByID(tree, "OTHER_COUNTRY_FOCUS") != nil {
		t.Error("an unrelated shared focus was pulled in")
	}

	if child := focusByID(tree, "SHARED_CHILD"); child.X != 9 || child.Y != 2 {
		t.Errorf("branch child at %d,%d", child.X, child.Y)
	}
}

func TestDanglingAnchorKeepsTheFocusInsteadOfDroppingTheTree(t *testing.T) {

	orphan := focusByID(demoTree(t), "ORPHAN_FOCUS")
	if orphan == nil || orphan.X != 7 || orphan.Y != 3 {
		t.Fatalf("orphan: %#v", orphan)
	}
}

func TestPrerequisitesKeepOrAndAndApart(t *testing.T) {
	deep := focusByID(demoTree(t), "DEEP_FOCUS")
	if len(deep.Prereq) != 1 || len(deep.Prereq[0]) != 2 {
		t.Fatalf("prereq groups: %#v", deep.Prereq)
	}
	child := focusByID(demoTree(t), "CHILD_FOCUS")
	if len(child.Mutex) != 1 || child.Mutex[0] != "SHARED_FOCUS" {
		t.Fatalf("mutex: %#v", child.Mutex)
	}
}

func TestFocusPayloadCarriesArtNamesAndFont(t *testing.T) {
	idx, err := NewVanillaIndex(focusPack(t), "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := idx.FocusTreePayload("demo_focus")
	if !ok {
		t.Fatal("payload missing")
	}
	art := payload["art"].(map[string]any)["unavailable"].(map[string]any)
	if art["vanilla_url"] != "https://obs.example.com/vanilla/gfx/interface/focusview/titlebar/focus_unavailable_bg.png" {
		t.Fatalf("plate art: %#v", art["vanilla_url"])
	}
	if exclusive := payload["art"].(map[string]any)["exclusive"].(map[string]any); exclusive["vanilla_url"] !=
		"https://obs.example.com/vanilla/gfx/interface/focusview/focus_link_exclusive.png" {
		t.Fatalf("exclusive link art: %#v", exclusive)
	}
	focuses := payload["focuses"].([]map[string]any)
	first := focuses[0]
	if first["name"] != "Root Focus" {
		t.Fatalf("focus name not localised: %#v", first["name"])
	}
	icon := first["icon_sprite"].(map[string]any)
	if icon["vanilla_url"] != "https://obs.example.com/vanilla/gfx/interface/goals/goal_generic_army.png" {
		t.Fatalf("icon art: %#v", icon["vanilla_url"])
	}
	if payload["spacing"].(map[string]int)["x"] != FocusSpacingX {
		t.Fatalf("spacing: %#v", payload["spacing"])
	}

	anchors := map[string]any{}
	for _, focus := range focuses {
		anchors[focus["id"].(string)] = focus["relative_to"]
	}
	if anchors["ROOT_FOCUS"] != nil || anchors["DEEP_FOCUS"] != "CHILD_FOCUS" {
		t.Fatalf("relative_to: %#v", anchors)
	}
}

func TestUnknownTreeIsNotFound(t *testing.T) {
	idx, err := NewVanillaIndex(focusPack(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.FocusTreeByID("no_such_tree"); ok {
		t.Fatal("unknown tree must not resolve")
	}
}

func TestFocusTreeChoicesBreakTiesOnSizeAndHonourLayers(t *testing.T) {
	tree := func(id, tag string, focuses int) string {
		body := fmt.Sprintf("focus_tree = { id = %s country = { factor = 0 modifier = { add = 10 tag = %s } }", id, tag)
		for i := 0; i < focuses; i++ {
			body += fmt.Sprintf(" focus = { id = %s_%d x = %d y = 0 }", id, i, i*2)
		}
		return body + " }"
	}
	base, err := NewIndex(map[string][]byte{
		"common/national_focus/german.txt": []byte(tree("german_focus", "GER", 3)),
		"common/national_focus/other.txt":  []byte(tree("austrian_focus", "AUS", 2)),
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"common/national_focus/custom.txt": []byte(tree("ger_custom_focus", "GER", 0)),

		"common/national_focus/other.txt": []byte(tree("austrian_focus", "HUN", 2)),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	if got := project.FocusTreeForTag("GER"); got != "german_focus" {
		t.Fatalf("GER tree = %q, want the tree with focuses over the empty one", got)
	}
	choices := project.FocusTreeChoices("GER")
	if len(choices) != 2 || choices[0].ID != "german_focus" || choices[0].Focuses != 3 || choices[1].ID != "ger_custom_focus" {
		t.Fatalf("GER choices = %+v", choices)
	}
	if got := project.FocusTreeForTag("AUS"); got != "" {
		t.Fatalf("AUS still gets the shadowed tree: %q", got)
	}
	if got := project.FocusTreeForTag("HUN"); got != "austrian_focus" {
		t.Fatalf("HUN tree = %q", got)
	}
}

func TestFocusTreeChoicesReachTheGenericTree(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/national_focus/generic.txt": []byte(`focus_tree = { id = generic_focus country = { factor = 1 } default = yes
			focus = { id = industrial_effort x = 0 y = 0 } }`),
		"common/national_focus/germany.txt": []byte(`focus_tree = { id = german_focus country = { factor = 0 modifier = { add = 10 tag = GER } }
			focus = { id = GER_rearm x = 0 y = 0 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.FocusTreeForTag("LUX"); got != "generic_focus" {
		t.Fatalf("LUX tree = %q, want the generic tree", got)
	}
	choices := idx.FocusTreeChoices("GER")
	if len(choices) != 2 || choices[0].ID != "german_focus" || choices[1].ID != "generic_focus" {
		t.Fatalf("GER choices = %+v, want its own tree first and the generic one after", choices)
	}

	fallback, err := NewIndex(map[string][]byte{
		"common/national_focus/generic.txt": []byte(`focus_tree = { id = mod_generic country = { factor = 0 } default = yes
			focus = { id = mod_start x = 0 y = 0 } }`),
		"common/national_focus/germany.txt": []byte(`focus_tree = { id = german_focus country = { factor = 0 modifier = { add = 10 tag = GER } }
			focus = { id = GER_rearm x = 0 y = 0 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fallback.FocusTreeForTag("LUX"); got != "mod_generic" {
		t.Fatalf("LUX tree = %q, want the default tree when nothing scores", got)
	}
	if choices := fallback.FocusTreeChoices("GER"); len(choices) != 1 || choices[0].ID != "german_focus" {
		t.Fatalf("GER choices = %+v: a default tree is not offered where a tree scores", choices)
	}
}

func TestFocusTreeShortcuts(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/national_focus/germany.txt": []byte(`focus_tree = { id = german_focus
			country = { factor = 0 modifier = { add = 10 tag = GER } }
			shortcut = { name = GER_oppose_shortcut target = GER_oppose_ww scroll_wheel_factor = 0.485 trigger = { has_dlc = "Gotterdammerung" } }
			shortcut = { name = GER_oppose_shortcut target = GER_oppose scroll_wheel_factor = 0.55 trigger = { NOT = { has_dlc = "Gotterdammerung" } } }
			shortcut = { name = GER_historical_shortcut target = GER_rhineland trigger = { always = yes } }
			shortcut = { name = GER_hidden_shortcut target = GER_rhineland trigger = { always = no } }
			shortcut = { name = GER_missing_shortcut target = GER_no_such_focus }
			focus = { id = GER_oppose_ww x = 0 y = 0 }
			focus = { id = GER_oppose x = 2 y = 0 }
			focus = { id = GER_rhineland x = 4 y = 0 } }`),
		"localisation/english/shortcuts_l_english.yml": []byte("l_english:\n GER_historical_shortcut:0 \"Historical Path\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	tree, ok := idx.FocusTreeByID("german_focus")
	if !ok {
		t.Fatal("tree missing")
	}
	got := make([]string, 0, len(tree.Shortcuts))
	for _, shortcut := range tree.Shortcuts {
		got = append(got, fmt.Sprintf("%s>%s@%g:%s", shortcut.Name, shortcut.Target, shortcut.Zoom, shortcut.Label))
	}
	want := []string{
		"GER_oppose_shortcut>GER_oppose_ww@0.485:GER_oppose_shortcut",
		"GER_historical_shortcut>GER_rhineland@0:Historical Path",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("shortcuts = %v, want %v", got, want)
	}
	payload, _ := idx.FocusTreePayload("german_focus")
	if shortcuts, _ := payload["shortcuts"].([]*FocusShortcut); len(shortcuts) != 2 {
		t.Fatalf("payload shortcuts = %#v", payload["shortcuts"])
	}
}

func TestFocusTreeChoicesOpenScriptedTriggers(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/national_focus/generic.txt": []byte(`focus_tree = { id = generic_focus country = { factor = 1 } default = yes
			focus = { id = industrial_effort x = 0 y = 0 } }`),
		"common/national_focus/norway.txt": []byte(`focus_tree = { id = norwegian_focus
			country = { factor = 0 modifier = { add = 10 NOR_AAT = yes } }
			focus = { id = NOR_start x = 0 y = 0 } }`),
		"common/scripted_triggers/NOR_scripted_triggers.txt": []byte(`NOR_AAT = { original_tag = NOR has_dlc = "Arms Against Tyranny" }
			LOOP_A = { LOOP_B = yes }
			LOOP_B = { LOOP_A = yes }`),
		"common/national_focus/loop.txt": []byte(`focus_tree = { id = loop_focus
			country = { factor = 0 modifier = { add = 10 LOOP_A = yes } }
			focus = { id = LOOP_start x = 0 y = 0 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.FocusTreeForTag("NOR"); got != "norwegian_focus" {
		t.Fatalf("NOR tree = %q, want the tree its scripted trigger names", got)
	}
	if got := idx.FocusTreeForTag("SWE"); got != "generic_focus" {
		t.Fatalf("SWE tree = %q", got)
	}
}
