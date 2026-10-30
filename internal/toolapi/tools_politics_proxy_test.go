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

func newPoliticsProxyTestServer(t *testing.T, proxy *RESTProxy) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	deps := Deps{Proxy: proxy}
	getServer := func(*http.Request) *Server {
		server := NewServer()
		registerPoliticsProxyTools(server, deps.Proxy)
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

func TestPoliticsProxyToolsUseCanonicalMethodsPathsAndBodies(t *testing.T) {
	type observed struct {
		method string
		path   string
		body   map[string]any
	}
	requests := make([]observed, 0, 6)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item := observed{method: r.Method, path: r.URL.Path}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&item.body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		}
		requests = append(requests, item)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "token": "preview-1"})
	}))
	defer upstream.Close()

	srv := newPoliticsProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)
	calls := []struct {
		name string
		args map[string]any
	}{
		{"create_ideology_root", map[string]any{
			"project_id": 7, "script_id": "civic_coalition", "body": "types = {} color = { 1 2 3 }",
			"name": map[string]any{"en": "Civic Coalition"}, "desc": map[string]any{"en": "A civic ROOT."},
		}},
		{"create_ideology_subtype", map[string]any{
			"project_id": 7, "script_id": "civic_reformism", "root_id": "civic_coalition",
			"name": map[string]any{"en": "Civic Reformism"}, "desc": map[string]any{"en": "A subtype."}, "random_disabled": true,
		}},
		{"update_ideology", map[string]any{
			"project_id": 7, "kind": "subtype", "script_id": "civic_reformism",
			"patch": map[string]any{"name": map[string]any{"en": "Renewed Civic Reformism"}, "random_disabled": false},
		}},
		{"get_politics_references", map[string]any{"project_id": 7}},
		{"preview_delete_ideology", map[string]any{"project_id": 7, "kind": "subtype", "script_id": "civic_coalition"}},
		{"apply_delete_ideology", map[string]any{"project_id": 7, "preview_token": "preview-1"}},
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

	want := []observed{
		{method: http.MethodPost, path: "/api/projects/7/politics/roots", body: map[string]any{
			"script_id": "civic_coalition", "body": "types = {} color = { 1 2 3 }",
			"name": map[string]any{"en": "Civic Coalition"}, "desc": map[string]any{"en": "A civic ROOT."},
		}},
		{method: http.MethodPost, path: "/api/projects/7/politics/subtypes", body: map[string]any{
			"script_id": "civic_reformism", "root_id": "civic_coalition",
			"name": map[string]any{"en": "Civic Reformism"}, "desc": map[string]any{"en": "A subtype."}, "random_disabled": true,
		}},
		{method: http.MethodPatch, path: "/api/projects/7/politics/definitions/subtype/civic_reformism", body: map[string]any{
			"name": map[string]any{"en": "Renewed Civic Reformism"}, "random_disabled": false,
		}},
		{method: http.MethodGet, path: "/api/projects/7/politics/references"},
		{method: http.MethodPost, path: "/api/projects/7/politics/delete-preview", body: map[string]any{"kind": "subtype", "script_id": "civic_coalition"}},
		{method: http.MethodPost, path: "/api/projects/7/politics/delete-apply", body: map[string]any{"preview_token": "preview-1"}},
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

func TestDiscardIdeologyConflictRequiresFreshMatchingUsageCount(t *testing.T) {
	tests := []struct {
		name            string
		scriptID        string
		acknowledged    int
		wantErrorText   string
		wantDeleteCalls int
	}{
		{name: "matching count deletes", scriptID: "broken_subtype", acknowledged: 3, wantDeleteCalls: 1},
		{name: "changed count refuses", scriptID: "broken_subtype", acknowledged: 2, wantErrorText: "usage_count 3", wantDeleteCalls: 0},
		{name: "missing conflict refuses", scriptID: "not_a_conflict", acknowledged: 3, wantErrorText: "get_politics_references", wantDeleteCalls: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var requests []struct{ method, path string }
			deleteCalls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, struct{ method, path string }{r.Method, r.URL.Path})
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/projects/7/politics/references":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"project_id": "7",
						"conflicts": []any{map[string]any{
							"script_id": "broken_subtype", "kind": "subtype", "root_id": "missing_root",
							"reason": "unknown_root", "detail": "root is missing", "usage_count": 3,
						}},
					})
				case r.Method == http.MethodDelete && r.URL.Path == "/api/projects/7/politics/conflicts/broken_subtype":
					deleteCalls++
					_ = json.NewEncoder(w).Encode(map[string]any{
						"script_id": "broken_subtype", "reason": "unknown_root", "usage_count": 3,
					})
				default:
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]any{"detail": "unexpected request"})
				}
			}))
			defer upstream.Close()

			srv := newPoliticsProxyTestServer(t, NewRESTProxy(upstream.URL))
			session := connectClient(t, srv.URL)
			result, err := session.CallTool(context.Background(), &CallToolParams{
				Name: "discard_ideology_conflict",
				Arguments: map[string]any{
					"project_id": 7, "script_id": tc.scriptID,
					"acknowledged_usage_count": tc.acknowledged,
				},
			})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if tc.wantErrorText == "" {
				if result.IsError {
					t.Fatalf("unexpected tool error: %s", contentText(result))
				}
				fields, ok := result.StructuredContent.(map[string]any)
				if !ok || fields["script_id"] != "broken_subtype" || fields["usage_count"] != float64(3) {
					t.Fatalf("DELETE result was not returned unchanged: %#v", result.StructuredContent)
				}
			} else {
				if !result.IsError {
					t.Fatalf("expected tool error, got %#v", result.StructuredContent)
				}
				if text := contentText(result); !strings.Contains(text, tc.wantErrorText) {
					t.Fatalf("error text = %q, want it to contain %q", text, tc.wantErrorText)
				}
			}
			if deleteCalls != tc.wantDeleteCalls {
				t.Fatalf("DELETE calls = %d, want %d; requests = %#v", deleteCalls, tc.wantDeleteCalls, requests)
			}
			if len(requests) == 0 || requests[0].method != http.MethodGet || requests[0].path != "/api/projects/7/politics/references" {
				t.Fatalf("first request must refresh references, got %#v", requests)
			}
		})
	}
}

