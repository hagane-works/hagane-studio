package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func onActionsSnapshot() Snapshot {
	return Snapshot{
		ProjectID: 77,
		Project:   []byte(`{"id":77,"name":"Demo","country_tag":"GER","start_year":1936,"export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"GER","name":"Germany","source":"created","ideology":"fascism"}`),
				json.RawMessage(`{"id":2,"tag":"ITA","name":"Italy","source":"created","ideology":"fascism"}`),
			},
			"event_namespaces": {
				json.RawMessage(`{"id":5,"namespace":"ger_spine","source":"created"}`),
			},
			"events": {

				json.RawMessage(`{"id":31,"namespace_id":5,"event_number":1,"country_tag":"GER","auto_fire_on_startup":true,"auto_fire_after_days":0,"on_action_attach":"","source":"created"}`),

				json.RawMessage(`{"id":32,"namespace_id":5,"event_number":2,"country_tag":"GER","auto_fire_on_startup":false,"auto_fire_after_days":30,"on_action_attach":"","source":"created"}`),

				json.RawMessage(`{"id":33,"namespace_id":5,"event_number":3,"country_tag":"GER","auto_fire_on_startup":false,"auto_fire_after_days":0,"on_action_attach":"on_unit_leader_promote_from_ranks","source":"created"}`),

				json.RawMessage(`{"id":34,"namespace_id":5,"event_number":4,"country_tag":"GER","auto_fire_on_startup":false,"auto_fire_after_days":0,"on_action_attach":"on_subject_annexed_by_overlord","source":"created"}`),

				json.RawMessage(`{"id":35,"namespace_id":5,"event_number":9,"country_tag":"GER","auto_fire_on_startup":false,"auto_fire_after_days":0,"on_action_attach":"","source":"created"}`),

				json.RawMessage(`{"id":36,"namespace_id":5,"event_number":11,"country_tag":"ITA","auto_fire_on_startup":true,"auto_fire_after_days":0,"on_action_attach":"","source":"imported"}`),
			},
			"super_events": {

				json.RawMessage(`{"id":41,"country_tag":"GER","number":7,"title_en":"Victory","auto_fire_on_startup":false,"auto_fire_after_days":652,"source":"created"}`),
			},
			"ideas": {
				json.RawMessage(`{"id":51,"idea_id":"ger_spirit","idea_type":"country","country_tag":"GER","name_en":"Spirit","auto_attach_on_start":true,"source":"created"}`),

				json.RawMessage(`{"id":52,"idea_id":"ger_endgame","idea_type":"country","country_tag":"GER","name_en":"Endgame","auto_attach_on_start":false,"source":"created"}`),

				json.RawMessage(`{"id":53,"idea_id":"ger_tank_co","idea_type":"tank_manufacturer","country_tag":"GER","name_en":"Tanks","auto_attach_on_start":true,"source":"created"}`),
			},
		},
	}
}

func onActionsFile(t *testing.T, artifact NativeArtifact, ctag string) string {
	t.Helper()
	suffix := "/common/on_actions/" + ctag + "_on_actions.txt"
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, suffix) {
			return string(entry.Bytes)
		}
	}
	var paths []string
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "on_actions") {
			paths = append(paths, entry.Path)
		}
	}
	t.Fatalf("no artifact entry ending in %q; on_actions entries=%v", suffix, paths)
	return ""
}

func hasOnActionsFile(artifact NativeArtifact, ctag string) bool {
	suffix := "/common/on_actions/" + ctag + "_on_actions.txt"
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, suffix) {
			return true
		}
	}
	return false
}

