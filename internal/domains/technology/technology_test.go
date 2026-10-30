package technology

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/project"
)

func newTestProject(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newMux(p *project.Project, game fs.FS) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) {
			if game == nil {
				return nil, os.ErrNotExist
			}
			return game, nil
		},
	})
	return mux
}

func call(t *testing.T, mux http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	switch value := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(value))
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, reader))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

func expect(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status=%d want %d body=%s", rec.Code, status, rec.Body.String())
	}
}

const base = "/api/go/projects/1/technologies"

func TestCreateListGetUpdateDelete(t *testing.T) {
	p := newTestProject(t)
	mux := newMux(p, nil)

	created := call(t, mux, "POST", base, map[string]any{"name": "Better Rifles", "tech_line": "armor", "research_cost": 2.5})
	expect(t, created, http.StatusCreated)
	tech := decode[Technology](t, created)
	if tech.TechID != "armor_better_rifles" || tech.Folder != "armour_folder" || tech.StartYear != 1936 ||
		tech.ResearchCost != 2.5 || tech.NameEn != "Better Rifles" || tech.Source != "created" || tech.AIWillDo != "factor = 1" {
		t.Fatalf("created=%+v", tech)
	}
	var shape map[string]any
	_ = json.Unmarshal(created.Body.Bytes(), &shape)
	for _, key := range []string{"folder_placements", "prerequisites", "warnings", "display_name_locs", "path", "xp_research_bonus", "position_y_is_explicit"} {
		if _, ok := shape[key]; !ok {
			t.Fatalf("response lacks %s: %s", key, created.Body.String())
		}
	}
	if shape["folder_placements"] != nil || shape["position_y_is_explicit"] != false {
		t.Fatalf("defaults: %s", created.Body.String())
	}

	again := decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "armor_better_rifles"}))
	if again.TechID != "armor_better_rifles_1" || again.Name != "New Technology" || again.TechLine != "infantry" {
		t.Fatalf("suffix=%+v", again)
	}

	other := decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "GER_tanks", "tech_line": "armor", "start_year": 1934}))

	list := decode[[]Technology](t, call(t, mux, "GET", base+"?tech_line=armor", nil))
	if len(list) != 2 || list[0].TechID != "GER_tanks" || list[1].TechID != "armor_better_rifles" {
		t.Fatalf("list by line=%+v", list)
	}
	visible := decode[[]Technology](t, call(t, mux, "GET", base+"?country_tag=USA", nil))
	if len(visible) != 2 {
		t.Fatalf("country filter must hide GER_tanks: %+v", visible)
	}

	got := call(t, mux, "GET", base+"/"+itoa(other.ID), nil)
	expect(t, got, http.StatusOK)
	expect(t, call(t, mux, "GET", base+"/999", nil), http.StatusNotFound)
	expect(t, call(t, mux, "GET", base+"/abc", nil), http.StatusNotFound)

	updated := call(t, mux, "PUT", base+"/"+itoa(tech.ID), map[string]any{
		"tech_id": "  renamed_tech ", "folder_placements": []any{map[string]any{"folder": "", "x": 3, "y": 4}},
		"modifiers": map[string]any{"research_speed_factor": 0.05},
	})
	expect(t, updated, http.StatusOK)
	tech = decode[Technology](t, updated)
	if tech.TechID != "renamed_tech" || tech.Folder != "armour_folder" || tech.PositionX != 3 || tech.PositionY != 4 ||
		tech.Modifiers["research_speed_factor"] != 0.05 || len(tech.FolderPlacements) != 1 || tech.FolderPlacements[0]["folder"] != "armour_folder" {
		t.Fatalf("updated=%+v", tech)
	}

	tech = decode[Technology](t, call(t, mux, "PUT", base+"/"+itoa(tech.ID), map[string]any{"position_x": 7}))
	if tech.PositionX != 7 || tech.FolderPlacements[0]["x"] != 7.0 {
		t.Fatalf("legacy move=%+v", tech)
	}

	expect(t, call(t, mux, "PUT", base+"/"+itoa(tech.ID), map[string]any{"name": nil}), http.StatusServiceUnavailable)
	expect(t, call(t, mux, "PUT", base+"/"+itoa(tech.ID), map[string]any{"tech_id": "GER_tanks"}), http.StatusServiceUnavailable)
	expect(t, call(t, mux, "PUT", base+"/"+itoa(tech.ID), "{"), http.StatusUnprocessableEntity)
	expect(t, call(t, mux, "PUT", base+"/999", map[string]any{"name": "x"}), http.StatusNotFound)

	expect(t, call(t, mux, "DELETE", base+"/"+itoa(tech.ID), nil), http.StatusNoContent)
	expect(t, call(t, mux, "DELETE", base+"/"+itoa(tech.ID), nil), http.StatusNotFound)
	if rows, _ := p.Rows(table); len(rows) != 2 {
		t.Fatalf("rows after delete=%d", len(rows))
	}
}

