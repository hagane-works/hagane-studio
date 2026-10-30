package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResolveIdeaPictureRefsThreeWayDecision(t *testing.T) {
	rows, err := nativeRows([]json.RawMessage{
		json.RawMessage(`{"id":1,"idea_id":"GER_uploaded","picture":"GFX_idea_whatever","source":"created"}`),
		json.RawMessage(`{"id":2,"idea_id":"GER_vanilla","picture":"GFX_idea_generic_fascism_drift_2","source":"created"}`),
		json.RawMessage(`{"id":3,"idea_id":"GER_vanilla_bare","picture":"new_economic_policy","source":"created"}`),
		json.RawMessage(`{"id":4,"idea_id":"GER_custom","picture":"GFX_idea_my_custom_art","source":"created"}`),
		json.RawMessage(`{"id":5,"idea_id":"GER_blank","picture":"   ","source":"created"}`),
		json.RawMessage(`{"id":6,"idea_id":"GER_none","source":"created"}`),
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	refs := map[int64]string{1: "GER_ger_uploaded"}
	sink := &imageSink{}
	resolveIdeaPictureRefs(rows, refs, false, nil, sink)
	if refs[1] != "GER_ger_uploaded" {
		t.Fatalf("uploaded ref must win: %#v", refs)
	}
	if refs[2] != "generic_fascism_drift_2" || refs[3] != "new_economic_policy" {
		t.Fatalf("vanilla names must be kept (prefix stripped): %#v", refs)
	}
	if _, ok := refs[4]; ok {
		t.Fatalf("unknown custom name must be dropped from refs: %#v", refs)
	}
	if _, ok := refs[5]; ok {
		t.Fatalf("blank picture must not produce a ref: %#v", refs)
	}
	if len(sink.warnings) != 1 || !strings.Contains(sink.warnings[0], `"GER_custom"`) || !strings.Contains(sink.warnings[0], `"GFX_idea_my_custom_art"`) {
		t.Fatalf("exactly one drop warning naming idea+picture expected, got %#v", sink.warnings)
	}

	refs = map[int64]string{}
	sink = &imageSink{}
	resolveIdeaPictureRefs(rows, refs, true, nil, sink)
	if refs[4] != "my_custom_art" || refs[2] != "generic_fascism_drift_2" {
		t.Fatalf("submod must keep every typed name: %#v", refs)
	}
	if len(sink.warnings) != 0 {
		t.Fatalf("submod must not warn: %#v", sink.warnings)
	}

	resolveIdeaPictureRefs(rows, map[int64]string{}, false, nil, nil)
}

func ideaScriptAndWarnings(t *testing.T, project string, ideaRow string) (string, []string) {
	t.Helper()
	snapshot := Snapshot{ProjectID: 42, Project: []byte(project), Tables: map[string][]json.RawMessage{
		"ideas": {json.RawMessage(ideaRow)},
	}}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range artifact.Entries {
		if entry.Path == "Demo/common/ideas/ger_ger.txt" {
			return string(entry.Bytes), artifact.Warnings
		}
	}
	t.Fatalf("idea script missing: %#v", artifact.Entries)
	return "", nil
}

const ideaTestProject = `{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`
const ideaTestSubmod = `{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"],"dependencies":["TNO"]}`

func TestBuildNativeShadowKeepsVanillaIdeaPictureReference(t *testing.T) {
	script, warnings := ideaScriptAndWarnings(t, ideaTestProject,
		`{"id":1,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"GFX_idea_generic_fascism_drift_2","source":"created"}`)
	if !strings.Contains(script, "picture = generic_fascism_drift_2") {
		t.Fatalf("vanilla idea picture reference was dropped:\n%s", script)
	}
	for _, w := range warnings {
		if strings.Contains(w, "Idea picture dropped") {
			t.Fatalf("vanilla name must not be reported as dropped: %q", w)
		}
	}
}

func TestBuildNativeShadowKeepsAnyIdeaPictureForSubmod(t *testing.T) {
	script, warnings := ideaScriptAndWarnings(t, ideaTestSubmod,
		`{"id":1,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"GFX_idea_GER_to_banish_want","source":"created"}`)
	if !strings.Contains(script, "picture = GER_to_banish_want") {
		t.Fatalf("submod base-mod sprite reference was dropped:\n%s", script)
	}
	for _, w := range warnings {
		if strings.Contains(w, "Idea picture dropped") {
			t.Fatalf("submod name must not be reported as dropped: %q", w)
		}
	}
}

func TestBuildNativeShadowWarnsWhenIdeaPictureIsDropped(t *testing.T) {
	script, warnings := ideaScriptAndWarnings(t, ideaTestProject,
		`{"id":1,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"GFX_idea_my_custom_art","source":"created"}`)
	if strings.Contains(script, "picture =") {
		t.Fatalf("unknown non-uploaded picture must be dropped (would be a '?' in game):\n%s", script)
	}
	var found string
	for _, w := range warnings {
		if strings.Contains(w, `"GER_demo"`) && strings.Contains(w, `"GFX_idea_my_custom_art"`) {
			found = w
		}
	}
	if found == "" {
		t.Fatalf("idea picture was dropped silently; warnings=%#v", warnings)
	}
}

func TestBuildNativeShadowKeepsIdeaPictureTheProjectDefines(t *testing.T) {
	gfx, _ := json.Marshal(map[string]any{
		"id": 7, "dest_path": "interface/sg119_authored_presentation.gfx", "is_binary": false, "raw_kind": "verbatim_override", "source": "created",
		"content": "spriteTypes = {\n\tSpriteType = {\n\t\tname = \"GFX_idea_SG_ancient_march\"\n\t\ttexturefile = \"gfx/interface/sg119/ancient_march_modifier.dds\"\n\t}\n}\n",
	})
	snapshot := Snapshot{ProjectID: 42, Project: []byte(ideaTestProject), Tables: map[string][]json.RawMessage{
		"ideas":             {json.RawMessage(`{"id":1,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name_en":"Demo","picture":"SG_ancient_march","source":"created"}`)},
		"project_raw_files": {gfx},
	}}
	if got := projectSpriteNames(snapshot, false); !got["gfx_idea_sg_ancient_march"] {
		t.Fatalf("own sprites = %v", got)
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var script string
	for _, entry := range artifact.Entries {
		if entry.Path == "Demo/common/ideas/ger_ger.txt" {
			script = string(entry.Bytes)
		}
	}
	if !strings.Contains(script, "picture = SG_ancient_march") {
		t.Fatalf("the project's own sprite was dropped:\n%s", script)
	}
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "Idea picture dropped") {
			t.Fatalf("the project's own sprite reported as dropped: %q", w)
		}
	}
}
