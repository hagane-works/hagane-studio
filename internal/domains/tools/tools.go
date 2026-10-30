package tools

import (
	"net/http"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/domains/tools/exportcheck"
	"haganeworks/internal/domains/tools/lint"
	"haganeworks/internal/domains/tools/rawfiles"
)

func Register(mux *http.ServeMux, deps domains.Deps) {
	open := deps.Open
	mux.HandleFunc("GET /api/go/projects/{id}/usage", apiutil.Handler(open, usage))
	mux.HandleFunc("GET /api/go/projects/{id}/audit-log", apiutil.Handler(open, auditLog))
	mux.HandleFunc("GET /api/go/projects/{id}/edit-targets", apiutil.Handler(open, editTargets))

	registerImageBatch(mux, deps)
	registerVanilla(mux, deps)
	registerStatic(mux)
	registerBasemod(mux, deps)

	exportcheck.Register(mux, deps)
	lint.Register(mux, deps)
	rawfiles.Register(mux, deps)
}

func writeJSON(w http.ResponseWriter, status int, body any) { apiutil.JSON(w, status, body) }
