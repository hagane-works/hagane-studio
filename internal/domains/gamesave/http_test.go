package gamesave

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

var gameFiles = map[string]string{
	"history/countries/GER - Germany.txt":    "capital = 64\nset_oob = \"GER_1936\"\n",
	"history/units/GER_1936.txt":             germanOOB,
	hqFile:                                   hqFixture,
	"common/decisions/political_actions.txt": decisionFileText,
	"common/units/equipment/infantry.txt":    infantryText,
	"common/wargoals/00_invasion.txt":        wargoalFile,
}

func TestSaveRoutes(t *testing.T) {
	p := newProject(t)
	s := newServer(t, p, index(t, gameFiles))

	out := s.post("decision-save", map[string]any{"category": "political_actions", "name": "improved_worker_conditions", "cost": 75}, http.StatusOK)
	if out["decision"] != "improved_worker_conditions" || out["file"] != "common/decisions/political_actions.txt" || out["imported"] != true {
		t.Fatalf("decision-save = %v", out)
	}
	files, _ := out["undo"].(map[string]any)["files"].([]any)
	step, _ := files[0].(map[string]any)
	if len(files) != 1 || step["before"] != nil || step["after_sha256"] == "" || step["file"] != "common/decisions/political_actions.txt" {
		t.Fatalf("undo = %v", out["undo"])
	}
	if s.forgotten != 1 || len(rawFiles(t, p)) != 1 {
		t.Fatalf("forgotten=%d files=%d", s.forgotten, len(rawFiles(t, p)))
	}
	if undone := s.post("undo-files", map[string]any{"files": files}, http.StatusOK); undone["undone"] != 1.0 || len(rawFiles(t, p)) != 0 {
		t.Fatalf("undo-files = %v, files %d", undone, len(rawFiles(t, p)))
	}
	if got := s.post("undo-files", map[string]any{"files": []any{}}, http.StatusUnprocessableEntity); got["detail"] != "nothing to undo" {
		t.Errorf("empty undo = %v", got)
	}
	if got := s.post("decision-save", []byte("{"), http.StatusUnprocessableEntity); got["detail"] != "invalid decision payload" {
		t.Errorf("bad body = %v", got)
	}
	if got := s.post("decision-save", map[string]any{"category": "political_actions", "name": "nope", "cost": 1}, http.StatusNotFound); got["detail"] != "the game files do not declare this decision" {
		t.Errorf("unknown decision = %v", got)
	}

	out = s.post("equipment-save", map[string]any{"name": "infantry_equipment_1", "values": map[string]any{"soft_attack": 7, "breakthrough": nil}}, http.StatusOK)
	if out["name"] != "infantry_equipment_1" || out["file"] != "common/units/equipment/infantry.txt" || out["imported"] != true || out["undo"] == nil {
		t.Fatalf("equipment-save = %v", out)
	}
	if got := s.post("equipment-save", map[string]any{"name": "infantry_equipment_1"}, http.StatusUnprocessableEntity); got["detail"] != "nothing to change" {
		t.Errorf("empty change = %v", got)
	}

	if out := s.post("wargoal-save", map[string]any{"name": "take_state", "values": map[string]any{"threat": 3}}, http.StatusOK); out["imported"] != true {
		t.Errorf("wargoal-save = %v", out)
	}

	if got := s.post("oob-take", map[string]any{"date": "1936.1.1"}, http.StatusUnprocessableEntity); got["detail"] != "tag is required" {
		t.Errorf("oob-take without a tag = %v", got)
	}
	if got := s.post("oob-take", map[string]any{"tag": "ENG"}, http.StatusUnprocessableEntity); got["detail"] != "the country has no battle order file" {
		t.Errorf("oob-take of a country without one = %v", got)
	}
	if out := s.post("oob-take", map[string]any{"tag": "ger", "date": "1936.1.1"}, http.StatusOK); out["oob_file"] != "GER_1936" || out["imported"] != true {
		t.Fatalf("oob-take = %v", out)
	}
	if out := s.post("oob-take", map[string]any{"tag": "GER"}, http.StatusOK); out["imported"] != false {
		t.Errorf("second oob-take = %v", out)
	}
	out = s.post("designer-save", map[string]any{"tag": "GER", "date": "1936.1.1", "template": "Panzer-Division", "name": "Leichte-Division",
		"names_group": "", "priority": 1, "regiments": []any{map[string]any{"type": "light_armor", "x": 0, "y": 0}}, "support": []any{}}, http.StatusOK)
	template, _ := out["template"].(map[string]any)
	if out["oob_file"] != "GER_1936" || out["imported"] != false || template["template_name"] != "Leichte-Division" || template["id"] == nil {
		t.Fatalf("designer-save = %v", out)
	}
	if got := s.post("designer-save", map[string]any{"tag": "GER", "name": "Infanterie-Division",
		"regiments": []any{map[string]any{"type": "infantry", "x": 0, "y": 0}}}, http.StatusConflict); got["detail"] != "a template with this name already exists" {
		t.Errorf("duplicate = %v", got)
	}

	out = s.post("designer-save", map[string]any{"tag": "GER", "date": "1936.1.1", "template": "AOK", "name": "AOK",
		"regiments": []any{map[string]any{"type": "hq_light_armor", "x": 0, "y": 0}}}, http.StatusOK)
	if hq, _ := out["template"].(map[string]any); hq["is_army_hq"] != true || out["oob_file"] != hqFile || out["imported"] != true {
		t.Fatalf("Army HQ save = %v", out)
	}

	if out := s.post("division-save", map[string]any{"oob": "GER_1936", "ordinal": 0, "experience": 0.5}, http.StatusOK); out["oob"] != "GER_1936" || out["row"] == nil {
		t.Errorf("division-save = %v", out)
	}
	if got := s.post("division-save", map[string]any{"oob": "GER_1936", "ordinal": 0}, http.StatusUnprocessableEntity); got["detail"] != "nothing to change" {
		t.Errorf("empty division-save = %v", got)
	}

	if got := s.post("equipment-design-save", map[string]any{"tag": "GER", "original": "Panzer II", "delete": true}, http.StatusConflict); got["detail"] != notInProjectHistory {
		t.Errorf("design delete = %v", got)
	}
	if out := s.post("intel-agency-create", map[string]any{"tag": "GER", "picture": "GFX_intelligence_agency_logo_ger", "name": "Abwehr"}, http.StatusOK); out["agency_id"] != "ger_abwehr" {
		t.Errorf("intel-agency-create = %v", out)
	}
	if got := s.post("mio-take", map[string]any{"id": " "}, http.StatusUnprocessableEntity); got["detail"] != "id is required" {
		t.Errorf("mio-take = %v", got)
	}
	if got := s.post("mio-take", map[string]any{"id": "GER_porsche_organization"}, http.StatusUnprocessableEntity); got["detail"] != "no file declares this organization" {
		t.Errorf("mio-take of an unknown company = %v", got)
	}
	for route, body := range map[string]any{
		"autonomy-save":        map[string]any{"id": "autonomy_colony", "values": map[string]any{"min_freedom_level": 0.5}},
		"special-project-save": map[string]any{"id": "sp_x", "complexity": "sp_complexity.large"},
		"intel-upgrade-save":   map[string]any{"id": "upgrade_x", "factories": 3},
		"intel-agency-save":    map[string]any{"file": intelPath, "ordinal": 0, "picture": "GFX_x", "default": "tag = GER"},
		"sub-unit-save":        map[string]any{"id": "infantry", "stats": map[string]any{"combat_width": 3}},
	} {
		s.post(route, body, http.StatusNotFound)
	}
	if got := s.post("oob-unit-save", map[string]any{"kind": "task_force", "oob": "GER_1936_naval", "name": "X"}, http.StatusNotFound); got["detail"] != "the battle order file's text is not available" {
		t.Errorf("oob-unit-save = %v", got)
	}
}

