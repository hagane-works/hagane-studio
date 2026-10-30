package events

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
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

type harness struct {
	t    *testing.T
	p    *project.Project
	mux  *http.ServeMux
	root string
}

func newHarness(t *testing.T, game fs.FS) *harness {
	t.Helper()
	root := t.TempDir()
	store := project.NewStore(root)
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	h := withProject(t, p, game)
	h.root = root
	return h
}

func withProject(t *testing.T, p *project.Project, game fs.FS) *harness {
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) {
			if game == nil {
				return nil, errors.New("no game")
			}
			return game, nil
		},
	})
	return &harness{t: t, p: p, mux: mux}
}

func TestEventsReadBackFromDisk(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("france")
	h.json(http.MethodPut, "/events/namespaces/"+ns, `{"shared_country_tags":["GER","ITA"]}`, http.StatusOK)
	id := jsonID(h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.4","event_number":4,
		"title":[{"text":"france.4.t.a","trigger":"tag = FRA"}],"timeout_days":12,"is_triggered_only":false,
		"options":[{"name":"Fine","ai_chance":{"base":10}}],"display_name_locs":{"zh":"标题"},"shared_country_tags":["GER"]}`, http.StatusCreated))
	paths := []string{"/events/" + id, "/events/namespaces", "/events?namespace_id=" + ns, "/events/" + id + "/preview"}
	before := map[string]string{}
	for _, path := range paths {
		before[path] = h.do(http.MethodGet, path, "").Body.String()
	}
	reopened, err := project.NewStore(h.root).Get(h.p.ID())
	if err != nil {
		t.Fatal(err)
	}
	again := withProject(t, reopened, nil)
	for _, path := range paths {
		if got := again.do(http.MethodGet, path, "").Body.String(); got != before[path] {
			t.Fatalf("%s changed after reopening:\n%s\n%s", path, before[path], got)
		}
	}
}

func (h *harness) do(method, path, body string) *httptest.ResponseRecorder {
	h.t.Helper()
	req := httptest.NewRequest(method, "/api/go/projects/1"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func (h *harness) json(method, path, body string, status int) map[string]any {
	h.t.Helper()
	rec := h.do(method, path, body)
	if rec.Code != status {
		h.t.Fatalf("%s %s: status %d, want %d: %s", method, path, rec.Code, status, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		h.t.Fatalf("%s %s: %v: %s", method, path, err, rec.Body.String())
	}
	return out
}

func (h *harness) list(path string) []map[string]any {
	h.t.Helper()
	rec := h.do(http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		h.t.Fatalf("GET %s: status %d: %s", path, rec.Code, rec.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		h.t.Fatalf("GET %s: %v: %s", path, err, rec.Body.String())
	}
	return out
}

func (h *harness) expect(method, path, body string, status int, contains string) {
	h.t.Helper()
	rec := h.do(method, path, body)
	if rec.Code != status || !strings.Contains(rec.Body.String(), contains) {
		h.t.Fatalf("%s %s: status %d body %s; want %d containing %q", method, path, rec.Code, rec.Body.String(), status, contains)
	}
}

func (h *harness) namespace(name string) string {
	h.t.Helper()
	ns := h.json(http.MethodPost, "/events/namespaces", `{"namespace":"`+name+`"}`, http.StatusCreated)
	return jsonID(ns)
}

func jsonID(row map[string]any) string { return jsonString(row["id"]) }

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), `"`)
}