func TestUpdateFlipsImportedToCreated(t *testing.T) {
	p := newTestProject(t)
	mux := newMux(p, nil)
	tech := decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "x", "source": "imported"}))
	if tech.Source != "imported" {
		t.Fatalf("source=%q", tech.Source)
	}
	tech = decode[Technology](t, call(t, mux, "PUT", base+"/"+itoa(tech.ID), map[string]any{"name": "Edited", "source": "imported"}))
	if tech.Source != "created" || tech.Name != "Edited" {
		t.Fatalf("flip=%+v", tech)
	}
	tech = decode[Technology](t, call(t, mux, "PUT", base+"/"+itoa(tech.ID), map[string]any{"source": "custom"}))
	if tech.Source != "custom" {
		t.Fatalf("explicit source=%q", tech.Source)
	}
}

func TestCreateWarnsOnBaseGameTechID(t *testing.T) {
	p := newTestProject(t)
	game := fstest.MapFS{
		"common/technologies/air.txt": {Data: []byte("technologies = {\n\t@1936 = 0\n\tCAS1 = { research_cost = 1 }\n}\n")},
	}
	mux := newMux(p, game)
	tech := decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "CAS1", "country_tag": "GER"}))
	if len(tech.Warnings) != 1 || !strings.Contains(tech.Warnings[0], "'GER_CAS1'") {
		t.Fatalf("warnings=%v", tech.Warnings)
	}
	tech = decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "my_tech"}))
	if len(tech.Warnings) != 0 {
		t.Fatalf("custom id warned: %v", tech.Warnings)
	}
}

func TestPreviewAndParsePreview(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.UpdateRecord(func(r project.Row) { r["tech_year_columns"] = []any{[]any{"@1936", 7.0}} }); err != nil {
		t.Fatal(err)
	}
	mux := newMux(p, nil)
	tech := decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "infantry_weapons_1", "research_cost": 1.5, "categories": []string{"infantry"}}))

	line := call(t, mux, "GET", base+"/preview/infantry", nil)
	expect(t, line, http.StatusOK)
	if script := decode[map[string]string](t, line)["script"]; !strings.Contains(script, "@1936 = 7") || !strings.Contains(script, "infantry_weapons_1 = {") {
		t.Fatalf("line preview=%s", script)
	}
	single := call(t, mux, "GET", base+"/preview/infantry?tech_id="+itoa(tech.ID), nil)
	expect(t, single, http.StatusOK)
	if script := decode[map[string]string](t, single)["script"]; !strings.Contains(script, "research_cost = 1.5") {
		t.Fatalf("single preview=%s", script)
	}
	expect(t, call(t, mux, "GET", base+"/preview/infantry?tech_id=99", nil), http.StatusNotFound)
	expect(t, call(t, mux, "GET", base+"/preview/infantry?tech_id=x", nil), http.StatusUnprocessableEntity)

	script := "technologies = {\n    parsed_tech = {\n        research_cost = 2.5\n        start_year = 1940\n        folder = { name = infantry_folder position = { x = 3 y = 9 } }\n        ai_will_do = { factor = 1 }\n    }\n}"
	parsed := call(t, mux, "POST", base+"/"+itoa(tech.ID)+"/parse-preview", map[string]any{"script": script})
	expect(t, parsed, http.StatusOK)
	body := parsed.Body.String()
	if !strings.Contains(body, `"tech_id":"parsed_tech"`) || !strings.Contains(body, `"position_y":9`) || !strings.Contains(body, `"parse_errors":[]`) {
		t.Fatalf("parse preview=%s", body)
	}
	empty := call(t, mux, "POST", base+"/"+itoa(tech.ID)+"/parse-preview", map[string]any{"script": "  \n"})
	if !strings.Contains(empty.Body.String(), `"severity":"info"`) {
		t.Fatalf("empty=%s", empty.Body.String())
	}
	nullScript := call(t, mux, "POST", base+"/"+itoa(tech.ID)+"/parse-preview", `{"script":null}`)
	expect(t, nullScript, http.StatusUnprocessableEntity)
	if !strings.Contains(nullScript.Body.String(), `"type":"string_type"`) {
		t.Fatalf("null script=%s", nullScript.Body.String())
	}
	expect(t, call(t, mux, "POST", base+"/abc/parse-preview", map[string]any{"script": "x"}), http.StatusUnprocessableEntity)
	expect(t, call(t, mux, "POST", base+"/99/parse-preview", map[string]any{"script": "x"}), http.StatusNotFound)
}

