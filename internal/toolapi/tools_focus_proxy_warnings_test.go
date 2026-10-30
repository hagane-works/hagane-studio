package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateFocus_PreservesExplicitIdentityAndRelaysWarnings(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 56, "focus_id": "GER_taken_1",
			"warnings": []string{"focus_id 'GER_taken' collided · auto-suffixed to 'GER_taken_1'."},
		})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "name": "Taken", "focus_id": "GER_taken"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["on_conflict"] != "error" {
		t.Errorf("upstream body on_conflict = %v, want explicit conflict rejection", gotBody["on_conflict"])
	}
	if gotBody["focus_id"] != "GER_taken" {
		t.Errorf("upstream body focus_id = %v, want the caller's explicit id forwarded verbatim", gotBody["focus_id"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["focus_id"] != "GER_taken_1" {
		t.Errorf("focus_id = %v, want the persisted (suffixed) id GER_taken_1", out["focus_id"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "focus_id 'GER_taken' collided · auto-suffixed to 'GER_taken_1'." {
		t.Fatalf("warnings = %v, want the upstream collision notice relayed verbatim (it used to be dropped)", out["warnings"])
	}
}

func TestCreateFocus_OmittedFocusIDIsNotSentAndNoWarningsKeyWhenClean(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 57, "focus_id": "GER_New_Focus", "warnings": []string{}})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "name": "New Focus"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if _, present := gotBody["focus_id"]; present {
		t.Errorf("upstream body must NOT carry focus_id when the caller omitted it (backend derives tag+name): %v", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if w, present := out["warnings"]; present {
		if arr, _ := w.([]any); len(arr) != 0 {
			t.Errorf("warnings = %v, want none on a clean create", w)
		}
	}
}
