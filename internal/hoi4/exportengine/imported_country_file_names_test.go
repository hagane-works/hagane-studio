package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// A history imported from a mod is written back under the file name it had,
// so a mod that replaces a base-game history file still replaces it.
func TestImportedHistoryKeepsItsFileName(t *testing.T) {
	snapshot := Snapshot{ProjectID: 2,
		Project: json.RawMessage(`{"id":2,"name":"Names fixture","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"history_files": {
				json.RawMessage(`{"id":1,"country_tag":"GER","source":"imported","capital":64,"source_file":"history/countries/GER - Germany.txt"}`),
				json.RawMessage(`{"id":2,"country_tag":"ITA","source":"created","capital":2}`),
			},
		}}
	files, err := BuildNativeTextTree(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"history/countries/GER - Germany.txt", "history/countries/ITA - zNames fixture.txt"} {
		if _, ok := files[want]; !ok {
			t.Errorf("%s not written; files: %v", want, keysOf(files))
		}
	}
}

// A country imported from a mod keeps the definition file its country_tags
// named, and the exported registry points at that file.
func TestImportedCountryKeepsItsDefinitionFileName(t *testing.T) {
	snapshot := Snapshot{ProjectID: 2,
		Project: json.RawMessage(`{"id":2,"name":"Names fixture","country_tag":"RUR","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"RUR","name":"Ruritania","color":[1,2,3],"graphical_culture":"western_european_gfx","graphical_culture_2d":"western_european_2d","auto_added":false,"source_file":"common/countries/fds_Ruritania.txt"}`),
				json.RawMessage(`{"id":2,"tag":"RUS","name":"Rus","color":[4,5,6],"graphical_culture":"western_european_gfx","graphical_culture_2d":"western_european_2d","auto_added":false}`),
			},
		}}
	files, err := BuildNativeTextTree(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"common/countries/fds_Ruritania.txt", "common/countries/RUS.txt"} {
		if _, ok := files[want]; !ok {
			t.Errorf("%s not written; files: %v", want, keysOf(files))
		}
	}
	registry := ""
	for name, body := range files {
		if strings.HasPrefix(name, "common/country_tags/") {
			registry += string(body)
		}
	}
	if !strings.Contains(registry, `RUR = "countries/fds_Ruritania.txt"`) || !strings.Contains(registry, `RUS = "countries/RUS.txt"`) {
		t.Errorf("registry:\n%s", registry)
	}
}
