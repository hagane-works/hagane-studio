package military

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

type handlers struct {
	deps domains.Deps
}

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps}
	for resource, route := range map[string]func(http.ResponseWriter, *http.Request, *project.Project, []string){
		"equipment":            h.equipment,
		"sub-units":            h.subUnits,
		"division-templates":   h.divisionTemplates,
		"division-deployments": h.divisionDeployments,
		"oob-units":            h.oobUnits,
		"doctrines":            h.doctrines,
		"mios":                 h.mios,
	} {
		handler := h.open(route)
		mux.HandleFunc("/api/go/projects/{id}/"+resource, handler)
		mux.HandleFunc("/api/go/projects/{id}/"+resource+"/{rest...}", handler)
	}
}

func (h *handlers) open(route func(http.ResponseWriter, *http.Request, *project.Project, []string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.deps.Open(r)
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		var rest []string
		if raw := r.PathValue("rest"); raw != "" {
			rest = strings.Split(raw, "/")
		}
		route(w, r, p, rest)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) { apiutil.JSON(w, status, body) }

func detail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}

func methodError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": message})
}

func detailError(status int, message string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": message}}
}

func fail(w http.ResponseWriter, err error, op, unavailable string) {
	var (
		status   *apiutil.StatusError
		notFound apiutil.NotFoundError
	)
	switch {
	case errors.As(err, &status), errors.As(err, &notFound), errors.Is(err, project.ErrNotFound):
		apiutil.Fail(w, err)
	default:
		log.Printf("%s: %v", op, err)
		detail(w, http.StatusServiceUnavailable, unavailable)
	}
}

func decodeBody(r *http.Request, limit int64, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, limit)).Decode(v)
}

func itemID(rest []string) (int64, bool) {
	if len(rest) != 1 {
		return 0, false
	}
	id, err := strconv.ParseInt(rest[0], 10, 64)
	return id, err == nil && id > 0
}

func queryPtr(r *http.Request, key string) *string {
	if values, ok := r.URL.Query()[key]; ok && len(values) > 0 {
		return &values[0]
	}
	return nil
}

func queryInt64Ptr(r *http.Request, key string) *int64 {
	if values, ok := r.URL.Query()[key]; ok && len(values) > 0 {
		if n, err := strconv.ParseInt(values[0], 10, 64); err == nil {
			return &n
		}
	}
	return nil
}

func writePreviewIntPathError(w http.ResponseWriter, field, input string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []map[string]any{{
		"type": "int_parsing", "loc": []any{"path", field},
		"msg":   "Input should be a valid integer, unable to parse string as an integer",
		"input": input,
	}}})
}

func previewID(w http.ResponseWriter, raw, field, missing string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writePreviewIntPathError(w, field, raw)
		return 0, false
	}
	if id <= 0 {
		detail(w, http.StatusNotFound, missing)
		return 0, false
	}
	return id, true
}
