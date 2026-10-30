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

func newMilitaryOOBTestServer(t *testing.T, buildDeps func(host string) Deps) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	host := listener.Addr().String()
	deps := buildDeps(host)

	getServer := func(*http.Request) *Server {
		server := NewServer()
		registerCreateSubUnit(server, deps.Proxy)
		registerCreateAirWingGroup(server, deps.Proxy)
		registerAddAirWing(server, deps.Proxy)
		registerCreateFleet(server, deps.Proxy)
		registerAddTaskForce(server, deps.Proxy)
		registerAddShip(server, deps.Proxy)
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

func assertUpstreamErrorSurfaces(t *testing.T, toolName string, args map[string]any, detail string) {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": detail})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{Name: toolName, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatalf("%s: expected IsError=true when the upstream returns 409", toolName)
	}
	if !strings.Contains(contentText(result), detail) {
		t.Errorf("%s: error text = %q, want it to contain %q", toolName, contentText(result), detail)
	}
}

func TestCreateSubUnit_ProxiesToUpstreamBackendAndReshapes(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 55, "sub_unit_id": "zzz_mtn_inf_1", "country_tag": "ITA", "group": "infantry",
		})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_sub_unit",
		Arguments: map[string]any{
			"project_id":  7,
			"sub_unit_id": "zzz_mtn_inf",
			"country_tag": "ITA",
			"cavalry":     true,
			"unit_types":  []string{"infantry"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/sub-units" {
		t.Errorf("unexpected upstream request: %s %s", gotMethod, gotPath)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}
	if gotBody["sub_unit_id"] != "zzz_mtn_inf" {
		t.Errorf("body[sub_unit_id] = %v, want zzz_mtn_inf", gotBody["sub_unit_id"])
	}
	if gotBody["cavalry"] != true {
		t.Errorf("body[cavalry] = %v, want true", gotBody["cavalry"])
	}

	if v, ok := gotBody["marines"]; !ok || v != nil {
		t.Errorf("body[marines] = %v (present=%v), want present and null (omitted tri-state)", v, ok)
	}

	if cats, ok := gotBody["categories"].([]any); !ok || len(cats) != 0 {
		t.Errorf("body[categories] = %v, want an empty array (not null)", gotBody["categories"])
	}
	if need, ok := gotBody["need"].(map[string]any); !ok || len(need) != 0 {
		t.Errorf("body[need] = %v, want an empty object (not null)", gotBody["need"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["sub_unit_db_id"] != float64(55) {
		t.Errorf("sub_unit_db_id = %v, want 55", out["sub_unit_db_id"])
	}
	if out["sub_unit_id"] != "zzz_mtn_inf_1" {
		t.Errorf("sub_unit_id = %v, want the auto-suffixed zzz_mtn_inf_1 read back from upstream", out["sub_unit_id"])
	}
	if out["country_tag"] != "ITA" || out["group"] != "infantry" {
		t.Errorf("country_tag/group = %v/%v, want ITA/infantry", out["country_tag"], out["group"])
	}
}

func TestCreateSubUnit_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	assertUpstreamErrorSurfaces(t, "create_sub_unit",
		map[string]any{"project_id": 7, "sub_unit_id": "zzz_mtn_inf"},
		"sub_unit_id 'zzz_mtn_inf' already exists")
}

func TestCreateAirWingGroup_ProxiesToUpstreamBackendAndReshapes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 21, "location": "798", "oob_file": "USA_1936", "country_tag": "USA",
		})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_air_wing_group",
		Arguments: map[string]any{
			"project_id": 7, "location": "798", "oob_file": "USA_1936", "country_tag": "USA", "ordering": 1,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/oob-units/air-wing-groups" {
		t.Errorf("unexpected upstream request: %s %s", gotMethod, gotPath)
	}
	if gotBody["location"] != "798" || gotBody["oob_file"] != "USA_1936" {
		t.Errorf("body location/oob_file = %v/%v", gotBody["location"], gotBody["oob_file"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["air_wing_group_id"] != float64(21) {
		t.Errorf("air_wing_group_id = %v, want 21", out["air_wing_group_id"])
	}
}

func TestCreateAirWingGroup_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	assertUpstreamErrorSurfaces(t, "create_air_wing_group",
		map[string]any{"project_id": 7, "location": "798"},
		"Project not found")
}

func TestAddAirWing_ProxiesToUpstreamBackendAndReshapes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 5, "group_id": 21, "equipment_type": "fighter_equipment_0", "amount": "100",
		})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "add_air_wing",
		Arguments: map[string]any{
			"project_id": 7, "group_id": 21, "equipment_type": "fighter_equipment_0", "amount": "100",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/oob-units/air-wings" {
		t.Errorf("unexpected upstream request: %s %s", gotMethod, gotPath)
	}
	if gotBody["group_id"] != float64(21) || gotBody["equipment_type"] != "fighter_equipment_0" {
		t.Errorf("body group_id/equipment_type = %v/%v", gotBody["group_id"], gotBody["equipment_type"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["air_wing_id"] != float64(5) || out["group_id"] != float64(21) {
		t.Errorf("air_wing_id/group_id = %v/%v, want 5/21", out["air_wing_id"], out["group_id"])
	}
}

func TestAddAirWing_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	assertUpstreamErrorSurfaces(t, "add_air_wing",
		map[string]any{"project_id": 7, "group_id": 999, "equipment_type": "fighter_equipment_0"},
		"Air wing group not found")
}

func TestCreateFleet_ProxiesToUpstreamBackendAndReshapes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 3, "name": "Home Fleet", "naval_base": "9601", "oob_file": "USA_1936",
		})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_fleet",
		Arguments: map[string]any{
			"project_id": 7, "name": "Home Fleet", "naval_base": "9601", "oob_file": "USA_1936",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/oob-units/fleets" {
		t.Errorf("unexpected upstream request: %s %s", gotMethod, gotPath)
	}
	if gotBody["naval_base"] != "9601" {
		t.Errorf("body[naval_base] = %v, want 9601", gotBody["naval_base"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["fleet_id"] != float64(3) {
		t.Errorf("fleet_id = %v, want 3", out["fleet_id"])
	}
	if out["naval_base"] != "9601" || out["oob_file"] != "USA_1936" {
		t.Errorf("naval_base/oob_file = %v/%v", out["naval_base"], out["oob_file"])
	}
}

func TestCreateFleet_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	assertUpstreamErrorSurfaces(t, "create_fleet",
		map[string]any{"project_id": 7, "naval_base": "9601"},
		"Project not found")
}

func TestAddTaskForce_ProxiesToUpstreamBackendAndReshapes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "fleet_id": 3, "name": "TF1", "location": "9601",
		})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "add_task_force",
		Arguments: map[string]any{
			"project_id": 7, "fleet_id": 3, "name": "TF1", "location": "9601",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/oob-units/task-forces" {
		t.Errorf("unexpected upstream request: %s %s", gotMethod, gotPath)
	}
	if gotBody["fleet_id"] != float64(3) {
		t.Errorf("body[fleet_id] = %v, want 3", gotBody["fleet_id"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["task_force_id"] != float64(9) || out["fleet_id"] != float64(3) {
		t.Errorf("task_force_id/fleet_id = %v/%v, want 9/3", out["task_force_id"], out["fleet_id"])
	}
}

func TestAddTaskForce_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	assertUpstreamErrorSurfaces(t, "add_task_force",
		map[string]any{"project_id": 7, "fleet_id": 999},
		"Fleet not found")
}

func TestAddShip_ProxiesToUpstreamBackendAndReshapes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 11, "task_force_id": 9, "name": "Ship1", "definition": "destroyer",
			"equipment_type": "destroyer_hull", "version_name": "Variant1",
		})
	}))
	defer upstream.Close()

	srv := newMilitaryOOBTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "add_ship",
		Arguments: map[string]any{
			"project_id": 7, "task_force_id": 9, "name": "Ship1", "definition": "destroyer",
			"equipment_type": "destroyer hull v2", "version_name": "Variant1",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/oob-units/ships" {
		t.Errorf("unexpected upstream request: %s %s", gotMethod, gotPath)
	}
	if gotBody["equipment_type"] != "destroyer hull v2" {
		t.Errorf("body[equipment_type] = %q, want the raw unvalidated value forwarded verbatim "+
			"(known Python bug, not this proxy's to fix)", gotBody["equipment_type"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["ship_id"] != float64(11) || out["task_force_id"] != float64(9) {
		t.Errorf("ship_id/task_force_id = %v/%v, want 11/9", out["ship_id"], out["task_force_id"])
	}
	if out["version_name"] != "Variant1" {
		t.Errorf("version_name = %v, want Variant1", out["version_name"])
	}
}

func TestAddShip_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	assertUpstreamErrorSurfaces(t, "add_ship",
		map[string]any{"project_id": 7, "task_force_id": 999},
		"Task force not found")
}
