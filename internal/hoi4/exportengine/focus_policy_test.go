package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func focusPolicySnapshot(projectJSON, treeRow string, nodeRows ...string) Snapshot {
	nodes := make([]json.RawMessage, 0, len(nodeRows))
	for _, row := range nodeRows {
		nodes = append(nodes, json.RawMessage(row))
	}
	return Snapshot{
		ProjectID: 31,
		Project:   json.RawMessage(projectJSON),
		Tables: map[string][]json.RawMessage{
			"focus_trees": {json.RawMessage(treeRow)},
			"focus_nodes": nodes,
		},
	}
}

const focusPolicyStandalone = `{"id":31,"name":"Demo","country_tag":"USA","export_languages":["english"],"dependencies":[]}`
const focusPolicySubmod = `{"id":31,"name":"Demo","country_tag":"USA","export_languages":["english"],"dependencies":["The Road to 56"]}`

func focusScript(t *testing.T, artifact NativeArtifact) (string, string) {
	t.Helper()
	var path, body string
	found := 0
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/common/national_focus/") {
			found++
			path, body = entry.Path, string(entry.Bytes)
		}
	}
	if found != 1 {
		var paths []string
		for _, entry := range artifact.Entries {
			paths = append(paths, entry.Path)
		}
		t.Fatalf("want exactly 1 national_focus file, got %d; entries=%v", found, paths)
	}
	return path, body
}

