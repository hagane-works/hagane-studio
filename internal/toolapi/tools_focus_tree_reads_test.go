package toolapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFocusTreeTestServer(t *testing.T, proxyURL string) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	getServer := func(*http.Request) *Server {
		server := NewServer()
		proxy := NewRESTProxy(proxyURL)
		registerGetFocusTree(server, proxy)
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

func fakeFocusTreeUpstream(t *testing.T, projectPath string, trees []map[string]any, nodesByTreeID map[int64][]map[string]any) (*httptest.Server, *string, *[]string) {
	t.Helper()
	var gotQuery string
	var hitPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPaths = append(hitPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == projectPath {
			gotQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode(trees)
			return
		}
		_ = nodesByTreeID
		t.Errorf("unexpected upstream path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv, &gotQuery, &hitPaths
}

func twoTreeFixture() ([]map[string]any, map[int64][]map[string]any) {
	nodes := map[int64][]map[string]any{
		3: {{"id": 101, "focus_id": "GER_anschluss", "name": "Anschluss", "x": 5, "y": 0,
			"icon": "GFX_focus_anschluss", "cost": 7.0, "relative_position_id": "GER_root",
			"prerequisites": []any{[]any{"GER_a", "GER_b"}}, "mutually_exclusive": []any{"GER_c"},
			"completion_reward": "add_political_power = 50", "available": "has_war = no",
			"bypass": "has_completed_focus = GER_other", "cancel": "has_war = yes",
			"select_effect": "add_stability = 0.01", "ai_will_do": "factor = 2"}},
		4: {{"id": 201, "focus_id": "ITA_mare_nostrum", "name": "Mare Nostrum", "x": 2, "y": 1,
			"prerequisites": []any{}, "mutually_exclusive": []any{}}},
	}
	trees := []map[string]any{
		{"id": 3, "tree_id": "ger_focus", "country_tag": "GER", "nodes": nodes[3]},
		{"id": 4, "tree_id": "ita_focus", "country_tag": "ITA", "nodes": nodes[4]},
	}
	return trees, nodes
}

func TestGetFocusTree_NoFilter_MergesTreesAndNodes(t *testing.T) {
	trees, nodes := twoTreeFixture()
	upstream, gotQuery, hitPaths := fakeFocusTreeUpstream(t, "/api/projects/7/focus-trees", trees, nodes)

	srv := newFocusTreeTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_focus_tree", Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	if *gotQuery != "" {
		t.Errorf("upstream query = %q, want empty (no country_tag filter)", *gotQuery)
	}
	if len(*hitPaths) != 1 || (*hitPaths)[0] != "/api/projects/7/focus-trees" {
		t.Errorf("upstream paths = %v, want exactly one list request", *hitPaths)
	}

	out := result.StructuredContent.(map[string]any)
	if out["project_id"] != float64(7) {
		t.Errorf("project_id = %v, want 7", out["project_id"])
	}
	got := out["focus_trees"].([]any)
	if len(got) != 2 {
		t.Fatalf("len(focus_trees) = %d, want 2", len(got))
	}
	first := got[0].(map[string]any)
	if first["focus_tree_id"] != float64(3) || first["tree_id"] != "ger_focus" || first["country_tag"] != "GER" {
		t.Errorf("first tree = %+v, want focus_tree_id 3 / ger_focus / GER", first)
	}
	firstNodes := first["nodes"].([]any)
	if len(firstNodes) != 1 {
		t.Fatalf("len(first tree nodes) = %d, want 1", len(firstNodes))
	}
	node := firstNodes[0].(map[string]any)
	if node["node_id"] != float64(101) {
		t.Errorf("node_id = %v, want 101 (renamed from REST's 'id')", node["node_id"])
	}
	if node["focus_id"] != "GER_anschluss" || node["name"] != "Anschluss" {
		t.Errorf("node = %+v, want GER_anschluss/Anschluss", node)
	}
	if node["x"] != float64(5) || node["y"] != float64(0) {
		t.Errorf("node x/y = %v/%v, want 5/0", node["x"], node["y"])
	}
	if node["icon"] != "GFX_focus_anschluss" || node["cost"] != float64(7) {
		t.Errorf("node icon/cost = %v/%v", node["icon"], node["cost"])
	}
	if node["completion_reward"] != "add_political_power = 50" || node["available"] != "has_war = no" {
		t.Errorf("node effects were not available for final verification: %+v", node)
	}
	if node["relative_position_id"] != "GER_root" || node["ai_will_do"] != "factor = 2" {
		t.Errorf("node authoring state is incomplete: %+v", node)
	}
	prereqs := node["prerequisites"].([]any)
	if len(prereqs) != 1 {
		t.Fatalf("prerequisites = %v, want one AND-group", prereqs)
	}
	group := prereqs[0].([]any)
	if len(group) != 2 || group[0] != "GER_a" || group[1] != "GER_b" {
		t.Errorf("prerequisites[0] = %v, want [GER_a GER_b]", group)
	}
	if node["prerequisite_expression"] != "(GER_a OR GER_b)" {
		t.Errorf("prerequisite_expression = %v", node["prerequisite_expression"])
	}
	mutex := node["mutually_exclusive"].([]any)
	if len(mutex) != 1 || mutex[0] != "GER_c" {
		t.Errorf("mutually_exclusive = %v, want [GER_c]", mutex)
	}

	second := got[1].(map[string]any)
	if second["country_tag"] != "ITA" {
		t.Errorf("second tree country_tag = %v, want ITA", second["country_tag"])
	}
}

func TestGetFocusTree_WithCountryTag_ForwardsUppercasedQueryParamAndSkipsUnmatchedNodes(t *testing.T) {
	var gotQuery string
	var hitPaths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPaths = append(hitPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/projects/7/focus-trees":
			gotQuery = r.URL.RawQuery
			all := []map[string]any{
				{"id": 3, "tree_id": "ger_focus", "country_tag": "GER", "nodes": []any{map[string]any{"id": 101, "focus_id": "GER_anschluss", "name": "Anschluss", "x": 5, "y": 0, "prerequisites": []any{}, "mutually_exclusive": []any{}}}},
				{"id": 4, "tree_id": "ita_focus", "country_tag": "ITA", "nodes": []any{}},
			}
			tag := r.URL.Query().Get("country_tag")
			filtered := []map[string]any{}
			for _, tr := range all {
				if tag == "" || tr["country_tag"] == tag {
					filtered = append(filtered, tr)
				}
			}
			_ = json.NewEncoder(w).Encode(filtered)
		default:
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	srv := newFocusTreeTestServer(t, upstream.URL)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "get_focus_tree", Arguments: map[string]any{"project_id": 7, "country_tag": "ger"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	if gotQuery != "country_tag=GER" {
		t.Errorf("upstream query = %q, want country_tag=GER (uppercased, case-insensitive parity "+
			"with _focus_tree_payload's country_tag.upper() filter)", gotQuery)
	}
	if len(hitPaths) != 1 || hitPaths[0] != "/api/projects/7/focus-trees" {
		t.Errorf("upstream paths = %v, want exactly the focus-tree list", hitPaths)
	}

	out := result.StructuredContent.(map[string]any)
	got := out["focus_trees"].([]any)
	if len(got) != 1 {
		t.Fatalf("focus_trees = %v, want exactly 1 (GER only)", got)
	}
	tree := got[0].(map[string]any)
	if tree["country_tag"] != "GER" || tree["tree_id"] != "ger_focus" {
		t.Errorf("tree = %v, want GER/ger_focus", tree)
	}
}
