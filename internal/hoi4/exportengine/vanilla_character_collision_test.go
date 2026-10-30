package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func charSnapshot(rows ...string) Snapshot {
	raws := make([]json.RawMessage, 0, len(rows))
	for _, r := range rows {
		raws = append(raws, json.RawMessage(r))
	}
	return Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"ROM","start_year":1936,"export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{

			"countries":  {json.RawMessage(`{"id":1,"tag":"ROM","name":"Romania","source":"created","ideology":"fascism"}`)},
			"characters": raws,
		},
	}
}

func TestVanillaCharacterIDCollisionIsReported(t *testing.T) {
	artifact, err := BuildNativeShadow(charSnapshot(
		`{"id":1,"tag":"ROM_ion_antonescu","country_tag":"ROM","name":"Antonescu","source":"created","replaces_vanilla_id":false,"role_type":"field_marshal"}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var hit string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "ROM_ion_antonescu") {
			hit = w
		}
	}
	if hit == "" {
		t.Fatalf("reusing a base-game character id must warn; warnings=%v", artifact.Warnings)
	}
	if !strings.Contains(hit, "Multiple character have the tag") {
		t.Fatalf("warning must name the engine error the author will see: %s", hit)
	}

	if !strings.Contains(hit, "already assigned") {
		t.Fatalf("an unflagged row is recruited too; the warning must say so: %s", hit)
	}

	var history string
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/history/countries/ROM") {
			history = string(entry.Bytes)
		}
	}
	if !strings.Contains(history, "recruit_character = ROM_ion_antonescu") {
		t.Fatalf("warning claims a duplicate recruit, but none was emitted:\n%s", history)
	}
}

func TestVanillaCharacterIDCollisionOmitsRecruitClauseWhenFlagged(t *testing.T) {
	artifact, err := BuildNativeShadow(charSnapshot(
		`{"id":1,"tag":"ROM_ion_antonescu","country_tag":"ROM","name":"Antonescu","source":"created","replaces_vanilla_id":true,"role_type":"field_marshal"}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var hit string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "ROM_ion_antonescu") {
			hit = w
		}
	}
	if hit == "" {
		t.Fatal("a replacement row still duplicates the DEFINITION and must warn")
	}
	if strings.Contains(hit, "already assigned") {
		t.Fatalf("a flagged row is not recruited — claiming the recruit error is a false alarm: %s", hit)
	}
	var history string
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/history/countries/ROM") {
			history = string(entry.Bytes)
		}
	}
	if strings.Contains(history, "recruit_character = ROM_ion_antonescu") {
		t.Fatalf("flagged row must not be recruited:\n%s", history)
	}
}

func TestOwnCharacterIDDoesNotWarn(t *testing.T) {
	artifact, err := BuildNativeShadow(charSnapshot(
		`{"id":1,"tag":"ROM_my_own_general","country_tag":"ROM","name":"Mine","source":"created","replaces_vanilla_id":false,"role_type":"field_marshal"}`,
	), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "ROM_my_own_general") {
			t.Fatalf("a non-vanilla id must not warn: %s", w)
		}
	}
}
