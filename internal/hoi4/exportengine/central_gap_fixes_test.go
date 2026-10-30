package exportengine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEmptyShadowRawFileShipsInsteadOfFailingTheExport(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"common/ideas/suppressed.txt","content":"","source":"created","raw_kind":"empty_shadow","is_binary":false,"generate_replace_path":false}`,
		`{"id":2,"dest_path":"common/ideas/normal.txt","content":"ideas = { }","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("empty_shadow must not fail the export: %v", err)
	}
	entry, ok := findEntry(artifact, "Demo/common/ideas/suppressed.txt")
	if !ok {
		var paths []string
		for _, e := range artifact.Entries {
			paths = append(paths, e.Path)
		}
		t.Fatalf("empty_shadow file missing — the empty file IS the suppression; entries=%v", paths)
	}
	if len(entry.Bytes) != 0 {
		t.Fatalf("empty_shadow must be zero bytes, got %q", entry.Bytes)
	}
	if entry.Origin != "raw_empty_shadow" {
		t.Fatalf("origin=%q want raw_empty_shadow", entry.Origin)
	}

	if descriptor := descriptorText(t, artifact); strings.Contains(descriptor, "common/ideas") {
		t.Fatalf("empty_shadow must not contribute a replace_path:\n%s", descriptor)
	}

	if _, ok := findEntry(artifact, "Demo/common/ideas/normal.txt"); !ok {
		t.Fatal("verbatim_override sibling was dropped")
	}
}

func TestUnknownRawKindStillFailsClosed(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"common/ideas/x.txt","content":"","source":"created","raw_kind":"something_new","is_binary":false}`,
	)
	if _, err := BuildNativeShadow(snapshot, nil, false); err == nil {
		t.Fatal("an unrecognised raw_kind must still fail closed")
	}
}

func brokenBlob(string, int64) ([]byte, error) { return []byte("not-an-image"), nil }

var errBlobMissing = errors.New("object not found")

func TestUnfetchablePortraitSkipsOnlyThatPortrait(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"characters": {
				json.RawMessage(`{"id":11,"tag":"GER_leader","country_tag":"GER","name":"Leader","source":"created","replaces_vanilla_id":false}`),
			},
			"entity_obs_paths": {
				json.RawMessage(`{"id":21,"entity_type":"character_portrait","entity_id":11,"obs_key":"p/gone.png"}`),
			},
		},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false,
		func(string, int64) ([]byte, error) { return nil, errBlobMissing })
	if err != nil {
		t.Fatalf("one unfetchable portrait must not fail the export: %v", err)
	}
	if !warnsAbout(artifact, "character_portrait") {
		t.Fatalf("no warning recorded; warnings=%v", artifact.Warnings)
	}

	if !anyPathContains(artifact, "/common/characters/") {
		t.Fatal("character script was dropped along with its portrait")
	}
	if anyPathContains(artifact, "/gfx/portraits/") {
		t.Fatal("portrait asset written despite an unfetchable object")
	}
}

func TestBrokenTechnologyIconSkipsOnlyThatIcon(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","start_year":1936}`),
		Tables: map[string][]json.RawMessage{
			"technologies": {
				json.RawMessage(`{"id":1,"tech_id":"GER_broken","tech_line":"industry","source":"created","name":"Broken"}`),
			},
			"entity_obs_paths": {
				json.RawMessage(`{"id":31,"entity_type":"tech_icon","entity_id":1,"obs_key":"tech/broken.png"}`),
			},
		},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, brokenBlob)
	if err != nil {
		t.Fatalf("one unreadable tech icon must not fail the export: %v", err)
	}
	if !warnsAbout(artifact, "technology GER_broken icon") {
		t.Fatalf("no warning recorded; warnings=%v", artifact.Warnings)
	}

	if anyPathContains(artifact, "_technologies.gfx") {
		t.Fatal("sprite block emitted for an icon that was skipped")
	}
	if !anyPathContains(artifact, "/common/technologies/") {
		t.Fatal("technology script was dropped along with its icon")
	}
}

