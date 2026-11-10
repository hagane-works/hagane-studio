package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// Imported rows go back to the file they came from, so a mod that replaces a
// base-game file by name (colony.txt, 00_intelligence_agencies.txt,
// the_gathering_storm.txt) still replaces it. Rows made in the app keep the
// project's own file.
func TestImportedRowsKeepTheirFileNames(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"opinion_modifiers": {
				json.RawMessage(`{"id":1,"modifier_id":"fds_opinion","body":{"value":10},"source":"imported","source_file":"common/opinion_modifiers/fds_opinion_modifiers.txt"}`),
			},
			"dynamic_modifiers": {
				json.RawMessage(`{"id":1,"dm_id":"fds_dm","modifier":{"political_power_gain":0.1},"source":"imported","source_file":"common/dynamic_modifiers/0_dynamic_modifiers.txt"}`),
				json.RawMessage(`{"id":2,"dm_id":"GER_demo_modifier","modifier":{"political_power_gain":0.1},"source":"created"}`),
			},
			"autonomy_states": {
				json.RawMessage(`{"id":1,"autonomy_id":"autonomy_colony","default":false,"source":"imported","source_file":"common/autonomous_states/colony.txt"}`),
				json.RawMessage(`{"id":2,"autonomy_id":"autonomy_puppet","default":false,"source":"imported","source_file":"common/autonomous_states/puppet.txt"}`),
			},
			"intelligence_agencies": {
				json.RawMessage(`{"id":3,"agency_id":"fds_agency","names":["FDS_AGENCY"],"source":"imported","source_file":"common/intelligence_agencies/00_intelligence_agencies.txt"}`),
			},
			"special_projects": {
				json.RawMessage(`{"id":4,"project_token":"sp_land","specialization":"land","source":"imported","source_file":"common/special_projects/projects/land_projects.txt"}`),
			},
			"bookmarks": {
				json.RawMessage(`{"id":5,"bookmark_id":"fds_bookmark_1936","date":"1936.1.1.12","source":"imported","source_file":"common/bookmarks/the_gathering_storm.txt"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[strings.TrimPrefix(entry.Path, "Demo/")] = string(entry.Bytes)
	}
	for path, want := range map[string]string{
		"common/opinion_modifiers/fds_opinion_modifiers.txt":        "fds_opinion",
		"common/dynamic_modifiers/0_dynamic_modifiers.txt":          "fds_dm",
		"common/dynamic_modifiers/GER_dynamic_modifiers.txt":        "GER_demo_modifier",
		"common/autonomous_states/colony.txt":                       "autonomy_colony",
		"common/autonomous_states/puppet.txt":                       "autonomy_puppet",
		"common/intelligence_agencies/00_intelligence_agencies.txt": "FDS_AGENCY",
		"common/special_projects/projects/land_projects.txt":        "sp_land",
		"common/bookmarks/the_gathering_storm.txt":                  "FDS_BOOKMARK_1936_NAME",
	} {
		if !strings.Contains(contents[path], want) {
			t.Errorf("%s does not hold %s", path, want)
		}
	}
	for _, path := range []string{
		"common/opinion_modifiers/GER_opinion_modifiers.txt", "common/autonomous_states/GER_autonomous_states.txt",
		"common/intelligence_agencies/GER_intelligence_agencies.txt", "common/special_projects/projects/GER_projects.txt",
		"common/bookmarks/fds_bookmark_1936.txt",
	} {
		if _, ok := contents[path]; ok {
			t.Errorf("%s written although every row has its own file", path)
		}
	}
	if strings.Contains(contents["common/dynamic_modifiers/0_dynamic_modifiers.txt"], "GER_demo_modifier") {
		t.Error("a modifier made in the app went into the imported file")
	}
}
