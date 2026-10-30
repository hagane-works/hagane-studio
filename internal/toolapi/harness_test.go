package toolapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/gameindextest"
)

type Deps struct {
	Proxy *RESTProxy

	ExportJobs *RESTProxy
}

type CallToolParams struct {
	Name      string
	Arguments any
}

type ListToolsParams struct{}

type wireContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
}

type wireResult struct {
	IsError    bool            `json:"is_error"`
	Content    []wireContent   `json:"content"`
	Structured json.RawMessage `json:"structured,omitempty"`
}

func toolHTTPHandler(getServer func(*http.Request) *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server := getServer(r)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			tools, err := server.ListTools(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(tools)
			return
		}
		var call struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), harnessToolKey{}, call.Name)
		result, err := server.CallTool(ctx, call.Name, call.Arguments)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		out := wireResult{IsError: result.IsError}
		for _, content := range result.Content {
			switch typed := content.(type) {
			case *TextContent:
				out.Content = append(out.Content, wireContent{Type: "text", Text: typed.Text})
			case *ImageContent:
				out.Content = append(out.Content, wireContent{Type: "image", Data: base64.StdEncoding.EncodeToString(typed.Data), MIMEType: typed.MIMEType})
			}
		}
		if result.StructuredContent != nil {
			encoded, err := json.Marshal(result.StructuredContent)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out.Structured = encoded
		}
		_ = json.NewEncoder(w).Encode(out)
	})
}

type harnessToolKey struct{}

type testSession struct{ endpoint string }

func connectClient(t *testing.T, endpoint string) *testSession {
	t.Helper()
	return &testSession{endpoint: endpoint}
}

func (s *testSession) Close() error { return nil }

func (s *testSession) CallTool(ctx context.Context, params *CallToolParams) (*CallToolResult, error) {
	arguments := params.Arguments
	if arguments == nil {
		arguments = map[string]any{}
	}
	encoded, err := json.Marshal(map[string]any{"name": params.Name, "arguments": arguments})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var detail bytes.Buffer
		_, _ = detail.ReadFrom(res.Body)
		return nil, fmt.Errorf("calling %s: %d %s", params.Name, res.StatusCode, bytes.TrimSpace(detail.Bytes()))
	}
	var wire wireResult
	if err := json.NewDecoder(res.Body).Decode(&wire); err != nil {
		return nil, err
	}
	result := &CallToolResult{IsError: wire.IsError}
	for _, content := range wire.Content {
		switch content.Type {
		case "text":
			result.Content = append(result.Content, &TextContent{Text: content.Text})
		case "image":
			data, _ := base64.StdEncoding.DecodeString(content.Data)
			result.Content = append(result.Content, &ImageContent{Data: data, MIMEType: content.MIMEType})
		}
	}
	if len(wire.Structured) > 0 {
		var structured any
		if err := json.Unmarshal(wire.Structured, &structured); err != nil {
			return nil, err
		}
		result.StructuredContent = structured
	}
	return result, nil
}

func (s *testSession) ListTools(ctx context.Context, _ *ListToolsParams) (*ListToolsResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var tools []*Tool
	if err := json.NewDecoder(res.Body).Decode(&tools); err != nil {
		return nil, err
	}
	return &ListToolsResult{Tools: tools}, nil
}

func toolRequest(name string, args map[string]any) *CallToolRequest {
	raw, _ := json.Marshal(args)
	return &CallToolRequest{Params: &CallToolParamsRaw{Name: name, Arguments: raw}}
}

func TestMain(m *testing.M) {
	restore := gameindex.UseStatic(gameindextest.Datasets())
	code := m.Run()
	restore()
	os.Exit(code)
}
