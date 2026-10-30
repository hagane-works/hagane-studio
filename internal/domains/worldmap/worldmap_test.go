package worldmap

import (
	"bytes"
	"errors"
	"image/color"
	"image/png"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
)

func TestWorldMapPayload(t *testing.T) {
	env := newTestEnv(t)
	out := want(t, env.get("?date=1936.1.1"), http.StatusOK)
	provinces, _ := out["provinces"].([]any)
	states, _ := out["states"].([]any)
	if len(provinces) != 4 || len(states) != 2 {
		t.Fatalf("provinces %v states %v", out["provinces"], out["states"])
	}
	for _, row := range provinces {
		cells := row.([]any)
		if intOf(cells[0]) == 2 && (intOf(cells[6]) != 1 || intOf(cells[7]) != 1) {
			t.Errorf("province 2 = %v, want state 1 region 1", cells)
		}
	}
	want(t, env.get("?date=1936-01-01"), http.StatusUnprocessableEntity)
	legend := want(t, env.get("/legend"), http.StatusOK)
	if items, _ := legend["legend"].([]any); len(items) != len(mapgen.Legend)-1 {
		t.Errorf("legend %v", legend)
	}

	want(t, env.get("/nothing"), http.StatusNotFound)
	want(t, env.do(http.MethodPut, "/land", nil, ""), http.StatusMethodNotAllowed)
}

func TestProvincesPNGIsTheGamesBitmap(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get("/provinces.png")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("%d %s", rec.Code, rec.Header())
	}
	picture, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range [][2]int{{0, 0}, {5, 20}, {20, 5}, {50, 30}} {
		r, g, b, _ := picture.At(at[0], at[1]).RGBA()
		want := testColours[testProvinceAt(at[0], at[1])]
		if uint8(r>>8) != want.R || uint8(g>>8) != want.G || uint8(b>>8) != want.B {
			t.Errorf("%v: %d %d %d, want %v", at, r>>8, g>>8, b>>8, want)
		}
	}
	tag := rec.Header().Get("ETag")
	again := httpGetWith(env, "/provinces.png", "If-None-Match", tag)
	if tag == "" || again.Code != http.StatusNotModified {
		t.Fatalf("revalidation: tag %q → %d", tag, again.Code)
	}
}

func httpGetWith(env *testEnv, path, header, value string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, testPrefix+path, nil)
	req.Header.Set(header, value)
	rec := httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	return rec
}

func TestTerrainEditAndUndo(t *testing.T) {
	env := newTestEnv(t)
	out := want(t, env.postJSON("/terrain", `{"provinces":{"1":"forest"}}`), http.StatusOK)
	if out["changed"] != 1.0 || out["look"] != false {
		t.Fatalf("terrain %v", out)
	}
	row := env.rawFile("map/definition.csv")
	if row == nil || !strings.Contains(str(row["content"]), "1;10;20;30;land;false;forest;1") {
		t.Fatalf("definition.csv %v", row)
	}
	land := want(t, env.get("/land"), http.StatusOK)
	if additions, _ := land["additions"].([]any); len(additions) != 1 {
		t.Fatalf("records %v", land)
	}
	changes, _ := land["map_changes"].(map[string]any)
	if changes["count"] != 1.0 || !strings.HasPrefix(str(changes["confirmation"]), "restore-1-") {
		t.Fatalf("map_changes %v", land["map_changes"])
	}

	water := want(t, env.postJSON("/terrain", `{"provinces":{"4":"forest"}}`), http.StatusUnprocessableEntity)
	if water["code"] != problemWaterTerrain {
		t.Errorf("water terrain %v", water)
	}
	want(t, env.postJSON("/terrain", `{"provinces":{"1":"lava"}}`), http.StatusUnprocessableEntity)

	undone := want(t, env.do(http.MethodDelete, "/land?latest=1", nil, ""), http.StatusOK)
	if u, _ := undone["undone"].(map[string]any); u["kind"] != "edit" {
		t.Fatalf("undone %v", undone)
	}
	if env.rawFile("map/definition.csv") != nil || len(env.rows(additionsTable)) != 0 {
		t.Fatal("undo left the edit behind")
	}
	nothing := want(t, env.do(http.MethodDelete, "/land?latest=1", nil, ""), http.StatusNotFound)
	if nothing["code"] != problemNothingToUndo {
		t.Errorf("nothing to undo: %v", nothing)
	}
}

