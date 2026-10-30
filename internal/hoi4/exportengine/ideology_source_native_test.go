package exportengine

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/ideologysource"
)

const ideologySourceNativePath = "common/ideologies/00_ideologies.txt"

func ideologySourceNativeLayers(source []byte) []ideologysource.Layer {
	return []ideologysource.Layer{{
		SourceKind: ideologysource.SourceVanilla, LoadOrder: 0,
		Files: []ideologysource.SourceFile{{Path: ideologySourceNativePath, Bytes: source}},
	}}
}

func ideologySourceNativeFixture(t *testing.T, source []byte, ids ...string) ([]ideologysource.Layer, ideologysource.Manifest) {
	t.Helper()
	layers := ideologySourceNativeLayers(source)
	discovery, err := ideologysource.Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Roots) != 1 {
		t.Fatalf("roots=%#v", discovery.Roots)
	}
	owner := discovery.Roots[0]
	manifest := ideologysource.Manifest{SchemaVersion: ideologysource.SchemaVersion}
	for _, id := range ids {
		manifest.Entries = append(manifest.Entries, ideologysource.ManifestEntry{
			SourceKind: owner.SourceKind, DependencyKey: owner.DependencyKey,
			OwnerPath: owner.OwnerPath, FileSHA256: owner.FileSHA256, RootSHA256: owner.RootSHA256,
			ScriptID: id, ParentRootID: owner.RootID,
		})
	}
	return layers, manifest
}

func ideologySourceNativeProject(t *testing.T, manifest ideologysource.Manifest, subtypes []map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"ideology_source_manifest": ideologySourceManifestValue(manifest, "vanilla", "ready"),
		"ideology_registry":        map[string]any{"sub_ideologies": subtypes},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func ideologySourceManifestValue(manifest ideologysource.Manifest, sourceKey, state string) map[string]any {
	return map[string]any{
		"schema_version": manifest.SchemaVersion, "entries": manifest.Entries,
		"state": state, "source_key": sourceKey,
	}
}

func ideologySourceFindingCodes(err error) map[string]bool {
	result := map[string]bool{}
	var findings ideologysource.FindingsError
	if errors.As(err, &findings) {
		for _, finding := range findings.Findings {
			result[finding.Code] = true
		}
	}
	return result
}

func TestNativeInheritedIdeologySourcePatchesExactOwnerWithoutReplacePath(t *testing.T) {
	source := []byte("# keep this comment\nideologies = {\n\tdemocratic = {\n\t\ttypes = { conservatism = { } }\n\t\tcolor = { 0 0 255 }\n\t}\n}\n")
	layers, manifest := ideologySourceNativeFixture(t, source, "demo_reformism")
	project := ideologySourceNativeProject(t, manifest, []map[string]any{{
		"id": "demo_reformism", "group": "democratic", "source": "project_managed",
		"name": map[string]string{"en": "Demo Reformism", "zh": "示例改革主义"},
		"desc": map[string]string{"en": "A source-locked subtype."},
	}})
	entries, err := nativeInheritedIdeologySource(project, layers, nil, "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries=%#v", entries)
	}
	byPath := map[string]Entry{}
	for _, entry := range entries {
		byPath[entry.Path] = entry
		if strings.Contains(strings.ToLower(entry.Path), "descriptor") || strings.Contains(string(entry.Bytes), "replace_path") {
			t.Fatalf("adapter emitted descriptor replacement material: %#v", entry)
		}
	}
	patched, ok := byPath["Demo/"+ideologySourceNativePath]
	if !ok || !bytes.Contains(patched.Bytes, []byte("# keep this comment")) ||
		!bytes.Contains(patched.Bytes, []byte("demo_reformism = {")) ||
		!bytes.Contains(patched.Bytes, []byte("can_be_randomly_selected = no")) {
		t.Fatalf("owner file was not byte-preservingly patched:\n%s", patched.Bytes)
	}
	english := byPath["Demo/localisation/english/replace/hagane_inherited_ideology_l_english.yml"].Bytes
	if !bytes.HasPrefix(english, []byte{0xef, 0xbb, 0xbf}) ||
		!bytes.Contains(english, []byte(" demo_reformism: \"Demo Reformism\"")) ||
		!bytes.Contains(english, []byte(" demo_reformism_desc: \"A source-locked subtype.\"")) ||
		bytes.Contains(english, []byte(":0")) {
		t.Fatalf("invalid inherited ideology localisation:\n%s", english)
	}
}

