package exportengine

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExportFindingsRoundTripFromDownloadedZIP(t *testing.T) {
	pathValue, lineValue := "events/demo.txt", 17
	findings := []ExportFinding{{
		Code: "event.reference_missing", Severity: "warning",
		Source: json.RawMessage(`{"kind":"event","identifier":"demo.7","provider":"created"}`),
		Path:   &pathValue, Line: &lineValue,
		Message:        "事件引用了未登记目标 demo.8",
		PlayerImpact:   "玩家触发选项后可能看不到后续事件。",
		EditRoute:      "到事件编辑器检查 demo.7 的第二个选项。",
		Acknowledged:   true,
		SourceEvidence: map[string]any{"target": "demo.8"},
	}}
	entries := []Entry{
		{Path: "Demo.mod", Bytes: []byte("name=Demo\n")},
		{Path: "Demo/descriptor.mod", Bytes: []byte("name=Demo\n")},
		{Path: "Demo/events/demo.txt", Bytes: []byte("add_namespace = demo\n")},
	}
	reportEntry, err := ExportFindingsEntry(entries, findings)
	if err != nil {
		t.Fatal(err)
	}
	entries = append(entries, reportEntry)
	destination := filepath.Join(t.TempDir(), "demo.zip")
	if _, err := Build(destination, "fp", entries, DefaultArtifactBudget()); err != nil {
		t.Fatal(err)
	}
	doc, reportPath, err := ReadExportFindingsFromZIP(destination)
	if err != nil {
		t.Fatal(err)
	}
	if reportPath != "Demo/"+ExportFindingsBasename {
		t.Fatalf("report path=%q", reportPath)
	}
	if doc.Schema != ExportFindingsSchema || len(doc.Findings) != 1 {
		t.Fatalf("document=%+v", doc)
	}
	got := doc.Findings[0]
	var gotSource, wantSource map[string]any
	if err := json.Unmarshal(got.Source, &gotSource); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(findings[0].Source, &wantSource); err != nil {
		t.Fatal(err)
	}
	if got.Code != findings[0].Code || !reflect.DeepEqual(gotSource, wantSource) ||
		got.Path == nil || *got.Path != pathValue || got.Line == nil || *got.Line != lineValue ||
		got.Message != findings[0].Message || got.PlayerImpact != findings[0].PlayerImpact ||
		got.EditRoute != findings[0].EditRoute || !got.Acknowledged ||
		!reflect.DeepEqual(got.SourceEvidence, findings[0].SourceEvidence) {
		t.Fatalf("finding did not survive ZIP readback:\n got=%+v\nwant=%+v", got, findings[0])
	}
}

func TestAcknowledgementDoesNotSuppressFreshUnacknowledgedRecheck(t *testing.T) {
	entries := []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("name=Demo\n")},
		{Path: "Demo/events/demo.txt", Bytes: []byte("option = { add_political_power = 1\\nadd_stability = 0.1 }\n")},
	}
	first := ExportFindingsFromLiteralNewlines(ScanLiteralNewlines(entries, english), true)
	if len(first) != 1 || !first[0].Acknowledged {
		t.Fatalf("acknowledged first result=%+v", first)
	}
	reportEntry, err := ExportFindingsEntry(entries, first)
	if err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "acknowledged.zip")
	if _, err := Build(zipPath, "fp", append(entries, reportEntry), DefaultArtifactBudget()); err != nil {
		t.Fatal(err)
	}
	persisted, _, err := ReadExportFindingsFromZIP(zipPath)
	if err != nil || !persisted.Findings[0].Acknowledged {
		t.Fatalf("persisted acknowledgement=%+v err=%v", persisted, err)
	}

	fresh := ExportFindingsFromLiteralNewlines(ScanLiteralNewlines(entries, english), false)
	if len(fresh) != 1 || fresh[0].Code != persisted.Findings[0].Code ||
		*fresh[0].Path != *persisted.Findings[0].Path || *fresh[0].Line != *persisted.Findings[0].Line ||
		fresh[0].Acknowledged {
		t.Fatalf("fresh recheck was suppressed or inherited acknowledgement: %+v", fresh)
	}
}

func TestExportFindingsRejectsUntraceableOrCollidingRecords(t *testing.T) {
	entries := []Entry{{Path: "Demo/descriptor.mod", Bytes: []byte("name=Demo\n")}}
	if _, err := ExportFindingsEntry(entries, []ExportFinding{{Message: "missing source"}}); err == nil {
		t.Fatal("untraceable finding must be rejected")
	}
	entries = append(entries, Entry{Path: "Demo/" + ExportFindingsBasename, Bytes: []byte("{}")})
	if _, err := ExportFindingsEntry(entries, nil); err == nil {
		t.Fatal("reserved report collision must not overwrite author bytes")
	}
}

