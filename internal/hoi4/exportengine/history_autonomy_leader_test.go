package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHistoryWritesAutonomyStatesAndCountryLeaders(t *testing.T) {
	snapshot := Snapshot{ProjectID: 2,
		Project: json.RawMessage(`{"id":2,"name":"Fixture","country_tag":"CHI","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"history_files": {
				json.RawMessage(`{"id":1,"country_tag":"CHI","source":"imported","capital":608,` +
					`"set_autonomy":[{"target":"SHX","autonomous_state":"fds_nominal","state_key":"autonomy_state","freedom_level":"0.0"},{"target":"MAN","autonomous_state":"autonomy_puppet"}],` +
					`"raw_script":"create_country_leader = {\n\tname = \"Ahmed Ben Bella\"\n\tideology = fds_095_personalist_dictatorship\n}"}`),
			},
		}}
	files, err := BuildNativeTextTree(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	history := ""
	for name, body := range files {
		if strings.HasPrefix(name, "history/countries/CHI") {
			history = string(body)
		}
	}
	for _, want := range []string{
		"\tautonomy_state = fds_nominal\n\tfreedom_level = 0.0\n",
		"\ttarget = MAN\n\tautonomous_state = autonomy_puppet\n}",
		"create_country_leader = {", "Ahmed Ben Bella",
	} {
		if !strings.Contains(history, want) {
			t.Errorf("history lost %q:\n%s", want, history)
		}
	}
}
