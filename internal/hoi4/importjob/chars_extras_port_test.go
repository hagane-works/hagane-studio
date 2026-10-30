package importjob

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

var charsExtrasPortMod = map[string]string{
	"localisation/english/test_l_english.yml": "l_english:\n" +
		" TST_leader_name:0 \"Leader of Test\"\n" +
		" ABC_shared:0 \"Shared Hero\"\n" +
		" tst_trait:0 \"Test Trait\"\n" +
		" tst_trait_desc:0 \"Trait description\"\n" +
		" TST_idea:0 \"Test Idea\"\n" +
		" TST_idea_desc:0 \"Idea desc\"\n" +
		" TST_mio:0 \"Test MIO\"\n" +
		" TST_mio_desc:0 \"MIO desc\"\n" +
		" TST_agency:0 \"Test Agency\"\n" +
		" specialization_test:0 \"Test Spec\"\n" +
		" specialization_test_desc:0 \"Spec desc\"\n" +
		" sp_test:0 \"Test Project\"\n" +
		" sp_test_desc:0 \"Project desc\"\n",
	"localisation/simp_chinese/test_l_simp_chinese.yml": "l_simp_chinese:\n" +
		" TST_leader_name:0 \"测试领袖\"\n" +
		" TST_idea:0 \"测试理念\"\n",
	"interface/test.gfx": `spriteTypes = {
	spriteType = { name = "GFX_portrait_TST_leader" texturefile = "gfx/leaders/TST/leader.dds" }
	spriteType = { name = "GFX_idea_tst" texturefile = "gfx/interface/ideas/idea_tst.dds" }
	spriteType = { name = "GFX_sp_test" texturefile = "gfx/interface/special_projects/sp_test.dds" }
	spriteType = { name = "GFX_doctrine_test" texturefile = "gfx/interface/doctrines/test.dds" }
}`,
	"common/dynamic_modifiers/dm.txt": `dm_test = { icon = GFX_x }`,
	"common/characters/TST.txt": `characters = {
	TST_leader = {
		name = TST_leader_name
		portraits = { civilian = { large = GFX_portrait_TST_leader small = "gfx/interface/ideas/small_leader.dds" } }
		country_leader = { ideology = fascism traits = { tst_trait } }
	}
	FOO_outsider = { name = FOO_outsider advisor = { slot = political_advisor idea_token = FOO_adv } }
	ABC_shared = { name = ABC_shared }
	generic_hero = { name = generic_hero }
}`,

	"common/characters/zz_override.txt": `characters = { generic_hero = { name = generic_hero_renamed } }`,
	"common/country_leader/traits.txt":  `leader_traits = { tst_trait = { random = no modifier = { political_power_gain = 0.1 } } }`,
	"common/ideas/TST.txt": `ideas = {
	country = {
		TST_idea = {
			allowed = { original_tag = TST }
			picture = GFX_idea_tst
			removal_cost = 25
			modifier = { stability_factor = 0.1 }
		}
		generic_idea = { modifier = { war_support_factor = 0.05 } }
		FOO_idea = { modifier = { stability_factor = 0.1 } }
	}
}`,
	"common/military_industrial_organization/organizations/TST.txt": `TST_mio = {
	allowed = { original_tag = TST }
	icon = GFX_idea_tst
	equipment_type = { infantry_equipment }
	research_categories = { infantry_weapons }
}
generic_mio = { icon = GFX_unknown_mio }`,
	"common/intelligence_agencies/agencies.txt": `intelligence_agency = {
	picture = GFX_intelligence_agency_logo_tst
	names = { "Secret Service" "TST_agency" }
	default = { tag = TST }
}`,
	"common/doctrines/grand_doctrines/test.txt": `mobile_test = { folder = land name = MOBILE_TEST icon = GFX_doctrine_test xp_cost = 100 xp_type = army }`,
	"common/doctrines/subdoctrines/test.txt":    `sub_test = { track = armor allow_in_multiple_tracks = yes xor = { other_sub } }`,
	"common/special_projects/specialization/test.txt": `specialization_test = {
	color = { 10 20 30 }
	blueprint_image = GFX_sp_test
	program_background = "gfx/interface/special_projects/bg_test.dds"
	facility_model = test_facility
}`,
	"common/special_projects/projects/test.txt": `sp_test = {
	specialization = specialization_test
	icon = GFX_sp_test
	complexity = sp_complexity_medium
	prototype_time = sp_time_cost_medium
}`,
}

