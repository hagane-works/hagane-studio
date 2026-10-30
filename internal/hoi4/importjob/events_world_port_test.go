package importjob

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/schema"
)

var worldTestMod = map[string]string{
	"events/tst.txt": `add_namespace = tst
country_event = {
	id = tst.1
	title = tst.1.t
	desc = tst.1.d
	picture = GFX_report_event_tst
	is_triggered_only = yes
	option = { name = tst.1.a }
}
add_namespace = germany_civil
country_event = { id = germany_civil.1 option = { name = ok } }`,

	"events/legacy_events.txt": `country_event = { id = 77 option = { name = ok } }`,
	"events/TST_se.txt": `add_namespace = TST_se
country_event = {
	id = TST_se.1
	immediate = { play_song = "tst_song" }
	option = { name = TST_se.1.a }
}`,
	"common/decisions/categories/tst_categories.txt": `tst_cat = {
	icon = GFX_cat_icon
	priority = 5
	allowed = { original_tag = TST }
}`,

	"common/decisions/tst_decisions.txt": `tst_cat = {
	tst_decision = {
		icon = GFX_decision_tst_icon
		cost = 25
		complete_effect = { add_political_power = 10 }
	}
	GER_bad = { cost = 1 }
}
other_cat = {
	tst_other_decision = { cost = 5 }
}`,
	"common/bookmarks/tst.txt": `bookmarks = {
	bookmark = {
		name = "TST_BM_NAME"
		desc = "TST_BM_DESC"
		date = 1936.1.1.12
		picture = "GFX_select_date_tst"
		default_country = "TST"
		default = yes
		TST = { history = "TST_HIST_DESC" ideology = neutrality }
	}
}`,
	"history/countries/TST - Test.txt": `capital = 1
set_research_slots = 3
set_stability = 0.5`,
	"history/states/1-Test.txt": `state = {
	id = 1
	name = "STATE_1"
	manpower = 42
	state_category = town
	provinces = { 1 2 }
	history = { owner = TST controller = TST add_core_of = TST buildings = { infrastructure = 2 } }
}`,
	"interface/tst.gfx": `spriteTypes = {
	spriteType = { name = "GFX_report_event_tst" texturefile = "gfx/event_pictures/tst.dds" }
	spriteType = { name = "GFX_cat_icon" texturefile = "gfx/interface/decisions/cat.dds" }
	spriteType = { name = "GFX_decision_tst_icon" texturefile = "gfx/interface/decisions/tst.dds" }
	spriteType = { name = "GFX_select_date_tst" texturefile = "gfx/interface/bookmarks/tst.dds" }
}`,
	"localisation/tst_l_english.yml": "l_english:\n" +
		" tst.1.t:0 \"Test Title\"\n" +
		" tst.1.d:0 \"Test Desc\"\n" +
		" tst_cat:0 \"Test Category\"\n" +
		" tst_cat_desc:0 \"Category Desc\"\n" +
		" tst_decision:0 \"Test Decision\"\n" +
		" TST_BM_NAME:0 \"Test Bookmark\"\n" +
		" TST_BM_DESC:0 \"Bookmark Desc\"\n" +
		" TST_SE_title_0001:0 \"Super Title\"\n",
	"localisation/simp_chinese/tst_l_simp_chinese.yml": "l_simp_chinese:\n" +
		" tst.1.t:0 \"测试标题\"\n" +
		" tst_cat:0 \"测试分类\"\n",
}