func TestBrokenFocusIconSkipsOnlyThatIcon(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"focus_trees":       {json.RawMessage(`{"id":1,"tree_id":"ger_demo_tree","country_tag":"GER","tree_kind":"primary","source":"created"}`)},
			"focus_nodes":       {json.RawMessage(`{"id":2,"focus_tree_id":1,"focus_id":"GER_demo_focus","name_en":"Demo","icon":"GFX_focus_ger_demo_focus","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[]}`)},
			"project_obs_icons": {json.RawMessage(`{"id":3,"project_id":42,"gfx_name":"GFX_focus_ger_demo_focus","obs_key":"focus/broken.png"}`)},
		},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, brokenBlob)
	if err != nil {
		t.Fatalf("one unreadable focus icon must not fail the export: %v", err)
	}
	if !warnsAbout(artifact, "focus icon GFX_focus_ger_demo_focus") {
		t.Fatalf("no warning recorded; warnings=%v", artifact.Warnings)
	}
	if !anyPathContains(artifact, "/common/national_focus/") {
		t.Fatal("focus tree was dropped along with its icon")
	}
}

func TestMissingResolverStillFailsClosedOnAssetPaths(t *testing.T) {
	cases := map[string]Snapshot{
		"portrait": {
			ProjectID: 42,
			Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["english"]}`),
			Tables: map[string][]json.RawMessage{
				"characters":       {json.RawMessage(`{"id":11,"tag":"GER_leader","country_tag":"GER","name":"Leader","source":"created","replaces_vanilla_id":false}`)},
				"entity_obs_paths": {json.RawMessage(`{"id":21,"entity_type":"character_portrait","entity_id":11,"obs_key":"p/x.png"}`)},
			},
		},
		"tech_icon": {
			ProjectID: 42,
			Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","start_year":1936}`),
			Tables: map[string][]json.RawMessage{
				"technologies":     {json.RawMessage(`{"id":1,"tech_id":"GER_x","tech_line":"industry","source":"created","name":"X"}`)},
				"entity_obs_paths": {json.RawMessage(`{"id":31,"entity_type":"tech_icon","entity_id":1,"obs_key":"tech/x.png"}`)},
			},
		},
	}
	for name, snapshot := range cases {
		if _, err := BuildNativeShadow(snapshot, nil, false); err == nil {
			t.Fatalf("%s: a nil resolver must fail closed, not degrade", name)
		}
	}
}

func TestReplacementLeaderGetsLocalisationNotABareKey(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {json.RawMessage(`{"id":1,"tag":"FRA","name":"France","source":"created","ideology":"neutrality"}`)},
			"characters": {
				json.RawMessage(`{"id":11,"tag":"FRA_petain","country_tag":"FRA","name":"Petain","name_en":"Philippe Petain","role_type":"country_leader","ideology":"despotism","source":"created","replaces_vanilla_id":true}`),
				json.RawMessage(`{"id":12,"tag":"FRA_new_general","country_tag":"FRA","name":"New General","name_en":"New General","source":"created","replaces_vanilla_id":false}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var scripts, loc string
	for _, entry := range artifact.Entries {
		switch {
		case strings.Contains(entry.Path, "/common/characters/"):
			scripts += string(entry.Bytes)
		case strings.Contains(entry.Path, "_characters_l_"):
			loc += string(entry.Bytes)
		}
	}
	if !strings.Contains(scripts, "FRA_petain") {
		t.Fatalf("replacement leader has no character definition:\n%s", scripts)
	}
	if !strings.Contains(loc, "FRA_petain") {
		t.Fatalf("replacement leader has no localisation — the game would show the raw key:\n%s", loc)
	}

	for _, entry := range artifact.Entries {
		if !strings.Contains(entry.Path, "/history/countries/") {
			continue
		}
		if strings.Contains(string(entry.Bytes), "recruit_character = FRA_petain") {
			t.Fatalf("a replacement leader must never be recruited again:\n%s", entry.Bytes)
		}
	}

	var history string
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/history/countries/") {
			history += string(entry.Bytes)
		}
	}
	if !strings.Contains(history, "recruit_character = FRA_new_general") {
		t.Fatalf("new character lost its recruit line:\n%s", history)
	}
}

func warnsAbout(artifact NativeArtifact, needle string) bool {
	for _, warning := range artifact.Warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

func anyPathContains(artifact NativeArtifact, needle string) bool {
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, needle) {
			return true
		}
	}
	return false
}
