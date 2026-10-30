package rawfiles

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"haganeworks/internal/hoi4/clausewitz"
)

const sanguoLine192 = "\t\tif = { limit = { country_exists = WEI WEI = { NOT = { has_country_flag = " +
	"sanguo_declared_emperor } } } WEI = { set_country_flag = sanguo_declared_emperor " +
	"set_country_leader_ideology = jianhao_chengdi set_cosmetic_tag = SANGUO_WEI_CHAO " +
	"custom_effect_tooltip = sanguo_tt_mandate_declared_emperor " +
	"country_event = { id = sanguo_sys.40 days = 0 }   " +
	"# P2-05: 按需重算改走净件(不推窗不募人) } }"

func codes(fs []clausewitz.AnalysisFinding) []string {
	out := []string{}
	for _, f := range fs {
		out = append(out, f.Code)
	}
	return out
}

func TestRawContentSanguoIncidentLineIsBlocker(t *testing.T) {
	report := AnalyzeRawContent(sanguoLine192, "events/HAN_sanguo_shiguan.txt", "")
	brace := report.BraceBlockers()
	if len(brace) == 0 || report.IsValid() {
		t.Fatalf("expected blockers, got %+v", report)
	}
	if report.NetDelta != 2 {
		t.Fatalf("net delta want 2 got %d", report.NetDelta)
	}
	if brace[0].Code != "clausewitz.unclosed_block" {
		t.Fatalf("code %s", brace[0].Code)
	}
	suspects, ok := brace[0].Evidence["commented_brace_lines"].([]map[string]any)
	if !ok || len(suspects) == 0 || suspects[0]["line"] != 1 {
		t.Fatalf("suspects %+v", brace[0].Evidence["commented_brace_lines"])
	}
	if !strings.Contains(suspects[0]["source_excerpt"].(string), "# P2-05") {
		t.Fatalf("excerpt %v", suspects[0]["source_excerpt"])
	}

	raw, _ := json.Marshal(brace[0])
	for _, key := range []string{`"code"`, `"severity"`, `"source"`, `"evidence"`, `"repair_route"`, `"start_line"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("json missing %s: %s", key, raw)
		}
	}
}

func TestRawContentWholeFileNamesIncidentLine(t *testing.T) {
	body := "country_event = {\n\tid = sanguo_sys.39\n\toption = {\n" + sanguoLine192 + "\n\t}\n}\n"
	_, rerr := GateRawContent("events/HAN_sanguo_shiguan.txt", body, false)
	if rerr == nil {
		t.Fatal("expected RawContentError")
	}
	if rerr.Code != RawContentBraceCode || rerr.NetDelta != 2 || rerr.RepairRoute != RawContentBraceRepairRoute {
		t.Fatalf("%+v", rerr)
	}
	if !strings.Contains(rerr.Message, "comment at line 4") {
		t.Fatalf("message should name the swallowed-brace line: %s", rerr.Message)
	}
	terr := rerr.AsTombstone()
	if terr.Code != RawContentBraceCode || len(terr.Findings) == 0 || terr.RepairRoute == "" {
		t.Fatalf("tombstone %+v", terr)
	}
}

func TestRawContentCommentAndQuoteBracesDoNotCount(t *testing.T) {
	for _, text := range []string{
		"add_political_power = 10 # { { { not braces } \nset_country_flag = x\n",
		"custom_effect_tooltip = \"a { b } c {\"\nset_country_flag = \"{x}\"\n",
		"name = \"he said \\\"{\\\" ok\"\nx = { y = 1 }\n",
		"country_event = {\n\tid = x.1\n\toption = { name = x.1.a }\n}\n",
	} {
		fs, net := ScanBraceStructure(text, nil, nil)
		if len(fs) != 0 || net != 0 {
			t.Fatalf("false positive on %q: %v net=%d", text, codes(fs), net)
		}
	}

	fs, net := ScanBraceStructure("if = { limit = { always = yes } # }\n", nil, nil)
	if net != 1 || len(fs) != 1 || fs[0].Code != "clausewitz.unclosed_block" {
		t.Fatalf("%v net=%d", codes(fs), net)
	}
}

func TestRawContentStrayAndUnterminatedQuote(t *testing.T) {
	fs, net := ScanBraceStructure("a = { b = 1 }\n}\nc = 2\n", nil, nil)
	if net != -1 || len(fs) != 1 || fs[0].Code != "clausewitz.stray_closing_brace" || fs[0].Evidence["line"] != 2 {
		t.Fatalf("%+v net=%d", fs, net)
	}
	fs, _ = ScanBraceStructure("a = { b = \"oops }\n", nil, nil)
	if !strings.Contains(strings.Join(codes(fs), ","), "clausewitz.unterminated_quote") {
		t.Fatalf("%v", codes(fs))
	}
}

func TestRawContentLiteralNewlineTxtWarnsYmlDoesNot(t *testing.T) {
	ws, rerr := GateRawContent("events/x.txt", "custom_effect_tooltip = \"line one\\nline two\"\n", false)
	if rerr != nil || len(ws) != 1 || ws[0].Code != RawContentLiteralNewline || ws[0].Severity != "warning" {
		t.Fatalf("ws=%+v err=%+v", ws, rerr)
	}
	ws, rerr = GateRawContent("localisation/english/x_l_english.yml", "l_english:\n k:0 \"a\\nb {\"\n", false)
	if rerr != nil || len(ws) != 0 {
		t.Fatalf("yml must not be gated: ws=%+v err=%+v", ws, rerr)
	}

	ws, _ = GateRawContent("events/x.txt", "# use \\n here\nx = 1\n", false)
	if len(ws) != 0 {
		t.Fatalf("%+v", ws)
	}

	if ws, rerr := GateRawContent("gfx/x.dds", "{{{", true); rerr != nil || len(ws) != 0 {
		t.Fatalf("binary gated: %+v %+v", ws, rerr)
	}
}

func TestValidateRawFileIntentRejectsUnbalancedVerbatim(t *testing.T) {
	_, terr := ValidateRawFileIntent("verbatim_override", "common/on_actions/x.txt",
		"on_actions = { on_startup = { # } }", false, false, "1.19.2", "", "")
	if terr == nil || terr.Code != RawContentBraceCode {
		t.Fatalf("want %s got %+v", RawContentBraceCode, terr)
	}
	if len(terr.Findings) == 0 || terr.RepairRoute != RawContentBraceRepairRoute {
		t.Fatalf("tombstone payload %+v", terr)
	}

	got, terr := ValidateRawFileIntent("verbatim_override", "common/on_actions/x.txt",
		"on_actions = { on_startup = { effect = { custom_effect_tooltip = \"a\\nb\" } } } # }", false, false, "1.19.2", "", "")
	if terr != nil || got.DestPath != "common/on_actions/x.txt" {
		t.Fatalf("%+v %+v", got, terr)
	}

	if _, terr := ValidateRawFileIntent("verbatim_override", "localisation/english/x_l_english.yml",
		"l_english:\n k:0 \"{\"", false, false, "1.19.2", "", ""); terr != nil {
		t.Fatalf("yml gated: %+v", terr)
	}
}

func TestRawFileWarningsShape(t *testing.T) {
	rf := withContentWarnings(RawFile{DestPath: "events/x.txt", Content: "x = \"a\\nb\""})
	if len(rf.Warnings) != 1 || rf.Warnings[0].Code != RawContentLiteralNewline {
		t.Fatalf("%+v", rf.Warnings)
	}
	rf = withContentWarnings(RawFile{DestPath: "events/x.txt", Content: "x = 1"})
	raw, _ := json.Marshal(rf)
	if !strings.Contains(string(raw), `"warnings":[]`) {
		t.Fatalf("warnings must serialise as [] not null: %s", raw)
	}
}

func TestRawContentLargeFileIsLinear(t *testing.T) {
	body := strings.Repeat("country_event = { id = x.1 option = { name = a } }\n", 6000)
	start := time.Now()
	report := AnalyzeRawContent(body, "events/big.txt", "")
	if !report.IsValid() || time.Since(start) > 3*time.Second {
		t.Fatalf("valid=%v elapsed=%s", report.IsValid(), time.Since(start))
	}
}
