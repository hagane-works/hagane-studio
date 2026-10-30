package exportengine

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanFinalArtifactSyntaxReportsEverySupportedBadWinner(t *testing.T) {
	report := ScanFinalArtifactSyntax([]Entry{
		{
			Path: "Demo/common/ideas/broken.txt", Origin: "raw_override",
			SourceEntity: "project_raw_files:1", Bytes: []byte("ideas = {\n bad = {\n"),
		},
		{
			Path: "Demo/events/pseudo.txt", Origin: "generated",
			Bytes: []byte("{ effect = { } }\n"),
		},
		{
			Path: "Demo/history/countries/GER.txt", Origin: "raw_override",
			Bytes: []byte("capital = 64\\nset_politics = { ruling_party = fascism }\n!\n"),
		},
		{
			Path: "Demo/localisation/english/bad_l_english.yml", Origin: "raw_override",
			Bytes: []byte("l_english:\n BAD KEY:0 \"value\"\n NEXT:0 \"never closes\n"),
		},
		{Path: "Demo/interface/not_scanned.gui", Bytes: []byte("containerWindowType = {")},
		{Path: "Demo/common/ideas/not_scanned.yml", Bytes: []byte("ideas = {")},
	}, english)

	if report.Scanned != 4 {
		t.Fatalf("scanned=%d findings=%#v", report.Scanned, report.Findings)
	}
	wants := map[string]bool{
		"clausewitz.unclosed_block":            false,
		"clausewitz.top_level_anonymous_block": false,
		"clausewitz.literal_backslash_n":       false,
		"clausewitz.unknown_token":             false,
		"localisation.missing_utf8_bom":        false,
		"localisation.invalid_key":             false,
		"localisation.unterminated_value":      false,
	}
	for _, finding := range report.Findings {
		if _, ok := wants[finding.Code]; ok {
			wants[finding.Code] = true
		}
		if finding.Path == "" || finding.Line < 1 || finding.Snippet == "" ||
			finding.Severity == "" || finding.PlayerImpact == "" || finding.SuggestedEdit == "" {
			t.Errorf("finding lost an actionable field: %#v", finding)
		}
		if strings.HasPrefix(finding.Path, "Demo/") {
			t.Errorf("path must be relative to the mod root: %#v", finding)
		}
	}
	for code, found := range wants {
		if !found {
			t.Errorf("missing %s; findings=%#v", code, report.Findings)
		}
	}
}

func TestScanFinalArtifactSyntaxKeepsCleanFilesQuiet(t *testing.T) {
	report := ScanFinalArtifactSyntax([]Entry{
		{Path: "Demo/common/ideas/good.txt", Bytes: []byte("ideas = { GER_good = { allowed = { always = yes } } }\n")},
		{Path: "Demo/events/good.txt", Bytes: []byte("add_namespace = good\ncountry_event = { id = good.1 }\n")},
		{Path: "Demo/history/states/1-Test.txt", Bytes: []byte("state = { id = 1 history = { owner = GER } }\n")},
		{Path: "Demo/localisation/english/good_l_english.yml", Bytes: []byte("\uFEFFl_english:\n GOOD_KEY:0 \"Good text\" # note\n")},

		{Path: "Demo/common/ideas/empty.txt", Origin: "raw_empty_shadow", Bytes: []byte{}},
	}, english)
	if report.Scanned != 5 || report.Count() != 0 {
		t.Fatalf("clean artifact report=%#v", report)
	}
}

func TestApplyFinalArtifactSyntaxIsAdvisoryAndPreservesBytes(t *testing.T) {
	original := []byte("ideas = {\n")
	artifact := NativeArtifact{Entries: []Entry{{
		Path: "Demo/common/ideas/bad.txt", Bytes: append([]byte(nil), original...),
		Origin: "raw_override", SourceEntity: "project_raw_files:7",
	}}}
	report := ApplyFinalArtifactSyntax(&artifact, english)
	if report.Count() == 0 || len(artifact.Warnings) == 0 {
		t.Fatalf("finding did not reach advisory surface: report=%#v artifact=%#v", report, artifact)
	}
	if !bytes.Equal(artifact.Entries[0].Bytes, original) {
		t.Fatalf("report-only scan rewrote shipped bytes: %q", artifact.Entries[0].Bytes)
	}
	if !containsString(artifact.Coverage, FinalArtifactSyntaxCoverage) {
		t.Fatalf("coverage=%v", artifact.Coverage)
	}
	joined := strings.Join(artifact.Warnings, "\n")
	for _, want := range []string{"common/ideas/bad.txt:1", "player impact:", "fix in:", "Raw files → common/ideas/bad.txt"} {
		if !strings.Contains(joined, want) {
			t.Errorf("warning lost %q: %s", want, joined)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
