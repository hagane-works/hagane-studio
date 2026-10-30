package worldout

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/cosmetic"
)

func cosmeticTestContext(t *testing.T) *emitCtx {
	t.Helper()
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	project := NewOMap()
	project.Set("export_languages", []Value{"en", "zh"})
	return &emitCtx{t: tables, s: &Scenario{Project: project, Tag: "ROM", ModName: "Demo", ModDirName: "demo", IncludeImported: true}, modDir: "demo/", triples: buildExportLangTriples([]string{"en", "zh"})}
}

func cosmeticTexts(records []Record) map[string]string {
	result := map[string]string{}
	for _, record := range records {
		result[record.Path] = record.Text
	}
	return result
}

func TestCosmeticHelperAuthoredFormsAndEscaping(t *testing.T) {
	c := cosmeticTestContext(t)
	name := ` 翁法罗斯 "帝国" \ $TOKEN$ §Y名称§! `
	def := cosmetic.Definition{Tag: "ROM_new", NameLocs: cosmetic.Names{"l_english": {"": "Germany", "ADJ": "German", "communism": "Socialist Republic"}, "l_simp_chinese": {"": name}}}
	c.emitCosmeticDefinitions(nil, map[string]cosmetic.Definition{def.Tag: def}, map[string]map[string]bool{"l_english": {"ROM_new_communism_ADJ": true}})
	files := cosmeticTexts(c.records)
	english := files["demo/localisation/english/replace/zz_demo_cosmetic_l_english.yml"]
	for _, line := range []string{` ROM_new_communism_ADJ:0 "German"`, ` ROM_new_communism_DEF:0 "Socialist Republic"`, ` ROM_new_DEF:0 "Germany"`} {
		if !strings.Contains(english, line) {
			t.Fatalf("missing authored form %s: %s", line, english)
		}
	}
	chinese := files["demo/localisation/simp_chinese/replace/zz_demo_cosmetic_l_simp_chinese.yml"]
	for _, text := range []string{english, chinese} {
		if !strings.HasPrefix(text, bomStr) || strings.Count(text, ":0 ") != 15 {
			t.Fatalf("missing BOM/forms: %q", text)
		}
	}
	if !strings.Contains(chinese, ` ROM_new_fascism_DEF:0 " 翁法罗斯 \"帝国\" \\ $TOKEN$ §Y名称§! "`) {
		t.Fatalf("escaping changed: %q", chinese)
	}
}

func TestCosmeticHelperPreservedEligibility(t *testing.T) {
	for _, include := range []bool{false, true} {
		c := cosmeticTestContext(t)
		c.s.IncludeImported = include
		def := cosmetic.Definition{Tag: "ROM_new", ImportedNameLocs: cosmetic.Names{"l_english": {"": "", "ADJ": "ineligible imported"}}}
		preserved := map[string]map[string]bool{"l_english": {"ROM_new": true}}
		c.emitCosmeticDefinitions(map[string]string{"ROM_new": "ROM"}, map[string]cosmetic.Definition{"ROM_new": def}, preserved)
		english := cosmeticTexts(c.records)["demo/localisation/english/replace/zz_demo_cosmetic_l_english.yml"]
		if strings.Contains(english, " ROM_new:0") {
			t.Fatal("eligible explicit imported key was shadowed")
		}
		if !strings.Contains(english, ` ROM_new_ADJ:0 "ROM New"`) {
			t.Fatal("ineligible imported occurrence made a missing/falsely imported key")
		}
		chinese := cosmeticTexts(c.records)["demo/localisation/simp_chinese/replace/zz_demo_cosmetic_l_simp_chinese.yml"]
		if !strings.Contains(chinese, ` ROM_new:0 "ROM New"`) {
			t.Fatal("English preserved key suppressed another locale")
		}
	}
}