func TestVanillaFocusIconSurvivesIntoTheEmittedScript(t *testing.T) {
	artifact, err := BuildNativeShadow(focusPolicySnapshot(
		focusPolicyStandalone,
		`{"id":1,"tree_id":"usa_new_tree","country_tag":"USA","tree_kind":"primary","source":"created"}`,
		`{"id":2,"focus_tree_id":1,"focus_id":"USA_vanilla_icon","name_en":"V","icon":"GFX_focus_usa_guarantee_the_american_dream","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
		`{"id":3,"focus_tree_id":1,"focus_id":"USA_vanilla_goal","name_en":"G","icon":"GFX_goal_generic_national_unity","text_icon":"GFX_focus_generic_diplomatic_treaty","cost":10,"x":2,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, body := focusScript(t, artifact)
	for _, want := range []string{
		"icon = GFX_focus_usa_guarantee_the_american_dream",
		"icon = GFX_goal_generic_national_unity",
		"text_icon = GFX_focus_generic_diplomatic_treaty",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("vanilla sprite rewritten to a placeholder — missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, "GFX_goal_unknown") {
		t.Fatalf("emitted the ? placeholder for a vanilla sprite:\n%s", body)
	}
}

func TestUnregisteredFocusIconStillFallsBackToPlaceholder(t *testing.T) {
	artifact, err := BuildNativeShadow(focusPolicySnapshot(
		focusPolicyStandalone,
		`{"id":1,"tree_id":"usa_new_tree","country_tag":"USA","tree_kind":"primary","source":"created"}`,
		`{"id":2,"focus_tree_id":1,"focus_id":"USA_made_up","name_en":"M","icon":"GFX_focus_never_registered_anywhere","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, body := focusScript(t, artifact)
	if strings.Contains(body, "GFX_focus_never_registered_anywhere") {
		t.Fatalf("emitted a reference to a sprite no .gfx defines:\n%s", body)
	}
	if !strings.Contains(body, "icon = GFX_goal_unknown") {
		t.Fatalf("unresolvable icon must fall back to GFX_goal_unknown:\n%s", body)
	}
}

func TestSubmodBaseModIconPassesThroughUngated(t *testing.T) {
	const treeRow = `{"id":1,"tree_id":"usa_new_tree","country_tag":"USA","tree_kind":"primary","source":"created"}`
	const nodeRow = `{"id":2,"focus_tree_id":1,"focus_id":"USA_basemod_icon","name_en":"B","icon":"GFX_focus_TNO_burgundian_system","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`

	submod, err := BuildNativeShadow(focusPolicySnapshot(focusPolicySubmod, treeRow, nodeRow), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, body := focusScript(t, submod); !strings.Contains(body, "icon = GFX_focus_TNO_burgundian_system") {
		t.Fatalf("submod base-mod sprite was gated away:\n%s", body)
	}

	standalone, err := BuildNativeShadow(focusPolicySnapshot(focusPolicyStandalone, treeRow, nodeRow), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, body := focusScript(t, standalone); !strings.Contains(body, "icon = GFX_goal_unknown") {
		t.Fatalf("standalone project must still gate an unknown sprite:\n%s", body)
	}
}

func TestImportedTreeOverridesTheVanillaFileNameItCameFrom(t *testing.T) {
	artifact, err := BuildNativeShadow(focusPolicySnapshot(
		focusPolicyStandalone,
		`{"id":1,"tree_id":"usa_focus","country_tag":"USA","tree_kind":"primary","source":"created","source_file":"common/national_focus/usa.txt"}`,
		`{"id":2,"focus_tree_id":1,"focus_id":"USA_x","name_en":"X","icon":"GFX_goal_unknown","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := focusScript(t, artifact)
	if path != "Demo/common/national_focus/usa.txt" {
		t.Fatalf("path=%q — a TAG_ prefix ADDS a second tree instead of overriding vanilla's", path)
	}
}

func TestImportedTreeWithNonVanillaIdKeepsItsOwnFileName(t *testing.T) {
	artifact, err := BuildNativeShadow(focusPolicySnapshot(
		focusPolicyStandalone,
		`{"id":1,"tree_id":"r56_usa_extra","country_tag":"USA","tree_kind":"primary","source":"created","source_file":"r56_usa_extra.txt"}`,
		`{"id":2,"focus_tree_id":1,"focus_id":"USA_y","name_en":"Y","icon":"GFX_goal_unknown","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := focusScript(t, artifact)
	if path != "Demo/common/national_focus/r56_usa_extra.txt" {
		t.Fatalf("path=%q want the author's own basename, re-rooted", path)
	}
}

func TestCreatedTreeKeepsTheNamespacedAdditionPath(t *testing.T) {
	artifact, err := BuildNativeShadow(focusPolicySnapshot(
		focusPolicyStandalone,
		`{"id":1,"tree_id":"new_tree","country_tag":"USA","tree_kind":"primary","source":"created"}`,
		`{"id":2,"focus_tree_id":1,"focus_id":"USA_z","name_en":"Z","icon":"GFX_goal_unknown","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := focusScript(t, artifact)
	if path != "Demo/common/national_focus/USA_new_tree.txt" {
		t.Fatalf("path=%q want the namespaced addition path", path)
	}
}

func TestCreatedTreeAlreadyNamespacedIsNotPrefixedTwice(t *testing.T) {
	for _, want := range []struct{ project, tree, path string }{
		{focusPolicyStandalone,
			`{"id":1,"tree_id":"usa_second_tree","country_tag":"USA","tree_kind":"primary","source":"created"}`,
			"Demo/common/national_focus/usa_second_tree.txt"},
		{`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["english"]}`,
			`{"id":1,"tree_id":"ger_demo_tree","country_tag":"GER","tree_kind":"primary","source":"created"}`,
			"Demo/common/national_focus/ger_demo_tree.txt"},
	} {
		artifact, err := BuildNativeShadow(focusPolicySnapshot(want.project, want.tree,
			`{"id":2,"focus_tree_id":1,"focus_id":"X_w","name_en":"W","icon":"GFX_goal_unknown","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
		), nil, false)
		if err != nil {
			t.Fatal(err)
		}
		path, _ := focusScript(t, artifact)
		if path != want.path {
			t.Fatalf("path=%q want %q — the id already starts with the namespace", path, want.path)
		}
	}
}

func TestUnsafeSourceFileDegradesToTheAdditionPath(t *testing.T) {
	for _, sourceFile := range []string{
		"../../secrets.txt",
		"common/ideas/usa.txt",
		"common/national_focus/deep/usa.txt",
		"usa.json",
		"/etc/usa.txt",
		"C:/usa.txt",
	} {
		artifact, err := BuildNativeShadow(focusPolicySnapshot(
			focusPolicyStandalone,
			`{"id":1,"tree_id":"new_tree","country_tag":"USA","tree_kind":"primary","source":"created","source_file":`+jsonString(sourceFile)+`}`,
			`{"id":2,"focus_tree_id":1,"focus_id":"USA_z","name_en":"Z","icon":"GFX_goal_unknown","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`,
		), nil, false)
		if err != nil {
			t.Fatalf("source_file=%q: %v", sourceFile, err)
		}
		path, _ := focusScript(t, artifact)
		if path != "Demo/common/national_focus/USA_new_tree.txt" {
			t.Fatalf("source_file=%q produced path=%q", sourceFile, path)
		}
	}
}

func jsonString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
