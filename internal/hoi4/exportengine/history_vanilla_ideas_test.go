package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHistoryAddIdeasKeepsBaseGameIdeas(t *testing.T) {
	snapshot := Snapshot{ProjectID: 2,
		Project: json.RawMessage(`{"id":2,"name":"Ideas fixture","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"history_files": {json.RawMessage(`{"id":1,"country_tag":"GER","source":"created",` +
				`"add_ideas_extra":["sour_loser","limited_conscription","my_project_idea","walther_funk","GER_franz_seldte","no_such_idea_anywhere"]}`)},
			"ideas": {json.RawMessage(`{"id":7,"idea_id":"my_project_idea","category":"country","country_tag":"GER","source":"created"}`)},

			"characters": {json.RawMessage(`{"id":3,"tag":"GER_walther_funk","country_tag":"GER","has_advisor_role":true,` +
				`"advisor_slot":"political_advisor","advisor_idea_token":"walther_funk","source":"imported"}`)},
		}}
	files, err := BuildNativeTextTree(snapshot, false)
	if err != nil {
		t.Fatal(err)
	}
	history := ""
	for name, body := range files {
		if strings.HasPrefix(name, "history/countries/GER") {
			history = string(body)
		}
	}
	if history == "" {
		t.Fatalf("no German history in %v", keysOf(files))
	}

	for _, want := range []string{"sour_loser", "limited_conscription", "my_project_idea", "walther_funk", "GER_franz_seldte"} {
		if !strings.Contains(history, "\t"+want+"\n") {
			t.Errorf("add_ideas lost %s:\n%s", want, history)
		}
	}
	if strings.Contains(history, "no_such_idea_anywhere") {
		t.Errorf("an idea nothing declares was written:\n%s", history)
	}
}

func keysOf(files map[string][]byte) []string {
	out := make([]string, 0, len(files))
	for name := range files {
		out = append(out, name)
	}
	return out
}
