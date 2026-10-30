package decisions

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
	mux.HandleFunc("/api/go/projects/{id}/decisions", h.route)
	mux.HandleFunc("/api/go/projects/{id}/decisions/{rest...}", h.route)
	mux.HandleFunc("/api/go/projects/{id}/decision-icons", h.iconCollection)
	mux.HandleFunc("/api/go/projects/{id}/decision-icons/{name}", h.iconCollection)
}

func writeJSON(w http.ResponseWriter, status int, body any) { apiutil.JSON(w, status, body) }

func detail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}

func methodNotAllowed(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": message, "detail": message})
}

func fail(w http.ResponseWriter, err error) {
	var (
		invalid  inputError
		notFound apiutil.NotFoundError
		conflict apiutil.ConflictError
		status   *apiutil.StatusError
	)
	switch {
	case errors.As(err, &invalid):
		detail(w, http.StatusBadRequest, err.Error())
	case errors.As(err, &notFound), errors.As(err, &conflict), errors.As(err, &status):
		apiutil.Fail(w, err)
	default:
		log.Printf("decisions: %v", err)
		detail(w, http.StatusServiceUnavailable, "decision store unavailable")
	}
}

func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20)).Decode(v)
}

func (h *handlers) route(w http.ResponseWriter, r *http.Request) {
	p, err := h.deps.Open(r)
	if err != nil {
		apiutil.Fail(w, err)
		return
	}
	var rest []string
	if raw := r.PathValue("rest"); raw != "" {
		rest = strings.Split(raw, "/")
	}
	if len(rest) > 3 {

		detail(w, http.StatusNotFound, "Project not found")
		return
	}

	if h.iconRoute(w, r, p, rest) {
		return
	}
	if len(rest) >= 1 && rest[0] == "categories" {
		h.categories(w, r, p, rest)
		return
	}
	if len(rest) == 2 && (rest[1] == "preview" || rest[1] == "parse-preview") {
		expected := http.MethodGet
		if rest[1] == "parse-preview" {
			expected = http.MethodPost
		}
		if r.Method != expected {
			detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil {
			writePreviewIntPathError(w, "decision_db_id", rest[0])
			return
		}
		if id <= 0 {
			detail(w, http.StatusNotFound, "Decision not found")
			return
		}
		if rest[1] == "preview" {
			decisionPreview(w, p, id)
		} else {
			decisionParsePreview(w, r, p, id)
		}
		return
	}
	switch len(rest) {
	case 0:
		h.decisionCollection(w, r, p)
	case 1, 2:
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil || id <= 0 {
			detail(w, http.StatusNotFound, "Decision not found")
			return
		}
		if len(rest) == 2 {
			if rest[1] != "fork" || r.Method != http.MethodPost {
				detail(w, http.StatusNotFound, "Not Found")
				return
			}
			h.fork(w, r, p, id)
			return
		}
		switch r.Method {
		case http.MethodGet:
			item, err := getDecision(p, id)
			respond(w, item, err, http.StatusOK)
		case http.MethodPut:
			var patch decisionPatch
			if err := decodeBody(r, &patch); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid decision payload")
				return
			}
			item, err := updateDecision(p, id, patch)
			respond(w, item, err, http.StatusOK)
		case http.MethodDelete:
			if err := deleteDecision(p, id); err != nil {
				fail(w, err)
				return
			}
			apiutil.NoContent(w)
		default:
			methodNotAllowed(w, "GET, PUT or DELETE only")
		}
	default:
		detail(w, http.StatusNotFound, "Not Found")
	}
}

func respond(w http.ResponseWriter, item any, err error, status int) {
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, status, item)
}

func (h *handlers) decisionCollection(w http.ResponseWriter, r *http.Request, p *project.Project) {
	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query()

		if scriptID := strings.TrimSpace(query.Get("decision_id")); scriptID != "" {
			item, err := decisionByScriptID(p, scriptID)
			if errors.Is(err, errDecisionMissing) {
				writeJSON(w, http.StatusOK, []Decision{})
				return
			}
			respond(w, []Decision{item}, err, http.StatusOK)
			return
		}
		var categoryID *int64
		if raw := query.Get("category_id"); raw != "" {
			if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
				categoryID = &v
			}
		}
		items, err := listDecisions(p, categoryID, query.Get("country_tag"))
		respond(w, items, err, http.StatusOK)
	case http.MethodPost:
		var in decisionCreateInput
		if err := decodeBody(r, &in); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid decision payload")
			return
		}
		if err := in.normalize(); err != nil {
			detail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		item, err := createDecision(p, in)
		respond(w, item, err, http.StatusCreated)
	default:
		methodNotAllowed(w, "GET or POST only")
	}
}

func (h *handlers) fork(w http.ResponseWriter, r *http.Request, p *project.Project, id int64) {
	var body struct {
		NewDecisionID string `json:"new_decision_id"`
	}
	if err := decodeBody(r, &body); err != nil || body.NewDecisionID == "" {
		detail(w, http.StatusUnprocessableEntity, "new_decision_id is required")
		return
	}
	newID := strings.TrimSpace(body.NewDecisionID)
	if newID == "" {
		detail(w, http.StatusBadRequest, "new_decision_id cannot be empty")
		return
	}
	item, err := forkDecision(p, id, newID)
	respond(w, item, err, http.StatusCreated)
}

func (h *handlers) categories(w http.ResponseWriter, r *http.Request, p *project.Project, rest []string) {
	switch {
	case len(rest) == 3 && rest[2] == "preview":
		if r.Method != http.MethodGet {
			detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}
		id, err := strconv.ParseInt(rest[1], 10, 64)
		if err != nil {
			writePreviewIntPathError(w, "cat_id", rest[1])
			return
		}
		if id <= 0 {
			detail(w, http.StatusNotFound, "Category not found")
			return
		}
		categoryPreview(w, p, id)
	case len(rest) == 1 && r.Method == http.MethodGet:
		items, err := listCategories(p, r.URL.Query().Get("country_tag"))
		respond(w, items, err, http.StatusOK)
	case len(rest) == 1 && r.Method == http.MethodPost:
		var in categoryCreateInput
		if err := decodeBody(r, &in); err != nil {
			detail(w, http.StatusUnprocessableEntity, "Invalid category payload")
			return
		}
		item, err := createCategory(p, in, categoryIDs(h.deps.Game))
		respond(w, item, err, http.StatusCreated)
	case len(rest) == 2:
		id, err := strconv.ParseInt(rest[1], 10, 64)
		if err != nil || id <= 0 {
			detail(w, http.StatusNotFound, "Category not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			item, err := getCategory(p, id)
			respond(w, item, err, http.StatusOK)
		case http.MethodPut:
			var patch categoryPatch
			if err := decodeBody(r, &patch); err != nil {
				detail(w, http.StatusUnprocessableEntity, "Invalid category payload")
				return
			}
			item, err := updateCategory(p, id, patch)
			respond(w, item, err, http.StatusOK)
		case http.MethodDelete:
			if err := deleteCategory(p, id); err != nil {
				fail(w, err)
				return
			}
			apiutil.NoContent(w)
		default:
			methodNotAllowed(w, "GET, PUT or DELETE only")
		}
	default:
		detail(w, http.StatusNotFound, "Not Found")
	}
}
