package countries

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func newProject(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newMux(p *project.Project) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return nil, errors.New("no game") },
	})
	return mux
}

func seed(t *testing.T, p *project.Project, table string, rows ...project.Row) []int64 {
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

type response struct {
	*httptest.ResponseRecorder
}

func (r response) json(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON (%d): %s", r.Code, r.Body.String())
	}
	return out
}

func call(mux *http.ServeMux, method, path, body string, headers ...string) response {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return response{rec}
}

func expectStatus(t *testing.T, r response, want int) {
	t.Helper()
	if r.Code != want {
		t.Fatalf("status %d, want %d: %s", r.Code, want, r.Body.String())
	}
}

func rows(t *testing.T, p *project.Project, table string) []project.Row {
	t.Helper()
	out, err := p.Rows(table)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func asJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func seedTwoCountries(t *testing.T, p *project.Project) (tst, other int64) {
	ids := seed(t, p, "countries",
		project.Row{"tag": "TST", "name": "Testland", "ideology": "neutrality", "color": []any{1, 2, 3}},
		project.Row{"tag": "OTH", "name": "Otherland", "ideology": "neutrality"},
	)
	return ids[0], ids[1]
}

func TestUpdateCountryRenamesEverywhere(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	seed(t, p, "characters", project.Row{"tag": "TST_leader", "country_tag": "TST"})
	seed(t, p, "focus_trees",
		project.Row{"tree_id": "tst_tree", "country_tag": "TST", "edit_revision": int64(3)},
		project.Row{"tree_id": "shared", "country_tag": "OTH", "shared_country_tags": []any{"TST", "NEW", "OTH"}},
		project.Row{"tree_id": "other", "country_tag": "OTH"},
	)
	seed(t, p, "states", project.Row{
		"state_id": int64(1), "owner": "TST", "controller": "TST",
		"add_core_of": []any{"TST", "OTH"}, "add_claim_by": []any{"OTH"},
		"set_province_controller": []any{map[string]any{"province": 5, "controller": "TST"}},
	})
	seed(t, p, "bookmarks", project.Row{
		"bookmark_id": "b", "default_country": "TST",
		"selectable_countries": []any{map[string]any{"tag": "TST", "minor": true}, map[string]any{"tag": "OTH"}},
	})
	seed(t, p, "equipments", project.Row{"country_tag": "TST"})
	seed(t, p, "oob_ships", project.Row{"owner": "TST"})
	mux := newMux(p)

	r := call(mux, "PUT", "/api/go/projects/1/countries/1", `{"tag":"NEW","name":"Newland"}`,
		"X-Edit-Revisions", `{"focus_tree:1":"3"}`)
	expectStatus(t, r, http.StatusOK)
	body := r.json(t)
	if body["tag"] != "NEW" || body["name"] != "Newland" || body["warning"] != nil {
		t.Fatalf("response = %v", body)
	}
	var revisions map[string]string
	_ = json.Unmarshal([]byte(r.Header().Get("X-Edit-Revisions")), &revisions)
	want := map[string]string{"focus_tree:1": "4", "focus_tree:2": "2", "bookmark:1": "2"}
	if !reflect.DeepEqual(revisions, want) {
		t.Fatalf("X-Edit-Revisions = %v, want %v", revisions, want)
	}
	if got := rows(t, p, "characters")[0]["country_tag"]; got != "NEW" {
		t.Fatalf("character tag = %v", got)
	}
	focus := rows(t, p, "focus_trees")
	if focus[0]["country_tag"] != "NEW" || asJSON(focus[1]["shared_country_tags"]) != `["NEW","OTH"]` {
		t.Fatalf("focus trees = %v / %v", focus[0]["country_tag"], focus[1]["shared_country_tags"])
	}
	if project.Int64(focus[2]["edit_revision"]) > 1 {
		t.Fatal("an untouched tree advanced its revision")
	}
	state := rows(t, p, "states")[0]
	if state["owner"] != "NEW" || state["controller"] != "NEW" || asJSON(state["add_core_of"]) != `["NEW","OTH"]` ||
		asJSON(state["set_province_controller"]) != `[{"controller":"NEW","province":5}]` {
		t.Fatalf("state = %v", state)
	}
	bookmark := rows(t, p, "bookmarks")[0]
	if bookmark["default_country"] != "NEW" || asJSON(bookmark["selectable_countries"]) != `[{"minor":true,"tag":"NEW"},{"tag":"OTH"}]` {
		t.Fatalf("bookmark = %v", bookmark)
	}
	if rows(t, p, "equipments")[0]["country_tag"] != "NEW" || rows(t, p, "oob_ships")[0]["owner"] != "NEW" {
		t.Fatal("rename-only columns did not follow")
	}
	record := p.Record()
	if record["country_tag"] != "NEW" || asJSON(record["country_tags"]) != `["NEW","OTH"]` || record["country_name"] != "Newland" {
		t.Fatalf("project record = %v %v %v", record["country_tag"], record["country_tags"], record["country_name"])
	}
}

func TestUpdateCountryErrors(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	seed(t, p, "focus_trees", project.Row{"tree_id": "t", "country_tag": "TST", "edit_revision": int64(2)})
	mux := newMux(p)

	r := call(mux, "PUT", "/api/go/projects/1/countries/1", `{"tag":"OTH"}`)
	expectStatus(t, r, http.StatusConflict)
	if r.json(t)["detail"] != "Country OTH already exists in this project" {
		t.Fatalf("detail = %v", r.json(t)["detail"])
	}
	r = call(mux, "PUT", "/api/go/projects/1/countries/1", `{"tag":"ab"}`)
	expectStatus(t, r, http.StatusUnprocessableEntity)
	if !strings.HasPrefix(r.json(t)["detail"].(string), "tag must match") {
		t.Fatalf("detail = %v", r.json(t)["detail"])
	}
	expectStatus(t, call(mux, "PUT", "/api/go/projects/1/countries/99", `{"name":"x"}`), http.StatusNotFound)
	expectStatus(t, call(mux, "PUT", "/api/go/projects/1/countries/abc", `{"name":"x"}`), http.StatusNotFound)
	expectStatus(t, call(mux, "PUT", "/api/go/projects/1/countries/1", `{"name":`), http.StatusUnprocessableEntity)
	expectStatus(t, call(mux, "PUT", "/api/go/projects/1/countries/1", `{"name":null}`), http.StatusServiceUnavailable)

	r = call(mux, "PUT", "/api/go/projects/1/countries/1", `{"tag":"NEW"}`, "X-Edit-Revisions", `{"focus_tree:1":"1"}`)
	expectStatus(t, r, http.StatusConflict)
	if code := r.json(t)["detail"].(map[string]any)["code"]; code != "save_conflict" {
		t.Fatalf("code = %v", code)
	}
	if rows(t, p, "countries")[0]["tag"] != "TST" || rows(t, p, "focus_trees")[0]["country_tag"] != "TST" {
		t.Fatal("a refused rename changed data")
	}
	expectStatus(t, call(mux, "PUT", "/api/go/projects/1/countries/1", `{}`, "X-Edit-Revisions", `nope`), http.StatusUnprocessableEntity)
}

func TestUpdateCountryNormalizesLocs(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	mux := newMux(p)
	r := call(mux, "PUT", "/api/go/projects/1/countries/1",
		`{"adjective_locs":{"en":" Testish ","fr":"Testois"},"ideology_name_locs":{"fascism":{"en":"Reich"}},"color":[9,8,7],"use_legacy_ai_pp_spend":true}`)
	expectStatus(t, r, http.StatusOK)
	body := r.json(t)
	if body["adjective_en"] != "Testish" || body["name_fascism_en"] != "Reich" || asJSON(body["color"]) != "[9,8,7]" || body["use_legacy_ai_pp_spend"] != true {
		t.Fatalf("response = %v", body)
	}
	if asJSON(body["adjective_locs"]) != `{"en":"Testish","fr":"Testois"}` {
		t.Fatalf("adjective_locs = %v", body["adjective_locs"])
	}

	r = call(mux, "PUT", "/api/go/projects/1/countries/1", `{"name_en":"Test Land","color":null}`)
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); body["name_en"] != "Test Land" || asJSON(body["color"]) != "[]" || body["adjective_en"] != "Testish" {
		t.Fatalf("response = %v", body)
	}
}

