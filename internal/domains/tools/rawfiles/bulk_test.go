package rawfiles

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postBulk(t *testing.T, mux *http.ServeMux, files [][2]string) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range files {
		_ = w.WriteField("paths", f[0])
		part, _ := w.CreateFormFile("files", "upload")
		_, _ = part.Write([]byte(f[1]))
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, base+"/bulk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	expect(t, rec, http.StatusOK)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func bulkStatus(out map[string]any) map[string]string {
	got := map[string]string{}
	for _, item := range out["results"].([]any) {
		res := item.(map[string]any)
		key := res["dest_path"].(string)
		got[key] = res["status"].(string)
		if as, ok := res["stored_as"].(string); ok {
			got[key] += "/" + as
		}
	}
	return got
}

func TestRawFilesBulkAddsManyFilesInOneWrite(t *testing.T) {
	p, mux := newProject(t)
	out := postBulk(t, mux, [][2]string{
		{"common/bop/mod_bop.txt", "mod_bop = { initial_value = 0 }\n"},
		{"gfx/interface/mod_icon.dds", "DDS \x00\x01 bytes"},
		{"common/scripted_triggers/broken.txt", "broken_trigger = {\n"},
		{"sound/mod_vo.asset", "sound = { name = mod file = \"mod.wav\" }\n"},
		{"../escape.txt", "x"},
		{"common/bop/mod_bop.txt", "sent twice"},
	})
	want := map[string]string{
		"common/bop/mod_bop.txt":              "rejected", // the second copy; the first is created below
		"gfx/interface/mod_icon.dds":          "created/binary",
		"common/scripted_triggers/broken.txt": "created/binary",
		"sound/mod_vo.asset":                  "created/text",
		"../escape.txt":                       "rejected",
	}
	got := bulkStatus(out)
	for path, status := range want {
		if got[path] != status {
			t.Errorf("%s: %q, want %q (all: %v)", path, got[path], status, got)
		}
	}
	if out["created"].(float64) != 4 {
		t.Fatalf("created = %v, want 4: %v", out["created"], got)
	}
	rows, _ := p.Rows(table)
	byPath := map[string]map[string]any{}
	for _, row := range rows {
		byPath[row["dest_path"].(string)] = row
	}
	if text := byPath["common/bop/mod_bop.txt"]; text == nil || text["content"] != "mod_bop = { initial_value = 0 }\n" || text["is_binary"] != false {
		t.Fatalf("text row = %#v", text)
	}
	bin := byPath["gfx/interface/mod_icon.dds"]
	stored, err := p.Blobs().Get(bin["content_obs_key"].(string), 1<<20)
	if err != nil || string(stored) != "DDS \x00\x01 bytes" {
		t.Fatalf("binary blob = %q, %v", stored, err)
	}
	broken := byPath["common/scripted_triggers/broken.txt"]
	stored, err = p.Blobs().Get(broken["content_obs_key"].(string), 1<<20)
	if err != nil || string(stored) != "broken_trigger = {\n" {
		t.Fatalf("a text file the gate refuses keeps its bytes: %q, %v", stored, err)
	}

	again := bulkStatus(postBulk(t, mux, [][2]string{
		{"common/bop/mod_bop.txt", "changed"},
		{"common/bop/other.txt", "other = { }\n"},
	}))
	if again["common/bop/mod_bop.txt"] != "exists" || again["common/bop/other.txt"] != "created/text" {
		t.Fatalf("resend: %v", again)
	}
}
