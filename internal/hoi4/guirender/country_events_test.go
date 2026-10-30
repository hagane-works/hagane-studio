package guirender

import (
	"reflect"
	"sort"
	"testing"

	"haganeworks/internal/hoi4/paradox"
)

func strictTagsOf(t *testing.T, text string) (allow, deny []string) {
	t.Helper()
	root, _, err := paradox.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	collectStrictTags(root.Entries, &allow, &deny)
	return allow, deny
}

func TestCollectStrictTags(t *testing.T) {
	for _, c := range []struct {
		text        string
		allow, deny []string
	}{
		{`tag = WEI has_war = yes`, []string{"WEI"}, nil},
		{`original_tag = SHU`, []string{"SHU"}, nil},
		{`OR = { tag = WEI tag = SHU }`, []string{"WEI", "SHU"}, nil},
		{`OR = { tag = GER has_war = yes }`, nil, nil},
		{`AND = { tag = WUU }`, []string{"WUU"}, nil},
		{`NOT = { tag = MAN }`, nil, []string{"MAN"}},
		{`NOT = { tag = MAN has_war = yes }`, nil, nil},
		{`tag = ROOT tag = event_target:x`, nil, nil},
	} {
		allow, deny := strictTagsOf(t, c.text)
		if !reflect.DeepEqual(allow, c.allow) || !reflect.DeepEqual(deny, c.deny) {
			t.Errorf("%s: allow %v deny %v, want %v %v", c.text, allow, deny, c.allow, c.deny)
		}
	}
}

func countryEventsIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"common/national_focus/germany.txt": []byte(`focus_tree = {
	id = german_focus
	country = { factor = 0 modifier = { add = 10 tag = GER } }
	focus = { id = GER_a completion_reward = { country_event = ger.1 } }
}`),
		"common/national_focus/generic.txt": []byte(`focus_tree = {
	id = generic_focus
	country = { factor = 1 }
	default = yes
	focus = { id = gen_a completion_reward = { country_event = { id = gen.1 days = 2 } } }
}`),
		"common/decisions/categories/cats.txt": []byte(`SHU_cat = { allowed = { tag = SHU } }
common_cat = { }`),
		"common/decisions/shu.txt": []byte(`SHU_cat = { SHU_d = { complete_effect = { country_event = shu.1 } } }
common_cat = { common_d = { complete_effect = { random_events = { 50 = dec.1 50 = 0 } } } }`),
		"events/ger.txt": []byte(`add_namespace = ger
country_event = { id = ger.1 is_triggered_only = yes option = { name = ger.1.a country_event = ger.2 } }
country_event = { id = ger.2 is_triggered_only = yes option = { name = ger.2.a } }
country_event = { id = trig.1 trigger = { tag = GER } option = { name = trig.1.a } }`),
		"events/other.txt": []byte(`country_event = { id = shu.1 is_triggered_only = yes option = { name = shu.1.a } }
country_event = { id = gen.1 is_triggered_only = yes option = { name = gen.1.a } }
country_event = { id = dec.1 is_triggered_only = yes option = { name = dec.1.a } }
country_event = { id = orphan.1 is_triggered_only = yes option = { name = orphan.1.a } }
country_event = { id = mixed.1 trigger = { OR = { tag = SHU has_war = yes } } option = { name = mixed.1.a } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestEventOwnershipGroupsACountrysEvents(t *testing.T) {
	ownership := countryEventsIndex(t).EventOwnership("GER")
	for id, want := range map[string]string{
		"ger.1": "country", "ger.2": "country", "trig.1": "country",
		"gen.1": "generic", "dec.1": "generic", "orphan.1": "generic", "mixed.1": "generic",
		"shu.1": "",
	} {
		if got := ownership.Group(id); got != want {
			t.Errorf("%s: group %q, want %q", id, got, want)
		}
	}
}

func TestSearchCountryEventsPageListsOwnFirst(t *testing.T) {
	page, total := countryEventsIndex(t).SearchCountryEventsPage("", 0, 100, "GER")
	if total != 7 || len(page) != 7 {
		t.Fatalf("total %d: %+v", total, page)
	}
	var own, generic []string
	for i, choice := range page {
		switch choice.Group {
		case "country":
			if len(generic) > 0 {
				t.Fatalf("own event %s after a generic one at %d", choice.ID, i)
			}
			own = append(own, choice.ID)
		case "generic":
			generic = append(generic, choice.ID)
		default:
			t.Fatalf("%s has no group", choice.ID)
		}
	}
	sort.Strings(own)
	if !reflect.DeepEqual(own, []string{"ger.1", "ger.2", "trig.1"}) {
		t.Fatalf("own %v", own)
	}
	if _, all := countryEventsIndex(t).SearchEventsPage("", 0, 100); all != 8 {
		t.Fatalf("the whole list %d", all)
	}
}

func TestDecisionsLeaveOutAnotherCountrysVisible(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/decisions/categories/cats.txt": []byte(`common_cat = { }
wei_cat = { visible = { tag = WEI } }`),
		"common/decisions/d.txt": []byte(`common_cat = {
	shu_only = { visible = { original_tag = SHU } }
	shu_or_war = { visible = { OR = { tag = SHU has_war = yes } } }
	anyone = { }
}
wei_cat = { wei_d = { } }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	names := func(tag string) []string {
		var out []string
		for _, category := range idx.DecisionsPayload(tag)["categories"].([]map[string]any) {
			for _, row := range category["decisions"].([]map[string]any) {
				out = append(out, row["name"].(string))
			}
		}
		sort.Strings(out)
		return out
	}
	if got := names("GER"); !reflect.DeepEqual(got, []string{"anyone", "shu_or_war"}) {
		t.Fatalf("GER sees %v", got)
	}
	if got := names("SHU"); !reflect.DeepEqual(got, []string{"anyone", "shu_only", "shu_or_war"}) {
		t.Fatalf("SHU sees %v", got)
	}
	if got := names("WEI"); !reflect.DeepEqual(got, []string{"anyone", "shu_or_war", "wei_d"}) {
		t.Fatalf("WEI sees %v", got)
	}
}
