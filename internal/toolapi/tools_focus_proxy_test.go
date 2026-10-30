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

func newFocusProxyHandler(deps Deps) http.Handler {
	getServer := func(*http.Request) *Server {
		server := NewServer()
		if deps.Proxy != nil {
			registerCreateFocusTree(server, deps.Proxy)
			registerCreateFocus(server, deps.Proxy)
			registerUpdateFocus(server, deps.Proxy)
			registerConnectFocus(server, deps.Proxy)
			registerSetFocusMutualExclusion(server, deps.Proxy)
			registerDeleteFocus(server, deps.Proxy)
		}
		return server
	}
	streamable := toolHTTPHandler(getServer)
	return streamable
}

func newFocusProxyTestServer(t *testing.T, buildDeps func(host string) Deps) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	host := listener.Addr().String()
	srv := httptest.NewUnstartedServer(newFocusProxyHandler(buildDeps(host)))
	srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestCreateFocusTree_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/focus-trees" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 21, "tree_id": "ita_main", "country_tag": "ITA"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_focus_tree",
		Arguments: map[string]any{"project_id": 7, "country_tag": "ITA", "tree_id": "ita_main"},
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
	if gotBody["on_conflict"] != "error" {
		t.Errorf("explicit tree id must reject duplicate creation: %+v", gotBody)
	}
	if gotBody["country_tag"] != "ITA" || gotBody["tree_id"] != "ita_main" {
		t.Errorf("upstream body = %+v, want country_tag=ITA tree_id=ita_main", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["focus_tree_id"] != float64(21) {
		t.Errorf("focus_tree_id = %v, want 21", out["focus_tree_id"])
	}
	if out["tree_id"] != "ita_main" {
		t.Errorf("tree_id = %v, want ita_main", out["tree_id"])
	}
	if out["country_tag"] != "ITA" {
		t.Errorf("country_tag = %v, want ITA", out["country_tag"])
	}
}

func TestCreateFocusTree_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Focus tree ita_main already exists in this project"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_focus_tree",
		Arguments: map[string]any{"project_id": 7, "country_tag": "ITA", "tree_id": "ita_main"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 409")
	}
	if !strings.Contains(contentText(result), "already exists in this project") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}

func TestCreateFocus_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/focus-trees/21/nodes" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 55, "focus_id": "ita_new_focus"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_focus",
		Arguments: map[string]any{
			"project_id": 7, "focus_tree_id": 21, "name": "New Focus",
			"name_zh": "新国策", "desc_zh": "中文说明",
			"display_name_locs":   map[string]any{"zh": "新国策"},
			"effect_tooltip_locs": map[string]any{"zh": "获得陆军经验"},
		},
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
	if gotBody["name"] != "New Focus" {
		t.Errorf("upstream body name = %v, want %q", gotBody["name"], "New Focus")
	}
	if gotBody["name_zh"] != "新国策" || gotBody["desc_zh"] != "中文说明" {
		t.Errorf("upstream localisation fields = %+v", gotBody)
	}
	if locs, ok := gotBody["display_name_locs"].(map[string]any); !ok || locs["zh"] != "新国策" {
		t.Errorf("upstream display_name_locs = %#v", gotBody["display_name_locs"])
	}
	if gotBody["icon"] != "GFX_goal_unknown" {
		t.Errorf("upstream body icon = %v, want default GFX_goal_unknown", gotBody["icon"])
	}
	if gotBody["cost"] != float64(10) {
		t.Errorf("upstream body cost = %v, want default 10", gotBody["cost"])
	}
	if _, isNull := gotBody["prerequisites"]; !isNull {
		t.Fatalf("upstream body missing prerequisites key entirely: %+v", gotBody)
	}
	if prereqs, ok := gotBody["prerequisites"].([]any); !ok || len(prereqs) != 0 {
		t.Errorf("upstream body prerequisites = %v, want empty array (not null)", gotBody["prerequisites"])
	}
	if mutex, ok := gotBody["mutually_exclusive"].([]any); !ok || len(mutex) != 0 {
		t.Errorf("upstream body mutually_exclusive = %v, want empty array (not null)", gotBody["mutually_exclusive"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["focus_node_id"] != float64(55) {
		t.Errorf("focus_node_id = %v, want 55", out["focus_node_id"])
	}
	if out["focus_id"] != "ita_new_focus" {
		t.Errorf("focus_id = %v, want ita_new_focus", out["focus_id"])
	}
	if out["name"] != "New Focus" || out["icon"] != "GFX_goal_unknown" || out["cost"] != float64(10) {
		t.Errorf("saved authoring fields = %+v", out)
	}
	if out["completion_reward"] != "" || out["prerequisite_expression"] != "none" {
		t.Errorf("saved effect/relationship fields = %+v", out)
	}
}

func TestCreateFocus_RecoversExactCommittedNodeAfterAmbiguousConflict(t *testing.T) {
	var posts, gets int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			posts++
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"detail": "focus_id 'per_done' already exists in this project"})
			return
		}
		gets++
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 77, "focus_id": "per_done", "name": "完成", "icon": "GFX_goal_unknown",
			"cost": 10, "x": 3, "y": 4, "prerequisites": []any{},
			"mutually_exclusive": []any{}, "completion_reward": "", "name_zh": "完成",
			"display_name_locs": map[string]any{"zh": "完成"},
		}})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_focus", Arguments: map[string]any{
			"project_id": 7, "focus_tree_id": 21, "focus_id": "per_done", "name": "完成",
			"name_zh": "完成", "display_name_locs": map[string]any{"zh": "完成"}, "x": 3, "y": 4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("exact committed retry should recover, got %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if posts != 1 || gets != 1 || out["focus_node_id"] != float64(77) || out["focus_id"] != "per_done" {
		t.Fatalf("posts=%d gets=%d out=%#v", posts, gets, out)
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) == 0 || !strings.Contains(warnings[0].(string), "do not retry") {
		t.Fatalf("recovery warning = %#v", out["warnings"])
	}
}

