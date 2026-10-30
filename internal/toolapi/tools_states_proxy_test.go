package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEditState_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/states/42" || r.Method != http.MethodPut {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 42, "state_id": 613, "name": "STATE_613",
			"display_name": map[string]any{"en": "Borneo", "zh": "婆罗洲"},
			"owner":        "ENG", "controller": "ENG", "source": "created",
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerEditState(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "edit_state",
		Arguments: map[string]any{"project_id": 7, "state_db_id": 42, "owner": "ENG"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}
	if gotBody["owner"] != "ENG" {
		t.Errorf("PUT body owner = %v, want ENG", gotBody["owner"])
	}
	if _, present := gotBody["controller"]; present {
		t.Errorf("PUT body has controller = %v, want the key ABSENT (omitted arg must not be sent — patch semantics)", gotBody["controller"])
	}
	if _, present := gotBody["display_name_locs"]; present {
		t.Errorf("PUT body has display_name_locs, want it absent when no rename was requested")
	}

	out := result.StructuredContent.(map[string]any)
	if out["state_db_id"] != float64(42) {
		t.Errorf("state_db_id = %v, want 42", out["state_db_id"])
	}
	if out["state_id"] != float64(613) {
		t.Errorf("state_id = %v, want 613", out["state_id"])
	}
	if out["owner"] != "ENG" {
		t.Errorf("owner = %v, want ENG", out["owner"])
	}
	if out["source"] != "created" {
		t.Errorf("source = %v, want created (flipped on edit)", out["source"])
	}
	displayName, _ := out["display_name"].(map[string]any)
	if displayName["zh"] != "婆罗洲" {
		t.Errorf("display_name = %+v, want the resolved zh name relayed from the REST response", out["display_name"])
	}
}

func TestEditState_RenameMergesExistingDisplayNameLocs(t *testing.T) {
	var gotPUTBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/states/42" {
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 42, "state_id": 613, "name": "STATE_613",
				"display_name_locs": map[string]any{"zh": "婆罗洲旧名"},
			})
		case http.MethodPut:
			gotPUTBody, _ = readJSONBody(r)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 42, "state_id": 613, "name": "STATE_613",
				"display_name": map[string]any{"en": "New Borneo", "zh": "婆罗洲旧名"},
				"owner":        "", "controller": "", "source": "created",
			})
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerEditState(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "edit_state",
		Arguments: map[string]any{"project_id": 7, "state_db_id": 42, "display_name_en": "New Borneo"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	locs, _ := gotPUTBody["display_name_locs"].(map[string]any)
	if locs["en"] != "New Borneo" {
		t.Errorf("PUT display_name_locs.en = %v, want New Borneo", locs["en"])
	}
	if locs["zh"] != "婆罗洲旧名" {
		t.Errorf("PUT display_name_locs.zh = %v, want the EXISTING zh value carried over (read-modify-write merge)", locs["zh"])
	}
}

func TestEditState_UpstreamNotFoundSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "State not found"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerEditState(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "edit_state",
		Arguments: map[string]any{"project_id": 7, "state_db_id": 999, "owner": "ENG"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "State not found") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestSeedStatesFromVanilla_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/states/seed-from-vanilla" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"seeded": 1046, "skipped": 3, "missing": []any{}, "warnings": []any{"Parse failed 999-Bad.txt: bad token"},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerSeedStatesFromVanilla(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "seed_states_from_vanilla",
		Arguments: map[string]any{"project_id": 7, "all_states": true},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}
	if gotBody["all"] != true {
		t.Errorf("PUT body all = %v, want true", gotBody["all"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["seeded"] != float64(1046) {
		t.Errorf("seeded = %v, want 1046", out["seeded"])
	}
	if out["skipped"] != float64(3) {
		t.Errorf("skipped = %v, want 3", out["skipped"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %+v, want 1 entry relayed", out["warnings"])
	}
}

func TestSeedStatesFromVanilla_ExplicitStateIDsForwardedAndMissingReported(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"seeded": 1, "skipped": 0, "missing": []any{9999}, "warnings": []any{},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerSeedStatesFromVanilla(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "seed_states_from_vanilla",
		Arguments: map[string]any{"project_id": 7, "state_ids": []any{613, 9999}},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	stateIDs, _ := gotBody["state_ids"].([]any)
	if len(stateIDs) != 2 {
		t.Fatalf("PUT body state_ids = %+v, want [613, 9999] forwarded verbatim", gotBody["state_ids"])
	}

	out := result.StructuredContent.(map[string]any)
	missing, _ := out["missing"].([]any)
	if len(missing) != 1 || missing[0] != float64(9999) {
		t.Errorf("missing = %+v, want [9999]", out["missing"])
	}
}

func TestSeedStatesFromVanilla_UpstreamUnavailableSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Vanilla data not available on this server"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerSeedStatesFromVanilla(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "seed_states_from_vanilla",
		Arguments: map[string]any{"project_id": 7, "all_states": true},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 503")
	}
	if !strings.Contains(contentText(result), "Vanilla data not available") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}
