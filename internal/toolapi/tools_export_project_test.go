package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fastExportPolling(t *testing.T, budget time.Duration) {
	t.Helper()
	prevInterval, prevBudget := exportJobPollInterval, exportJobPollBudget
	exportJobPollInterval, exportJobPollBudget = time.Millisecond, budget
	t.Cleanup(func() { exportJobPollInterval, exportJobPollBudget = prevInterval, prevBudget })
}

func TestExportProject_CreatesJobPollsAndSummarizes(t *testing.T) {
	fastExportPolling(t, 5*time.Second)
	var gotAuth string
	var gotBody map[string]any
	var statusCalls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/go/export/jobs":
			gotAuth = r.Header.Get("Authorization")
			gotBody, _ = readJSONBody(r)
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"job_id": "1754000000-42", "project_id": 7, "status": "queued",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/go/export/jobs/7/1754000000-42/download":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write([]byte("PK-zip-bytes"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/go/export/jobs/7/1754000000-42":
			if statusCalls.Add(1) == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"job_id": "1754000000-42", "status": "running",
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"job_id": "1754000000-42", "status": "complete",
				"filename": "German Spring.zip", "size_bytes": 123456, "sha256": "ab12",
				"files": []string{
					"descriptor.mod",
					"common/ideas/GER_spring.txt",
					"localisation/english/GER_spring_l_english.yml",
					"GER_CROSS_REF_SKIPPED.log",
				},
				"reused_from_job_id": "1753990000-42",
			})
		default:
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	jobs := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerExportProject(s, jobs) })
	session := connectClient(t, srv.URL)

	saveTo := t.TempDir()
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name: "export_project",
		Arguments: map[string]any{
			"project_id":    7,
			"replace_paths": "common/ideas, common/decisions,,",
			"save_to":       saveTo,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result.Content)
	}
	if gotAuth != "" {
		t.Errorf("jobs endpoint saw Authorization = %q; the tools forward no credential", gotAuth)
	}
	replacePaths, _ := gotBody["replace_paths"].([]any)
	if len(replacePaths) != 2 || replacePaths[0] != "common/ideas" || replacePaths[1] != "common/decisions" {
		t.Errorf("POST body replace_paths = %v, want the comma-separated string split/trimmed to 2 entries", gotBody["replace_paths"])
	}
	if gotBody["include_imported"] != false || gotBody["acknowledge_blockers"] != false {
		t.Errorf("POST body include_imported/acknowledge_blockers = %v/%v, want false/false defaults",
			gotBody["include_imported"], gotBody["acknowledge_blockers"])
	}
	if _, present := gotBody["build_identity"]; present {
		t.Errorf("POST body has build_identity — the tool must not opt exports into test-build identity (frontend-only opt-in)")
	}

	out := result.StructuredContent.(map[string]any)
	if out["project_id"] != float64(7) {
		t.Errorf("project_id = %v", out["project_id"])
	}
	if out["file_count"] != float64(3) {
		t.Errorf("file_count = %v, want 3 after filtering the reserved _CROSS_REF_SKIPPED.log entry", out["file_count"])
	}
	files, _ := out["files"].([]any)
	if len(files) != 3 {
		t.Fatalf("len(files) = %d, want 3: %v", len(files), out["files"])
	}
	for _, f := range files {
		if strings.HasSuffix(f.(string), "_CROSS_REF_SKIPPED.log") {
			t.Errorf("files still contains the skipped-log entry: %v", f)
		}
	}
	if files[0] != "common/ideas/GER_spring.txt" {
		t.Errorf("files[0] = %v, want sorted order", files[0])
	}
	if out["size_bytes"] != float64(123456) || out["suggested_filename"] != "German Spring.zip" {
		t.Errorf("size_bytes/suggested_filename = %v/%v", out["size_bytes"], out["suggested_filename"])
	}
	saved, _ := out["saved_to"].(string)
	if saved != filepath.Join(saveTo, "German Spring.zip") {
		t.Errorf("saved_to = %q, want the ZIP under its suggested name in save_to", saved)
	}
	if data, err := os.ReadFile(saved); err != nil || string(data) != "PK-zip-bytes" {
		t.Errorf("saved ZIP = %q, %v", data, err)
	}
	for _, gone := range []string{"download_url", "artifact_url", "sha256", "reused_from_job_id"} {
		if _, present := out[gone]; present {
			t.Errorf("%s is a web-version field", gone)
		}
	}
	if out["job_id"] != "1754000000-42" {
		t.Errorf("job_id = %v", out["job_id"])
	}
	if statusCalls.Load() < 2 {
		t.Errorf("status endpoint polled %d times, want >=2 (running → complete)", statusCalls.Load())
	}
}

func TestExportProject_FailedJobSurfacesError(t *testing.T) {
	fastExportPolling(t, 5*time.Second)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "j-failed", "status": "queued"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"job_id": "j-failed", "status": "failed",
			"error": "capture Go export snapshot: connection refused",
		})
	}))
	defer upstream.Close()

	jobs := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerExportProject(s, jobs) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "export_project",
		Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool transport error (want a tool-level error): %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for a failed job")
	}
	text := contentText(result)
	if !strings.Contains(text, "j-failed") || !strings.Contains(text, "capture Go export snapshot") {
		t.Errorf("error text = %q, want the job id + the job's error detail", text)
	}
}

func TestExportProject_OwnershipRejectionSurfaces(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "project access denied"})
	}))
	defer upstream.Close()

	jobs := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerExportProject(s, jobs) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "export_project",
		Arguments: map[string]any{"project_id": 999},
	})
	if err != nil {
		t.Fatalf("CallTool transport error (want a tool-level error): %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError when the jobs policy rejects the project")
	}
}

func TestExportProject_PollBudgetExceededKeepsJobHonest(t *testing.T) {
	fastExportPolling(t, 30*time.Millisecond)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "j-slow", "status": "queued"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"job_id": "j-slow", "status": "running"})
	}))
	defer upstream.Close()

	jobs := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerExportProject(s, jobs) })
	session := connectClient(t, srv.URL)

	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "export_project",
		Arguments: map[string]any{"project_id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool transport error (want a tool-level error): %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError once the poll budget is exhausted")
	}
	text := contentText(result)
	if !strings.Contains(text, "j-slow") || !strings.Contains(text, "Export page") {
		t.Errorf("timeout error = %q, want the job id + the Export page", text)
	}
}

func TestExportProject_SaveToMustBeAFolder(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
	}))
	defer upstream.Close()
	jobs := NewRESTProxy(upstream.URL)
	srv := newProxyOnlyTestServer(t, func(s *Server) { registerExportProject(s, jobs) })
	result, err := connectClient(t, srv.URL).CallTool(context.Background(), &CallToolParams{
		Name:      "export_project",
		Arguments: map[string]any{"project_id": 7, "save_to": filepath.Join(t.TempDir(), "missing")},
	})
	if err != nil || !result.IsError || !strings.Contains(contentText(result), "existing folder") {
		t.Fatalf("missing save_to folder: %v %s", err, contentText(result))
	}
}
