package exportengine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

const laneLScript = `lane_l_category = {
	lane_l_decision = {
		available = {
			has_completed_focus = LANE_L_present
			has_completed_focus = LANE_L_absent
		}
		complete_effect = {
			unlock_decision_tooltip = LANE_L_known_decision
			unlock_decision_tooltip = LANE_L_ghost_decision
			add_tech_bonus = {
				bonus = 0.5
				uses = 1
				category = industry
			}
			add_tech_bonus = {
				bonus = 0.5
				uses = 1
				category = lane_l_not_a_category
			}
			targeted_modifier = {
				attack_bonus = 0.1
			}
		}
	}
}
`

func rawRow(id int, destPath, content string) string {
	row := map[string]any{
		"id":        id,
		"dest_path": destPath,
		"content":   content,
		"source":    "created",
		"raw_kind":  "verbatim_override",
		"is_binary": false,
	}
	encoded, err := json.Marshal(row)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func lintSnapshot(projectJSON string, extraRaw []string, tables map[string][]string) Snapshot {
	rows := append([]string{rawRow(1, "common/decisions/lane_l.txt", laneLScript)}, extraRaw...)
	built := map[string][]json.RawMessage{}
	raws := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		raws = append(raws, json.RawMessage(row))
	}
	built["project_raw_files"] = raws
	for table, values := range tables {
		out := make([]json.RawMessage, 0, len(values))
		for _, value := range values {
			out = append(out, json.RawMessage(value))
		}
		built[table] = out
	}
	return Snapshot{
		ProjectID: 46,
		Project:   json.RawMessage(projectJSON),
		Tables:    built,
	}
}

