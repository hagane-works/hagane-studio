package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"

	"haganeworks/internal/project"
	"haganeworks/internal/server"
	"haganeworks/internal/settings"
	"haganeworks/internal/workbench"
)

const maxJSONBody = 4 << 20

type projects struct {
	settings *settings.Store
	mu       sync.Mutex
	root     string
	store    *project.Store
}

var errNoProjectsDir = errors.New("choose a creation folder first")

func (p *projects) current() (*project.Store, error) {
	root := p.settings.Get().ProjectsDir
	if root == "" {
		return nil, errNoProjectsDir
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.store == nil || p.root != root {
		p.root = root
		p.store = project.NewStore(root)
	}
	return p.store, nil
}

func (p *projects) open(r *http.Request) (*project.Project, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return nil, project.ErrNotFound
	}
	store, err := p.current()
	if err != nil {
		return nil, err
	}
	return store.Get(id)
}

func registerProjects(mux *http.ServeMux, p *projects) {
	mux.HandleFunc("GET /api/go/projects", p.list)
	mux.HandleFunc("POST /api/go/projects", p.create)
	mux.HandleFunc("POST /api/go/projects/auto-detect-descriptor", autoDetectDescriptor)
	mux.HandleFunc("GET /api/go/projects/{id}", p.get)
	mux.HandleFunc("PUT /api/go/projects/{id}", p.update)
	mux.HandleFunc("DELETE /api/go/projects/{id}", p.delete)
	mux.HandleFunc("GET /api/go/projects/{id}/edit-revisions", p.editRevisions)
}

func (p *projects) list(w http.ResponseWriter, r *http.Request) {
	store, err := p.current()
	if err != nil {
		fail(w, err)
		return
	}
	rows, err := store.List()
	if err != nil {
		fail(w, err)
		return
	}
	items := make([]project.Row, len(rows))
	for i, row := range rows {
		items[i] = workbench.ProjectListRecord(row)
	}
	server.JSON(w, http.StatusOK, map[string]any{"projects": items, "items": items, "total": len(items)})
}

func (p *projects) create(w http.ResponseWriter, r *http.Request) {
	store, err := p.current()
	if err != nil {
		fail(w, err)
		return
	}
	var in workbench.CreateInput
	if err := server.ReadJSON(r, maxJSONBody, &in); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	created, err := workbench.CreateProject(store, in)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusCreated, created)
}

func (p *projects) get(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, workbench.ProjectRecord(opened))
}

func (p *projects) update(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxJSONBody))
	if err != nil {
		server.Fail(w, http.StatusBadRequest, err)
		return
	}
	updated, err := workbench.UpdateProject(opened, body)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, updated)
}

func (p *projects) delete(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	store, err := p.current()
	if err != nil {
		fail(w, err)
		return
	}
	if err := store.Delete(opened.ID()); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *projects) editRevisions(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	revisions := map[string]any{}
	for _, source := range []struct{ table, kind string }{{"focus_trees", "focus_tree"}, {"bookmarks", "bookmark"}} {
		rows, err := opened.Rows(source.table)
		if err != nil {
			fail(w, err)
			return
		}
		for _, row := range rows {
			if revision, ok := row["edit_revision"]; ok && revision != nil {

				revisions[source.kind+":"+strconv.FormatInt(project.Int64(row["id"]), 10)] = strconv.FormatInt(project.Int64(revision), 10)
			}
		}
	}
	server.JSON(w, http.StatusOK, map[string]any{"edit_revisions": revisions})
}

func autoDetectDescriptor(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DescriptorText string `json:"descriptor_text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	server.JSON(w, http.StatusOK, workbench.ParseDescriptor(body.DescriptorText))
}

func fail(w http.ResponseWriter, err error) {
	var validation workbench.ValidationError
	switch {
	case errors.Is(err, project.ErrNotFound), errors.Is(err, workbench.ErrNotFound):
		server.Fail(w, http.StatusNotFound, errors.New("not found"))
	case errors.As(err, &validation):
		server.Fail(w, http.StatusUnprocessableEntity, err)
	case errors.Is(err, errNoProjectsDir):
		server.Fail(w, http.StatusConflict, err)
	default:
		server.Fail(w, http.StatusInternalServerError, err)
	}
}