func TestRestatePaintsAProvinceOver(t *testing.T) {
	env := newTestEnv(t)
	out := want(t, env.postJSON("/restate", `{"provinces":{"2":"OTH"}}`), http.StatusOK)
	if out["changed"] == 0.0 {
		t.Fatalf("restate %v", out)
	}
	one, two := env.stateRow(1), env.stateRow(2)
	if one == nil || two == nil {
		t.Fatalf("states not seeded: %v", env.rows(statesTable))
	}
	if got := intsOf(one["provinces"]); len(got) != 1 || got[0] != 1 {
		t.Errorf("state 1 provinces %v", got)
	}
	if got := intsOf(two["provinces"]); len(got) != 2 {
		t.Errorf("state 2 provinces %v", got)
	}
	if str(one["source"]) != "created" || str(two["owner"]) != "OTH" {
		t.Errorf("rows %v %v", one, two)
	}

	unknown := want(t, env.postJSON("/restate", `{"provinces":{"1":"ZZZ"}}`), http.StatusUnprocessableEntity)
	if unknown["code"] != problemNoCountry {
		t.Errorf("unknown country %v", unknown)
	}
	want(t, env.do(http.MethodDelete, "/land?latest=1", nil, ""), http.StatusOK)
	if len(env.rows(statesTable)) != 0 {
		t.Fatalf("undo left seeded states: %v", env.rows(statesTable))
	}
}

func TestFloodKeepsAStateAProvince(t *testing.T) {
	env := newTestEnv(t)
	drowned := want(t, env.postJSON("/flood", `{"provinces":[3]}`), http.StatusUnprocessableEntity)
	if drowned["code"] != problemStateDrowned {
		t.Fatalf("whole state %v", drowned)
	}
	out := want(t, env.postJSON("/flood", `{"provinces":[1]}`), http.StatusOK)
	if out["changed"] != 1.0 || out["states"] != 1.0 {
		t.Fatalf("flood %v", out)
	}
	if got := intsOf(env.stateRow(1)["provinces"]); len(got) != 1 || got[0] != 2 {
		t.Errorf("state 1 keeps %v", got)
	}
	definition := env.rawFile("map/definition.csv")
	if definition == nil || !strings.Contains(str(definition["content"]), "1;10;20;30;lake;false;lakes;") {
		t.Fatalf("definition.csv %v", definition)
	}
	for _, picture := range []string{"map/terrain.bmp", "map/heightmap.bmp", "map/rivers.bmp"} {
		row := env.rawFile(picture)
		if row == nil || row["is_binary"] != true || !env.p.Blobs().Exists(str(row["content_obs_key"])) {
			t.Errorf("%s not stored: %v", picture, row)
		}
	}
	already := want(t, env.postJSON("/flood", `{"provinces":[4]}`), http.StatusUnprocessableEntity)
	if already["code"] != "already_water" {
		t.Errorf("water %v", already)
	}

	restored := want(t, env.do(http.MethodDelete, "/land", nil, ""), http.StatusOK)
	if restored["restored"] != 1.0 || env.rawFile("map/definition.csv") != nil || len(env.rows(statesTable)) != 0 {
		t.Fatalf("restore %v, raw %v, states %v", restored, env.rows(rawFilesTable), env.rows(statesTable))
	}
}

