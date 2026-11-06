package importjob

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

// A mod that changes a base-game country keeps the base game's country_tags
// line and replaces only common/countries/<Name>.txt. That file is kept.
func TestCountryDefinitionsNoTagLineNamesAreKept(t *testing.T) {
	root := t.TempDir()
	germany := "graphical_culture = western_european_gfx\ngraphical_culture_2d = western_european_2d\ncolor = { 1 2 3 }\n"
	for rel, body := range map[string]string{
		"common/country_tags/00_countries.txt": "FDS = \"countries/Fds.txt\"\n",
		"common/countries/Fds.txt":             "graphical_culture = western_european_gfx\ncolor = { 4 5 6 }\n",
		"common/countries/Germany.txt":         germany,
		"common/countries/colors.txt":          "GER = { color = rgb { 1 2 3 } }\n",
	} {
		countryPortWrite(t, root, rel, []byte(body))
	}
	p := countryPortProject(t)
	undeclaredImport(t, p, root)

	row := countryPortOne(t, p, "project_raw_files", "dest_path", "common/countries/Germany.txt")
	if row["content"] != germany || row["note"] != countryDefinitionNote || row["source"] != "imported" {
		t.Fatalf("Germany.txt raw row: %v", row)
	}
	for _, dest := range []string{"common/countries/Fds.txt", "common/countries/colors.txt"} {
		if rows := countryPortFind(t, p, "project_raw_files", "dest_path", dest); len(rows) != 0 {
			t.Fatalf("%s kept as a country definition: %v", dest, rows)
		}
	}

	// Gone from the mod, gone from the project.
	if err := os.Remove(filepath.Join(root, "common", "countries", "Germany.txt")); err != nil {
		t.Fatal(err)
	}
	undeclaredImport(t, p, root)
	if rows := countryPortFind(t, p, "project_raw_files", "dest_path", "common/countries/Germany.txt"); len(rows) != 0 {
		t.Fatalf("stale Germany.txt kept: %v", rows)
	}
}

func undeclaredImport(t *testing.T, p *project.Project, root string) {
	t.Helper()
	s := newSession(context.Background(), root, AllOptions(), p.Record())
	if _, err := s.stageCountries(CountryStageOptions{ImportCountries: true}); err != nil {
		t.Fatal(err)
	}
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		return s.mergeCountries(tx, CountryMergeOptions{ConflictPolicy: "overwrite"})
	}); err != nil {
		t.Fatal(err)
	}
}
