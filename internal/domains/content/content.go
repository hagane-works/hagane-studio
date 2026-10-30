package content

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

type routeFunc func(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string)

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps}
	routes := map[string]routeFunc{
		"dynamic-modifiers":      h.dynamicModifiers,
		"autonomy-states":        h.autonomyStates,
		"scripted-snippets":      h.scriptedSnippets,
		"scripted-localisations": h.scriptedLocalisations,
		"opinion-modifiers":      h.opinionModifiers,
		"modifier-definitions":   h.modifierDefinitions,
		"wargoals":               h.wargoals,
		"ai-strategy-plans":      h.aiStrategyPlans,
		"ai-templates":           h.aiTemplates,
		"intelligence-agencies":  h.agencies,
		"super-events":           h.superEvents,
		"special-projects":       h.specialProjects,
		"ui-panels":              h.uiPanels,
	}
	for name, fn := range routes {
		handler := h.route(fn)
		mux.HandleFunc("/api/go/projects/{id}/"+name, handler)
		mux.HandleFunc("/api/go/projects/{id}/"+name+"/{rest...}", handler)
	}
}

func (h *handlers) route(fn routeFunc) http.HandlerFunc {
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
		fn(p, w, r, rest)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) { apiutil.JSON(w, status, body) }

func detail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}

func statusErr(status int, message string) error {
	return &apiutil.StatusError{Status: status, Body: map[string]any{"detail": message}}
}

var errConstraint = errors.New("constraint violation")

func fail(w http.ResponseWriter, err error, unavailable string) {
	var status *apiutil.StatusError
	if errors.As(err, &status) {
		apiutil.Fail(w, err)
		return
	}
	if !errors.Is(err, errConstraint) {
		log.Printf("content: %s: %v", unavailable, err)
	}
	detail(w, http.StatusServiceUnavailable, unavailable)
}

func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20)).Decode(v)
}

type crud struct {
	listKey string

	missing string

	unavailable string

	bothKeys bool

	list   func(p *project.Project, r *http.Request) (any, int, error)
	create func(p *project.Project, r *http.Request) (any, error)

	get    func(p *project.Project, id int64) (any, error)
	update func(p *project.Project, r *http.Request, id int64) (any, error)
	remove func(p *project.Project, id int64) error
}

func (c crud) methodNotAllowed(w http.ResponseWriter, message string) {
	body := map[string]string{"error": message}
	if c.bothKeys {
		body["detail"] = message
	}
	writeJSON(w, http.StatusMethodNotAllowed, body)
}

func (c crud) serve(p *project.Project, w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, total, err := c.list(p, r)
			if err != nil {
				fail(w, err, c.unavailable)
				return
			}
			if c.listKey == "" {
				writeJSON(w, http.StatusOK, items)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{c.listKey: items, "items": items, "total": total})
		case http.MethodPost:
			item, err := c.create(p, r)
			if err != nil {
				fail(w, err, c.unavailable)
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			c.methodNotAllowed(w, "GET or POST only")
		}
		return
	}
	id, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil || id <= 0 || len(rest) != 1 {
		detail(w, http.StatusNotFound, c.missing)
		return
	}
	switch {
	case r.Method == http.MethodGet && c.get != nil:
		item, err := c.get(p, id)
		if err != nil {
			fail(w, err, c.unavailable)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case r.Method == http.MethodPut:
		item, err := c.update(p, r, id)
		if err != nil {
			fail(w, err, c.unavailable)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case r.Method == http.MethodDelete:
		if err := c.remove(p, id); err != nil {
			fail(w, err, c.unavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case c.get == nil:
		c.methodNotAllowed(w, "PUT or DELETE only")
	default:
		c.methodNotAllowed(w, "GET, PUT or DELETE only")
	}
}

func queryPtr(r *http.Request, key string) *string {
	if values, ok := r.URL.Query()[key]; ok && len(values) > 0 {
		return &values[0]
	}
	return nil
}
