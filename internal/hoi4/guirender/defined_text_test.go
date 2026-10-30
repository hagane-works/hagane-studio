package guirender

import "testing"

func definedTextIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/bookmarks/storm.txt": []byte(`bookmarks = { bookmark = {
			name = STORM date = 1936.1.1.12 default = yes
			TAG = { history = TAG_DESC ideology = fascism }
			"---" = { history = OTHER_DESC }
			SML = { minor = yes history = SML_DESC ideology = neutrality }
		} }`),
		"history/countries/TAG - Tag.txt":   []byte("capital = 1\nset_politics = { ruling_party = fascism }\n"),
		"history/countries/SML - Small.txt": []byte("capital = 2\nset_politics = { ruling_party = neutrality }\n"),
		"common/scripted_localisation/films.txt": []byte(`
			defined_text = {
				name = GetFilmIndustry
				text = { trigger = { is_major = yes } localization_key = DOMESTIC_FILMS }
				text = { localization_key = IMPORTED_FILMS }
			}
			defined_text = {
				name = GetFilmTitle
				text = { random_list = { 10 = { localization_key = FILM_A } 10 = { localization_key = FILM_B } } }
			}
			defined_text = {
				name = GetStrength
				text = { trigger = { num_of_factories > 50 } localization_key = STRONG }
				text = { localization_key = WEAK }
			}
		`),
		"localisation/english/films_l_english.yml": []byte("l_english:\n" +
			" film_decision: \"[ROOT.GetFilmIndustry]\"\n" +
			" film_bare: \"We use [GetFilmIndustry]\"\n" +
			" film_title: \"Now showing: [ROOT.GetFilmTitle]\"\n" +
			" film_strength: \"We are [ROOT.GetStrength]\"\n" +
			" DOMESTIC_FILMS: \"Utilize Domestic Film Industry\"\n" +
			" IMPORTED_FILMS: \"Import Foreign Films\"\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestDefinedTextPicksTheOptionTheStartDecides(t *testing.T) {
	idx := definedTextIndex(t)
	cases := []struct{ key, tag, want string }{

		{"film_decision", "TAG", "Utilize Domestic Film Industry"},
		{"film_decision", "SML", "Import Foreign Films"},

		{"film_bare", "TAG", "We use Utilize Domestic Film Industry"},

		{"film_title", "TAG", "Now showing: {ROOT.GetFilmTitle}"},
		{"film_strength", "TAG", "We are {ROOT.GetStrength}"},
	}
	for _, c := range cases {
		if got, _ := idx.LocaliseFor(c.key, c.tag, "1936.1.1"); got != c.want {
			t.Errorf("%s for %s = %q, want %q", c.key, c.tag, got, c.want)
		}
	}

	if got, _ := idx.Localise("film_decision"); got != "{ROOT.GetFilmIndustry}" {
		t.Errorf("film_decision with no country = %q", got)
	}
}
