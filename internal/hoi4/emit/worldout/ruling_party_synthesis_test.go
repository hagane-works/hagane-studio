package worldout

import (
	"strings"
	"testing"
)

func TestRulingPartyIsSynthesizedOnlyForCountriesTheProjectMade(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := NewOMap()
	project.Set("export_languages", []Value{"en"})

	country := func(tag string, autoAdded bool) *OMap {
		row := NewOMap()
		row.Set("tag", tag)
		row.Set("ideology", "neutrality")
		row.Set("auto_added", autoAdded)
		return row
	}
	content := NewOMap()
	for _, tag := range []string{"GER", "KRX", "ZQX"} {
		content.Set(tag, true)
	}
	scenario := &Scenario{
		Project:       project,
		Tag:           "ZQX",
		ModName:       "Ruling Party Test",
		ModDirName:    "ruling_party_test",
		ExistenceSets: map[string]map[string]bool{"idea": {}, "focus": {}},
		ContentByTag:  content,

		Countries: []*OMap{country("GER", false), country("KRX", true), country("ZQX", false)},
	}
	records, _ := EmitScenario(tables, scenario)
	history := map[string]string{}
	for _, record := range records {
		if rest, ok := strings.CutPrefix(record.Path, "ruling_party_test/history/countries/"); ok {
			history[strings.SplitN(rest, " ", 2)[0]] = record.Text
		}
	}
	for _, tag := range []string{"GER", "KRX"} {
		if strings.Contains(history[tag], "ruling_party") {
			t.Fatalf("%s got a ruling party from its row:\n%s", tag, history[tag])
		}
	}
	if !strings.Contains(history["ZQX"], "ruling_party = neutrality") {
		t.Fatalf("the project's own country lost its ruling party: %q", history["ZQX"])
	}
}
