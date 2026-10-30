package military

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type parityCase struct {
	Input      string           `json:"input"`
	SourceFile string           `json:"source_file"`
	Rows       []map[string]any `json:"rows"`
	Tail       string           `json:"tail"`
	Warnings   []string         `json:"warnings"`
}

func jsonNative(t *testing.T, value any) any {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPythonMilitaryImporterFixtureParity(t *testing.T) {
	raw, err := os.ReadFile("testdata/python_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]parityCase
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}

	equipment := ImportEquipment(fixture["equipment"].Input, fixture["equipment"].SourceFile)
	if got, want := jsonNative(t, equipment.Rows), jsonNative(t, fixture["equipment"].Rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("equipment rows drifted\n got: %#v\nwant: %#v", got, want)
	}
	if !reflect.DeepEqual(equipment.Warnings, fixture["equipment"].Warnings) {
		t.Fatalf("equipment warnings = %#v, want %#v", equipment.Warnings, fixture["equipment"].Warnings)
	}

	subUnits := ImportSubUnits(fixture["sub_units"].Input, fixture["sub_units"].SourceFile)
	if got, want := jsonNative(t, subUnits.Rows), jsonNative(t, fixture["sub_units"].Rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("sub-unit rows drifted\n got: %#v\nwant: %#v", got, want)
	}
	if !reflect.DeepEqual(subUnits.Warnings, fixture["sub_units"].Warnings) {
		t.Fatalf("sub-unit warnings = %#v, want %#v", subUnits.Warnings, fixture["sub_units"].Warnings)
	}

	divisions := ImportDivisionTemplates(fixture["divisions"].Input)
	if got, want := jsonNative(t, divisions.Rows), jsonNative(t, fixture["divisions"].Rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("division rows drifted\n got: %#v\nwant: %#v", got, want)
	}
	if divisions.Tail != fixture["divisions"].Tail || !reflect.DeepEqual(divisions.Warnings, fixture["divisions"].Warnings) {
		t.Fatalf("division tail/warnings drifted\n tail=%q\n want=%q\n warnings=%#v want=%#v",
			divisions.Tail, fixture["divisions"].Tail, divisions.Warnings, fixture["divisions"].Warnings)
	}
}

func TestMilitaryImporterEmptyAndParseWarnings(t *testing.T) {
	if got := ImportEquipment("ideas = {}", ""); !reflect.DeepEqual(got.Warnings, []string{"no `equipments = { ... }` wrapper found"}) || len(got.Rows) != 0 {
		t.Fatalf("equipment missing wrapper = %#v", got)
	}
	if got := ImportSubUnits("ideas = {}", ""); !reflect.DeepEqual(got.Warnings, []string{"no `sub_units = { ... }` wrapper found"}) || len(got.Rows) != 0 {
		t.Fatalf("sub-unit missing wrapper = %#v", got)
	}
	if got := ImportDivisionTemplates("units = {}"); !reflect.DeepEqual(got.Warnings, []string{"no `division_template = { ... }` block found"}) || got.Tail != "units = {}" {
		t.Fatalf("division missing template = %#v", got)
	}
}