func TestNativeInheritedIdeologySourceFailsClosedOnSourceDrift(t *testing.T) {
	source := []byte("ideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")
	_, manifest := ideologySourceNativeFixture(t, source, "demo_reformism")
	project := ideologySourceNativeProject(t, manifest, []map[string]any{{
		"id": "demo_reformism", "group": "democratic", "source": "project_managed", "name": map[string]string{"en": "Demo"},
	}})
	drifted := bytes.Replace(source, []byte("0 0 255"), []byte("10 10 255"), 1)
	_, err := nativeInheritedIdeologySource(project, ideologySourceNativeLayers(drifted), nil, "Demo")
	codes := ideologySourceFindingCodes(err)
	if !codes["ideology_source_file_drift"] || !codes["ideology_source_root_drift"] {
		t.Fatalf("err=%v codes=%v", err, codes)
	}
}

func TestNativeInheritedIdeologySourceRejectsRawOwnershipConflict(t *testing.T) {
	source := []byte("ideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")
	layers, manifest := ideologySourceNativeFixture(t, source, "demo_reformism")
	project := ideologySourceNativeProject(t, manifest, []map[string]any{{
		"id": "demo_reformism", "group": "democratic", "source": "project_managed", "name": map[string]string{"en": "Demo"},
	}})
	_, err := nativeInheritedIdeologySource(project, layers, []string{"events/x.txt", `Demo\COMMON\IDEOLOGIES\mine.txt`}, "Demo")
	var blocker IdeologySourceNativeError
	if !errors.As(err, &blocker) || blocker.Code != "ideology_source_raw_conflict" || blocker.Path != "demo/common/ideologies/mine.txt" {
		t.Fatalf("err=%#v", err)
	}
}

func TestNativeInheritedIdeologySourceRejectsMissingRegistrySubtype(t *testing.T) {
	source := []byte("ideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")
	layers, manifest := ideologySourceNativeFixture(t, source, "demo_reformism")
	_, err := nativeInheritedIdeologySource(ideologySourceNativeProject(t, manifest, nil), layers, nil, "Demo")
	if !ideologySourceFindingCodes(err)["ideology_source_subtype_missing"] {
		t.Fatalf("err=%v", err)
	}
}

