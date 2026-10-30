package exportcheck

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func testGame() fstest.MapFS {
	return fstest.MapFS{
		"common/country_tags/00_countries.txt": {Data: []byte("GER = \"countries/Germany.txt\"\nFRA = \"countries/France.txt\"\n")},
		"common/country_tags/zz_dynamic.txt":   {Data: []byte("dynamic_tags = yes\nD01 = \"countries/D01.txt\"\n")},
		"history/states/1-France.txt":          {Data: []byte("state = { id = 1 name = \"STATE_1\" history = { owner = FRA 1939.1.1 = { owner = GER } } provinces = { 1 } }\n")},
		"history/states/2-Germany.txt":         {Data: []byte("state = { id = 2 history = { owner = GER } provinces = { 2 } }\n")},
	}
}

func newTestServer(t *testing.T) (*project.Project, *http.ServeMux) {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return testGame(), nil },
	})
	return p, mux
}

func call(t *testing.T, mux *http.ServeMux, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status=%d want %d body=%s", rec.Code, status, rec.Body.String())
	}
}

func insert(t *testing.T, p *project.Project, table string, rows ...project.Row) []int64 {
	t.Helper()
	var ids []int64
	if err := p.Write([]string{table}, func(tx *project.Tx) error {
		for _, row := range rows {
			ids = append(ids, project.Int64(tx.Insert(table, row)["id"]))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return ids
}

func issueCodes(issues any) map[string]int {
	out := map[string]int{}
	list, _ := issues.([]any)
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out[m["code"].(string)]++
		}
	}
	return out
}

func TestCheckShapeOnEmptyProject(t *testing.T) {
	_, mux := newTestServer(t)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rec := call(t, mux, method, "/api/go/projects/1/export/check")
		wantStatus(t, rec, http.StatusOK)
		body := decodeBody(t, rec)
		for _, key := range []string{"ok", "blocker_count", "warning_count", "blockers", "warnings", "issues"} {
			if _, ok := body[key]; !ok {
				t.Fatalf("%s check misses %q: %v", method, key, body)
			}
		}
		if body["ok"] != true {
			t.Fatalf("empty project must pass the gate: %v", body)
		}
	}
	rec := call(t, mux, http.MethodDelete, "/api/go/projects/1/export/check")
	wantStatus(t, rec, http.StatusMethodNotAllowed)
}

func TestCheckSeededIssues(t *testing.T) {
	p, mux := newTestServer(t)
	insert(t, p, "technologies",
		project.Row{"tech_id": "tech_a", "prerequisites": []any{"tech_b"}, "source": "created"},
		project.Row{"tech_id": "tech_b", "prerequisites": []any{"tech_a"}, "source": "created"})
	insert(t, p, "ideas", project.Row{"idea_id": "TST_idea", "country_tag": "TST", "idea_type": "country", "source": "created"})
	group := insert(t, p, "oob_air_wing_groups", project.Row{"oob_file": "TST_1936", "country_tag": "TST", "location": "", "source": "created"})
	insert(t, p, "oob_air_wings", project.Row{"group_id": group[0], "equipment_type": "", "source": "created"})

	rec := call(t, mux, http.MethodPost, "/api/go/projects/1/export/check")
	wantStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	got := issueCodes(body["issues"])
	for _, code := range []string{"tech_circular_prereq", "idea_no_modifier", "oob_air_location_missing", "oob_air_equipment_type_missing"} {
		if got[code] == 0 {
			t.Fatalf("missing %s in %v", code, got)
		}
	}
	if body["ok"] != false {
		t.Fatalf("air wing blockers must fail the gate: %v", body["ok"])
	}
	blockers, _ := body["blockers"].([]any)
	if int(body["blocker_count"].(float64)) != len(blockers) || len(blockers) < 2 {
		t.Fatalf("blocker_count %v vs %d", body["blocker_count"], len(blockers))
	}
	for _, issue := range body["issues"].([]any) {
		if m := issue.(map[string]any); m["severity"] == "info" {
			t.Fatalf("/export/check drops info issues: %v", m)
		}
	}
}

