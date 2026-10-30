package worldout

import (
	"os"
	"strings"
	"testing"
)

func dateBlock(date, script string) *OMap {
	m := NewOMap()
	m.Set("date", date)
	m.Set("script", script)
	return m
}

func oldRenderHistoryDateBlocks(blocks []Value) string {
	var rendered []string
	for _, b := range blocks {
		bm, ok := b.(*OMap)
		if !ok {
			panic("date block must be a mapping")
		}
		date, err := validateHistoryDate(eStr(bm, "date"))
		if err != nil {
			panic("invalid history date: " + eStr(bm, "date"))
		}
		script := eStr(bm, "script")
		rendered = append(rendered, date+" = {"+script+"}")
	}
	return strings.Join(rendered, "\n\n")
}

func TestNegativeControlInvalidHistoryDate(t *testing.T) {
	blocks := []Value{
		dateBlock("1936.1.1", " capital = 219 "),
		dateBlock("", " declare_war_on = { target = GER } "),
		dateBlock("1939.9.1", " add_ideas = SOV_war_economy "),
	}

	var panicked any
	func() {
		defer func() { panicked = recover() }()
		oldRenderHistoryDateBlocks(blocks)
	}()
	if panicked == nil {
		t.Fatal("negative control is void: the pre-change function no longer panics")
	}
	t.Logf("BEFORE panic: %v", panicked)

	out, notices, err := renderHistoryDateBlocksTolerant(blocks, "SOV")
	if err != nil {
		t.Fatalf("an invalid date must degrade, not fail closed: %v", err)
	}
	if len(notices) != 1 {
		t.Fatalf("want 1 notice, got %v", notices)
	}
	notice := joinWarnings(notices)
	t.Logf("AFTER  notice: %s", notice)
	t.Logf("AFTER  script:\n%s", out)

	for _, want := range []string{
		"SOV history file",
		"dated block #2",
		"the date is empty",
		"history workbench",
		"never happen",
	} {
		if !strings.Contains(notice, want) {
			t.Errorf("notice missing %q: %s", want, notice)
		}
	}

	if !strings.Contains(out, "1936.1.1 = {") || !strings.Contains(out, "1939.9.1 = {") {
		t.Errorf("healthy blocks were lost:\n%s", out)
	}
	if strings.Contains(out, "declare_war_on") {
		t.Errorf("skipped block leaked into the script:\n%s", out)
	}
	if strings.HasPrefix(out, " = {") || strings.Contains(out, "\n = {") {
		t.Errorf("emitted a keyless block:\n%s", out)
	}
}

func TestOutOfRangeHistoryDateDegradesWithItsOwnReason(t *testing.T) {
	_, notices, err := renderHistoryDateBlocksTolerant(
		[]Value{dateBlock("1936.13.1", " capital = 219 ")}, "GER")
	if err != nil {
		t.Fatalf("unexpected hard failure: %v", err)
	}
	if len(notices) != 1 ||
		!strings.Contains(joinWarnings(notices), "month 1-12") {
		t.Fatalf("want the range-specific reason, got: %v", notices)
	}
	t.Logf("notice: %s", joinWarnings(notices))
}

func TestNonMappingDateBlockFailsClosed(t *testing.T) {
	out, notices, err := renderHistoryDateBlocksTolerant(
		[]Value{dateBlock("1936.1.1", " capital = 219 "), "1939.9.1"}, "SOV")
	if err == nil {
		t.Fatal("a shape fault must fail closed, not degrade")
	}
	t.Logf("hard error: %v", err)
	if out != "" || notices != nil {
		t.Errorf("fail-closed must not hand back partial output: %q %v", out, notices)
	}
	for _, want := range []string{"SOV history file", "dated block #2", "report it"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q: %v", want, err)
		}
	}
	if !strings.Contains(err.Error(), "not something in your project") {
		t.Errorf("a fault the author cannot fix must say so: %v", err)
	}
}

func TestHistoryDateBlocksHealthyBytesUnchanged(t *testing.T) {
	blocks := []Value{
		dateBlock("1936.1.1", " capital = 219 "),
		dateBlock("1939.9.1.12", " add_ideas = SOV_war_economy "),
	}
	out, notices, err := renderHistoryDateBlocksTolerant(blocks, "SOV")
	if err != nil || len(notices) != 0 {
		t.Fatalf("healthy input produced %v / %v", notices, err)
	}
	want := "1936.1.1 = { capital = 219 }\n\n1939.9.1.12 = { add_ideas = SOV_war_economy }"
	if out != want {
		t.Errorf("byte drift\n want: %q\n got:  %q", want, out)
	}
	if legacy := renderHistoryDateBlocks(blocks); legacy != out {
		t.Errorf("shim drifted from the tolerant form\n %q\n %q", legacy, out)
	}
}

func TestCentralSeamConsumesHistoryDateNotices(t *testing.T) {
	src, err := os.ReadFile("emit_world.go")
	if err != nil {
		t.Fatalf("cannot read the seam file: %v", err)
	}
	text := string(src)
	if !strings.Contains(text, "renderHistoryDateBlocksTolerant") {
		t.Error("SEAM NOT LANDED: emit_world.go:748 still calls the notice-discarding " +
			"renderHistoryDateBlocks, so a skipped dated block is never reported.")
	}
	if !strings.Contains(text, "c.politics = append(c.politics, dateNotices...)") {
		t.Error("SEAM NOT LANDED: emit_world.go does not forward the notices into " +
			"c.politics (→ Meta.Politics), so they never reach the author.")
	}
}