func TestNativeInheritedIdeologySourceRejectsMalformedAndIDMismatch(t *testing.T) {
	source := []byte("ideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")
	layers, manifest := ideologySourceNativeFixture(t, source, "demo_reformism")
	t.Run("malformed registry", func(t *testing.T) {
		project := json.RawMessage(`{"ideology_source_manifest":` + string(mustJSON(t, ideologySourceManifestValue(manifest, "vanilla", "ready"))) + `,"ideology_registry":{"sub_ideologies":"bad"}}`)
		_, err := nativeInheritedIdeologySource(project, layers, nil, "Demo")
		var blocker IdeologySourceNativeError
		if !errors.As(err, &blocker) || blocker.Code != "ideology_source_registry_malformed" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("parent mismatch", func(t *testing.T) {
		project := ideologySourceNativeProject(t, manifest, []map[string]any{{
			"id": "demo_reformism", "group": "fascism", "source": "project_managed", "name": map[string]string{"en": "Demo"},
		}})
		_, err := nativeInheritedIdeologySource(project, layers, nil, "Demo")
		if !ideologySourceFindingCodes(err)["ideology_source_parent_drift"] {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("malformed source", func(t *testing.T) {
		project := ideologySourceNativeProject(t, manifest, []map[string]any{{
			"id": "demo_reformism", "group": "democratic", "source": "project_managed", "name": map[string]string{"en": "Demo"},
		}})
		broken := ideologySourceNativeLayers([]byte("ideologies = { democratic = {"))
		_, err := nativeInheritedIdeologySource(project, broken, nil, "Demo")
		if !ideologySourceFindingCodes(err)["ideology_source_malformed"] {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("ambiguous source", func(t *testing.T) {
		project := ideologySourceNativeProject(t, manifest, []map[string]any{{
			"id": "demo_reformism", "group": "democratic", "source": "project_managed", "name": map[string]string{"en": "Demo"},
		}})
		ambiguous := ideologySourceNativeLayers(source)
		ambiguous[0].Files = append(ambiguous[0].Files, ideologysource.SourceFile{Path: ideologySourceNativePath, Bytes: source})
		_, err := nativeInheritedIdeologySource(project, ambiguous, nil, "Demo")
		if !ideologySourceFindingCodes(err)["ideology_source_file_ambiguous"] {
			t.Fatalf("err=%v", err)
		}
	})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestNativeInheritedIdeologySourceIsDeterministicAndEmptyManifestIsNoop(t *testing.T) {
	source := []byte("ideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")
	layers, manifest := ideologySourceNativeFixture(t, source, "demo_z", "demo_a")
	subs := []map[string]any{
		{"id": "demo_z", "group": "democratic", "source": "project_managed", "name": map[string]string{"zh": "乙", "en": "Zed"}},
		{"id": "demo_a", "group": "democratic", "source": "created", "name": map[string]string{"en": "Alpha", "zh": "甲"}},
	}
	first, err := nativeInheritedIdeologySource(ideologySourceNativeProject(t, manifest, subs), layers, []string{"events/z.txt", "events/a.txt"}, "Demo")
	if err != nil {
		t.Fatal(err)
	}
	manifest.Entries[0], manifest.Entries[1] = manifest.Entries[1], manifest.Entries[0]
	subs[0], subs[1] = subs[1], subs[0]
	second, err := nativeInheritedIdeologySource(ideologySourceNativeProject(t, manifest, subs), layers, []string{"events/a.txt", "events/z.txt"}, "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("nondeterministic:\nfirst=%#v\nsecond=%#v", first, second)
	}
	empty, err := nativeInheritedIdeologySource(json.RawMessage(`{"ideology_source_manifest":{},"ideology_registry":"malformed"}`), nil, []string{"common/ideologies/conflict.txt"}, "Demo")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty manifest must be a no-op: entries=%#v err=%v", empty, err)
	}
}

func ideologySourceImportedFixture(t *testing.T, source []byte, id string) ([]ideologysource.Layer, ideologysource.Manifest) {
	t.Helper()
	layers := []ideologysource.Layer{
		{SourceKind: ideologysource.SourceVanilla, LoadOrder: 0, Files: []ideologysource.SourceFile{}},
		{SourceKind: ideologysource.SourceDependency, DependencyKey: "imported", LoadOrder: 1, Files: []ideologysource.SourceFile{{Path: ideologySourceNativePath, Bytes: source}}},
	}
	discovery, err := ideologysource.Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Roots) != 1 {
		t.Fatalf("roots=%#v", discovery.Roots)
	}
	owner := discovery.Roots[0]
	return layers, ideologysource.Manifest{SchemaVersion: ideologysource.SchemaVersion, Entries: []ideologysource.ManifestEntry{{
		SourceKind: owner.SourceKind, DependencyKey: owner.DependencyKey,
		OwnerPath: owner.OwnerPath, FileSHA256: owner.FileSHA256, RootSHA256: owner.RootSHA256,
		ScriptID: id, ParentRootID: owner.RootID,
	}}}
}

func ideologySourceArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		header := &zip.FileHeader{Name: path, Method: zip.Deflate}
		header.SetMode(0o600)
		member, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(files[path]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func ideologySourceNativeSnapshot(t *testing.T, source []byte, state string, includeAsset bool) (Snapshot, []byte, string) {
	t.Helper()
	_, manifest := ideologySourceImportedFixture(t, source, "demo_reformism")
	project, err := json.Marshal(map[string]any{
		"id": 77, "name": "Demo", "country_tag": "GER", "export_languages": []string{"en", "zh"},
		"dependencies":             []string{"Imported Base"},
		"ideology_source_manifest": ideologySourceManifestValue(manifest, "imported", state),
		"ideology_registry": map[string]any{"sub_ideologies": []map[string]any{{
			"id": "demo_reformism", "group": "democratic", "source": "project_managed",
			"name": map[string]string{"en": "Demo Reformism", "zh": "示例改革主义"},
			"desc": map[string]string{"en": "Source locked."},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	archive := ideologySourceArchive(t, map[string][]byte{ideologySourceNativePath: source})
	key := "private/ideology/imported.zip"
	tables := map[string][]json.RawMessage{}
	if includeAsset {
		size := int64(len(source))
		asset, err := json.Marshal(map[string]any{
			"asset_type": "ideology_source", "gfx_name": "imported", "obs_key": key,
			"provenance": map[string]any{
				"schema_version": 1, "source_key": "imported", "dependency_key": "imported",
				"descriptor_replace_paths": []string{},
				"files":                    []map[string]any{{"path": ideologySourceNativePath, "sha256": fmt.Sprintf("%x", sha256.Sum256(source)), "size_bytes": size}},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		tables["project_obs_assets"] = []json.RawMessage{asset}
	}
	return Snapshot{ProjectID: 77, Project: project, Tables: tables}, archive, key
}

func ideologySourceNativeErrorCode(err error) string {
	var blocker IdeologySourceNativeError
	if errors.As(err, &blocker) {
		return blocker.Code
	}
	return ""
}

func TestBuildNativeShadowShipsReadyInheritedIdeologySourceEndToEnd(t *testing.T) {
	source := []byte("# source bytes stay\nideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")
	snapshot, archive, key := ideologySourceNativeSnapshot(t, source, "ready", true)
	resolveCalls := 0
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(got string, limit int64) ([]byte, error) {
		resolveCalls++
		if got != key || limit != ideologySourceArchiveLimit {
			t.Fatalf("resolve got=(%q,%d)", got, limit)
		}
		return archive, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolveCalls != 1 {
		t.Fatalf("resolver calls=%d, private snapshot was not prefetched/reused", resolveCalls)
	}
	byPath := map[string]Entry{}
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry
	}
	patched := byPath["Demo/"+ideologySourceNativePath].Bytes
	if !bytes.Contains(patched, []byte("# source bytes stay")) || !bytes.Contains(patched, []byte("demo_reformism = {")) {
		t.Fatalf("inherited owner was not shipped:\n%s", patched)
	}
	if strings.Contains(string(byPath["Demo.mod"].Bytes), `replace_path="common/ideologies"`) ||
		artifact.Timeline == nil || containsCanonicalString(artifact.Timeline.Fingerprint.EffectiveReplacePaths, "common/ideologies") {
		t.Fatalf("inherited source incorrectly requested directory replacement: timeline=%#v descriptor=%s", artifact.Timeline, byPath["Demo.mod"].Bytes)
	}
	if !containsCanonicalString(artifact.Coverage, "ideology_source") {
		t.Fatalf("coverage=%v", artifact.Coverage)
	}
	prefetched := false
	for _, item := range collectPrefetchKeys(snapshot, false) {
		if item.Key == key && item.Limit == ideologySourceArchiveLimit {
			prefetched = true
		}
	}
	if !prefetched {
		t.Fatal("ready inherited ideology snapshot missing from prefetch plan")
	}
	plan, err := BuildTimelinePlanFromSnapshot(snapshot, nil, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	withPlan, err := BuildNativeShadowWithTimelinePlan(snapshot, plan, false, func(got string, limit int64) ([]byte, error) {
		if got != key || limit != ideologySourceArchiveLimit {
			t.Fatalf("timeline-plan resolve got=(%q,%d)", got, limit)
		}
		return archive, nil
	}, english)
	if err != nil {
		t.Fatal(err)
	}
	if entry := func() Entry {
		for _, candidate := range withPlan.Entries {
			if candidate.Path == "Demo/"+ideologySourceNativePath {
				return candidate
			}
		}
		return Entry{}
	}(); !bytes.Contains(entry.Bytes, []byte("demo_reformism = {")) {
		t.Fatal("BuildNativeShadowWithTimelinePlan did not run the inherited source adapter")
	}
}

func TestBuildNativeShadowShipsBundledVanillaSubtypeWithoutPrivateBlob(t *testing.T) {
	tables, err := testWorldTables(t)
	if err != nil {
		t.Fatal(err)
	}
	layers := []ideologysource.Layer{{
		SourceKind: ideologysource.SourceVanilla, LoadOrder: 0,
		Files: []ideologysource.SourceFile{{Path: ideologySourceNativePath, Bytes: []byte(tables.IdeologyBaseText)}},
	}}
	discovery, err := ideologysource.Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	var owner ideologysource.RootOwner
	for _, candidate := range discovery.Roots {
		if candidate.RootID == "democratic" {
			owner = candidate
			break
		}
	}
	if owner.RootID == "" {
		t.Fatal("bundled vanilla snapshot has no democratic ROOT")
	}
	manifest := ideologysource.Manifest{SchemaVersion: ideologysource.SchemaVersion, Entries: []ideologysource.ManifestEntry{{
		SourceKind: owner.SourceKind, OwnerPath: owner.OwnerPath,
		FileSHA256: owner.FileSHA256, RootSHA256: owner.RootSHA256,
		ScriptID: "pc01_social_liberalism", ParentRootID: owner.RootID,
	}}}
	project := ideologySourceNativeProject(t, manifest, []map[string]any{{
		"id": "pc01_social_liberalism", "group": "democratic", "source": "project_managed",
		"name": map[string]string{"en": "PC-01 Social Liberalism"},
	}})
	var decoded map[string]any
	if err := json.Unmarshal(project, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded["id"], decoded["name"], decoded["country_tag"] = 78, "PC01", "TST"
	project = mustJSON(t, decoded)
	snapshot := Snapshot{
		ProjectID:   78,
		Fingerprint: ideologysource.Hash([]byte(tables.IdeologyBaseText)),
		Project:     project,
		Tables:      map[string][]json.RawMessage{},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]Entry{}
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry
	}
	ownerPath := "PC01/" + owner.OwnerPath
	patched := byPath[ownerPath].Bytes
	if !bytes.Contains(patched, []byte("pc01_social_liberalism = {")) || !bytes.Contains(patched, []byte("conservatism = {")) {
		t.Fatalf("bundled vanilla owner was not preserved and patched at %s:\n%s", ownerPath, patched)
	}
	if strings.Contains(string(byPath["PC01.mod"].Bytes), `replace_path="common/ideologies"`) {
		t.Fatal("bundled vanilla subtype incorrectly requested directory takeover")
	}

	destination := os.Getenv("P0_POLITICS_INHERITED_ZIP_OUT")
	if destination == "" {
		destination = filepath.Join(t.TempDir(), "P0_POLITICS_INHERITED_SOURCE.zip")
	}
	manifestOut, err := Build(destination, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write inherited-source ZIP: %v", err)
	}
	zipEntries := pb04ZIPEntries(t, destination)
	for _, entry := range artifact.Entries {
		if !bytes.Equal(zipEntries[entry.Path], entry.Bytes) {
			t.Fatalf("inherited-source ZIP readback differs at %s", entry.Path)
		}
	}
	duplicate := filepath.Join(t.TempDir(), "determinism.zip")
	second, err := Build(duplicate, snapshot.Fingerprint, artifact.Entries, DefaultArtifactBudget())
	if err != nil {
		t.Fatalf("write deterministic inherited-source comparison ZIP: %v", err)
	}
	if second.ZIPSHA256 != manifestOut.ZIPSHA256 || second.ZIPBytes != manifestOut.ZIPBytes {
		t.Fatalf("inherited-source ZIP is not deterministic: first=%+v second=%+v", manifestOut, second)
	}
	t.Logf("inherited-source ZIP=%s bytes=%d sha256=%s", destination, manifestOut.ZIPBytes, manifestOut.ZIPSHA256)
}

func TestBuildNativeShadowInheritedIdeologySourceStableBlockers(t *testing.T) {
	source := []byte("ideologies = { democratic = { types = { conservatism = { } } color = { 0 0 255 } } }\n")

	t.Run("stale lock is applied to the re-imported source", func(t *testing.T) {
		locked, _, _ := ideologySourceNativeSnapshot(t, source, "stale", false)
		reimported := []byte("ideologies = { democratic = { types = { conservatism = { } liberalism = { } } color = { 0 0 255 } } }\n")
		snapshot, archive, key := ideologySourceNativeSnapshot(t, reimported, "stale", true)
		snapshot.Project = locked.Project
		artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(got string, _ int64) ([]byte, error) {
			if got != key {
				t.Fatalf("key=%q", got)
			}
			return archive, nil
		})
		if err != nil {
			t.Fatalf("stale lock must not fail the export: %v", err)
		}
		var patched []byte
		for _, entry := range artifact.Entries {
			if strings.HasSuffix(entry.Path, "/"+ideologySourceNativePath) {
				patched = entry.Bytes
			}
		}
		if !bytes.Contains(patched, []byte("demo_reformism = {")) || !bytes.Contains(patched, []byte("liberalism = {")) {
			t.Fatalf("want the subtype on the re-imported file:\n%s", patched)
		}
		relocked := false
		for _, warning := range artifact.Warnings {
			if strings.HasPrefix(warning, "ideology_source_skipped:") {
				t.Fatalf("relocked subtype reported skipped: %s", warning)
			}
			relocked = relocked || strings.HasPrefix(warning, "ideology_source_relocked:") && strings.Contains(warning, "demo_reformism")
		}
		if !relocked {
			t.Fatalf("missing relocked note: %v", artifact.Warnings)
		}
	})
	t.Run("stale lock whose ROOT is gone is left out and says why", func(t *testing.T) {
		locked, _, _ := ideologySourceNativeSnapshot(t, source, "stale", false)
		reimported := []byte("ideologies = { fascism = { types = { nazism = { } } color = { 0 0 0 } } }\n")
		snapshot, archive, _ := ideologySourceNativeSnapshot(t, reimported, "stale", true)
		snapshot.Project = locked.Project
		artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return archive, nil })
		if err != nil {
			t.Fatalf("stale lock must not fail the export: %v", err)
		}
		warned := false
		for _, warning := range artifact.Warnings {
			warned = warned || strings.HasPrefix(warning, "ideology_source_skipped:") && strings.Contains(warning, "ideology_source_owner_missing") && strings.Contains(warning, "demo_reformism")
		}
		if !warned {
			t.Fatalf("missing skipped warning with the reason: %v", artifact.Warnings)
		}
	})

	t.Run("stale lock that cannot be relocked is left out with a warning", func(t *testing.T) {
		snapshot, _, _ := ideologySourceNativeSnapshot(t, source, "stale", false)
		artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
		if err != nil {
			t.Fatalf("stale lock must not fail the export: %v", err)
		}
		warned := false
		for _, warning := range artifact.Warnings {
			if strings.HasPrefix(warning, "ideology_source_skipped:") && strings.Contains(warning, "imported again") && strings.Contains(warning, "demo_reformism") {
				warned = true
			}
		}
		if !warned {
			t.Fatalf("missing skipped-subtype warning: %v", artifact.Warnings)
		}
		for _, entry := range artifact.Entries {
			if strings.Contains(entry.SourceEntity, "ideology_source:") {
				t.Fatalf("stale lock still emitted %s", entry.Path)
			}
		}
	})

	t.Run("stale manifest without entries is no manifest", func(t *testing.T) {
		snapshot, _, _ := ideologySourceNativeSnapshot(t, source, "stale", false)
		var project map[string]any
		if err := json.Unmarshal(snapshot.Project, &project); err != nil {
			t.Fatal(err)
		}
		project["ideology_source_manifest"] = map[string]any{"state": "stale", "source_key": "imported", "snapshot_updated_at": "2026-10-23T15:00:49Z"}
		raw, err := json.Marshal(project)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.Project = raw
		artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
		if err != nil {
			t.Fatalf("an empty stale manifest must not fail the export: %v", err)
		}
		for _, warning := range artifact.Warnings {
			if strings.HasPrefix(warning, "ideology_source_skipped:") {
				t.Fatalf("nothing was locked, so nothing should be reported skipped: %s", warning)
			}
		}
	})
	t.Run("snapshot missing", func(t *testing.T) {
		snapshot, _, _ := ideologySourceNativeSnapshot(t, source, "ready", false)
		_, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
		if ideologySourceNativeErrorCode(err) != "ideology_source_snapshot_missing" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("resolver missing", func(t *testing.T) {
		snapshot, _, _ := ideologySourceNativeSnapshot(t, source, "ready", true)
		_, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
		if ideologySourceNativeErrorCode(err) != "ideology_source_snapshot_resolver_required" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("archive hash drift", func(t *testing.T) {
		snapshot, _, key := ideologySourceNativeSnapshot(t, source, "ready", true)
		drifted := ideologySourceArchive(t, map[string][]byte{ideologySourceNativePath: []byte("changed")})
		_, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(got string, _ int64) ([]byte, error) {
			if got != key {
				t.Fatalf("key=%q", got)
			}
			return drifted, nil
		})
		if ideologySourceNativeErrorCode(err) != "ideology_source_snapshot_archive_drift" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("archive member set differs", func(t *testing.T) {
		snapshot, _, key := ideologySourceNativeSnapshot(t, source, "ready", true)
		unexpected := ideologySourceArchive(t, map[string][]byte{
			ideologySourceNativePath:                      source,
			"common/ideologies/unexpected_ideologies.txt": source,
		})
		_, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(got string, _ int64) ([]byte, error) {
			if got != key {
				t.Fatalf("key=%q", got)
			}
			return unexpected, nil
		})
		if ideologySourceNativeErrorCode(err) != "ideology_source_snapshot_archive_invalid" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("archive member missing", func(t *testing.T) {
		snapshot, _, key := ideologySourceNativeSnapshot(t, source, "ready", true)
		missing := ideologySourceArchive(t, map[string][]byte{})
		_, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(got string, _ int64) ([]byte, error) {
			if got != key {
				t.Fatalf("key=%q", got)
			}
			return missing, nil
		})
		if ideologySourceNativeErrorCode(err) != "ideology_source_snapshot_archive_drift" {
			t.Fatalf("err=%v", err)
		}
	})

	skipped := func(t *testing.T, artifact NativeArtifact, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		warned := false
		for _, warning := range artifact.Warnings {
			if strings.HasPrefix(warning, "ideology_source_skipped:") && strings.Contains(warning, "demo_reformism") {
				warned = true
			}
		}
		if !warned {
			t.Fatalf("warnings=%v", artifact.Warnings)
		}
		for _, entry := range artifact.Entries {
			if bytes.Contains(entry.Bytes, []byte("demo_reformism = {")) {
				t.Fatalf("inherited subtype shipped in %s", entry.Path)
			}
		}
	}
	t.Run("the project's own ideology file wins", func(t *testing.T) {
		snapshot, archive, key := ideologySourceNativeSnapshot(t, source, "ready", true)
		snapshot.Tables["project_raw_files"] = []json.RawMessage{json.RawMessage(`{
			"id":9,"dest_path":"common/ideologies/authored.txt","content":"ideologies = { }",
			"source":"created","raw_kind":"verbatim_override","is_binary":false
		}`)}
		artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(got string, _ int64) ([]byte, error) {
			if got != key {
				t.Fatalf("key=%q", got)
			}
			return archive, nil
		})
		skipped(t, artifact, err)
		authored := false
		for _, entry := range artifact.Entries {
			if strings.HasSuffix(entry.Path, "common/ideologies/authored.txt") {
				authored = true
			}
		}
		if !authored {
			t.Fatal("the project's own ideology file was not shipped")
		}
	})
	t.Run("replace_path=common/ideologies wins", func(t *testing.T) {
		snapshot, archive, key := ideologySourceNativeSnapshot(t, source, "ready", true)
		plan, err := BuildTimelinePlanFromSnapshot(snapshot, []string{"common/ideologies"}, nil, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := BuildNativeShadowWithTimelinePlan(snapshot, plan, false, func(got string, _ int64) ([]byte, error) {
			if got != key {
				t.Fatalf("key=%q", got)
			}
			return archive, nil
		}, english)
		skipped(t, artifact, err)
	})
}

func TestBuildNativeShadowKeepsProjectRootBesideInheritedSource(t *testing.T) {
	tables, err := testWorldTables(t)
	if err != nil {
		t.Fatal(err)
	}
	layers := []ideologysource.Layer{{
		SourceKind: ideologysource.SourceVanilla, LoadOrder: 0,
		Files: []ideologysource.SourceFile{{Path: ideologySourceNativePath, Bytes: []byte(tables.IdeologyBaseText)}},
	}}
	discovery, err := ideologysource.Discover(layers)
	if err != nil {
		t.Fatal(err)
	}
	var owner ideologysource.RootOwner
	for _, candidate := range discovery.Roots {
		if candidate.RootID == "neutrality" {
			owner = candidate
		}
	}
	if owner.RootID == "" {
		t.Fatal("bundled vanilla snapshot has no neutrality ROOT")
	}
	manifest := ideologysource.Manifest{SchemaVersion: ideologysource.SchemaVersion, Entries: []ideologysource.ManifestEntry{{
		SourceKind: owner.SourceKind, OwnerPath: owner.OwnerPath,
		FileSHA256: owner.FileSHA256, RootSHA256: owner.RootSHA256,
		ScriptID: "qa_corporatist_monarchy", ParentRootID: owner.RootID,
	}}}
	project := ideologySourceNativeProject(t, manifest, []map[string]any{
		{"id": "qa_corporatist_monarchy", "group": "neutrality", "source": "project_managed",
			"name": map[string]string{"en": "Corporatist Monarchy"}},
		{"id": "qa_revolutionary_syndicalism", "group": "qa_syndicalism", "source": "project_managed",
			"name": map[string]string{"en": "Revolutionary Syndicalism"}},
	})
	var decoded map[string]any
	if err := json.Unmarshal(project, &decoded); err != nil {
		t.Fatal(err)
	}
	registry := decoded["ideology_registry"].(map[string]any)
	registry["groups"] = []string{"qa_syndicalism"}
	registry["root_definitions"] = []map[string]any{{
		"id": "qa_syndicalism", "source": "project_managed",
		"body": "types = {\n}\ncolor = { 200 120 0 }\nrules = {\n\tcan_force_government = yes\n}\nai_neutral = yes",
		"name": map[string]string{"en": "Syndicalist"}, "desc": map[string]string{"en": "Union Rule"},
	}}
	decoded["id"], decoded["name"], decoded["country_tag"] = 459, "QA459", "POR"
	snapshot := Snapshot{
		ProjectID:   459,
		Fingerprint: ideologysource.Hash([]byte(tables.IdeologyBaseText)),
		Project:     mustJSON(t, decoded),
		Tables:      map[string][]json.RawMessage{},
	}
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string][]byte{}
	for _, entry := range artifact.Entries {
		byPath[entry.Path] = entry.Bytes
	}
	if patched := string(byPath["QA459/"+owner.OwnerPath]); !strings.Contains(patched, "qa_corporatist_monarchy = {") {
		t.Fatalf("inherited subtype missing from %s:\n%s", owner.OwnerPath, patched)
	}
	roots := string(byPath["QA459/common/ideologies/zz_QA459_ideologies.txt"])
	for _, want := range []string{"\tqa_syndicalism = {", "\t\tqa_revolutionary_syndicalism = {", "color = { 200 120 0 }"} {
		if !strings.Contains(roots, want) {
			t.Errorf("project ROOT file missing %q:\n%s", want, roots)
		}
	}
	if strings.Contains(roots, "\tneutrality = {") {
		t.Errorf("project ROOT file redefines the inherited neutrality ROOT:\n%s", roots)
	}
	loc := string(byPath["QA459/localisation/english/replace/zz_QA459_ideology_l_english.yml"])
	for _, want := range []string{`qa_syndicalism:0 "Syndicalist"`, `qa_syndicalism_desc:0 "Union Rule"`, `qa_revolutionary_syndicalism:0 "Revolutionary Syndicalism"`} {
		if !strings.Contains(loc, want) {
			t.Errorf("ROOT localisation missing %q:\n%s", want, loc)
		}
	}

	if strings.Contains(string(byPath["QA459.mod"]), `replace_path="common/ideologies"`) {
		t.Error("the inherited source's directory was taken over with replace_path")
	}
	for path, body := range byPath {
		if strings.Contains(string(body), "common/ideologies NOT emitted") {
			t.Errorf("%s still says the ideologies were not emitted", path)
		}
	}
}
