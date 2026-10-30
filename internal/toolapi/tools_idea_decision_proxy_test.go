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

func newIdeaDecisionProxyTestServer(t *testing.T, buildDeps func(host string) Deps, register func(*Server, *RESTProxy)) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	host := listener.Addr().String()
	deps := buildDeps(host)

	getServer := func(*http.Request) *Server {
		server := NewServer()
		register(server, deps.Proxy)
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

func TestCreateDecisionCategory_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/decisions/categories" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "category_id": "political_decisions", "warnings": []string{}})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateDecisionCategory)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_decision_category",
		Arguments: map[string]any{"project_id": 7, "category_id": "political_decisions"},
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
	if out["category_db_id"] != float64(5) {
		t.Errorf("category_db_id = %v, want 5", out["category_db_id"])
	}
	if out["category_id"] != "political_decisions" {
		t.Errorf("category_id = %v", out["category_id"])
	}
}

func TestCreateDecisionCategory_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "category_id 'political_decisions' already exists in this project. Pick a unique id."})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateDecisionCategory)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_decision_category",
		Arguments: map[string]any{"project_id": 7, "category_id": "political_decisions"},
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

func TestCreateDecision_ProxiesToUpstreamBackend(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/decisions" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 88, "decision_id": "build_the_thing", "category_id": 5, "warnings": []string{},
		})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateDecision)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_decision",
		Arguments: map[string]any{
			"project_id": 7, "category_id": 5, "decision_id": "build_the_thing", "cost": 50,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["category_id"] != float64(5) {
		t.Errorf("upstream saw category_id = %v, want 5", gotBody["category_id"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["decision_db_id"] != float64(88) {
		t.Errorf("decision_db_id = %v, want 88", out["decision_db_id"])
	}
	if out["category_id"] != float64(5) {
		t.Errorf("category_id = %v, want 5", out["category_id"])
	}
}

func TestCreateDecision_UpstreamErrorSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Category not found in this project"})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateDecision)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_decision",
		Arguments: map[string]any{"project_id": 7, "category_id": 999, "decision_id": "x"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true when the upstream returns 400")
	}
	if !strings.Contains(contentText(result), "Category not found in this project") {
		t.Errorf("error text = %q", contentText(result))
	}
}

func TestUpdateDecision_ProxiesOnlySuppliedFields(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/decisions/88" || r.Method != http.MethodPut {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 88, "decision_id": "build_the_thing"})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerUpdateDecision)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "update_decision",
		Arguments: map[string]any{"project_id": 7, "decision_db_id": 88, "cost": 75},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["cost"] != float64(75) {
		t.Errorf("upstream saw cost = %v, want 75", gotBody["cost"])
	}
	if _, present := gotBody["name_en"]; present {
		t.Errorf("upstream body contains unsupplied field name_en = %v, want it omitted entirely", gotBody["name_en"])
	}
}

func TestUpdateDecisionAllowedUsesBackendPatchContract(t *testing.T) {
	for _, test := range []struct {
		name    string
		allowed *string
	}{
		{name: "historical country trigger", allowed: stringPointer("original_tag = GER")},
		{name: "explicit empty clears trigger", allowed: stringPointer("")},
		{name: "omitted preserves trigger"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body map[string]any
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPut || r.URL.Path != "/api/projects/7/decisions/9406" {
					t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 9406, "decision_id": "GER_test"})
			}))
			defer upstream.Close()
			srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
				return Deps{Proxy: NewRESTProxy(upstream.URL)}
			}, registerUpdateDecision)
			session := connectClient(t, srv.URL)
			args := map[string]any{"project_id": 7, "decision_db_id": 9406, "available": "always = yes"}
			if test.allowed != nil {
				args["allowed"] = *test.allowed
			}
			result := callTool(t, session, "update_decision", args)
			if result.IsError || calls != 1 {
				t.Fatalf("first invocation must reach the backend once: calls=%d result=%s", calls, contentText(result))
			}
			got, supplied := body["allowed"]
			if supplied != (test.allowed != nil) || test.allowed != nil && got != *test.allowed {
				t.Fatalf("allowed patch lost omission/empty semantics: %#v", body)
			}
			if body["available"] != "always = yes" {
				t.Fatalf("existing supplied fields were not preserved: %#v", body)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }

