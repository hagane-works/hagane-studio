package ideas

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func Register(mux *http.ServeMux, deps domains.Deps) {
	h := func(fn func(p *project.Project, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
		return apiutil.Handler(deps.Open, fn)
	}
	mux.HandleFunc("GET /api/go/projects/{id}/ideas", h(list))
	mux.HandleFunc("POST /api/go/projects/{id}/ideas", h(create))
	mux.HandleFunc("GET /api/go/projects/{id}/ideas/preview", h(preview))
	mux.HandleFunc("GET /api/go/projects/{id}/ideas/{iid}", h(get))
	mux.HandleFunc("PUT /api/go/projects/{id}/ideas/{iid}", h(update))
	mux.HandleFunc("DELETE /api/go/projects/{id}/ideas/{iid}", h(remove))
	mux.HandleFunc("POST /api/go/projects/{id}/ideas/{iid}/fork", h(fork))
	mux.HandleFunc("POST /api/go/projects/{id}/ideas/{iid}/parse-preview", h(parsePreview))
	mux.HandleFunc("POST /api/go/projects/{id}/ideas/{iid}/upload-picture", h(uploadPicture))
	mux.HandleFunc("GET /api/go/projects/{id}/ideas/{iid}/picture", h(picture))
	mux.HandleFunc("POST /api/go/projects/{id}/import/ideas", h(importIdeas))
	mux.HandleFunc("GET /api/go/vanilla/ideas", vanillaList)
	mux.HandleFunc("GET /api/go/vanilla/ideas/index", vanillaIdeasIndex)
	mux.HandleFunc("GET /api/go/vanilla/ideas/{category}/{idea_id}", vanillaDetail)
}

func ideaID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("iid"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errIdeaMissing
	}
	return id, nil
}

func list(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	items, err := listIdeas(p, r.URL.Query().Get("idea_type"), r.URL.Query().Get("country_tag"))
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, items)
	return nil
}

func get(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := ideaID(r)
	if err != nil {
		return err
	}
	item, err := getIdea(p, id)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func create(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	var in createInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&in); err != nil {
		return apiutil.ValidationError("Invalid idea payload")
	}
	in.normalize()
	item, err := createIdea(p, in)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}

func update(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := ideaID(r)
	if err != nil {
		return err
	}
	var patch patchInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&patch); err != nil {
		return apiutil.ValidationError("Invalid idea payload")
	}
	item, err := updateIdea(p, id, patch)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusOK, item)
	return nil
}

func remove(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := ideaID(r)
	if err != nil {
		return err
	}
	if err := deleteIdea(p, id); err != nil {
		return err
	}
	apiutil.NoContent(w)
	return nil
}

func fork(p *project.Project, w http.ResponseWriter, r *http.Request) error {
	id, err := ideaID(r)
	if err != nil {
		return err
	}
	var body struct {
		NewIdeaID string `json:"new_idea_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil || body.NewIdeaID == "" {
		return apiutil.ValidationError("new_idea_id is required")
	}
	newID := strings.TrimSpace(body.NewIdeaID)
	if newID == "" {
		return apiutil.BadRequestError("new_idea_id cannot be empty")
	}
	item, err := forkIdea(p, id, newID)
	if err != nil {
		return err
	}
	apiutil.JSON(w, http.StatusCreated, item)
	return nil
}