func TestProvinceEdit(t *testing.T) {
	env := newTestEnv(t)
	out := want(t, env.postJSON("/province", `{"province":2,"state":2,"terrain":"hills"}`), http.StatusOK)
	changed, _ := out["changed"].([]any)
	if len(changed) != 2 {
		t.Fatalf("edit %v", out)
	}
	if got := intsOf(env.stateRow(2)["provinces"]); len(got) != 2 || got[1] != 2 {
		t.Errorf("state 2 %v", got)
	}

	last := want(t, env.postJSON("/province", `{"province":1,"state":2}`), http.StatusUnprocessableEntity)
	if last["code"] != problemStateLastProvince {
		t.Errorf("last province %v", last)
	}
	sea := want(t, env.postJSON("/province", `{"province":4,"state":1}`), http.StatusUnprocessableEntity)
	if sea["code"] != problemStateLandOnly {
		t.Errorf("sea %v", sea)
	}
	want(t, env.postJSON("/province", `{}`), http.StatusUnprocessableEntity)
	want(t, env.do(http.MethodDelete, "/land?latest=1", nil, ""), http.StatusOK)
	if len(env.rows(statesTable)) != 0 || env.rawFile("map/definition.csv") != nil {
		t.Fatalf("undo left %v %v", env.rows(statesTable), env.rows(rawFilesTable))
	}
}

func TestProvinceEditChangesBetweenLandAndWater(t *testing.T) {
	env := newTestEnv(t)
	sunk := want(t, env.postJSON("/province", `{"province":2,"kind":"lake"}`), http.StatusOK)
	if sunk["changed"] != 1.0 {
		t.Fatalf("sink = %v", sunk)
	}
	if got := intsOf(env.stateRow(1)["provinces"]); len(got) != 1 || got[0] != 1 {
		t.Fatalf("state after lake = %v", got)
	}
	raised := want(t, env.postJSON("/province", `{"province":2,"kind":"land","state":2,"region":1,"terrain":"hills","continent":1}`), http.StatusOK)
	changed, _ := raised["changed"].([]any)
	if len(changed) < 5 {
		t.Fatalf("raise = %v", raised)
	}
	if got := intsOf(env.stateRow(2)["provinces"]); len(got) != 2 || got[1] != 2 {
		t.Fatalf("state after raise = %v", got)
	}
	definition := env.rawFile("map/definition.csv")
	if definition == nil || !strings.Contains(str(definition["content"]), "2;40;50;60;land;true;hills;1") {
		t.Fatalf("definition after raise = %v", definition)
	}
	for _, picture := range []string{"map/terrain.bmp", "map/heightmap.bmp", "map/rivers.bmp"} {
		if env.rawFile(picture) == nil {
			t.Fatalf("%s not saved", picture)
		}
	}
}

func TestRivers(t *testing.T) {
	env := newTestEnv(t)
	read := want(t, env.get("/rivers"), http.StatusOK)
	if read["width"] != float64(testW) || len(read["at"].([]any)) != 0 {
		t.Fatalf("rivers %v", read)
	}

	at := func(x, y int) int { return y*testW + x }
	body := `{"pixels":[[` + itoa(at(4, 4)) + `,0],[` + itoa(at(4, 5)) + `,3],[` + itoa(at(4, 6)) + `,3]]}`
	rec := env.postJSON("/rivers", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("rivers: %d %s", rec.Code, rec.Body.String())
	}
	row := env.rawFile("map/rivers.bmp")
	if row == nil || row["is_binary"] != true {
		t.Fatalf("rivers.bmp %v", row)
	}
	again := want(t, env.get("/rivers"), http.StatusOK)
	if len(again["at"].([]any)) == 0 {
		t.Errorf("drawn rivers not read back: %v", again)
	}
	want(t, env.postJSON("/rivers", `{"pixels":[]}`), http.StatusUnprocessableEntity)
}