func TestCreateIdea_ProxiesToUpstreamBackendWithDefaults(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/9/ideas" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 3, "idea_id": "new_idea", "name": "钢铁意志", "picture": "TST_iron_will",
			"name_en": "Iron Will", "name_zh": "钢铁意志", "desc_zh": "更强的军队",
			"display_name_locs":    map[string]string{"zh": "钢铁意志"},
			"description_locs":     map[string]string{"zh": "更强的军队"},
			"auto_attach_on_start": false, "warnings": []string{},
		})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateIdea)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_idea",
		Arguments: map[string]any{
			"project_id": 9, "name": "钢铁意志", "name_en": "Iron Will", "name_zh": "钢铁意志",
			"desc_zh": "更强的军队", "picture": "TST_iron_will", "auto_attach_on_start": false,
			"display_name_locs": map[string]any{"zh": "钢铁意志"},
			"description_locs":  map[string]any{"zh": "更强的军队"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}

	if gotBody["name"] != "钢铁意志" {
		t.Errorf("upstream saw name = %v", gotBody["name"])
	}
	if gotBody["idea_type"] != "country" {
		t.Errorf("upstream saw idea_type = %v, want default 'country'", gotBody["idea_type"])
	}
	if gotBody["removal_cost"] != float64(-1) {
		t.Errorf("upstream saw removal_cost = %v, want default -1", gotBody["removal_cost"])
	}
	if gotBody["name_zh"] != "钢铁意志" || gotBody["desc_zh"] != "更强的军队" || gotBody["auto_attach_on_start"] != false {
		t.Errorf("upstream national-spirit localisation/lifecycle = %#v", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["idea_db_id"] != float64(3) {
		t.Errorf("idea_db_id = %v, want 3", out["idea_db_id"])
	}
	if out["picture"] != "TST_iron_will" || out["name_zh"] != "钢铁意志" || out["auto_attach_on_start"] != false {
		t.Errorf("saved national-spirit fields = %#v", out)
	}
}

func TestCreateIdea_RemovalCostZeroIsNotTreatedAsUnset(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 4, "idea_id": "free_idea"})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerCreateIdea)
	session := connectClient(t, srv.URL)

	_, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_idea",
		Arguments: map[string]any{"project_id": 9, "removal_cost": 0},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if gotBody["removal_cost"] != float64(0) {
		t.Errorf("upstream saw removal_cost = %v, want explicit 0 (not the -1 default)", gotBody["removal_cost"])
	}
}

func TestUpdateIdea_ProxiesOnlySuppliedFields(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/9/ideas/3" || r.Method != http.MethodPut {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 3, "idea_id": "renamed_idea", "name": "Renamed", "picture": "generic_build_infrastructure",
			"name_zh": "新标签", "desc_zh": "新说明", "display_name_locs": map[string]string{"zh": "新标签"},
			"description_locs": map[string]string{"zh": "新说明"}, "auto_attach_on_start": false,
		})
	}))
	defer upstream.Close()

	srv := newIdeaDecisionProxyTestServer(t, func(host string) Deps {
		return Deps{Proxy: NewRESTProxy(upstream.URL)}
	}, registerUpdateIdea)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_idea",
		Arguments: map[string]any{
			"project_id": 9, "idea_db_id": 3, "picture": "generic_build_infrastructure",
			"name_zh": "新标签", "desc_zh": "新说明", "auto_attach_on_start": false,
			"display_name_locs": map[string]any{"zh": "新标签"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["picture"] != "generic_build_infrastructure" {
		t.Errorf("upstream saw picture = %v", gotBody["picture"])
	}
	if gotBody["name_zh"] != "新标签" || gotBody["desc_zh"] != "新说明" || gotBody["auto_attach_on_start"] != false {
		t.Errorf("upstream national-spirit patch = %#v", gotBody)
	}
	if _, present := gotBody["modifier"]; present {
		t.Errorf("upstream body contains unsupplied field modifier, want it omitted entirely")
	}
	out := result.StructuredContent.(map[string]any)
	if out["idea_id"] != "renamed_idea" {
		t.Errorf("idea_id = %v", out["idea_id"])
	}
	if out["name_zh"] != "新标签" || out["picture"] != "generic_build_infrastructure" || out["auto_attach_on_start"] != false {
		t.Errorf("saved national-spirit patch = %#v", out)
	}
}
