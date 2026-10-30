package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newProxyOnlyTestServer(t *testing.T, register func(*Server)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(toolHTTPHandler(func(*http.Request) *Server {
		server := NewServer()
		register(server)
		return server
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCreateMio_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/mios" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "mio_id": "USA_test_mio"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateMio(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_mio",
		Arguments: map[string]any{"project_id": 7, "mio_id": "USA_test_mio"},
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
	out := result.StructuredContent.(map[string]any)
	if out["mio_db_id"] != float64(12) {
		t.Errorf("mio_db_id = %v, want 12", out["mio_db_id"])
	}
	if out["mio_id"] != "USA_test_mio" {
		t.Errorf("mio_id = %v, want USA_test_mio", out["mio_id"])
	}
}

func TestCreateMio_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "mio_id 'USA_test_mio' already exists in this project."})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateMio(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_mio",
		Arguments: map[string]any{"project_id": 7, "mio_id": "USA_test_mio"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists in this project") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestAddMioTrait_ReadsExistingTraitsAndAppends(t *testing.T) {
	var gotPUTBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/mios/12" {
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 12, "mio_id": "USA_test_mio",
				"traits": []any{map[string]any{"token": "existing_trait", "kind": "initial"}},
			})
		case http.MethodPut:
			body, _ := readJSONBody(r)
			gotPUTBody = body
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 12, "mio_id": "USA_test_mio",
				"traits": []any{
					map[string]any{"token": "existing_trait", "kind": "initial"},
					map[string]any{"token": "new_trait", "kind": "add"},
				},
			})
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddMioTrait(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_mio_trait",
		Arguments: map[string]any{"project_id": 7, "mio_db_id": 12, "token": "new_trait"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	traits, _ := gotPUTBody["traits"].([]any)
	if len(traits) != 2 {
		t.Fatalf("PUT traits length = %d, want 2 (existing + appended, never replaced)", len(traits))
	}
	first, _ := traits[0].(map[string]any)
	if first["token"] != "existing_trait" {
		t.Errorf("PUT traits[0].token = %v, want existing_trait preserved", first["token"])
	}
	second, _ := traits[1].(map[string]any)
	if second["token"] != "new_trait" || second["kind"] != "add" {
		t.Errorf("PUT traits[1] = %+v, want the new trait with default kind=add", second)
	}

	out := result.StructuredContent.(map[string]any)
	if out["trait_count"] != float64(2) {
		t.Errorf("trait_count = %v, want 2", out["trait_count"])
	}
}

func TestAddMioTrait_UpstreamNotFoundSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "MIO not found"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddMioTrait(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_mio_trait",
		Arguments: map[string]any{"project_id": 7, "mio_db_id": 999, "token": "new_trait"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream GET 404s")
	}
	if !strings.Contains(contentText(result), "MIO not found") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestAddMioTrait_MissingTokenNeverReachesUpstream(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddMioTrait(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_mio_trait",
		Arguments: map[string]any{"project_id": 7, "mio_db_id": 12, "token": "  "},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for a blank token")
	}
	if called {
		t.Error("upstream was called despite a validation failure that should short-circuit locally")
	}
}

func TestCreateIntelligenceAgency_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/intelligence-agencies" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "agency_id": "usa_oss", "name_en": "OSS"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) {
		registerCreateIntelligenceAgency(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_intelligence_agency",
		Arguments: map[string]any{"project_id": 7, "agency_id": "usa_oss", "name_en": "OSS"},
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
	out := result.StructuredContent.(map[string]any)
	if out["agency_db_id"] != float64(5) {
		t.Errorf("agency_db_id = %v, want 5", out["agency_db_id"])
	}
	if out["agency_id"] != "usa_oss" {
		t.Errorf("agency_id = %v, want usa_oss", out["agency_id"])
	}
}

func TestCreateIntelligenceAgency_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "agency_id 'usa_oss' already exists in this project."})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateIntelligenceAgency(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_intelligence_agency",
		Arguments: map[string]any{"project_id": 7, "agency_id": "usa_oss"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists in this project") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestAddAgencyUpgrade_NewBranchIsCreatedAndExistingBranchesPreserved(t *testing.T) {
	var gotPUTBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/intelligence-agencies/5" {
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 5, "agency_id": "usa_oss",
				"upgrade_branches": []any{
					map[string]any{"branch": "other_branch", "upgrades": []any{map[string]any{"token": "other_tok"}}},
				},
			})
		case http.MethodPut:
			body, _ := readJSONBody(r)
			gotPUTBody = body
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 5, "agency_id": "usa_oss",
				"upgrade_branches": []any{
					map[string]any{"branch": "other_branch", "upgrades": []any{map[string]any{"token": "other_tok"}}},
					map[string]any{"branch": "espionage", "upgrades": []any{map[string]any{"token": "new_upgrade"}}},
				},
			})
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddAgencyUpgrade(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "add_agency_upgrade",
		Arguments: map[string]any{
			"project_id": 7, "agency_db_id": 5, "branch": "espionage", "token": "new_upgrade",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	branches, _ := gotPUTBody["upgrade_branches"].([]any)
	if len(branches) != 2 {
		t.Fatalf("PUT upgrade_branches length = %d, want 2 (existing branch preserved + new branch)", len(branches))
	}
	firstBranch, _ := branches[0].(map[string]any)
	if firstBranch["branch"] != "other_branch" {
		t.Errorf("PUT upgrade_branches[0].branch = %v, want other_branch preserved untouched", firstBranch["branch"])
	}
	secondBranch, _ := branches[1].(map[string]any)
	if secondBranch["branch"] != "espionage" {
		t.Errorf("PUT upgrade_branches[1].branch = %v, want espionage (newly created)", secondBranch["branch"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["branch_count"] != float64(2) {
		t.Errorf("branch_count = %v, want 2", out["branch_count"])
	}
}

func TestAddAgencyUpgrade_UpstreamNotFoundSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Intelligence agency 999 not found in project 7."})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddAgencyUpgrade(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_agency_upgrade",
		Arguments: map[string]any{"project_id": 7, "agency_db_id": 999, "branch": "espionage", "token": "new_upgrade"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream GET 404s")
	}
	if !strings.Contains(contentText(result), "not found") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestCreateSpecialProject_ProxiesToUpstreamBackend(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/special-projects/projects" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 3, "project_token": "jet_engine", "specialization": "aviation_track",
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateSpecialProject(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_special_project",
		Arguments: map[string]any{
			"project_id": 7, "project_token": "jet_engine", "specialization": "aviation_track",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["project_token"] != "jet_engine" {
		t.Errorf("upstream saw project_token = %v, want jet_engine", gotBody["project_token"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["special_project_db_id"] != float64(3) {
		t.Errorf("special_project_db_id = %v, want 3", out["special_project_db_id"])
	}
	if out["specialization"] != "aviation_track" {
		t.Errorf("specialization = %v, want aviation_track", out["specialization"])
	}
}

func TestCreateSpecialProject_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Special project 'jet_engine' already exists"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateSpecialProject(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_special_project",
		Arguments: map[string]any{"project_id": 7, "project_token": "jet_engine", "specialization": "aviation_track"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestCreateAutonomyState_ProxiesToUpstreamBackendAndSurfacesAutoSuffixWarning(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/autonomy-states" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "autonomy_id": "puppet_state_2",
			"warnings": []any{"autonomy_id 'puppet_state' collided · auto-suffixed to 'puppet_state_2'."},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateAutonomyState(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_autonomy_state",
		Arguments: map[string]any{"project_id": 7, "autonomy_id": "puppet_state"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	out := result.StructuredContent.(map[string]any)
	if out["autonomy_db_id"] != float64(9) {
		t.Errorf("autonomy_db_id = %v, want 9", out["autonomy_db_id"])
	}
	if out["autonomy_id"] != "puppet_state_2" {
		t.Errorf("autonomy_id = %v, want puppet_state_2 (auto-suffixed, read back)", out["autonomy_id"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %+v, want exactly 1 auto-suffix advisory", out["warnings"])
	}
}

func TestCreateAutonomyState_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "autonomy_id required"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateAutonomyState(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_autonomy_state",
		Arguments: map[string]any{"project_id": 7, "autonomy_id": ""},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 422")
	}
	if !strings.Contains(contentText(result), "autonomy_id required") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func readJSONBody(r *http.Request) (map[string]any, error) {
	var body map[string]any
	err := json.NewDecoder(r.Body).Decode(&body)
	return body, err
}
