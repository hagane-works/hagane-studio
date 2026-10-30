package exportengine

import (
	"encoding/json"
	"path"
	"testing"
)

func TestRefreshFinalArtifactFindingsReplacesReferenceEvidenceAndLog(t *testing.T) {
	missing := "demo/common/national_focus/demo.txt"
	entries := []Entry{
		{Path: "demo.mod", Bytes: []byte(`name="demo"`)},
		{Path: missing, Bytes: []byte(`focus_tree = { id = demo focus = { id = root prerequisite = { focus = absent } } }`)},
		{Path: "demo/" + ReferenceIntegrityLogBasename, Bytes: []byte("stale"), SourceEntity: "reference_integrity"},
	}
	existing := []ExportFinding{{
		Code: "old_reference", Source: json.RawMessage(`{"kind":"reference_integrity","identifier":"old"}`),
	}}
	snapshot := referenceSnapshot(`{"id":61,"name":"Demo","dependencies":[]}`)
	refreshedEntries, refreshedFindings := RefreshFinalArtifactFindings(entries, existing, &snapshot, english)

	foundCurrent := false
	for _, finding := range refreshedFindings {
		if finding.Code == "old_reference" {
			t.Fatal("stale reference finding survived final rescan")
		}
		if finding.Code == "focus_prerequisite" {
			foundCurrent = true
		}
	}
	if !foundCurrent {
		t.Fatalf("final missing prerequisite was not reported: %#v", refreshedFindings)
	}
	logs := 0
	for _, entry := range refreshedEntries {
		if path.Base(entry.Path) == ReferenceIntegrityLogBasename {
			logs++
			if string(entry.Bytes) == "stale" {
				t.Fatal("stale reference log survived final rescan")
			}
		}
	}
	if logs != 1 {
		t.Fatalf("reference logs=%d entries=%#v", logs, refreshedEntries)
	}
}

func TestRefreshFinalArtifactFindingsPreservesAuthorReferenceLogCollision(t *testing.T) {
	entries := []Entry{
		{Path: "demo.mod", Bytes: []byte(`name="demo"`)},
		{Path: "demo/common/national_focus/demo.txt", Bytes: []byte(`focus_tree = { id = demo focus = { id = root prerequisite = { focus = absent } } }`)},
		{Path: "demo/" + ReferenceIntegrityLogBasename, Bytes: []byte("author-owned")},
	}
	snapshot := referenceSnapshot(`{"id":61,"name":"Demo","dependencies":[]}`)
	refreshedEntries, _ := RefreshFinalArtifactFindings(entries, nil, &snapshot, english)
	logs := 0
	for _, entry := range refreshedEntries {
		if path.Base(entry.Path) == ReferenceIntegrityLogBasename {
			logs++
			if string(entry.Bytes) != "author-owned" {
				t.Fatalf("author log was overwritten: %q", entry.Bytes)
			}
		}
	}
	if logs != 1 {
		t.Fatalf("reference logs=%d entries=%#v", logs, refreshedEntries)
	}
}
