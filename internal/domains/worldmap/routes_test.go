package worldmap

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

func TestRoutesBesideTheGameView(t *testing.T) {
	env := newTestEnv(t)
	game := testGame(t)
	deps := domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return env.p, nil },
		Game: func() (fs.FS, error) { return game, nil },
	}
	mux := http.NewServeMux()
	svc := gameview.New(deps)
	gameview.RegisterWith(mux, svc)
	RegisterWith(mux, deps, svc)
	for path, status := range map[string]int{
		"/api/go/projects/1/gui-render/status":                  http.StatusOK,
		"/api/go/projects/1/gui-render/world-map?date=1936.1.1": http.StatusOK,
		"/api/go/projects/1/gui-render/world-map/legend":        http.StatusOK,
		"/api/go/projects/1/gui-render/world-map/land":          http.StatusOK,
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != status {
			t.Errorf("%s: %d %s", path, rec.Code, truncate(rec.Body.String()))
		}
	}
}