func TestKnownFolders(t *testing.T) {
	p := newTestProject(t)
	game := fstest.MapFS{
		"common/technology_tags/00_technology.txt": {Data: []byte(`technology_folders = {
	infantry_folder = { ledger = army }
	armour_folder = { ledger = army available = { has_dlc = "No Step Back" } }
	land_doctrine_folder = { ledger = army doctrine = yes }
}`)},
	}
	mux := newMux(p, game)
	for _, line := range []string{"infantry", "rocketry", "rocketry", "alpha"} {
		call(t, mux, "POST", base, map[string]any{"tech_line": line})
	}
	known := decode[KnownTechnologyFoldersResponse](t, call(t, mux, "GET", base+"/known-folders", nil))
	if !reflect.DeepEqual(known.ProjectCustomLines, []string{"alpha", "rocketry"}) {
		t.Fatalf("custom=%v", known.ProjectCustomLines)
	}
	armour := known.Vanilla["armour_folder"]
	if len(known.Vanilla) != 3 || armour.Ledger != "army" || armour.DLCGate == nil || *armour.DLCGate != "No Step Back" ||
		!known.Vanilla["land_doctrine_folder"].Doctrine || known.Vanilla["infantry_folder"].DLCGate != nil {
		t.Fatalf("vanilla=%+v", known.Vanilla)
	}

	empty := decode[KnownTechnologyFoldersResponse](t, call(t, newMux(p, nil), "GET", base+"/known-folders", nil))
	if empty.Vanilla == nil || len(empty.Vanilla) != 0 {
		t.Fatalf("no game=%+v", empty)
	}
}

func TestReachability(t *testing.T) {
	p := newTestProject(t)
	err := p.Write([]string{"focus_trees", "focus_nodes", "events", "decisions"}, func(tx *project.Tx) error {
		tree := tx.Insert("focus_trees", project.Row{"tree_id": "ger_tree", "country_tag": "GER"})
		tx.Insert("focus_nodes", project.Row{"focus_tree_id": tree["id"], "focus_id": "GER_guns", "name": "x",
			"completion_reward": "set_technology = { infantry_weapons1 = 1 popup = yes }"})
		shared := tx.Insert("focus_trees", project.Row{"tree_id": "shared", "shared_country_tags": []any{"ITA"}})
		tx.Insert("focus_nodes", project.Row{"focus_tree_id": shared["id"], "focus_id": "ITA_radar", "name": "x",
			"select_effect": "set_technology = { radar = 1 }"})
		tx.Insert("events", project.Row{"event_id": "tst.1", "immediate": "",
			"options": []any{map[string]any{"effects": "set_technology = { radio = 1 }"}}})
		tx.Insert("decisions", project.Row{"decision_id": "tst_decision", "complete_effect": "set_technology = { radio = 1 }"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := newMux(p, nil)
	ger := decode[map[string]any](t, call(t, mux, "GET", base+"/reachability/ger", nil))
	if ger["country_tag"] != "GER" || !reflect.DeepEqual(ger["tech_ids"], []any{"infantry_weapons1"}) {
		t.Fatalf("ger=%v", ger)
	}
	tst := decode[map[string]any](t, call(t, mux, "GET", base+"/reachability/TST", nil))
	if !reflect.DeepEqual(tst["tech_ids"], []any{"radio"}) {
		t.Fatalf("project-tag fallback=%v", tst)
	}
	paths := decode[struct {
		TechID string           `json:"tech_id"`
		Paths  []TechUnlockPath `json:"paths"`
	}](t, call(t, mux, "GET", base+"/reachability/TST/radio/paths", nil))
	if paths.TechID != "radio" || len(paths.Paths) != 2 || paths.Paths[0].SourceKind != "event" ||
		paths.Paths[0].FieldName != "option[0].effects" || paths.Paths[1].SourceName != "tst_decision" {
		t.Fatalf("paths=%+v", paths)
	}
}

func TestTechReachabilityExtractionPythonParityFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/tech_reachability_extract_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name     string   `json:"name"`
		Effect   string   `json:"effect"`
		Expected []string `json:"expected"`
	}
	if err = json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if got := extractSetTechnology(tc.Effect); !slices.Equal(got, tc.Expected) && !(len(got) == 0 && len(tc.Expected) == 0) {
				t.Fatalf("got=%v want=%v", got, tc.Expected)
			}
		})
	}
}

