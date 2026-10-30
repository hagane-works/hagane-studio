package exportmerge

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/cwtools"
	"haganeworks/internal/hoi4/exportengine"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/exportplan/timeline"
)

var english = exportmsg.For("en")

func useCWTools(t *testing.T) {
	t.Helper()
	dir, _ := filepath.Abs(filepath.Join("..", "..", "..", filepath.FromSlash(cwtools.Folder)))
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(cwtools.ModifiersFile))); err != nil {
		t.Skipf("no CWTools rules: %v", err)
	}
	cwtools.SetDir(dir)
	t.Cleanup(func() { cwtools.SetDir("") })
}

func TestPythonContractDecisionConflictAndResolution(t *testing.T) {
	baseline := cloneJSON(snapshotDefaults["decision"]).(map[string]any)
	baseline["decision_id"] = "usa_w3_decide"
	baseline["cost"] = int64(50)
	baseline["complete_effect"] = "add_pp = 50"
	row := cloneJSON(baseline).(map[string]any)
	row["source_snapshot"] = cloneJSON(baseline)
	row["cost"] = int64(100)
	snapshot := snapshotWithRows(t, "decisions", row)
	target := NewTargetMod()
	target.Decisions["usa_w3_decide"] = map[string]any{
		"decision_id": "usa_w3_decide", "category_id": "usa_w3_cat",
		"cost": int64(300), "complete_effect": "add_pp = 50",
	}

	preview, err := BuildPreview(snapshot, target)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Totals.Conflict != 1 || preview.Totals.TotalConflictPaths != 1 {
		t.Fatalf("Python fixture totals mismatch: %+v", preview.Totals)
	}
	reports := preview.Reports["decision"]
	if len(reports) != 1 || reports[0].Conflicts[0].Path != "cost" {
		t.Fatalf("Python fixture report mismatch: %#v", reports)
	}
	final, _, err := ComputeFinalOps(snapshot, target, ResolutionRequest{Resolutions: map[string]string{
		"decision/usa_w3_decide/cost": "keep_user",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := final["decision/usa_w3_decide"]; len(got) != 1 || got[0]["value"] != int64(100) {
		t.Fatalf("keep_user parity mismatch: %#v", got)
	}
}

func TestPythonFixtureParityAllDiffableEntities(t *testing.T) {
	cases := []struct {
		kind, table, idField, id, field string
		userValue, targetValue          any
	}{
		{kind: "character", table: "characters", idField: "tag", id: "USA_fixture", field: "skill", userValue: int64(2), targetValue: int64(3)},
		{kind: "event", table: "events", idField: "event_id", id: "usa.1", field: "timeout_days", userValue: int64(2), targetValue: int64(3)},
		{kind: "decision", table: "decisions", idField: "decision_id", id: "usa_decide", field: "cost", userValue: int64(2), targetValue: int64(3)},
		{kind: "focus_node", table: "focus_nodes", idField: "focus_id", id: "usa_focus", field: "cost", userValue: float64(5), targetValue: float64(6)},
		{kind: "idea", table: "ideas", idField: "idea_id", id: "usa_idea", field: "level", userValue: int64(2), targetValue: int64(3)},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			baseline := cloneJSON(snapshotDefaults[tc.kind]).(map[string]any)
			baseline[tc.idField] = tc.id
			row := cloneJSON(baseline).(map[string]any)
			row["source_snapshot"] = cloneJSON(baseline)
			row[tc.field] = tc.userValue
			targetRow := cloneJSON(baseline).(map[string]any)
			targetRow[tc.field] = tc.targetValue
			target := NewTargetMod()
			target.ForType(tc.kind)[tc.id] = targetRow

			preview, err := BuildPreview(snapshotWithRows(t, tc.table, row), target)
			if err != nil {
				t.Fatal(err)
			}
			if preview.Totals.Conflict != 1 || preview.Totals.TotalConflictPaths != 1 {
				t.Fatalf("Python fixture totals mismatch: %+v", preview.Totals)
			}
			reports := preview.Reports[tc.kind]
			if len(reports) != 1 || reports[0].Status != "conflict" || len(reports[0].Conflicts) != 1 || reports[0].Conflicts[0].Path != tc.field {
				t.Fatalf("Python fixture report mismatch: %#v", reports)
			}
		})
	}
}

func TestDiffExportZIPRoundTripAndAtomicPublish(t *testing.T) {
	base := writeZIP(t, map[string][]byte{
		"descriptor.mod": []byte("name=\"W3\"\n"),
		"common/decisions/usa_w3.txt": []byte(`usa_w3_cat = {
    usa_w3_decide = {
        cost = 300
        complete_effect = { add_pp = 50 }
    }
}`),
	})
	archive, err := ReadArchive(context.Background(), base, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	target := ParseTarget(context.Background(), archive)
	if got := target.Decisions["usa_w3_decide"]; got == nil || asString(got["cost"]) != "300" {
		t.Fatalf("target parser did not recover decision: %#v warnings=%v", got, target.Warnings)
	}
	destination := filepath.Join(t.TempDir(), "merged.zip")
	manifest, err := ApplyOpsToArchive(context.Background(), archive, target, map[string][]DiffOp{
		"decision/usa_w3_decide": {{"op": "replace", "path": "cost", "old": int64(300), "value": int64(100)}},
	}, "USA", destination, "fixture-fingerprint")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ZIPBytes == 0 || manifest.ManifestPath == "" {
		t.Fatalf("missing artifact manifest: %+v", manifest)
	}
	merged, err := ReadArchive(context.Background(), destination, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	parsed := ParseTarget(context.Background(), merged)
	if got := asString(parsed.Decisions["usa_w3_decide"]["cost"]); got != "100" {
		t.Fatalf("merged cost=%s want 100; warnings=%v", got, parsed.Warnings)
	}
	if string(merged.ByPath["descriptor.mod"]) != "name=\"W3\"\n" {
		t.Fatal("untouched member was not byte-preserved")
	}

	previous, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyOpsToArchive(context.Background(), archive, target, map[string][]DiffOp{
		"unknown/id": {{"op": "add", "path": "x", "value": 1}},
	}, "USA", destination, "bad")
	if err == nil {
		t.Fatal("expected malformed type failure")
	}
	after, _ := os.ReadFile(destination)
	if string(previous) != string(after) {
		t.Fatal("failed merge replaced the previous artifact")
	}
}

func TestReadArchiveSafetyBudgetsAndCancellation(t *testing.T) {
	unsafe := writeZIP(t, map[string][]byte{"../escape.txt": []byte("no")})
	if _, err := ReadArchive(context.Background(), unsafe, DefaultArchiveBudget()); !errors.Is(err, ErrUnsafeArchive) {
		t.Fatalf("traversal err=%v", err)
	}

	tooMany := writeZIP(t, map[string][]byte{"a": []byte("a"), "b": []byte("b")})
	budget := DefaultArchiveBudget()
	budget.MaxEntries = 1
	if _, err := ReadArchive(context.Background(), tooMany, budget); !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("entry budget err=%v", err)
	}

	zipBomb := writeZIP(t, map[string][]byte{"bomb.bin": bytes.Repeat([]byte{0}, 2<<20)})
	if _, err := ReadArchive(context.Background(), zipBomb, DefaultArchiveBudget()); !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("compression-ratio budget err=%v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	valid := writeZIP(t, map[string][]byte{"a.txt": []byte("a")})
	if _, err := ReadArchive(canceled, valid, DefaultArchiveBudget()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel err=%v", err)
	}
}

func TestValidateResolutionRequestPythonStatusContract(t *testing.T) {
	for _, tc := range []ResolutionRequest{
		{Resolutions: map[string]string{"bad": "keep_user"}},
		{Resolutions: map[string]string{"idea/x/name": "manual"}},
		{Resolutions: map[string]string{"idea/x/name": "explode"}},
	} {
		if err := ValidateResolutionRequest(tc); err == nil {
			t.Fatalf("accepted invalid payload %#v", tc)
		}
	}
}

func TestAppendPythonContractRerootDescriptorAndOnActions(t *testing.T) {
	baseFile := writeZIP(t, map[string][]byte{
		"Base/descriptor.mod":                       []byte("version = \"2\"\nname = \"Base Name\"\ntags = { \"Base\" }\nreplace_path = \"common/x\"\n"),
		"Base/common/on_actions/USA_on_actions.txt": []byte("base_hook = yes\n"),
		"Base/common/decisions/x.txt":               []byte("base decision"),
		"Base/common/ideas/Case.txt":                []byte("base case"),
	})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	platform := []exportengine.Entry{
		{Path: "Platform.mod", Bytes: []byte("install metadata")},
		{Path: "Platform/descriptor.mod", Bytes: []byte("version = \"1\"\nname = \"Platform\"\ntags = { \"Base\" \"Added\" }\ndependencies = { \"DLC\" }\n")},
		{Path: "Platform/common/on_actions/USA_on_actions.txt", Bytes: []byte("platform_hook = yes\n")},
		{Path: "Platform/common/decisions/x.txt", Bytes: []byte("platform decision")},
		{Path: "Platform/common/ideas/case.txt", Bytes: []byte("platform case")},
		{Path: "Platform/common/technologies/y.txt", Bytes: []byte("platform technology")},
	}
	destination := filepath.Join(t.TempDir(), "append.zip")
	manifest, conflicts, err := MergeAppend(context.Background(), base, platform, destination, "append-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ZIPBytes == 0 {
		t.Fatal("empty append artifact")
	}
	if len(conflicts) != 1 || conflicts[0] != "Base/common/decisions/x.txt" {
		t.Fatalf("conflicts=%v", conflicts)
	}
	merged, err := ReadArchive(context.Background(), destination, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := merged.ByPath["Platform.mod"]; exists {
		t.Fatal("root platform metadata survived reroot")
	}
	if string(merged.ByPath["Base/common/on_actions/USA_on_actions.txt"]) != "base_hook = yes\n" ||
		string(merged.ByPath["Base/common/on_actions/USA_hagane_on_actions.txt"]) != "platform_hook = yes\n" {
		t.Fatal("on_actions collision was destructive")
	}
	if string(merged.ByPath["Base/common/decisions/x.txt"]) != "platform decision" {
		t.Fatal("platform-wins conflict policy not preserved")
	}
	if string(merged.ByPath["Base/common/ideas/Case.txt"]) != "platform case" {
		t.Fatal("case-folded platform winner did not retain the base path spelling")
	}
	descriptor := string(merged.ByPath["Base/descriptor.mod"])
	for _, want := range []string{`version="2"`, `name="Base Name"`, `"Base"`, `"Added"`, `"DLC"`, `replace_path="common/x"`} {
		if !strings.Contains(descriptor, want) {
			t.Fatalf("descriptor missing %q:\n%s", want, descriptor)
		}
	}
}

func TestAdvancedExportsReplaceStaleFindingsWithCurrentEvidence(t *testing.T) {
	staleDoc, err := exportengine.NewExportFindingsDocument([]exportengine.ExportFinding{{
		Code: "stale", Source: json.RawMessage(`{"producer":"old"}`), Message: "stale finding",
	}})
	if err != nil {
		t.Fatal(err)
	}
	staleBytes, _ := json.Marshal(staleDoc)
	baseFile := writeZIP(t, map[string][]byte{
		"Demo/descriptor.mod":                         []byte("name=Demo\n"),
		"Demo/" + exportengine.ExportFindingsBasename: staleBytes,
	})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	current := []exportengine.ExportFinding{{
		Code: "current", Source: json.RawMessage(`{"producer":"advanced"}`), Message: "current finding", Acknowledged: true,
	}}

	diffPath := filepath.Join(t.TempDir(), "diff-current.zip")
	diffManifest, err := ApplyOpsToArchiveWithFindings(context.Background(), base, nil, map[string][]DiffOp{}, "DEMO", diffPath, "", current, english)
	if err != nil {
		t.Fatal(err)
	}
	diffDoc, _, err := exportengine.ReadExportFindingsFromZIP(diffPath)
	if err != nil || len(diffDoc.Findings) != 1 || diffDoc.Findings[0].Code != "current" {
		t.Fatalf("diff findings=%+v err=%v", diffDoc.Findings, err)
	}
	secondPath := filepath.Join(t.TempDir(), "diff-other-finding.zip")
	other := append([]exportengine.ExportFinding(nil), current...)
	other[0].Code = "other-current"
	secondManifest, err := ApplyOpsToArchiveWithFindings(context.Background(), base, nil, map[string][]DiffOp{}, "DEMO", secondPath, "", other, english)
	if err != nil {
		t.Fatal(err)
	}
	if diffManifest.Fingerprint == secondManifest.Fingerprint {
		t.Fatal("advanced diff identity ignored the persisted finding set")
	}

	appendPath := filepath.Join(t.TempDir(), "append-current.zip")
	if _, _, err := MergeAppendWithFindings(context.Background(), base, []exportengine.Entry{{
		Path: "Platform/descriptor.mod", Bytes: []byte("name=Platform\n"),
	}}, appendPath, "append-findings", current, english); err != nil {
		t.Fatal(err)
	}
	appendDoc, _, err := exportengine.ReadExportFindingsFromZIP(appendPath)
	if err != nil || len(appendDoc.Findings) != 1 || appendDoc.Findings[0].Code != "current" {
		t.Fatalf("append findings=%+v err=%v", appendDoc.Findings, err)
	}
}

func TestDiffFindingsRescanExactFinalWinners(t *testing.T) {
	useCWTools(t)
	baseFile := writeZIP(t, map[string][]byte{
		"Demo/descriptor.mod":                         []byte("name=Demo\n"),
		"Demo/common/ideas/literal.txt":               []byte("ideas = { country = { bad = { } } }\\nset_variable = { x = 1 }\n"),
		"Demo/common/ideas/registry.txt":              []byte("ideas = { country = { bad = { modifier = { skill_card_taught_invalid = 1 } } } }\n"),
		"Demo/localisation/english/bad_l_english.yml": []byte("l_english:\n BAD KEY:0 \"unterminated\n"),
	})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "diff-rescanned.zip")
	if _, err := ApplyOpsToArchiveWithFindings(context.Background(), base, nil, map[string][]DiffOp{}, "DEMO", destination, "", nil, english); err != nil {
		t.Fatal(err)
	}
	doc, _, err := exportengine.ReadExportFindingsFromZIP(destination)
	if err != nil {
		t.Fatal(err)
	}
	codes := exportFindingCodes(doc.Findings)
	for _, want := range []string{"script.literal_newline", "localisation.missing_utf8_bom", "localisation.invalid_key", "modifier.unknown"} {
		if !codes[want] {
			t.Errorf("final diff scan missed %s; findings=%+v", want, doc.Findings)
		}
	}
}

func TestAdvancedFindingsModePublishesOneEmptyCurrentReport(t *testing.T) {
	baseFile := writeZIP(t, map[string][]byte{"Demo/descriptor.mod": []byte("name=Demo\n")})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "diff-clean-findings.zip")
	if _, err := ApplyOpsToArchiveWithFindings(context.Background(), base, nil, map[string][]DiffOp{}, "DEMO", destination, "", nil, english); err != nil {
		t.Fatal(err)
	}
	doc, _, err := exportengine.ReadExportFindingsFromZIP(destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Findings) != 0 {
		t.Fatalf("clean final artifact findings=%+v", doc.Findings)
	}
}

func TestAppendFindingsRescanMergedArtifactAndPreserveSnapshotOnlyEvidence(t *testing.T) {
	useCWTools(t)
	baseFile := writeZIP(t, map[string][]byte{
		"Base/descriptor.mod":          []byte("name=Base\n"),
		"Base/common/ideas/winner.txt": []byte("ideas = { country = { old = { modifier = { political_power_factor = 0.1 } } } }\n"),
	})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	stalePath := "common/ideas/stale.txt"
	staleLine := 7
	inherited, err := exportengine.NewExportFindingsDocument([]exportengine.ExportFinding{
		{
			Code: "clausewitz.parse_failed", Source: json.RawMessage(`{"kind":"final_artifact_syntax","identifier":"stale"}`),
			Path: &stalePath, Line: &staleLine, Message: "stale pre-merge result",
		},
		{
			Code: "reference.missing", Source: json.RawMessage(`{"kind":"reference_integrity","identifier":"demo.flag"}`),
			Path: &stalePath, Line: &staleLine, Message: "snapshot-backed relationship evidence",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	inheritedBytes, _ := json.Marshal(inherited)
	platform := []exportengine.Entry{
		{Path: "Platform/descriptor.mod", Bytes: []byte("name=Platform\n")},
		{Path: "Platform/" + exportengine.ExportFindingsBasename, Bytes: inheritedBytes},
		{Path: "Platform/common/ideas/winner.txt", Bytes: []byte("ideas = { country = { new = { } } }\\nset_variable = { x = 1 }\n")},
		{Path: "Platform/common/ideas/registry.txt", Bytes: []byte("ideas = { country = { bad = { modifier = { skill_card_taught_invalid = 1 } } } }\n")},
		{Path: "Platform/localisation/english/bad_l_english.yml", Bytes: []byte("l_english:\n BAD KEY:0 \"unterminated\n")},
	}
	destination := filepath.Join(t.TempDir(), "append-rescanned.zip")
	if _, _, err := MergeAppendWithFindings(context.Background(), base, platform, destination, "", nil, english); err != nil {
		t.Fatal(err)
	}
	doc, _, err := exportengine.ReadExportFindingsFromZIP(destination)
	if err != nil {
		t.Fatal(err)
	}
	codes := exportFindingCodes(doc.Findings)
	for _, want := range []string{"reference.missing", "script.literal_newline", "localisation.missing_utf8_bom", "modifier.unknown"} {
		if !codes[want] {
			t.Errorf("final append scan missed %s; findings=%+v", want, doc.Findings)
		}
	}
	for _, finding := range doc.Findings {
		if finding.Code == "clausewitz.parse_failed" && finding.Message == "stale pre-merge result" {
			t.Fatalf("stale pre-merge syntax finding survived: %+v", finding)
		}
	}
}

func TestAppendWithSnapshotPreservesTimelineManifestIdentity(t *testing.T) {
	baseFile := writeZIP(t, map[string][]byte{"Base/descriptor.mod": []byte("name=Base\n")})
	base, err := ReadArchive(context.Background(), baseFile, DefaultArchiveBudget())
	if err != nil {
		t.Fatal(err)
	}
	metadata := &timeline.ArtifactMetadata{TimelinePlanSHA256: "timeline-proof"}
	destination := filepath.Join(t.TempDir(), "append-timeline.zip")
	snapshot := snapshotWithRows(t, "bookmarks")
	if _, _, err := MergeAppendWithFindingsAndSnapshot(context.Background(), base, []exportengine.Entry{{
		Path: "Platform/descriptor.mod", Bytes: []byte("name=Platform\n"),
	}}, destination, "append-timeline", nil, snapshot, metadata, english); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != "Base/"+exportengine.ArtifactManifestBasename {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		var embedded exportengine.Manifest
		err = json.NewDecoder(rc).Decode(&embedded)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if embedded.Timeline == nil || embedded.Timeline.TimelinePlanSHA256 != "timeline-proof" {
			t.Fatalf("timeline metadata=%+v", embedded.Timeline)
		}
		return
	}
	t.Fatal("embedded artifact manifest missing")
}

func exportFindingCodes(findings []exportengine.ExportFinding) map[string]bool {
	out := make(map[string]bool, len(findings))
	for _, finding := range findings {
		out[finding.Code] = true
	}
	return out
}

func snapshotWithRows(t *testing.T, table string, rows ...map[string]any) exportengine.Snapshot {
	t.Helper()
	snapshot := exportengine.Snapshot{Schema: exportengine.SnapshotSchema, ProjectID: 182, Project: json.RawMessage(`{"id":182,"name":"fixture","country_tag":"USA"}`), Tables: map[string][]json.RawMessage{}}
	for _, row := range rows {
		raw, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.Tables[table] = append(snapshot.Tables[table], raw)
	}
	fingerprint, err := exportengine.SnapshotFingerprint(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Fingerprint = fingerprint
	return snapshot
}

func writeZIP(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "fixture.zip")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for name, body := range entries {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}
