package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSessionlessRoutes(t *testing.T) {
	const host = "127.0.0.1:7384"
	srv, err := New(host, fstest.MapFS{}, "")
	if err != nil {
		t.Fatal(err)
	}
	srv.Sessionless.HandleFunc("POST /api/app/show", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv.Mux.HandleFunc("GET /api/app/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	call := func(method, path string, header map[string]string) int {
		req := httptest.NewRequest(method, "http://"+host+path, nil)
		for key, value := range header {
			req.Header.Set(key, value)
		}
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	if code := call("POST", "/api/app/show", nil); code != http.StatusNoContent {
		t.Errorf("sessionless route: %d", code)
	}
	if code := call("GET", "/api/app/show", nil); code != http.StatusUnauthorized {
		t.Errorf("sessionless route with another method: %d", code)
	}
	if code := call("GET", "/api/app/status", nil); code != http.StatusUnauthorized {
		t.Errorf("session route without the cookie: %d", code)
	}
	if code := call("POST", "/api/app/show", map[string]string{"Origin": "https://example.com"}); code != http.StatusForbidden {
		t.Errorf("sessionless route from another site: %d", code)
	}
	if code := call("POST", "/api/app/show", map[string]string{"Sec-Fetch-Site": "cross-site"}); code != http.StatusForbidden {
		t.Errorf("sessionless route from another site (no Origin): %d", code)
	}
}

func TestToolsRoutes(t *testing.T) {
	const host = "127.0.0.1:7384"
	const token = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	for _, configured := range []string{token, ""} {
		srv, err := New(host, fstest.MapFS{}, configured)
		if err != nil {
			t.Fatal(err)
		}
		srv.Tools.HandleFunc("GET /api/tools/{name}", func(w http.ResponseWriter, r *http.Request) {
			if r.PathValue("name") != "list_projects" {
				t.Errorf("path value %q", r.PathValue("name"))
			}
			w.WriteHeader(http.StatusOK)
		})
		call := func(header map[string]string) *httptest.ResponseRecorder {
			req := httptest.NewRequest("GET", "http://"+host+"/api/tools/list_projects", nil)
			for key, value := range header {
				req.Header.Set(key, value)
			}
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			return rec
		}
		want := http.StatusOK
		if configured == "" {
			want = http.StatusUnauthorized
		}
		if rec := call(map[string]string{"Authorization": "Bearer " + token}); rec.Code != want {
			t.Errorf("token %q configured: right token %d, want %d", configured, rec.Code, want)
		}
		for _, header := range []map[string]string{
			nil,
			{"Authorization": "Bearer wrong"},
			{"Authorization": token},
			{"Cookie": sessionCookie + "=" + srv.token},
		} {
			rec := call(header)
			if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("token %q configured, %v: %d", configured, header, rec.Code)
			}
		}
		if rec := call(map[string]string{"Authorization": "Bearer " + token, "Origin": "https://example.com"}); rec.Code != http.StatusForbidden {
			t.Errorf("tools route from another site: %d", rec.Code)
		}
		req := httptest.NewRequest("GET", "http://localhost:7384/api/tools/list_projects", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("tools route under another host name: %d", rec.Code)
		}
	}
}