func TestValidateShapeLevelsAndCounts(t *testing.T) {
	p, mux := newTestServer(t)
	insert(t, p, "ideas", project.Row{"idea_id": "TST_idea", "country_tag": "TST", "idea_type": "country", "source": "created"})
	tree := insert(t, p, "focus_trees", project.Row{"tree_id": "tst_focus", "country_tag": "TST", "tree_kind": "country", "source": "created"})
	insert(t, p, "focus_nodes",
		project.Row{"focus_tree_id": tree[0], "focus_id": "TST_a", "prerequisites": []any{}, "mutually_exclusive": []any{}},
		project.Row{"focus_tree_id": tree[0], "focus_id": "TST_b", "prerequisites": []any{[]any{"TST_a"}}, "mutually_exclusive": []any{}})

	rec := call(t, mux, http.MethodGet, "/api/go/projects/1/export/validate")
	wantStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	summary := body["summary"].(map[string]any)
	if summary["ideas"] != float64(1) || summary["focus_trees"] != float64(1) || summary["focus_nodes"] != float64(2) {
		t.Fatalf("summary counts: %v", summary)
	}
	if _, ok := summary["entity_obs_paths"]; ok {
		t.Fatal("non-content tables are not counted")
	}
	for _, issue := range body["issues"].([]any) {
		m := issue.(map[string]any)
		want := map[string]string{"blocker": "error", "warning": "warning"}[m["severity"].(string)]
		if want == "" {
			want = "info"
		}
		if m["level"] != want {
			t.Fatalf("level %v for severity %v", m["level"], m["severity"])
		}
	}
	audit := body["asset_audit"].(map[string]any)
	for _, key := range []string{"idea_pictures_missing", "event_pictures_missing", "decision_icons_missing",
		"tech_icons_missing", "focus_icons_missing", "portraits_missing", "total_missing"} {
		if _, ok := audit[key]; !ok {
			t.Fatalf("asset_audit misses %s: %v", key, audit)
		}
	}
	artifact := body["artifact"].(map[string]any)
	if artifact["status"] != "deferred" {
		t.Fatalf("artifact: %v", artifact)
	}
}

func TestValidateAssetAuditUsesBlobStore(t *testing.T) {
	p, mux := newTestServer(t)
	ids := insert(t, p, "decisions", project.Row{"decision_id": "TST_decide", "icon": "GFX_decision_TST_decide", "source": "created"})
	key := p.Blobs().EnvPrefix() + "/decisions/TST_decide.png"
	insert(t, p, "entity_obs_paths", project.Row{"entity_type": "decision_icon", "entity_id": ids[0], "obs_key": key})

	audit := func() map[string]any {
		rec := call(t, mux, http.MethodGet, "/api/go/projects/1/export/validate")
		wantStatus(t, rec, http.StatusOK)
		return decodeBody(t, rec)["asset_audit"].(map[string]any)
	}
	if got := audit(); got["decision_icons_missing"] != float64(1) || got["total_missing"] != float64(1) {
		t.Fatalf("a mapped icon without bytes is missing: %v", got)
	}
	if err := p.Blobs().Put(key, "image/png", 4, strings.NewReader("png!")); err != nil {
		t.Fatal(err)
	}
	if got := audit()["decision_icons_missing"]; got != float64(0) {
		t.Fatalf("a stored icon is not missing: %v", got)
	}
}

func TestValidateCompactPaging(t *testing.T) {
	p, mux := newTestServer(t)
	insert(t, p, "ideas",
		project.Row{"idea_id": "TST_a", "country_tag": "TST", "source": "created"},
		project.Row{"idea_id": "TST_b", "country_tag": "TST", "source": "created"})
	rec := call(t, mux, http.MethodGet, "/api/go/projects/1/export/validate?view=compact&limit=1")
	wantStatus(t, rec, http.StatusOK)
	first := decodeBody(t, rec)
	if first["view"] != "compact" || first["returned"] != float64(1) || first["done"] != false {
		t.Fatalf("first page: %v", first)
	}
	revision := first["revision"].(string)
	rec = call(t, mux, http.MethodGet, "/api/go/projects/1/export/validate?view=compact&limit=100&offset=1&revision="+url.QueryEscape(revision))
	wantStatus(t, rec, http.StatusOK)
	if decodeBody(t, rec)["done"] != true {
		t.Fatal("second page should finish")
	}
	rec = call(t, mux, http.MethodGet, "/api/go/projects/1/export/validate?view=compact&offset=1")
	wantStatus(t, rec, http.StatusUnprocessableEntity)
	if decodeBody(t, rec)["code"] != "invalid_validation_page" {
		t.Fatal("continuation without revision is refused")
	}
	rec = call(t, mux, http.MethodGet, "/api/go/projects/1/export/validate?view=compact&offset=1&revision=stale")
	wantStatus(t, rec, http.StatusConflict)
}

