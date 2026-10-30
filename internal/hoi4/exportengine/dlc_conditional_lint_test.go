package exportengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dlcFixtureEntries(t *testing.T) []Entry {
	t.Helper()
	root := filepath.Join("testdata", "dlc")
	entries := []Entry{
		{Path: "Demo.mod", Bytes: []byte("name=\"Demo\"")},
		{Path: "Demo/descriptor.mod", Bytes: []byte("name=\"Demo\"")},
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		entries = append(entries, Entry{Path: "Demo/" + filepath.ToSlash(rel), Bytes: data, Origin: "raw"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func dlcFindHit(hits []DLCHit, path string, dlc string, line int) (DLCHit, bool) {
	for _, h := range hits {
		if h.Path == path && h.DLC == dlc && (line == 0 || h.Line == line) {
			return h, true
		}
	}
	return DLCHit{}, false
}

func TestDLCScanParsesNestedIfLimitNotAndMultiDLC(t *testing.T) {
	report := ScanDLCConditionals(dlcFixtureEntries(t), english)
	if report.Total() != 8 {
		t.Fatalf("total = %d, want 8; hits=%+v", report.Total(), report.Hits)
	}
	if len(report.ParseFailed) != 0 {
		t.Fatalf("fixture must parse: %v", report.ParseFailed)
	}
	states := "history/states/1-X.txt"

	h, ok := dlcFindHit(report.Hits, states, "La Resistance", 9)
	if !ok {
		t.Fatalf("La Resistance hit missing: %+v", report.Hits)
	}
	if h.Negated || h.Context != "if" || strings.Join(h.Guarded, ",") != "transfer_state_to,add_core_of" || !h.Known {
		t.Fatalf("unexpected hit: %+v", h)
	}
	if h.Dir != "history/states" {
		t.Fatalf("dir = %q", h.Dir)
	}

	h, ok = dlcFindHit(report.Hits, states, "By Blood Alone", 0)
	if !ok || !h.Negated || h.Context != "if" || strings.Join(h.Guarded, ",") != "set_state_flag" {
		t.Fatalf("NOT hit wrong: %+v ok=%v", h, ok)
	}

	for _, name := range []string{"Man the Guns", "No Step Back"} {
		h, ok = dlcFindHit(report.Hits, states, name, 0)
		if !ok || h.Negated || h.Context != "if" || strings.Join(h.Guarded, ",") != "add_building_construction" {
			t.Fatalf("AND hit %s wrong: %+v ok=%v", name, h, ok)
		}
	}

	h, ok = dlcFindHit(report.Hits, states, "la resistance", 0)
	if !ok || h.Known || h.Suggest != "La Resistance" {
		t.Fatalf("typo hit wrong: %+v ok=%v", h, ok)
	}

	focus := "common/national_focus/han.txt"
	h, ok = dlcFindHit(report.Hits, focus, "Man the Guns", 5)
	if !ok || h.Context != "available of focus id=HAN_navy" || len(h.Guarded) != 0 {
		t.Fatalf("available hit wrong: %+v ok=%v", h, ok)
	}

	h, ok = dlcFindHit(report.Hits, focus, "Man the Guns", 12)
	if !ok || h.Context != "else_if" || strings.Join(h.Guarded, ",") != "navy_experience" {
		t.Fatalf("else_if hit wrong: %+v ok=%v", h, ok)
	}

	h, ok = dlcFindHit(report.Hits, focus, "Wojtek Expansion Pack", 0)
	if !ok || h.Known || h.Suggest != "" {
		t.Fatalf("unknown hit wrong: %+v ok=%v", h, ok)
	}

	if report.ByDLC["Man the Guns"] != 3 || report.DLCCount() != 6 {
		t.Fatalf("aggregation wrong: %+v", report.ByDLC)
	}
	if report.ByDir["history/states"] != 5 || report.ByDir["common/national_focus"] != 3 {
		t.Fatalf("dir aggregation wrong: %+v", report.ByDir)
	}

	if report.Hits[0].Dir != "history/states" || report.Hits[len(report.Hits)-1].Dir != "common/national_focus" {
		t.Fatalf("history/states must sort first: %+v", report.Hits)
	}
	if got := report.UnknownNames(); strings.Join(got, "|") != "Wojtek Expansion Pack|la resistance" {
		t.Fatalf("unknown names = %v", got)
	}
}

func TestDLCHeadlineAndLog(t *testing.T) {
	report := ScanDLCConditionals(dlcFixtureEntries(t), english)
	head := report.Headline()
	want := "Blocks that need a DLC (has_dlc): 8; DLCs involved: 6 (Man the Guns, By Blood Alone, La Resistance…). " +
		"They run only for players who own that DLC — check _DLC_CONDITIONAL_LINT.log (history/states: 5). " +
		"Unknown DLC names (2): Wojtek Expansion Pack, la resistance."
	if head != want {
		t.Fatalf("headline:\n got %q\nwant %q", head, want)
	}
	entry := DLCConditionalLogEntry("Demo", report)
	if entry.Path != "Demo/"+DLCConditionalLogName || entry.SourceEntity != "dlc_conditional_lint" {
		t.Fatalf("log entry: %+v", entry)
	}
	log := string(entry.Bytes)
	for _, must := range []string{
		"# " + head,
		"## La Resistance  (1)",
		`history/states/1-X.txt:9  has_dlc "La Resistance"  if  -> transfer_state_to, add_core_of`,
		`NOT has_dlc "By Blood Alone"  if  -> set_state_flag`,
		"## la resistance  (1)  [unknown]",
		`la resistance  [unknown] did you mean "La Resistance"?`,
		`common/national_focus/han.txt:5  has_dlc "Man the Guns"  available of focus id=HAN_navy`,
		"# By folder:",
		"history/states",
	} {
		if !strings.Contains(log, must) {
			t.Fatalf("log missing %q:\n%s", must, log)
		}
	}
	if DLCConditionalLogEntry("", report).Path != "" {
		t.Fatal("no modDir → no entry")
	}
}

func TestDLCScanZeroHitsWritesNothing(t *testing.T) {
	entries := []Entry{
		{Path: "Demo.mod", Bytes: []byte("x")},
		{Path: "Demo/descriptor.mod", Bytes: []byte("x")},
		{Path: "Demo/history/states/2-Y.txt", Bytes: []byte("state = { id = 2 history = { owner = HAN } }")},
		{Path: "Demo/localisation/english/x_l_english.yml", Bytes: []byte("l_english:\n k: \"has_dlc = \\\"La Resistance\\\"\"\n")},
		{Path: "Demo/_NOTES.log", Bytes: []byte(`has_dlc = "La Resistance"`)},
	}
	report := ScanDLCConditionals(entries, english)
	if report.Total() != 0 || report.Headline() != "" {
		t.Fatalf("expected nothing: total=%d head=%q", report.Total(), report.Headline())
	}
	if DLCConditionalLogEntry("Demo", report).Path != "" {
		t.Fatal("zero hits must not write a log")
	}
	var nilReport *DLCReport
	if nilReport.Headline() != "" || nilReport.Total() != 0 || DLCConditionalLogEntry("Demo", nilReport).Path != "" {
		t.Fatal("nil report must be inert")
	}
}

func TestDLCScanSkipsUnparseableFileWithoutFailing(t *testing.T) {
	entries := []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("x")},
		{Path: "Demo/events/bad.txt", Bytes: []byte(`country_event = { id = x.1 trigger = { has_dlc = "La Resistance" } option = { name = a } } } }`)},
		{Path: "Demo/events/good.txt", Bytes: []byte(`country_event = { id = x.2 trigger = { has_dlc = "La Resistance" } }`)},
	}
	report := ScanDLCConditionals(entries, english)

	if report.ByDLC["La Resistance"] < 1 {
		t.Fatalf("good file hit lost: %+v", report)
	}
	h, ok := dlcFindHit(report.Hits, "events/good.txt", "La Resistance", 0)
	if !ok || h.Context != "trigger of country_event id=x.2" || h.Dir != "events" {
		t.Fatalf("event trigger context wrong: %+v ok=%v", h, ok)
	}
}

func TestDLCScanGuardedKeysCapped(t *testing.T) {
	var body strings.Builder
	body.WriteString(`if = { limit = { has_dlc = "La Resistance" }`)
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&body, " set_variable = { v%d = 1 } effect_%d = yes", i, i)
	}
	body.WriteString(" }")
	entries := []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("x")},
		{Path: "Demo/common/on_actions/x.txt", Bytes: []byte(body.String())},
	}
	report := ScanDLCConditionals(entries, english)
	if report.Total() != 1 {
		t.Fatalf("total = %d", report.Total())
	}
	g := report.Hits[0].Guarded
	if len(g) != dlcMaxGuardedKeys+1 || g[0] != "set_variable" || !strings.HasPrefix(g[len(g)-1], "+") {
		t.Fatalf("guarded = %v", g)
	}
}