func TestBuildNativeShadowEmitsOnActionsFile(t *testing.T) {
	artifact, err := BuildNativeShadow(onActionsSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	body := onActionsFile(t, artifact, "GER")

	if !strings.HasPrefix(body, "on_actions = {") || !strings.HasSuffix(strings.TrimRight(body, "\n"), "}") {
		t.Fatalf("body is not one on_actions block:\n%s", body)
	}

	if !strings.Contains(body, "on_startup = {") {
		t.Fatalf("no on_startup hook:\n%s", body)
	}
	if !strings.Contains(body, "GER = {") {
		t.Fatalf("on_startup body is not manually scoped to GER:\n%s", body)
	}

	for _, want := range []string{
		"country_event = ger_spine.1",
		"country_event = { id = ger_spine.2 days = 30 }",
		"country_event = { id = ger_se.7 days = 652 }",
		"on_unit_leader_promote_from_ranks_veteran = {",
		"on_unit_leader_promote_from_ranks_green = {",
		"on_subject_annexed = {",
		"add_ideas = {",
		"ger_spirit",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GER on_actions missing %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{
		"ger_spine.9",
		"ger_endgame",
		"ger_tank_co",
		"on_unit_leader_promote_from_ranks = {",
		"on_subject_annexed_by_overlord",
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("GER on_actions must not contain %q:\n%s", unwanted, body)
		}
	}
}

func TestBuildNativeShadowOnActionsHonoursImportedFilter(t *testing.T) {
	excluded, err := BuildNativeShadow(onActionsSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if hasOnActionsFile(excluded, "ITA") {
		t.Fatalf("imported-only country got an on_actions file:\n%s", onActionsFile(t, excluded, "ITA"))
	}
	included, err := BuildNativeShadow(onActionsSnapshot(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if body := onActionsFile(t, included, "ITA"); !strings.Contains(body, "country_event = ger_spine.11") {
		t.Fatalf("imported event not hooked with includeImported=true:\n%s", body)
	}
}

func TestBuildNativeShadowSkipsEmptyOnActions(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 78,
		Project:   []byte(`{"id":78,"name":"Bare","country_tag":"GER","start_year":1936,"export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {json.RawMessage(`{"id":1,"tag":"GER","name":"Germany","source":"created","ideology":"fascism"}`)},

			"characters": {json.RawMessage(`{"id":11,"tag":"GER_leader","character_id":"GER_leader","country_tag":"GER","name":"Leader","source":"created","replaces_vanilla_id":false}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/common/on_actions/") {
			t.Fatalf("emitted a content-free on_actions file %q: %q", entry.Path, string(entry.Bytes))
		}
	}
	for _, covered := range artifact.Coverage {
		if covered == "on_actions" {
			t.Fatal("coverage claims on_actions with no on_actions entry")
		}
	}
}

func TestBuildNativeShadowOnActionsSubmodFocusTreeReload(t *testing.T) {
	tables := map[string][]json.RawMessage{
		"countries":   {json.RawMessage(`{"id":1,"tag":"GER","name":"Germany","source":"created","ideology":"fascism"}`)},
		"focus_trees": {json.RawMessage(`{"id":61,"tree_id":"ger_submod_tree","country_tag":"GER","tree_kind":"primary","source":"created"}`)},
	}
	standalone, err := BuildNativeShadow(Snapshot{
		ProjectID: 79,
		Project:   []byte(`{"id":79,"name":"Solo","country_tag":"GER","start_year":1936,"export_languages":["english"],"dependencies":[]}`),
		Tables:    tables,
	}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if hasOnActionsFile(standalone, "GER") {
		t.Fatalf("standalone project re-issued load_focus_tree:\n%s", onActionsFile(t, standalone, "GER"))
	}

	submod, err := BuildNativeShadow(Snapshot{
		ProjectID: 80,
		Project:   []byte(`{"id":80,"name":"Sub","country_tag":"GER","start_year":1936,"export_languages":["english"],"dependencies":["Old World Blues"]}`),
		Tables:    tables,
	}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	body := onActionsFile(t, submod, "GER")
	if !strings.Contains(body, "load_focus_tree = ger_submod_tree") {
		t.Fatalf("submod did not re-issue load_focus_tree:\n%s", body)
	}

	if !strings.Contains(body, "on_startup = {") {
		t.Fatalf("load_focus_tree reload is not under on_startup:\n%s", body)
	}
}

func TestOnActionsNeverReferencesAnAbsentTraitOverlayEvent(t *testing.T) {
	for _, includeImported := range []bool{false, true} {
		artifact, err := BuildNativeShadow(onActionsSnapshot(), nil, includeImported)
		if err != nil {
			t.Fatal(err)
		}
		var overlayFiles string
		for _, entry := range artifact.Entries {
			if strings.Contains(entry.Path, "_trait_overlay") {
				overlayFiles += entry.Path + "\n"
			}
		}
		for _, entry := range artifact.Entries {
			if !strings.Contains(entry.Path, "/common/on_actions/") {
				continue
			}
			if strings.Contains(string(entry.Bytes), "_trait_overlay") && overlayFiles == "" {
				t.Fatalf("%s fires a trait_overlay event but no artifact entry defines one:\n%s",
					entry.Path, string(entry.Bytes))
			}
		}
	}
}
