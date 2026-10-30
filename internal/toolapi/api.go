package toolapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxCallBody = 32 << 20

func Register(mux *http.ServeMux, api http.Handler) {
	proxy := NewProgramProxy(api)
	h := &handler{tools: New(proxy), proxy: proxy}
	mux.HandleFunc("GET /api/tools", h.list)
	mux.HandleFunc("GET /api/tools/fields", h.fieldsIndex)
	mux.HandleFunc("GET /api/tools/fields/{doc}", h.fieldDoc)
	mux.HandleFunc("GET /api/tools/{name}", h.describe)
	mux.HandleFunc("POST /api/tools/{name}", h.call)
}

type handler struct {
	tools *Server
	proxy *RESTProxy
}

type programStatus struct {
	Version     string         `json:"version"`
	Port        int64          `json:"port"`
	Ready       bool           `json:"ready"`
	ProjectsDir string         `json:"projects_dir"`
	GameDir     string         `json:"game_dir"`
	GameIndex   map[string]any `json:"game_index,omitempty"`
}

func (h *handler) status(ctx context.Context) (programStatus, error) {
	app, err := h.proxy.Call(ctx, "GET", "/api/app/status", nil)
	if err != nil {
		return programStatus{}, err
	}
	index, err := h.proxy.Call(ctx, "GET", "/api/app/game-index", nil)
	if err != nil {
		return programStatus{}, err
	}
	settings, _ := app["settings"].(map[string]any)
	return programStatus{
		Version: stringField(app, "version"), Port: int64Field(app, "port"), Ready: boolField(app, "ready"),
		ProjectsDir: stringField(settings, "projects_dir"), GameDir: stringField(settings, "game_dir"),
		GameIndex: index,
	}, nil
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	status, err := h.status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "status_unavailable", err.Error())
		return
	}
	tools, err := h.tools.ListTools(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tools_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"program": status, "tools": tools})
}

func (h *handler) find(ctx context.Context, name string) (*Tool, error) {
	tools, err := h.tools.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	for _, tool := range tools {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, &UnknownToolError{Name: name}
}

func (h *handler) describe(w http.ResponseWriter, r *http.Request) {
	tool, err := h.find(r.Context(), r.PathValue("name"))
	if err != nil {
		writeToolNotFound(w, err)
		return
	}
	out := map[string]any{"name": tool.Name, "description": tool.Description, "input_schema": tool.InputSchema}
	if workbench, ok := toolFieldRef[tool.Name]; ok {
		if doc, err := readFieldDoc(workbench); err == nil {
			out["field_reference"] = map[string]any{"name": workbench, "text": doc}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) fieldsIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"names": append(listFieldWorkbenches(), "tokens"), "text": buildFieldsIndex()})
}

func (h *handler) fieldDoc(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("doc")
	doc, err := readFieldDoc(name)
	if err != nil {
		writeError(w, http.StatusNotFound, "unknown_field_reference", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "text": doc})
}

func (h *handler) call(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCallBody))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "input_too_large", fmt.Sprintf("the input is over %d MB", maxCallBody>>20))
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_arguments", err.Error())
		return
	}
	if strings.TrimSpace(string(raw)) == "" {
		raw = []byte("{}")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		writeError(w, http.StatusBadRequest, "invalid_arguments", "the input must be a JSON object")
		return
	}
	result, err := h.tools.CallTool(r.Context(), name, raw)
	if err != nil {
		var unknown *UnknownToolError
		if errors.As(err, &unknown) {
			writeToolNotFound(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_arguments", err.Error())
		return
	}
	status, body := resultBody(result)
	writeJSON(w, status, body)
}

func resultBody(result *CallToolResult) (int, map[string]any) {
	body := map[string]any{}
	if result.StructuredContent != nil {
		if fields, ok := structuredFields(result.StructuredContent); ok && fields != nil {
			body = fields
		} else {
			body["result"] = result.StructuredContent
		}
	}
	var texts []string
	var images []map[string]any
	for _, content := range result.Content {
		switch typed := content.(type) {
		case *TextContent:
			texts = append(texts, typed.Text)
		case *ImageContent:
			images = append(images, map[string]any{"mime_type": typed.MIMEType, "data_base64": base64.StdEncoding.EncodeToString(typed.Data)})
		}
	}
	if len(images) > 0 {
		body["images"] = images
	}
	if result.StructuredContent == nil && len(texts) > 0 && !result.IsError {

		var fields map[string]any
		if len(texts) == 1 && json.Unmarshal([]byte(texts[0]), &fields) == nil && fields != nil {
			for key, value := range fields {
				body[key] = value
			}
		} else {
			body["text"] = strings.Join(texts, "\n")
		}
	}
	if !result.IsError {
		return http.StatusOK, body
	}
	status, code := http.StatusUnprocessableEntity, "tool_error"
	var restErr *RESTError
	switch {
	case errors.Is(result.err, errInvalidArguments):
		status, code = http.StatusBadRequest, "invalid_arguments"
	case errors.As(result.err, &restErr) && (restErr.Status >= 400 && restErr.Status < 500 || restErr.Status == http.StatusServiceUnavailable):
		status = restErr.Status
		if restErr.Code != "" {
			code = restErr.Code
		}
	}
	if existing, _ := body["code"].(string); existing != "" {
		code = existing
	}
	body["code"] = code
	if _, present := body["detail"]; !present {
		detail, _ := body["error"].(string)
		if detail == "" {
			detail = strings.Join(texts, "\n")
		}
		if detail == "" {
			detail = "the tool failed"
		}
		body["detail"] = detail
	}
	return status, body
}

func writeToolNotFound(w http.ResponseWriter, err error) {
	writeError(w, http.StatusNotFound, "unknown_tool", err.Error()+"; `hw tools` lists them")
}

func writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]any{"detail": detail, "code": code})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	encoded, err := json.Marshal(body)
	if err != nil {
		status = http.StatusInternalServerError
		encoded = []byte(`{"detail":"the answer could not be encoded","code":"encode_failed"}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}