func TestNewLand(t *testing.T) {
	env := newTestEnv(t)
	sketch := pngOf(t, 12, 12, color.RGBA{160, 210, 100, 255})
	fields := map[string]string{"x": "40", "y": "8", "width": "14", "height": "14", "seed": "1", "land": "60", "name": "Atlantis", "owner": "TST"}
	preview := want(t, env.postForm("/land-preview", map[string][]byte{"sketch": sketch}, fields), http.StatusOK)
	summary, _ := preview["summary"].(map[string]any)
	overlay, _ := preview["overlay"].(map[string]any)
	if summary["land"] == 0.0 || !strings.HasPrefix(str(overlay["provinces"]), "data:image/png;base64,") {
		t.Fatalf("preview %v", preview)
	}
	if len(env.rows(additionsTable)) != 0 {
		t.Fatal("a preview saved something")
	}
	tagBefore := env.get("/provinces.png").Header().Get("ETag")

	saved := want(t, env.postForm("/land", map[string][]byte{"sketch": sketch}, fields), http.StatusCreated)
	addition, _ := saved["addition"].(map[string]any)
	stateIDs := intsOf(addition["state_ids"])
	if len(stateIDs) == 0 || stateIDs[0] != 3 || addition["region_id"] != 3.0 || addition["name"] != "Atlantis" {
		t.Fatalf("addition %v", addition)
	}
	if row := env.stateRow(3); row == nil || str(row["owner"]) != "TST" {
		t.Fatalf("new state %v", row)
	}
	if env.rawFile("map/strategicregions/3-Atlantis.txt") == nil || env.rawFile("map/provinces.bmp") == nil {
		t.Fatalf("files %v", env.rows(rawFilesTable))
	}
	texts := 0
	for _, row := range env.rows(locTable) {
		if str(row["key"]) == "STRATEGICREGION_3" && str(row["value"]) == "Atlantis" {
			texts++
		}
	}
	if texts != 1 {
		t.Errorf("region name texts: %d", texts)
	}
	rec := env.get("/provinces.png")
	if tag := rec.Header().Get("ETag"); rec.Code != http.StatusOK || tag == tagBefore || !strings.HasPrefix(tag, `"p`) {
		t.Fatalf("the project's own picture: %d %q (was %q)", rec.Code, tag, tagBefore)
	}

	payload := want(t, env.get("?date=1936.1.1"), http.StatusOK)
	if provinces, _ := payload["provinces"].([]any); len(provinces) <= 4 {
		t.Errorf("provinces after new land: %d", len(provinces))
	}
	check := want(t, env.get("/check"), http.StatusOK)
	if _, ok := check["ok"].(bool); !ok {
		t.Errorf("check %v", check)
	}

	fields["owner"] = "ZZZ"
	want(t, env.postForm("/land", map[string][]byte{"sketch": sketch}, fields), http.StatusUnprocessableEntity)

	restored := want(t, env.do(http.MethodDelete, "/land", nil, ""), http.StatusOK)
	if restored["restored"] != 1.0 || len(env.rows(rawFilesTable)) != 0 || len(env.rows(statesTable)) != 0 {
		t.Fatalf("restore %v: raw %d states %d", restored, len(env.rows(rawFilesTable)), len(env.rows(statesTable)))
	}
	for _, row := range env.rows(locTable) {
		if str(row["key"]) == "STRATEGICREGION_3" {
			t.Fatal("the region name outlived the map")
		}
	}
	if tag := env.get("/provinces.png").Header().Get("ETag"); tag != tagBefore {
		t.Errorf("after restore the picture is %q, want the game's %q", tag, tagBefore)
	}
}