func defaultLintSnapshot() Snapshot {
	return lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`,
		nil,
		map[string][]string{
			"focus_nodes": {`{"id":1,"focus_id":"LANE_L_present","focus_tree_id":1}`},
			"decisions":   {`{"id":1,"decision_id":"LANE_L_known_decision"}`},
		},
	)
}

func logText(t *testing.T, artifact NativeArtifact, path string) string {
	t.Helper()
	entry, ok := findEntry(artifact, path)
	if !ok {
		var paths []string
		for _, e := range artifact.Entries {
			paths = append(paths, e.Path)
		}
		t.Fatalf("no %s in the artifact; entries=%v", path, paths)
	}
	return string(entry.Bytes)
}

func bodyLines(text string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

func TestCrossRefPositionsPointAtRealBytes(t *testing.T) {
	artifact, err := BuildNativeShadow(defaultLintSnapshot(), nil, false)
	if err != nil {
		t.Fatalf("an informational lint must never fail the export: %v", err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")
	findingRE := regexp.MustCompile(`^(\S+) · (\S+) -> (\S+?):(\S+) · (.+):(\d+):(\d+)$`)
	checked := 0
	for _, line := range bodyLines(text) {
		if strings.HasPrefix(line, "warning · ") {
			continue
		}
		match := findingRE.FindStringSubmatch(line)
		if match == nil {
			t.Fatalf("finding line does not parse: %q", line)
		}
		targetID, relPath := match[4], match[5]
		entry, ok := findEntry(artifact, "Demo/"+relPath)
		if !ok {
			t.Fatalf("finding names a file the artifact does not carry: %q", relPath)
		}
		lineNo, _ := strconv.Atoi(match[6])
		column, _ := strconv.Atoi(match[7])
		fileLines := strings.Split(string(entry.Bytes), "\n")
		if lineNo < 1 || lineNo > len(fileLines) {
			t.Fatalf("line %d is outside %s (%d lines)", lineNo, relPath, len(fileLines))
		}
		source := fileLines[lineNo-1]
		runes := []rune(source)
		if column < 1 || column-1+utf8.RuneCountInString(targetID) > len(runes) {
			t.Fatalf("column %d is outside %s:%d (%q)", column, relPath, lineNo, source)
		}
		got := string(runes[column-1 : column-1+utf8.RuneCountInString(targetID)])
		if got != targetID {
			t.Fatalf("%s:%d:%d holds %q, report claims %q", relPath, lineNo, column, got, targetID)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no positioned findings were produced, so the position check proved nothing")
	}
}

func TestCrossRefReportsOnlyUnresolvedReferences(t *testing.T) {
	artifact, err := BuildNativeShadow(defaultLintSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")
	for _, want := range []string{
		"missing · has_completed_focus -> focus:LANE_L_absent · common/decisions/lane_l.txt:",
		"missing · unlock_decision_tooltip -> decision:LANE_L_ghost_decision · common/decisions/lane_l.txt:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q:\n%s", want, text)
		}
	}

	for _, banned := range []string{"LANE_L_present", "LANE_L_known_decision"} {
		if strings.Contains(text, banned) {
			t.Fatalf("resolved reference %q was reported as a finding:\n%s", banned, text)
		}
	}

	if !strings.Contains(text, "warning · targeted_modifier[missing_tag · ") {
		t.Fatalf("targeted_modifier without `tag = X` was not surfaced:\n%s", text)
	}
}

func TestCrossRefHeaderClaimsExactlyWhatIsChecked(t *testing.T) {
	artifact, err := BuildNativeShadow(defaultLintSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")

	for _, want := range []string{
		"# References between workbenches (notices only)",
		"# No assignment in your content was deleted or rewritten.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("header line dropped: %q\n%s", want, text)
		}
	}
	supported := []string{
		"autonomy_state", "character", "decision", "dynamic_modifier", "event",
		"focus", "idea", "mio", "opinion_modifier", "technology", "trait",
	}
	want := "# Reference types checked: " + strings.Join(supported, ", ") + "."
	if !strings.Contains(text, want) {
		t.Fatalf("header does not claim exactly _SUPPORTED_TYPES.\nwant line: %s\ngot:\n%s", want, text)
	}
	if strings.Contains(text, "NOT checked") {
		t.Fatalf("the coverage-gap disclaimer outlived the gap:\n%s", text)
	}
}

func TestEveryCheckedTypeHasANonEmptyCatalogue(t *testing.T) {
	sets := crossRefExistenceSets(defaultLintSnapshot())

	projectOnly := map[string]bool{"focus": true, "decision": true, "mio": true}

	wantAtLeast := map[string]int{
		"idea": 1, "event": 1, "trait": 1, "technology": 1,
		"character": 1, "opinion_modifier": 1, "dynamic_modifier": 1,
		"autonomy_state": 1,
	}
	for _, targetType := range checkedRefTypes() {
		if projectOnly[targetType] {
			continue
		}
		floor, ok := wantAtLeast[targetType]
		if !ok {
			t.Fatalf("checked type %q has no catalogue expectation", targetType)
		}
		if got := len(sets[targetType]); got < floor {
			t.Fatalf("%s catalogue holds %d ids, want >= %d — an empty catalogue "+
				"turns every vanilla reference into a false finding", targetType, got, floor)
		}
	}
}

func TestVanillaReferencesResolveWithoutProjectRows(t *testing.T) {
	script := `lane_l_category = {
	lane_l_vanilla = {
		available = {
			has_idea = limited_exports
			has_trait = brilliant_strategist
		}
		complete_effect = {
			country_event = generic.1
			promote_character = SOV_iosif_stalin
			add_dynamic_modifier = { modifier = fallout_atomic }
			add_opinion_modifier = { target = USA modifier = embargo }
			set_technology = { infantry_weapons = 1 }
		}
	}
}
`
	snapshot := lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`, nil, nil)
	snapshot.Tables["project_raw_files"] = []json.RawMessage{
		json.RawMessage(rawRow(1, "common/decisions/lane_l.txt", script)),
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if entry, ok := findEntry(artifact, "Demo/_CROSS_REF_SKIPPED.log"); ok {
		t.Fatalf("base-game references were reported as unresolved:\n%s", entry.Bytes)
	}
}

func TestSuperEventIDsResolveAsEvents(t *testing.T) {
	snapshot := lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"USA","dependencies":[]}`, nil,
		map[string][]string{"super_events": {`{"id":1,"country_tag":"","number":7}`}},
	)
	snapshot.Tables["project_raw_files"] = []json.RawMessage{
		json.RawMessage(rawRow(1, "common/decisions/lane_l.txt",
			"d = { x = { complete_effect = { country_event = usa_se.7 country_event = usa_se.8 } } }\n")),
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")
	if strings.Contains(text, "usa_se.7") {
		t.Fatalf("a modeled SuperEvent id was reported unresolved:\n%s", text)
	}
	if !strings.Contains(text, "event:usa_se.8") {
		t.Fatalf("an unmodeled SuperEvent id was not reported:\n%s", text)
	}
}

func TestTokenLintMatchesPythonBodyFormat(t *testing.T) {
	artifact, err := BuildNativeShadow(defaultLintSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_TOKEN_LINT.log")
	if !strings.HasPrefix(text, "# Unknown tokens\n") {
		t.Fatalf("header line drifted:\n%s", text)
	}
	body := bodyLines(text)
	want := []string{"unknown_tech_category · lane_l_not_a_category: 1"}
	if fmt.Sprint(body) != fmt.Sprint(want) {
		t.Fatalf("token lint body = %v, want %v\n%s", body, want, text)
	}

	for _, line := range body {
		if strings.Contains(line, "industry") {
			t.Fatalf("a legitimate vanilla category was flagged: %q", line)
		}
	}

	for _, banned := range []string{"ERROR", "INVALID", "BLOCKED"} {
		if strings.Contains(strings.ToUpper(text), banned) {
			t.Fatalf("informational lint worded as a blocker (%s):\n%s", banned, text)
		}
	}
}

func TestTokenLintHonoursCategoriesTheModDeclares(t *testing.T) {
	snapshot := lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`,
		[]string{rawRow(2, "common/technology_tags/lane_l.txt",
			"technology_categories = { lane_l_not_a_category }\n")},
		map[string][]string{
			"focus_nodes": {`{"id":1,"focus_id":"LANE_L_present","focus_tree_id":1}`},
			"decisions":   {`{"id":1,"decision_id":"LANE_L_known_decision"}`},
		},
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findEntry(artifact, "Demo/_TOKEN_LINT.log"); ok {
		t.Fatalf("category declared in common/technology_tags/ was still reported:\n%s",
			logText(t, artifact, "Demo/_TOKEN_LINT.log"))
	}
}