func TestNamespaceLifecycle(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.json(http.MethodPost, "/events/namespaces", `{"namespace":"  france ","country_tag":"fra"}`, http.StatusCreated)
	if ns["namespace"] != "france" || ns["country_tag"] != "FRA" || ns["scope_kind"] != "own" ||
		ns["event_count"] != float64(0) || ns["shared_country_tags"] != nil || ns["project_id"] != float64(1) {
		t.Fatalf("created namespace: %v", ns)
	}
	def := h.json(http.MethodPost, "/events/namespaces", `{"namespace":"other"}`, http.StatusCreated)
	if def["country_tag"] != "TST" {
		t.Fatalf("country tag should default to the project's: %v", def)
	}
	h.expect(http.MethodPost, "/events/namespaces", `{"namespace":"france"}`, http.StatusConflict, "namespace 'france' already exists in this project. Pick a unique name.")
	h.expect(http.MethodPost, "/events/namespaces", `{"namespace":"  "}`, http.StatusBadRequest, "namespace cannot be empty")
	h.expect(http.MethodPost, "/events/namespaces", `not json`, http.StatusUnprocessableEntity, "Invalid namespace payload")

	id := jsonID(ns)
	h.json(http.MethodPost, "/events", `{"namespace_id":`+id+`,"event_id":"france.1","event_number":1}`, http.StatusCreated)
	if got := h.list("/events/namespaces"); len(got) != 2 || got[0]["event_count"] != float64(1) || got[1]["event_count"] != float64(0) {
		t.Fatalf("list: %v", got)
	}
	if got := h.list("/events/namespaces?country_tag=FRA"); len(got) != 1 || got[0]["namespace"] != "france" {
		t.Fatalf("country filter: %v", got)
	}

	updated := h.json(http.MethodPut, "/events/namespaces/"+id, `{"namespace":" gaul ","country_tag":" ger ","shared_country_tags":["ITA"]}`, http.StatusOK)
	if updated["namespace"] != "gaul" || updated["country_tag"] != "GER" || jsonString(updated["shared_country_tags"]) != `["ITA"]` || updated["event_count"] != float64(1) {
		t.Fatalf("updated namespace: %v", updated)
	}
	h.expect(http.MethodPut, "/events/namespaces/"+id, `{"namespace":"other"}`, http.StatusConflict, "namespace 'other' already exists")
	h.expect(http.MethodPut, "/events/namespaces/999", `{}`, http.StatusNotFound, "Namespace not found")

	h.expect(http.MethodDelete, "/events/namespaces/"+id, "", http.StatusNoContent, "")
	if events := h.list("/events"); len(events) != 0 {
		t.Fatalf("deleting a namespace must delete its events: %v", events)
	}
	h.expect(http.MethodDelete, "/events/namespaces/"+id, "", http.StatusNotFound, "Namespace not found")
}

