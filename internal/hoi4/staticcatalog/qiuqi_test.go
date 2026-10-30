package staticcatalog

import "testing"

func TestBundledModifierTranslationsMatchPythonCatalog(t *testing.T) {
	got := LoadModifierTranslations("")
	if len(got) != 288 {
		t.Fatalf("count = %d, want Python catalog count 288", len(got))
	}
	if entry := got["army_attack_factor"]; entry.NameZH != "陆军攻击" || entry.DescZH != "" {
		t.Fatalf("army_attack_factor = %+v", entry)
	}
}

func TestModifierDescriptionSiblingAndFirstWins(t *testing.T) {
	raw := []byte("l_english:\n MODIFIER_DEMO:0 \"first\"\n MODIFIER_DEMO:1 \"second\"\n MODIFIER_DEMO_DESC:0 \"desc\"\n MODIFIER_LONELY_DESC:0 \"literal name\"\n")
	got := ParseModifierTranslations(raw)
	if entry := got["demo"]; entry.NameZH != "first" || entry.DescZH != "desc" {
		t.Fatalf("demo = %+v", entry)
	}
	if entry := got["lonely_desc"]; entry.NameZH != "literal name" || entry.DescZH != "" {
		t.Fatalf("lonely_desc = %+v", entry)
	}
}
