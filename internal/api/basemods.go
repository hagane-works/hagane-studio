package api

import (
	"errors"
	"net/http"

	"haganeworks/internal/server"
	"haganeworks/internal/workbench"
)

func registerBaseMods(mux *http.ServeMux, p *projects) {
	mux.HandleFunc("GET /api/app/projects/{id}/base-mods", p.baseMods)
	mux.HandleFunc("PUT /api/app/projects/{id}/base-mods", p.setBaseMod)
}

func (p *projects) baseMods(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"base_mods": workbench.BaseMods(opened)})
}

func (p *projects) setBaseMod(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	var body struct {
		Dependency string `json:"dependency"`
		Folder     string `json:"folder"`
	}
	if err := server.ReadJSON(r, 1<<16, &body); err != nil {
		server.Fail(w, http.StatusBadRequest, err)
		return
	}
	if body.Folder != "" {
		if body.Folder, err = absoluteDir(body.Folder, false); err != nil {
			server.Fail(w, http.StatusUnprocessableEntity, err)
			return
		}
	}
	name, err := workbench.SetBaseMod(opened, body.Dependency, body.Folder)
	if errors.Is(err, workbench.ErrNotAMod) {
		server.JSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": err.Error(), "code": "not_a_mod"})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"dependency": name, "base_mods": workbench.BaseMods(opened)})
}
