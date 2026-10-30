package guirender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const taglandEvents = `
add_namespace = Tagland
add_namespace = tagland_news

country_event = {
	id = Tagland.1
	title = Tagland.1.t
	desc = Tagland.1.d
	picture = GFX_report_event_tagland_parade

	is_triggered_only = yes
	fire_only_once = yes

	option = {
		name = Tagland.1.a
		add_political_power = 75
	}
	option = {
		name = Tagland.1.b
		trigger = { tag = TWO }
	}
}

# Both fields are written as variants; only the installed-content ones can be
# decided here.
country_event = {
	id = Tagland.3
	title = {
		text = Tagland.3.t_dlc
		trigger = { has_dlc = "Man the Guns" }
	}
	title = {
		text = Tagland.3.t_base
	}
	desc = {
		text = Tagland.3.d_nodlc
		trigger = { NOT = { has_dlc = "Man the Guns" } }
	}
	desc = {
		text = Tagland.3.d_dlc
		trigger = { has_dlc = "Man the Guns" }
	}
	picture = GFX_report_event_tagland_parade

	option = { name = Tagland.3.a }
}

# Nothing here can be decided without a save game.
country_event = {
	id = Tagland.4
	title = Tagland.4.t
	desc = {
		text = Tagland.4.d_peace
		trigger = { NOT = { has_war_with = TWO } }
	}
	desc = {
		text = Tagland.4.d_war
		trigger = { has_war_with = TWO }
	}
	picture = GFX_report_event_tagland_parade

	option = { name = Tagland.4.a }
}

# A hidden event: no picture, no translation, and it fires two others.
country_event = {
	id = Tagland.5
	title = Tagland.5.t

	is_triggered_only = no
	hidden = yes

	immediate = {
		TWO = {
			country_event = { id = Tagland.900 days = 5 }
		}
		news_event = { id = tagland_news.900 hours = 6 }
	}

	option = { }
}

news_event = {
	id = tagland_news.1
	title = tagland_news.1.t
	desc = tagland_news.1.d
	picture = report_event_tagland_parade

	major = yes
	is_triggered_only = yes

	option = { name = tagland_news.1.a }
}

state_event = {
	id = tagland.2
	title = tagland.2.t
	picture = NOSUCHPICTURE

	option = { name = tagland.2.a }
}
`

const taglandPictures = `
spriteTypes = {
	spriteType = {
		name = "GFX_report_event_tagland_parade"
		texturefile = "gfx/event_pictures/report_event_tagland_parade.dds"
	}
}
`

const taglandLoc = `l_english:
 Tagland.1.t: "The Parade"
 Tagland.1.d: "The regiments march past."
 Tagland.1.a: "A fine sight."
 Tagland.1.b: "We were not invited."
 Tagland.3.t_dlc: "The Fleet Sails"
 Tagland.3.t_base: "The Army Marches"
 Tagland.3.d_nodlc: "We have no navy."
 Tagland.3.d_dlc: "The battleships put to sea."
 Tagland.4.d_peace: "The border is quiet."
 Tagland.4.d_war: "The border is burning."
 tagland_news.1.t: "Tagland Parades"
 tagland_news.1.d: "Wire services report a parade."
 tagland_news.1.a: "Noted."
 tagland.2.t: "A Province Stirs"
 tagland.2.a: "Send the police."
`

func eventIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"events/Tagland.txt":               []byte(taglandEvents),
		"interface/eventpictures.gfx":      []byte(taglandPictures),
		"localisation/english/tagland.yml": []byte(taglandLoc),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestEventPayloadCarriesTheWindowsThreeHops(t *testing.T) {
	payload := eventIndex(t).EventPayload("Tagland.1")
	if payload == nil {
		t.Fatal("Tagland.1 was not indexed")
	}
	if payload["kind"] != "country" || payload["namespace"] != "Tagland" {
		t.Errorf("kind/namespace = %v/%v", payload["kind"], payload["namespace"])
	}
	if payload["title"] != "The Parade" || payload["desc"] != "The regiments march past." {
		t.Errorf("title/desc = %v/%v, want the localised text", payload["title"], payload["desc"])
	}

	if payload["picture"] != "GFX_report_event_tagland_parade" {
		t.Errorf("picture = %v", payload["picture"])
	}
	if payload["triggered_only"] != true || payload["fire_only_once"] != true ||
		payload["hidden"] != false || payload["major"] != false {
		t.Errorf("flags = %v", payload)
	}
	options, _ := payload["options"].([]map[string]any)
	if len(options) != 2 {
		t.Fatalf("options = %v, want one per button the engine draws", options)
	}
	if options[0]["name"] != "Tagland.1.a" || options[0]["label"] != "A fine sight." {
		t.Errorf("first option = %v", options[0])
	}

	if options[1]["name"] != "Tagland.1.b" {
		t.Errorf("second option = %v", options[1])
	}
}

func TestEventTextTakesTheFirstDecidableVariant(t *testing.T) {
	idx := eventIndex(t)
	payload := idx.EventPayload("Tagland.3")

	if payload["title"] != "The Fleet Sails" {
		t.Errorf("title = %v, want the variant whose has_dlc holds", payload["title"])
	}

	if payload["desc"] != "The battleships put to sea." {
		t.Errorf("desc = %v, want the second variant", payload["desc"])
	}
}

func TestUndecidableVariantsFallBackToTheFirstDeclared(t *testing.T) {
	payload := eventIndex(t).EventPayload("Tagland.4")

	if payload["desc"] != "The border is quiet." {
		t.Errorf("desc = %v, want the first declared variant", payload["desc"])
	}
}

func TestFiredEventsAreNotDeclarations(t *testing.T) {
	idx := eventIndex(t)

	for _, id := range []string{"Tagland.900", "tagland_news.900"} {
		if payload := idx.EventPayload(id); payload != nil {
			t.Errorf("%s was indexed from an effect call: %v", id, payload)
		}
	}
}

func TestHiddenEventKeepsItsFlagsAndDropsTextItHasNot(t *testing.T) {
	payload := eventIndex(t).EventPayload("Tagland.5")
	if payload["hidden"] != true {
		t.Errorf("hidden = %v", payload["hidden"])
	}

	if payload["triggered_only"] != false {
		t.Errorf("triggered_only = %v", payload["triggered_only"])
	}

	if _, ok := payload["title"]; ok {
		t.Errorf("title = %v, want the key omitted", payload["title"])
	}
	if payload["picture"] != "" {
		t.Errorf("picture = %v, want empty for an event nobody sees", payload["picture"])
	}
	options, _ := payload["options"].([]map[string]any)
	if len(options) != 1 || options[0]["name"] != "" {
		t.Fatalf("options = %v, want the empty option kept so the count matches", options)
	}
	if _, ok := options[0]["label"]; ok {
		t.Errorf("a nameless option got a label: %v", options[0])
	}
}

func TestEventPictureRecoversTheMissingGFXPrefix(t *testing.T) {
	idx := eventIndex(t)

	if picture := idx.EventPayload("tagland_news.1")["picture"]; picture != "GFX_report_event_tagland_parade" {
		t.Errorf("picture = %v, want the declared sprite name", picture)
	}

	if picture := idx.EventPayload("tagland.2")["picture"]; picture != "NOSUCHPICTURE" {
		t.Errorf("picture = %v, want it left as written", picture)
	}
}