func TestConfirmedRestore(t *testing.T) {
	env := newTestEnv(t)
	want(t, env.postJSON("/terrain", `{"provinces":{"1":"forest"}}`), http.StatusOK)
	changes := want(t, env.get("/land"), http.StatusOK)["map_changes"].(map[string]any)
	confirmation := str(changes["confirmation"])
	want(t, env.postJSON("/terrain", `{"provinces":{"2":"hills"}}`), http.StatusOK)
	moved := want(t, env.do(http.MethodDelete, "/land?confirm="+confirmation, nil, ""), http.StatusConflict)
	if moved["code"] != problemMapChangesMoved || len(env.rows(additionsTable)) != 2 {
		t.Fatalf("moved %v", moved)
	}
	latest := want(t, env.get("/land"), http.StatusOK)["map_changes"].(map[string]any)
	out := want(t, env.do(http.MethodDelete, "/land?confirm="+str(latest["confirmation"]), nil, ""), http.StatusOK)
	if out["restored"] != 2.0 || len(out["removed"].([]any)) != 2 || len(env.rows(additionsTable)) != 0 {
		t.Fatalf("restore %v", out)
	}

	again := want(t, env.do(http.MethodDelete, "/land?confirm=restore-2-9", nil, ""), http.StatusOK)
	if again["restored"] != 0.0 {
		t.Fatalf("again %v", again)
	}
}

func TestSketchPreview(t *testing.T) {
	env := newTestEnv(t)
	sketch := pngOf(t, 40, 20, color.RGBA{40, 80, 170, 255})
	island := mustDecodePNG(t, sketch)
	for y := 5; y < 15; y++ {
		for x := 10; x < 30; x++ {
			island.Set(x, y, color.RGBA{160, 210, 100, 255})
		}
	}
	out := want(t, env.postForm("/sketch-preview", map[string][]byte{"sketch": encodePNG(t, island)}, map[string]string{"width": "512", "height": "256", "seed": "3"}), http.StatusOK)
	provinces, _ := out["provinces"].(map[string]any)
	if provinces["land"] == 0.0 || provinces["sea"] == 0.0 {
		t.Fatalf("sketch %v", out["provinces"])
	}
	want(t, env.postForm("/sketch-preview", map[string][]byte{"sketch": []byte("no")}, nil), http.StatusUnprocessableEntity)
	want(t, env.postForm("/sketch-preview", map[string][]byte{"sketch": sketch}, map[string]string{"width": "8192", "height": "4096"}), http.StatusUnprocessableEntity)
}

func TestRepairOnAWholeMap(t *testing.T) {
	env := newTestEnv(t)
	out := want(t, env.postJSON("/repair", `{}`), http.StatusUnprocessableEntity)
	if out["code"] != problemNothing {
		t.Fatalf("repair %v", out)
	}
	check := want(t, env.get("/check"), http.StatusOK)
	if check["ok"] != true {
		t.Fatalf("check %v", check)
	}
}

