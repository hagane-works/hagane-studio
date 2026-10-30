package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuditTagsTakesFilesAndPathsInAnyOrder(t *testing.T) {
	a := newTestAPI(t)
	var created map[string]any
	a.call("POST", "/api/go/projects", map[string]any{"name": "Audit", "country_tag": "TST", "country_name": "Testland"}, &created)
	path := "/api/go/projects/" + jsonNumber(int64(created["id"].(float64))) + "/import/audit-tags"

	files := []struct{ path, body string }{
		{"common/country_tags/00_countries.txt", "ABC = \"countries/Abc.txt\"\nDEF = \"countries/Def.txt\"\n"},
		{"common/countries/Abc.txt", "color = { 10 20 30 }\n"},
	}
	orders := map[string]func(w *multipart.Writer){
		"each file before its path": func(w *multipart.Writer) {
			for _, f := range files {
				writeFilePart(t, w, f.body)
				writeField(t, w, f.path)
			}
		},
		"files, then paths": func(w *multipart.Writer) {
			for _, f := range files {
				writeFilePart(t, w, f.body)
			}
			for _, f := range files {
				writeField(t, w, f.path)
			}
		},
		"paths, then files": func(w *multipart.Writer) {
			for _, f := range files {
				writeField(t, w, f.path)
			}
			for _, f := range files {
				writeFilePart(t, w, f.body)
			}
		},
	}
	for name, write := range orders {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		write(form)
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", path, &body)
		req.Header.Set("Content-Type", form.FormDataContentType())
		rec := httptest.NewRecorder()
		a.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
		var result struct {
			Detected []struct {
				Tag string `json:"tag"`
			} `json:"detected"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Detected) != 2 || result.Detected[0].Tag != "ABC" || result.Detected[1].Tag != "DEF" {
			t.Fatalf("%s: detected = %+v", name, result.Detected)
		}
	}
}

func writeField(t *testing.T, w *multipart.Writer, value string) {
	t.Helper()
	if err := w.WriteField("paths", value); err != nil {
		t.Fatal(err)
	}
}

func writeFilePart(t *testing.T, w *multipart.Writer, body string) {
	t.Helper()
	part, err := w.CreateFormFile("files", "blob")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}
