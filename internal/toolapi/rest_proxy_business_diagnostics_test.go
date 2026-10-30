package toolapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

const realTechSurfaceBusinessError = `{"detail":"Technology surface branch is not exportable","findings":[{"Code":"tech_surface_branch_member_missing","Message":"Project-owned technologies in a certified P0 folder must be added through a declared branch.","Branch":"","TechID":"GER_qa431_research"},{"Code":"tech_surface_gridbox_missing","Message":"This project-owned root technology has no generated <root>_tree gridbox, so it would not be visible in the game.","Branch":"","TechID":"GER_qa431_research"}]}`

func TestRESTProxyTechSurfaceFindingsReachTheToolResult(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/projects/7/technologies/surface/branches" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(realTechSurfaceBusinessError))
	}))
	defer upstream.Close()
	server := newTechSurfaceProxyTestServer(t, NewRESTProxy(upstream.URL))
	session := connectClient(t, server.URL)
	result, err := session.CallTool(context.Background(), &CallToolParams{
		Name:      "create_technology_surface_branch",
		Arguments: map[string]any{"project_id": 7, "entry_id": "army", "target_id": "infantry", "root": map[string]any{"tech_id": "MOD_root"}},
	})
	if err != nil || !result.IsError {
		t.Fatalf("expected actionable tool error: result=%+v err=%v", result, err)
	}
	var modelMessage struct {
		Code        string                    `json:"code"`
		Recoverable bool                      `json:"recoverable"`
		Findings    []techSurfaceErrorFinding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(contentText(result)), &modelMessage); err != nil {
		t.Fatalf("model received no machine-readable diagnostic: %s: %v", contentText(result), err)
	}
	if modelMessage.Code != "invalid_request" || !modelMessage.Recoverable || len(modelMessage.Findings) != 2 ||
		modelMessage.Findings[0].Code != "tech_surface_branch_member_missing" || modelMessage.Findings[1].Code != "tech_surface_gridbox_missing" ||
		modelMessage.Findings[0].TechID != "GER_qa431_research" || !strings.Contains(modelMessage.Findings[1].Message, "gridbox") {
		t.Fatalf("business findings lost at the tool API boundary: %+v", modelMessage)
	}
}

func TestTechSurfaceBusinessDiagnosticProjectionDropsUnknownSensitiveFields(t *testing.T) {
	raw := []byte(`{"detail":"Bearer top-level-private https://private.invalid/body","headers":{"Authorization":"Bearer header-private"},"url":"https://private.invalid/top","findings":[{"Code":"tech_surface_gridbox_missing","Message":"Missing gridbox. https://private.invalid/message\nAuthorization: Bearer message-private","Branch":"MOD_branch","TechID":"MOD_root","raw_body":"private-finding-value","headers":{"Cookie":"private-cookie"}},{"Code":"provider_error","Message":"private-provider-message"}]}`)
	for _, status := range []int{400, 409, 422} {
		projected := extractRESTErrorDetail(status, raw)
		for _, forbidden := range []string{"top-level-private", "header-private", "private.invalid", "message-private", "private-finding-value", "private-cookie", "private-provider-message", `"headers"`, `"url"`, `"raw_body"`} {
			if strings.Contains(projected, forbidden) {
				t.Fatalf("status %d leaked %q: %s", status, forbidden, projected)
			}
		}
		var result map[string]any
		if json.Unmarshal([]byte(projected), &result) != nil || result["recoverable"] != true || len(result["findings"].([]any)) != 1 {
			t.Fatalf("wrong projection: %s", projected)
		}
		finding := result["findings"].([]any)[0].(map[string]any)
		if len(finding) != 4 || finding["TechID"] != "MOD_root" || finding["Branch"] != "MOD_branch" {
			t.Fatalf("finding allowlist mismatch: %+v", finding)
		}
	}
}

func TestTechSurfaceBusinessDiagnosticProjectionIsBoundedAndMarksTruncation(t *testing.T) {
	findings := []map[string]any{}
	for index := 0; index < maxTechSurfaceErrorFindings+10; index++ {
		findings = append(findings, map[string]any{"Code": "tech_surface_gridbox_missing", "Message": strings.Repeat("科技", 2000), "Branch": strings.Repeat("a", 2000), "TechID": "https://private.invalid/id"})
	}
	raw, _ := json.Marshal(map[string]any{"detail": "not exportable", "findings": findings})
	var result struct {
		Findings  []techSurfaceErrorFinding `json:"findings"`
		Truncated bool                      `json:"truncated"`
	}
	projected := extractRESTErrorDetail(422, raw)
	if json.Unmarshal([]byte(projected), &result) != nil || !result.Truncated || len(result.Findings) != maxTechSurfaceErrorFindings {
		t.Fatalf("unbounded findings: %s", projected)
	}
	for _, finding := range result.Findings {
		if !utf8.ValidString(finding.Message) || utf8.RuneCountInString(finding.Message) > maxTechSurfaceErrorMessage ||
			len(finding.Branch) > maxTechSurfaceErrorIdentity || finding.TechID != "" {
			t.Fatalf("unbounded or unsafe finding: %+v", finding)
		}
	}

	raw, _ = json.Marshal(map[string]any{"findings": []map[string]any{{"Code": "tech_surface_gridbox_missing", "Message": strings.Repeat("http://a ", 100), "TechID": strings.Repeat("a", 200)}}})
	if json.Unmarshal([]byte(extractRESTErrorDetail(422, raw)), &result) != nil || !result.Truncated ||
		utf8.RuneCountInString(result.Findings[0].Message) > maxTechSurfaceErrorMessage || result.Findings[0].TechID != "" {
		t.Fatalf("redaction expanded the bound or returned a partial identity: %+v", result)
	}
}

func TestRESTProxyUnrelatedErrorsRetainTheirExistingDetail(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{
		{404, realTechSurfaceBusinessError},
		{500, realTechSurfaceBusinessError},
		{422, `{"detail":"Sub-unit not found"}`},
		{409, `{"detail":"version conflict","findings":[{"Code":"unknown_business_code","Message":"unchanged"}]}`},
		{400, `{"detail":{"code":"invalid_request","message":"structured existing detail"}}`},
		{422, `not JSON`},
	} {
		if got, want := extractRESTErrorDetail(test.status, []byte(test.body)), extractDetail([]byte(test.body)); got != want {
			t.Fatalf("status %d changed unrelated error: got=%q want=%q", test.status, got, want)
		}
	}
}