func TestNewWorldFromTheGame(t *testing.T) {
	env := newTestEnv(t)

	if err := (&mapStore{p: env.p, game: func() (fs.FS, error) { return testGame(t), nil }}).SeedWorldMapStates([]int{2}); err != nil {
		t.Fatal(err)
	}
	ownRow := env.stateRow(2)
	if ownRow == nil {
		t.Fatal("state 2 not seeded")
	}
	var history project.Row
	if err := env.p.Write([]string{historyTable}, func(tx *project.Tx) error {
		history = tx.Insert(historyTable, project.Row{"country_tag": "TST", "capital": int64(1), "source": "imported"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	fields := map[string]string{"source": "game", "tags": "TST,OTH", "per_state": "1", "languages": "en"}
	preview := want(t, env.postForm("/world-preview", nil, fields), http.StatusOK)
	layout, _ := preview["state_layout"].([]any)
	if len(layout) == 0 {
		t.Fatalf("preview %v", truncate(str(preview["summary"])))
	}
	saved := want(t, env.postForm("/world", nil, fields), http.StatusCreated)
	addition, _ := saved["addition"].(map[string]any)
	if addition["kind"] != "world" || len(intsOf(addition["state_ids"])) != len(layout) {
		t.Fatalf("world %v", addition)
	}
	if len(env.rows(statesTable)) != len(layout) || env.stateRow(1) == nil {
		t.Fatalf("states %v", env.rows(statesTable))
	}

	changed := want(t, env.postForm("/world", nil, fields), http.StatusConflict)
	if changed["code"] != problemMapChanged {
		t.Fatalf("second world %v", changed)
	}

	var items []string
	for _, row := range layout {
		id := intOf(row.(map[string]any)["state_id"])
		items = append(items, `{"state_id":`+itoa(id)+`,"display_name_locs":{"en":"Land `+itoa(id)+`"},"capital_name_locs":{"en":"Town of `+itoa(id)+`"}}`)
	}
	named := want(t, env.postJSON("/names", `{"state_names":[`+strings.Join(items, ",")+`]}`), http.StatusOK)
	status, _ := named["name_status"].(map[string]any)
	if status["complete"] != true {
		t.Fatalf("names %v", named)
	}
	if locs, _ := env.stateRow(1)["display_name_locs"].(map[string]any); locs["en"] != "Land 1" {
		t.Errorf("state 1 named %v", locs)
	}

	bad := want(t, env.postJSON("/names", `{"state_names":[{"state_id":1,"display_name_locs":{"en":"State 1"}}]}`), http.StatusUnprocessableEntity)
	if !strings.Contains(str(bad["detail"]), "placeholder") {
		t.Errorf("placeholder %v", bad)
	}

	if row := env.rows(historyTable)[0]; str(row["source"]) != "created" {
		t.Errorf("TST's start was not moved: %v", row)
	}
	want(t, env.do(http.MethodDelete, "/land", nil, ""), http.StatusOK)
	if rows := env.rows(statesTable); len(rows) != 1 || project.Int64(rows[0]["id"]) != project.Int64(ownRow["id"]) ||
		str(rows[0]["source"]) != "imported" || len(env.rows(rawFilesTable)) != 0 {
		t.Fatalf("restore: states %v raw %d", rows, len(env.rows(rawFilesTable)))
	}
	if row := env.rows(historyTable)[0]; project.Int64(row["id"]) != project.Int64(history["id"]) || intOf(row["capital"]) != 1 || str(row["source"]) != "imported" {
		t.Errorf("TST's start not put back: %v", row)
	}
	for _, row := range env.rows(locTable) {
		if strings.HasPrefix(str(row["key"]), "VICTORY_POINTS_") {
			t.Fatalf("a capital name outlived the world: %v", row)
		}
	}

	want(t, env.postJSON("/names", `{"state_names":[{"state_id":1,"display_name_locs":{"en":"Land"}}]}`), http.StatusConflict)
}

func TestWithoutAGame(t *testing.T) {
	env := newTestEnv(t)
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return env.p, nil },
		Game: func() (fs.FS, error) { return nil, errors.New("not set up") },
	})
	env.mux = mux
	if out := want(t, env.get("?date=1936.1.1"), http.StatusOK); len(out["provinces"].([]any)) != 0 {
		t.Errorf("map without a game: %v", out)
	}
	want(t, env.get("/provinces.png"), http.StatusServiceUnavailable)
	for _, path := range []string{"/rivers", "/check"} {
		if out := want(t, env.get(path), http.StatusServiceUnavailable); out["code"] != problemUnreadable {
			t.Errorf("%s: %v", path, out)
		}
	}
	want(t, env.postJSON("/terrain", `{"provinces":{"1":"forest"}}`), http.StatusServiceUnavailable)
	want(t, env.postForm("/world-preview", nil, map[string]string{"source": "game", "tags": "TST"}), http.StatusServiceUnavailable)
	if out := want(t, env.get("/land"), http.StatusOK); out["map_changes"] != nil {
		t.Errorf("land %v", out)
	}
}