func TestCreateFocus_DoesNotRecoverDifferentExistingNode(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"detail": "focus_id already exists"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id": 77, "focus_id": "per_done", "name": "Different", "icon": "GFX_goal_unknown",
			"cost": 10, "x": 3, "y": 4, "prerequisites": []any{}, "mutually_exclusive": []any{},
		}})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	result, err := connectClient(t, srv.URL).CallTool(context.Background(), &CallToolParams{
		Name: "create_focus", Arguments: map[string]any{
			"project_id": 7, "focus_tree_id": 21, "focus_id": "per_done", "name": "完成", "x": 3, "y": 4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(contentText(result), "already exists") {
		t.Fatalf("different row must remain a conflict: %+v", result)
	}
}

func TestFocusPrerequisiteExpressionMakesAndOrUnambiguous(t *testing.T) {
	tests := []struct {
		name   string
		groups [][]string
		want   string
	}{
		{name: "none", groups: nil, want: "none"},
		{name: "one", groups: [][]string{{"per_start"}}, want: "per_start"},
		{name: "or merge", groups: [][]string{{"per_armor", "per_refinery"}}, want: "(per_armor OR per_refinery)"},
		{name: "and requirements", groups: [][]string{{"per_armor"}, {"per_refinery"}}, want: "per_armor AND per_refinery"},
		{name: "empty group", groups: [][]string{{}}, want: "UNSATISFIABLE(empty group)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := focusPrerequisiteExpression(tc.groups); got != tc.want {
				t.Fatalf("focusPrerequisiteExpression(%v) = %q, want %q", tc.groups, got, tc.want)
			}
		})
	}
}

func TestCreateFocus_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "focus_id auto-suffix exhausted 1000 attempts"})
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
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 422")
	}
	if !strings.Contains(contentText(result), "auto-suffix exhausted") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}

