package exportengine

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/exportmsg"
)

func TestScanLiteralNewlinesSeesGeneratedAndRawFinishedEntries(t *testing.T) {
	entries := []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte(`name="Demo"`)},
		{Path: "Demo/events/SOV_PRC.txt", Bytes: []byte("country_event = {\n immediate = { a = yes\\nb = yes }\n}\n"), Origin: "generated"},
		{Path: "Demo/common/scripted_effects/raw.txt", Bytes: []byte("raw = { x = 1\\ny = 2 }\n"), Origin: "raw"},
		{Path: "Demo/localisation/english/x_l_english.yml", Bytes: []byte("l_english:\n x:1 \"a\\nb\"\n")},
		{Path: "Demo/history/countries/comment.txt", Bytes: []byte("x = 1 # example \\n is documentation\n")},
	}
	report := ScanLiteralNewlines(entries, english)
	if report.Count() != 2 {
		t.Fatalf("want generated+raw hits only, got %#v", report.Hits)
	}
	if report.Hits[0].Path != "common/scripted_effects/raw.txt" || report.Hits[1].Path != "events/SOV_PRC.txt" {
		t.Fatalf("unexpected stable path ordering: %#v", report.Hits)
	}
	if !strings.Contains(report.Headline(), "what follows will not take effect") {
		t.Fatalf("headline is not creator-facing: %q", report.Headline())
	}

	zh := ScanLiteralNewlines(entries, chinese)
	if want := chinese.Text("newline.headline", exportmsg.Args{"count": 2, "log": LiteralNewlineLintLogName}); zh.Headline() != want || want == report.Headline() {
		t.Fatalf("Chinese headline = %q, want %q", zh.Headline(), want)
	}
}

func TestLiteralNewlineLogNamesExactRepairLocations(t *testing.T) {
	report := LiteralNewlineReport{Hits: []LiteralNewlineHit{{
		Path: "events/SOV_PRC.txt", Line: 40, Column: 18, Excerpt: `a = yes\nb = yes`,
	}}}
	entry := LiteralNewlineLogEntry("Demo", report)
	if entry.Path != "Demo/"+LiteralNewlineLintLogName {
		t.Fatalf("unexpected log path: %q", entry.Path)
	}
	body := string(entry.Bytes)
	for _, want := range []string{"events/SOV_PRC.txt:40:18", "what follows will not take effect", "a = yes\\nb = yes"} {
		if !strings.Contains(body, want) {
			t.Fatalf("log missing %q:\n%s", want, body)
		}
	}
}

func TestLiteralNewlineScannerCountsMultipleLocations(t *testing.T) {
	hits := literalNewlineHits("events/x.txt", "a = yes\\nb = yes\\nc = yes")
	if len(hits) != 2 || hits[0].Line != 1 || hits[0].Column != 8 || hits[1].Line != 1 {
		t.Fatalf("unexpected positions: %#v", hits)
	}
}