func TestDeletePreviewAndDelete(t *testing.T) {
	p := newProject(t)
	seed(t, p, "countries", project.Row{"tag": "ZZX", "name": "Zed"}, project.Row{"tag": "OTH", "name": "Other"})
	trees := seed(t, p, "focus_trees", project.Row{"tree_id": "z", "country_tag": "ZZX"}, project.Row{"tree_id": "o", "country_tag": "OTH"})
	seed(t, p, "focus_nodes", project.Row{"focus_tree_id": trees[0], "focus_id": "a"}, project.Row{"focus_tree_id": trees[1], "focus_id": "b"})
	ns := seed(t, p, "event_namespaces", project.Row{"namespace": "zzx", "country_tag": "ZZX"})
	seed(t, p, "events", project.Row{"event_id": "zzx.1", "namespace_id": ns[0], "country_tag": "OTH"})
	seed(t, p, "ideas", project.Row{"idea_id": "zi", "country_tag": "ZZX"}, project.Row{"idea_id": "oi", "country_tag": "OTH"})
	seed(t, p, "states", project.Row{"state_id": int64(1), "owner": "zzx"})
	mux := newMux(p)

	r := call(mux, "GET", "/api/go/projects/1/countries/1/delete-preview", "")
	expectStatus(t, r, http.StatusOK)
	if !strings.Contains(r.Body.String(), `"cascade_counts":{"characters":0,"focus_trees":1,"ideas":1,"custom_traits":0,"event_namespaces":1,"events":0,`) {
		t.Fatalf("cascade_counts order/values: %s", r.Body.String())
	}
	body := r.json(t)
	if body["total"] != 3.0 || body["owned_states"] != 1.0 || body["vanilla_tag"] != false || body["will_be_last_country"] != false || body["tag"] != "ZZX" {
		t.Fatalf("preview = %v", body)
	}

	r = call(mux, "DELETE", "/api/go/projects/1/countries/1", "")
	expectStatus(t, r, http.StatusConflict)
	if body := r.json(t); body["code"] != "country_owns_states" || body["owned_states"] != 1.0 {
		t.Fatalf("body = %v", body)
	}
	if err := p.Write([]string{"states"}, func(tx *project.Tx) error {
		tx.Delete("states", func(project.Row) bool { return true })
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r = call(mux, "DELETE", "/api/go/projects/1/countries/1", "")
	expectStatus(t, r, http.StatusNoContent)
	if len(rows(t, p, "countries")) != 1 || len(rows(t, p, "focus_trees")) != 1 || len(rows(t, p, "focus_nodes")) != 1 ||
		len(rows(t, p, "events")) != 0 || len(rows(t, p, "ideas")) != 1 {
		t.Fatal("cascade did not remove exactly the country's rows")
	}
	if rec := p.Record(); rec["country_tag"] != "OTH" || asJSON(rec["country_tags"]) != `["OTH"]` {
		t.Fatalf("project = %v %v", rec["country_tag"], rec["country_tags"])
	}
	r = call(mux, "DELETE", "/api/go/projects/1/countries/2", "")
	expectStatus(t, r, http.StatusBadRequest)
	if r.json(t)["detail"] != "Cannot delete the last country from a project" {
		t.Fatalf("detail = %v", r.json(t)["detail"])
	}
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/countries/7/delete-preview", ""), http.StatusNotFound)
}

func TestDeleteVanillaCountryKeepsStates(t *testing.T) {
	p := newProject(t)
	seed(t, p, "countries", project.Row{"tag": "GER", "name": "Germany"}, project.Row{"tag": "OTH", "name": "Other"})
	seed(t, p, "states", project.Row{"state_id": int64(64), "owner": "GER"})
	mux := newMux(p)
	r := call(mux, "GET", "/api/go/projects/1/countries/1/delete-preview", "")
	if body := r.json(t); body["vanilla_tag"] != true {
		t.Fatalf("GER should be a vanilla tag: %v", body)
	}
	expectStatus(t, call(mux, "DELETE", "/api/go/projects/1/countries/1", ""), http.StatusNoContent)
	if len(rows(t, p, "states")) != 1 {
		t.Fatal("states must stay")
	}
}

func TestFlipAll(t *testing.T) {
	p := newProject(t)
	seed(t, p, "characters", project.Row{"tag": "a", "source": "imported"}, project.Row{"tag": "b", "source": "created"})
	seed(t, p, "focus_trees", project.Row{"tree_id": "t", "source": "imported"})
	seed(t, p, "project_raw_files", project.Row{"dest_path": "x.txt", "source": "imported"})
	mux := newMux(p)

	r := call(mux, "POST", "/api/go/projects/1/flip-all", `{"dry_run":true}`)
	expectStatus(t, r, http.StatusOK)
	if !strings.HasPrefix(r.Body.String(), `{"country_tag":"TST","dry_run":true,"flipped":{"characters":1,"technologies":0,"ideas":0,"focus_trees":1,`) {
		t.Fatalf("body = %s", r.Body.String())
	}
	if body := r.json(t); body["total"] != 3.0 {
		t.Fatalf("total = %v", body["total"])
	}
	if rows(t, p, "characters")[0]["source"] != "imported" {
		t.Fatal("dry run changed data")
	}
	r = call(mux, "POST", "/api/go/projects/1/flip-all", "")
	expectStatus(t, r, http.StatusOK)
	if r.Header().Get("X-Edit-Revisions") != `{"focus_tree:1":"2"}` {
		t.Fatalf("revisions = %q", r.Header().Get("X-Edit-Revisions"))
	}
	for _, table := range []string{"characters", "focus_trees", "project_raw_files"} {
		if rows(t, p, table)[0]["source"] != "created" {
			t.Fatalf("%s not flipped", table)
		}
	}
	expectStatus(t, call(mux, "POST", "/api/go/projects/1/flip-all", `[`), http.StatusUnprocessableEntity)
}

func TestVocabulary(t *testing.T) {
	p := newProject(t)
	seed(t, p, "characters",
		project.Row{"tag": "a", "source": "imported", "advisor_slot": "political_advisor"},
		project.Row{"tag": "b", "source": "imported", "advisor_slot": "high_command"},
		project.Row{"tag": "c", "source": "created", "advisor_slot": "theorist"},
		project.Row{"tag": "d", "source": "imported", "advisor_slot": "political_advisor"},
	)
	seed(t, p, "ideas", project.Row{"idea_id": "x", "source": "imported", "idea_type": "country"})
	r := call(newMux(p), "GET", "/api/go/projects/1/base-mod-vocabulary", "")
	expectStatus(t, r, http.StatusOK)
	if r.Body.String() != `{"advisor_slots":["high_command","political_advisor"],"idea_types":["country"]}` {
		t.Fatalf("body = %q", r.Body.String())
	}
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func upload(mux *http.ServeMux, path string, data []byte) response {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "image.png")
	_, _ = part.Write(data)
	_ = writer.Close()
	req := httptest.NewRequest("POST", path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return response{rec}
}

func TestCover(t *testing.T) {
	p := newProject(t)
	mux := newMux(p)
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/cover", ""), http.StatusNotFound)
	r := upload(mux, "/api/go/projects/1/cover", pngBytes(t, 300, 300))
	expectStatus(t, r, http.StatusOK)
	body := r.json(t)
	if body["filename"] != "thumbnail.png" || !strings.HasPrefix(body["preview_url"].(string), "/api/go/projects/1/cover?v=") {
		t.Fatalf("body = %v", body)
	}
	if p.Record()["picture"] != "thumbnail.png" {
		t.Fatal("picture not set")
	}
	r = call(mux, "GET", "/api/go/projects/1/cover", "")
	expectStatus(t, r, http.StatusOK)
	if r.Header().Get("Content-Type") != "image/png" || r.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("headers = %v", r.Header())
	}
	expectStatus(t, call(mux, "DELETE", "/api/go/projects/1/cover", ""), http.StatusNoContent)
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/cover", ""), http.StatusNotFound)
	if p.Record()["picture"] != "" {
		t.Fatal("picture not cleared")
	}
	r = upload(mux, "/api/go/projects/1/cover", []byte("not an image"))
	expectStatus(t, r, http.StatusUnprocessableEntity)
}

func TestCountryFlags(t *testing.T) {
	p := newProject(t)
	seedTwoCountries(t, p)
	mux := newMux(p)
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/countries/1/flag", ""), http.StatusNotFound)
	r := upload(mux, "/api/go/projects/1/countries/1/upload-flag", pngBytes(t, 164, 104))
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); body["flag_image_path"] != "flags/TST.tga" || body["tag"] != "TST" {
		t.Fatalf("body = %v", body)
	}
	r = call(mux, "GET", "/api/go/projects/1/countries/1/flag", "")
	expectStatus(t, r, http.StatusOK)
	img, err := png.Decode(bytes.NewReader(r.Body.Bytes()))
	if err != nil || img.Bounds().Dx() != 82 || img.Bounds().Dy() != 52 {
		t.Fatalf("flag is not the 82x52 rendition: %v %v", err, img)
	}
	expectStatus(t, upload(mux, "/api/go/projects/1/countries/9/upload-flag", pngBytes(t, 10, 10)), http.StatusNotFound)

	expectStatus(t, call(mux, "GET", "/api/go/projects/1/countries/1/flag?ideology=fascism", ""), http.StatusNotFound)
	r = upload(mux, "/api/go/projects/1/countries/1/upload-flag?ideology=Fascism", pngBytes(t, 82, 52))
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); asJSON(body["flag_paths_by_ideology"]) != `{"fascism":"flags/TST_fascism.tga"}` {
		t.Fatalf("body = %v", body)
	}
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/countries/1/flag?ideology=fascism", ""), http.StatusOK)
	r = upload(mux, "/api/go/projects/1/countries/1/upload-flag?ideology=unknown_root", pngBytes(t, 82, 52))
	expectStatus(t, r, http.StatusBadRequest)
}

func TestIdeologyIcons(t *testing.T) {
	p := newProject(t)
	mux := newMux(p)
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/ideology-icons/fascism", ""), http.StatusNotFound)
	r := upload(mux, "/api/go/projects/1/ideology-icons/fascism/upload", pngBytes(t, 40, 40))
	expectStatus(t, r, http.StatusOK)
	if body := r.json(t); asJSON(body["ideology_icon_paths"]) != `{"fascism":"ideology_icons/fascism.dds"}` {
		t.Fatalf("body = %v", body["ideology_icon_paths"])
	}
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/ideology-icons/fascism", ""), http.StatusOK)
	expectStatus(t, call(mux, "GET", "/api/go/projects/1/ideology-icons/nope", ""), http.StatusBadRequest)
}