func TestReadExportFindingsRejectsMissingOrUnknownSchema(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.zip")
	f, err := os.Create(missing)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	writer, _ := zw.Create("Demo/descriptor.mod")
	_, _ = writer.Write([]byte("name=Demo\n"))
	_ = zw.Close()
	_ = f.Close()
	if _, _, err := ReadExportFindingsFromZIP(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing report error=%v", err)
	}

	unknown := filepath.Join(t.TempDir(), "unknown.zip")
	f, _ = os.Create(unknown)
	zw = zip.NewWriter(f)
	writer, _ = zw.Create("Demo/" + ExportFindingsBasename)
	_, _ = writer.Write([]byte(`{"schema":"future.v9","findings":[]}`))
	_ = zw.Close()
	_ = f.Close()
	if _, _, err := ReadExportFindingsFromZIP(unknown); err == nil {
		t.Fatal("unknown schema must not be silently interpreted as v1")
	}
}

func TestStructuredFindingAdaptersPreserveProducerFields(t *testing.T) {
	syntax := ExportFindingsFromFinalSyntax(&FinalArtifactSyntaxReport{Findings: []FinalArtifactSyntaxFinding{{
		Code: "clausewitz.parse_failed", Path: "events/bad.txt", Line: 9,
		Snippet: "option = {", Severity: "attention", PlayerImpact: "事件可能无法加载",
		SuggestedEdit: "事件编辑器", Detail: "unexpected EOF",
	}}}, false)
	if len(syntax) != 1 || syntax[0].Message != "unexpected EOF" ||
		syntax[0].PlayerImpact != "事件可能无法加载" || syntax[0].EditRoute != "事件编辑器" ||
		syntax[0].SourceEvidence["snippet"] != "option = {" {
		t.Fatalf("syntax adapter lost producer fields: %+v", syntax)
	}

	refs := ExportFindingsFromReferenceIntegrity(ReferenceIntegrityReport{Findings: []ReferenceIntegrityFinding{{
		Code: "focus.prerequisite_missing", File: "common/national_focus/demo.txt", Line: 5,
		Target: "DEMO_missing", Summary: "前置国策不存在", Consequence: "国策树可能打不开",
		EditAt: "国策编辑器",
	}}}, true)
	if len(refs) != 1 || refs[0].Message != "前置国策不存在" ||
		refs[0].PlayerImpact != "国策树可能打不开" || refs[0].EditRoute != "国策编辑器" ||
		!refs[0].Acknowledged || refs[0].SourceEvidence["target"] != "DEMO_missing" {
		t.Fatalf("reference adapter lost producer fields: %+v", refs)
	}

	registry := ExportFindingsFromRegistryValidation(RegistryReport{Findings: []RegistryFinding{{
		Code: "modifier.unknown", Status: "需要检查", Path: "common/ideas/demo.txt", Line: 12,
		Token: "demo_bonus_typo", Text: "修正项未在当前游戏资料中登记；请检查名称",
	}}}, false)
	if len(registry) != 1 || registry[0].Message != "修正项未在当前游戏资料中登记；请检查名称" ||
		registry[0].PlayerImpact != "" || registry[0].EditRoute != "" ||
		registry[0].SourceEvidence["status"] != "需要检查" ||
		registry[0].SourceEvidence["token"] != "demo_bonus_typo" {
		t.Fatalf("registry adapter invented or lost producer fields: %+v", registry)
	}
}

func TestExportFindingsPathFallsBackToZIPRootForMultipleRoots(t *testing.T) {
	entries := []Entry{
		{Path: "Alpha/descriptor.mod", Bytes: []byte("name=Alpha\n")},
		{Path: "Beta/descriptor.mod", Bytes: []byte("name=Beta\n")},
	}
	entry, err := ExportFindingsEntry(entries, nil)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Path != ExportFindingsBasename {
		t.Fatalf("ambiguous roots must place the report at ZIP root, got %q", entry.Path)
	}
}

func TestMergeExportFindingsEntryReplacesOneCurrentReport(t *testing.T) {
	entries := []Entry{{Path: "Demo/descriptor.mod", Bytes: []byte("name=Demo\n")}}
	first := ExportFinding{Code: "first", Source: json.RawMessage(`{"producer":"native"}`), Message: "first finding"}
	report, err := ExportFindingsEntry(entries, []ExportFinding{first})
	if err != nil {
		t.Fatal(err)
	}
	entries = append(entries, report)
	second := ExportFinding{Code: "second", Source: json.RawMessage(`{"producer":"advanced"}`), Message: "second finding", Acknowledged: true}
	merged, err := MergeExportFindingsEntry(entries, []ExportFinding{second})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("merged entries=%d want descriptor plus one report", len(merged))
	}
	doc, err := DecodeExportFindingsDocument(merged[1].Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Findings) != 2 || doc.Findings[0].Code != "first" || doc.Findings[1].Code != "second" || !doc.Findings[1].Acknowledged {
		t.Fatalf("combined findings=%+v", doc.Findings)
	}
}
