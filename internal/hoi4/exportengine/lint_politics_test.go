package exportengine

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
)

func TestScanPoliticsWarningsEmpty(t *testing.T) {
	r := ScanPoliticsWarnings(worldout.Meta{}, english)
	if r.Count() != 0 || r.Headline() != "" || r.LogText() != "" || r.LogEntry("Demo").Path != "" {
		t.Fatalf("empty meta must be silent: %+v", r)
	}
	var nilReport *PoliticsReport
	if nilReport.Count() != 0 || nilReport.Headline() != "" {
		t.Fatal("nil report must be silent")
	}
}

func politicsFixture() worldout.Meta {
	return worldout.Meta{Politics: []exportmsg.Message{
		{Key: "politics.ruling_party_dropped", Args: exportmsg.Args{"subject": "GER", "value": "x"}},
		{Key: "politics.duplicate_history", Args: exportmsg.Args{"subject": "ITA", "file": "history/countries/ITA - zDemo.txt"}},
		{Key: "politics.date_block_skipped", Args: exportmsg.Args{"subject": "GER", "number": 2, "reason": exportmsg.Message{Key: "history_date.empty"}}},
		{Key: "politics.note.ideologies_full", Args: exportmsg.Args{"subs": "fascism: x", "groups": "y"}},
		{Key: "politics.bookmark_picture", Args: exportmsg.Args{"subject": "demo_start", "picture": "custom", "fallback": "GFX_select_date_1936"}},
	}}
}

func TestScanPoliticsWarningsHeadlineNamesCountriesAndCountsNotes(t *testing.T) {
	r := ScanPoliticsWarnings(politicsFixture(), english)
	if r.Count() != 5 || len(r.Warnings()) != 4 || len(r.Notes()) != 1 {
		t.Fatalf("count=%d warnings=%d notes=%d", r.Count(), len(r.Warnings()), len(r.Notes()))
	}
	h := r.Headline()
	want := "Politics export: warnings for GER, ITA, demo_start (4) — part of their history or politics was dropped or skipped " +
		"(ruling party, duplicate history rows, dated blocks or bookmark pictures); details in _POLITICS_EXPORT_WARNINGS.log. " +
		"Also notes about common/ideologies: 1."
	if h != want {
		t.Fatalf("headline\n got: %s\nwant: %s", h, want)
	}
	log := r.LogText()
	if !strings.HasPrefix(log, "# Politics and ideology export warnings\n") ||
		!strings.HasSuffix(log, "the export uses \"GFX_select_date_1936\" instead.\n") ||
		!strings.Contains(log, "\n\nGER: the history's ruling_party \"x\"") ||
		!strings.Contains(log, "\n[note] common/ideologies/00_ideologies.txt was written") {
		t.Fatalf("log shape:\n%s", log)
	}
	entry := r.LogEntry("Demo")
	if entry.Path != "Demo/"+PoliticsLogName || entry.SourceEntity != "politics_export_lint" || string(entry.Bytes) != log {
		t.Fatalf("entry: %+v", entry)
	}
}

func TestScanPoliticsWarningsFollowsTheInterfaceLanguage(t *testing.T) {
	meta := politicsFixture()
	r := ScanPoliticsWarnings(meta, chinese)
	log := r.LogText()
	header := strings.SplitN(chinese.Text("politics.log.header", nil), "\n", 2)[0]
	if !strings.HasPrefix(log, "# "+header+"\n") || !strings.Contains(log, chinese.Render(meta.Politics[0])) {
		t.Fatalf("log not in Chinese:\n%s", log)
	}
	if english.Text("politics.log.header", nil) == chinese.Text("politics.log.header", nil) ||
		!strings.Contains(r.Headline(), "GER, ITA, demo_start") || !strings.Contains(r.Headline(), PoliticsLogName) {
		t.Fatalf("headline: %s", r.Headline())
	}
}

func TestScanPoliticsWarningsNotesOnlyHeadline(t *testing.T) {
	r := ScanPoliticsWarnings(worldout.Meta{Politics: []exportmsg.Message{
		{Key: "politics.note.ideologies_raw", Args: exportmsg.Args{"subs": "fascism: x", "groups": "y"}},
	}}, english)
	if got := r.Headline(); got != "Politics export: notes about common/ideologies and replace_path (1); details in _POLITICS_EXPORT_WARNINGS.log." {
		t.Fatalf("headline: %q", got)
	}
}

func TestScanPoliticsWarningsCapsTheSubjectList(t *testing.T) {
	var lines []exportmsg.Message
	for _, tag := range []string{"AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG"} {
		lines = append(lines, exportmsg.Message{Key: "politics.country_ideology_dropped", Args: exportmsg.Args{"subject": tag, "value": "z"}})
	}
	h := ScanPoliticsWarnings(worldout.Meta{Politics: lines}, english).Headline()
	if !strings.Contains(h, "warnings for AAA, BBB, CCC, DDD, EEE and 2 more (7) —") {
		t.Fatalf("headline: %q", h)
	}
}