func TestCosmeticHelperColorsRegistryAndVanilla(t *testing.T) {
	c := cosmeticTestContext(t)
	c.t.CosmeticTags["vanilla_id"] = true
	c.t.CosmeticTags["vanilla_untouched"] = true
	authored, imported, ui := cosmetic.RGB{1, 2, 3}, cosmetic.RGB{4, 5, 6}, cosmetic.RGB{7, 8, 9}
	defs := map[string]cosmetic.Definition{"registry_only": {Tag: "registry_only", Color: &authored, ImportedColor: &imported, ImportedColorUI: &ui}, "vanilla_id": {Tag: "vanilla_id", Color: &authored}}
	c.emitCosmeticDefinitions(map[string]string{"vanilla_untouched": "ROM"}, defs, nil)
	colors := cosmeticTexts(c.records)["demo/common/countries/cosmetic.txt"]
	for _, needle := range []string{"registry_only = {\n\tcolor = rgb { 1 2 3 }\n\tcolor_ui = rgb { 7 8 9 }", "vanilla_id = {"} {
		if !strings.Contains(colors, needle) {
			t.Fatalf("missing %q: %s", needle, colors)
		}
	}
	if strings.Contains(colors, "vanilla_untouched") {
		t.Fatal("unmodeled vanilla tag was changed")
	}
	c = cosmeticTestContext(t)
	c.s.IncludeImported = false
	c.emitCosmeticDefinitions(nil, defs, nil)
	colors = cosmeticTexts(c.records)["demo/common/countries/cosmetic.txt"]
	if !strings.Contains(colors, "registry_only = {\n\tcolor = rgb { 1 2 3 }\n\tcolor_ui = rgb { 1 2 3 }") {
		t.Fatalf("excluded imported UI leaked: %s", colors)
	}
}

func TestCosmeticHelperLegacyBytes(t *testing.T) {

	original := cosmeticTestContext(t)
	owner := NewOMap()
	owner.Set("tag", "ROM")
	owner.Set("color", []Value{json.Number("10"), json.Number("20"), json.Number("30")})
	owner.Set("color_ui", []Value{json.Number("40"), json.Number("50"), json.Number("60")})
	original.s.Countries = []*OMap{owner}
	original.emitCountrySections(map[string]string{"ROM_greater_kingdom": "ROM"})
	next := cosmeticTestContext(t)
	next.s.Countries = original.s.Countries
	next.emitCosmeticDefinitions(map[string]string{"ROM_greater_kingdom": "ROM"}, nil, nil)
	oldFiles := cosmeticTexts(original.records)
	for _, record := range next.records {
		if oldFiles[record.Path] != record.Text {
			t.Fatalf("legacy bytes changed at %s\nold=%q\nnew=%q", record.Path, oldFiles[record.Path], record.Text)
		}
	}
	expected := bomStr + "l_english:\n"
	for _, suffix := range []string{"", "_DEF", "_ADJ", "_democratic", "_democratic_DEF", "_democratic_ADJ", "_fascism", "_fascism_DEF", "_fascism_ADJ", "_neutrality", "_neutrality_DEF", "_neutrality_ADJ", "_communism", "_communism_DEF", "_communism_ADJ"} {
		expected += " ROM_greater_kingdom" + suffix + ":0 \"ROM Greater Kingdom\"\n"
	}
	if actual := cosmeticTexts(next.records)["demo/localisation/english/replace/zz_demo_cosmetic_l_english.yml"]; actual != expected {
		t.Fatalf("legacy order/version/whitespace changed: %q", actual)
	}
}

func TestCosmeticHelperOwnerAndHashCompatibility(t *testing.T) {
	for _, tag := range []string{"ROM_greater_kingdom", "mixed-case.Name", "USA_TEST", "éCOLE_国"} {
		got, _ := cosmetic.ResolveColors(cosmetic.Definition{Tag: tag}, nil, nil)
		legacy := deriveTagColor(tag)
		if got != (cosmetic.RGB{int(legacy[0]), int(legacy[1]), int(legacy[2])}) {
			t.Fatalf("hash changed for %q", tag)
		}
		if cosmetic.Humanize(tag) != humanizeCosmeticName(tag) {
			t.Fatalf("humanize changed for %q", tag)
		}
	}
	owner := NewOMap()
	owner.Set("tag", "ROM")
	owner.Set("color", []Value{json.Number("1"), json.Number("2"), json.Number("3")})
	owner.Set("color_ui", []Value{json.Number("4"), json.Number("5"), json.Number("6")})
	color, ui := CosmeticOwnerColors(owner, nil)
	if color == nil || ui == nil || *color != (cosmetic.RGB{1, 2, 3}) || *ui != (cosmetic.RGB{4, 5, 6}) {
		t.Fatal("owner fallback lost")
	}
}

