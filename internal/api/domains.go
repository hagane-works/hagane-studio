package api

import (
	"net/http"

	"haganeworks/internal/domains"
	"haganeworks/internal/domains/characters"
	"haganeworks/internal/domains/content"
	"haganeworks/internal/domains/countries"
	"haganeworks/internal/domains/decisions"
	"haganeworks/internal/domains/events"
	"haganeworks/internal/domains/focus"
	"haganeworks/internal/domains/gamesave"
	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/domains/ideas"
	"haganeworks/internal/domains/military"
	"haganeworks/internal/domains/states"
	"haganeworks/internal/domains/technology"
	"haganeworks/internal/domains/tools"
	"haganeworks/internal/domains/worldmap"
)

func registerDomains(mux *http.ServeMux, deps domains.Deps) {
	characters.Register(mux, deps)
	content.Register(mux, deps)
	countries.Register(mux, deps)
	decisions.Register(mux, deps)
	events.Register(mux, deps)
	focus.Register(mux, deps)
	ideas.Register(mux, deps)
	military.Register(mux, deps)
	states.Register(mux, deps)
	states.RegisterList(mux, deps)
	technology.Register(mux, deps)
	tools.Register(mux, deps)

	game := gameview.New(deps)
	gameview.RegisterWith(mux, game)
	gamesave.RegisterWith(mux, deps, game)
	worldmap.RegisterWith(mux, deps, game)
}
