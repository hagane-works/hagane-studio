package api

import (
	"errors"
	"haganeworks/internal/gameindex"
	"net/http"
	"strings"

	"haganeworks/internal/server"
	"haganeworks/internal/workbench"
)

func registerCountries(mux *http.ServeMux, p *projects) {
	mux.HandleFunc("GET /api/go/projects/{id}/countries", p.listCountries)
	mux.HandleFunc("POST /api/go/projects/{id}/countries", p.createCountry)
	mux.HandleFunc("GET /api/go/projects/{id}/countries/grouped", p.groupedCountries)
	mux.HandleFunc("POST /api/go/projects/{id}/countries/take-in", p.takeInCountry)
	mux.HandleFunc("GET /api/go/tags", countryTags)
}

func (p *projects) listCountries(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	items, err := workbench.ListCountries(opened)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"countries": items, "items": items, "total": len(items)})
}

func (p *projects) createCountry(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	var in workbench.CountryCreateInput
	if err := server.ReadJSON(r, maxJSONBody, &in); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, errors.New("Invalid country payload"))
		return
	}
	item, err := workbench.CreateCountry(opened, in)
	if err != nil {
		failCountry(w, err)
		return
	}
	server.JSON(w, http.StatusCreated, item)
}

func (p *projects) groupedCountries(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	report, err := workbench.GroupedCountries(opened)
	if err != nil {
		fail(w, err)
		return
	}
	server.JSON(w, http.StatusOK, report)
}

func (p *projects) takeInCountry(w http.ResponseWriter, r *http.Request) {
	opened, err := p.open(r)
	if err != nil {
		fail(w, err)
		return
	}
	var body struct {
		Tag string `json:"tag"`
	}
	if err := server.ReadJSON(r, 1<<16, &body); err != nil {
		server.Fail(w, http.StatusUnprocessableEntity, errors.New("Invalid country payload"))
		return
	}
	country, created, err := workbench.TakeInCountry(opened, strings.ToUpper(strings.TrimSpace(body.Tag)))
	if err != nil {
		failCountry(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	server.JSON(w, status, map[string]any{"country": country, "created": created})
}

func failCountry(w http.ResponseWriter, err error) {
	var conflict workbench.ErrCountryConflict
	switch {
	case errors.As(err, &conflict):
		server.Fail(w, http.StatusConflict, err)
	case errors.Is(err, workbench.ErrCountryMissing):
		server.Fail(w, http.StatusNotFound, err)
	default:
		fail(w, err)
	}
}

func countryTags(w http.ResponseWriter, r *http.Request) {
	if err := gameindex.CheckReady(gameindex.WorldTables); err != nil {
		gameindex.WriteHTTPError(w, err)
		return
	}
	result, err := workbench.SearchCountryTags(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		if !gameindex.WriteHTTPError(w, err) {
			fail(w, err)
		}
		return
	}
	server.JSON(w, http.StatusOK, result)
}
