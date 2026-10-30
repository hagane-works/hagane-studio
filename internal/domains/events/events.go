package events

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

const prefix = "/api/go/projects/{id}/events"

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := &handlers{deps: deps}
	open := deps.Open
	mux.HandleFunc("GET "+prefix+"/namespaces", apiutil.Handler(open, h.listNamespaces))
	mux.HandleFunc("POST "+prefix+"/namespaces", apiutil.Handler(open, h.createNamespace))
	mux.HandleFunc("PUT "+prefix+"/namespaces/{nsid}", apiutil.Handler(open, h.updateNamespace))
	mux.HandleFunc("DELETE "+prefix+"/namespaces/{nsid}", apiutil.Handler(open, h.deleteNamespace))
	mux.HandleFunc("GET "+prefix+"/namespaces/{nsid}/preview", apiutil.Handler(open, h.namespacePreview))

	mux.HandleFunc("GET "+prefix, apiutil.Handler(open, h.listEvents))
	mux.HandleFunc("POST "+prefix, apiutil.Handler(open, h.createEvent))
	mux.HandleFunc("GET "+prefix+"/{eid}", apiutil.Handler(open, h.getEvent))
	mux.HandleFunc("PUT "+prefix+"/{eid}", apiutil.Handler(open, h.updateEvent))
	mux.HandleFunc("DELETE "+prefix+"/{eid}", apiutil.Handler(open, h.deleteEvent))
	mux.HandleFunc("POST "+prefix+"/{eid}/fork", apiutil.Handler(open, h.forkEvent))
	mux.HandleFunc("GET "+prefix+"/{eid}/preview", apiutil.Handler(open, h.eventPreview))
	mux.HandleFunc("POST "+prefix+"/{eid}/parse-preview", apiutil.Handler(open, h.parsePreview))
	mux.HandleFunc("GET "+prefix+"/{eid}/lint", apiutil.Handler(open, h.lint))
	mux.HandleFunc("POST "+prefix+"/{eid}/upload-picture", apiutil.Handler(open, h.uploadPicture))
	mux.HandleFunc("GET "+prefix+"/{eid}/picture", apiutil.Handler(open, h.picture))
}

type handlers struct {
	deps domains.Deps
}

func pathID(r *http.Request, name string, missing error) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, missing
	}
	return id, nil
}

func decodeBody(r *http.Request, limit int64, v any, detail string) error {
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, limit)).Decode(v); err != nil {
		return apiutil.ValidationError(detail)
	}
	return nil
}

func (h *handlers) listNamespaces(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	items, err := listNamespaces(p, r.URL.Query().Get("country_tag"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func (h *handlers) createNamespace(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in namespaceCreateInput
	if err := decodeBody(r, 1<<20, &in, "Invalid namespace payload"); err != nil {
		return err
	}
	item, err := createNamespace(p, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}

func (h *handlers) updateNamespace(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	nsID, err := pathID(r, "nsid", errNamespaceMissing)
	if err != nil {
		return err
	}
	var patch namespacePatch
	if err := decodeBody(r, 1<<20, &patch, "Invalid namespace payload"); err != nil {
		return err
	}
	item, err := updateNamespace(p, nsID, patch)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func (h *handlers) deleteNamespace(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	nsID, err := pathID(r, "nsid", errNamespaceMissing)
	if err != nil {
		return err
	}
	if err := deleteNamespace(p, nsID); err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func (h *handlers) listEvents(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()

	if scriptID := strings.TrimSpace(query.Get("event_id")); scriptID != "" {
		item, found, err := eventByScriptID(p, scriptID)
		if err != nil {
			return err
		}
		items := []eventResponse{}
		if found {
			items = append(items, item)
		}
		apiutil.JSON(w, http.StatusOK, items)
		return nil
	}
	var nsID *int64
	if raw := query.Get("namespace_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			nsID = &v
		}
	}
	items, err := listEvents(p, nsID, query.Get("country_tag"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func (h *handlers) createEvent(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in eventCreateInput
	if err := decodeBody(r, 8<<20, &in, "Invalid event payload"); err != nil {
		return err
	}
	if err := in.normalize(); err != nil {
		return apiutil.ValidationError(err.Error())
	}
	item, err := createEvent(p, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}

func (h *handlers) getEvent(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errEventMissing)
	if err != nil {
		return err
	}
	item, err := getEvent(p, eventID)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func (h *handlers) updateEvent(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errEventMissing)
	if err != nil {
		return err
	}
	var patch eventPatch
	if err := decodeBody(r, 8<<20, &patch, "Invalid event payload"); err != nil {
		return err
	}
	item, err := updateEvent(p, eventID, patch)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func (h *handlers) deleteEvent(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errEventMissing)
	if err != nil {
		return err
	}
	if err := deleteEvent(p, eventID); err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func (h *handlers) forkEvent(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errEventMissing)
	if err != nil {
		return err
	}
	var body struct {
		NewEventID string `json:"new_event_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&body); err != nil || body.NewEventID == "" {
		return apiutil.ValidationError("new_event_id is required")
	}
	item, err := forkEvent(p, eventID, strings.TrimSpace(body.NewEventID))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}

type lintWarning struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type lintResponse struct {
	Warnings []lintWarning `json:"warnings"`
}

func (h *handlers) lint(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	eventID, err := pathID(r, "eid", errEventMissing)
	if err != nil {
		return err
	}
	if _, err := getEvent(p, eventID); err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, lintResponse{Warnings: []lintWarning{}})
	return nil
}
