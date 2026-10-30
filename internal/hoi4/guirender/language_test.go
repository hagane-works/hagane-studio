package guirender

import "testing"

func TestGameLanguageReadsLocalesAndGameNames(t *testing.T) {
	for in, want := range map[string]string{
		"zh": "simp_chinese", "zh-CN": "simp_chinese", "simp_chinese": "simp_chinese", "l_simp_chinese": "simp_chinese",
		"ru": "russian", "ja": "japanese", "ko": "korean", "es": "spanish", "fr": "french", "de": "german",
		"en": "english", "pt-BR": "braz_por", "": "", "xx": "", "klingon": "",
	} {
		if got := GameLanguage(in); got != want {
			t.Errorf("GameLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInLanguageReadsEachLayersTextAndFallsBackToEnglish(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"localisation/english/base_l_english.yml": []byte("l_english:\n BASE_KEY:0 \"Base\"\n BASE_ONLY_EN:0 \"Only English\"\n"),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"localisation/english/mod_l_english.yml":                 []byte("l_english:\n MOD_KEY:0 \"Mod\"\n MOD_ONLY_EN:0 \"Mod English\"\n"),
		"localisation/simp_chinese/mod_l_simp_chinese.yml":       []byte("\ufeffl_simp_chinese:\n MOD_KEY:0 \"模组\"\n"),
		"localisation/russian/mod_l_russian.yml":                 []byte("l_russian:\n MOD_KEY:0 \"Мод\"\n"),
		"localisation/simp_chinese/replace/x_l_simp_chinese.yml": []byte("l_simp_chinese:\n BASE_KEY:0 \"模组改原版\"\n"),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	zh := project.InLanguage("zh", map[string]string{"BASE_KEY": "原版", "BASE_ONLY_EN": "原版中文"})
	for key, want := range map[string]string{
		"MOD_KEY":      "模组",
		"BASE_KEY":     "模组改原版",
		"BASE_ONLY_EN": "原版中文",
		"MOD_ONLY_EN":  "Mod English",
	} {
		if got, _ := zh.Localise(key); got != want {
			t.Errorf("zh %s = %q, want %q", key, got, want)
		}
	}
	if zh.Language() != "simp_chinese" {
		t.Fatalf("language %q", zh.Language())
	}
	if got, _ := project.InLanguage("ru", nil).Localise("MOD_KEY"); got != "Мод" {
		t.Fatalf("ru = %q", got)
	}
	if got, _ := project.Localise("MOD_KEY"); got != "Mod" || project.Language() != "english" {
		t.Fatalf("the index itself changed: %q %q", got, project.Language())
	}
	if project.InLanguage("en", nil) != project || project.InLanguage("klingon", nil) != project {
		t.Fatal("English or an unknown language is not the index itself")
	}
}

func TestLanguageVariantsReplaceTheirPlainSibling(t *testing.T) {
	plain := map[string]any{"name": "subscription_widget", "raw": map[string]any{}, "children": []map[string]any{}}
	chinese := map[string]any{"name": "subscription_widget_simp_chinese", "raw": map[string]any{"hide": "yes"}, "children": []map[string]any{}}
	other := map[string]any{"name": "subscription_divider", "raw": map[string]any{}, "children": []map[string]any{}}
	window := map[string]any{"name": "mainmenu", "raw": map[string]any{}, "children": []map[string]any{other, plain, chinese}}

	if got := withLanguageVariants(window, "english"); !sameNode(got, window) {
		t.Fatal("English changed the window")
	}
	zh := withLanguageVariants(window, "simp_chinese")
	children := nodeChildren(zh)
	if !sameNode(children[0], other) {
		t.Fatal("an unrelated element was copied")
	}
	if children[1]["raw"].(map[string]any)["hide"] != "yes" || children[2]["raw"].(map[string]any)["hide"] != "no" {
		t.Fatalf("zh: plain %v chinese %v", children[1]["raw"], children[2]["raw"])
	}
	if _, set := plain["raw"].(map[string]any)["hide"]; set || chinese["raw"].(map[string]any)["hide"] != "yes" {
		t.Fatal("the index's own nodes were changed")
	}

	if got := withLanguageVariants(window, "russian"); !sameNode(got, window) {
		t.Fatal("a language with no variant changed the window")
	}
}

func TestLanguageTextReplaceFolderWins(t *testing.T) {
	index, err := NewIndex(map[string][]byte{
		"localisation/english/a_l_english.yml":                              []byte("l_english:\n KEY:0 \"Key\"\n"),
		"localisation/simp_chinese/replace/zz_game_view_l_simp_chinese.yml": []byte("l_simp_chinese:\n KEY:0 \"改过的\"\n"),
		"localisation/simp_chinese/zzz_l_simp_chinese.yml":                  []byte("l_simp_chinese:\n KEY:0 \"原来的\"\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := index.InLanguage("zh", nil).Localise("KEY"); got != "改过的" {
		t.Fatalf("zh KEY = %q", got)
	}
}