func TestDLCScanPerformanceThousandHits(t *testing.T) {
	var body strings.Builder
	body.WriteString("state = {\n\tid = 9\n\thistory = {\n")
	names := []string{"La Resistance", "Man the Guns", "By Blood Alone", "No Compromise, No Surrender", "Gotterdammerung"}
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&body, "\t\tif = {\n\t\t\tlimit = { has_dlc = \"%s\" }\n\t\t\ttransfer_state_to = COG\n\t\t\tadd_core_of = COG\n\t\t}\n", names[i%len(names)])
	}
	body.WriteString("\t}\n}\n")
	entries := []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("x")},
		{Path: "Demo/history/states/9-Big.txt", Bytes: []byte(body.String())},
	}
	start := time.Now()
	report := ScanDLCConditionals(entries, english)
	log := report.LogText()
	elapsed := time.Since(start)
	if report.Total() != 1000 || report.DLCCount() != 5 {
		t.Fatalf("total=%d dlcs=%d", report.Total(), report.DLCCount())
	}
	if elapsed > time.Second {
		t.Fatalf("1000 hits took %v (> 1s)", elapsed)
	}
	if strings.Count(log, "\n") < 1000 {
		t.Fatalf("log too short: %d lines", strings.Count(log, "\n"))
	}
	if !strings.HasPrefix(report.Headline(), "Blocks that need a DLC (has_dlc): 1000; DLCs involved: 5 (") {
		t.Fatalf("headline = %q", report.Headline())
	}
}
