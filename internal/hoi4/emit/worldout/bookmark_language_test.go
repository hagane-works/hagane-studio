package worldout

import (
	"strings"
	"testing"
)

func TestBookmarkNameCrossFillsEveryExportContentLanguage(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := NewOMap()
	project.Set("export_languages", []Value{"zh"})
	bookmark := NewOMap()
	bookmark.Set("bookmark_id", "iron_curtain")
	bookmark.Set("date", "1964.5.1.12")
	names := NewOMap()
	names.Set("zh", "铁幕")
	bookmark.Set("display_name_locs", names)

	records, meta := EmitScenario(tables, &Scenario{
		Project: project, Tag: "GER", ModName: "Demo", ModDirName: "Demo",
		ExistenceSets: map[string]map[string]bool{"idea": {}, "focus": {}},
		Bookmarks:     []*OMap{bookmark},
	})
	byPath := map[string]string{}
	for _, record := range records {
		byPath[record.Path] = record.Text
	}
	for _, path := range []string{
		"Demo/localisation/english/replace/iron_curtain_l_english.yml",
		"Demo/localisation/simp_chinese/replace/iron_curtain_l_simp_chinese.yml",
	} {
		text := byPath[path]
		if !strings.HasPrefix(text, bomStr) ||
			!strings.Contains(text, `IRON_CURTAIN_NAME: "铁幕"`) {
			t.Fatalf("%s does not define the referenced NAME with BOM:\n%q", path, text)
		}
	}
	if script := byPath["Demo/common/bookmarks/iron_curtain.txt"]; !strings.Contains(script, `name = "IRON_CURTAIN_NAME"`) {
		t.Fatalf("bookmark script lost NAME reference:\n%s", script)
	}
	joined := joinWarnings(meta.Symmetry)
	if !strings.Contains(joined,
		"iron_curtain: the bookmark has no name in en, so its zh name was used") {
		t.Fatalf("missing-language finding not recorded: %v", meta.Symmetry)
	}
	if strings.Contains(joined, "bookmark · IRON_CURTAIN_NAME") {
		t.Fatalf("cross-filled key sets must be symmetric: %v", meta.Symmetry)
	}
}

func TestBookmarkNameDoesNotInventAppLocaleAsContentLanguage(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatalf("load tables: %v", err)
	}
	project := NewOMap()
	project.Set("export_languages", []Value{"en"})
	project.Set("app_locale", "zh")
	bookmark := NewOMap()
	bookmark.Set("bookmark_id", "demo")
	bookmark.Set("name_en", "Demo")
	records, _ := EmitScenario(tables, &Scenario{
		Project: project, Tag: "GER", ModName: "Demo", ModDirName: "Demo",
		ExistenceSets: map[string]map[string]bool{"idea": {}, "focus": {}},
		Bookmarks:     []*OMap{bookmark},
	})
	for _, record := range records {
		if strings.Contains(record.Path, "/simp_chinese/") {
			t.Fatalf("AppLocale leaked into content output: %s", record.Path)
		}
	}
}

func TestBookmarkSymmetryNegativeControlDetectsRemovedName(t *testing.T) {
	warnings := assertYmlSymmetricMulti(
		[]string{"en", "zh"},
		map[string]string{
			"en": "l_english:\n IRON_CURTAIN_NAME: \"Iron Curtain\"\n",
			"zh": "l_simp_chinese:\n",
		},
		"bookmark",
	)
	if got := joinWarnings(warnings); !strings.Contains(got, "bookmark · IRON_CURTAIN_NAME: written in en, missing in zh") {
		t.Fatalf("negative control did not fail on the removed key: %v", warnings)
	}
}
