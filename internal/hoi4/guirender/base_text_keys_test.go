package guirender

import "testing"

func TestEventPayloadCarriesTextKeysAndOrigin(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"events/Tagland.txt": []byte(`
add_namespace = tagland
country_event = {
	id = tagland.1
	title = tagland.1.t
	desc = tagland.1.d
	option = { name = tagland.1.a }
}
country_event = {
	id = tagland.2
	title = tagland.2.t
	option = { name = tagland.2.a }
}`),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"events/Modded.txt": []byte(`
add_namespace = tagland
country_event = {
	id = tagland.2
	title = tagland.2.mod_title
	option = { name = tagland.2.a }
}`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	first := project.EventPayloadFor("tagland.1", "", "")
	if first["origin"] != "vanilla" || first["title_key"] != "tagland.1.t" || first["desc_key"] != "tagland.1.d" {
		t.Errorf("base-game event = %v", first)
	}
	second := project.EventPayloadFor("tagland.2", "", "")
	if second["origin"] != "mod" || second["title_key"] != "tagland.2.mod_title" {
		t.Errorf("event the mod redeclares = %v", second)
	}
	if _, has := second["desc_key"]; has {
		t.Errorf("an event without desc should carry no desc_key: %v", second)
	}
	if project.EventPayloadFor("tagland.9", "", "") != nil {
		t.Error("an unknown event should come back nil")
	}
}

func TestCharacterNameKeyFindsTheLayer(t *testing.T) {
	base, err := newIndex(map[string][]byte{
		"common/characters/GER.txt": []byte(`characters = {
			GER_franz_seldte = { name = GER_franz_seldte_name advisor = { slot = political_advisor } }
			GER_no_name = { advisor = { slot = political_advisor } }
		}`),
		"common/idea_tags/00_idea.txt": []byte(ideaTags),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	project, err := NewIndexWithFallback(map[string][]byte{
		"common/characters/GER.txt": []byte(`characters = {
			GER_mod_person = { name = GER_mod_person advisor = { slot = political_advisor } }
		}`),
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		token, key, origin string
		ok                 bool
	}{
		{"GER_franz_seldte", "GER_franz_seldte_name", "vanilla", true},
		{"GER_mod_person", "GER_mod_person", "mod", true},
		{"GER_missing", "", "", false},
	} {
		key, origin, ok := project.CharacterNameKey(tc.token)
		if key != tc.key || origin != tc.origin || ok != tc.ok {
			t.Errorf("%s = (%q, %q, %v), want (%q, %q, %v)", tc.token, key, origin, ok, tc.key, tc.origin, tc.ok)
		}
	}
	if key, _, ok := project.CharacterNameKey("GER_no_name"); !ok || key == "" {
		t.Errorf("a person without name = should still have a key, got %q %v", key, ok)
	}
}

func TestCharacterNameKeyFindsPeopleFiledUnderNoTag(t *testing.T) {
	index, err := newIndex(map[string][]byte{
		"common/characters/Warlords_China.txt": []byte(`characters = {
			GXC_li_zongren = { name = GXC_li_zongren_name country_leader = { ideology = despotism } }
		}`),
	}, "vanilla")
	if err != nil {
		t.Fatal(err)
	}
	key, origin, ok := index.CharacterNameKey("GXC_li_zongren")
	if !ok || key != "GXC_li_zongren_name" || origin != "vanilla" {
		t.Errorf("person from a file named for no tag = (%q, %q, %v)", key, origin, ok)
	}
}
