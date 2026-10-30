package importjob

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

func countryPortWrite(t *testing.T, root, rel string, body []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func flagTestTGA(r, g, b byte) []byte {
	const width, height = 82, 52
	out := make([]byte, 18, 18+width*height*3)
	out[2] = 2
	binary.LittleEndian.PutUint16(out[12:], width)
	binary.LittleEndian.PutUint16(out[14:], height)
	out[16], out[17] = 24, 0x20
	for i := 0; i < width*height; i++ {
		out = append(out, b, g, r)
	}
	return out
}

func countryPortJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func countryPortRows(t *testing.T, p *project.Project, table string) []project.Row {
	t.Helper()
	rows, err := p.Rows(table)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func countryPortFind(t *testing.T, p *project.Project, table, column string, value any) []project.Row {
	t.Helper()
	var out []project.Row
	for _, row := range countryPortRows(t, p, table) {
		if row[column] == value {
			out = append(out, row)
		}
	}
	return out
}

func countryPortOne(t *testing.T, p *project.Project, table, column string, value any) project.Row {
	t.Helper()
	rows := countryPortFind(t, p, table, column, value)
	if len(rows) != 1 {
		t.Fatalf("%s.%s=%v: %d rows", table, column, value, len(rows))
	}
	return rows[0]
}

func countryPortMod(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range map[string]string{
		"common/country_tags/00_countries.txt": "TST = \"countries/Test.txt\"\nABC = \"countries/Abc.txt\"\ndynamic_tags = yes\nD01 = \"countries/D01.txt\"\n",
		"common/country_tags/01_more.txt":      "TST = \"countries/Other.txt\"\n",
		"common/countries/Test.txt":            "graphical_culture = western_european_gfx\ngraphical_culture_2d = western_european_2d\ncolor = { 10 20 30 }\nuse_legacy_ai_pp_spend = yes\n",
		"common/countries/bad.txt":             "Broken_tag = { color = { 1 2 } }\n",
		"common/countries/colors.txt":          "ABC = { color = rgb { 1 2 3 } }\nTST_kingdom = { color = rgb { 50 60 70 } color_ui = rgb { 80 90 100 } }\nTST_dup = { color = { 5 5 5 } }\n",
		"common/countries/zz_conflict.txt":     "TST_dup = { color = { 6 6 6 } }\n",
		"common/factions/test_faction.txt":     "faction = { id = test }\n",
		"localisation/english/test_l_english.yml": "\ufeffl_english:\n TST:0 \"Testland\"\n TST_ADJ:0 \"Testish\"\n TST_fascism:0 \"Fascist Testland\"\n" +
			" TST_kingdom:0 \"Kingdom of Test\"\n TST_kingdom_ADJ:0 \"\"\n TST_history_DEF:0 \"the History\"\n",
		"localisation/simp_chinese/test_l_simp_chinese.yml": "\ufeffl_simp_chinese:\n TST:0 \"测试国\"\n",
		"gfx/flags/TST_communism.png":                       "not a picture",
		"gfx/flags/XXX.png":                                 "no such country",
	} {
		countryPortWrite(t, root, rel, []byte(body))
	}
	countryPortWrite(t, root, "gfx/flags/TST.tga", flagTestTGA(200, 30, 30))
	countryPortWrite(t, root, "gfx/flags/TST_fascism.tga", flagTestTGA(30, 30, 30))
	countryPortWrite(t, root, "gfx/flags/TST_old.tga", flagTestTGA(1, 2, 3))
	countryPortWrite(t, root, "gfx/flags/ABC.tga", flagTestTGA(9, 9, 9))
	return root
}

func countryPortProject(t *testing.T) *project.Project {
	t.Helper()
	p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func countryPortImport(t *testing.T, p *project.Project, root, policy string, beforeCosmetic func(*project.Tx)) (*session, CountryCounts, int, []string) {
	t.Helper()
	options := AllOptions()
	options.ConflictPolicy = policy
	s := newSession(context.Background(), root, options, p.Record())
	if _, err := s.stageLocalisation(nil); err != nil {
		t.Fatal(err)
	}
	counts, err := s.stageCountries(CountryStageOptions{ImportCountries: true, ImportFactions: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.stageCosmeticTags(); err != nil {
		t.Fatal(err)
	}
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		if err := s.mergeCountries(tx, CountryMergeOptions{ConflictPolicy: policy}); err != nil {
			return err
		}
		if err := s.mergeLocalisation(tx); err != nil {
			return err
		}
		if beforeCosmetic != nil {
			beforeCosmetic(tx)
		}
		return s.mergeCosmeticTags(tx)
	}); err != nil {
		t.Fatal(err)
	}
	var kept int
	var notes []string
	blobs := p.Blobs()
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		var err error
		kept, notes, err = s.persistImportedCountryFlags(tx, blobs)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return s, counts, kept, notes
}

func TestCountryCosmeticFlagImportPort(t *testing.T) {
	root := countryPortMod(t)
	p := countryPortProject(t)

	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		tx.Insert("countries", project.Row{
			"tag": "ABC", "name": "Author ABC", "color_ui": []any{9, 9, 9}, "flag_image_path": "flags/ABC.tga",
			"display_name_locs": map[string]any{}, "adjective_locs": map[string]any{}, "ideology_name_locs": map[string]any{},
		})
		tx.Insert("project_raw_files", project.Row{"dest_path": "common/countries/zz_conflict.txt", "content": "author", "generate_replace_path": false, "note": "mine", "is_binary": false, "source": "created"})
		tx.Insert("project_raw_files", project.Row{"dest_path": "common/countries/old.txt", "content": "old", "generate_replace_path": false, "note": "Cosmetic colors retained as raw: stale", "is_binary": false, "source": "imported"})
		tag := tx.Insert("project_cosmetic_tags", project.Row{"tag": "TST_kingdom", "source": "created", "color": []any{1, 1, 1}, "revision": 1})
		tx.Insert("project_cosmetic_tag_names", project.Row{"cosmetic_tag_id": tag["id"], "locale": "l_english", "suffix": "", "value": "Author Kingdom", "source": "created"})
		tx.Insert("project_cosmetic_tag_names", project.Row{"cosmetic_tag_id": tag["id"], "locale": "l_english", "suffix": "DEF", "value": "old def", "source": "imported"})
		tx.Insert("history_files", project.Row{"country_tag": "TST", "set_cosmetic_tag": "TST_history", "set_popularities": map[string]any{}, "date_blocks": []any{}, "source": "imported"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	s, counts, kept, notes := countryPortImport(t, p, root, "overwrite", nil)
	if counts.Countries != 2 || counts.Factions != 1 || counts.DynamicSkipped != 1 || len(counts.Warnings) != 0 {
		t.Fatalf("counts=%+v", counts)
	}

	tst := countryPortOne(t, p, "countries", "tag", "TST")
	for column, want := range map[string]string{
		"name": `"Testland"`, "name_en": `"Testland"`, "name_zh": `"测试国"`, "adjective_en": `"Testish"`, "adjective_zh": `""`,
		"color": `[10,20,30]`, "color_ui": `[]`, "ideology": `"neutrality"`,
		"graphical_culture": `"western_european_gfx"`, "graphical_culture_2d": `"western_european_2d"`, "use_legacy_ai_pp_spend": `true`,
		"display_name_locs": `{"en":"Testland","zh":"测试国"}`, "adjective_locs": `{"en":"Testish"}`,
		"ideology_name_locs": `{"communism":{},"democratic":{},"fascism":{"en":"Fascist Testland"},"neutrality":{}}`,
		"name_fascism_en":    `"Fascist Testland"`, "name_fascism_zh": `null`, "name_democratic_en": `null`,
		"flag_image_path": `"flags/TST.tga"`, "flag_paths_by_ideology": `{"fascism":"flags/TST_fascism.tga"}`,
		"has_obs": `true`, "auto_added": `true`,
	} {
		if got := countryPortJSON(t, tst[column]); got != want {
			t.Errorf("TST.%s = %s, want %s", column, got, want)
		}
	}
	abc := countryPortOne(t, p, "countries", "tag", "ABC")
	for column, want := range map[string]string{
		"name": `"ABC"`, "name_en": `"ABC"`, "color": `[1,2,3]`, "graphical_culture": `""`, "use_legacy_ai_pp_spend": `null`,
		"color_ui": `[9,9,9]`, "flag_image_path": `"flags/ABC.tga"`, "auto_added": `false`, "has_obs": `false`,
		"ideology_name_locs": `{"communism":{},"democratic":{},"fascism":{},"neutrality":{}}`,
	} {
		if got := countryPortJSON(t, abc[column]); got != want {
			t.Errorf("ABC.%s = %s, want %s", column, got, want)
		}
	}
	if rows := countryPortFind(t, p, "countries", "tag", "D01"); len(rows) != 0 {
		t.Fatal("a dynamic tag became a country")
	}
	record := p.Record()
	if got := countryPortJSON(t, record["country_tags"]); got != `["ABC","TST"]` || record["country_tag"] != "TST" || record["country_name"] != "Testland" {
		t.Fatalf("record: tags=%s tag=%v name=%v", got, record["country_tag"], record["country_name"])
	}

	faction := countryPortOne(t, p, "project_raw_files", "dest_path", "common/factions/test_faction.txt")
	if faction["content"] != "faction = { id = test }\n" || faction["note"] != countryFactionNote || faction["source"] != "imported" ||
		faction["is_binary"] != false || faction["generate_replace_path"] != false || faction["raw_kind"] != "verbatim_override" {
		t.Fatalf("faction raw row: %v", faction)
	}
	for _, dest := range []string{"common/countries/bad.txt", "common/countries/colors.txt"} {
		row := countryPortOne(t, p, "project_raw_files", "dest_path", dest)
		if row["note"] != cosmeticRawNote || row["source"] != "imported" || !strings.Contains(row["content"].(string), "color") {
			t.Fatalf("%s raw row: %v", dest, row)
		}
	}
	if author := countryPortOne(t, p, "project_raw_files", "dest_path", "common/countries/zz_conflict.txt"); author["content"] != "author" || author["note"] != "mine" {
		t.Fatalf("the author's raw file changed: %v", author)
	}
	if len(countryPortFind(t, p, "project_raw_files", "dest_path", "common/countries/old.txt")) != 0 {
		t.Fatal("the stale imported cosmetic raw file was kept")
	}
	warnings := strings.Join(s.sortedWarnings(), "\n")
	if !strings.Contains(warnings, "Cosmetic colors preserved as raw in common/countries/bad.txt") ||
		!strings.Contains(warnings, "Conflicting cosmetic color definitions for TST_dup") {
		t.Fatalf("warnings: %s", warnings)
	}

	kingdom := countryPortOne(t, p, "project_cosmetic_tags", "tag", "TST_kingdom")
	if countryPortJSON(t, kingdom["color"]) != `[1,1,1]` || countryPortJSON(t, kingdom["imported_color"]) != `[50,60,70]` ||
		countryPortJSON(t, kingdom["imported_color_ui"]) != `[80,90,100]` || kingdom["source"] != "created" || project.Int64(kingdom["revision"]) != 3 {
		t.Fatalf("TST_kingdom: %v", kingdom)
	}
	history := countryPortOne(t, p, "project_cosmetic_tags", "tag", "TST_history")
	if history["source"] != "imported" || history["imported_color"] != nil || project.Int64(history["revision"]) != 2 {
		t.Fatalf("TST_history: %v", history)
	}
	for _, tag := range []string{"ABC", "TST_dup", "Broken_tag"} {
		if len(countryPortFind(t, p, "project_cosmetic_tags", "tag", tag)) != 0 {
			t.Fatalf("%s should not be a cosmetic tag", tag)
		}
	}
	names := map[string]string{}
	for _, row := range countryPortRows(t, p, "project_cosmetic_tag_names") {
		owner := "TST_kingdom"
		if project.Int64(row["cosmetic_tag_id"]) == project.Int64(history["id"]) {
			owner = "TST_history"
		}
		names[owner+"/"+row["locale"].(string)+"/"+row["suffix"].(string)+"/"+row["source"].(string)] = row["value"].(string)
	}
	wantNames := map[string]string{
		"TST_kingdom/l_english//created":     "Author Kingdom",
		"TST_kingdom/l_english//imported":    "Kingdom of Test",
		"TST_kingdom/l_english/ADJ/imported": "",
		"TST_history/l_english/DEF/imported": "the History",
	}
	if countryPortJSON(t, names) != countryPortJSON(t, wantNames) {
		t.Fatalf("cosmetic names: %v", names)
	}

	if kept != 2 || len(notes) != 1 || !strings.Contains(notes[0], "gfx/flags/TST_communism.png was not kept") {
		t.Fatalf("flags kept=%d notes=%v", kept, notes)
	}
	tstID := project.Int64(tst["id"])
	base := countryPortFind(t, p, "entity_obs_paths", "entity_type", "country_flag")
	if len(base) != 1 || project.Int64(base[0]["entity_id"]) != tstID ||
		!importassets.IsEntityAssetKey("local", "u1", "flag", tstID, "country_flag", base[0]["obs_key"].(string)) {
		t.Fatalf("base flag rows: %v", base)
	}
	variant := countryPortOne(t, p, "project_obs_assets", "asset_type", "country_flag_ideology")
	if variant["gfx_name"] != "TST_fascism" || !importassets.IsNamedAssetKey("local", "u1", "flag", "TST_fascism", variant["obs_key"].(string)) ||
		countryPortJSON(t, variant["provenance"]) != `{}` {
		t.Fatalf("ideology flag row: %v", variant)
	}
	for _, key := range []string{base[0]["obs_key"].(string), variant["obs_key"].(string)} {
		path, err := p.Blobs().Path(key)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("stored flag %s: %v", key, err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil || img.Bounds().Dx() == 0 {
			t.Fatalf("stored flag %s is not a PNG: %v", key, err)
		}
	}

	_, _, kept, notes = countryPortImport(t, p, root, "overwrite", nil)
	if kept != 0 || len(notes) != 1 {
		t.Fatalf("re-import flags kept=%d notes=%v", kept, notes)
	}
	if n := len(countryPortRows(t, p, "countries")); n != 2 {
		t.Fatalf("countries after re-import: %d", n)
	}
	tst = countryPortOne(t, p, "countries", "tag", "TST")
	if tst["flag_image_path"] != "flags/TST.tga" || tst["has_obs"] != true || countryPortJSON(t, tst["flag_paths_by_ideology"]) != `{"fascism":"flags/TST_fascism.tga"}` {
		t.Fatalf("re-import touched TST's flags: %v", tst)
	}
	if len(countryPortFind(t, p, "entity_obs_paths", "entity_type", "country_flag")) != 1 ||
		len(countryPortFind(t, p, "project_obs_assets", "asset_type", "country_flag_ideology")) != 1 {
		t.Fatal("re-import duplicated flag rows")
	}
	if len(countryPortFind(t, p, "project_raw_files", "dest_path", "common/factions/test_faction.txt")) != 1 {
		t.Fatal("re-import duplicated the faction")
	}
	if revision := project.Int64(countryPortOne(t, p, "project_cosmetic_tags", "tag", "TST_kingdom")["revision"]); revision != 3 {
		t.Fatalf("an unchanged re-import bumped TST_kingdom's revision to %d", revision)
	}
	if revision := project.Int64(countryPortOne(t, p, "project_cosmetic_tags", "tag", "TST_history")["revision"]); revision != 2 {
		t.Fatalf("an unchanged re-import bumped TST_history's revision to %d", revision)
	}
	if n := len(countryPortRows(t, p, "project_cosmetic_tag_names")); n != 4 {
		t.Fatalf("cosmetic names after re-import: %d", n)
	}
}

func TestCountryMergeConflictPolicies(t *testing.T) {
	for _, policy := range []string{"skip", "overwrite"} {
		t.Run(policy, func(t *testing.T) {
			root := t.TempDir()
			countryPortWrite(t, root, "common/country_tags/00.txt", []byte("TST = \"countries/Test.txt\"\nABC = \"countries/Abc.txt\"\n"))
			countryPortWrite(t, root, "common/countries/Abc.txt", []byte("color = { 4 5 6 }\n"))
			countryPortWrite(t, root, "common/factions/a.txt", []byte("new a"))
			countryPortWrite(t, root, "common/factions/b.txt", []byte("new b"))
			p := countryPortProject(t)
			if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
				tx.Insert("countries", project.Row{"tag": "ABC", "name": "Author ABC", "color": []any{7, 7, 7}, "display_name_locs": map[string]any{}, "adjective_locs": map[string]any{}, "ideology_name_locs": map[string]any{}})
				tx.Insert("project_raw_files", project.Row{"dest_path": "common/factions/a.txt", "content": "author a", "generate_replace_path": false, "note": "mine", "is_binary": false, "source": "created"})
				tx.Insert("project_raw_files", project.Row{"dest_path": "common/factions/b.txt", "content": "old b", "generate_replace_path": false, "note": "old", "is_binary": false, "source": "imported"})
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			countryPortImport(t, p, root, policy, nil)
			abc := countryPortOne(t, p, "countries", "tag", "ABC")
			b := countryPortOne(t, p, "project_raw_files", "dest_path", "common/factions/b.txt")
			if policy == "skip" {
				if abc["name"] != "Author ABC" || countryPortJSON(t, abc["color"]) != `[7,7,7]` || b["content"] != "old b" {
					t.Fatalf("skip changed existing rows: %v %v", abc, b)
				}
			} else if abc["name"] != "ABC" || countryPortJSON(t, abc["color"]) != `[4,5,6]` || b["content"] != "new b" || b["note"] != countryFactionNote {
				t.Fatalf("overwrite did not update imported rows: %v %v", abc, b)
			}
			if a := countryPortOne(t, p, "project_raw_files", "dest_path", "common/factions/a.txt"); a["content"] != "author a" {
				t.Fatalf("the author's faction file was replaced: %v", a)
			}
			if tst := countryPortOne(t, p, "countries", "tag", "TST"); tst["name"] != "TST" || countryPortJSON(t, tst["color"]) != `[128,128,128]` {
				t.Fatalf("new country without metadata: %v", tst)
			}

			if record := p.Record(); countryPortJSON(t, record["country_tags"]) != `["ABC","TST"]` || record["country_tag"] != "TST" || record["country_name"] != "TST" {
				t.Fatalf("record: %v", record)
			}
		})
	}
}

func TestCountryMergeAdoptsFirstCountryTag(t *testing.T) {
	root := t.TempDir()
	countryPortWrite(t, root, "common/country_tags/00.txt", []byte("NEW = \"countries/New.txt\"\n"))
	countryPortWrite(t, root, "localisation/english/x_l_english.yml", []byte("l_english:\n NEW:0 \"Newland\"\n"))
	p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "uuid": "u2", "country_tag": "OLD", "country_tags": []any{"OLD"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	countryPortImport(t, p, root, "overwrite", nil)
	if record := p.Record(); countryPortJSON(t, record["country_tags"]) != `["NEW"]` || record["country_tag"] != "NEW" || record["country_name"] != "Newland" {
		t.Fatalf("record: %v", record)
	}
}

func TestStagedCountryFlagsTakesTheLargeFlagsOnly(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"gfx/flags/HJM.png", "gfx/flags/HJM.tga",
		"gfx/flags/hjm_feudalist.tga",
		"gfx/flags/medium/HJM.tga",
		"gfx/flags/small/HJM.tga",
		"gfx/flags/AB.tga",
		"gfx/flags/readme.txt",
		"gfx/flags/GER_old.dds",
	} {
		countryPortWrite(t, root, rel, []byte("x"))
	}
	got, err := stagedCountryFlags(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got["HJM"][""]) != "HJM.tga" || filepath.Base(got["HJM"]["feudalist"]) != "hjm_feudalist.tga" {
		t.Fatalf("HJM flags: %v", got["HJM"])
	}
	if len(got) != 2 || len(got["HJM"]) != 2 || filepath.Base(got["GER"]["old"]) != "GER_old.dds" {
		t.Fatalf("staged flags: %v", got)
	}
	none, err := stagedCountryFlags(t.TempDir())
	if err != nil || none != nil {
		t.Fatalf("a mod without gfx/flags: %v %v", none, err)
	}
}

func TestFlagIdeologiesFollowExport(t *testing.T) {
	known := flagIdeologies([]byte(`{"groups":["feudalist","Libertarian_Socialist"],"root_definitions":[{"id":"conservative"}],"sub_ideologies":[]}`))
	for _, name := range []string{"democratic", "fascism", "neutrality", "communism", "feudalist", "libertarian_socialist", "conservative"} {
		if !known[name] {
			t.Errorf("%s should be a flag ideology", name)
		}
	}
	if known["old"] || known[""] {
		t.Fatalf("unexpected: %v", known)
	}
	if len(flagIdeologies([]byte(`{}`))) != 4 {
		t.Fatal("an empty registry still knows the base game's four")
	}
}

func TestImportedCountryFlagsFollowTheRegistry(t *testing.T) {
	root := t.TempDir()
	countryPortWrite(t, root, "common/country_tags/00.txt", []byte("HJM = \"countries/HJM.txt\"\n"))
	countryPortWrite(t, root, "gfx/flags/HJM_feudalist.tga", flagTestTGA(10, 20, 30))
	countryPortWrite(t, root, "gfx/flags/HJM_old.tga", flagTestTGA(10, 20, 30))
	p, err := project.NewStore(t.TempDir()).Create(project.Row{
		"name": "t", "uuid": "u3", "country_tag": "HJM", "country_tags": []any{"HJM"},
		"ideology_registry": map[string]any{"groups": []any{"feudalist"}, "root_definitions": []any{}},
	}, "t")
	if err != nil {
		t.Fatal(err)
	}
	_, _, kept, notes := countryPortImport(t, p, root, "overwrite", nil)
	hjm := countryPortOne(t, p, "countries", "tag", "HJM")
	if kept != 1 || len(notes) != 0 || countryPortJSON(t, hjm["flag_paths_by_ideology"]) != `{"feudalist":"flags/HJM_feudalist.tga"}` ||
		hjm["flag_image_path"] != "" || hjm["has_obs"] != false {
		t.Fatalf("kept=%d notes=%v HJM=%v", kept, notes, hjm)
	}
}
