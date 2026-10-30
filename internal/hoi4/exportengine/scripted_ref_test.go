package exportengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func loadPredicateFixture(t *testing.T) (script string, flag, skip []string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "scripted", "predicate_cases.txt"))
	if err != nil {
		t.Fatal(err)
	}
	script = string(raw)
	line := regexp.MustCompile(`^\s*([A-Za-z0-9_.]+)\s*=\s*[^{#]*#\s*(FLAG|SKIP)`)
	inner := regexp.MustCompile(`\{\s*([A-Za-z0-9_.]+)\s*=\s*\w+\s*\}\s*#\s*(FLAG|SKIP)`)
	for _, l := range strings.Split(script, "\n") {
		m := line.FindStringSubmatch(l)
		if m == nil {
			m = inner.FindStringSubmatch(l)
		}
		if m == nil {
			continue
		}
		if m[2] == "FLAG" {
			flag = append(flag, m[1])
		} else {
			skip = append(skip, m[1])
		}
	}
	if len(flag) < 5 || len(skip) < 15 {
		t.Fatalf("fixture parsed into flag=%v skip=%v — annotation regex drifted", flag, skip)
	}
	return script, flag, skip
}

func scriptedFixtureEntries(t *testing.T) []Entry {
	t.Helper()
	script, _, _ := loadPredicateFixture(t)
	defs, err := os.ReadFile(filepath.Join("testdata", "scripted", "defined_effects.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return []Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("name=\"Demo\"\n")},
		{Path: "Demo/common/national_focus/demo.txt", Bytes: []byte(script)},
		{Path: "Demo/common/scripted_effects/demo_effects.txt", Bytes: defs},
		{Path: "Demo/localisation/english/demo_l_english.yml", Bytes: []byte("l_english:\n my_custom_effect = yes\n")},
	}
}

func TestScriptedPredicateMatchesFixtureTable(t *testing.T) {
	_, flag, skip := loadPredicateFixture(t)
	refs := ScanScriptedRefs(scriptedFixtureEntries(t), "Demo")
	got := map[string]ScriptedRef{}
	for _, ref := range refs {
		got[ref.Token] = ref
	}
	for _, token := range flag {
		if _, ok := got[token]; !ok {
			t.Errorf("FLAG %q not reported; reported=%v", token, tokensOf(refs))
		}
	}
	for _, token := range skip {
		if _, ok := got[token]; ok {
			t.Errorf("SKIP %q was reported (false positive)", token)
		}
	}
	if len(got) != len(uniq(flag)) {
		t.Errorf("reported %d tokens, fixture expects %d: %v", len(got), len(uniq(flag)), tokensOf(refs))
	}

	ref := got["my_custom_effect"]
	if !strings.HasPrefix(ref.First, "common/national_focus/demo.txt:") {
		t.Fatalf("First=%q", ref.First)
	}
	if ref.Count != 1 {
		t.Fatalf("my_custom_effect Count=%d, want 1", ref.Count)
	}
	if len(ref.Files) != 1 || ref.Files[0] != "common/national_focus/demo.txt" {
		t.Fatalf("Files=%v", ref.Files)
	}

	for i := 1; i < len(refs); i++ {
		if refs[i-1].Token > refs[i].Token {
			t.Fatalf("not sorted: %v", tokensOf(refs))
		}
	}
}

func TestScriptedPredicateClauses(t *testing.T) {
	cases := []struct {
		key, scalar string
		want        bool
		why         string
	}{
		{"my_custom_effect", "yes", true, "custom call"},
		{"my_custom_effect", "no", true, "custom call, = no"},
		{"my_custom_effect", "YES", true, "scalar.lower() in (yes,no)"},
		{"my_custom_effect", "maybe", false, "not yes/no"},
		{"", "yes", false, "empty key"},
		{"add_ideas", "yes", false, "vanilla effect"},
		{"has_war", "yes", false, "vanilla trigger"},
		{"cancel_if_invalid", "yes", false, "structural field"},
		{"fire_only_once", "yes", false, "_SCRIPT_WRAP_KEYS"},
		{"is_border_war", "yes", false, "_SCRIPT_WRAP_KEYS supplement"},
		{"has_completed_focus", "yes", false, "CROSS_REFS key"},
		{"unlock_decision_tooltip", "yes", false, "CROSS_REFS key"},
		{"ger.1", "yes", false, "has ."},
		{"ROOT", "yes", false, "isupper"},
		{"GFX_1", "yes", false, "isupper (digits ignored)"},
		{"GER_custom", "yes", true, "not isupper as a whole"},
		{"instant_build", "yes", false, "Go narrowing: vanilla yes/no key (effect parameter)"},
		{"keep_completed", "no", false, "Go narrowing: vanilla yes/no key (effect parameter)"},
		{"designer", "yes", false, "Go narrowing: vanilla yes/no key (idea structure, exporter emits it)"},
		{"can_be_randomly_selected", "no", false, "Go narrowing: vanilla yes/no key (ideology structure, exporter emits it)"},
	}
	for _, c := range cases {
		if got := isScriptedCall(c.key, c.scalar, nil); got != c.want {
			t.Errorf("isScriptedCall(%q, %q)=%v want %v (%s)", c.key, c.scalar, got, c.want, c.why)
		}
	}
	if !isScriptedCall("defined_here", "yes", nil) || isScriptedCall("defined_here", "yes", map[string]bool{"defined_here": true}) {
		t.Error("artifact-defined names must resolve, undefined ones must not")
	}
}

