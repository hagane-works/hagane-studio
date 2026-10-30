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

func newFamiliesTestServer(t *testing.T, proxyURL string, register func(*Server, *RESTProxy)) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	proxy := NewRESTProxy(proxyURL)

	getServer := func(*http.Request) *Server {
		server := NewServer()
		register(server, proxy)
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

func registerOpinionModifierTools(s *Server, proxy *RESTProxy) {
	registerCreateOpinionModifier(s, proxy)
	registerUpdateOpinionModifier(s, proxy)
}

func registerModifierDefinitionTools(s *Server, proxy *RESTProxy) {
	registerCreateModifierDefinition(s, proxy)
	registerUpdateModifierDefinition(s, proxy)
}

func registerWargoalTools(s *Server, proxy *RESTProxy) {
	registerCreateWargoal(s, proxy)
	registerUpdateWargoal(s, proxy)
}

func TestWargoalCatalogGuidanceMatchesAcceptedTriggerShape(t *testing.T) {
	srv := newFamiliesTestServer(t, "http://unused.invalid", registerWargoalTools)
	session := connectClient(t, srv.URL)
	catalog, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range catalog.Tools {
		if tool.Name != "create_wargoal" {
			continue
		}
		if !strings.Contains(tool.Description, "without outer braces") || strings.Contains(tool.Description, "full { ... }") {
			t.Fatalf("tool-level guidance contradicts accepted trigger bodies: %s", tool.Description)
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil || !strings.Contains(string(schema), "without outer braces") || strings.Contains(string(schema), "e.g. {\\n") {
			t.Fatalf("field guidance contradicts accepted trigger bodies: %s err=%v", schema, err)
		}
		return
	}
	t.Fatal("create_wargoal missing from live tool catalog")
}

func registerAiStrategyPlanTools(s *Server, proxy *RESTProxy) {
	registerCreateAiStrategyPlan(s, proxy)
	registerUpdateAiStrategyPlan(s, proxy)
}

func registerAiTemplateTools(s *Server, proxy *RESTProxy) {
	registerCreateAiTemplate(s, proxy)
	registerUpdateAiTemplate(s, proxy)
}

func TestCreateOpinionModifier(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotMethod, gotPath = r.Header.Get("Authorization"), r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 55, "modifier_id": "war_ally_support", "country_tag": "GER"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerOpinionModifierTools)
	session := connectClient(t, srv.URL)

	decay := 0.0
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_opinion_modifier",
		Arguments: map[string]any{
			"project_id":  7,
			"modifier_id": "war_ally_support",
			"country_tag": "GER",
			"value":       25,
			"decay":       decay,
			"days":        90,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/7/opinion-modifiers" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/7/opinion-modifiers", gotMethod, gotPath)
	}
	if gotAuth != "" {
		t.Errorf("upstream saw Authorization = %q; the tools forward no credential", gotAuth)
	}

	if gotBody["decay"] != float64(0) {
		t.Errorf("upstream body decay = %v, want explicit 0 to be forwarded", gotBody["decay"])
	}
	if gotBody["days"] != float64(90) {
		t.Errorf("upstream body days = %v, want 90", gotBody["days"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["opinion_modifier_id"] != float64(55) {
		t.Errorf("opinion_modifier_id = %v, want 55", out["opinion_modifier_id"])
	}
}

func TestUpdateOpinionModifier(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 55, "modifier_id": "war_ally_support", "country_tag": "GER"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerOpinionModifierTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_opinion_modifier",
		Arguments: map[string]any{
			"project_id":          7,
			"opinion_modifier_id": 55,
			"value":               40,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/projects/7/opinion-modifiers/55" {
		t.Errorf("upstream request = %s %s, want PUT /api/projects/7/opinion-modifiers/55", gotMethod, gotPath)
	}

	if len(gotBody) != 1 {
		t.Errorf("upstream body = %+v, want exactly {value: 40}", gotBody)
	}
	if gotBody["value"] != float64(40) {
		t.Errorf("upstream body value = %v, want 40", gotBody["value"])
	}
}

func TestCreateOpinionModifier_UpstreamConflictSurfacesAsToolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "Opinion modifier 'war_ally_support' already exists."})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerOpinionModifierTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_opinion_modifier",
		Arguments: map[string]any{"project_id": 7, "modifier_id": "war_ally_support"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true on upstream 409")
	}
	if !strings.Contains(contentText(result), "already exists") {
		t.Errorf("error text = %q, want upstream detail relayed", contentText(result))
	}
}

