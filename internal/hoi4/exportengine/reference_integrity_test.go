package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func referenceSnapshot(project string) Snapshot {
	return Snapshot{ProjectID: 61, Project: json.RawMessage(project), Tables: map[string][]json.RawMessage{}}
}

func findingCodes(report ReferenceIntegrityReport) map[string]ReferenceIntegrityFinding {
	out := map[string]ReferenceIntegrityFinding{}
	for _, finding := range report.Findings {
		out[finding.Code] = finding
	}
	return out
}

func TestReferenceIntegrityReadsFinalRawAndGeneratedFocusFiles(t *testing.T) {
	entries := []Entry{
		{
			Path: "Demo/common/national_focus/generated.txt", Origin: "generated",
			Bytes: []byte(`focus_tree = {
	id = GER_demo
	focus = { id = GER_Path }
	focus = {
		id = GER_End
		prerequisite = { focus = ger_path }
		mutually_exclusive = { focus = GER_Ghost }
	}
}
`),
		},
		{
			Path: "Demo/common/decisions/raw.txt", Origin: "raw",
			Bytes: []byte(`decisions = {
	GER_check = { available = { has_completed_focus = Ger_Path } }
}
`),
		},
	}
	report := BuildReferenceIntegrityReport(
		referenceSnapshot(`{"id":61,"name":"Demo","dependencies":[]}`), entries, english,
	)
	codes := findingCodes(report)
	for _, code := range []string{
		"focus_prerequisite_case", "focus_mutually_exclusive", "has_completed_focus_case",
	} {
		if _, ok := codes[code]; !ok {
			t.Fatalf("missing %s; findings=%+v", code, report.Findings)
		}
	}
	if got := codes["focus_prerequisite_case"]; got.File != "common/national_focus/generated.txt" || got.Line != 6 {
		t.Fatalf("prerequisite locator = %+v", got)
	}
	if got := codes["has_completed_focus_case"]; got.File != "common/decisions/raw.txt" || got.Line != 2 {
		t.Fatalf("raw has_completed_focus locator = %+v", got)
	}
	for _, finding := range report.Findings {
		if finding.Consequence == "" || finding.EditAt == "" {
			t.Fatalf("finding lacks player consequence or editing entrance: %+v", finding)
		}
	}
}

func TestReferenceIntegrityChecksDeploymentAndCreateUnitAgainstCountryOOB(t *testing.T) {
	entries := []Entry{
		{
			Path:  "Demo/history/countries/GER - Germany.txt",
			Bytes: []byte("capital = 64\noob = \"GER_1964\"\n"),
		},
		{
			Path: "Demo/history/units/GER_1964.txt",
			Bytes: []byte(`division_template = {
	name = "Royal Infantry"
	regiments = { infantry = { x = 0 y = 0 } }
}
units = {
	division = {
		name = "First"
		division_template = "Royal Infantry"
		location = 1
	}
	division = {
		name = "Second"
		division_template = "Ghost Division"
		location = 2
	}
}
`),
		},
		{
			Path: "Demo/events/ger_units.txt",
			Bytes: []byte(`add_namespace = ger_units
country_event = {
	id = ger_units.1
	immediate = {
		create_unit = {
			division = "name = \"Good\" division_template = \"Royal Infantry\" start_equipment_factor = 1"
			owner = GER
		}
		create_unit = {
			division = "name = \"Bad\" division_template = \"Royal infantry\" start_equipment_factor = 1"
			owner = GER
		}
	}
}
`),
		},
	}
	report := BuildReferenceIntegrityReport(
		referenceSnapshot(`{"id":61,"name":"Demo","dependencies":[]}`), entries, english,
	)
	codes := findingCodes(report)
	if got := codes["oob_deployment_template"]; got.Target != "GER:Ghost Division" || got.File != "history/units/GER_1964.txt" {
		t.Fatalf("OOB deployment finding = %+v", got)
	}
	if got := codes["create_unit_template"]; got.Target != "GER:Royal infantry" || got.File != "events/ger_units.txt" {
		t.Fatalf("create_unit finding = %+v", got)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("resolved Royal Infantry should stay quiet; findings=%+v", report.Findings)
	}
}