func TestDiscardIdeologyConflictSchemaRequiresAcknowledgedUsageCount(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer upstream.Close()
	srv := newPoliticsProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)
	result, err := session.ListTools(context.Background(), &ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range result.Tools {
		if tool.Name != "discard_ideology_conflict" {
			continue
		}
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal schema: %v", err)
		}
		var schema struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("decode schema: %v", err)
		}
		for _, field := range schema.Required {
			if field == "acknowledged_usage_count" {
				return
			}
		}
		t.Fatalf("acknowledged_usage_count is not required: %s", raw)
	}
	t.Fatal("discard_ideology_conflict not registered")
}

func TestPoliticsAuthoringToolsRejectOwnershipAndUnsafeFieldsBeforeProxy(t *testing.T) {
	type authoringCase struct {
		name string
		base map[string]any
	}
	cases := []authoringCase{
		{name: "create_ideology_root", base: map[string]any{
			"project_id": 7, "script_id": "civic_coalition", "body": "types = {} color = { 1 2 3 }", "name": map[string]any{"en": "Civic"},
		}},
		{name: "create_ideology_subtype", base: map[string]any{
			"project_id": 7, "script_id": "civic_reformism", "root_id": "civic_coalition", "name": map[string]any{"en": "Reformism"},
		}},
		{name: "update_ideology", base: map[string]any{
			"project_id": 7, "kind": "subtype", "script_id": "civic_reformism", "patch": map[string]any{"name": map[string]any{"en": "Renewed"}},
		}},
	}
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer upstream.Close()
	srv := newPoliticsProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	for _, tc := range cases {
		for _, field := range []string{"source", "owner", "hash", "force", "cascade", "rewrite", "takeover"} {
			called = false
			args := clonePoliticsProxyArguments(tc.base)
			args[field] = "forbidden"
			result, err := session.CallTool(context.Background(), &CallToolParams{Name: tc.name, Arguments: args})
			if err != nil {
				t.Fatalf("%s field %s CallTool: %v", tc.name, field, err)
			}
			if !result.IsError {
				t.Errorf("%s accepted unsafe field %q", tc.name, field)
			}
			if called {
				t.Errorf("%s forwarded unsafe field %q", tc.name, field)
			}
		}
	}

	for _, field := range []string{"script_id", "root_id", "source", "owner", "hash", "force", "cascade", "rewrite", "takeover"} {
		called = false
		args := clonePoliticsProxyArguments(cases[2].base)
		args["patch"].(map[string]any)[field] = "forbidden"
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: "update_ideology", Arguments: args})
		if err != nil {
			t.Fatalf("update patch field %s CallTool: %v", field, err)
		}
		if !result.IsError {
			t.Errorf("update_ideology patch accepted forbidden field %q", field)
		}
		if called {
			t.Errorf("update_ideology forwarded forbidden patch field %q", field)
		}
	}
}

func clonePoliticsProxyArguments(source map[string]any) map[string]any {
	raw, _ := json.Marshal(source)
	var clone map[string]any
	_ = json.Unmarshal(raw, &clone)
	return clone
}

func TestApplyDeleteIdeologyRejectsEveryUnsafeEscapeHatchBeforeProxy(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer upstream.Close()
	srv := newPoliticsProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, srv.URL)

	for _, field := range []string{"force", "cascade", "rewrite", "takeover"} {
		called = false
		args := map[string]any{"project_id": 7, "preview_token": "preview-1", field: true}
		result, err := session.CallTool(context.Background(), &CallToolParams{Name: "apply_delete_ideology", Arguments: args})
		if err != nil {
			t.Fatalf("%s CallTool: %v", field, err)
		}
		if !result.IsError {
			t.Errorf("unsafe field %q was accepted", field)
		}
		if called {
			t.Errorf("unsafe field %q reached the project service", field)
		}
	}
}