func TestCosmeticHelperProjectOwnerIndependentOfExportFilter(t *testing.T) {
	for _, includeImported := range []bool{false, true} {
		for _, hasExportedHistory := range []bool{false, true} {
			c := cosmeticTestContext(t)
			c.s.IncludeImported = includeImported
			c.s.CosmeticOwners = map[string]string{"shared_tag": "ROM", "unemitted_tag": "ROM"}
			for _, entry := range []struct{ tag, channel string }{{"ROM", "10"}, {"GER", "20"}} {
				country := NewOMap()
				country.Set("tag", entry.tag)
				country.Set("color", []Value{json.Number(entry.channel), json.Number("2"), json.Number("3")})
				c.s.Countries = append(c.s.Countries, country)
			}
			references := map[string]string{}
			if hasExportedHistory {
				references["shared_tag"] = "GER"
			}
			c.emitCosmeticDefinitions(references, map[string]cosmetic.Definition{"shared_tag": {Tag: "shared_tag"}}, nil)
			colors := cosmeticTexts(c.records)["demo/common/countries/cosmetic.txt"]
			if !strings.Contains(colors, "shared_tag = {\n\tcolor = rgb { 10 2 3 }") {
				t.Fatalf("stable owner lost: includeImported=%v history=%v: %s", includeImported, hasExportedHistory, colors)
			}
			if strings.Contains(colors, "unemitted_tag") {
				t.Fatal("owner map added an otherwise excluded tag")
			}
		}
	}
}

func TestCosmeticOwnerColorsUsesActualCountryDefaults(t *testing.T) {
	custom := NewOMap()
	custom.Set("tag", "XYZ")
	color, ui := CosmeticOwnerColors(custom, nil)
	expected := deriveTagColor("XYZ")
	if color == nil || *color != (cosmetic.RGB{int(expected[0]), int(expected[1]), int(expected[2])}) || ui != nil {
		t.Fatalf("custom owner's generated country color lost: %v/%v", color, ui)
	}
	vanilla := NewOMap()
	vanilla.Set("color", []Value{json.Number("12"), json.Number("34"), json.Number("56")})
	vanilla.Set("graphical_culture", "western_european_gfx")
	owner := NewOMap()
	owner.Set("tag", "ROM")
	owner.Set("graphical_culture", "asian_gfx")
	color, ui = CosmeticOwnerColors(owner, map[string]*OMap{"ROM": vanilla})
	if color == nil || *color != (cosmetic.RGB{12, 34, 56}) || ui != nil {
		t.Fatalf("custom-GFX owner must retain its resolved vanilla map color: %v/%v", color, ui)
	}
	if color, ui := CosmeticOwnerColors(nil, nil); color != nil || ui != nil {
		t.Fatal("missing owner must not invent an owner color")
	}
}

func TestCosmeticCentralScenarioWiring(t *testing.T) {
	c := cosmeticTestContext(t)
	fields := reflect.ValueOf(c.s).Elem()
	registry := fields.FieldByName("CosmeticDefinitions")
	preserved := fields.FieldByName("PreservedLocKeys")
	if !registry.IsValid() || !preserved.IsValid() {
		t.Fatal("central seam pending: Scenario.CosmeticDefinitions and Scenario.PreservedLocKeys are required")
	}
	registry.Set(reflect.ValueOf(map[string]cosmetic.Definition{"registry_only": {Tag: "registry_only", NameLocs: cosmetic.Names{"l_english": {"ADJ": "New adjective"}}}}))
	preserved.Set(reflect.ValueOf(map[string]map[string]bool{"l_english": {"registry_only": true}}))
	records, _ := EmitScenario(c.t, c.s)
	files := cosmeticTexts(records)
	english := files["demo/localisation/english/replace/zz_demo_cosmetic_l_english.yml"]
	if !strings.Contains(english, ` registry_only_fascism_ADJ:0 "New adjective"`) {
		t.Fatal("central section did not emit registry-only authored names")
	}
	if strings.Contains(english, " registry_only:0") {
		t.Fatal("central section shadowed preserved exact key")
	}
	if !strings.Contains(english, ` registry_only_DEF:0 "Registry Only"`) {
		t.Fatal("central section left missing suffix without fallback")
	}
}
