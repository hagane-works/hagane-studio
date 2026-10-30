package importjob

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func cosmeticMinimalMod(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"common/country_tags/00_tags.txt":             "ROM = \"countries/Romania.txt\"\n",
		"common/countries/Romania.txt":                "color = { 10 20 30 }\n",
		"common/countries/cosmetic.txt":               "ROM_greater_kingdom = { color = rgb { 12 34 56 } color_ui = rgb { 65 43 21 } }\nABC = { color = { 1 2 3 } color_ui = { 4 5 6 } }\nHSV_tag = { color = hsv { 0.5 1.0 1.0 } }\n",
		"history/countries/ROM - Romania.txt":         "set_cosmetic_tag = ROM_greater_kingdom\n",
		"localisation/english/cosmetic_l_english.yml": "\ufeffl_english:\n ROM_greater_kingdom:0 \"Greater Kingdom\"\n ROM_greater_kingdom_ADJ:0 \"\"\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCosmeticLegacyParserReproduction(t *testing.T) {
	root := cosmeticMinimalMod(t)
	body, err := os.ReadFile(filepath.Join(root, "common", "countries", "cosmetic.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := ParseSharedCountryColors(decodeScriptBytes(body))
	if _, ok := got["ROM_greater_kingdom"]; ok {
		t.Fatal("baseline changed: long tag unexpectedly retained")
	}
	if _, ok := got["HSV_tag"]; ok {
		t.Fatal("baseline changed: HSV unexpectedly retained")
	}
	if !reflect.DeepEqual(got["ABC"], []int{4, 5, 6}) {
		t.Fatalf("baseline color_ui preference = %v", got)
	}
	t.Logf("real minimal mod bytes through existing Go parser: long tag LOST; HSV LOST; separate map color MUTATED to UI: %v", got)
}
