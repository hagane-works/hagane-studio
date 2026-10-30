package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativeTimelinePlanDrivesDescriptorAndDefines(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 72,
		Project:   []byte(`{"id":72,"name":"Late Start","country_tag":"USA","start_date":"1964.5.1.12","end_date":"1974.1.1"}`),
		Tables: map[string][]json.RawMessage{
			"bookmarks": {json.RawMessage(`{"id":9,"bookmark_id":"late_start","name_en":"Late","date":"1964.5.1.12","source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]string{}
	for _, entry := range artifact.Entries {
		entries[entry.Path] = string(entry.Bytes)
	}
	defines := entries["Late Start/common/defines/Late Start_defines.lua"]
	if !strings.Contains(defines, `START_DATE = "1964.5.1.12"`) || !strings.Contains(defines, `END_DATE = "1974.1.1.12"`) {
		t.Fatalf("timeline defines missing or wrong: %q", defines)
	}
	for _, path := range []string{"Late Start.mod", "Late Start/descriptor.mod"} {
		if !strings.Contains(entries[path], `replace_path="common/bookmarks"`) {
			t.Fatalf("%s did not use the timeline plan replace path: %q", path, entries[path])
		}
	}
}

func TestRawTimelineDefinesRemainTheFinalWinner(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 73,
		Project:   []byte(`{"id":73,"name":"Raw Winner","country_tag":"USA","start_date":"1964.5.1.12"}`),
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {json.RawMessage(`{"id":5,"dest_path":"common/defines/Raw Winner_defines.lua","content":"NDefines.NGame.START_DATE = \"1965.1.1\"\n","raw_kind":"verbatim_override","source":"created"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range artifact.Entries {
		if entry.Path == "Raw Winner/common/defines/Raw Winner_defines.lua" {
			if got := string(entry.Bytes); !strings.Contains(got, `START_DATE = "1965.1.1"`) || strings.Contains(got, "1964.5.1.12") {
				t.Fatalf("raw defines did not remain the final winner: %q", got)
			}
			return
		}
	}
	t.Fatal("raw defines entry missing")
}
