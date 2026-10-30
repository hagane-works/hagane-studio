package toolapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type RESTProxy struct {
	BaseURL string
	Client  *http.Client
	mapPath func(string) (string, error)
}

func NewRESTProxy(baseURL string) *RESTProxy {
	return &RESTProxy{BaseURL: baseURL, Client: &http.Client{Timeout: 30 * time.Second}}
}

func NewProgramProxy(api http.Handler) *RESTProxy {
	return &RESTProxy{
		BaseURL: "http://haganeworks.local",
		Client:  &http.Client{Timeout: 30 * time.Second, Transport: handlerTransport{api}},
		mapPath: programTargetPath,
	}
}

func (p *RESTProxy) withTimeout(timeout time.Duration) *RESTProxy {
	clone := *p
	clone.Client = &http.Client{Timeout: timeout, Transport: p.Client.Transport}
	return &clone
}

type handlerTransport struct{ handler http.Handler }

func (t handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		req.Body = http.NoBody
	}
	req.RequestURI = req.URL.RequestURI()
	response := &responseBuffer{header: http.Header{}}
	t.handler.ServeHTTP(response, req)
	if response.status == 0 {
		response.status = http.StatusOK
	}
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", response.status, http.StatusText(response.status)),
		StatusCode:    response.status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        response.header,
		Body:          io.NopCloser(bytes.NewReader(response.body.Bytes())),
		ContentLength: int64(response.body.Len()),
		Request:       req,
	}, nil
}

type responseBuffer struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *responseBuffer) Header() http.Header { return r.header }

func (r *responseBuffer) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

func (r *responseBuffer) Write(data []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	return r.body.Write(data)
}

type RESTError struct {
	Status int
	Detail string
	Code   string
}

func (e *RESTError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return fmt.Sprintf("upstream request failed with status %d", e.Status)
}

func (p *RESTProxy) Call(ctx context.Context, method, path string, body any) (map[string]any, error) {
	raw, err := p.callRaw(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decoding upstream response: %w", err)
	}
	enrichDisplayNames(decoded, contentLocaleFrom(ctx))
	return decoded, nil
}

func (p *RESTProxy) callRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	return p.callRawLimited(ctx, method, path, body, 8<<20)
}

