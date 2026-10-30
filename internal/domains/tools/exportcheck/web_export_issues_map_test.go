package exportcheck

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestStateNoLandIssuesOnTheBaseGameMap(t *testing.T) {
	game := fstest.MapFS{"map/definition.csv": {Data: []byte(`0;0;0;0;land;false;unknown;0
715;1;1;1;land;false;plains;1
6778;2;2;2;land;false;plains;1
11662;3;3;3;land;false;plains;1
13157;4;4;4;lake;false;lakes;0
13158;5;5;5;lake;false;lakes;0
`)}}
	setGame(func() (fs.FS, error) { return game, nil })
	t.Cleanup(func() { setGame(nil) })
	states := []ValidateState{
		{StateID: 357, Name: "New England", Provinces: []int64{13158}},
		{StateID: 358, Name: "New York", Provinces: []int64{13157, 6778}},
		{StateID: 1082, Name: "", Provinces: []int64{11662, 715}},
	}
	got := stateNoLandIssues(states, nil)
	if len(got) != 1 {
		t.Fatalf("want one issue, got %+v", got)
	}
	issue := got[0]
	if issue["code"] != "map.state_no_land" || issue["severity"] != "blocker" || issue["repair_route"] != "map_studio.repair" {
		t.Fatalf("issue %+v", issue)
	}
	if evidence := issue["evidence"].(map[string]any); evidence["state_id"] != int64(357) || evidence["name"] != "New England" {
		t.Fatalf("evidence %+v", evidence)
	}

	setGame(nil)
	if got := stateNoLandIssues(states, nil); len(got) != 0 {
		t.Fatalf("no game folder: %+v", got)
	}
}

func TestStateNoLandIssuesReadsTheProjectsOwnDefinitions(t *testing.T) {
	raw := []gateRawFile{{DestPath: "map/definition.csv", Content: "0;0;0;0;land;false;unknown;0\n1;1;1;1;land;false;plains;1\n2;2;2;2;lake;false;lakes;0\n"}}
	states := []ValidateState{{StateID: 1, Provinces: []int64{1, 2}}, {StateID: 2, Provinces: []int64{2}}}
	if got := stateNoLandIssues(states, raw); len(got) != 1 || got[0]["entity_ref"].(map[string]any)["id"] != int64(2) {
		t.Fatalf("got %+v", got)
	}

	if got := stateNoLandIssues(states, []gateRawFile{{DestPath: "map/definition.csv", IsBinary: true}}); len(got) != 0 {
		t.Fatalf("binary definitions: %+v", got)
	}
}
