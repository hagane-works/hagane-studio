package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/emit/locout"
)

func TestPlaceholderIDStemsComeFromLocoutSet(t *testing.T) {
	for _, name := range locout.PlaceholderDisplayNamesSorted() {
		token := strings.ReplaceAll(name, " ", "_")
		if !IsPlaceholderDerivedID(token) {
			t.Fatalf("placeholder display name %q → id %q must be recognised", name, token)
		}
		if !IsPlaceholderDerivedID(token + "_3") {
			t.Fatalf("collision-suffixed %q_3 must be recognised", token)
		}
	}
}

func TestIsPlaceholderDerivedIDBoundaries(t *testing.T) {
	hits := []string{"new_idea", "new_idea_1", "new_idea_47", "NEW_IDEA", "New_Focus_2",
		"new_technology", "new_character_9", "  new_idea_3  "}
	for _, id := range hits {
		if !IsPlaceholderDerivedID(id) {
			t.Errorf("%q must be flagged", id)
		}
	}

	misses := []string{"", "   ", "new_ideas", "new_idea_bank", "ger_new_idea",
		"new_idea_", "new_idea_1a", "newidea", "new_ideal", "great_depression",
		"new_industry_2", "focus_new_technology"}
	for _, id := range misses {
		if IsPlaceholderDerivedID(id) {
			t.Errorf("%q must NOT be flagged", id)
		}
	}
}

func placeholderIDSnapshot(tables map[string][]json.RawMessage) Snapshot {
	return Snapshot{
		ProjectID: 42,
		Project:   json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables:    tables,
	}
}

func TestScanPlaceholderIDsGroupsPerTable(t *testing.T) {
	report := ScanPlaceholderIDs(placeholderIDSnapshot(map[string][]json.RawMessage{
		"ideas": {
			json.RawMessage(`{"id":1,"idea_id":"new_idea","name":"New Idea"}`),
			json.RawMessage(`{"id":2,"idea_id":"new_idea_1","name":"New Idea"}`),
			json.RawMessage(`{"id":3,"idea_id":"great_depression","name":"Great Depression"}`),
			json.RawMessage(`{"id":4,"idea_id":"new_idea_1","name":"dup row"}`),
		},
		"focus_nodes":  {json.RawMessage(`{"id":5,"focus_id":"NEW_FOCUS","name":"New Focus"}`)},
		"technologies": {json.RawMessage(`{"id":6,"tech_id":"ger_radar"}`)},
		"characters":   {json.RawMessage(`{"id":7,"tag":"new_character_2"}`)},
	}), english)
	if report.Count() != 4 {
		t.Fatalf("want 4 distinct placeholder ids, got %d (%v)", report.Count(), report.IDs)
	}
	if report.Tables["ideas"] != 2 || report.Tables["focus_nodes"] != 1 ||
		report.Tables["characters"] != 1 || report.Tables["technologies"] != 0 {
		t.Fatalf("per-table counts wrong: %v", report.Tables)
	}
	head := report.Headline()
	for _, want := range []string{"new_idea", "new_idea_1", "NEW_FOCUS", "new_character_2",
		"shows the raw ID instead", "characters: 1", "focus_nodes: 1", "ideas: 2"} {
		if !strings.Contains(head, want) {
			t.Fatalf("headline missing %q:\n%s", want, head)
		}
	}
	if strings.Contains(head, "ger_radar") {
		t.Fatalf("author-written id leaked into the headline:\n%s", head)
	}
}

func TestScanPlaceholderIDsQuietAndCapped(t *testing.T) {
	if got := ScanPlaceholderIDs(placeholderIDSnapshot(nil), english).Headline(); got != "" {
		t.Fatalf("empty snapshot must be silent, got %q", got)
	}
	clean := ScanPlaceholderIDs(placeholderIDSnapshot(map[string][]json.RawMessage{
		"ideas":      {json.RawMessage(`{"id":1,"idea_id":"GER_demo"}`)},
		"characters": {json.RawMessage(`{"id":2}`)},
	}), english)
	if clean.Count() != 0 || clean.Headline() != "" {
		t.Fatalf("author-named project must be silent: %v", clean.IDs)
	}
	rows := make([]json.RawMessage, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, json.RawMessage(`{"id":`+itoaTest(i)+`,"idea_id":"new_idea_`+itoaTest(i)+`"}`))
	}
	head := ScanPlaceholderIDs(placeholderIDSnapshot(map[string][]json.RawMessage{"ideas": rows}), english).Headline()
	if !strings.Contains(head, "new_idea_5 and 4 more") {
		t.Fatalf("headline must cap at 8 ids and say 'and 4 more':\n%s", head)
	}
}

func itoaTest(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
