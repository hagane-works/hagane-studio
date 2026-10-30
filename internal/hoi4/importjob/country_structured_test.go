package importjob

import (
	"testing"
)

func TestParseCountryTagsDynamicFirstWinsAndExoticLines(t *testing.T) {
	rows := parseCountryTags("\ufeffUSA = \"countries/USA.txt\"\rUSA = \"countries/X.txt\"\u2028dynamic_tags = yes\u0085D01 = \"countries/D01.txt\"")
	if len(rows) != 2 || rows[0].Tag != "USA" || rows[0].Path != "countries/usa.txt" || rows[0].Dynamic {
		t.Fatalf("rows=%+v", rows)
	}
	if rows[1].Tag != "D01" || !rows[1].Dynamic {
		t.Fatalf("dynamic=%+v", rows[1])
	}
}

func TestCountryMetadataRGBPreferenceAndCulture(t *testing.T) {
	m := countryMetadata(`graphical_culture = asian_gfx
graphical_culture_2d = asian_2d
color = { 1 2 3 }
color_ui = rgb { 4 5 6 }
use_legacy_ai_pp_spend = no`)
	rgb := m["color"].([]int)
	if rgb[0] != 4 || rgb[2] != 6 || m["graphical_culture"] != "asian_gfx" || m["use_legacy_ai_pp_spend"] != false {
		t.Fatalf("m=%v", m)
	}
}

func TestSharedCountryMetadataBalancedBlocks(t *testing.T) {
	m := sharedCountryMetadata("USA = { color = rgb { 1 2 3 } }\nGER = { color = { 4 5 6 } }")
	if m["USA"]["color"].([]int)[0] != 1 || m["GER"]["color"].([]int)[2] != 6 {
		t.Fatalf("m=%v", m)
	}
}

func TestSafeFactionPath(t *testing.T) {
	if !safeModPath("common/factions/x.txt") || safeModPath("../x.txt") || safeModPath("C:/x.txt") {
		t.Fatal("path gate")
	}
}

func TestDecodeFactionContentRemovesPostgresNULs(t *testing.T) {

	got := decodeFactionContent([]byte{0xff, 0xfe, 'f', 0, '=', 0, '{', 0, '}', 0})
	if got != "f={}" {
		t.Fatalf("decoded faction content=%q", got)
	}
}

func TestCountryMergePersistsCanonicalLocaleMaps(t *testing.T) {
	root := t.TempDir()
	countryPortWrite(t, root, "common/country_tags/00.txt", []byte("TST = \"countries/Test.txt\"\n"))
	loc := func(name, adjective, democratic string) {
		countryPortWrite(t, root, "localisation/english/t_l_english.yml", []byte("l_english:\n TST:0 \""+name+"\"\n TST_ADJ:0 \""+adjective+"\"\n TST_democratic:0 \""+democratic+"\"\n"))
	}
	p := countryPortProject(t)
	loc("Testland", "Testish", "Free Testland")
	countryPortImport(t, p, root, "overwrite", nil)
	tst := countryPortOne(t, p, "countries", "tag", "TST")
	if countryPortJSON(t, tst["display_name_locs"]) != `{"en":"Testland"}` || countryPortJSON(t, tst["adjective_locs"]) != `{"en":"Testish"}` ||
		countryPortJSON(t, tst["ideology_name_locs"]) != `{"communism":{},"democratic":{"en":"Free Testland"},"fascism":{},"neutrality":{}}` {
		t.Fatalf("locale maps: %v", tst)
	}
	loc("Testia", "Testian", "Free Testia")
	countryPortImport(t, p, root, "overwrite", nil)
	tst = countryPortOne(t, p, "countries", "tag", "TST")
	if countryPortJSON(t, tst["display_name_locs"]) != `{"en":"Testia"}` || countryPortJSON(t, tst["adjective_locs"]) != `{"en":"Testian"}` ||
		countryPortJSON(t, tst["ideology_name_locs"]) != `{"communism":{},"democratic":{"en":"Free Testia"},"fascism":{},"neutrality":{}}` ||
		tst["name_democratic_en"] != "Free Testia" || tst["adjective_en"] != "Testian" {
		t.Fatalf("re-import did not replace the locale maps: %v", tst)
	}
}