func TestListEventsFiltersByNamespaceAndKeepsDeclarationOrder(t *testing.T) {
	idx := eventIndex(t)
	all := idx.ListEvents("")
	if len(all) != 6 {
		t.Fatalf("ListEvents(\"\") = %d rows, want the six declarations", len(all))
	}
	order := []string{}
	for _, row := range all {
		order = append(order, row["id"].(string))
	}
	want := []string{"Tagland.1", "Tagland.3", "Tagland.4", "Tagland.5", "tagland_news.1", "tagland.2"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want the file's declaration order %v", order, want)
		}
	}

	chapter := idx.ListEvents("TAGLAND")
	if len(chapter) != 5 {
		t.Fatalf("Tagland namespace = %d rows, want five", len(chapter))
	}
	if news := idx.ListEvents("tagland_news"); len(news) != 1 || news[0]["kind"] != "news" {
		t.Errorf("tagland_news = %v", news)
	}
	if none := idx.ListEvents("no_such_namespace"); len(none) != 0 {
		t.Errorf("unknown namespace = %v, want no rows and no panic", none)
	}
}

func TestUnknownEventIsNotAnError(t *testing.T) {
	if payload := eventIndex(t).EventPayload("nobody.1"); payload != nil {
		t.Errorf("unknown id = %v, want nil so the route answers 404", payload)
	}
}

func TestModEventShadowsTheBaseGameOne(t *testing.T) {
	base := eventIndex(t)
	mod, err := NewIndexWithFallback(map[string][]byte{
		"events/Tagland.txt": []byte(`
add_namespace = Tagland
country_event = {
	id = Tagland.1
	title = Tagland.1.t
	picture = GFX_report_event_tagland_parade
	option = { name = Tagland.1.a }
}
`),
		"localisation/english/mod.yml": []byte("l_english:\n Tagland.1.t: \"The Rehearsal\"\n"),
	}, base)
	if err != nil {
		t.Fatalf("NewIndexWithFallback: %v", err)
	}

	payload := mod.EventPayload("Tagland.1")
	if options, _ := payload["options"].([]map[string]any); len(options) != 1 {
		t.Errorf("options = %v, want the mod's single button", options)
	}
	seen := map[string]int{}
	for _, row := range mod.ListEvents("") {
		seen[row["id"].(string)]++
	}
	if seen["Tagland.1"] != 1 {
		t.Errorf("Tagland.1 listed %d times", seen["Tagland.1"])
	}
	if seen["tagland_news.1"] != 1 {
		t.Errorf("the base game's other events were dropped: %v", seen)
	}

	if payload["title"] != "The Rehearsal" {
		t.Errorf("title = %v, want the mod's translation", payload["title"])
	}
}

func TestRealGameEventCorpus(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("HOI4_GAME_ROOT"))
	if root == "" {
		t.Skip("HOI4_GAME_ROOT unset; set it to a Hearts of Iron IV install to run the real-corpus gate")
	}
	files := map[string][]byte{}
	for _, sub := range []struct {
		dir string
		ext string
	}{{"events", ".txt"}, {"interface", ".gfx"}, {"localisation/english", ".yml"}} {
		dir := filepath.Join(root, filepath.FromSlash(sub.dir))
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", sub.dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), sub.ext) {
				continue
			}
			body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[sub.dir+"/"+entry.Name()] = body
		}
	}
	idx, err := NewIndex(files)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	all := idx.ListEvents("")

	if len(all) < 6000 || len(all) > 9000 {
		t.Fatalf("indexed %d events, want the declarations only (~6974)", len(all))
	}
	seen := map[string]bool{}
	for _, row := range all {
		id, _ := row["id"].(string)
		if id == "" {
			t.Fatalf("an event came back without an id: %v", row)
		}
		if seen[id] {
			t.Fatalf("%s listed twice", id)
		}
		seen[id] = true
	}

	anschluss := idx.EventPayload("germany.4")
	if anschluss == nil {
		t.Fatal("germany.4 missing")
	}
	if anschluss["title"] != "Anschluss" || anschluss["namespace"] != "germany" ||
		anschluss["kind"] != "country" || anschluss["triggered_only"] != true {
		t.Errorf("germany.4 = %v", anschluss)
	}
	if anschluss["picture"] != "GFX_report_event_german_speech" {
		t.Errorf("germany.4 picture = %v", anschluss["picture"])
	}
	options, _ := anschluss["options"].([]map[string]any)
	if len(options) == 0 || options[0]["label"] != "How can anyone say that Austria is not German?!" {
		t.Errorf("germany.4 options = %v", options)
	}

	munich, _ := idx.EventPayload("germany.40")["desc"].(string)
	if !strings.HasPrefix(munich, "German demands for the Sudetenland") {
		t.Errorf("germany.40 desc = %q", munich)
	}

	if picture := idx.EventPayload("AAT_norway_communist.7")["picture"]; picture != "GFX_report_event_NOR_royal_family_exiled" {
		t.Errorf("AAT_norway_communist.7 picture = %v", picture)
	}

	if picture := idx.EventPayload("news.10")["picture"]; picture != "SOMEPIC" {
		t.Errorf("news.10 picture = %v", picture)
	}
	if picture := idx.EventPayload("britain.14")["picture"]; picture != "[GetHitlerCroatianHandshakeEventPicture]" {
		t.Errorf("britain.14 picture = %v", picture)
	}

	swiss := idx.ListEvents("bba_switzerland")
	if len(swiss) == 0 {
		t.Fatal("BBA_Switzerland namespace came back empty")
	}
	for _, row := range swiss {
		if !strings.EqualFold(row["namespace"].(string), "BBA_Switzerland") {
			t.Fatalf("namespace filter leaked %v", row["id"])
		}
	}
	news := idx.ListEvents("NEWS")
	if len(news) == 0 || len(news) >= len(all) {
		t.Fatalf("news namespace = %d rows out of %d", len(news), len(all))
	}
}

