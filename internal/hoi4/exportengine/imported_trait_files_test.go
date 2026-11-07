package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc/characters"
)

// A mod that replaces the base game's trait file by name gets its traits
// back in that file, not in a second `<tag>_custom_traits.txt` that would
// define every trait again. Traits made in the app keep the project's file.
func TestImportedTraitsGoBackToTheirOwnFile(t *testing.T) {
	parsed, _, err := characters.ImportCountryLeaderTraits(`leader_traits = { fds_strongman = { random = no modifier = { stability_factor = 0.1 } } }`)
	if err != nil {
		t.Fatal(err)
	}
	unitParsed, _, err := characters.ImportUnitLeaderTraits(`leader_traits = { fds_brawler = { type = land_leader attack_skill = 1 } }`)
	if err != nil {
		t.Fatal(err)
	}
	var rows []json.RawMessage
	add := func(row map[string]any, id int, kind, file string) {
		row["id"], row["trait_type"], row["country_tag"] = id, kind, "DPR"
		// What import stores (chars_extras.go charsExtrasMergeTraits).
		row["source"], row["source_file"] = "imported", file
		raw, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, raw)
	}
	for _, row := range characters.NativeEntities(parsed) {
		add(row, 1, "country_leader", "common/country_leader/00_traits.txt")
	}
	for _, row := range characters.NativeEntities(unitParsed) {
		add(row, 2, "unit_leader", "common/unit_leader/00_traits.txt")
	}
	rows = append(rows, json.RawMessage(`{"id":3,"trait_id":"DPR_new_trait","trait_type":"country_leader","country_tag":"DPR","name_en":"New","source":"created"}`))
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"DPR","export_languages":["en"]}`),
		Tables:    map[string][]json.RawMessage{"custom_traits": rows},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	for path, want := range map[string]string{
		"Demo/common/country_leader/00_traits.txt":         "fds_strongman",
		"Demo/common/unit_leader/00_traits.txt":            "fds_brawler",
		"Demo/common/country_leader/DPR_custom_traits.txt": "DPR_new_trait",
	} {
		if !strings.Contains(contents[path], want) {
			t.Errorf("%s missing %q", path, want)
		}
	}
	if strings.Contains(contents["Demo/common/country_leader/DPR_custom_traits.txt"], "fds_strongman") {
		t.Errorf("imported trait is defined again in the project's trait file")
	}
	if _, ok := contents["Demo/common/unit_leader/DPR_custom_unit_traits.txt"]; ok {
		t.Errorf("a project unit trait file is written when every unit trait is imported")
	}
}
