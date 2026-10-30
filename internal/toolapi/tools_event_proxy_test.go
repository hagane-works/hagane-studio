package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateEventNamespace_ProxiesToUpstreamBackend(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "namespace": "usa_politics", "country_tag": "USA"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerCreateEventNamespace(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_event_namespace",
		Arguments: map[string]any{"project_id": 7, "namespace": "usa_politics", "country_tag": "USA"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/events/namespaces" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/events/namespaces", gotMethod, gotPath)
	}
	if gotBody["namespace"] != "usa_politics" || gotBody["country_tag"] != "USA" || gotBody["scope_kind"] != "own" {
		t.Errorf("upstream body = %+v, want namespace/country_tag forwarded + scope_kind defaulted to 'own'", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["namespace_id"] != float64(3) {
		t.Errorf("namespace_id = %v, want 3", out["namespace_id"])
	}
}

func TestCreateEventNamespace_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "namespace 'usa_politics' already exists in this project. Pick a unique name."})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerCreateEventNamespace(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_event_namespace",
		Arguments: map[string]any{"project_id": 7, "namespace": "usa_politics"},
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

func TestCreateEvent_ProxiesToUpstreamBackend(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 21, "event_id": "usa_politics.1", "event_number": 1,
			"warnings": []any{"event_id auto-suffixed to usa_politics.1"},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerCreateEvent(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_event",
		Arguments: map[string]any{
			"project_id": 7, "namespace_id": 3, "event_id": "usa_politics.1", "event_number": 1,
			"title": "A New Deal",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/events" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/events", gotMethod, gotPath)
	}
	if gotBody["event_type"] != "country_event" || gotBody["is_triggered_only"] != true {
		t.Errorf("upstream body = %+v, want event_type/is_triggered_only defaulted", gotBody)
	}
	if gotBody["title"] != "A New Deal" {
		t.Errorf("upstream body = %+v, want title forwarded", gotBody)
	}
	if _, ok := gotBody["options"].([]any); !ok {
		t.Errorf("upstream body = %+v, want options defaulted to []", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["event_db_id"] != float64(21) || out["event_id"] != "usa_politics.1" || out["event_number"] != float64(1) {
		t.Errorf("reshaped output = %+v", out)
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 {
		t.Errorf("warnings = %+v, want 1 entry echoed back", out["warnings"])
	}
}

func TestCreateEvent_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Namespace not found in this project"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerCreateEvent(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_event",
		Arguments: map[string]any{"project_id": 7, "namespace_id": 999, "event_id": "x.1", "event_number": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 400")
	}
	if !strings.Contains(contentText(result), "Namespace not found in this project") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestUpdateEvent_ProxiesToUpstreamBackend(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 21, "event_id": "usa_politics.1"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerUpdateEvent(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_event",
		Arguments: map[string]any{"project_id": 7, "event_db_id": 21, "title": "New Title"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/projects/7/events/21" {
		t.Errorf("upstream request = %s %s, want PUT /api/projects/7/events/21", gotMethod, gotPath)
	}
	if gotBody["title"] != "New Title" {
		t.Errorf("upstream body = %+v, want title forwarded", gotBody)
	}
	if len(gotBody) != 1 {
		t.Errorf("upstream body = %+v, want ONLY the supplied field (patch semantics)", gotBody)
	}

	out := result.StructuredContent.(map[string]any)
	if out["event_db_id"] != float64(21) || out["event_id"] != "usa_politics.1" {
		t.Errorf("reshaped output = %+v", out)
	}
}

func TestUpdateEvent_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Event not found"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerUpdateEvent(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_event",
		Arguments: map[string]any{"project_id": 7, "event_db_id": 999, "title": "Ghost"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "Event not found") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}

func TestAddEventOption_ProxiesGetThenPutToUpstreamBackend(t *testing.T) {
	var calls []string
	var putBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 21, "event_id": "usa_politics.1",
				"options": []any{map[string]any{"name": "existing_option", "trigger": "", "effects": "", "hidden_effects": "", "ai_chance": map[string]any{}}},
			})
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 21, "event_id": "usa_politics.1",
				"options": []any{
					map[string]any{"name": "existing_option"},
					map[string]any{"name": "new_option"},
				},
			})
		}
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerAddEventOption(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_event_option",
		Arguments: map[string]any{"project_id": 7, "event_db_id": 21, "name": "new_option", "effects": "add_political_power = 50"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if len(calls) != 2 || calls[0] != "GET /api/projects/7/events/21" || calls[1] != "PUT /api/projects/7/events/21" {
		t.Errorf("upstream calls = %v, want a GET (read current options) followed by a PUT (replace with appended list)", calls)
	}
	opts, _ := putBody["options"].([]any)
	if len(opts) != 2 {
		t.Fatalf("PUT options = %+v, want the existing option PLUS the new one (append, not replace-from-scratch)", putBody["options"])
	}
	appended, _ := opts[1].(map[string]any)
	if appended["name"] != "new_option" || appended["effects"] != "add_political_power = 50" {
		t.Errorf("appended option = %+v", appended)
	}

	out := result.StructuredContent.(map[string]any)
	if out["option_count"] != float64(2) {
		t.Errorf("option_count = %v, want 2", out["option_count"])
	}
}

func TestAddEventOption_UpstreamGetErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Event not found"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyToolTestServer(t, func(s *Server) {
		registerAddEventOption(s, proxy)
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_event_option",
		Arguments: map[string]any{"project_id": 7, "event_db_id": 999, "name": "opt"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the GET returns 404")
	}
	if !strings.Contains(contentText(result), "Event not found") {
		t.Errorf("error text = %q, want the upstream's own detail relayed", contentText(result))
	}
}
