package guirender

import "testing"

func TestMapEditAccessors(t *testing.T) {
	idx, err := NewIndexWithFallback(map[string][]byte{
		"map/strategicregions/7-Sea.txt":  []byte("strategic_region={ id=7 name=\"STRATEGICREGION_7\" provinces={ 10 11 } }"),
		"map/strategicregions/9-Land.txt": []byte("strategic_region={ id=9 name=\"STRATEGICREGION_9\" provinces={ 1 2 } }"),
		"history/states/3-Land.txt":       []byte("state={ id=3 name=\"STATE_3\" provinces={ 1 2 } }"),
		"map/continent.txt":               []byte("continents = {\n\teurope\n\tasia\n}\n"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state, ok := idx.StateOfProvince(2); !ok || state != 3 {
		t.Errorf("state of 2: %d %t", state, ok)
	}
	if _, ok := idx.StateOfProvince(10); ok {
		t.Error("a sea province in a state")
	}
	if provinces, ok := idx.StateProvinces(3); !ok || len(provinces) != 2 || provinces[1] != 2 {
		t.Errorf("state 3: %v %t", provinces, ok)
	}
	if n := idx.ContinentCount(); n != 2 {
		t.Errorf("%d continents", n)
	}
	if file, ok := idx.StrategicRegionFile(7); !ok || file != "map/strategicregions/7-Sea.txt" {
		t.Errorf("region 7 file %q %t", file, ok)
	}
	if id, file, ok := idx.StrategicRegionOf(11); !ok || id != 7 || file != "map/strategicregions/7-Sea.txt" {
		t.Errorf("region of 11: %d %q %t", id, file, ok)
	}
	if most := idx.MaxStrategicRegionID(); most != 9 {
		t.Errorf("highest region %d", most)
	}
}

func TestCapitalsAndDivisionLocations(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/country_tags/00_countries.txt": []byte(`TAG = "countries/Tagland.txt"`),
		"history/countries/TAG - Tagland.txt":  []byte("capital = 64\nset_oob = \"TAG_1936\"\n1939.1.1 = { set_oob = \"TAG_1939\" }\n"),
		"history/units/TAG_1936.txt": []byte(`units = {
	division = { division_template = "A" location = 11 }
	division = { division_template = "A" location = 11 }
	division = { division_template = "A" location = 12 }
}`),
		"history/units/TAG_1939.txt":  []byte(`units = { division = { division_template = "A" location = 99 } }`),
		"history/units/TAG_other.txt": []byte(`units = { division = { division_template = "A" location = 98 } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if tags := idx.CapitalStates()[64]; len(tags) != 1 || tags[0] != "TAG" {
		t.Errorf("capital states %v", idx.CapitalStates())
	}
	at := idx.DivisionLocations("1936.1.1")
	if at[11] != 2 || at[12] != 1 || at[99] != 0 || at[98] != 0 {
		t.Errorf("1936 divisions %v", at)
	}
	if later := idx.DivisionLocations("1939.1.1"); later[99] != 1 {
		t.Errorf("1939 divisions %v", later)
	}
}