func TestPyIsUpper(t *testing.T) {
	for s, want := range map[string]bool{
		"GER": true, "GFX_FOO_1": true, "R": true, "ROOT": true,
		"_": false, "123": false, "": false, "Ger": false, "GER_custom": false, "gfx": false,
	} {
		if got := pyIsUpper(s); got != want {
			t.Errorf("pyIsUpper(%q)=%v want %v", s, got, want)
		}
	}
}

func scriptedSnapshot(t *testing.T, extraRaw ...string) Snapshot {
	t.Helper()
	script, _, _ := loadPredicateFixture(t)
	defs, err := os.ReadFile(filepath.Join("testdata", "scripted", "defined_effects.txt"))
	if err != nil {
		t.Fatal(err)
	}
	rows := append([]string{
		rawRow(1, "common/national_focus/demo.txt", script),
		rawRow(2, "common/scripted_effects/demo_effects.txt", string(defs)),
	}, extraRaw...)
	raws := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		raws = append(raws, json.RawMessage(row))
	}
	return Snapshot{
		ProjectID: 46,
		Project:   json.RawMessage(`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`),
		Tables:    map[string][]json.RawMessage{"project_raw_files": raws},
	}
}

func TestScriptedSectionInCrossRefLog(t *testing.T) {
	artifact, err := BuildNativeShadow(scriptedSnapshot(t), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")

	if !strings.HasPrefix(text, "# References between workbenches (notices only)\n") {
		t.Fatalf("log header changed:\n%s", text)
	}
	if !strings.Contains(text, "\n# informational · scripted_effect_trigger\n") {
		t.Fatalf("scripted section header missing:\n%s", text)
	}
	for _, want := range []string{
		"raw file defines it. Nothing here blocks the export.",
		"informational · scripted_effect_trigger · GER_custom_trigger: 1 · first at common/national_focus/demo.txt:",
		"informational · scripted_effect_trigger · my_custom_effect: 1 · first at common/national_focus/demo.txt:",
		"informational · scripted_effect_trigger · deeper_custom_call: 1 · first at common/national_focus/demo.txt:",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	for _, never := range []string{"instant_build", "defined_here", "sibling_helper", "cancel_if_invalid", "add_ideas"} {
		if strings.Contains(text, "· "+never+":") {
			t.Errorf("false positive %q in log:\n%s", never, text)
		}
	}

	bodyRe := regexp.MustCompile(`^informational · scripted_effect_trigger · [A-Za-z0-9_]+: \d+ · first at \S+:\d+:\d+$`)
	seen := 0
	for _, line := range bodyLines(text) {
		if strings.HasPrefix(line, "informational · ") {
			seen++
			if !bodyRe.MatchString(line) {
				t.Errorf("body line off-format: %q", line)
			}
		}
	}
	if seen != 6 {
		t.Errorf("expected 6 scripted body lines, got %d:\n%s", seen, text)
	}
	if !strings.HasSuffix(text, "\n") {
		t.Fatal("log must end with a newline")
	}
}

func TestScriptedFindingsAloneStillWriteTheLog(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 46,
		Project:   json.RawMessage(`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`),
		Tables: map[string][]json.RawMessage{"project_raw_files": {json.RawMessage(rawRow(
			1, "common/decisions/x.txt", "cat = {\n\td = {\n\t\tcomplete_effect = { my_custom_effect = yes }\n\t}\n}\n",
		))}},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")
	if !strings.Contains(text, "informational · scripted_effect_trigger · my_custom_effect: 1 · first at common/decisions/x.txt:3:23") {
		t.Fatalf("log:\n%s", text)
	}
	if _, ok := findEntry(artifact, "Demo/_TOKEN_LINT.log"); ok {
		t.Fatal("token lint log written for a clean project")
	}
}

func TestScriptedCleanProjectWritesNothing(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 46,
		Project:   json.RawMessage(`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`),
		Tables: map[string][]json.RawMessage{"project_raw_files": {json.RawMessage(rawRow(
			1, "common/decisions/x.txt",
			"cat = {\n\td = {\n\t\tcomplete_effect = { add_political_power = 10 set_rule = { can_create_factions = yes } instant_build = yes }\n\t}\n}\n",
		))}},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findEntry(artifact, "Demo/_CROSS_REF_SKIPPED.log"); ok {
		t.Fatalf("log written for a project with only vanilla parameter keys:\n%s", logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log"))
	}
	lint, err := BuildLintReport(snapshot, artifact.Entries, english)
	if err != nil {
		t.Fatal(err)
	}
	if h := lint.ScriptedHeadline(); h != "" {
		t.Fatalf("headline for a clean project: %q", h)
	}
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "scripted effects or triggers") {
			t.Fatalf("warning for a clean project: %q", w)
		}
	}
}

func TestScriptedHeadlineWording(t *testing.T) {
	if got := scriptedHeadline(nil, english); got != "" {
		t.Fatalf("empty scan must give an empty headline, got %q", got)
	}
	one := scriptedHeadline([]ScriptedRef{{Token: "my_custom_effect", Count: 3}}, english)
	want := "Custom scripted effects or triggers the program does not know (1): my_custom_effect. " +
		"They are kept as written and work only if a base mod or raw file defines them — see _CROSS_REF_SKIPPED.log."
	if one != want {
		t.Fatalf("one-name headline:\n got %q\nwant %q", one, want)
	}
	var many []ScriptedRef
	for i := 0; i < 7; i++ {
		many = append(many, ScriptedRef{Token: fmt.Sprintf("fx_%d", i), Count: 1})
	}
	h := scriptedHeadline(many, english)
	for _, want := range []string{"does not know (7)", "fx_0, fx_1, fx_2, fx_3, fx_4 and 2 more", "kept as written", "work only if a base mod or raw file defines them"} {
		if !strings.Contains(h, want) {
			t.Errorf("headline missing %q: %q", want, h)
		}
	}
	if strings.Contains(h, "fx_5") {
		t.Errorf("headline must cap at 5 tokens: %q", h)
	}

	artifact, err := BuildNativeShadow(scriptedSnapshot(t), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	lint, err := BuildLintReport(scriptedSnapshot(t), artifact.Entries, english)
	if err != nil {
		t.Fatal(err)
	}
	if h := lint.ScriptedHeadline(); !strings.HasPrefix(h, "Custom scripted effects or triggers the program does not know (6): GER_custom_trigger, Mixed_Case_call, another_custom_check, deeper_custom_call, my_custom_effect and 1 more.") {
		t.Fatalf("headline from artifact: %q", h)
	}
}

func TestScriptedScanSkipsUnparseableFiles(t *testing.T) {
	refs := ScanScriptedRefs([]Entry{
		{Path: "Demo/descriptor.mod", Bytes: []byte("name=\"Demo\"\n")},
		{Path: "Demo/common/decisions/broken.txt", Bytes: []byte("this = { my_custom_effect = yes\n")},
	}, "Demo")
	if len(refs) != 0 {
		t.Fatalf("findings from an unparseable file: %v", tokensOf(refs))
	}
}

func TestSeamScriptedHeadlineReachesNativeWarnings(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"],"dependencies":[]}`),
		Tables: map[string][]json.RawMessage{
			"focus_trees": {json.RawMessage(`{"id":1,"tree_id":"ger_demo_tree","country_tag":"GER","tree_kind":"primary","source":"created"}`)},
			"focus_nodes": {json.RawMessage(`{"id":2,"focus_tree_id":1,"focus_id":"GER_demo_focus","name_en":"Demo","icon":"GFX_focus_ger_demo_focus","cost":10,"x":0,"y":0,"prerequisites":[],"mutually_exclusive":[],"completion_reward":"my_custom_effect = yes\nadd_political_power = 10"}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	script, ok := findEntry(artifact, "Demo/common/national_focus/ger_demo_tree.txt")
	if !ok || !strings.Contains(string(script.Bytes), "my_custom_effect = yes") {
		t.Fatalf("focus reward did not ship as expected: ok=%v\n%s", ok, script.Bytes)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")
	if !strings.Contains(text, "informational · scripted_effect_trigger · my_custom_effect: 1 · first at common/national_focus/ger_demo_tree.txt:") {
		t.Fatalf("log:\n%s", text)
	}

	want := "Custom scripted effects or triggers the program does not know (1): my_custom_effect. " +
		"They are kept as written and work only if a base mod or raw file defines them — see _CROSS_REF_SKIPPED.log."
	for _, w := range artifact.Warnings {
		if w == want {
			return
		}
	}
	t.Fatalf("SEAM NOT LANDED (native.go must append lint.ScriptedHeadline()): want %q in Warnings=%q", want, artifact.Warnings)
}

func tokensOf(refs []ScriptedRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Token)
	}
	return out
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
