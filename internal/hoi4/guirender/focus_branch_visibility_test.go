package guirender

import (
	"os"
	"testing"
)

func TestFocusBranchVisibility(t *testing.T) {
	files := focusPack(t)
	files["common/national_focus/branches.txt"] = []byte(`focus_tree = { id = branches
 focus = { id = old x = 5 allow_branch = { NOT = { has_dlc = "Gotterdammerung" } if = { limit = { has_country_flag = unknown } always = no } } }
 focus = { id = old_child prerequisite = { focus = old } }
 focus = { id = old_grandchild prerequisite = { focus = old_child } }
 focus = { id = new x = 5 allow_branch = { has_dlc = "Gotterdammerung" has_country_flag = unknown } }
 focus = { id = new_child prerequisite = { focus = new } }
 focus = { id = join relative_position_id = old x = 4 y = 3 prerequisite = { focus = old focus = new } }
 focus = { id = both prerequisite = { focus = old } prerequisite = { focus = new } }
 focus = { id = unknown allow_branch = { has_country_flag = unknown } }
 focus = { id = unknown_or allow_branch = { OR = { has_dlc = "Gotterdammerung" has_country_flag = unknown } } }
 focus = { id = missing prerequisite = { focus = missing_parent } }
 shortcut = { name = old target = old }
 shortcut = { name = new target = new }
 }`)
	base, err := NewVanillaIndex(files, "https://example.test/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(nil, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, all := range []bool{true, false, true} {
		idx.SetOwnedDLCs(nil, all)
		baseTree, _ := base.FocusTreeByID("branches")
		if focusByID(baseTree, "new") == nil || focusByID(baseTree, "old") != nil {
			t.Fatal("project filtering mutated shared vanilla")
		}
		tree, ok := idx.FocusTreeByID("branches")
		if !ok {
			t.Fatal("missing tree")
		}
		for id, want := range map[string]bool{"old": !all, "old_child": !all, "old_grandchild": !all, "new": all, "new_child": all, "join": true, "both": false, "unknown": true, "unknown_or": true, "missing": true} {
			if (focusByID(tree, id) != nil) != want {
				t.Errorf("all=%v focus=%s want visible=%v", all, id, want)
			}
		}
		join := focusByID(tree, "join")
		if join.X != 9 || join.Y != 3 {
			t.Errorf("anchor changed: %+v", join)
		}
		if len(tree.Shortcuts) != 1 {
			t.Errorf("shortcuts=%v", tree.Shortcuts)
		}
	}
}

func TestFocusBranchRealGermanTree(t *testing.T) {
	path := os.Getenv("HOI4_GERMAN_FOCUS_SOURCE")
	if path == "" {
		t.Skip("optional real game fixture")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	files := focusPack(t)
	files["common/national_focus/germany.txt"] = raw
	idx, err := NewVanillaIndex(files, "https://example.test/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	for _, all := range []bool{true, false} {
		idx.SetOwnedDLCs(nil, all)
		tree, ok := idx.FocusTreeByID("german_focus")
		if !ok {
			t.Fatal("missing german tree")
		}
		if (focusByID(tree, "GER_oppose_hitler_ww") != nil) != all || (focusByID(tree, "GER_oppose_hitler") != nil) == all {
			t.Errorf("wrong DLC branches all=%v", all)
		}
		positions := map[[2]int]string{}
		overlaps := 0
		for _, f := range tree.Focuses {
			p := [2]int{f.X, f.Y}
			if prev := positions[p]; prev != "" {
				overlaps++
				t.Logf("overlap %s / %s at %v", prev, f.ID, p)
			}
			positions[p] = f.ID
		}
		t.Logf("all=%v focuses=%d overlaps=%d", all, len(tree.Focuses), overlaps)
		if overlaps != 0 {
			t.Errorf("remaining overlaps: %d", overlaps)
		}
	}
}