func TestBaseInstalls(t *testing.T) {
	p, mux := newTestServer(t)
	rec := call(t, mux, http.MethodGet, "/api/go/projects/1/export/base-installs")
	wantStatus(t, rec, http.StatusOK)
	if got := decodeBody(t, rec)["base_installs"].([]any); len(got) != 0 {
		t.Fatalf("no dependencies → no cards: %v", got)
	}
	if _, err := p.UpdateRecord(func(r project.Row) { r["dependencies"] = []any{" Kaiserreich ", "", 7, "Kaiserreich"} }); err != nil {
		t.Fatal(err)
	}
	rec = call(t, mux, http.MethodGet, "/api/go/projects/1/export/base-installs")
	wantStatus(t, rec, http.StatusOK)
	cards := decodeBody(t, rec)["base_installs"].([]any)
	if len(cards) != 2 {
		t.Fatalf("stripped strings keep order and duplicates: %v", cards)
	}
	card := cards[0].(map[string]any)
	if card["dependency"] != "Kaiserreich" || card["matched"] != false || card["display_name"] != "Kaiserreich" || card["mod_key"] != "" {
		t.Fatalf("unmatched card: %v", card)
	}
	wantStatus(t, call(t, mux, http.MethodPost, "/api/go/projects/1/export/base-installs"), http.StatusMethodNotAllowed)
}

func TestRunCheck(t *testing.T) {
	p, mux := newTestServer(t)
	rec := call(t, mux, http.MethodGet, "/api/go/run-check/projects/1")
	wantStatus(t, rec, http.StatusOK)
	if issues := decodeBody(t, rec)["issues"].([]any); len(issues) != 0 {
		t.Fatalf("empty project: %v", issues)
	}
	tree := insert(t, p, "focus_trees", project.Row{"tree_id": "tst_focus", "country_tag": "TST", "tree_kind": "country", "source": "created"})
	insert(t, p, "focus_nodes",
		project.Row{"focus_tree_id": tree[0], "focus_id": "TST_a", "prerequisites": []any{[]any{"TST_missing"}}},
		project.Row{"focus_tree_id": tree[0], "focus_id": "TST_a"})
	imported := insert(t, p, "focus_trees", project.Row{"tree_id": "imp", "tree_kind": "country", "source": "imported"})
	insert(t, p, "focus_nodes", project.Row{"focus_tree_id": imported[0], "focus_id": "IMP_a", "prerequisites": []any{[]any{"IMP_missing"}}})

	rec = call(t, mux, http.MethodGet, "/api/go/run-check/projects/1")
	wantStatus(t, rec, http.StatusOK)
	got := issueCodes(decodeBody(t, rec)["issues"])
	if got["focus_prereq_not_found"] != 1 || got["focus_id_duplicate"] != 1 {
		t.Fatalf("created tree findings only: %v", got)
	}
	rec = call(t, mux, http.MethodGet, "/api/go/run-check/projects/1?include_imported=true")
	if got := issueCodes(decodeBody(t, rec)["issues"]); got["focus_prereq_not_found"] != 2 {
		t.Fatalf("imported tree included on request: %v", got)
	}
}

func TestMissingProject(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return nil, project.ErrNotFound },
		Game: func() (fs.FS, error) { return testGame(), nil },
	})
	for _, path := range []string{"/api/go/projects/9/export/check", "/api/go/projects/9/export/validate", "/api/go/projects/9/export/base-installs"} {
		rec := call(t, mux, http.MethodGet, path)
		wantStatus(t, rec, http.StatusNotFound)
		if decodeBody(t, rec)["detail"] != "Project not found" {
			t.Fatalf("%s: %s", path, rec.Body.String())
		}
	}
	rec := call(t, mux, http.MethodGet, "/api/go/run-check/projects/9")
	wantStatus(t, rec, http.StatusNotFound)
}

