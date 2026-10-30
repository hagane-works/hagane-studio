package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

func tombstoneRow(t *testing.T, body string) *emitmisc.NS {
	t.Helper()
	row, err := emitmisc.DecodeEntityJSON([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestIsTombstonedNeverFiltersEmptyShadowRows(t *testing.T) {
	rows := []string{
		`{"id":1,"dest_path":"common/on_actions/TNO_Reich_on_actions.txt","raw_kind":"empty_shadow","content":"","source":"created","target_layer":"tno","target_sha256":"abc"}`,
		`{"id":2,"dest_path":"common/raids/raids.txt","raw_kind":"verbatim_override","content":"x","source":"imported"}`,
		`{"id":3,"dest_path":"common/units/legacy.txt","content":"x"}`,
	}
	for _, body := range rows {
		row := tombstoneRow(t, body)
		if IsTombstoned(row) {
			t.Fatalf("row was filtered out of the export: %s", body)
		}
	}
}

func TestFullExportKeepsEveryLiveRawFile(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {
				json.RawMessage(`{"id":1,"dest_path":"common/raids/imported.txt","content":"imported body","raw_kind":"verbatim_override","source":"imported"}`),
				json.RawMessage(`{"id":2,"dest_path":"common/on_actions/authored.txt","content":"authored body","raw_kind":"verbatim_override","source":"created"}`),
				json.RawMessage(`{"id":3,"dest_path":"common/units/legacy.txt","content":"legacy body"}`),
			},
		},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]string{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = string(entry.Bytes)
	}
	for path, want := range map[string]string{
		"Demo/common/raids/imported.txt":      "imported body",
		"Demo/common/on_actions/authored.txt": "authored body",
		"Demo/common/units/legacy.txt":        "legacy body",
	} {
		if got, ok := entries[path]; !ok || !strings.Contains(got, want) {
			t.Fatalf("live raw file dropped from a full export: %s (present=%v)", path, ok)
		}
	}
}
