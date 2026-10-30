package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func rawFileSnapshot(rows ...string) Snapshot {
	raws := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		raws = append(raws, json.RawMessage(row))
	}
	return Snapshot{
		ProjectID: 46,
		Project:   json.RawMessage(`{"id":46,"name":"Demo","country_tag":"GER"}`),
		Tables:    map[string][]json.RawMessage{"project_raw_files": raws},
	}
}

func findEntry(artifact NativeArtifact, path string) (Entry, bool) {
	for _, entry := range artifact.Entries {
		if entry.Path == path {
			return entry, true
		}
	}
	return Entry{}, false
}

func TestRawOverrideLogWrittenAtZipRoot(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"common/ideas/extra.txt","content":"ideas = { }","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
		`{"id":2,"dest_path":"events/mine.txt","content":"country_event = { }","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	entry, ok := findEntry(artifact, "Demo/_RAW_OVERRIDE_FILES.log")
	if !ok {
		var paths []string
		for _, e := range artifact.Entries {
			paths = append(paths, e.Path)
		}
		t.Fatalf("no raw override log inside the mod folder; entries=%v", paths)
	}
	text := string(entry.Bytes)
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("log must end with a newline: %q", text)
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 3+2 {
		t.Fatalf("want 3 header lines + 2 entries, got %d:\n%s", len(lines), text)
	}
	if lines[0] != "# Raw override files" || lines[2] != "" {
		t.Fatalf("header drifted:\n%s", text)
	}
	for _, want := range []string{"WROTE: common/ideas/extra.txt", "WROTE: events/mine.txt"} {
		if !strings.Contains(text, want) {
			t.Fatalf("log missing %q:\n%s", want, text)
		}
	}
}

func TestNoRawOverrideLogWithoutOverrides(t *testing.T) {
	artifact, err := BuildNativeShadow(rawFileSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findEntry(artifact, "Demo/_RAW_OVERRIDE_FILES.log"); ok {
		t.Fatal("empty raw override log was written")
	}
}
