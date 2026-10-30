package toolapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newProxyToolTestServer(t *testing.T, register func(*Server)) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	getServer := func(*http.Request) *Server {
		server := NewServer()
		register(server)
		return server
	}
	streamable := toolHTTPHandler(getServer)
	handler := streamable

	srv := httptest.NewUnstartedServer(handler)
	srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestUpdateCharacter_ProxiesToUpstreamBackend(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 55, "tag": "ITA_rommel", "role_type": "general"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerUpdateCharacter(s, proxy, "/api")
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_character",
		Arguments: map[string]any{
			"project_id": 7, "character_id": 55, "name": "Erwin Rommel", "skill": 5,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/projects/7/characters/55" {
		t.Errorf("upstream request = %s %s, want PUT /api/projects/7/characters/55", gotMethod, gotPath)
	}
	if gotBody["name"] != "Erwin Rommel" || gotBody["skill"] != float64(5) {
		t.Errorf("upstream body = %+v, want name/skill forwarded", gotBody)
	}
	if _, present := gotBody["tag"]; present {
		t.Errorf("upstream body = %+v, unsupplied field 'tag' must be omitted (patch semantics)", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["character_id"] != float64(55) {
		t.Errorf("character_id = %v, want 55", out["character_id"])
	}
	if out["tag"] != "ITA_rommel" || out["role_type"] != "general" {
		t.Errorf("reshaped output = %+v", out)
	}
}

func TestUpdateCharacter_ProjectServicePrefix(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 55, "tag": "ITA_rommel", "role_type": "general"})
	}))
	defer upstream.Close()
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerUpdateCharacter(s, NewRESTProxy(upstream.URL), "")
	})
	session := connectClient(t, srv.URL)
	if _, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_character",
		Arguments: map[string]any{"project_id": 7, "character_id": 55, "skill": 2},
	}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotPath != "/projects/7/characters/55" {
		t.Errorf("path = %s, want /projects/7/characters/55", gotPath)
	}
}

func TestUpdateCharacter_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Character not found"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerUpdateCharacter(s, proxy, "/api")
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_character",
		Arguments: map[string]any{"project_id": 7, "character_id": 999, "name": "Ghost"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "Character not found") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestAssignCharacterRole_ProxiesToUpstreamBackend(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 55, "tag": "ITA_rommel", "role_type": "general",
			"has_advisor_role": true, "advisor_slot": "political_advisor",
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerAssignCharacterRole(s, proxy, "/api")
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "assign_character_role",
		Arguments: map[string]any{
			"project_id": 7, "character_id": 55,
			"has_advisor_role": true, "advisor_slot": "political_advisor",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/projects/7/characters/55" {
		t.Errorf("upstream request = %s %s, want PUT /api/projects/7/characters/55", gotMethod, gotPath)
	}
	if gotBody["has_advisor_role"] != true || gotBody["advisor_slot"] != "political_advisor" {
		t.Errorf("upstream body = %+v, want has_advisor_role/advisor_slot forwarded", gotBody)
	}
	if _, present := gotBody["role_type"]; present {
		t.Errorf("upstream body = %+v, unsupplied field 'role_type' must be omitted (patch semantics)", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["has_advisor_role"] != true || out["advisor_slot"] != "political_advisor" {
		t.Errorf("reshaped output = %+v", out)
	}
}

func TestAssignCharacterRole_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "advisor_slot is not a recognized slot"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerAssignCharacterRole(s, proxy, "/api")
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "assign_character_role",
		Arguments: map[string]any{"project_id": 7, "character_id": 55, "advisor_slot": "bogus"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 400")
	}
	if !strings.Contains(contentText(result), "not a recognized slot") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}