func TestSearchEventsMatchesIdThenTitle(t *testing.T) {
	idx := eventIndex(t)
	ids := func(q string, limit int) []string {
		out := []string{}
		for _, choice := range idx.SearchEvents(q, limit) {
			out = append(out, choice.ID)
		}
		return out
	}
	if got := ids("tagland.", 10); strings.Join(got, ",") != "Tagland.1,Tagland.3,Tagland.4,Tagland.5,tagland.2" {
		t.Fatalf("prefix search = %v", got)
	}

	if got := ids("news", 10); strings.Join(got, ",") != "tagland_news.1" {
		t.Fatalf("contains search = %v", got)
	}

	parade := idx.SearchEvents("PARADE", 10)
	if len(parade) != 2 || parade[0].ID != "Tagland.1" || parade[0].Title != "The Parade" || parade[1].ID != "tagland_news.1" {
		t.Fatalf("title search = %+v", parade)
	}
	if got := ids("", 2); len(got) != 2 {
		t.Fatalf("limit ignored: %v", got)
	}
	if got := ids("no such thing", 10); len(got) != 0 {
		t.Fatalf("a miss returned %v", got)
	}
}

func TestSearchEventsPagePagesThroughEveryMatch(t *testing.T) {
	idx := eventIndex(t)
	all, total := idx.SearchEventsPage("tagland.", 0, 100)
	if total != 5 || len(all) != 5 {
		t.Fatalf("all = %d of %d", len(all), total)
	}
	first, total := idx.SearchEventsPage("tagland.", 0, 2)
	rest, restTotal := idx.SearchEventsPage("tagland.", 2, 2)
	last, _ := idx.SearchEventsPage("tagland.", 4, 2)
	paged := append(append(append([]EventChoice{}, first...), rest...), last...)
	if total != 5 || restTotal != 5 || len(paged) != 5 {
		t.Fatalf("pages = %d, totals %d/%d", len(paged), total, restTotal)
	}
	for i := range all {
		if paged[i].ID != all[i].ID {
			t.Fatalf("page order differs at %d: %s vs %s", i, paged[i].ID, all[i].ID)
		}
	}
	if beyond, total := idx.SearchEventsPage("tagland.", 50, 2); len(beyond) != 0 || total != 5 {
		t.Fatalf("past the end = %v of %d", beyond, total)
	}
}
