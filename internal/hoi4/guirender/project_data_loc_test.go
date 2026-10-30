package guirender

import "testing"

func TestProjectDataLocFile(t *testing.T) {
	index, err := newIndex(map[string][]byte{
		"localisation/english/ABK_characters_l_english.yml":           []byte("l_english:\n ABK_boris:0 \"Boris\"\n"),
		"localisation/simp_chinese/ABK_characters_l_simp_chinese.yml": []byte("l_simp_chinese:\n ABK_boris:0 \"波罗斯\"\n"),
		"localisation/english/imported_l_english.yml":                 []byte("l_english:\n ABK_imported:0 \"Kept\"\n"),
		"localisation/english/raw_l_english.yml":                      []byte("l_english:\n ABK_raw:0 \"Verbatim\"\n"),
		"localisation/english/replace/zz_game_view_l_english.yml":     []byte("l_english:\n ABK_rewritten:0 \"Mine\"\n"),
	}, "mod")
	if err != nil {
		t.Fatal(err)
	}
	index.SetProjectFileSources(map[string]string{
		"localisation/english/ABK_characters_l_english.yml":           "character_localisation",
		"localisation/simp_chinese/ABK_characters_l_simp_chinese.yml": "character_localisation",
		"localisation/english/imported_l_english.yml":                 "project_loc_file",
		"localisation/english/raw_l_english.yml":                      "project_raw_files:12",
		"localisation/english/replace/zz_game_view_l_english.yml":     "project_loc_file",
	})
	for _, tc := range []struct {
		key, header, file string
		owned             bool
	}{
		{"ABK_boris", "l_english", "localisation/english/ABK_characters_l_english.yml", true},
		{"ABK_boris", "l_simp_chinese", "localisation/simp_chinese/ABK_characters_l_simp_chinese.yml", true},
		{"ABK_boris", "l_russian", "", false},
		{"ABK_imported", "l_english", "", false},
		{"ABK_raw", "l_english", "", false},
		{"ABK_rewritten", "l_english", "", false},
		{"neutrality", "l_english", "", false},
	} {
		file, owned := index.ProjectDataLocFile(tc.key, tc.header)
		if file != tc.file || owned != tc.owned {
			t.Errorf("%s in %s = (%q, %v), want (%q, %v)", tc.key, tc.header, file, owned, tc.file, tc.owned)
		}
	}

	bare, err := newIndex(map[string][]byte{
		"localisation/english/ABK_characters_l_english.yml": []byte("l_english:\n ABK_boris:0 \"Boris\"\n"),
	}, "mod")
	if err != nil {
		t.Fatal(err)
	}
	if file, owned := bare.ProjectDataLocFile("ABK_boris", "l_english"); owned {
		t.Errorf("no provenance, still owned by %q", file)
	}
}