func TestOwnerTagsPriorityMatchesPython(t *testing.T) {
	proj := reachabilityProject{primary: "GER", tags: []string{"GER", "AUS"}}
	if got := ownerTags("ita", []string{"FRA"}, proj); !reflect.DeepEqual(got, map[string]bool{"ITA": true}) {
		t.Fatalf("own priority=%v", got)
	}
	if got := ownerTags("", []string{"fra", "bel"}, proj); !reflect.DeepEqual(got, map[string]bool{"FRA": true, "BEL": true}) {
		t.Fatalf("shared priority=%v", got)
	}
	if got := ownerTags("", nil, proj); !reflect.DeepEqual(got, map[string]bool{"GER": true, "AUS": true}) {
		t.Fatalf("fallback=%v", got)
	}
}

func TestTechCountrySignalMirrorsPython(t *testing.T) {
	cases := map[string]string{
		"USA_army_rangers": "USA", "r56_USA_train_side_tanks": "USA", "HON_jungle_infantry": "HON",
		"MTG_naval_treaty_X": "", "r56_militia_tech": "", "infantry_weapons_1": "", "selva_tech": "",
		"Flexible_AT": "", "": "",
	}
	for id, want := range cases {
		if got := techCountrySignal(id); got != want {
			t.Fatalf("signal(%q)=%q want %q", id, got, want)
		}
	}
	if !techVisibleForCountry("USA_army_rangers", "USA") || techVisibleForCountry("USA_army_rangers", "GER") || !techVisibleForCountry("infantry_weapons_1", "GER") {
		t.Fatal("visibility rules broken")
	}
	if len(techLineFolders) != 13 || techLineFolders["armor"] != "armour_folder" || techLineFolders["mtg_naval"] != "mtgnavalfolder" {
		t.Fatal("folder map drift vs TECH_LINE_FOLDERS")
	}
}

