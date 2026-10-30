package toolapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTechSurfaceProxyTestServer(t *testing.T, proxy *RESTProxy) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	deps := Deps{Proxy: proxy}
	getServer := func(*http.Request) *Server {
		server := NewServer()
		registerTechSurfaceProxyTools(server, deps.Proxy)
		return server
	}
	handler := toolHTTPHandler(getServer)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

type techSurfaceObservedRequest struct {
	method string
	path   string
	body   map[string]any
}

func TestTechSurfaceProxyToolsUseCanonicalMethodsPathsAndSafeBodies(t *testing.T) {
	requests := make([]techSurfaceObservedRequest, 0, 9)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item := techSurfaceObservedRequest{method: r.Method, path: r.URL.Path}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&item.body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		}
		requests = append(requests, item)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/technologies/surface/branches" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "MOD_root"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": "MOD_root"})
	}))
	defer upstream.Close()

	srv := newTechSurfaceProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)
	calls := []struct {
		name string
		args map[string]any
	}{
		{"get_technology_surface_environment", map[string]any{"project_id": 7}},
		{"confirm_technology_surface_environment", map[string]any{"project_id": 7, "compatibility_mode": "all_players", "no_step_back": true, "by_blood_alone": false, "man_the_guns": true}},
		{"list_technology_surface_branches", map[string]any{"project_id": 7}},
		{"get_technology_surface_branch", map[string]any{"project_id": 7, "branch_id": "MOD_root"}},
		{"create_technology_surface_branch", map[string]any{
			"project_id": 7, "entry_id": "army", "target_id": "infantry",
			"root": map[string]any{"tech_id": "MOD_root", "name": "Root", "description_locs": map[string]any{"en": "Root desc"}},
		}},
		{"update_technology_surface_branch", map[string]any{"project_id": 7, "branch_id": "MOD_root", "name": "Renamed", "enable_equipments": []string{"MOD_rifle"}}},
		{"delete_technology_surface_branch", map[string]any{"project_id": 7, "branch_id": "MOD_root"}},
		{"add_technology_surface_node", map[string]any{
			"project_id": 7, "branch_id": "MOD_root", "parent_technology_id": "MOD_root",
			"technology": map[string]any{"tech_id": "MOD_child", "name": "Child", "description_locs": map[string]any{"en": "Child desc"}},
		}},
		{"repair_technology_surface_timeline", map[string]any{"project_id": 7}},
	}
	for _, call := range calls {
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: call.name, Arguments: call.args})
		if err != nil {
			t.Fatalf("%s CallTool: %v", call.name, err)
		}
		if result.IsError {
			t.Fatalf("%s unexpected tool error: %s", call.name, contentText(result))
		}
	}

	want := []techSurfaceObservedRequest{
		{method: http.MethodGet, path: "/api/projects/7/technologies/surface"},
		{method: http.MethodPut, path: "/api/projects/7/technologies/surface/environment", body: map[string]any{"compatibility_mode": "all_players", "dlc": map[string]any{"no_step_back": true, "by_blood_alone": false, "man_the_guns": true}}},
		{method: http.MethodGet, path: "/api/projects/7/technologies/surface/branches"},
		{method: http.MethodGet, path: "/api/projects/7/technologies/surface/branches/MOD_root"},
		{method: http.MethodPost, path: "/api/projects/7/technologies/surface/branches", body: map[string]any{
			"id": "MOD_root", "entry_id": "army", "target_id": "infantry",
			"root": map[string]any{"tech_id": "MOD_root", "name": "Root", "description_locs": map[string]any{"en": "Root desc"}},
		}},
		{method: http.MethodPut, path: "/api/projects/7/technologies/surface/branches/MOD_root", body: map[string]any{"name": "Renamed", "enable_equipments": []any{"MOD_rifle"}}},
		{method: http.MethodDelete, path: "/api/projects/7/technologies/surface/branches/MOD_root"},
		{method: http.MethodPost, path: "/api/projects/7/technologies/surface/branches/MOD_root/technologies", body: map[string]any{
			"parent_technology_id": "MOD_root", "technology": map[string]any{"tech_id": "MOD_child", "name": "Child", "description_locs": map[string]any{"en": "Child desc"}},
		}},
		{method: http.MethodPost, path: "/api/projects/7/technologies/surface/repair"},
	}
	if len(requests) != len(want) {
		t.Fatalf("request count = %d, want %d: %#v", len(requests), len(want), requests)
	}
	for i := range want {
		if requests[i].method != want[i].method || requests[i].path != want[i].path {
			t.Errorf("request[%d] = %s %s, want %s %s", i, requests[i].method, requests[i].path, want[i].method, want[i].path)
		}
		gotBody, _ := json.Marshal(requests[i].body)
		wantBody, _ := json.Marshal(want[i].body)
		if string(gotBody) != string(wantBody) {
			t.Errorf("request[%d] body = %s, want %s", i, gotBody, wantBody)
		}
	}
}

func TestCreateTechnologySurfaceBranchRejectsProfileOwnedFieldsBeforeProxy(t *testing.T) {
	assertTechSurfaceFieldsRejected(t, "create_technology_surface_branch", func(field string) map[string]any {
		args := map[string]any{
			"project_id": 7, "entry_id": "army", "target_id": "infantry",
			"root": map[string]any{"tech_id": "MOD_root", "description_locs": map[string]any{"en": "desc"}},
		}
		if field == "surface" || field == "slot" || field == "format" || field == "slot_size" || field == "x" || field == "y" {
			args[field] = "forbidden"
		} else {
			args["root"].(map[string]any)[field] = "forbidden"
		}
		return args
	}, []string{"surface", "slot", "format", "slot_size", "x", "y", "folder", "tech_line", "start_year", "position_x", "position_y", "folder_placements", "path", "source"})
}

func TestAddTechnologySurfaceNodeRejectsProfileOwnedFieldsBeforeProxy(t *testing.T) {
	assertTechSurfaceFieldsRejected(t, "add_technology_surface_node", func(field string) map[string]any {
		technology := map[string]any{"tech_id": "MOD_child", "description_locs": map[string]any{"en": "desc"}, field: "forbidden"}
		return map[string]any{"project_id": 7, "branch_id": "MOD_root", "parent_technology_id": "MOD_root", "technology": technology}
	}, []string{"surface", "folder", "slot", "format", "slot_size", "x", "y", "tech_line", "start_year", "position_x", "position_y", "folder_placements", "path", "source"})
}

func TestUpdateTechnologySurfaceBranchRejectsProfileOwnedFieldsBeforeProxy(t *testing.T) {
	assertTechSurfaceFieldsRejected(t, "update_technology_surface_branch", func(field string) map[string]any {
		return map[string]any{"project_id": 7, "branch_id": "MOD_root", "name": "safe", field: "forbidden"}
	}, []string{"surface", "folder", "slot", "gridbox_id", "format", "slot_size", "x", "y", "tech_line", "start_year", "position_x", "position_y", "folder_placements", "path", "source"})
}

func assertTechSurfaceFieldsRejected(t *testing.T, toolName string, arguments func(string) map[string]any, fields []string) {
	t.Helper()
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer upstream.Close()
	srv := newTechSurfaceProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)
	for _, field := range fields {
		called = false
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: toolName, Arguments: arguments(field)})
		if err != nil {
			t.Fatalf("%s field %s CallTool: %v", toolName, field, err)
		}
		if !result.IsError {
			t.Errorf("%s accepted profile-owned field %q", toolName, field)
		}
		if called {
			t.Errorf("%s forwarded profile-owned field %q to the project service", toolName, field)
		}
	}
}