func TestEventCreateListAndFilters(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("france")
	other := h.namespace("italy")

	h.expect(http.MethodPost, "/events", `{"event_id":"france.1","event_number":1}`, http.StatusUnprocessableEntity, "namespace_id is required")
	h.expect(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_number":1}`, http.StatusUnprocessableEntity, "event_id is required")
	h.expect(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.1"}`, http.StatusUnprocessableEntity, "event_number is required")
	h.expect(http.MethodPost, "/events", `{"namespace_id":999,"event_id":"france.1","event_number":1}`, http.StatusBadRequest, "Namespace not found in this project")
	h.expect(http.MethodPost, "/events", `{"namespace_id":"x"}`, http.StatusUnprocessableEntity, "Invalid event payload")

	ev := h.json(http.MethodPost, "/events", `{
		"namespace_id":`+ns+`,"event_id":" france.2 ","event_number":2,
		"title":"A title","description":"A description",
		"immediate":"add_stability = 0.1\\nadd_war_support = 0.1",
		"options":[{"name":"Take it","effects":"a = yes\\nb = yes"},{"name":"france.2.b"}]
	}`, http.StatusCreated)
	if ev["event_id"] != "france.2" || ev["event_type"] != "country_event" || ev["is_triggered_only"] != true ||
		ev["country_tag"] != "TST" || ev["source"] != "created" || ev["scope_kind"] != "own" ||
		ev["title_en"] != "A title" || ev["desc_en"] != "A description" || jsonString(ev["warnings"]) != "[]" ||
		jsonString(ev["raw_extras"]) != "{}" || ev["immediate"] != "add_stability = 0.1\nadd_war_support = 0.1" {
		t.Fatalf("created event: %v", ev)
	}
	options := ev["options"].([]any)
	first, second := options[0].(map[string]any), options[1].(map[string]any)
	if first["name_en"] != "Take it" || first["effects"] != "a = yes\nb = yes" || jsonString(first["display_name_locs"]) != `{"en":"Take it"}` {
		t.Fatalf("option text was not normalised: %v", first)
	}
	if _, ok := second["name_en"]; ok {
		t.Fatalf("a key-shaped option name must not become text: %v", second)
	}

	dup := h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.2","event_number":2}`, http.StatusCreated)
	if dup["event_id"] != "france.3" || dup["event_number"] != float64(3) ||
		jsonString(dup["warnings"]) != `["event_id 'france.2' collided · auto-suffixed to 'france.3'."]` {
		t.Fatalf("collision: %v", dup)
	}
	h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.1","event_number":1}`, http.StatusCreated)
	h.json(http.MethodPost, "/events", `{"namespace_id":`+other+`,"event_id":"italy.1","event_number":1,"country_tag":"ita"}`, http.StatusCreated)

	all := h.list("/events?namespace_id=" + ns)
	if len(all) != 3 || all[0]["event_id"] != "france.1" || all[1]["event_id"] != "france.2" || all[2]["event_id"] != "france.3" {
		t.Fatalf("namespace list must be ordered by event number: %v", all)
	}
	if got := h.list("/events?country_tag=ITA"); len(got) != 1 || got[0]["event_id"] != "italy.1" {
		t.Fatalf("country filter: %v", got)
	}
	if got := h.list("/events?event_id=%20france.3%20"); len(got) != 1 || got[0]["event_id"] != "france.3" {
		t.Fatalf("event_id lookup: %v", got)
	}
	if got := h.list("/events?event_id=nope.1"); len(got) != 0 {
		t.Fatalf("missing event_id lookup must be []: %v", got)
	}
	if got := h.list("/events"); len(got) != 4 {
		t.Fatalf("all events: %v", got)
	}

	h.expect(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.9","event_number":9,"trigger":"x = \"a\\nb\""}`,
		http.StatusUnprocessableEntity, "trigger:")
}

func TestEventGetUpdateDelete(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("france")
	ev := h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.100","event_number":100,"picture":"GFX_report_event_TST_france100"}`, http.StatusCreated)
	id := jsonID(ev)
	h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.7","event_number":7}`, http.StatusCreated)

	got := h.json(http.MethodGet, "/events/"+id, "", http.StatusOK)
	if got["event_id"] != "france.100" {
		t.Fatalf("get: %v", got)
	}
	h.expect(http.MethodGet, "/events/999", "", http.StatusNotFound, "Event not found")
	h.expect(http.MethodGet, "/events/abc", "", http.StatusNotFound, "Event not found")

	updated := h.json(http.MethodPut, "/events/"+id, `{
		"event_id":" france.101 ","title":"New title","after":null,"trigger":"a = yes\\nb = yes",
		"display_name_locs":{"zh":"标题"}
	}`, http.StatusOK)
	if updated["event_id"] != "france.101" || updated["title_en"] != "New title" || updated["after"] != "" ||
		updated["trigger"] != "a = yes\nb = yes" || updated["picture"] != "GFX_report_event_TST_france101" ||
		jsonString(updated["display_name_locs"]) != `{"zh":"标题"}` {
		t.Fatalf("updated: %v", updated)
	}

	custom := h.json(http.MethodPut, "/events/"+id, `{"event_id":"france.102","picture":"GFX_custom"}`, http.StatusOK)
	if custom["picture"] != "GFX_custom" {
		t.Fatalf("custom picture renamed: %v", custom)
	}
	h.expect(http.MethodPut, "/events/"+id, `{"event_id":"france.7"}`, http.StatusConflict, "event_id 'france.7' already exists in this project")
	h.expect(http.MethodPut, "/events/"+id, `{"namespace_id":999}`, http.StatusBadRequest, "Namespace not found in this project")
	h.expect(http.MethodPut, "/events/"+id, `{"immediate":"x = \"a\\nb\""}`, http.StatusBadRequest, "immediate:")
	h.expect(http.MethodPut, "/events/"+id, `{"event_type":null}`, http.StatusUnprocessableEntity, "event_type cannot be null")
	h.expect(http.MethodPut, "/events/"+id, `{"event_number":"x"}`, http.StatusUnprocessableEntity, "Invalid event payload")

	if err := h.p.Write([]string{eventsTable}, func(tx *project.Tx) error {
		_, err := tx.Update(eventsTable, project.Int64(json.Number(id)), func(row project.Row) { row["source"] = "imported" })
		return err
	}); err != nil {
		t.Fatal(err)
	}
	flipped := h.json(http.MethodPut, "/events/"+id, `{}`, http.StatusOK)
	if flipped["source"] != "created" {
		t.Fatalf("imported event must flip to created: %v", flipped)
	}

	h.expect(http.MethodDelete, "/events/"+id, "", http.StatusNoContent, "")
	h.expect(http.MethodDelete, "/events/"+id, "", http.StatusNotFound, "Event not found")
}

func TestEventFork(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("france")
	src := h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.1","event_number":1,"trigger":"tag = FRA","source":"imported","options":[{"name":"x"}]}`, http.StatusCreated)
	id := jsonID(src)
	h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.5","event_number":5}`, http.StatusCreated)

	fork := h.json(http.MethodPost, "/events/"+id+"/fork", `{"new_event_id":" france.200 "}`, http.StatusCreated)
	if fork["event_id"] != "france.200" || fork["event_number"] != float64(200) || fork["source"] != "created" ||
		fork["trigger"] != "tag = FRA" || jsonID(fork) == id || jsonString(fork["options"]) != jsonString(src["options"]) {
		t.Fatalf("fork: %v", fork)
	}
	if original := h.json(http.MethodGet, "/events/"+id, "", http.StatusOK); original["source"] != "imported" {
		t.Fatalf("fork must leave the original untouched: %v", original)
	}
	h.expect(http.MethodPost, "/events/"+id+"/fork", `{"new_event_id":"france"}`, http.StatusBadRequest, "new_event_id must match 'namespace.NUMBER'")
	h.expect(http.MethodPost, "/events/"+id+"/fork", `{"new_event_id":"france.1"}`, http.StatusBadRequest, "new_event_id must differ from the original")
	h.expect(http.MethodPost, "/events/"+id+"/fork", `{"new_event_id":"france.5"}`, http.StatusConflict, "event_id 'france.5' already exists in this project")
	h.expect(http.MethodPost, "/events/999/fork", `{"new_event_id":"france.300"}`, http.StatusNotFound, "Source event not found")
	h.expect(http.MethodPost, "/events/"+id+"/fork", `{}`, http.StatusUnprocessableEntity, "new_event_id is required")
}

func TestEventPreviews(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("france")
	ev := h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.1","event_number":1,"title":"france.1.t","description":"france.1.d"}`, http.StatusCreated)
	id := jsonID(ev)

	nsScript := h.json(http.MethodGet, "/events/namespaces/"+ns+"/preview", "", http.StatusOK)["script"].(string)
	if !strings.Contains(nsScript, "add_namespace = france") || !strings.Contains(nsScript, "id = france.1") {
		t.Fatalf("namespace preview: %q", nsScript)
	}
	evScript := h.json(http.MethodGet, "/events/"+id+"/preview", "", http.StatusOK)["script"].(string)
	if evScript != nsScript {
		t.Fatalf("single-event preview differs from its namespace's:\n%q\n%q", evScript, nsScript)
	}
	h.expect(http.MethodGet, "/events/nope/preview", "", http.StatusUnprocessableEntity, `"loc":["path","event_db_id"]`)
	h.expect(http.MethodGet, "/events/0/preview", "", http.StatusNotFound, "Event not found")
	h.expect(http.MethodGet, "/events/999/preview", "", http.StatusNotFound, "Event not found")
	h.expect(http.MethodGet, "/events/namespaces/nope/preview", "", http.StatusUnprocessableEntity, `"loc":["path","ns_id"]`)
	h.expect(http.MethodGet, "/events/namespaces/999/preview", "", http.StatusNotFound, "Namespace not found")

	h.expect(http.MethodGet, "/events/"+id+"/lint", "", http.StatusOK, `{"warnings":[]}`)
	h.expect(http.MethodGet, "/events/999/lint", "", http.StatusNotFound, "Event not found")
}

func TestEventParsePreview(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("test")
	id := jsonID(h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"test.1","event_number":1}`, http.StatusCreated))
	path := "/events/" + id + "/parse-preview"

	script := "add_namespace = test\ncountry_event = {\n    id = test.7\n    title = test.7.t\n    timeout_days = 5\n    custom_effect = { add_stability = 0.1 }\n}"
	body, _ := json.Marshal(map[string]string{"script": script})
	got := h.json(http.MethodPost, path, string(body), http.StatusOK)
	entity := got["entity"].(map[string]any)
	if entity["event_id"] != "test.7" || entity["event_number"] != float64(7) || jsonString(got["parse_errors"]) != "[]" {
		t.Fatalf("parse result: %v", got)
	}
	if raw, ok := entity["raw_extras"].(map[string]any); !ok || !strings.Contains(raw["custom_effect"].(string), "add_stability") {
		t.Fatalf("raw_extras must stay apply-able: %v", entity["raw_extras"])
	}
	if verdict, ok := got["extras"].(map[string]any)["_scope_verdict"].(map[string]any); !ok || verdict["kind"] != "global" {
		t.Fatalf("scope verdict extras: %v", got["extras"])
	}

	empty := h.json(http.MethodPost, path, `{"script":"  \n"}`, http.StatusOK)
	if empty["entity"] != nil || jsonString(empty["parse_errors"]) != `[{"line":1,"msg":"Script is empty","severity":"info"}]` {
		t.Fatalf("empty script: %v", empty)
	}
	multi, _ := json.Marshal(map[string]string{"script": "add_namespace = test\ncountry_event = { id = test.1 }\ncountry_event = { id = test.2 }"})
	got = h.json(http.MethodPost, path, string(multi), http.StatusOK)
	if jsonString(got["warnings"]) != `["Script defined 2 events; only the first (test.1) was used. Apply this and edit additional events separately."]` {
		t.Fatalf("multiple events: %v", got)
	}
	none := h.json(http.MethodPost, path, `{"script":"x = 1"}`, http.StatusOK)
	if errs := none["parse_errors"].([]any); len(errs) != 1 || errs[0].(map[string]any)["severity"] != "error" {
		t.Fatalf("no event: %v", none)
	}
	h.expect(http.MethodPost, path, `{}`, http.StatusUnprocessableEntity, `"loc":["body","script"]`)
	h.expect(http.MethodPost, path, `{"script":null}`, http.StatusUnprocessableEntity, `"type":"string_type"`)
	h.expect(http.MethodPost, path, `{"script":`, http.StatusUnprocessableEntity, `"type":"json_invalid"`)
	h.expect(http.MethodPost, "/events/999/parse-preview", `{"script":"x"}`, http.StatusNotFound, "Event not found")
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (h *harness) upload(path string, data []byte) *httptest.ResponseRecorder {
	h.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "picture.png")
	if err != nil {
		h.t.Fatal(err)
	}
	part.Write(data)
	form.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/go/projects/1"+path, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func pictureSize(t *testing.T, rec *httptest.ResponseRecorder) image.Point {
	t.Helper()
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("picture: status %d type %q: %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	return img.Bounds().Size()
}

func TestEventPictureUploadAndServe(t *testing.T) {
	h := newHarness(t, nil)
	ns := h.namespace("france")
	ev := h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.1","event_number":1,"country_tag":"fra"}`, http.StatusCreated)
	id := jsonID(ev)

	h.expect(http.MethodGet, "/events/"+id+"/picture", "", http.StatusNotFound, "Asset not uploaded")
	rec := h.upload("/events/"+id+"/upload-picture", testPNG(t, 300, 250))
	if rec.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["picture"] != "GFX_report_event_FRA_france1" || resp["filename"] != "event_FRA_france1.png" ||
		resp["preview_url"] != "/api/go/projects/1/events/"+id+"/picture" || jsonString(resp["warnings"]) != "[]" {
		t.Fatalf("upload response: %v", resp)
	}
	if got := h.json(http.MethodGet, "/events/"+id, "", http.StatusOK); got["picture"] != "GFX_report_event_FRA_france1" {
		t.Fatalf("event picture not set: %v", got)
	}
	rows, _ := h.p.Rows("events")
	if rows[0]["has_obs"] != true {
		t.Fatalf("has_obs not set: %v", rows[0])
	}
	paths, _ := h.p.Rows("entity_obs_paths")
	if len(paths) != 1 || paths[0]["entity_type"] != "event_picture" || !strings.HasPrefix(paths[0]["obs_key"].(string), "local/p/u1/upload/event/"+id+"/event_picture/") {
		t.Fatalf("mapping: %v", paths)
	}
	if size := pictureSize(t, h.do(http.MethodGet, "/events/"+id+"/picture", "")); size != (image.Point{210, 176}) {
		t.Fatalf("event picture size %v", size)
	}

	if rec := h.upload("/events/"+id+"/upload-picture", testPNG(t, 100, 100)); rec.Code != http.StatusOK {
		t.Fatalf("second upload: %d %s", rec.Code, rec.Body.String())
	}
	if paths, _ := h.p.Rows("entity_obs_paths"); len(paths) != 1 {
		t.Fatalf("mapping duplicated: %v", paths)
	}

	news := jsonID(h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.2","event_number":2,"event_type":"news_event"}`, http.StatusCreated))
	if rec := h.upload("/events/"+news+"/upload-picture", testPNG(t, 500, 300)); rec.Code != http.StatusOK {
		t.Fatalf("news upload: %d %s", rec.Code, rec.Body.String())
	}
	if size := pictureSize(t, h.do(http.MethodGet, "/events/"+news+"/picture", "")); size != (image.Point{397, 153}) {
		t.Fatalf("news picture size %v", size)
	}

	h.expect(http.MethodPost, "/events/999/upload-picture", "", http.StatusNotFound, "Asset target not found")
	if rec := h.upload("/events/"+id+"/upload-picture", []byte("not an image")); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Failed to decode image") {
		t.Fatalf("bad image: %d %s", rec.Code, rec.Body.String())
	}
	h.expect(http.MethodGet, "/events/999/picture", "", http.StatusNotFound, "Asset not uploaded")
}

func TestEventPictureImportedAndBaseGame(t *testing.T) {
	game := fstest.MapFS{
		"interface/eventpictures.gfx":    {Data: []byte("spriteTypes = {\n\tspriteType = {\n\t\tname = \"GFX_report_event_vanilla\"\n\t\ttexturefile = \"gfx/event_pictures/vanilla.png\"\n\t}\n}\n")},
		"gfx/event_pictures/vanilla.png": {Data: testPNG(t, 20, 10)},
	}
	h := newHarness(t, game)
	ns := h.namespace("france")
	imported := jsonID(h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.1","event_number":1,"picture":"GFX_mod_picture"}`, http.StatusCreated))
	vanilla := jsonID(h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.2","event_number":2,"picture":"GFX_report_event_vanilla"}`, http.StatusCreated))
	missing := jsonID(h.json(http.MethodPost, "/events", `{"namespace_id":`+ns+`,"event_id":"france.3","event_number":3,"picture":"GFX_nowhere"}`, http.StatusCreated))

	key := "local/p/u1/import/event_picture/mod_picture.png"
	if err := h.p.Blobs().Put(key, "image/png", -1, bytes.NewReader(testPNG(t, 30, 15))); err != nil {
		t.Fatal(err)
	}
	if err := h.p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		tx.Insert("project_obs_assets", project.Row{"asset_type": "event_picture", "gfx_name": "GFX_mod_picture", "obs_key": key})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if size := pictureSize(t, h.do(http.MethodGet, "/events/"+imported+"/picture", "")); size != (image.Point{30, 15}) {
		t.Fatalf("imported picture size %v", size)
	}
	if size := pictureSize(t, h.do(http.MethodGet, "/events/"+vanilla+"/picture", "")); size != (image.Point{20, 10}) {
		t.Fatalf("base-game picture size %v", size)
	}
	h.expect(http.MethodGet, "/events/"+missing+"/picture", "", http.StatusNotFound, "Asset not uploaded")
}

func TestMethodRouting(t *testing.T) {
	h := newHarness(t, nil)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/events/1/lint"},
		{http.MethodPost, "/events/1/preview"},
		{http.MethodGet, "/events/1/fork"},
		{http.MethodGet, "/events/namespaces/1"},
	} {
		if rec := h.do(tc.method, tc.path, ""); rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s: status %d", tc.method, tc.path, rec.Code)
		}
	}
}
