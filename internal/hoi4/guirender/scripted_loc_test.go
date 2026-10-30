package guirender

import "testing"

func scriptedIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/country_tags/00_countries.txt": []byte("GER = \"countries/Germany.txt\"\nDEN = \"countries/Denmark.txt\"\n"),
		"history/countries/GER - Germany.txt":  []byte("capital = 64\nset_politics = { ruling_party = fascism }\n"),
		"history/countries/DEN - Denmark.txt":  []byte("set_politics = { ruling_party = democratic }\n"),
		"history/states/64-Brandenburg.txt":    []byte("state = { id = 64 name = STATE_64 history = { owner = GER } }\n"),
		"localisation/english/c_l_english.yml": []byte("l_english:\n" +
			" GER:0 \"Germany\"\n GER_fascism:0 \"German Reich\"\n GER_fascism_DEF:0 \"the German Reich\"\n" +
			" GER_fascism_ADJ:0 \"German\"\n GER_fascism_party:0 \"NSDAP\"\n" +
			" DEN:0 \"Denmark\"\n DEN_DEF:0 \"Denmark\"\n DEN_ADJ:0 \"Danish\"\n STATE_64:0 \"Brandenburg\"\n" +
			" ev.t:0 \"[GER.GetNameDefCap] marches\"\n" +
			" ev.d:0 \"[ROOT.GetAdjective] troops enter [Den.GetNameDef]; [FROM.GetName] protests.\"\n" +
			" ev.c:0 \"[GER.Capital.GetName] rejoices under the [GER.GetRulingParty]\"\n" +
			" ev.n:0 \"A dispute within $GER_fascism_party$\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestLiteralTagCallsResolveEverywhere(t *testing.T) {
	idx := scriptedIndex(t)
	if got, _ := idx.Localise("ev.t"); got != "The German Reich marches" {
		t.Errorf("ev.t = %q", got)
	}
	if got, _ := idx.Localise("ev.c"); got != "Brandenburg rejoices under the NSDAP" {
		t.Errorf("ev.c = %q", got)
	}
}

func TestRootResolvesToTheCountryOnScreen(t *testing.T) {
	idx := scriptedIndex(t)
	got, _ := idx.LocaliseFor("ev.d", "GER", "")
	if got != "German troops enter Denmark; {FROM.GetName} protests." {
		t.Errorf("ev.d for GER = %q", got)
	}

	if plain, _ := idx.Localise("ev.d"); plain != "{ROOT.GetAdjective} troops enter Denmark; {FROM.GetName} protests." {
		t.Errorf("ev.d with no root = %q", plain)
	}
}

func TestNestedKeysResolveInPlainText(t *testing.T) {
	idx := scriptedIndex(t)
	if got, _ := idx.Localise("ev.n"); got != "A dispute within NSDAP" {
		t.Errorf("ev.n = %q", got)
	}
}
