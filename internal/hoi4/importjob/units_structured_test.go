package importjob

import (
	"os"
	"path/filepath"
	"testing"

	unitsparser "haganeworks/internal/hoi4/importsvc/units"
)

func TestUnitFilesIncludesOnlyUnitsTrees(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"common/units/a.txt", "history/units/GER.txt", "history/countries/GER.txt"} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := unitFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestUnitPayloadCountsHierarchy(t *testing.T) {
	wire := unitsparser.AirNavalOOBJSON(`air_wings = { 1 = { fighter = { owner = GER amount = 10 } name = "A" } } units = { fleet = { name = "F" task_force = { name = "T" location = 1 ship = { name = "S" definition = destroyer } } } }`)
	p := map[string]any{"groups": wire["groups"], "fleets": wire["fleets"]}
	var c UnitCounts
	countsFromUnitPayload(&c, p)
	if c.AirWingGroups != 1 || c.AirWings != 1 || c.Fleets != 1 || c.TaskForces != 1 || c.Ships != 1 {
		t.Fatalf("counts %#v", c)
	}
}