func TestFolderPlacementsNormalizeAndKeepLegacyFirst(t *testing.T) {
	placements := normalizedFolderPlacements([]map[string]any{
		{"folder": "", "x": 1.5, "y": 3.25},
		{"folder": "light_air_techs_folder", "x": -2.75, "y": 4.5},
	}, "infantry_folder")
	folder, x, y, ok := firstFolderPlacement(placements)
	if !ok || folder != "infantry_folder" || x != 1.5 || y != 3.25 {
		t.Fatalf("legacy first placement = %q, %v, %v, %t", folder, x, y, ok)
	}
	if placements[1]["folder"] != "light_air_techs_folder" || placements[1]["x"] != -2.75 || placements[1]["y"] != 4.5 {
		t.Fatalf("source order or fractional placement changed: %#v", placements)
	}
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(0, 0, color.NRGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTechIconUploadListServeAndSize(t *testing.T) {
	p := newTestProject(t)
	mux := newMux(p, nil)
	tech := decode[Technology](t, call(t, mux, "POST", base, map[string]any{"tech_id": "my_tech"}))

	if icons := decode[map[string][]string](t, call(t, mux, "GET", "/api/go/projects/1/tech-icons", nil)); len(icons["icons"]) != 0 {
		t.Fatalf("icons before upload=%v", icons)
	}
	expect(t, call(t, mux, "GET", "/api/go/projects/1/tech-icons/my_tech", nil), http.StatusNotFound)

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("file", "icon.png")
	_, _ = part.Write(testPNG(t, 100, 80))
	_ = form.Close()
	req := httptest.NewRequest("POST", base+"/"+itoa(tech.ID)+"/upload-icon", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	expect(t, rec, http.StatusOK)
	uploaded := decode[map[string]any](t, rec)
	if uploaded["icon"] != "GFX_my_tech_medium" || uploaded["preview_url"] != "/api/go/projects/1/tech-icons/my_tech" || uploaded["filename"] != "my_tech.png" {
		t.Fatalf("upload=%v", uploaded)
	}

	icons := decode[map[string][]string](t, call(t, mux, "GET", "/api/go/projects/1/tech-icons", nil))
	if !reflect.DeepEqual(icons["icons"], []string{"my_tech"}) {
		t.Fatalf("icons=%v", icons)
	}
	served := call(t, mux, "GET", "/api/go/projects/1/tech-icons/my_tech", nil)
	expect(t, served, http.StatusOK)
	img, err := png.Decode(bytes.NewReader(served.Body.Bytes()))
	if err != nil || img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 || served.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("served icon err=%v type=%q", err, served.Header().Get("Content-Type"))
	}
	paths, _ := p.Rows("entity_obs_paths")
	if len(paths) != 1 || paths[0]["entity_type"] != "tech_icon" {
		t.Fatalf("entity_obs_paths=%v", paths)
	}

	size := decode[map[string]any](t, call(t, mux, "GET", "/api/go/projects/1/tech-icons/my_tech/size", nil))
	if size["width"] != 64.0 || size["height"] != 64.0 || size["base_game"] != false {
		t.Fatalf("size=%v", size)
	}
	expect(t, call(t, mux, "POST", base+"/999/upload-icon", nil), http.StatusNotFound)
}

func TestVanillaTechIconFromGame(t *testing.T) {
	pix := make([]byte, 8*8*4)
	for i := range pix {
		pix[i] = 128
	}
	dds := imgcodec.EncodeDXT5DDS(pix, 8, 8)
	game := fstest.MapFS{
		"interface/Technologies.gfx": {Data: []byte(`spriteTypes = {
	spriteType = { name = "GFX_infantry_weapons_medium" texturefile = "gfx/interface/technologies/infantry_weapons.dds" }
	spriteType = { name = "GFX_archetype_tank_medium" texturefile = "gfx/interface/archetypes/tank.dds" }
}`)},
		"gfx/interface/technologies/infantry_weapons.dds":   {Data: dds},
		"dlc/dlc001_pack/gfx/interface/archetypes/tank.dds": {Data: dds},
		"common/units/equipment/tanks.txt": {Data: []byte(`equipments = {
	tank = { is_archetype = yes picture = archetype_tank }
	tank_1 = { archetype = tank }
}`)},
		"common/technologies/armor.txt": {Data: []byte(`technologies = { basic_tank = { enable_equipments = { tank_1 } } }`)},
	}
	mux := newMux(newTestProject(t), game)
	direct := call(t, mux, "GET", "/api/go/vanilla/tech-icon/infantry_weapons", nil)
	expect(t, direct, http.StatusOK)
	if direct.Header().Get("X-Vanilla-Icon-Resolution") != "direct" {
		t.Fatalf("resolution=%q", direct.Header().Get("X-Vanilla-Icon-Resolution"))
	}
	if img, err := png.Decode(bytes.NewReader(direct.Body.Bytes())); err != nil || img.Bounds().Dx() != 8 {
		t.Fatalf("png err=%v", err)
	}
	archetype := call(t, mux, "GET", "/api/go/vanilla/tech-icon/basic_tank", nil)
	expect(t, archetype, http.StatusOK)
	if archetype.Header().Get("X-Vanilla-Icon-Resolution") != "archetype" {
		t.Fatalf("archetype resolution=%q", archetype.Header().Get("X-Vanilla-Icon-Resolution"))
	}
	expect(t, call(t, mux, "GET", "/api/go/vanilla/tech-icon/unknown", nil), http.StatusNotFound)
	expect(t, call(t, newMux(newTestProject(t), nil), "GET", "/api/go/vanilla/tech-icon/x", nil), http.StatusServiceUnavailable)
}

func itoa(v int64) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