func (p *RESTProxy) callRawLimited(ctx context.Context, method, path string, body any, maxResponseBytes int64) ([]byte, error) {
	if p == nil || strings.TrimSpace(p.BaseURL) == "" {
		return nil, errors.New("the project API is not configured")
	}
	if maxResponseBytes <= 0 {
		return nil, errors.New("response byte limit must be positive")
	}
	if p.mapPath != nil {
		mapped, err := p.mapPath(path)
		if err != nil {
			return nil, err
		}
		path = mapped
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if revisions, ok := ctx.Value(editTokensKey{}).(map[string]string); ok {
		encoded, _ := json.Marshal(revisions)
		req.Header.Set("X-Edit-Revisions", string(encoded))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxResponseBytes {
		return nil, fmt.Errorf("upstream response exceeds %d bytes", maxResponseBytes)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RESTError{Status: resp.StatusCode, Detail: extractRESTErrorDetail(resp.StatusCode, raw), Code: extractRESTErrorCode(raw)}
	}
	if receipt, ok := ctx.Value(editReceiptKey{}).(map[string]string); ok {
		var values map[string]string
		if json.Unmarshal([]byte(resp.Header.Get("X-Edit-Revisions")), &values) == nil {
			for key, token := range values {
				receipt[key] = token
			}
		}
	}
	return raw, nil
}

func extractRESTErrorCode(raw []byte) string {
	var envelope struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(raw, &envelope) != nil || len(envelope.Code) == 0 || len(envelope.Code) > 64 {
		return ""
	}
	for _, c := range envelope.Code {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return ""
		}
	}
	return envelope.Code
}

const (
	maxTechSurfaceErrorFindings = 12
	maxTechSurfaceErrorMessage  = 384
	maxTechSurfaceErrorIdentity = 128
)

var techSurfaceErrorFindingCodes = map[string]bool{
	"tech_surface_all_players_placement_extra": true, "tech_surface_all_players_placement_missing": true,
	"tech_surface_all_players_unsupported": true, "tech_surface_baseline_mismatch": true,
	"tech_surface_branch_id_invalid": true, "tech_surface_branch_limit": true,
	"tech_surface_branch_member_duplicate": true, "tech_surface_branch_member_folder_mismatch": true,
	"tech_surface_branch_member_invalid": true, "tech_surface_branch_member_missing": true,
	"tech_surface_branch_member_unlinked": true, "tech_surface_branch_missing": true,
	"tech_surface_branch_path_external": true, "tech_surface_compatibility_mode_invalid": true,
	"tech_surface_folder_not_certified": true, "tech_surface_gridbox_missing": true,
	"tech_surface_primary_placement_invalid": true, "tech_surface_profile_unknown": true,
	"tech_surface_required_dlc_placement_invalid": true, "tech_surface_root_duplicate": true,
	"tech_surface_root_folder_mismatch": true, "tech_surface_root_missing": true,
	"tech_surface_root_not_first_member": true, "tech_surface_root_not_member": true,
	"tech_surface_root_not_root": true, "tech_surface_schema_invalid": true,
	"tech_surface_slot_not_certified": true, "tech_surface_slot_occupied": true,
	"tech_surface_submod_unsupported": true, "tech_surface_timeline_exhausted": true,
	"tech_surface_timeline_position_mismatch": true, "tech_surface_timeline_year_mismatch": true,
}

var (
	techSurfaceDiagnosticURL    = regexp.MustCompile(`(?i)\b(?:https?|ftp|file)://[^\s<>"']+`)
	techSurfaceDiagnosticHeader = regexp.MustCompile(`(?i)\b(?:authorization|proxy-authorization|cookie|set-cookie|x-api-key)\s*[:=][^\r\n]*`)
	techSurfaceDiagnosticID     = regexp.MustCompile(`^[A-Za-z0-9_.:-]*$`)
)

type techSurfaceErrorFinding struct {
	Code    string
	Message string
	Branch  string
	TechID  string
}

func extractRESTErrorDetail(status int, raw []byte) string {
	if status != http.StatusBadRequest && status != http.StatusConflict && status != http.StatusUnprocessableEntity {
		return extractDetail(raw)
	}
	var envelope struct {
		Findings []techSurfaceErrorFinding `json:"findings"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return extractDetail(raw)
	}
	findings := make([]techSurfaceErrorFinding, 0, min(len(envelope.Findings), maxTechSurfaceErrorFindings))
	truncated := false
	for _, finding := range envelope.Findings {
		if !techSurfaceErrorFindingCodes[finding.Code] {
			continue
		}
		if len(findings) == maxTechSurfaceErrorFindings {
			truncated = true
			break
		}
		finding.Message = boundedTechSurfaceDiagnostic(finding.Message, maxTechSurfaceErrorMessage, &truncated)
		finding.Message = techSurfaceDiagnosticURL.ReplaceAllString(finding.Message, "[redacted URL]")
		finding.Message = techSurfaceDiagnosticHeader.ReplaceAllString(finding.Message, "[redacted header]")
		finding.Message = boundedTechSurfaceDiagnostic(finding.Message, maxTechSurfaceErrorMessage, &truncated)
		finding.Branch = boundedTechSurfaceDiagnosticID(finding.Branch, &truncated)
		finding.TechID = boundedTechSurfaceDiagnosticID(finding.TechID, &truncated)
		findings = append(findings, finding)
	}
	if len(findings) == 0 {
		return extractDetail(raw)
	}

	result, _ := json.Marshal(struct {
		Code        string                    `json:"code"`
		Detail      string                    `json:"detail"`
		Recoverable bool                      `json:"recoverable"`
		Findings    []techSurfaceErrorFinding `json:"findings"`
		Truncated   bool                      `json:"truncated"`
	}{"invalid_request", "Technology surface validation failed. Resolve these findings before retrying.", true, findings, truncated})
	return string(result)
}

func boundedTechSurfaceDiagnostic(value string, limit int, truncated *bool) string {
	value = strings.TrimSpace(value)
	count := 0
	for index := range value {
		if count == limit {
			*truncated = true
			return value[:index]
		}
		count++
	}
	return value
}

func boundedTechSurfaceDiagnosticID(value string, truncated *bool) string {
	value = strings.TrimSpace(value)

	if len(value) > maxTechSurfaceErrorIdentity || !techSurfaceDiagnosticID.MatchString(value) {
		*truncated = true
		return ""
	}
	return value
}

func extractDetail(raw []byte) string {
	var envelope struct {
		Detail any `json:"detail"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Detail == nil {
		return string(raw)
	}
	if s, ok := envelope.Detail.(string); ok {
		return s
	}
	reencoded, err := json.Marshal(envelope.Detail)
	if err != nil {
		return string(raw)
	}
	return string(reencoded)
}

func proxyAuthedCall(ctx context.Context, proxy *RESTProxy, method, path string, body any) (map[string]any, error) {
	result, err := proxy.Call(ctx, method, path, body)
	if err != nil {
		var restErr *RESTError
		if errors.As(err, &restErr) {
			return nil, restErr
		}

		return nil, errors.New("The program did not complete this request. Please retry shortly.")
	}
	return result, nil
}

func boolField(m map[string]any, key string) bool {
	b, _ := m[key].(bool)
	return b
}
