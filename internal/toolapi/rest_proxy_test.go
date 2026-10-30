package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRESTProxyMachineCodeIsBoundedAndDoesNotChangeErrorText(t *testing.T) {
	for _, code := range []string{"generation_receipt_conflict", "", "https://private.example/token", strings.Repeat("x", 65), "Invalid_Request", "bad\ncode"} {
		t.Run(code, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "detail": "unchanged safe detail"})
			}))
			defer upstream.Close()
			_, err := NewRESTProxy(upstream.URL).Call(context.Background(), http.MethodPost, "/test", nil)
			restErr, ok := err.(*RESTError)
			if !ok || restErr.Error() != "unchanged safe detail" || restErr.Status != 409 {
				t.Fatalf("error contract changed: %#v", err)
			}
			want := ""
			if code == "generation_receipt_conflict" {
				want = code
			}
			if restErr.Code != want {
				t.Fatalf("unsafe or lost code: %q", restErr.Code)
			}
		})
	}
}

func TestRESTProxy_ForwardsBody(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "tag": "ITA"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	result, err := proxy.Call(context.Background(), "POST", "/api/projects/7/countries",
		map[string]any{"tag": "ITA", "name": "Italy"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/projects/7/countries" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["tag"] != "ITA" {
		t.Errorf("body[tag] = %v, want ITA", gotBody["tag"])
	}
	if result["tag"] != "ITA" {
		t.Errorf("result[tag] = %v, want ITA", result["tag"])
	}
}

func TestRESTProxy_SurfacesUpstreamErrorDetail(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Country ITA already exists in this project"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	_, err := proxy.Call(context.Background(), "POST", "/api/projects/7/countries", map[string]any{"tag": "ITA"})
	if err == nil {
		t.Fatal("expected an error for a 409 response")
	}
	restErr, ok := err.(*RESTError)
	if !ok {
		t.Fatalf("err type = %T, want *RESTError", err)
	}
	if restErr.Status != http.StatusConflict {
		t.Errorf("Status = %d, want 409", restErr.Status)
	}
	if restErr.Detail != "Country ITA already exists in this project" {
		t.Errorf("Detail = %q, want the passed-through FastAPI detail string", restErr.Detail)
	}
}

func TestRESTProxy_StructuredDetailReencodes(t *testing.T) {

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"detail": map[string]any{"error": "ctd_blockers", "blockers": []any{map[string]any{"code": "x"}}},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	_, err := proxy.Call(context.Background(), "POST", "/x", nil)
	restErr := err.(*RESTError)
	var reparsed map[string]any
	if jsonErr := json.Unmarshal([]byte(restErr.Detail), &reparsed); jsonErr != nil {
		t.Fatalf("Detail is not valid JSON: %q (%v)", restErr.Detail, jsonErr)
	}
	if reparsed["error"] != "ctd_blockers" {
		t.Errorf("reparsed detail = %+v, want error=ctd_blockers", reparsed)
	}
}

func TestProgramProxyCallsTheAPIInProcessAndFailsClosed(t *testing.T) {
	var gotPath, gotRevisions string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/go/projects/{id}/wargoals", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"wargoals": []any{}})
	})
	mux.HandleFunc("PUT /api/go/projects/{id}/focus-trees/{tid}", func(w http.ResponseWriter, r *http.Request) {
		gotRevisions = r.Header.Get("X-Edit-Revisions")
		w.Header().Set("X-Edit-Revisions", `{"focus_tree:3":"8"}`)
		w.WriteHeader(http.StatusTeapot)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "refused", "code": "edit_revision_conflict"})
	})
	proxy := NewProgramProxy(mux)
	if _, err := proxy.Call(context.Background(), http.MethodGet, "/api/projects/7/wargoals?x=1", nil); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/go/projects/7/wargoals?x=1" {
		t.Fatalf("mapped path = %q", gotPath)
	}
	ctx := context.WithValue(context.Background(), editTokensKey{}, map[string]string{"focus_tree:3": "7"})
	_, err := proxy.withTimeout(time.Minute).Call(ctx, http.MethodPut, "/api/projects/7/focus-trees/3", map[string]any{})
	restErr, ok := err.(*RESTError)
	if !ok || restErr.Status != http.StatusTeapot || restErr.Detail != "refused" || restErr.Code != "edit_revision_conflict" {
		t.Fatalf("error through the in-process transport (and a timeout clone): %#v", err)
	}
	if gotRevisions != `{"focus_tree:3":"7"}` {
		t.Errorf("X-Edit-Revisions = %q", gotRevisions)
	}
	for _, path := range []string{"/api/auth/me", "/api/ai-art/generate", "/api/projects/../app/quit", "/api/app/settings", "https://example.com/api/projects"} {
		if _, err := proxy.Call(context.Background(), http.MethodGet, path, nil); err == nil {
			t.Errorf("%s was not refused", path)
		}
	}
}
