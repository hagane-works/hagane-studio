package exportengine

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func cosmeticCollisionSnapshot(t *testing.T, secondTag, firstAdjective, secondName string) Snapshot {
	t.Helper()
	snapshot := cosmeticZIPSnapshot()
	snapshot.Project = json.RawMessage(`{"id":4242,"name":"CosmeticZip","country_tag":"ROM","export_languages":["en"]}`)
	snapshot.Tables["project_cosmetic_tags"] = []json.RawMessage{
		json.RawMessage(`{"id":1,"tag":"ROM_demo","source":"created"}`),
		cosmeticZIPRow(t, map[string]any{"id": 2, "tag": secondTag, "source": "created"}),
	}
	snapshot.Tables["project_cosmetic_tag_names"] = []json.RawMessage{
		cosmeticZIPRow(t, map[string]any{"cosmetic_tag_id": 1, "locale": "l_english", "suffix": "ADJ", "value": firstAdjective, "source": "created"}),
		cosmeticZIPRow(t, map[string]any{"cosmetic_tag_id": 2, "locale": "l_english", "suffix": "", "value": secondName, "source": "created"}),
	}
	return snapshot
}

func TestCosmeticCollisionCentralBlocksConflictingFinalZIP(t *testing.T) {
	snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_ADJ", "Royal", "Confederation")
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		for _, evidence := range []string{"ROM_demo_ADJ", "l_english", "ROM_demo", "cosmetic"} {
			if !strings.Contains(err.Error(), evidence) {
				t.Fatalf("collision error lost evidence %q: %v", evidence, err)
			}
		}
		return
	}
	fingerprint, err := SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "cosmetic-collision-negative.zip")
	if _, err := BuildWithMetadata(filename, fingerprint, artifact.Entries, DefaultArtifactBudget(), artifact.Timeline); err != nil {
		t.Fatalf("unrelated artifact failure: %v", err)
	}
	archive, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	var body []byte
	const target = "CosmeticZip/localisation/english/replace/zz_CosmeticZip_cosmetic_l_english.yml"
	for _, entry := range archive.File {
		if entry.Name != target {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err = io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(string(body), ` ROM_demo_ADJ:0 "Royal"`) || !strings.Contains(string(body), ` ROM_demo_ADJ:0 "Confederation"`) {
		t.Fatalf("negative fixture did not create conflicting final bytes: %q", body)
	}
	t.Logf("native warnings: %q", artifact.Warnings)
	t.Logf("native final syntax findings: %+v", ScanFinalArtifactSyntax(artifact.Entries, english).Findings)
	t.Logf("exact final loc path=%s base64=%s", target, base64.StdEncoding.EncodeToString(body))
	t.Fatal("two incompatible cosmetic names reached a successfully written final ZIP")
}

func TestCosmeticCollisionDetectorReportsFinalEvidence(t *testing.T) {
	snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_ADJ", "Royal", "Confederation")
	entries, err := nativeWorldFromSnapshot(t, snapshot, "CosmeticZip", false)
	if err != nil {
		t.Fatal(err)
	}
	err = validateCosmeticGeneratedCollisions(snapshot, false, entries)
	var collision *CosmeticNameCollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("expected structured collision, got %v", err)
	}
	if collision.Key != "ROM_demo_ADJ" || collision.Locale != "l_english" ||
		collision.Path != "CosmeticZip/localisation/english/replace/zz_CosmeticZip_cosmetic_l_english.yml" ||
		strings.Join(collision.Tags, ",") != "ROM_demo,ROM_demo_ADJ" ||
		len(collision.Lines) != 2 || collision.Lines[0] != 4 || collision.Lines[1] != 17 ||
		strings.Join(collision.Values, ",") != "Royal,Confederation" {
		t.Fatalf("incomplete physical evidence: %+v", collision)
	}

	for _, outside := range []struct{ origin, source, path string }{
		{"raw_override", "project_raw_files:1", collision.Path},
		{"generated", "world", "CosmeticZip/localisation/english/replace/countries_l_english.yml"},
		{"generated", "event_localisation", collision.Path},
	} {
		copyEntries := append([]Entry(nil), entries...)
		for i := range copyEntries {
			if copyEntries[i].Path == collision.Path {
				copyEntries[i].Origin, copyEntries[i].SourceEntity, copyEntries[i].Path = outside.origin, outside.source, outside.path
			}
		}
		if err := validateCosmeticGeneratedCollisions(snapshot, false, copyEntries); err != nil {
			t.Fatalf("scope expanded to %+v: %v", outside, err)
		}
	}
}

