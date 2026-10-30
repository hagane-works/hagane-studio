package lint

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func seeded(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpdateRecord(func(r project.Row) { r["start_date"] = "1936.1.1.12" }); err != nil {
		t.Fatal(err)
	}
	tables := []string{"event_namespaces", "events", "focus_trees", "focus_nodes", "ui_panels",
		"scripted_snippets", "bookmarks", "characters"}
	err = p.Write(tables, func(tx *project.Tx) error {
		ns := tx.Insert("event_namespaces", project.Row{"namespace": "tst", "country_tag": "TST"})
		tx.Insert("events", project.Row{
			"namespace_id": ns["id"], "event_id": "tst.1", "title_en": "Hello", "country_tag": "TST",
			"is_triggered_only": false, "mean_time_to_happen": "factor = 2",
			"trigger": "any_country = { any_state = { is_core_of = ROOT } }", "immediate": "", "after": "",
		})
		tree := tx.Insert("focus_trees", project.Row{"tree_id": "tst_focus", "country_tag": "TST"})
		tx.Insert("focus_nodes", project.Row{
			"focus_tree_id": tree["id"], "focus_id": "TST_start", "name": "Start",
			"available": "any_country = { has_war = yes }",
		})
		tx.Insert("ui_panels", project.Row{
			"token": "A B", "display_name_locs": map[string]any{}, "raw_extras": map[string]any{},
			"visible_trigger": "my_missing = yes\nmy_known = yes",
			"controls":        []any{map[string]any{"kind": "icon", "name": "pic", "sprite": ""}},
		})
		tx.Insert("scripted_snippets", project.Row{"kind": "trigger", "name": "my_known", "body": ""})
		tx.Insert("bookmarks", project.Row{"bookmark_id": "late", "date": "1937.1.1.12", "source": "created", "default_flag": false})
		tx.Insert("characters", project.Row{"tag": "TST_leader", "name": "Leader", "country_tag": "TST"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newMux(p *project.Project) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, domains.Deps{Open: func(r *http.Request) (*project.Project, error) {
		if r.PathValue("id") != "1" {
			return nil, project.ErrNotFound
		}
		return p, nil
	}})
	return mux
}

func do(t *testing.T, mux *http.ServeMux, method, path string, body *bytes.Buffer, ctype string) (int, map[string]any) {
	t.Helper()
	if body == nil {
		body = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, body)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func codes(report map[string]any) map[string]bool {
	out := map[string]bool{}
	findings, _ := report["findings"].([]any)
	for _, f := range findings {
		out[f.(map[string]any)["code"].(string)] = true
	}
	return out
}

func multipartFiles(t *testing.T, files, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for field, content := range files {
		fw, err := mw.CreateFormFile(field, field+".txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if len(files) == 0 && len(fields) == 0 {
		_ = mw.WriteField("noise", "x")
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestPerfRoute(t *testing.T) {
	mux := newMux(seeded(t))
	status, report := do(t, mux, "GET", "/api/go/lint/perf/1", nil, "")
	if status != 200 {
		t.Fatalf("status=%d body=%v", status, report)
	}
	got := codes(report)
	for _, code := range []string{"perf.nested_scope_loop", "perf.mtth_no_floor", "perf.hot_scope_loop"} {
		if !got[code] {
			t.Errorf("missing %s in %v", code, got)
		}
	}
	scanned := report["scanned"].(map[string]any)
	if scanned["events"] != float64(1) || scanned["focus_nodes"] != float64(1) || scanned["characters"] != float64(1) {
		t.Errorf("scanned = %v", scanned)
	}
	if v, ok := report["export_profile"]; !ok || v != nil {
		t.Errorf("export_profile = %v (present %v), want null", v, ok)
	}

	for _, f := range report["findings"].([]any) {
		finding := f.(map[string]any)
		if finding["code"] == "perf.mtth_no_floor" && finding["file"] != "events/TST.txt" {
			t.Errorf("mtth file = %v", finding["file"])
		}
	}
}

func TestStartDatesRoute(t *testing.T) {
	mux := newMux(seeded(t))
	status, report := do(t, mux, "GET", "/api/go/lint/start-dates/1", nil, "")
	if status != 200 {
		t.Fatalf("status=%d body=%v", status, report)
	}
	if report["start_date"] != "1936.1.1.12" || report["custom_start"] != false {
		t.Errorf("start_date/custom_start = %v/%v", report["start_date"], report["custom_start"])
	}
	if !codes(report)["timeline.project_earliest_bookmark_mismatch"] {
		t.Errorf("want project_earliest_bookmark_mismatch, got %v", codes(report))
	}
}

func TestPanelContractCoreRoute(t *testing.T) {
	mux := newMux(seeded(t))
	status, report := do(t, mux, "GET", "/api/go/lint/panel-contract-core/1", nil, "")
	if status != 200 || report["scope"] != "database_core" || report["scanned_panels"] != float64(1) {
		t.Fatalf("status=%d body=%v", status, report)
	}
	got := codes(report)
	if !got["panel.unknown_scripted"] || !got["panel.sprite_empty"] {
		t.Errorf("codes = %v", got)
	}
	for _, f := range report["findings"].([]any) {
		finding := f.(map[string]any)
		if finding["code"] == "panel.unknown_scripted" && finding["detail"].(map[string]any)["symbol"] != "my_missing" {
			t.Errorf("a project scripted trigger was flagged: %v", finding)
		}
	}
}

func TestReportRoutesProjectErrors(t *testing.T) {
	mux := newMux(seeded(t))
	for _, path := range []string{"/api/go/lint/perf/", "/api/go/lint/start-dates/", "/api/go/lint/panel-contract-core/"} {
		if status, body := do(t, mux, "GET", path+"nope", nil, ""); status != 422 || body["detail"] != "project_id must be an integer" {
			t.Errorf("%snope: %d %v", path, status, body)
		}
		if status, body := do(t, mux, "GET", path+"2", nil, ""); status != 404 || body["detail"] != "Project not found" {
			t.Errorf("%s2: %d %v", path, status, body)
		}
	}
}

func TestErrorLogRoute(t *testing.T) {
	mux := newMux(seeded(t))
	logText := "[16:22:33][1][game.cpp:1]: Duplicate idea. GER_x\n" +
		"[16:22:34][2][game.cpp:1]: Duplicate idea. GER_x\n" +
		"[16:22:35][3][eventmanager.cpp:1]: Malformed token in event tst.1\n"
	body, ctype := multipartFiles(t, map[string]string{"file": logText}, nil)
	status, report := do(t, mux, "POST", "/api/go/lint/error-log/1", body, ctype)
	if status != 200 {
		t.Fatalf("status=%d body=%v", status, report)
	}
	if report["lines_parsed"] != float64(3) || report["unique_signatures"] != float64(2) || report["mapped"] != float64(1) {
		t.Errorf("counts: %v", report)
	}
	mapped := false
	for _, f := range report["findings"].([]any) {
		ref, _ := f.(map[string]any)["entity_ref"].(map[string]any)
		if ref != nil && ref["kind"] == "event" && ref["entity_id"] == "tst.1" && ref["name"] == "Hello" {
			mapped = true
		}
	}
	if !mapped {
		t.Errorf("event tst.1 not mapped: %v", report["findings"])
	}
}

func TestErrorLogValidationOrder(t *testing.T) {
	mux := newMux(seeded(t))
	big := strings.Repeat("x", maxLogUploadBytes+1)
	for _, tc := range []struct {
		name, path string
		files      map[string]string
		want       int
		detail     string
	}{
		{"missing field", "/api/go/lint/error-log/2", nil, 422, "file is required"},
		{"empty file", "/api/go/lint/error-log/2", map[string]string{"file": ""}, 400, "error.log is empty"},
		{"too large", "/api/go/lint/error-log/2", map[string]string{"file": big}, 413, "error.log too large (> 25 MB)"},
		{"missing project", "/api/go/lint/error-log/2", map[string]string{"file": "x"}, 404, "Project not found"},
		{"bad id first", "/api/go/lint/error-log/nope", nil, 422, "project_id must be an integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, ctype := multipartFiles(t, tc.files, nil)
			status, out := do(t, mux, "POST", tc.path, body, ctype)
			if status != tc.want || out["detail"] != tc.detail {
				t.Fatalf("status=%d body=%v, want %d %q", status, out, tc.want, tc.detail)
			}
		})
	}
}

func TestPlaytestBundleRoute(t *testing.T) {
	mux := newMux(seeded(t))
	gameLog := "[16:00:00][1936.01.01.12][game.cpp:1]: Start-date: 1936.1.1.12\n"
	for _, tc := range []struct {
		name     string
		path     string
		files    map[string]string
		fields   map[string]string
		want     int
		contains string
	}{
		{"game log mismatches the bookmark", "/api/go/lint/playtest-bundle/1", map[string]string{"game_log": gameLog}, nil, 200,
			`"playtest.start_date_mismatch"`},
		{"no files", "/api/go/lint/playtest-bundle/1", nil, nil, 400, "Upload at least one of game.log"},
		{"empty files count as absent", "/api/go/lint/playtest-bundle/1", map[string]string{"game_log": ""}, nil, 400,
			"Upload at least one of game.log"},
		{"dlc load order", "/api/go/lint/playtest-bundle/1", map[string]string{"dlc_load_json": `{"enabled_mods":["a.mod","b.mod"]}`}, nil, 200,
			`"playtest.enabled_mods"`},
		{"identity missing path is reported", "/api/go/lint/playtest-bundle/1", map[string]string{"system_log": "Active Mod Count: 1\n"},
			map[string]string{"expected_mod_name": "My Mod"}, 200, "expected mod identity requires name and path"},
		{"project before files", "/api/go/lint/playtest-bundle/2", nil, nil, 404, "Project not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, ctype := multipartFiles(t, tc.files, tc.fields)
			req := httptest.NewRequest("POST", tc.path, body)
			req.Header.Set("Content-Type", ctype)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("status=%d want %d body=%.500s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	body, ctype := multipartFiles(t, map[string]string{"game_log": gameLog}, nil)
	_, report := do(t, mux, "POST", "/api/go/lint/playtest-bundle/1", body, ctype)
	if report["built_date"] != "1937.1.1.12" || report["loaded_date"] != "1936.1.1.12" {
		t.Errorf("built/loaded = %v/%v", report["built_date"], report["loaded_date"])
	}
}

func TestDecodeUTF8ReplaceMatchesPython(t *testing.T) {
	for _, tc := range []struct {
		in   []byte
		want string
	}{
		{[]byte("plain ascii"), "plain ascii"},
		{[]byte("中文正常"), "中文正常"},
		{[]byte{0xe2, 0x82}, "�"},
		{[]byte{0xff}, "�"},
		{[]byte{0xc3}, "�"},
		{[]byte{0xf0, 0x9f, 0x98}, "�"},
		{[]byte{0xe2, 0x82, 0xff}, "��"},
		{[]byte("abc\xffdef"), "abc�def"},
		{[]byte{0xed, 0xa0, 0x80}, "���"},
		{[]byte{0xc0, 0xaf}, "��"},
		{[]byte{0xe2, 0x82, 0xac}, "€"},
		{[]byte{0xf0, 0x9f, 0x98, 0x80}, "😀"},
	} {
		if got := decodeUTF8Replace(tc.in); got != tc.want {
			t.Errorf("decodeUTF8Replace(% x) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestErrorLogReportEnvelope(t *testing.T) {
	rep := BuildLogReportFromIndex(NewLogProjectIndex(), "[1][2][mod.cpp:3]: incorrect checksum for DLC\n")
	for _, k := range []string{"findings", "totals", "count", "lines_parsed",
		"unique_signatures", "mapped", "unmapped", "by_bucket"} {
		if _, ok := rep[k]; !ok {
			t.Errorf("report missing key %q", k)
		}
	}
}
