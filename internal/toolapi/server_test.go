package toolapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func newAuthedTestServer(t *testing.T, buildDeps func(host string) Deps) *httptest.Server {
	t.Helper()
	deps := buildDeps("")
	proxy := deps.Proxy
	if proxy == nil {
		proxy = NewRESTProxy("http://127.0.0.1:1")
	}
	settled := *proxy
	settled.Client = &http.Client{Timeout: proxy.Client.Timeout, Transport: settledProjects{next: proxy.Client.Transport}}
	tools := New(&settled)
	srv := httptest.NewServer(toolHTTPHandler(func(*http.Request) *Server { return tools }))
	t.Cleanup(srv.Close)
	return srv
}

func TestListProjects(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/projects" {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]any{
			{"id": 7, "name": "German Spring", "country_tag": "GER"},
			{"id": 9, "name": "Romania Rework", "country_tag": "ROM"},
		}, "total": 2})
	}))
	defer upstream.Close()
	srv := newAuthedTestServer(t, func(host string) Deps { return Deps{Proxy: NewRESTProxy(upstream.URL)} })

	result, err := connectClient(t, srv.URL).CallTool(context.Background(), &CallToolParams{Name: "list_projects"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(result))
	}
	out := result.StructuredContent.(map[string]any)
	if out["total"] != float64(2) {
		t.Errorf("total = %v, want 2", out["total"])
	}
	projects, _ := out["projects"].([]any)
	if len(projects) != 2 {
		t.Fatalf("projects = %v", projects)
	}
	first, _ := projects[0].(map[string]any)
	if first["id"] != float64(7) || first["name"] != "German Spring" || len(first) != 2 {
		t.Errorf("first = %v, want only id and name", first)
	}
}

func TestNew_RegistersTheDesktopToolSet(t *testing.T) {
	tools, err := New(NewRESTProxy("http://127.0.0.1:1")).ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		schema, _ := tool.InputSchema.(map[string]any)
		if schema["type"] != "object" {
			t.Errorf("%s: input schema %v", tool.Name, schema)
		}
	}
	if strings.Join(names, ",") != strings.Join(desktopToolNames, ",") {
		t.Errorf("tools differ from desktopToolNames:\n got %v\nwant %v", names, desktopToolNames)
	}
}

func contentText(result *CallToolResult) string {
	var b strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

type settledProjects struct{ next http.RoundTripper }

var settledProjectPath = regexp.MustCompile(`^/api/projects/(\d+)(/bookmarks)?$`)

func (s settledProjects) RoundTrip(req *http.Request) (*http.Response, error) {
	match := settledProjectPath.FindStringSubmatch(req.URL.Path)
	tool, _ := req.Context().Value(harnessToolKey{}).(string)
	if match == nil || req.Method != http.MethodGet || !usesAutomaticAuthoringDefaults(tool) {
		next := s.next
		if next == nil {
			next = http.DefaultTransport
		}
		return next.RoundTrip(req)
	}
	var body any = []any{}
	if match[2] == "" {
		id, _ := strconv.ParseInt(match[1], 10, 64)
		body = map[string]any{
			"id": id, "name": "Test project", "start_date": nil, "end_date": "1949.1.1", "dependencies": []any{},
			"export_languages": []string{"en", "zh", "ru", "fr", "de", "es", "ja", "ko", "pl", "pt"},
		}
	}
	encoded, _ := json.Marshal(body)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(bytes.NewReader(encoded)), Request: req}, nil
}