func TestCosmeticCollisionRegistryAndLegacyReference(t *testing.T) {
	for _, registryTag := range []string{"ROM_demo", "ROM_demo_ADJ"} {
		t.Run(registryTag, func(t *testing.T) {
			snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_ADJ", "Royal", "Confederation")
			legacyTag, parentIndex := "ROM_demo_ADJ", 0
			if registryTag == "ROM_demo_ADJ" {
				legacyTag, parentIndex = "ROM_demo", 1
			}
			snapshot.Tables["project_cosmetic_tags"] = snapshot.Tables["project_cosmetic_tags"][parentIndex : parentIndex+1]
			snapshot.Tables["history_files"] = []json.RawMessage{cosmeticZIPRow(t, map[string]any{
				"id": 1, "country_tag": "ROM", "set_cosmetic_tag": legacyTag, "source": "created",
			})}
			entries, err := nativeWorldFromSnapshot(t, snapshot, "CosmeticZip", false)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateCosmeticGeneratedCollisions(snapshot, false, entries); err == nil {
				t.Fatal("registry versus legacy reference collision was missed")
			}

			snapshot.Tables["history_files"] = []json.RawMessage{cosmeticZIPRow(t, map[string]any{
				"id": 1, "country_tag": "ROM", "set_cosmetic_tag": legacyTag, "source": "imported",
			})}
			cosmeticFinalZIP(t, snapshot, false)
		})
	}
}

func TestCosmeticCollisionAdjacentFinalZIPs(t *testing.T) {
	t.Run("different keys", func(t *testing.T) {
		snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_other", "Royal", "Confederation")
		names := cosmeticZIPNames(t, cosmeticFinalZIP(t, snapshot, false))
		cosmeticZIPRequireName(t, names, "l_english", "ROM_demo", "ADJ", "Royal")
		cosmeticZIPRequireName(t, names, "l_english", "ROM_demo_other", "", "Confederation")
	})
	t.Run("same final value", func(t *testing.T) {
		snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_ADJ", "Royal", "Royal")
		values := cosmeticZIPNames(t, cosmeticFinalZIP(t, snapshot, false))["l_english"]["ROM_demo_ADJ"]
		if len(values) == 0 {
			t.Fatal("equal value disappeared")
		}
		for _, value := range values {
			if value != "Royal" {
				t.Fatalf("unexpected equal-key value %q", value)
			}
		}
	})
	t.Run("raw final replacement", func(t *testing.T) {
		snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_ADJ", "Royal", "Confederation")
		const path = "localisation/english/replace/zz_CosmeticZip_cosmetic_l_english.yml"
		const body = "\ufeffl_english:\n ROM_demo_ADJ:0 \"Chosen raw value\"\n"
		snapshot.Tables["project_raw_files"] = []json.RawMessage{cosmeticZIPRow(t, map[string]any{
			"id": 1, "dest_path": path, "content": body, "raw_kind": "verbatim_override", "source": "created",
		})}
		if got := string(cosmeticFinalZIP(t, snapshot, false)["CosmeticZip/"+path]); got != body {
			t.Fatalf("raw final bytes changed: %q", got)
		}
	})
	t.Run("preserved suppresses one generated occurrence", func(t *testing.T) {
		snapshot := cosmeticCollisionSnapshot(t, "ROM_demo_ADJ", "Royal", "Confederation")
		snapshot.Tables["project_cosmetic_tag_names"] = snapshot.Tables["project_cosmetic_tag_names"][1:]
		snapshot.Tables["project_loc_entries"] = []json.RawMessage{cosmeticZIPRow(t, map[string]any{
			"id": 1, "locale": "l_english", "key": "ROM_demo_ADJ", "value": "Imported", "version": 0,
			"source_file": "localisation/english/original_l_english.yml", "source": "imported",
		})}
		names := cosmeticZIPNames(t, cosmeticFinalZIP(t, snapshot, true))
		cosmeticZIPRequireName(t, names, "l_english", "ROM_demo_ADJ", "", "Confederation")
	})
	t.Run("pure legacy retained", func(t *testing.T) {
		snapshot := cosmeticZIPSnapshot()
		for index, tag := range []string{"ROM_demo", "ROM_demo_ADJ"} {
			snapshot.Tables["history_files"] = append(snapshot.Tables["history_files"], cosmeticZIPRow(t, map[string]any{
				"id": index + 1, "country_tag": []string{"ROM", "HUN"}[index], "set_cosmetic_tag": tag, "source": "created",
			}))
		}
		if got := cosmeticZIPNames(t, cosmeticFinalZIP(t, snapshot, false))["l_english"]["ROM_demo_ADJ"]; len(got) != 2 {
			t.Fatalf("legacy occurrence behavior changed: %q", got)
		}
	})
}

func nativeWorldFromSnapshot(t *testing.T, snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	t.Helper()
	rows, err := decodePreservedLocRows(snapshot.Tables["project_loc_entries"])
	if err != nil {
		return nil, err
	}
	return nativeWorld(snapshot, modDir, includeImported, nil, nil, nil, rows)
}
