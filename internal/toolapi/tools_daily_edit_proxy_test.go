package toolapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type dailyEditRequest struct {
	method string
	path   string
	body   map[string]any
}

func TestDailyEditToolsReachRealProjectServiceRoutes(t *testing.T) {
	var mu sync.Mutex
	requests := []dailyEditRequest{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requests = append(requests, dailyEditRequest{method: r.Method, path: r.URL.Path, body: body})
		mu.Unlock()

		id := int64(1)
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if r.Method == http.MethodPut && len(parts) > 0 {
			id, _ = strconv.ParseInt(parts[len(parts)-1], 10, 64)
		}
		response := map[string]any{"id": id}
		for key, value := range body {
			response[key] = value
		}
		switch {
		case strings.Contains(r.URL.Path, "decisions/categories"):
			response["category_id"] = "CHI_category"
		case strings.Contains(r.URL.Path, "events/namespaces"):
			response["namespace"] = "chi"
		case strings.Contains(r.URL.Path, "/mios"):
			response["mio_id"] = "CHI_mio"
		case strings.Contains(r.URL.Path, "intelligence-agencies"):
			response["agency_id"] = "chi_agency"
		case strings.Contains(r.URL.Path, "specializations"):
			response["specialization_id"] = "specialization_nuclear"
		case strings.Contains(r.URL.Path, "special-projects/projects"):
			response["project_token"] = "sp_reactor"
		case strings.Contains(r.URL.Path, "raw-files"):
			response["dest_path"] = "common/on_actions/chi_agent.txt"
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(server *Server) {
		registerDailyEditTools(server, proxy)
	})
	session := connectClient(t, srv.URL)

	tests := []struct {
		name, method, path, changedField string
		args                             map[string]any
	}{
		{"update_decision_category", "PUT", "/api/projects/7/decisions/categories/2", "priority", map[string]any{"project_id": 7, "id": 2, "priority": 9}},
		{"update_event_namespace", "PUT", "/api/projects/7/events/namespaces/3", "country_tag", map[string]any{"project_id": 7, "id": 3, "country_tag": "CHI"}},
		{"update_mio", "PUT", "/api/projects/7/mios/4", "icon", map[string]any{"project_id": 7, "id": 4, "icon": "GFX_mio_chi"}},
		{"update_intelligence_agency", "PUT", "/api/projects/7/intelligence-agencies/5", "picture", map[string]any{"project_id": 7, "id": 5, "picture": "GFX_agency_chi"}},
		{"create_specialization", "POST", "/api/projects/7/special-projects/specializations", "specialization_id", map[string]any{"project_id": 7, "specialization_id": "specialization_nuclear"}},
		{"update_specialization", "PUT", "/api/projects/7/special-projects/specializations/6", "facility_model", map[string]any{"project_id": 7, "id": 6, "facility_model": "nuclear_facility"}},
		{"update_special_project", "PUT", "/api/projects/7/special-projects/projects/7", "complexity", map[string]any{"project_id": 7, "id": 7, "complexity": "complex"}},
		{"update_air_wing_group", "PUT", "/api/projects/7/oob-units/air-wing-groups/8", "location", map[string]any{"project_id": 7, "id": 8, "location": "608"}},
		{"update_air_wing", "PUT", "/api/projects/7/oob-units/air-wings/9", "amount", map[string]any{"project_id": 7, "id": 9, "amount": "120"}},
		{"update_fleet", "PUT", "/api/projects/7/oob-units/fleets/10", "naval_base", map[string]any{"project_id": 7, "id": 10, "naval_base": "10000"}},
		{"update_task_force", "PUT", "/api/projects/7/oob-units/task-forces/11", "name", map[string]any{"project_id": 7, "id": 11, "name": "第一舰队"}},
		{"update_ship", "PUT", "/api/projects/7/oob-units/ships/12", "version_name", map[string]any{"project_id": 7, "id": 12, "version_name": "1936 Model"}},
		{"create_raw_file", "POST", "/api/projects/7/raw-files", "content", map[string]any{"project_id": 7, "dest_path": "common/on_actions/chi_agent.txt", "content": "on_actions = { }"}},
		{"update_raw_file", "PUT", "/api/projects/7/raw-files/13", "note", map[string]any{"project_id": 7, "id": 13, "note": "agent edit"}},
	}

	for index, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &CallToolParams{Name: tc.name, Arguments: tc.args})
			if err != nil || result.IsError {
				t.Fatalf("CallTool err=%v content=%+v", err, result.Content)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(requests) != index+1 {
				t.Fatalf("request count=%d, want %d", len(requests), index+1)
			}
			got := requests[index]
			if got.method != tc.method || got.path != tc.path {
				t.Fatalf("route=%s %s, want %s %s", got.method, got.path, tc.method, tc.path)
			}
			if _, ok := got.body[tc.changedField]; !ok {
				t.Errorf("body dropped %q: %+v", tc.changedField, got.body)
			}
			if _, leaked := got.body["project_id"]; leaked {
				t.Errorf("project_id leaked into entity payload: %+v", got.body)
			}
		})
	}
}

func TestBatchUpdateEntitiesUsesNestedFocusRouteAndReportsPartialFailures(t *testing.T) {
	var paths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		body, _ := readJSONBody(r)
		body["id"] = 11
		body["focus_id"] = "CHI_first"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(server *Server) {
		registerBatchUpdateEntities(server, proxy)
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "batch_update_entities",
		Arguments: map[string]any{
			"project_id": 7,
			"operations": []any{
				map[string]any{"kind": "focus", "id": 11, "parent_id": 5, "fields": map[string]any{"cost": 5}},
				map[string]any{"kind": "not_a_kind", "id": 1, "fields": map[string]any{"name": "bad"}},
				map[string]any{"kind": "focus", "id": 12, "parent_id": 5, "fields": map[string]any{"x": 2}},
			},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool err=%v result=%+v", err, result)
	}
	out := result.StructuredContent.(map[string]any)
	if out["requested"] != float64(3) || out["attempted"] != float64(3) || out["updated"] != float64(2) || out["failed"] != float64(1) {
		t.Fatalf("batch counts = %+v", out)
	}
	want := "/api/projects/7/focus-trees/5/nodes/"
	if len(paths) != 2 || paths[0] != want+"11" || paths[1] != want+"12" {
		t.Errorf("focus paths = %v", paths)
	}
}

func TestBatchUpdateEntitiesRejectsServerOwnedFieldsBeforeWrite(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer upstream.Close()
	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(server *Server) { registerBatchUpdateEntities(server, proxy) })
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "batch_update_entities", Arguments: map[string]any{"project_id": 7,
			"operations": []any{map[string]any{"kind": "country", "id": 2, "fields": map[string]any{"project_id": 999}}}},
	})
	if err != nil || result.IsError {
		t.Fatalf("batch reports item failures as structured output: err=%v content=%+v", err, result.Content)
	}
	if called {
		t.Fatal("server-owned field reached upstream")
	}
	out := result.StructuredContent.(map[string]any)
	if out["failed"] != float64(1) {
		t.Fatalf("output = %s", fmt.Sprint(out))
	}
}