func charsExtrasPortWriteMod(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func charsExtrasPortProject(t *testing.T, tags ...any) *project.Project {
	t.Helper()
	p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": tags}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func charsExtrasPortImport(t *testing.T, p *project.Project, root string, options Options) *session {
	t.Helper()
	s := newSession(context.Background(), root, options, p.Record())
	if _, err := s.stageLocalisation(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.stageGFX(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.stageSimpleEntities(nil); err != nil {
		t.Fatal(err)
	}
	stage := CharsExtrasStageOptions{ImportCharacters: options.Characters, ImportIdeas: options.Ideas, CountryTag: options.CountryTag}
	if _, err := s.stageCharsExtras(stage, nil); err != nil {
		t.Fatal(err)
	}
	s.stageCharsExtrasNeededImages()
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		return s.mergeCharsExtras(tx, CharsExtrasMergeOptions{ConflictPolicy: options.ConflictPolicy})
	}); err != nil {
		t.Fatal(err)
	}
	return s
}

func charsExtrasPortRows(t *testing.T, p *project.Project, table string) []project.Row {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(p.Dir(), "data", table+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var rows []project.Row
	if err := decoder.Decode(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func charsExtrasPortRow(t *testing.T, p *project.Project, table string, key ...string) project.Row {
	t.Helper()
	for _, row := range charsExtrasPortRows(t, p, table) {
		match := true
		for i := 0; i+1 < len(key); i += 2 {
			if row[key[i]] != key[i+1] {
				match = false
			}
		}
		if match {
			return row
		}
	}
	t.Fatalf("%s: no row %v", table, key)
	return nil
}

func charsExtrasPortCheck(t *testing.T, label string, row project.Row, want map[string]string) {
	t.Helper()
	columns := make([]string, 0, len(want))
	for column := range want {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	for _, column := range columns {
		value, ok := row[column]
		if !ok {
			t.Errorf("%s: column %s missing", label, column)
			continue
		}
		raw, _ := json.Marshal(value)
		if string(raw) != want[column] {
			t.Errorf("%s.%s = %s, want %s", label, column, raw, want[column])
		}
	}
}

func charsExtrasPortKeys(t *testing.T, p *project.Project, table, column string) string {
	t.Helper()
	var keys []string
	for _, row := range charsExtrasPortRows(t, p, table) {
		keys = append(keys, row[column].(string))
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func charsExtrasPortEdit(t *testing.T, p *project.Project, table string, key []string, change func(project.Row)) {
	t.Helper()
	id := project.Int64(charsExtrasPortRow(t, p, table, key...)["id"])
	if err := p.Write([]string{table}, func(tx *project.Tx) error {
		_, err := tx.Update(table, id, change)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCharsExtrasPortStageKeepsSimpleEntitiesAndLastWins(t *testing.T) {
	root := t.TempDir()
	charsExtrasPortWriteMod(t, root, charsExtrasPortMod)
	s := newSession(context.Background(), root, AllOptions(), nil)
	if _, err := s.stageSimpleEntities(nil); err != nil {
		t.Fatal(err)
	}
	staged, err := s.stageCharsExtras(CharsExtrasStageOptions{ImportCharacters: true, ImportIdeas: true}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if staged != 16 {
		t.Fatalf("staged %d rows", staged)
	}
	counts := s.entityCounts()
	want := map[string]int64{"dynamic_modifier": 1, "character": 4, "trait:country_leader": 1, "idea": 3, "mio": 2, "intel_agency": 1, "doctrine": 2, "specialization": 1, "special_project": 1}
	for kind, n := range want {
		if counts[kind] != n {
			t.Errorf("%s: %d rows, want %d (all: %v)", kind, counts[kind], n, counts)
		}
	}
	for _, e := range s.entities {
		if e.kind == "character" && e.naturalKey == "generic_hero" && e.sourceFile != "common/characters/zz_override.txt" {
			t.Fatalf("generic_hero from %s, want the later file", e.sourceFile)
		}
	}
}

func TestCharsExtrasPortStageFlagsAndOverride(t *testing.T) {
	root := t.TempDir()
	charsExtrasPortWriteMod(t, root, charsExtrasPortMod)
	s := newSession(context.Background(), root, AllOptions(), nil)
	if _, err := s.stageCharsExtras(CharsExtrasStageOptions{ImportCharacters: false, ImportIdeas: true, CountryTag: " xyz "}, nil); err != nil {
		t.Fatal(err)
	}
	for i := range s.entities {
		e := &s.entities[i]
		if e.kind == "character" || strings.HasPrefix(e.kind, "trait:") {
			t.Fatalf("character family staged without ImportCharacters: %s", e.kind)
		}
		if got := e.data().str("_country_tag_override"); got != "XYZ" {
			t.Fatalf("%s %s: override %q", e.kind, e.naturalKey, got)
		}
	}
	s = newSession(context.Background(), root, AllOptions(), nil)
	if _, err := s.stageCharsExtras(CharsExtrasStageOptions{ImportCharacters: true}, nil); err != nil {
		t.Fatal(err)
	}
	for i := range s.entities {
		e := &s.entities[i]
		if e.kind != "character" && !strings.HasPrefix(e.kind, "trait:") {
			t.Fatalf("idea family staged without ImportIdeas: %s", e.kind)
		}
		if _, ok := e.data().text("_country_tag_override"); ok {
			t.Fatal("override set without a TAG")
		}
	}
}

func TestCharsExtrasPortFirstImport(t *testing.T) {
	root := t.TempDir()
	charsExtrasPortWriteMod(t, root, charsExtrasPortMod)
	p := charsExtrasPortProject(t, "TST", "ABC")
	s := charsExtrasPortImport(t, p, root, AllOptions())

	if got := charsExtrasPortKeys(t, p, "characters", "tag"); got != "ABC_shared,TST_leader,generic_hero" {
		t.Fatalf("characters %s", got)
	}
	charsExtrasPortCheck(t, "TST_leader", charsExtrasPortRow(t, p, "characters", "tag", "TST_leader"), map[string]string{
		"country_tag": `"TST"`, "name": `"TST_leader_name"`, "localisation_key": `"TST_leader_name"`,
		"name_en": `"Leader of Test"`, "name_zh": `"测试领袖"`,
		"display_name_locs": `{"en":"Leader of Test","zh":"测试领袖"}`, "description_locs": `{}`,
		"is_replacement": `true`, "replaces_vanilla_id": `true`, "is_portrait_modified": `false`, "has_obs": `false`,
		"source": `"imported"`, "source_file": `"common/characters/TST.txt"`,
		"large_portrait": `"GFX_portrait_TST_leader"`, "small_portrait": `"gfx/interface/ideas/small_leader.dds"`,
		"description": `""`, "desc_en": `""`, "desc_zh": `""`,
		"skill": `1`, "advisor_cost": `150`, "advisor_removal_cost": `-1`, "operative_skill_level": `1`,
		"can_be_captured": `true`, "always_show_on_actions_tooltip": `null`, "advisor_slot": `""`,
		"operative_traits": `[]`, "operative_nationalities": `[]`, "scientist_data": `{}`,
	})
	charsExtrasPortCheck(t, "ABC_shared", charsExtrasPortRow(t, p, "characters", "tag", "ABC_shared"), map[string]string{
		"country_tag": `"ABC"`, "name_en": `"Shared Hero"`, "name_zh": `""`, "display_name_locs": `{"en":"Shared Hero"}`,
	})
	charsExtrasPortCheck(t, "generic_hero", charsExtrasPortRow(t, p, "characters", "tag", "generic_hero"), map[string]string{
		"country_tag": `"TST"`, "name": `"generic_hero_renamed"`, "localisation_key": `"generic_hero_renamed"`,
		"name_en": `"generic_hero_renamed"`, "display_name_locs": `{}`, "source_file": `"common/characters/zz_override.txt"`,
	})

	charsExtrasPortCheck(t, "tst_trait", charsExtrasPortRow(t, p, "custom_traits", "trait_type", "country_leader", "trait_id", "tst_trait"), map[string]string{
		"country_tag": `"TST"`, "name_en": `""`, "desc_en": `""`,
		"display_name_locs": `{"en":"Test Trait"}`, "description_locs": `{"en":"Trait description"}`, "random": `false`,
	})

	if got := charsExtrasPortKeys(t, p, "ideas", "idea_id"); got != "TST_idea,generic_idea" {
		t.Fatalf("ideas %s", got)
	}
	charsExtrasPortCheck(t, "TST_idea", charsExtrasPortRow(t, p, "ideas", "idea_id", "TST_idea"), map[string]string{
		"country_tag": `"TST"`, "scope_kind": `"own"`, "shared_country_tags": `null`,
		"name_en": `"Test Idea"`, "name_zh": `"测试理念"`, "desc_en": `"Idea desc"`, "desc_zh": `""`,
		"display_name_locs": `{"en":"Test Idea","zh":"测试理念"}`, "description_locs": `{"en":"Idea desc"}`,
		"idea_type": `"country"`, "picture": `"GFX_idea_tst"`, "removal_cost": `25`, "auto_attach_on_start": `true`,
		"cancel_if_invalid": `true`, "has_obs": `false`, "source": `"imported"`, "source_file": `"common/ideas/TST.txt"`,
	})
	charsExtrasPortCheck(t, "generic_idea", charsExtrasPortRow(t, p, "ideas", "idea_id", "generic_idea"), map[string]string{
		"country_tag": `""`, "scope_kind": `"global"`, "removal_cost": `-1`, "display_name_locs": `{}`,
	})

	charsExtrasPortCheck(t, "TST_mio", charsExtrasPortRow(t, p, "military_industrial_organizations", "mio_id", "TST_mio"), map[string]string{
		"country_tag": `"TST"`, "scope_kind": `"own"`, "name_en": `"Test MIO"`, "desc_en": `"MIO desc"`,
		"equipment_type": `["infantry_equipment"]`, "source": `"imported"`,
	})

	charsExtrasPortCheck(t, "generic_mio", charsExtrasPortRow(t, p, "military_industrial_organizations", "mio_id", "generic_mio"), map[string]string{
		"country_tag": `"TST"`, "scope_kind": `"own"`, "shared_country_tags": `null`, "name_en": `""`, "display_name_locs": `{}`,
	})

	charsExtrasPortCheck(t, "agency", charsExtrasPortRow(t, p, "intelligence_agencies", "agency_id", "secret_service"), map[string]string{
		"country_tag": `"TST"`, "scope_kind": `"own"`, "shared_country_tags": `null`,
		"name_en": `"Test Agency"`, "name_zh": `""`, "display_name_locs": `{"en":"Test Agency"}`, "description_locs": `{}`,
		"names": `["Secret Service","TST_agency"]`, "source": `"imported"`,
	})

	charsExtrasPortCheck(t, "mobile_test", charsExtrasPortRow(t, p, "doctrines", "entity_type", "grand_doctrine", "entity_key", "mobile_test"), map[string]string{
		"folder_key": `"land"`, "name": `"MOBILE_TEST"`, "icon": `"GFX_doctrine_test"`, "xp_cost": `100`, "xp_type": `"army"`,
		"xor": `null`, "allow_in_multiple_tracks": `null`, "source": `"imported"`, "source_file": `"grand_doctrines/test.txt"`,
		"display_name_locs": `{}`,
	})
	charsExtrasPortCheck(t, "sub_test", charsExtrasPortRow(t, p, "doctrines", "entity_type", "subdoctrine", "entity_key", "sub_test"), map[string]string{
		"track_key": `"armor"`, "allow_in_multiple_tracks": `true`, "xp_cost": `0`,
	})

	charsExtrasPortCheck(t, "specialization", charsExtrasPortRow(t, p, "special_project_specializations", "specialization_id", "specialization_test"), map[string]string{
		"country_tag": `"TST"`, "name_en": `"Test Spec"`, "desc_en": `"Spec desc"`, "color": `[10,20,30]`,
		"description_locs": `{"en":"Spec desc"}`, "facility_model": `"test_facility"`, "source": `"imported"`,
	})
	charsExtrasPortCheck(t, "sp_test", charsExtrasPortRow(t, p, "special_projects", "project_token", "sp_test"), map[string]string{
		"country_tag": `"TST"`, "specialization": `"specialization_test"`, "name_en": `"Test Project"`, "desc_en": `"Project desc"`,
		"icon": `"GFX_sp_test"`, "complexity": `"sp_complexity_medium"`, "prototype_time": `"sp_time_cost_medium"`,
		"project_tags": `[]`, "source": `"imported"`,
	})

	var images []string
	for path := range s.neededImages {
		images = append(images, path)
	}
	sort.Strings(images)
	want := "gfx/interface/ideas/idea_tst.dds,gfx/interface/ideas/small_leader.dds,gfx/interface/special_projects/bg_test.dds,gfx/interface/special_projects/sp_test.dds,gfx/leaders/TST/leader.dds"
	if got := strings.Join(images, ","); got != want {
		t.Fatalf("needed images\n got %s\nwant %s", got, want)
	}
}

func TestCharsExtrasPortReimportOverwriteAndSkip(t *testing.T) {
	root := t.TempDir()
	charsExtrasPortWriteMod(t, root, charsExtrasPortMod)
	p := charsExtrasPortProject(t, "TST", "ABC")
	charsExtrasPortImport(t, p, root, AllOptions())
	counts := map[string]int{}
	tables := []string{"characters", "custom_traits", "ideas", "military_industrial_organizations", "intelligence_agencies", "doctrines", "special_project_specializations", "special_projects"}
	for _, table := range tables {
		counts[table] = len(charsExtrasPortRows(t, p, table))
	}

	edit := func() {
		charsExtrasPortEdit(t, p, "characters", []string{"tag", "TST_leader"}, func(row project.Row) {
			row["name"], row["is_portrait_modified"], row["description_locs"] = "edited", true, map[string]any{"en": "kept"}
		})
		charsExtrasPortEdit(t, p, "ideas", []string{"idea_id", "TST_idea"}, func(row project.Row) {
			row["removal_cost"], row["has_obs"], row["auto_attach_on_start"] = 999, true, false
		})
		charsExtrasPortEdit(t, p, "intelligence_agencies", []string{"agency_id", "secret_service"}, func(row project.Row) {
			row["name_en"], row["scope_kind"], row["description_locs"] = "edited", "global", map[string]any{"en": "kept"}
		})
		charsExtrasPortEdit(t, p, "doctrines", []string{"entity_type", "grand_doctrine", "entity_key", "mobile_test"}, func(row project.Row) {
			row["name"], row["display_name_locs"] = "edited", map[string]any{"en": "kept"}
		})
		charsExtrasPortEdit(t, p, "special_projects", []string{"project_token", "sp_test"}, func(row project.Row) {
			row["name_en"], row["country_tag"] = "edited", "XXX"
		})
		charsExtrasPortEdit(t, p, "custom_traits", []string{"trait_type", "country_leader", "trait_id", "tst_trait"}, func(row project.Row) {
			row["name_en"] = "edited"
		})
	}

	edit()
	charsExtrasPortImport(t, p, root, AllOptions())
	for _, table := range tables {
		if n := len(charsExtrasPortRows(t, p, table)); n != counts[table] {
			t.Fatalf("%s: %d rows after re-import, want %d", table, n, counts[table])
		}
	}
	charsExtrasPortCheck(t, "TST_leader", charsExtrasPortRow(t, p, "characters", "tag", "TST_leader"), map[string]string{
		"name": `"TST_leader_name"`, "is_portrait_modified": `true`, "description_locs": `{"en":"kept"}`,
	})
	charsExtrasPortCheck(t, "TST_idea", charsExtrasPortRow(t, p, "ideas", "idea_id", "TST_idea"), map[string]string{
		"removal_cost": `25`, "has_obs": `true`, "auto_attach_on_start": `false`,
	})
	charsExtrasPortCheck(t, "agency", charsExtrasPortRow(t, p, "intelligence_agencies", "agency_id", "secret_service"), map[string]string{
		"name_en": `"Test Agency"`, "scope_kind": `"global"`, "description_locs": `{"en":"kept"}`,
	})
	charsExtrasPortCheck(t, "mobile_test", charsExtrasPortRow(t, p, "doctrines", "entity_type", "grand_doctrine", "entity_key", "mobile_test"), map[string]string{
		"name": `"MOBILE_TEST"`, "display_name_locs": `{"en":"kept"}`,
	})
	charsExtrasPortCheck(t, "sp_test", charsExtrasPortRow(t, p, "special_projects", "project_token", "sp_test"), map[string]string{
		"name_en": `"Test Project"`, "country_tag": `"XXX"`,
	})
	charsExtrasPortCheck(t, "tst_trait", charsExtrasPortRow(t, p, "custom_traits", "trait_type", "country_leader", "trait_id", "tst_trait"), map[string]string{
		"name_en": `""`,
	})

	edit()
	charsExtrasPortWriteMod(t, root, map[string]string{
		"common/ideas/extra.txt": `ideas = { country = { TST_new_idea = { removal_cost = 5 } } }`,
	})
	options := AllOptions()
	options.ConflictPolicy = "skip"
	charsExtrasPortImport(t, p, root, options)
	charsExtrasPortCheck(t, "TST_leader", charsExtrasPortRow(t, p, "characters", "tag", "TST_leader"), map[string]string{"name": `"edited"`})
	charsExtrasPortCheck(t, "TST_idea", charsExtrasPortRow(t, p, "ideas", "idea_id", "TST_idea"), map[string]string{"removal_cost": `999`})
	charsExtrasPortCheck(t, "agency", charsExtrasPortRow(t, p, "intelligence_agencies", "agency_id", "secret_service"), map[string]string{"name_en": `"edited"`})
	charsExtrasPortCheck(t, "mobile_test", charsExtrasPortRow(t, p, "doctrines", "entity_type", "grand_doctrine", "entity_key", "mobile_test"), map[string]string{"name": `"edited"`})
	charsExtrasPortCheck(t, "sp_test", charsExtrasPortRow(t, p, "special_projects", "project_token", "sp_test"), map[string]string{"name_en": `"edited"`})
	charsExtrasPortCheck(t, "tst_trait", charsExtrasPortRow(t, p, "custom_traits", "trait_type", "country_leader", "trait_id", "tst_trait"), map[string]string{"name_en": `"edited"`})
	charsExtrasPortCheck(t, "TST_new_idea", charsExtrasPortRow(t, p, "ideas", "idea_id", "TST_new_idea"), map[string]string{
		"removal_cost": `5`, "country_tag": `""`, "source": `"imported"`,
	})
	if got := charsExtrasPortKeys(t, p, "ideas", "idea_id"); got != "TST_idea,TST_new_idea,generic_idea" {
		t.Fatalf("ideas after skip import: %s", got)
	}
}

func TestCharsExtrasPortCountryTagOverride(t *testing.T) {
	root := t.TempDir()
	charsExtrasPortWriteMod(t, root, map[string]string{
		"common/characters/c.txt": `characters = {
	XYZ_boss = { name = XYZ_boss }
	FOO_x = { name = FOO_x }
	TST_z = { name = TST_z }
	generic_y = { name = generic_y }
}`,
		"common/ideas/i.txt":                            `ideas = { country = { XYZ_idea = { removal_cost = 1 } FOO_idea = { removal_cost = 1 } } }`,
		"common/country_leader/t.txt":                   `leader_traits = { some_trait = { random = no } }`,
		"common/military_industrial_organization/m.txt": `generic_mio = { icon = GFX_m }`,
		"common/intelligence_agencies/a.txt":            `intelligence_agency = { names = { "Agency" } }`,
		"common/special_projects/specialization/s.txt":  `spec_x = { facility_model = f }`,
		"common/special_projects/projects/p.txt":        `sp_x = { specialization = spec_x }`,
		"common/doctrines/folders/f.txt":                `land_folder = { name = LAND }`,
		"localisation/english/x_l_english.yml":          "l_english:\n Agency:0 \"The Agency\"\n",
	})
	p := charsExtrasPortProject(t, "TST")
	options := AllOptions()
	options.CountryTag = "XYZ"
	charsExtrasPortImport(t, p, root, options)

	if got := charsExtrasPortKeys(t, p, "characters", "tag"); got != "TST_z,XYZ_boss,generic_y" {
		t.Fatalf("characters %s", got)
	}
	for tag, want := range map[string]string{"XYZ_boss": `"XYZ"`, "TST_z": `"TST"`, "generic_y": `"XYZ"`} {
		charsExtrasPortCheck(t, tag, charsExtrasPortRow(t, p, "characters", "tag", tag), map[string]string{"country_tag": want})
	}
	if got := charsExtrasPortKeys(t, p, "ideas", "idea_id"); got != "XYZ_idea" {
		t.Fatalf("ideas %s", got)
	}
	charsExtrasPortCheck(t, "trait", charsExtrasPortRow(t, p, "custom_traits", "trait_id", "some_trait"), map[string]string{"country_tag": `"XYZ"`})
	charsExtrasPortCheck(t, "mio", charsExtrasPortRow(t, p, "military_industrial_organizations", "mio_id", "generic_mio"), map[string]string{"country_tag": `"XYZ"`})
	charsExtrasPortCheck(t, "agency", charsExtrasPortRow(t, p, "intelligence_agencies", "agency_id", "agency"), map[string]string{
		"country_tag": `"XYZ"`, "name_en": `"The Agency"`,
	})
	charsExtrasPortCheck(t, "spec", charsExtrasPortRow(t, p, "special_project_specializations", "specialization_id", "spec_x"), map[string]string{"country_tag": `"XYZ"`})
	charsExtrasPortCheck(t, "sp", charsExtrasPortRow(t, p, "special_projects", "project_token", "sp_x"), map[string]string{"country_tag": `"XYZ"`})
	charsExtrasPortCheck(t, "folder", charsExtrasPortRow(t, p, "doctrines", "entity_type", "folder", "entity_key", "land_folder"), map[string]string{"name": `"LAND"`})
}

func TestCharsExtrasPortFillsOlderCharacters(t *testing.T) {
	root := t.TempDir()
	charsExtrasPortWriteMod(t, root, map[string]string{"common/characters/c.txt": `characters = { generic_y = { name = generic_y } }`})
	p := charsExtrasPortProject(t, "TST")
	if err := p.Write([]string{"characters"}, func(tx *project.Tx) error {
		tx.Insert("characters", project.Row{"tag": "old_one", "name": "old", "description": nil, "desc_en": "kept", "updated_at": "2000-01-01T00:00:00"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := p.Write([]string{"characters"}, func(tx *project.Tx) error {
		rows := tx.Rows("characters")
		next := append([]project.Row(nil), rows...)
		next[0] = project.Row{}
		for k, v := range rows[0] {
			next[0][k] = v
		}
		next[0]["updated_at"] = "2000-01-01T00:00:00"
		tx.Replace("characters", next)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	charsExtrasPortImport(t, p, root, AllOptions())
	charsExtrasPortCheck(t, "old_one", charsExtrasPortRow(t, p, "characters", "tag", "old_one"), map[string]string{
		"description": `""`, "desc_en": `"kept"`, "desc_zh": `""`, "updated_at": `"2000-01-01T00:00:00"`, "source": `null`,
	})
	charsExtrasPortCheck(t, "generic_y", charsExtrasPortRow(t, p, "characters", "tag", "generic_y"), map[string]string{
		"description": `""`, "desc_en": `""`, "desc_zh": `""`,
	})
}

func TestCharsExtrasPortNeededImagePaths(t *testing.T) {
	s := newSession(context.Background(), t.TempDir(), AllOptions(), nil)
	s.gfx["GFX_a"] = "gfx/a.dds"
	s.entities = []entityRow{
		{kind: "special_project", naturalKey: "sp", payload: []byte(`{"icon":"GFX_a","picture":"INTERFACE/x\\\\y.dds","blueprint_image":"other/z.dds","program_background":"gfx/p\\q.dds","large_portrait":"GFX_missing"}`)},
		{kind: "doctrine", naturalKey: "d", payload: []byte(`{"icon":"gfx/doctrine.dds"}`)},
		{kind: "trait:scientist", naturalKey: "t", payload: []byte(`{"icon":"gfx/trait.dds"}`)},
	}
	s.stageCharsExtrasNeededImages()
	var images []string
	for path := range s.neededImages {
		images = append(images, path)
	}
	sort.Strings(images)

	if got, want := strings.Join(images, "|"), `INTERFACE/x/y.dds|gfx/a.dds|gfx/p\q.dds`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCharsExtrasPortAgencyLocKey(t *testing.T) {
	cases := []struct {
		names any
		want  string
	}{
		{[]any{"Secret Service", "TST_agency"}, "TST_agency"},
		{[]any{"Two Words", "a-b"}, "fallback"},
		{[]any{nil, "", json.Number("42")}, "42"},
		{[]any{"Абвер.1"}, "Абвер.1"},
		{"not_an_array", "fallback"},
		{nil, "fallback"},
	}
	for _, tc := range cases {
		if got := charsExtrasAgencyLocKey(payload{"names": tc.names}, "fallback"); got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.names, got, tc.want)
		}
	}
}

func TestCharsExtrasPortScopeHelpers(t *testing.T) {
	for key, want := range map[string]bool{"GER_x": true, "GER": true, "AB_x": true, "ABCDEF_x": false, "A_x": false, "Ger_x": false, "x_GER": false, "_GER": false} {
		if _, got := charsExtrasTagPrefix(key); got != want {
			t.Errorf("%s: tag-like %v, want %v", key, got, want)
		}
	}
	proj := charsExtrasProject{countryTag: "TST", countryTags: []any{"ABC"}}
	for key, want := range map[string]bool{"TST_a": true, "ABC_a": true, "FOO_a": false, "foo_a": true} {
		if got := proj.inScope(key, payload{}); got != want {
			t.Errorf("%s: in scope %v, want %v", key, got, want)
		}
	}
	if !proj.inScope("XYZ_a", payload{"_country_tag_override": "XYZ"}) || proj.inScope("TST_a", payload{"_country_tag_override": "XYZ"}) {
		t.Error("the override replaces the project's tag in the scope check")
	}
	if (charsExtrasProject{}).inScope("TST_a", payload{}) {
		t.Error("a NULL project tag matches nothing")
	}
	if got := charsExtrasNullableBool(payload{"a": "", "b": true, "c": "no", "d": "maybe"}, "a"); got != nil {
		t.Errorf("empty: %v", got)
	}
	for key, want := range map[string]any{"b": true, "c": false, "d": nil, "missing": nil} {
		if got := charsExtrasNullableBool(payload{"b": true, "c": "no", "d": "maybe"}, key); got != want {
			t.Errorf("%s: %v, want %v", key, got, want)
		}
	}
}

func TestCharsExtrasPortParseFamilies(t *testing.T) {
	cases := []struct{ path, body, kind string }{
		{"common/characters/USA.txt", `characters={ USA_test={ name=USA_TEST advisor={ slot=political_advisor idea_token=USA_test_advisor } } }`, "character"},
		{"common/country_leader/traits.txt", `leader_traits={ USA_test_trait={ random=no modifier={ political_power_gain=0.1 } } }`, "trait:country_leader"},
		{"common/ideas/test.txt", `ideas={ country={ USA_test_idea={ picture=GFX_test modifier={ stability_factor=0.1 } } } }`, "idea"},
		{"common/special_projects/projects/test.txt", `sp_test={ specialization=specialization_nuclear icon=GFX_sp_test }`, "special_project"},
	}
	for _, tc := range cases {
		rows, err := parseCharacterExtra(tc.path, tc.body)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if len(rows) != 1 || rows[0].kind != tc.kind || len(rows[0].payload) == 0 {
			t.Fatalf("%s: rows %#v", tc.path, rows)
		}
	}
	if rows, err := parseCharacterExtra("gfx/interface/x.txt", "x={}"); err != nil || len(rows) != 0 {
		t.Fatalf("unrelated file routed: %v %#v", err, rows)
	}
}
