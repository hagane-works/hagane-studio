package gameview

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"runtime"
	"slices"
	"testing"
	"time"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

func heapMB() float64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.HeapAlloc) / (1 << 20)
}

func TestRealGame(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("set HOI4_GAME_DIR to run against a real installation")
	}
	store := project.NewStore(t.TempDir())
	record, err := workbench.CreateProject(store, workbench.CreateInput{Name: "Real Game", CountryTag: "TST", CountryName: "Testland"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Get(project.Int64(record["id"]))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return os.DirFS(dir), nil },
	})
	t.Logf("heap before: %.0f MB", heapMB())
	get := func(target string) (int, []byte, time.Duration) {
		started := time.Now()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec.Code, rec.Body.Bytes(), time.Since(started)
	}
	base := "/api/go/projects/1/gui-render/"
	targets := []string{
		"status", "windows",
		"window?name=mainmenu_panel_bottom", "window?name=frontend_background", "window?name=topbar", "window?name=countrypoliticsview",
		"frontend", "bookmarks", "alerts",
		"characters?tag=GER&date=1936.1.1", "decisions?tag=GER&date=1936.1.1",
		"focus-tree?tag=GER", "production?tag=GER&date=1936.1.1", "diplomacy?tag=GER&date=1936.1.1",
		"tech-tree?tag=GER&date=1936.1.1&folder=infantry_folder", "country-stats?tag=GER&date=1936.1.1",
		"sprite?name=GFX_goal_unknown", "sprite?name=GFX_frontend_bg",
		"window?name=mainmenu_panel_bottom&lang=zh", "loc?keys=FE_SINGLE_PLAYER&lang=zh",
	}
	for round := 1; round <= 2; round++ {
		for _, target := range targets {
			code, body, took := get(base + target)
			if code != http.StatusOK {
				t.Errorf("round %d %s: %d %s", round, target, code, truncate(body))
				continue
			}
			t.Logf("round %d %-60s %3d %8d bytes %v", round, target, code, len(body), took.Round(time.Millisecond))
		}
		t.Logf("heap after round %d: %.0f MB", round, heapMB())
	}
	var status map[string]any
	_, body, _ := get(base + "status")
	_ = json.Unmarshal(body, &status)
	t.Logf("status: %v", status)

	_, topbar, _ := get(base + "window?name=topbar")
	textures := []string{"gfx/cursors/normal.png", "gfx/loadingscreens/load_1.png"}
	for _, match := range regexp.MustCompile(`/api/go/gui-vanilla/([^"?]+)`).FindAllStringSubmatch(string(topbar), -1) {
		if len(textures) < 8 && !slices.Contains(textures, match[1]) {
			textures = append(textures, match[1])
		}
	}
	for _, texture := range textures {
		code, body, took := get("/api/go/gui-vanilla/" + texture)
		t.Logf("texture %-45s %3d %8d bytes %v", texture, code, len(body), took.Round(time.Millisecond))
		code, _, took = get("/api/go/gui-vanilla/" + texture)
		t.Logf("texture %-45s %3d (cached) %v", texture, code, took.Round(time.Millisecond))
	}
	t.Logf("heap at end: %.0f MB", heapMB())

	sweepIdle(0)
	t.Logf("heap after letting idle caches go: %.0f MB", heapMB())
}

func truncate(body []byte) string {
	if len(body) > 300 {
		return string(body[:300]) + "…"
	}
	return string(body)
}
