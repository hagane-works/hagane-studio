package tools

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/build"
	"haganeworks/internal/hoi4/staticcatalog"
)

func registerStatic(mux *http.ServeMux) {

	mux.HandleFunc("GET /api/go/cwtool/whitelist", func(w http.ResponseWriter, r *http.Request) {
		raw, err := gameindex.Peek(gameindex.CwtoolWhitelist)
		if errors.Is(err, gameindex.ErrNoGame) {
			raw, err = build.CwtoolRulesWithoutGame()
		}
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	})

	mux.HandleFunc("GET /api/go/qiuqi/modifiers", func(w http.ResponseWriter, r *http.Request) {
		translations := staticcatalog.DefaultModifierTranslations()
		writeJSON(w, http.StatusOK, map[string]any{
			"version": 1, "count": len(translations), "translations": translations,
		})
	})
	mux.HandleFunc("GET /api/go/qiuqi/modifier/{key...}", func(w http.ResponseWriter, r *http.Request) {
		rawKey := r.PathValue("key")
		key := strings.ToLower(strings.TrimSpace(rawKey))
		entry, ok := staticcatalog.DefaultModifierTranslations()[key]
		if key == "" || !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"detail": fmt.Sprintf("No QIUQI translation for modifier %q", rawKey),
			})
			return
		}
		writeJSON(w, http.StatusOK, entry)
	})

	mux.HandleFunc("GET /api/go/workflow-templates", func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{"templates": []any{}, "total": 0}
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("scope")), "marketplace") {
			body["scope"] = "marketplace"
		}
		writeJSON(w, http.StatusOK, body)
	})
	notFound := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "workflow template not found"})
	}
	mux.HandleFunc("GET /api/go/workflow-templates/{tid}", notFound)
	mux.HandleFunc("GET /api/go/workflow-templates/{tid}/preview", notFound)
}
