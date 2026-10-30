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

func writeCompactValidationFixture(w http.ResponseWriter, payload map[string]any) {
	issues, _ := payload["issues"].([]any)
	count := len(issues)
	if rows, ok := payload["issues"].([]map[string]any); ok {
		count = len(rows)
	}
	payload["view"], payload["is_snapshot"] = "compact", false
	payload["revision"], payload["done"] = strings.Repeat("a", 64), true
	payload["offset"], payload["limit"], payload["total"], payload["returned"], payload["next_offset"] = 0, 20, count, count, nil
	_ = json.NewEncoder(w).Encode(payload)
}

func newValidateTestServer(t *testing.T, buildDeps func(host string) Deps) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	host := listener.Addr().String()
	deps := buildDeps(host)

	getServer := func(*http.Request) *Server {
		server := NewServer()
		registerValidateProject(server, deps.Proxy)
		return server
	}
	streamable := toolHTTPHandler(getServer)

	srv := httptest.NewUnstartedServer(streamable)
	srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestValidateProject_ProxiesIssuesAndSummaryOnly(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/projects/9/export/validate" {
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Query().Get("view") != "compact" {
			t.Error("validation must be compact before crossing HTTP, not truncated inside the tool API")
		}
		if r.URL.Query().Get("include_imported") != "" {
			t.Errorf("include_imported = %q, want omitted (defaults to False upstream)", r.URL.Query().Get("include_imported"))
		}
		writeCompactValidationFixture(w, map[string]any{
			"issues":  []map[string]any{{"code": "character_missing_name", "severity": "blocker", "level": "error"}},
			"summary": map[string]any{"characters": 3, "focus_nodes": 12},

			"asset_audit": map[string]any{"missing": []any{"foo.png"}},
			"artifact":    map[string]any{"status": "deferred"},
		})
	}))
	defer upstream.Close()

	srv := newValidateTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "validate_project", Arguments: map[string]any{"project_id": 9},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if _, ok := out["asset_audit"]; ok {
		t.Errorf("asset_audit leaked into validate_project output: %v", out)
	}
	if _, ok := out["artifact"]; ok {
		t.Errorf("artifact leaked into validate_project output: %v", out)
	}
	issues, ok := out["issues"].([]any)
	if !ok || len(issues) != 1 || issues[0].(map[string]any)["code"] != "character_missing_name" {
		t.Errorf("issues = %v", out["issues"])
	}
	summary, ok := out["summary"].(map[string]any)
	if !ok || summary["focus_nodes"] != float64(12) {
		t.Errorf("summary = %v, want focus_nodes: 12", out["summary"])
	}
}

func TestValidateProjectRejectsMissingCompactContract(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer upstream.Close()

	srv := newValidateTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "validate_project", Arguments: map[string]any{"project_id": 4},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError || !strings.Contains(contentText(result), "no clean result was established") {
		t.Fatalf("an unknown upstream payload became a clean empty validation: %#v", result)
	}
}

func TestValidateProjectDeclaresItsOwnCoverageGap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCompactValidationFixture(w, map[string]any{"issues": []any{}, "summary": map[string]any{}})
	}))
	defer upstream.Close()

	srv := newValidateTestServer(t, func(host string) Deps {
		return Deps{
			Proxy: NewRESTProxy(upstream.URL),
		}
	})
	session := connectClient(t, srv.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "validate_project", Arguments: map[string]any{"project_id": 9},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	out := result.StructuredContent.(map[string]any)
	coverage, ok := out["coverage"].(map[string]any)
	if !ok {
		t.Fatalf("a zero-issue result must still state what it does NOT cover: %v", out)
	}
	if excludes, _ := coverage["excludes"].(string); excludes == "" {
		t.Errorf("coverage.excludes must name the other warning stream: %v", coverage)
	}
	if from, _ := coverage["excluded_warnings_available_from"].(string); from != "export_project" {
		t.Errorf("coverage must point at the tool that has the rest, got %v", coverage)
	}
}
