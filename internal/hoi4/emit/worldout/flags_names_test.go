package worldout

import (
	"strings"
	"testing"
)

func TestCountryNamesIncludeProjectManagedCustomRoot(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}

	project := NewOMap()
	project.Set("export_languages", []Value{"en"})
	project.Set("ideology_registry", managedRegistry())

	customNames := NewOMap()
	customNameLocales := NewOMap()
	customNameLocales.Set("en", "Technocratic Commonwealth")
	customNames.Set("technocracy", customNameLocales)

	country := NewOMap()
	country.Set("tag", "TCH")
	country.Set("name_en", "Test Commonwealth")
	country.Set("adjective_en", "Commonwealth")
	country.Set("ideology_name_locs", customNames)

	records, _ := EmitScenario(tables, &Scenario{
		Project:       project,
		Tag:           "TCH",
		ModName:       "Dynamic root country names",
		ModDirName:    "dynamic_root_names",
		Countries:     []*OMap{country},
		ExistenceSets: map[string]map[string]bool{"idea": {}, "focus": {}},
	})

	body := ""
	for _, record := range records {
		if strings.HasSuffix(record.Path, "/TCH_country_l_english.yml") {
			body = record.Text
			break
		}
	}
	if body == "" {
		t.Fatalf("country localisation record not emitted: %+v", records)
	}

	for _, want := range []string{
		`TCH_technocracy: "Technocratic Commonwealth"`,
		`TCH_technocracy_DEF: "Technocratic Commonwealth"`,
		`TCH_technocracy_ADJ: "Commonwealth"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dynamic ROOT country localisation missing %q:\n%s", want, body)
		}
	}
}
