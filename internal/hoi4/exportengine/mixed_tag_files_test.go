package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// One imported file can define content for several countries: shared laws
// next to one country's variants, or decisions whose categories belong to
// different countries. The export must write that file once with all of it.
func TestFileSharedByCountriesKeepsEveryCountrysContent(t *testing.T) {
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 42,
		Project: json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"ideas": {
				json.RawMessage(`{"id":1,"idea_id":"shared_law","idea_type":"mobilization_laws","country_tag":"","source":"imported","source_file":"common/ideas/laws.txt","modifier":{},"display_name_locs":{},"description_locs":{}}`),
				json.RawMessage(`{"id":2,"idea_id":"USA_law","idea_type":"mobilization_laws","country_tag":"USA","source":"imported","source_file":"common/ideas/laws.txt","modifier":{},"display_name_locs":{},"description_locs":{}}`),
				json.RawMessage(`{"id":3,"idea_id":"SOV_law","idea_type":"mobilization_laws","country_tag":"SOV","source":"imported","source_file":"common/ideas/laws.txt","modifier":{},"display_name_locs":{},"description_locs":{}}`),
			},
			"decision_categories": {
				json.RawMessage(`{"id":1,"category_id":"shared_category","country_tag":"","source":"imported","source_file":"common/decisions/categories/00_decision_categories.txt"}`),
				json.RawMessage(`{"id":2,"category_id":"usa_category","country_tag":"USA","source":"imported","source_file":"common/decisions/categories/00_decision_categories.txt"}`),
				// Defined only by the base game; the mod's decisions file
				// just wraps its decisions in it.
				json.RawMessage(`{"id":3,"category_id":"ARG_balancing_act_decisions","country_tag":"ARG","source":"imported"}`),
			},
			"decisions": {
				json.RawMessage(`{"id":1,"decision_id":"shared_decision","category_id":1,"source":"imported","source_file":"common/decisions/endgame.txt","display_name_locs":{},"description_locs":{}}`),
				json.RawMessage(`{"id":2,"decision_id":"usa_decision","category_id":2,"country_tag":"USA","source":"imported","source_file":"common/decisions/endgame.txt","display_name_locs":{},"description_locs":{}}`),
				json.RawMessage(`{"id":3,"decision_id":"ARG_decision","category_id":3,"country_tag":"ARG","source":"imported","source_file":"common/decisions/ARG.txt","display_name_locs":{},"description_locs":{}}`),
			},
		},
	}
	files, err := BuildNativeTextTree(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	ideas := string(files["common/ideas/laws.txt"])
	for _, id := range []string{"shared_law", "USA_law", "SOV_law"} {
		if !strings.Contains(ideas, id+" = {") {
			t.Errorf("common/ideas/laws.txt lost %s:\n%s", id, ideas)
		}
	}
	decisions := string(files["common/decisions/endgame.txt"])
	for _, id := range []string{"shared_category", "shared_decision", "usa_category", "usa_decision"} {
		if !strings.Contains(decisions, id+" = {") {
			t.Errorf("common/decisions/endgame.txt lost %s:\n%s", id, decisions)
		}
	}
	categories := string(files["common/decisions/categories/00_decision_categories.txt"])
	for _, id := range []string{"shared_category", "usa_category"} {
		if !strings.Contains(categories, id+" = {") {
			t.Errorf("common/decisions/categories/00_decision_categories.txt lost %s:\n%s", id, categories)
		}
	}
	for name, body := range files {
		if strings.HasPrefix(name, "common/decisions/categories/") && name != "common/decisions/categories/00_decision_categories.txt" {
			t.Errorf("categories also written to %s", name)
		}
		if strings.HasPrefix(name, "common/decisions/categories/") && strings.Contains(string(body), "ARG_balancing_act_decisions") {
			t.Errorf("%s redefines a category only the base game defines", name)
		}
	}
	if arg := string(files["common/decisions/ARG.txt"]); !strings.Contains(arg, "ARG_balancing_act_decisions = {") || !strings.Contains(arg, "ARG_decision = {") {
		t.Errorf("common/decisions/ARG.txt lost the decision or its category wrapper:\n%s", arg)
	}
}
