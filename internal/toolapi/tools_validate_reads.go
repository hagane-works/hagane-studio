package toolapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

type validateProjectIn struct {
	ProjectID  int64  `json:"project_id"`
	Offset     int    `json:"offset,omitempty" jsonschema:"zero-based page offset; use returned next_offset, not a file line number"`
	Limit      int    `json:"limit,omitempty" jsonschema:"default 20, maximum 100; the byte budget may return fewer items"`
	Revision   string `json:"revision,omitempty" jsonschema:"returned validation revision; required for offset>0 and issue details"`
	IssueIndex *int   `json:"issue_index,omitempty" jsonschema:"exact issue_index from a compact page; requires revision"`
	DetailPath string `json:"detail_path,omitempty" jsonschema:"JSON pointer from an issue/detail locator; empty reads the issue's child summaries, never its full body"`
}

func registerValidateProject(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "validate_project",

		Description: "Run the program's ENTITY-level validation (cross-references, CTD/collision " +
			"checks) and return issues + entity counts, so you can self-correct before export. " +
			"NOT a complete picture: artifact-level warnings (produced while building the ZIP — " +
			"dropped content, skipped images, unreferenced super events) are a SEPARATE stream " +
			"that only export_project returns. A clean result here does NOT mean the project " +
			"exports cleanly. Read both before treating a project as release-ready. " +
			"Results are compact pages (default 20, max 100), with total/done/next_offset, " +
			"global issue_counts, unchanged entity summary counts and revision; is_snapshot=false. " +
			"Continue only for the requested coverage, with the SAME revision; a changed revision requires restarting. " +
			"Inspect a necessary issue via issue_index + revision + detail_path (JSON pointer). " +
			"Detail pages contain child summaries or 256-codepoint text chunks, never an unbounded issue.",
	}, func(ctx context.Context, _ *CallToolRequest, in validateProjectIn) (*CallToolResult, map[string]any, error) {
		out, err := fetchValidationPayload(ctx, proxy, in)
		return nil, out, err
	})
}

func fetchValidationPayload(ctx context.Context, proxy *RESTProxy, in validateProjectIn) (map[string]any, error) {
	if in.Offset < 0 || in.Limit < 0 || in.Limit > 100 || ((in.Offset > 0 || in.IssueIndex != nil) && in.Revision == "") {
		return nil, fmt.Errorf("invalid_request: use limit 1..100 and the previous revision for continuation/detail")
	}
	q := url.Values{"view": {"compact"}, "offset": {strconv.Itoa(in.Offset)}}
	if in.Limit != 0 {
		q.Set("limit", strconv.Itoa(in.Limit))
	}
	if in.Revision != "" {
		q.Set("revision", in.Revision)
	}
	if in.IssueIndex != nil {
		q.Set("issue_index", strconv.Itoa(*in.IssueIndex))
		q.Set("detail_path", in.DetailPath)
	} else if in.DetailPath != "" {
		return nil, fmt.Errorf("invalid_request: detail_path requires issue_index and revision")
	}
	result, err := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/export/validate?%s", in.ProjectID, q.Encode()), nil)
	if err != nil {
		var upstream *RESTError
		if errors.As(err, &upstream) {
			if upstream.Status == 409 {
				return nil, fmt.Errorf("invalid_request: validation changed; restart at offset 0 without mixing revisions")
			}
			if upstream.Status == 422 {
				return nil, fmt.Errorf("invalid_request: invalid validation page/detail; use offset>=0, limit 1..100 and the returned revision/locators")
			}
		}
		return nil, err
	}
	if !validCompactValidationPage(result, in) {
		return nil, fmt.Errorf("compact validation unavailable: the service did not return a bounded validation page; no clean result was established")
	}

	issues := result["issues"]
	if issues == nil {
		issues = []any{}
	}
	summary := result["summary"]
	if summary == nil {
		summary = map[string]any{}
	}

	out := map[string]any{
		"issues":  issues,
		"summary": summary,
		"coverage": map[string]any{
			"includes": "entity-level validation (cross-references, CTD/collision checks)",
			"excludes": "artifact-level warnings raised while building the ZIP " +
				"(dropped content, skipped images, unreferenced super events)",
			"excluded_warnings_available_from": "export_project",
			"note": "These two warning streams do not overlap. A clean result here " +
				"does NOT mean the project exports cleanly.",
		},
	}
	for _, key := range []string{"view", "issue_counts", "issue_total", "revision", "is_snapshot", "offset", "limit", "total", "returned", "next_offset", "done", "issue_index", "detail_path", "detail_items", "detail_unit"} {
		if value, present := result[key]; present {
			out[key] = value
		}
	}
	return out, nil
}

func validCompactValidationPage(result map[string]any, in validateProjectIn) bool {
	revision, _ := result["revision"].(string)
	done, ok := result["done"].(bool)
	if result["view"] != "compact" || result["is_snapshot"] != false || revision == "" || !ok {
		return false
	}
	if _, ok := result["summary"].(map[string]any); !ok {
		return false
	}
	key := "issues"
	if in.IssueIndex != nil {
		key = "detail_items"
	}
	items, ok := result[key].([]any)
	if !ok {
		return false
	}
	total, okTotal := result["total"].(float64)
	returned, okReturned := result["returned"].(float64)
	offset, okOffset := result["offset"].(float64)
	if !okTotal || !okReturned || !okOffset || total < 0 || total != float64(int64(total)) ||
		offset != float64(in.Offset) || returned != float64(len(items)) || returned > total-offset || (returned == 0 && total > 0) {
		return false
	}
	if done != (offset+returned == total) {
		return false
	}
	if !done && result["next_offset"] != offset+returned {
		return false
	}
	return true
}