func TestSaveRoutesWithoutAnIndex(t *testing.T) {
	p := newProject(t)
	s := newServer(t, p, nil)
	for _, route := range []string{"decision-save", "equipment-save", "wargoal-save", "designer-save", "division-save",
		"autonomy-save", "special-project-save", "intel-upgrade-save", "intel-agency-save", "oob-unit-save", "sub-unit-save"} {
		if got := s.post(route, map[string]any{}, http.StatusServiceUnavailable); got["detail"] != "GUI source snapshot unavailable" {
			t.Errorf("%s = %v", route, got)
		}
	}
	s.post("oob-take", map[string]any{"tag": "GER"}, http.StatusServiceUnavailable)
	s.post("mio-take", map[string]any{"id": "x"}, http.StatusServiceUnavailable)
	s.post("equipment-design-save", map[string]any{"tag": "GER", "name": "X", "type": "t"}, http.StatusServiceUnavailable)
	if s.forgotten != 0 {
		t.Errorf("forgotten %d times", s.forgotten)
	}
}

func TestSaveThroughTheGameView(t *testing.T) {
	game := fstest.MapFS{}
	for name, text := range gameFiles {
		game[name] = &fstest.MapFile{Data: []byte(text)}
	}
	game["launcher-settings.json"] = &fstest.MapFile{Data: []byte(`{"rawVersion":"1.2.3.0"}`)}
	p := newProject(t)
	deps := domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return game, nil },
	}
	svc := gameview.New(deps)
	mux := http.NewServeMux()
	gameview.RegisterWith(mux, svc)
	RegisterWith(mux, deps, svc)

	post := func(route, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/go/projects/1/gui-render/"+route, bytes.NewReader([]byte(body))))
		return rec
	}
	rec := post("equipment-save", `{"name":"infantry_equipment_1","values":{"soft_attack":7}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"imported":true`) {
		t.Fatalf("equipment-save: %d %s", rec.Code, rec.Body.String())
	}
	index, found, err := svc.Index(context.Background(), p)
	if err != nil || !found {
		t.Fatalf("index: %v %v", found, err)
	}
	file, body, _, ok := index.EquipmentSource("infantry_equipment_1")
	if !ok || file != "common/units/equipment/infantry.txt" || !strings.Contains(string(body), "soft_attack = 7") {
		t.Fatalf("the game view reads %q: %s", file, body)
	}

	if rec := post("equipment-save", `{"name":"infantry_equipment_1","values":{"defense":30}}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"imported":false`) {
		t.Fatalf("second save: %d %s", rec.Code, rec.Body.String())
	}
	files := rawFiles(t, p)
	if len(files) != 1 || !strings.Contains(files[0].Content, "soft_attack = 7") || !strings.Contains(files[0].Content, "defense = 30") {
		t.Fatalf("raw files = %+v", files)
	}

	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/go/projects/1/gui-render/status", nil))
	if get.Code != http.StatusOK {
		t.Errorf("status: %d %s", get.Code, get.Body.String())
	}

	if rec := post("oob-take", `{"tag":"GER","date":"1936.1.1"}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"oob_file":"GER_1936"`) {
		t.Errorf("oob-take: %d %s", rec.Code, rec.Body.String())
	}
}
