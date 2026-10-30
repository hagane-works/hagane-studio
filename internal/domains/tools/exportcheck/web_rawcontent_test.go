package exportcheck

import (
	"encoding/json"
	"haganeworks/internal/hoi4/clausewitz"
	"strings"
	"testing"
	"time"
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

func TestRawContentLargeFileIsLinear(t *testing.T) {
	body := strings.Repeat("country_event = { id = x.1 option = { name = a } }\n", 6000)
	start := time.Now()
	report := AnalyzeRawContent(body, "events/big.txt", "")
	if !report.IsValid() || time.Since(start) > 3*time.Second {
		t.Fatalf("valid=%v elapsed=%s", report.IsValid(), time.Since(start))
	}
}

func TestRawScriptFindingsGateEventOptionAndFriends(t *testing.T) {
	ev := Event{ID: 7, EventID: "sanguo_sys.39",
		Options: []any{map[string]any{"name": "a", "effects": sanguoLine192}}}
	okEv := Event{ID: 8, EventID: "ok.1", Immediate: "set_country_flag = x # }",
		Options: []any{map[string]any{"effects": "custom_effect_tooltip = \"{\""}}}
	dec := Decision{ID: 1, DecisionID: "d1", RemoveEffect: "if = { limit = { always = yes } # }"}
	idea := Idea{ID: 2, IdeaID: "i1", OnRemove: "}", OnAdd: "custom_effect_tooltip = \"a\\nb\""}
	raws := []RawScriptRawFile{
		{ID: 3, DestPath: "common/on_actions/x.txt", Source: "created", Content: "on_actions = { on_startup = { effect = { } }"},
		{ID: 4, DestPath: "localisation/english/x_l_english.yml", Source: "created", Content: "l_english:\n a:0 \"{\""},
		{ID: 5, DestPath: "events/imported.txt", Source: "imported", Content: "{"},
	}
	findings := RawScriptFindings([]Event{ev, okEv}, []Decision{dec}, []Idea{idea}, raws)
	got := map[string]string{}
	for _, f := range findings {
		ent := f["source"].(map[string]any)["entity"].(map[string]any)
		got[ent["type"].(string)+"/"+ent["id"].(string)+"/"+ent["field"].(string)] = f["code"].(string) + ":" + f["severity"].(string)
	}
	want := map[string]string{
		"event/sanguo_sys.39/options[0].effects":   "clausewitz.unclosed_block:blocker",
		"decision/d1/remove_effect":                "clausewitz.unclosed_block:blocker",
		"idea/i1/on_remove":                        "clausewitz.stray_closing_brace:blocker",
		"idea/i1/on_add":                           RawContentLiteralNewline + ":warning",
		"raw_file/common/on_actions/x.txt/content": "clausewitz.unclosed_block:blocker",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: want %s got %q (all=%v)", k, v, got[k], got)
		}
	}
	for k := range got {
		if strings.HasPrefix(k, "event/ok.1") || strings.Contains(k, "l_english") || strings.Contains(k, "imported") {
			t.Errorf("false positive %s", k)
		}
	}

	issue := P0FindingToExportIssue(findings[0])
	if issue["level"] != "error" || issue["severity"] != "blocker" {
		t.Fatalf("%+v", issue)
	}
	ref, _ := issue["entity_ref"].(map[string]any)
	if ref["type"] != "event" {
		t.Fatalf("entity_ref %+v", issue["entity_ref"])
	}
}
