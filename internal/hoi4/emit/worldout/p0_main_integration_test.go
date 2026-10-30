package worldout

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCustomPoliticsAndOOBShareExistingHistoryBuffer(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := NewOMap()
	project.Set("export_languages", []Value{"en"})
	project.Set("ideology_registry", managedRegistry())

	history := NewOMap()
	history.Set("country_tag", "TCH")
	history.Set("set_politics_ruling_party", "technocracy")
	popularities := NewOMap()
	popularities.Set("technocracy", json.Number("60"))
	popularities.Set("democratic", json.Number("40"))
	history.Set("set_popularities", popularities)

	country := NewOMap()
	country.Set("tag", "TCH")
	content := NewOMap()
	content.Set("TCH", true)
	scenario := &Scenario{
		Project:       project,
		Tag:           "TCH",
		ModName:       "Integration Test",
		ModDirName:    "integration_test",
		ExistenceSets: map[string]map[string]bool{"idea": {}, "focus": {}},
		ContentByTag:  content,
		Countries:     []*OMap{country},
		HistoryFiles:  []*OMap{history},
		OOBFiles:      []OOBFile{{Path: "history/units/TCH_1936.txt", Tags: []string{"TCH"}}},
	}
	records, _ := EmitScenario(tables, scenario)
	body := ""
	for _, record := range records {
		if strings.HasPrefix(record.Path, "integration_test/history/countries/TCH - ") {
			body = record.Text
			break
		}
	}
	if body == "" {
		t.Fatalf("combined country-history record was not emitted: %+v", records)
	}

	if got := strings.Count(body, "set_politics = {"); got != 1 {
		t.Fatalf("set_politics blocks = %d, want exactly one:\n%s", got, body)
	}
	for _, want := range []string{
		"ruling_party = technocracy",
		"technocracy = 60",
		"democratic = 40",
		`set_oob = "TCH_1936"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("combined history lost %q:\n%s", want, body)
		}
	}
}
