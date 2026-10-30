package guirender

import "testing"

func TestColorsFileComesBeforeTheCountryFile(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/country_tags/00_countries.txt": []byte("GER = \"countries/Germany.txt\"\nZQX = \"countries/Zeta.txt\"\n"),
		"common/countries/Germany.txt":         []byte("color = { 106 119 89 }\n"),
		"common/countries/Zeta.txt":            []byte("color = { 1 2 3 }\n"),
		"common/countries/colors.txt":          []byte("GER = {\n\tcolor = HSV { 0.1 0.15 0.4 }\n\tcolor_ui = rgb { 138 155 116 }\n}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := idx.CountryMapColor("GER"); got != hsvToRGB(0.1, 0.15, 0.4) {
		t.Errorf("GER %v, want colors.txt's %v", got, hsvToRGB(0.1, 0.15, 0.4))
	}
	if got, _ := idx.CountryMapColor("ZQX"); got != [3]uint8{1, 2, 3} {
		t.Errorf("ZQX %v, want its own file's", got)
	}
}
