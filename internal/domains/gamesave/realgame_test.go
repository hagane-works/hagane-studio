package gamesave

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"haganeworks/internal/domains"
	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

func TestRealGameSaves(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("set HOI4_GAME_DIR to run against a real installation")
	}
	p := newProject(t)
	deps := domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return os.DirFS(dir), nil },
	}
	svc := gameview.New(deps)
	mux := http.NewServeMux()
	gameview.RegisterWith(mux, svc)
	RegisterWith(mux, deps, svc)
	post := func(route string, body any) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(body)
		started := time.Now()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/go/projects/1/gui-render/"+route, bytes.NewReader(raw)))
		out := map[string]any{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", route, rec.Code, rec.Body.String())
		}
		summary := map[string]any{}
		for key, value := range out {
			if key != "undo" && key != "template" {
				summary[key] = value
			}
		}
		t.Logf("%-22s %v %v", route, time.Since(started).Round(time.Millisecond), summary)
		return out
	}

	out := post("equipment-save", map[string]any{"name": "infantry_equipment_1", "values": map[string]any{"soft_attack": 7.5}})
	if out["imported"] != true {
		t.Fatalf("equipment-save = %v", out)
	}
	index, _, err := svc.Index(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if _, body, line, ok := index.EquipmentSource("infantry_equipment_1"); !ok || !strings.Contains(string(body), "soft_attack = 7.5") {
		t.Errorf("the game view still reads the game's infantry_equipment_1 (line %d)", line)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/go/projects/1/gui-render/decision-save",
		strings.NewReader(`{"category":"political_actions","name":"targeted_race_for_the_bomb","cost":75}`)))
	var saved map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil || rec.Code != http.StatusOK || saved["imported"] != true {
		t.Fatalf("decision-save: %d %s", rec.Code, rec.Body.String())
	}
	post("undo-files", saved["undo"])

	if out := post("oob-take", map[string]any{"tag": "GER", "date": "1936.1.1.12"}); out["imported"] != true {
		t.Errorf("oob-take = %v", out)
	}
	if rows := rows(t, p, divisionTemplateTable); len(rows) < 2 {
		t.Errorf("Germany's templates taken in: %d", len(rows))
	}
	templates := index.DivisionTemplatesOf("GER", "1936.1.1.12")
	if len(templates) == 0 {
		t.Fatal("Germany has no templates")
	}
	post("designer-save", map[string]any{"tag": "GER", "date": "1936.1.1.12", "template": templates[0].Name, "name": templates[0].Name,
		"priority": 1, "regiments": []any{map[string]any{"type": "infantry", "x": 0, "y": 0}}, "support": []any{}})
	if out := post("designer-save", map[string]any{"tag": "GER", "date": "1936.1.1.12", "template": "AOK", "name": "AOK",
		"regiments": []any{map[string]any{"type": "hq_light_armor", "x": 0, "y": 0}}}); out["oob_file"] != hqFile {
		t.Errorf("Army HQ save = %v", out)
	}
	if out := post("mio-take", map[string]any{"id": "GER_porsche_organization"}); out["imported"] != 25.0 {
		t.Errorf("mio-take = %v", out)
	}
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	t.Logf("heap after: %.0f MB; raw files %d", float64(stats.HeapAlloc)/(1<<20), len(rawFiles(t, p)))
}