func TestSubmodNeverReportsMissing(t *testing.T) {
	snapshot := lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"GER","dependencies":["The Road to 56"]}`,
		nil,
		map[string][]string{
			"focus_nodes": {`{"id":1,"focus_id":"LANE_L_present","focus_tree_id":1}`},
			"decisions":   {`{"id":1,"decision_id":"LANE_L_known_decision"}`},
		},
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	text := logText(t, artifact, "Demo/_CROSS_REF_SKIPPED.log")
	if strings.Contains(text, "missing · ") {
		t.Fatalf("a submod's base-mod reference was called missing:\n%s", text)
	}
	if !strings.Contains(text, "base_mod_not_indexed · has_completed_focus -> focus:LANE_L_absent") {
		t.Fatalf("expected base_mod_not_indexed for an unindexed base reference:\n%s", text)
	}
}

func TestNoLogsWhenBothChecksFindNothing(t *testing.T) {
	snapshot := lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`,
		nil,
		map[string][]string{
			"focus_nodes": {`{"id":1,"focus_id":"LANE_L_present","focus_tree_id":1}`},
			"decisions":   {`{"id":1,"decision_id":"LANE_L_known_decision"}`},
		},
	)

	snapshot.Tables["project_raw_files"] = []json.RawMessage{json.RawMessage(rawRow(
		1, "common/decisions/lane_l.txt",
		"lane_l_category = {\n\tlane_l_decision = {\n\t\tavailable = { has_completed_focus = LANE_L_present }\n\t}\n}\n",
	))}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"Demo/_CROSS_REF_SKIPPED.log", "Demo/_TOKEN_LINT.log"} {
		if _, ok := findEntry(artifact, path); ok {
			t.Fatalf("%s written for a clean project:\n%s", path, logText(t, artifact, path))
		}
	}
}

func TestLintLogsSitInsideTheModFolder(t *testing.T) {
	artifact, err := BuildNativeShadow(defaultLintSnapshot(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"Demo/_CROSS_REF_SKIPPED.log", "Demo/_TOKEN_LINT.log"} {
		entry, ok := findEntry(artifact, path)
		if !ok {
			t.Fatalf("%s not found inside the mod folder", path)
		}
		if !strings.HasSuffix(string(entry.Bytes), "\n") {
			t.Fatalf("%s must end with a newline", path)
		}
	}
	if _, ok := findEntry(artifact, "Demo_CROSS_REF_SKIPPED.log"); ok {
		t.Fatal("report written as a ZIP-root sibling; Python puts it inside <modDir>/")
	}
}

func TestUnparseableScriptNeverBreaksTheExport(t *testing.T) {
	snapshot := lintSnapshot(
		`{"id":46,"name":"Demo","country_tag":"GER","dependencies":[]}`,
		[]string{rawRow(2, "common/decisions/broken.txt", "this = { is = { not\n")},
		map[string][]string{"decisions": {`{"id":1,"decision_id":"LANE_L_known_decision"}`}},
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("a malformed author file must not fail the export: %v", err)
	}
	if text, ok := findEntry(artifact, "Demo/_CROSS_REF_SKIPPED.log"); ok {
		if strings.Contains(string(text.Bytes), "broken.txt") {
			t.Fatalf("findings claimed from a file that did not parse:\n%s", text.Bytes)
		}
	}
}
