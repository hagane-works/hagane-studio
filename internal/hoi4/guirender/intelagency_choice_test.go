package guirender

import "testing"

const testBaseAgencies = `
intelligence_agency = {
	picture = GFX_intelligence_agency_logo_ger
	names = { "Abwehr" }
	default = { tag = GER }
	available = { original_tag = GER }
}
intelligence_agency = {
	picture = GFX_intelligence_agency_logo_generic_1
	names = { "Rooke" }
	default = { always = yes }
	available = { always = yes }
}
intelligence_agency = {
	picture = GFX_intelligence_agency_logo_generic_2
	names = { "Tower" }
	default = { always = yes }
	available = { always = yes }
}
`

const testModAgency = `
intelligence_agency = {
	picture = GFX_intelligence_agency_logo_usa
	names = { "Service de Renseignement QA" }
	default = { tag = LUX }
	available = { original_tag = LUX }
}
`

func TestIntelAgencyOpensOnTheCountrysOwnInsignia(t *testing.T) {
	base, err := NewIndex(map[string][]byte{
		"common/intelligence_agencies/00_intelligence_agencies.txt": []byte(testBaseAgencies),
		"history/countries/GER - Germany.txt":                       []byte("capital = 64\n"),
		"history/countries/LUX - Luxemburg.txt":                     []byte("capital = 8\n"),
		"history/countries/POR - Portugal.txt":                      []byte("capital = 112\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := NewIndexWithFallback(map[string][]byte{
		"common/intelligence_agencies/LUX_intelligence_agencies.txt": []byte(testModAgency),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		tag, want string
	}{
		{"LUX", "Service de Renseignement QA"},
		{"GER", "Abwehr"},
		{"POR", "Rooke"},
	} {
		if got := mod.IntelAgencyPayload(c.tag, "1936.1.1")["name"]; got != c.want {
			t.Errorf("%s opens on %q, want %q", c.tag, got, c.want)
		}
	}
}