func writeWorldTestMod(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func newWorldTestProject(t *testing.T) *project.Project {
	t.Helper()
	p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func importWorld(t *testing.T, p *project.Project, root, policy string) (*session, EventsWorldCounts) {
	t.Helper()
	options := AllOptions()
	options.ConflictPolicy = policy
	s := newSession(context.Background(), root, options, p.Record())
	if _, err := s.stageLocalisation(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.stageGFX(); err != nil {
		t.Fatal(err)
	}
	counts, err := s.stageEventsWorld(EventsWorldStageOptions{ProjectID: p.ID(), ImportEvents: true, ImportDecisions: true, ImportBookmarks: true, ImportHistoryFiles: true, ImportStates: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		return s.mergeEventsWorld(tx, EventsWorldMergeOptions{ConflictPolicy: policy})
	}); err != nil {
		t.Fatal(err)
	}
	return s, counts
}

func worldRowsBy(t *testing.T, p *project.Project, table, column string) map[string]project.Row {
	t.Helper()
	rows, err := p.Rows(table)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]project.Row{}
	for _, row := range rows {
		key := jsonText(row[column])
		if _, dup := out[key]; dup {
			t.Fatalf("%s: two rows with %s=%s", table, column, key)
		}
		out[key] = row
	}
	return out
}

func worldInt(v any) int64 { return project.Int64(v) }

func TestEventsWorldPortImportsEveryFamily(t *testing.T) {
	root := writeWorldTestMod(t, worldTestMod)
	p := newWorldTestProject(t)
	s, counts := importWorld(t, p, root, "overwrite")

	if counts["event"] != 2 || counts["event_namespace"] != 1 || counts["super_event"] != 1 || counts["bookmark"] != 1 || counts["history"] != 1 || counts["state"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if counts["decision"] != 2 || counts["decision_category"] != 3 {
		t.Errorf("decision counts = %v (GER_bad must be filtered, both tst_cat records staged)", counts)
	}

	namespaces := worldRowsBy(t, p, "event_namespaces", "namespace")
	if len(namespaces) != 2 || namespaces["tst"] == nil || namespaces["legacy_events"] == nil {
		t.Fatalf("namespaces = %v", namespaces)
	}
	if ns := namespaces["tst"]; ns["country_tag"] != "TST" || ns["scope_kind"] != "own" || ns["shared_country_tags"] != nil {
		t.Errorf("namespace tst = %v", ns)
	}

	events := worldRowsBy(t, p, "events", "event_id")
	if events["germany_civil.1"] != nil {
		t.Error("foreign event imported")
	}
	ev := events["tst.1"]
	if ev == nil {
		t.Fatalf("events = %v", events)
	}
	if worldInt(ev["namespace_id"]) != worldInt(namespaces["tst"]["id"]) {
		t.Errorf("tst.1 namespace_id = %v", ev["namespace_id"])
	}
	if ev["title_en"] != "Test Title" || ev["title_zh"] != "测试标题" || ev["desc_en"] != "Test Desc" || ev["desc_zh"] != "" {
		t.Errorf("tst.1 texts = %v / %v / %v / %v", ev["title_en"], ev["title_zh"], ev["desc_en"], ev["desc_zh"])
	}
	if locs, _ := ev["display_name_locs"].(map[string]any); locs["en"] != "Test Title" || locs["zh"] != "测试标题" {
		t.Errorf("tst.1 display_name_locs = %v", ev["display_name_locs"])
	}
	if ev["picture"] != "GFX_report_event_tst" || ev["is_triggered_only"] != true || ev["source"] != "imported" || ev["source_file"] != "events/tst.txt" || ev["event_type"] != "country_event" {
		t.Errorf("tst.1 = %v", ev)
	}
	if ev["has_obs"] != false || ev["on_action_attach"] != "" || worldInt(ev["auto_fire_after_days"]) != 0 || ev["auto_fire_on_startup"] != false {
		t.Errorf("tst.1 insert-only columns = %v", ev)
	}
	if legacy := events["77"]; legacy == nil || worldInt(legacy["namespace_id"]) != worldInt(namespaces["legacy_events"]["id"]) {
		t.Errorf("event 77 = %v", legacy)
	}

	categories := worldRowsBy(t, p, "decision_categories", "category_id")
	cat := categories["tst_cat"]
	if cat == nil || cat["icon"] != "GFX_cat_icon" || worldInt(cat["priority"]) != 5 || !strings.Contains(jsonText(cat["allowed"]), "original_tag = TST") {
		t.Fatalf("tst_cat = %v (the definition must win over the bare block)", cat)
	}
	if cat["name_en"] != "Test Category" || cat["name_zh"] != "测试分类" || cat["desc_en"] != "Category Desc" || cat["visibility"] != "visible" {
		t.Errorf("tst_cat texts = %v", cat)
	}
	if other := categories["other_cat"]; other == nil || other["icon"] != "" || other["scope_kind"] == nil {
		t.Errorf("other_cat = %v", other)
	}

	decisions := worldRowsBy(t, p, "decisions", "decision_id")
	if decisions["GER_bad"] != nil {
		t.Error("foreign decision imported")
	}
	dec := decisions["tst_decision"]
	if dec == nil || worldInt(dec["category_id"]) != worldInt(cat["id"]) || worldInt(dec["cost"]) != 25 || dec["icon"] != "GFX_decision_tst_icon" {
		t.Fatalf("tst_decision = %v", dec)
	}
	if dec["name_en"] != "Test Decision" || !strings.Contains(jsonText(dec["complete_effect"]), "add_political_power") || dec["source_file"] != "common/decisions/tst_decisions.txt" {
		t.Errorf("tst_decision = %v", dec)
	}
	if other := decisions["tst_other_decision"]; other == nil || other["name_en"] != "tst_other_decision" || worldInt(other["category_id"]) != worldInt(categories["other_cat"]["id"]) {
		t.Errorf("tst_other_decision = %v", other)
	}
	if locs, _ := dec["effect_tooltip_locs"].(map[string]any); locs == nil || len(locs) != 0 {
		t.Errorf("effect_tooltip_locs = %v", dec["effect_tooltip_locs"])
	}

	supers, _ := p.Rows("super_events")
	if len(supers) != 1 {
		t.Fatalf("super_events = %v", supers)
	}
	se := supers[0]
	if se["country_tag"] != "TST" || worldInt(se["number"]) != 1 || se["title_en"] != "Super Title" || se["song_token"] != "tst_song" || se["source"] != "imported" {
		t.Errorf("super event = %v", se)
	}
	if q, _ := se["quote_locs"].(map[string]any); q == nil || len(q) != 0 {
		t.Errorf("quote_locs = %v", se["quote_locs"])
	}

	bookmarks := worldRowsBy(t, p, "bookmarks", "bookmark_id")
	bm := bookmarks["tst_bm"]
	if bm == nil || bm["name_en"] != "Test Bookmark" || bm["desc_en"] != "Bookmark Desc" || bm["default_country"] != "TST" || bm["default_flag"] != true || bm["picture"] != "GFX_select_date_tst" {
		t.Fatalf("bookmarks = %v", bookmarks)
	}
	if countries, _ := bm["selectable_countries"].([]any); len(countries) != 1 || worldInt(bm["edit_revision"]) != 1 {
		t.Errorf("bookmark = %v", bm)
	}

	history := worldRowsBy(t, p, "history_files", "country_tag")
	h := history["TST"]
	if h == nil || worldInt(h["capital"]) != 1 || worldInt(h["set_research_slots"]) != 3 || jsonText(h["set_stability"]) != "0.5" || h["source"] != "imported" {
		t.Fatalf("history = %v", history)
	}
	if tags, _ := h["add_to_faction_tags"].([]any); tags == nil {
		t.Errorf("add_to_faction_tags = %v", h["add_to_faction_tags"])
	}

	states := worldRowsBy(t, p, "states", "state_id")
	st := states["1"]
	if st == nil || st["name"] != "STATE_1" || worldInt(st["manpower"]) != 42 || st["owner"] != "TST" || st["controller"] != "TST" || st["source_file"] != "history/states/1-Test.txt" {
		t.Fatalf("states = %v", states)
	}
	snapshot, _ := st["source_snapshot"].(map[string]any)
	if snapshot["schema_version"] != "hagane.state-source-snapshot.v1" || worldInt(snapshot["state_id"]) != 1 || snapshot["source_path"] != "history/states/1-Test.txt" || snapshot["owner"] != "TST" {
		t.Errorf("source_snapshot = %v", snapshot)
	}
	if ids, _ := snapshot["province_ids"].([]any); len(ids) != 2 {
		t.Errorf("province_ids = %v", snapshot["province_ids"])
	}

	images := strings.Join(s.neededImageList(), ",")
	for _, want := range []string{"gfx/event_pictures/tst.dds", "gfx/interface/decisions/cat.dds", "gfx/interface/decisions/tst.dds", "gfx/interface/bookmarks/tst.dds"} {
		if !strings.Contains(images, want) {
			t.Errorf("needed images %s lack %s", images, want)
		}
	}
}

func TestEventsWorldPortReimportSkipAndOverwrite(t *testing.T) {
	root := writeWorldTestMod(t, worldTestMod)
	p := newWorldTestProject(t)
	importWorld(t, p, root, "overwrite")
	before := map[string]int{}
	for _, table := range []string{"event_namespaces", "events", "decision_categories", "decisions", "super_events", "bookmarks", "history_files", "states"} {
		rows, _ := p.Rows(table)
		before[table] = len(rows)
	}
	edit := func() {
		t.Helper()
		if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
			for _, change := range []struct{ table, column, value string }{
				{"events", "title_en", "edited"}, {"decisions", "icon", "edited"}, {"decision_categories", "icon", "edited"},
				{"bookmarks", "name_en", "edited"}, {"history_files", "set_oob", "edited"}, {"states", "name", "edited"},
				{"super_events", "song_token", "edited"},
			} {
				for _, row := range tx.Rows(change.table) {
					if _, err := tx.Update(change.table, project.Int64(row["id"]), func(r project.Row) { r[change.column] = change.value }); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(policy string, want func(imported string) string) {
		t.Helper()
		for _, c := range []struct{ table, key, id, column, imported string }{
			{"events", "event_id", "tst.1", "title_en", "Test Title"},
			{"decisions", "decision_id", "tst_decision", "icon", "GFX_decision_tst_icon"},
			{"decision_categories", "category_id", "tst_cat", "icon", "GFX_cat_icon"},
			{"bookmarks", "bookmark_id", "tst_bm", "name_en", "Test Bookmark"},
			{"history_files", "country_tag", "TST", "set_oob", ""},
			{"states", "state_id", "1", "name", "STATE_1"},
			{"super_events", "number", "1", "song_token", "tst_song"},
		} {
			rows := worldRowsBy(t, p, c.table, c.key)
			if len(rows) != before[c.table] {
				t.Errorf("%s after %s: %d rows, want %d", c.table, policy, len(rows), before[c.table])
			}
			if got := rows[c.id][c.column]; got != want(c.imported) {
				t.Errorf("%s %s.%s after %s = %v, want %v", c.table, c.id, c.column, policy, got, want(c.imported))
			}
		}
	}

	edit()
	importWorld(t, p, root, "skip")
	check("skip", func(string) string { return "edited" })
	importWorld(t, p, root, "overwrite")
	check("overwrite", func(imported string) string { return imported })
}

func TestEventsWorldPortCategoryRules(t *testing.T) {
	root := writeWorldTestMod(t, map[string]string{
		"localisation/x_l_english.yml": "l_english:\n lonely_cat:0 \"Lonely\"\n lonely_cat_desc:0 \"Lonely desc\"\n",
	})
	p := newWorldTestProject(t)
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		tx.Insert("decision_categories", project.Row{"category_id": "propaganda_efforts", "name_en": "Kept", "display_name_locs": map[string]any{"en": "Kept"}, "description_locs": map[string]any{}, "scope_kind": "global", "visibility": "hidden"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	s := newSession(context.Background(), root, AllOptions(), p.Record())
	if _, err := s.stageLocalisation(nil); err != nil {
		t.Fatal(err)
	}
	stage := func(kind, key, parent, source string, data payload) {
		s.world.rows = append(s.world.rows, worldRow{kind: kind, key: key, parent: parent, source: source, data: data})
	}
	stage("decision_category", "propaganda_efforts", "", `common\decisions\categories\00_decision_categories.txt`,
		payload{"category_id": "propaganda_efforts", "icon": "generic_propaganda", "priority": "5"})
	stage("decision_category", "propaganda_efforts", "", "common/decisions/_generic_decisions.txt",
		payload{"category_id": "propaganda_efforts"})
	stage("decision_category", "unused_cat", "", "common/decisions/categories/x.txt", payload{"icon": "x"})
	stage("decision", "war_propaganda", "propaganda_efforts", "common/decisions/_generic_decisions.txt",
		payload{"decision_id": "war_propaganda", "cost": "150"})
	stage("decision", "lonely_decision", "lonely_cat", "common/decisions/x.txt", payload{})
	if err := p.Write(schema.Tables(), func(tx *project.Tx) error {
		return s.mergeEventsWorld(tx, EventsWorldMergeOptions{ConflictPolicy: "overwrite"})
	}); err != nil {
		t.Fatal(err)
	}
	categories := worldRowsBy(t, p, "decision_categories", "category_id")
	cat := categories["propaganda_efforts"]
	if cat["icon"] != "generic_propaganda" || worldInt(cat["priority"]) != 5 {
		t.Errorf("category = icon %v priority %v, want the definition's generic_propaganda / 5", cat["icon"], cat["priority"])
	}
	if cat["name_en"] != "Kept" || jsonText(cat["display_name_locs"]) != `{"en":"Kept"}` || cat["visibility"] != "hidden" {
		t.Errorf("existing names/visibility overwritten: %v", cat)
	}
	if categories["unused_cat"] != nil {
		t.Error("a category no staged decision files under was imported")
	}
	lonely := categories["lonely_cat"]
	if lonely == nil || lonely["scope_kind"] != "global" || lonely["country_tag"] != "" || lonely["name_en"] != "Lonely" || lonely["desc_en"] != "Lonely desc" || lonely["icon"] != nil {
		t.Errorf("fallback category = %v", lonely)
	}
	decisions := worldRowsBy(t, p, "decisions", "decision_id")
	if d := decisions["war_propaganda"]; d == nil || worldInt(d["cost"]) != 150 || worldInt(d["category_id"]) != worldInt(cat["id"]) {
		t.Errorf("war_propaganda = %v", d)
	}
	if d := decisions["lonely_decision"]; d == nil || worldInt(d["category_id"]) != worldInt(lonely["id"]) {
		t.Errorf("lonely_decision = %v", d)
	}
}

func TestEventsWorldPortStageOptions(t *testing.T) {
	root := writeWorldTestMod(t, worldTestMod)
	p := newWorldTestProject(t)
	s := newSession(context.Background(), root, AllOptions(), p.Record())
	counts, err := s.stageEventsWorld(EventsWorldStageOptions{ProjectID: p.ID(), ImportStates: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 1 || counts["state"] != 1 || len(s.world.rows) != 1 {
		t.Errorf("counts = %v, rows = %d", counts, len(s.world.rows))
	}
}