func TestCreateModifierDefinition(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "modifier_id": "custom_stability_bonus"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerModifierDefinitionTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_modifier_definition",
		Arguments: map[string]any{
			"project_id":  12,
			"modifier_id": "custom_stability_bonus",
			"color_type":  "good",
			"value_type":  "percentage",
			"precision":   0,
			"categories":  []string{"country", "politics"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projects/12/modifier-definitions" {
		t.Errorf("upstream request = %s %s, want POST /api/projects/12/modifier-definitions", gotMethod, gotPath)
	}

	if gotBody["precision"] != float64(0) {
		t.Errorf("upstream body precision = %v, want explicit 0 forwarded", gotBody["precision"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["modifier_definition_id"] != float64(3) {
		t.Errorf("modifier_definition_id = %v, want 3", out["modifier_definition_id"])
	}
}

func TestUpdateModifierDefinition(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "modifier_id": "custom_stability_bonus"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerModifierDefinitionTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_modifier_definition",
		Arguments: map[string]any{
			"project_id":             12,
			"modifier_definition_id": 3,
			"postfix":                "days",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/12/modifier-definitions/3" {
		t.Errorf("upstream path = %q, want /api/projects/12/modifier-definitions/3", gotPath)
	}
	if len(gotBody) != 1 || gotBody["postfix"] != "days" {
		t.Errorf("upstream body = %+v, want exactly {postfix: days}", gotBody)
	}
}

func TestCreateWargoal(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "wargoal_key": "take_state_wg"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerWargoalTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_wargoal",
		Arguments: map[string]any{
			"project_id":  4,
			"wargoal_key": "take_state_wg",
			"allowed":     "always = yes",
			"safe_scalars": map[string]any{
				"take_states_cost": "10",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/4/wargoals" {
		t.Errorf("upstream path = %q, want /api/projects/4/wargoals", gotPath)
	}
	if gotBody["allowed"] != "always = yes" {
		t.Errorf("upstream body allowed = %v", gotBody["allowed"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["wargoal_id"] != float64(9) {
		t.Errorf("wargoal_id = %v, want 9", out["wargoal_id"])
	}
}

func TestUpdateWargoal(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "wargoal_key": "take_state_wg_v2"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerWargoalTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_wargoal",
		Arguments: map[string]any{
			"project_id":  4,
			"wargoal_id":  9,
			"wargoal_key": "take_state_wg_v2",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/4/wargoals/9" {
		t.Errorf("upstream path = %q, want /api/projects/4/wargoals/9", gotPath)
	}
	if len(gotBody) != 1 || gotBody["wargoal_key"] != "take_state_wg_v2" {
		t.Errorf("upstream body = %+v, want exactly {wargoal_key: take_state_wg_v2}", gotBody)
	}
	out := result.StructuredContent.(map[string]any)
	if out["wargoal_key"] != "take_state_wg_v2" {
		t.Errorf("wargoal_key = %v, want take_state_wg_v2", out["wargoal_key"])
	}
}

func TestCreateAiStrategyPlan(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 21, "plan_id": "rearmament_focus"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerAiStrategyPlanTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_ai_strategy_plan",
		Arguments: map[string]any{
			"project_id": 8,
			"plan_id":    "rearmament_focus",
			"name":       "Rearmament",
			"weight":     []string{"{\n\tfactor = 1\n}"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/8/ai-strategy-plans" {
		t.Errorf("upstream path = %q, want /api/projects/8/ai-strategy-plans", gotPath)
	}
	weight, _ := gotBody["weight"].([]any)
	if len(weight) != 1 || weight[0] != "{\n\tfactor = 1\n}" {
		t.Errorf("upstream body weight = %v", gotBody["weight"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["ai_strategy_plan_id"] != float64(21) {
		t.Errorf("ai_strategy_plan_id = %v, want 21", out["ai_strategy_plan_id"])
	}
}

func TestUpdateAiStrategyPlan(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 21, "plan_id": "rearmament_focus"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerAiStrategyPlanTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ai_strategy_plan",
		Arguments: map[string]any{
			"project_id":          8,
			"ai_strategy_plan_id": 21,
			"desc":                "Prioritizes military buildup focuses.",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/8/ai-strategy-plans/21" {
		t.Errorf("upstream path = %q, want /api/projects/8/ai-strategy-plans/21", gotPath)
	}
	if len(gotBody) != 1 || gotBody["desc"] != "Prioritizes military buildup focuses." {
		t.Errorf("upstream body = %+v, want exactly {desc: ...}", gotBody)
	}
}

func TestCreateAiTemplate(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 14, "template_id": "infantry_main"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerAiTemplateTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_ai_template",
		Arguments: map[string]any{
			"project_id":  5,
			"template_id": "infantry_main",
			"role":        "infantry",
			"target_templates": []map[string]any{
				{"id": "infantry_default", "body": "{\n\tregiments = {\n\t\tinfantry = 6\n\t}\n}"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/5/ai-templates" {
		t.Errorf("upstream path = %q, want /api/projects/5/ai-templates", gotPath)
	}
	targets, _ := gotBody["target_templates"].([]any)
	if len(targets) != 1 {
		t.Fatalf("upstream body target_templates = %v, want 1 entry", gotBody["target_templates"])
	}
	first, _ := targets[0].(map[string]any)
	if first["id"] != "infantry_default" {
		t.Errorf("target_templates[0].id = %v, want infantry_default", first["id"])
	}
	out := result.StructuredContent.(map[string]any)
	if out["ai_template_id"] != float64(14) {
		t.Errorf("ai_template_id = %v, want 14", out["ai_template_id"])
	}
}

func TestUpdateAiTemplate(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 14, "template_id": "infantry_main"})
	}))
	defer upstream.Close()

	srv := newFamiliesTestServer(t, upstream.URL, registerAiTemplateTools)
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "update_ai_template",
		Arguments: map[string]any{
			"project_id":     5,
			"ai_template_id": 14,
			"role":           "armor",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotPath != "/api/projects/5/ai-templates/14" {
		t.Errorf("upstream path = %q, want /api/projects/5/ai-templates/14", gotPath)
	}
	if len(gotBody) != 1 || gotBody["role"] != "armor" {
		t.Errorf("upstream body = %+v, want exactly {role: armor}", gotBody)
	}
}