func TestUpdateFocus_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/focus-trees/21/nodes/55" || r.Method != http.MethodPut {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 55, "focus_id": "ita_new_focus"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "node_id": 55, "name": "Renamed", "cost": 5.0},
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
	if len(gotBody) != 2 {
		t.Fatalf("upstream body = %+v, want exactly {name, cost} — unsupplied fields must be absent, not null", gotBody)
	}
	if gotBody["name"] != "Renamed" || gotBody["cost"] != float64(5) {
		t.Errorf("upstream body = %+v, want name=Renamed cost=5", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["focus_node_id"] != float64(55) {
		t.Errorf("focus_node_id = %v, want 55", out["focus_node_id"])
	}
	if out["focus_id"] != "ita_new_focus" {
		t.Errorf("focus_id = %v, want ita_new_focus", out["focus_id"])
	}
	if out["name"] != "Renamed" || out["cost"] != float64(5) {
		t.Errorf("supplied saved values must be echoed when the upstream omits them: %+v", out)
	}
	if _, present := out["prerequisites"]; present {
		t.Errorf("unchanged prerequisites were not returned upstream and must not be invented: %+v", out)
	}
	if _, present := out["prerequisite_expression"]; present {
		t.Errorf("an unknown saved expression must not be reported as 'none': %+v", out)
	}
}

func TestUpdateFocus_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Focus node not found"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "node_id": 999, "name": "Renamed"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "Focus node not found") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}

func TestConnectFocus_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/focus-trees/21/nodes/55" || r.Method != http.MethodPut {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 55, "focus_id": "ita_new_focus",
			"prerequisites":      []any{[]any{"ita_a", "ita_b"}},
			"mutually_exclusive": []any{"ita_c"},
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
		Name: "connect_focus",
		Arguments: map[string]any{
			"project_id": 7, "focus_tree_id": 21, "node_id": 55,
			"prerequisites":      []any{[]any{"ita_a", "ita_b"}},
			"mutually_exclusive": []any{"ita_c"},
		},
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
	if len(gotBody) != 2 {
		t.Fatalf("upstream body = %+v, want exactly {prerequisites, mutually_exclusive}", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["focus_node_id"] != float64(55) {
		t.Errorf("focus_node_id = %v, want 55", out["focus_node_id"])
	}
	prereqs, _ := out["prerequisites"].([]any)
	if len(prereqs) != 1 {
		t.Fatalf("prerequisites = %v, want one AND-group", out["prerequisites"])
	}
	group, _ := prereqs[0].([]any)
	if len(group) != 2 || group[0] != "ita_a" || group[1] != "ita_b" {
		t.Errorf("prerequisites[0] = %v, want [ita_a ita_b]", group)
	}
	mutex, _ := out["mutually_exclusive"].([]any)
	if len(mutex) != 1 || mutex[0] != "ita_c" {
		t.Errorf("mutually_exclusive = %v, want [ita_c]", out["mutually_exclusive"])
	}
}

func TestConnectFocus_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Focus node not found"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "connect_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "node_id": 999, "prerequisites": []any{}},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "Focus node not found") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}

func TestSetFocusMutualExclusionUpdatesBothSidesAtomically(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/projects/7/focus-trees/21/nodes/55/mutual-exclusion" {
			t.Fatalf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{
				"id": 55, "focus_id": "per_german", "mutually_exclusive": []any{"per_british"},
				"tree_edit_revision": "8",
			},
			map[string]any{
				"id": 56, "focus_id": "per_british", "mutually_exclusive": []any{"per_german"},
				"tree_edit_revision": "8",
			},
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
		Name: "set_focus_mutual_exclusion",
		Arguments: map[string]any{
			"project_id": 7, "focus_tree_id": 21, "node_id": 55, "other_node_id": 56, "enabled": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	if gotBody["other_node_id"] != float64(56) || gotBody["enabled"] != true {
		t.Fatalf("upstream body = %#v", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["tree_edit_revision"] != "8" || out["relation"] != "per_german MUTUALLY EXCLUSIVE WITH per_british" {
		t.Fatalf("result = %#v", out)
	}
	rows := out["focuses"].([]any)
	if len(rows) != 2 {
		t.Fatalf("focuses = %#v", rows)
	}
	left := rows[0].(map[string]any)
	mutex := left["mutually_exclusive"].([]any)
	if len(mutex) != 1 || mutex[0] != "per_british" {
		t.Fatalf("left mutex = %#v", mutex)
	}
}

func TestDeleteFocus_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/focus-trees/21/nodes/55" || r.Method != http.MethodDelete {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "delete_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "node_id": 55},
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
	if out["deleted"] != true {
		t.Errorf("deleted = %v, want true", out["deleted"])
	}
	if out["node_id"] != float64(55) {
		t.Errorf("node_id = %v, want 55", out["node_id"])
	}
	if out["focus_tree_id"] != float64(21) {
		t.Errorf("focus_tree_id = %v, want 21", out["focus_tree_id"])
	}
}

func TestDeleteFocus_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Focus node not found"})
	}))
	defer upstream.Close()

	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "delete_focus",
		Arguments: map[string]any{"project_id": 7, "focus_tree_id": 21, "node_id": 999},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 404")
	}
	if !strings.Contains(contentText(result), "Focus node not found") {
		t.Errorf("error text = %q, want the upstream's own detail message relayed", contentText(result))
	}
}

func TestCreateFocusTreeOmittedIDPreservesAutomaticAllocation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if _, present := body["on_conflict"]; present {
			t.Errorf("automatic allocation changed: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 21, "tree_id": "ger_custom_focus", "country_tag": "GER"})
	}))
	defer upstream.Close()
	srv := newFocusProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	})
	result, err := connectClient(t, srv.URL).CallTool(context.Background(), &CallToolParams{Name: "create_focus_tree", Arguments: map[string]any{"project_id": 7}})
	if err != nil || result.IsError {
		t.Fatalf("automatic tree creation failed: %v", err)
	}
}