func TestParseReplacePaths(t *testing.T) {
	got := parseReplacePaths(" common/ideas , ,history/countries,")
	if len(got) != 2 || got[0] != "common/ideas" || got[1] != "history/countries" {
		t.Fatalf("%q", got)
	}
}

func TestGameVanillaFacts(t *testing.T) {
	setGame(func() (fs.FS, error) { return testGame(), nil })
	d := exportGateVanillaOnce()
	if !d.tagSet["GER"] || !d.tagSet["FRA"] || d.tagSet["D01"] {
		t.Fatalf("country tags (dynamic excluded): %v", d.CountryTags)
	}
	if d.ownerByState[1] != "FRA" || d.ownerByState[2] != "GER" || len(d.StateIDs) != 2 {
		t.Fatalf("state owners from the undated history block: %v %v", d.ownerByState, d.StateIDs)
	}
}

func TestRunCheckAndGatePolitics(t *testing.T) {
	p, mux := newTestServer(t)
	if _, err := p.UpdateRecord(func(r project.Row) {
		r["ideology_registry"] = map[string]any{
			"groups":           []any{"techno"},
			"root_definitions": []any{map[string]any{"id": "techno", "name": "Technocracy"}},
		}
	}); err != nil {
		t.Fatal(err)
	}
	insert(t, p, "countries", project.Row{"tag": "TST", "name": "Test", "ideology": "techno"})
	rec := call(t, mux, http.MethodGet, "/api/go/run-check/projects/1")
	wantStatus(t, rec, http.StatusOK)
	if got := issueCodes(decodeBody(t, rec)["issues"]); got["politics_custom_root_no_random_subtype"] != 1 {
		t.Fatalf("custom root without a random subtype: %v", got)
	}
	rec = call(t, mux, http.MethodPost, "/api/go/projects/1/export/check")
	wantStatus(t, rec, http.StatusOK)
	if body := decodeBody(t, rec); body["ok"] != false {
		t.Fatalf("the politics blocker reaches the export gate too: %v", issueCodes(body["issues"]))
	}
}

func TestCheckOnBroadProject(t *testing.T) {
	p, mux := newTestServer(t)
	insert(t, p, "characters", project.Row{"name": "Leader", "tag": "TST_leader", "country_tag": "TST", "role_type": "country_leader",
		"ideology": "fascism_ideology", "traits": []any{"warmonger"}, "source": "created"})
	ns := insert(t, p, "event_namespaces", project.Row{"namespace": "tst"})
	insert(t, p, "events", project.Row{"namespace_id": ns[0], "event_id": "tst.1", "event_number": 1, "event_type": "country_event",
		"title": "t", "options": []any{map[string]any{"name": "tst.1.a", "effects": "add_political_power = 10"}}, "source": "created"})
	cat := insert(t, p, "decision_categories", project.Row{"category_id": "TST_cat", "source": "created"})
	insert(t, p, "decisions", project.Row{"category_id": cat[0], "decision_id": "TST_d", "complete_effect": "add_stability = 0.1", "source": "created"})
	insert(t, p, "states", project.Row{"state_id": 1, "name": "STATE_1", "owner": "TST", "provinces": []any{2, 3}, "source": "created"})
	insert(t, p, "history_files", project.Row{"country_tag": "TST", "capital": 1, "set_politics_ruling_party": "neutrality",
		"set_popularities": map[string]any{"neutrality": 100}, "source": "created"})
	insert(t, p, "bookmarks", project.Row{"bookmark_id": "TST_bm", "date": "1936.1.1.12", "default_country": "TST",
		"selectable_countries": []any{map[string]any{"tag": "TST", "ideology": "neutrality"}}, "source": "created"})
	insert(t, p, "project_raw_files", project.Row{"dest_path": "common/scripted_effects/tst.txt", "content": "tst_effect = { add_stability = 0.1 ", "source": "created"})
	for _, path := range []string{"/api/go/projects/1/export/check", "/api/go/projects/1/export/validate", "/api/go/run-check/projects/1"} {
		rec := call(t, mux, http.MethodGet, path)
		wantStatus(t, rec, http.StatusOK)
	}
	rec := call(t, mux, http.MethodGet, "/api/go/projects/1/export/check")
	if got := issueCodes(decodeBody(t, rec)["issues"]); len(got) == 0 {
		t.Fatal("the unbalanced raw script should be reported")
	}
}
