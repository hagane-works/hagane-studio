package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateDivisionTemplate_ProxiesToUpstreamBackend(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/projects/7/division-templates" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 31, "project_id": 7, "oob_file": "GER_1936",
			"template_name": "Infantry Div 1", "country_tag": "GER",
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateDivisionTemplate(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "create_division_template",
		Arguments: map[string]any{
			"project_id": 7, "oob_file": "GER_1936", "template_name": "Infantry Div",
			"regiments": []map[string]any{{"token": "infantry", "x": 0, "y": 0}},
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
	if gotBody["template_name"] != "Infantry Div" {
		t.Errorf("POST body template_name = %v", gotBody["template_name"])
	}
	regiments, _ := gotBody["regiments"].([]any)
	if len(regiments) != 1 {
		t.Errorf("POST body regiments = %v, want the 1 sent battalion", gotBody["regiments"])
	}
	if support, ok := gotBody["support"].([]any); !ok || len(support) != 0 {

		t.Errorf("POST body support = %v (%T), want an empty JSON array", gotBody["support"], gotBody["support"])
	}
	if _, present := gotBody["is_locked"]; present && gotBody["is_locked"] != nil {
		t.Errorf("POST body is_locked = %v, want absent/null when not supplied", gotBody["is_locked"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["template_id"] != float64(31) {
		t.Errorf("template_id = %v, want 31", out["template_id"])
	}
	if out["template_name"] != "Infantry Div 1" {
		t.Errorf("template_name = %v, want the auto-suffixed name READ BACK from the response", out["template_name"])
	}
	if out["oob_file"] != "GER_1936" || out["country_tag"] != "GER" {
		t.Errorf("oob_file/country_tag = %v/%v", out["oob_file"], out["country_tag"])
	}
}

func TestCreateDivisionTemplate_DefaultsMatchPythonSignature(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "oob_file": "untitled", "template_name": "New Division", "country_tag": "",
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerCreateDivisionTemplate(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_division_template",
		Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["template_name"] != "New Division" {
		t.Errorf("omitted template_name reached upstream as %v, want the Python signature default 'New Division'", gotBody["template_name"])
	}
	if gotBody["ordering"] != float64(0) {
		t.Errorf("ordering = %v, want 0", gotBody["ordering"])
	}
}

func TestAddDivisionDeployment_ProxiesAndRelaysWarnings(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/7/division-deployments" || r.Method != http.MethodPost {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 55, "division_template": "Infantry Div 1", "location": 6583, "count": 3,
			"oob_file": "GER_1936", "country_tag": "GER",
			"warnings": []string{"actionable:unowned_province|province=6583|tag=GER"},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddDivisionDeployment(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "add_division_deployment",
		Arguments: map[string]any{
			"project_id": 7, "oob_file": "GER_1936",
			"division_template": "Infantry Div 1", "location": 6583, "count": 3,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["division_template"] != "Infantry Div 1" || gotBody["location"] != float64(6583) {
		t.Errorf("POST body division_template/location = %v/%v", gotBody["division_template"], gotBody["location"])
	}
	if gotBody["count"] != float64(3) {
		t.Errorf("POST body count = %v, want 3", gotBody["count"])
	}
	if raw, ok := gotBody["raw_extras"].(map[string]any); !ok || len(raw) != 0 {

		t.Errorf("POST body raw_extras = %v (%T), want an empty JSON object", gotBody["raw_extras"], gotBody["raw_extras"])
	}

	out := result.StructuredContent.(map[string]any)
	if out["deployment_id"] != float64(55) {
		t.Errorf("deployment_id = %v, want 55", out["deployment_id"])
	}
	if out["location"] != float64(6583) || out["count"] != float64(3) {
		t.Errorf("location/count = %v/%v", out["location"], out["count"])
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "actionable:unowned_province|province=6583|tag=GER" {
		t.Errorf("warnings = %v, want the two-tier signal relayed verbatim", out["warnings"])
	}
}

func TestAddDivisionDeployment_OmittedCountDefaultsTo1AndLocationIsNull(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readJSONBody(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 56, "division_template": "", "location": nil, "count": 1,
			"oob_file": "", "country_tag": "", "warnings": []string{"actionable:no_location"},
		})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddDivisionDeployment(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_division_deployment",
		Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotBody["count"] != float64(1) {
		t.Errorf("omitted count reached upstream as %v, want the Python signature default 1", gotBody["count"])
	}
	if loc, present := gotBody["location"]; !present || loc != nil {
		t.Errorf("POST body location = %v (present=%v), want an explicit null (Python sends location=None)", loc, present)
	}

	out := result.StructuredContent.(map[string]any)
	if out["location"] != nil {
		t.Errorf("out.location = %v, want null relayed (0 would invent a province id)", out["location"])
	}
}

func TestAddDivisionDeployment_UpstreamErrorSurfacesDetail(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "count must be 1..1000"})
	}))
	defer upstream.Close()

	proxy := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerAddDivisionDeployment(s, proxy) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "add_division_deployment",
		Arguments: map[string]any{"project_id": 7, "count": 0},
	})
	if err != nil {
		t.Fatalf("CallTool transport error (want a tool-level error): %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for an upstream 422")
	}
	if text := contentText(result); !strings.Contains(text, "count must be 1..1000") {
		t.Errorf("error text = %q, want the upstream detail surfaced", text)
	}
}

func TestDivisionWriteTools_RegisteredThroughNewHandler(t *testing.T) {
	srv := newAuthedTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy("http://unused.invalid"),
		}
	})
	session := connectClient(t, srv.URL)

	listed := map[string]bool{}
	result, err := session.ListTools(context.Background(), &ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range result.Tools {
		listed[tool.Name] = true
	}
	for _, want := range []string{"create_division_template", "add_division_deployment"} {
		if !listed[want] {
			t.Errorf("tool %q not registered through the real NewHandler — server.go seam block A not landed?", want)
		}
	}
}
