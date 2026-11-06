package importjob

import (
	"context"
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

// A mod that changes the base game's countries without declaring them in its
// own country_tags still lists their colors in common/countries/colors.txt.
// Those tags are countries, not cosmetic tags.
func TestColorsForBaseGameCountriesAreNotCosmeticTags(t *testing.T) {
	root := t.TempDir()
	colors := "FDS = { color = rgb { 1 1 1 } }\n" +
		"GRE = { color = rgb { 2 2 2 } }\n" +
		"ENG = { color = rgb { 3 3 3 } }\n" +
		"GRE_neutrality = { color = rgb { 4 4 4 } }\n"
	for rel, body := range map[string]string{
		"common/country_tags/00_countries.txt": "FDS = \"countries/Fds.txt\"\n",
		"common/countries/Fds.txt":             "color = { 1 1 1 }\n",
		"common/countries/colors.txt":          colors,
		"history/countries/GRE - Greece.txt":   "capital = 47\n",
	} {
		countryPortWrite(t, root, rel, []byte(body))
	}
	p := countryPortProject(t)
	baseGame := countryAliasData{tags: map[string]bool{"ENG": true}}

	s := cosmeticColorsImport(t, p, root, baseGame)
	if len(s.cosmetic.colorTags) != 1 || s.cosmetic.colorTags[0] != "GRE_neutrality" {
		t.Fatalf("staged cosmetic tags = %v, want only GRE_neutrality", s.cosmetic.colorTags)
	}
	for _, tag := range []string{"FDS", "GRE", "ENG"} {
		if len(countryPortFind(t, p, "project_cosmetic_tags", "tag", tag)) != 0 {
			t.Fatalf("country %s became a cosmetic tag", tag)
		}
	}
	countryPortOne(t, p, "project_cosmetic_tags", "tag", "GRE_neutrality")
	row := countryPortOne(t, p, "project_raw_files", "dest_path", "common/countries/colors.txt")
	if row["note"] != countryColorsRawNote || row["source"] != "imported" || row["content"] != colors {
		t.Fatalf("colors.txt raw row: %v", row)
	}

	// Once the project defines every country the file colors, the colors
	// travel with the countries and the stale raw row goes.
	countryPortWrite(t, root, "common/countries/colors.txt", []byte("FDS = { color = rgb { 1 1 1 } }\nGRE_neutrality = { color = rgb { 4 4 4 } }\n"))
	cosmeticColorsImport(t, p, root, baseGame)
	if rows := countryPortFind(t, p, "project_raw_files", "dest_path", "common/countries/colors.txt"); len(rows) != 0 {
		t.Fatalf("stale colors.txt raw row kept: %v", rows)
	}
}

func cosmeticColorsImport(t *testing.T, p *project.Project, root string, baseGame countryAliasData) *session {
	t.Helper()
	s := newSession(context.Background(), root, AllOptions(), p.Record())
	s.facts = &gameFacts{aliases: baseGame}
	if _, err := s.stageCountries(CountryStageOptions{ImportCountries: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.stageCosmeticTags(); err != nil {
		t.Fatal(err)
	}
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		if err := s.mergeCountries(tx, CountryMergeOptions{ConflictPolicy: "overwrite"}); err != nil {
			return err
		}
		return s.mergeCosmeticTags(tx)
	}); err != nil {
		t.Fatal(err)
	}
	return s
}