func TestReferenceIntegritySubmodSaysSourceUnregistered(t *testing.T) {
	entries := []Entry{{
		Path: "Sub/common/national_focus/sub.txt",
		Bytes: []byte(`focus_tree = {
	id = SUB
	focus = {
		id = SUB_Start
		prerequisite = { focus = BASE_Unknown }
	}
}
`),
	}}
	report := BuildReferenceIntegrityReport(
		referenceSnapshot(`{"id":61,"name":"Sub","dependencies":["TNO"]}`), entries, english,
	)
	if len(report.Findings) != 1 {
		t.Fatalf("findings=%+v", report.Findings)
	}
	got := report.Findings[0].Summary
	if !strings.Contains(got, "has no registered source") || !strings.Contains(got, "does not prove it is missing") {
		t.Fatalf("submod wording does not acknowledge unknown source: %s", got)
	}
	for _, forbidden := range []string{"is not in this export", "does not exist"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("submod finding made an unsupported absence claim %q: %s", forbidden, got)
		}
	}
}

func TestReferenceIntegrityReusesExistingIdeaAndOpinionCoverage(t *testing.T) {
	snapshot := referenceSnapshot(`{"id":61,"name":"Demo","dependencies":[]}`)
	snapshot.Tables["ideas"] = []json.RawMessage{
		json.RawMessage(`{"id":1,"idea_id":"known_idea"}`),
	}
	snapshot.Tables["opinion_modifiers"] = []json.RawMessage{
		json.RawMessage(`{"id":1,"modifier_id":"known_opinion"}`),
	}
	entries := []Entry{{
		Path: "Demo/events/references.txt",
		Bytes: []byte(`x = {
	add_ideas = { known_idea ghost_idea }
	swap_ideas = { remove_idea = known_idea add_idea = ghost_swap }
	add_opinion_modifier = { target = ENG modifier = known_opinion }
	add_opinion_modifier = { target = ENG modifier = ghost_opinion }
}
`),
	}}
	report, err := BuildLintReport(snapshot, entries, english)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(report.CrossRef, "\n")
	for _, want := range []string{"idea:ghost_idea", "idea:ghost_swap", "opinion_modifier:ghost_opinion"} {
		if !strings.Contains(text, want) {
			t.Fatalf("existing cross_ref coverage missing %q:\n%s", want, text)
		}
	}
	for _, resolved := range []string{"idea:known_idea", "opinion_modifier:known_opinion"} {
		if strings.Contains(text, resolved) {
			t.Fatalf("existing cross_ref reported resolved %q:\n%s", resolved, text)
		}
	}
}

func TestReferenceIntegrityReportIsAdvisoryAndActionable(t *testing.T) {
	finding := ReferenceIntegrityFinding{
		Code: "focus_prerequisite", File: "common/national_focus/a.txt", Line: 7,
		Target: "A:B", Summary: "Prerequisite B is not in this export.",
		Consequence: "Players may be unable to continue.", EditAt: "The focus tree's prerequisites.",
	}
	report := ReferenceIntegrityReport{Findings: []ReferenceIntegrityFinding{finding}}
	entries := report.Entries("Demo")
	if len(entries) != 1 || entries[0].Path != "Demo/_REFERENCE_INTEGRITY.log" {
		t.Fatalf("report entries=%+v", entries)
	}
	text := string(entries[0].Bytes)
	for _, want := range []string{
		"notices only", "common/national_focus/a.txt:7", "Player impact: Players may be unable to continue.", "Where to fix: The focus tree's",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "blocked") {
		t.Fatalf("advisory report was worded as a block:\n%s", text)
	}

	zh := ReferenceIntegrityReport{Findings: []ReferenceIntegrityFinding{finding}, msg: chinese}
	text = string(zh.Entries("Demo")[0].Bytes)
	for _, want := range []string{"# " + chinese.Text("refs.log.title", nil), chinese.Text("refs.label.review", nil) + " · focus_prerequisite · common/national_focus/a.txt:7"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Chinese report missing %q:\n%s", want, text)
		}
	}
}

func TestNativeReferenceIntegritySeamPublishesFinalArtifactReport(t *testing.T) {
	snapshot := referenceSnapshot(`{"id":61,"name":"Demo","country_tag":"GER","dependencies":[]}`)
	snapshot.Tables["project_raw_files"] = []json.RawMessage{json.RawMessage(rawRow(
		1, "common/national_focus/broken.txt",
		"focus_tree = { id = GER focus = { id = GER_A prerequisite = { focus = GER_GHOST } } }\n",
	))}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findEntry(artifact, "Demo/_REFERENCE_INTEGRITY.log"); !ok {
		t.Fatal("central seam missing: final artifact did not publish reference-integrity findings")
	}
	if !strings.Contains(strings.Join(artifact.Warnings, "\n"), "Script reference check") {
		t.Fatal("central seam missing: export page received no player-facing reference-integrity headline")
	}
}
