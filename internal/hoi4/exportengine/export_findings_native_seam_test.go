package exportengine

import (
	"encoding/json"
	"path"
	"strings"
	"testing"
)

func TestNativeArtifactPublishesStructuredFindingReadback(t *testing.T) {
	snapshot := rawFileSnapshot(`{"id":1,"dest_path":"events/literal_newline.txt","content":"country_event = { immediate = { a = yes\\nb = yes } }","source":"created","raw_kind":"verbatim_override","is_binary":false}`)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(artifact.Coverage, "export_findings_v1") {
		t.Fatalf("structured finding coverage missing: %v", artifact.Coverage)
	}
	entry, ok := findEntry(artifact, artifactRoot(artifact)+"/"+ExportFindingsBasename)
	if !ok {
		t.Fatalf("downloadable finding record missing: %#v", artifact.Entries)
	}
	var document ExportFindingsDocument
	if err := json.Unmarshal(entry.Bytes, &document); err != nil {
		t.Fatal(err)
	}
	if document.Schema != ExportFindingsSchema || len(document.Findings) == 0 {
		t.Fatalf("finding record is empty or has the wrong schema: %s", entry.Bytes)
	}
	if document.Findings[0].Acknowledged {
		t.Fatal("fresh artifact finding was incorrectly marked acknowledged")
	}
}

func TestNativeArtifactPersistsTimelineFindings(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 91,
		Project:   json.RawMessage(`{"id":91,"name":"Late Campaign","country_tag":"GER","start_date":"1964.5.1","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{"bookmarks": {
			json.RawMessage(`{"id":1,"bookmark_id":"late","date":"1964.5.1","source":"created"}`),
		}},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := findEntry(artifact, artifactRoot(artifact)+"/"+ExportFindingsBasename)
	if !ok {
		t.Fatal("timeline findings record missing")
	}
	var document ExportFindingsDocument
	if err := json.Unmarshal(entry.Bytes, &document); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range document.Findings {
		if finding.Code == "timeline.end_date_defaulted" {
			found = true
			if finding.SourceEvidence["actual"] != "2064.1.1.12" {
				t.Fatalf("timeline evidence lost: %#v", finding)
			}
			if finding.Message == "" || finding.PlayerImpact == "" || finding.EditRoute == "" {
				t.Fatalf("player-facing timeline explanation lost: %#v", finding)
			}
		}
	}
	if !found {
		t.Fatalf("timeline finding missing: %#v", document.Findings)
	}
}

func TestNativeArtifactSkipsLegacyRawFindingsSpoof(t *testing.T) {
	snapshot := rawFileSnapshot(`{"id":1,"dest_path":"reports/_hagane_export_findings.JSON","content":"{\"schema\":\"spoof\"}","source":"created","raw_kind":"verbatim_override","is_binary":false}`)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	want := artifactRoot(artifact) + "/" + ExportFindingsBasename
	count := 0
	for _, entry := range artifact.Entries {
		if strings.EqualFold(path.Base(entry.Path), ExportFindingsBasename) {
			count++
			if entry.Path != want {
				t.Fatalf("legacy raw spoof survived: %q", entry.Path)
			}
		}
	}
	if count > 1 {
		t.Fatalf("findings record count=%d entries=%#v", count, artifact.Entries)
	}
}

func TestNativeRawSelectionSkipsLegacyReferenceReportCollision(t *testing.T) {
	snapshot := rawFileSnapshot(`{"id":1,"dest_path":"reports/_reference_integrity.LOG","content":"author legacy","source":"created","raw_kind":"verbatim_override","is_binary":false}`)
	entries, _, err := nativeInlineRawFiles(snapshot, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("reserved legacy relationship report survived raw selection: %#v", entries)
	}
}
