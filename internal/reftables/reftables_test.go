package reftables

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuiltInTables(t *testing.T) {
	ideologies := Ideologies()
	if !reflect.DeepEqual(ideologies.Groups, []string{"democratic", "communism", "fascism", "neutrality"}) || len(ideologies.SubIdeologies) != 22 {
		t.Fatalf("ideologies: %v, %d sub-ideologies", ideologies.Groups, len(ideologies.SubIdeologies))
	}
	if !ideologies.IsGroup("fascism") || ideologies.IsGroup("rexism") || !ideologies.IsSub("rexism") || ideologies.GroupOf("rexism") != "fascism" {
		t.Error("group / sub-ideology lookups")
	}
	if got := ideologies.Members("neutrality"); !reflect.DeepEqual(got, []string{"neutrality", "despotism", "oligarchism", "anarchism", "moderatism", "centrism", "japan_militarism_ideology"}) {
		t.Errorf("neutrality members = %v", got)
	}
	alerts := Alerts()
	if len(alerts.Engine) != 70 || alerts.Engine[0] != (EngineAlert{8, "alert_no_research", 1}) || alerts.TextKeys["alert_losing_trains"] != [2]string{"ALERT_LOSING_TRAINS", ""} {
		t.Errorf("alerts: %d, first %+v", len(alerts.Engine), alerts.Engine[0])
	}
	var colors textColors
	if err := json.Unmarshal(Raw(TextColorsFile), &colors); err != nil || colors.Global["G"] != "rgb(0, 159, 3)" || colors.Fonts["hoi_18mbs"]["G"] != "rgb(86, 172, 91)" {
		t.Errorf("text colours: %+v %v", colors, err)
	}
}

func TestLoadReadsThePlayersTables(t *testing.T) {
	t.Cleanup(func() { Load("") })
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(IdeologiesFile, "\xef\xbb\xbf"+`{"groups": ["syndicalism"], "sub_ideologies": [{"id": "radical_socialism", "group": "syndicalism"}]}`)
	write(AlertsFile, `{"engine": [], "text_keys": {}}`)
	write(TextColorsFile, `{"global": {"G": `)
	Load(dir)

	if got := Ideologies(); !reflect.DeepEqual(got.Groups, []string{"syndicalism"}) || got.IsGroup("democratic") || got.GroupOf("radical_socialism") != "syndicalism" {
		t.Errorf("ideologies = %+v", got)
	}
	if string(Raw(IdeologiesFile)) != `{"groups":["syndicalism"],"sub_ideologies":[{"id":"radical_socialism","group":"syndicalism"}]}` {
		t.Errorf("raw ideologies = %s", Raw(IdeologiesFile))
	}
	if got := Alerts(); len(got.Engine) != 0 || len(got.TextKeys) != 0 {
		t.Errorf("alerts = %+v", got)
	}
	var colors textColors
	if err := json.Unmarshal(Raw(TextColorsFile), &colors); err != nil || colors.Global["G"] == "" {
		t.Errorf("a broken text_colors.json replaced the built-in one: %s", Raw(TextColorsFile))
	}

	Load("")
	if len(Ideologies().SubIdeologies) != 22 || len(Alerts().Engine) != 70 {
		t.Error("Load(\"\") did not restore the built-in tables")
	}
}

func TestUseRestores(t *testing.T) {
	restore := Use(map[string][]byte{IdeologiesFile: []byte(`{"groups": [], "sub_ideologies": []}`)})
	if len(Ideologies().Groups) != 0 || len(Alerts().Engine) != 70 {
		t.Fatal("Use did not replace only the given table")
	}
	restore()
	if len(Ideologies().Groups) != 4 {
		t.Fatal("restore did not put the built-in table back")
	}
}
