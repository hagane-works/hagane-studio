package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/gameindex"
	"haganeworks/internal/project"
)

func newProject(t *testing.T) *project.Project {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newServer(t *testing.T, p *project.Project, game fs.FS) *http.ServeMux {
	t.Helper()
	if game != nil {
		useGameIndex(t, game)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(r *http.Request) (*project.Project, error) {
			if r.PathValue("id") != strconv.FormatInt(p.ID(), 10) {
				return nil, project.ErrNotFound
			}
			return p, nil
		},
		Game: func() (fs.FS, error) {
			if game == nil {
				return nil, errors.New("set the Hearts of Iron IV folder in settings first")
			}
			return game, nil
		},
	})
	return mux
}

func call(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body is not a JSON object: %v\n%s", err, rec.Body.String())
	}
	return out
}

func expect(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	return decode(t, rec)
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 10), G: 80, B: uint8(y * 10), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRegisterHasNoConflicts(t *testing.T) {
	newServer(t, newProject(t), nil)
}

func TestUsageAndAuditLog(t *testing.T) {
	p := newProject(t)
	if err := p.Write([]string{"characters", "ideas"}, func(tx *project.Tx) error {
		tx.Insert("characters", project.Row{"character_id": "TST_a", "name": "A"})
		tx.Insert("characters", project.Row{"character_id": "TST_b", "name": "B"})
		tx.Insert("ideas", project.Row{"idea_id": "TST_i", "category": "country"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	mux := newServer(t, p, nil)
	pid := strconv.FormatInt(p.ID(), 10)

	rec := call(t, mux, "GET", "/api/go/projects/"+pid+"/usage", "")
	body := expect(t, rec, 200)
	if body["project_name"] != "t" || body["entity_total"] != float64(3) || body["audit_count"] != float64(0) {
		t.Fatalf("usage = %v", body)
	}

	raw := rec.Body.String()
	if !strings.Contains(raw, `"entity_counts":{"characters":2,"ideas":1,"focus_trees":0,`) ||
		!strings.Contains(raw, `"military_industrial_orgs":0,"opinion_modifiers":0,"modifier_definitions":0}`) {
		t.Fatalf("entity_counts order: %s", raw)
	}
	tokens := body["tokens"].(map[string]any)
	if tokens["scope"] != "owner_user_total_approximate" || tokens["total_calls"] != float64(0) {
		t.Fatalf("tokens = %v", tokens)
	}

	body = expect(t, call(t, mux, "GET", "/api/go/projects/"+pid+"/audit-log?limit=10&offset=5", ""), 200)
	if body["total"] != float64(0) || body["limit"] != float64(10) || body["offset"] != float64(5) || len(body["logs"].([]any)) != 0 {
		t.Fatalf("audit-log = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/projects/"+pid+"/audit-log?limit=201", ""), 422)
	if body["detail"] != "limit must be between 1 and 200" {
		t.Fatalf("detail = %v", body["detail"])
	}
	body = expect(t, call(t, mux, "GET", "/api/go/projects/"+pid+"/audit-log?offset=-1", ""), 422)
	if body["detail"] != "offset must be >= 0" {
		t.Fatalf("detail = %v", body["detail"])
	}
	expect(t, call(t, mux, "GET", "/api/go/projects/999/usage", ""), 404)
}

func TestEditTargets(t *testing.T) {
	p := newProject(t)
	mux := newServer(t, p, nil)
	pid := strconv.FormatInt(p.ID(), 10)
	body := expect(t, call(t, mux, "GET", "/api/go/projects/"+pid+"/edit-targets", ""), 200)
	if body["mod_name"] != "t" || body["primary_tag"] != "TST" || body["is_loaded_existing_mod"] != false {
		t.Fatalf("metadata = %v", body)
	}
	if _, failed := body["error"]; failed {
		t.Fatalf("index failed: %v", body["error"])
	}
	byKind := body["by_kind"].(map[string]any)
	total := 0
	for _, entry := range byKind {
		total += int(entry.(map[string]any)["count"].(float64))
	}
	if total != int(body["total_files"].(float64)) || total == 0 {
		t.Fatalf("total_files %v vs kinds %v", body["total_files"], byKind)
	}
	if _, ok := byKind["descriptor"]; !ok {
		t.Fatalf("no descriptor kind: %v", byKind)
	}

	again := call(t, mux, "GET", "/api/go/projects/"+pid+"/edit-targets", "")
	first := call(t, mux, "GET", "/api/go/projects/"+pid+"/edit-targets?include_imported=false", "")
	if again.Body.String() != first.Body.String() {
		t.Fatal("repeat call differs")
	}

	body = expect(t, call(t, mux, "GET", "/api/go/projects/"+pid+"/edit-targets?include_imported=maybe", ""), 422)
	detail := body["detail"].([]any)[0].(map[string]any)
	if detail["type"] != "bool_parsing" {
		t.Fatalf("detail = %v", detail)
	}
}

func TestClassifyEditTargets(t *testing.T) {
	got := classifyEditTargets(editTargetsResponse{PrimaryTag: "TST", ByKind: map[string]*editTargetKind{}}, []string{
		"mod/common/decisions/categories/TST_cats.txt",
		"mod/common/decisions/shared.txt",
		"mod/localisation/english/zz_TST_l_english.yml",
		"mod/descriptor.mod",
		"mod/",
		"mod/other/x.txt",
		"mod/_CROSS_REF_SKIPPED.log",
	})
	if got.TotalFiles != 5 {
		t.Fatalf("total = %d", got.TotalFiles)
	}
	if got.ByKind["decision_category"].Modes["append"] != 1 || got.ByKind["decision"].Modes["shared"] != 1 ||
		got.ByKind["localisation"].Modes["append"] != 1 || got.ByKind["descriptor"].Modes["overwrite"] != 1 ||
		got.ByKind["other"].Count != 1 || got.ByKind["decision"].Label != "决议" {
		t.Fatalf("by_kind = %+v", got.ByKind)
	}
}

func TestImageBatchPlaceholderAndOverlay(t *testing.T) {
	mux := newServer(t, newProject(t), nil)
	body := expect(t, call(t, mux, "POST", "/api/go/image-batch/generate-placeholder",
		`{"entities":[{"entity_id":"TST_idea","label":"Idea"}],"kind":"focus","output":"json"}`), 200)
	results := body["results"].([]any)
	if body["count"] != float64(1) || len(results) != 1 {
		t.Fatalf("placeholder = %v", body)
	}
	raw, _ := base64.StdEncoding.DecodeString(results[0].(map[string]any)["b64"].(string))
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil || img.Bounds().Dx() != 95 || img.Bounds().Dy() != 95 {
		t.Fatalf("placeholder image: %v %v", err, img)
	}
	rec := call(t, mux, "POST", "/api/go/image-batch/generate-placeholder", `{"entities":[{"entity_id":"a"}]}`)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("zip output: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	body = expect(t, call(t, mux, "POST", "/api/go/image-batch/generate-placeholder", `{"entities":[]}`), 400)
	if body["detail"] != "entities list is empty" {
		t.Fatalf("detail = %v", body)
	}
	expect(t, call(t, mux, "POST", "/api/go/image-batch/generate-placeholder", `{"entities":[{}],"width":0}`), 400)

	item := base64.StdEncoding.EncodeToString(pngBytes(t, 64, 32))
	body = expect(t, call(t, mux, "POST", "/api/go/image-batch/overlay-text",
		`{"items":[{"b64_image":"`+item+`","text":"HI"},{"b64_image":"%%%"},{"b64_image":"`+item+`","color":[300,0,0]}]}`), 200)
	rows := body["results"].([]any)
	if body["ok"] != float64(1) || body["errors"] != float64(2) ||
		rows[1].(map[string]any)["error"] != "b64_image is not valid base64" ||
		rows[2].(map[string]any)["error"] != "RGB values must be integers between 0 and 255" {
		t.Fatalf("overlay = %v", body)
	}
	expect(t, call(t, mux, "POST", "/api/go/image-batch/overlay-text", `{"items":[],"output":"json"}`), 400)
	expect(t, call(t, mux, "POST", "/api/go/image-batch/overlay-text", `{"items":[{}],"output":"tar"}`), 400)
}

func TestImageBatchConvertDDS(t *testing.T) {
	mux := newServer(t, newProject(t), nil)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("files", "a.png")
	part.Write(pngBytes(t, 8, 8))
	part, _ = writer.CreateFormFile("files", "broken.png")
	part.Write([]byte("not an image"))
	writer.WriteField("target_width", "4")
	writer.WriteField("target_height", "4")
	writer.Close()
	req := httptest.NewRequest("POST", "/api/go/image-batch/convert-dds", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("convert: %d %s", rec.Code, rec.Body.String())
	}
	var frames []map[string]any
	scanner := bufio.NewScanner(rec.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		if line, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
			var frame map[string]any
			if err := json.Unmarshal([]byte(line), &frame); err != nil {
				t.Fatal(err)
			}
			frames = append(frames, frame)
		}
	}
	if len(frames) != 4 || frames[0]["event"] != "start" || frames[3]["event"] != "done" {
		t.Fatalf("frames = %v", frames)
	}
	ok, bad := frames[1], frames[2]
	if ok["status"] != "ok" || ok["width"] != float64(4) || bad["status"] != "error" ||
		bad["error"] != "Could not decode image (corrupt or unsupported format)" {
		t.Fatalf("file frames = %v / %v", ok, bad)
	}
	dds, _ := base64.StdEncoding.DecodeString(ok["b64_output"].(string))
	if !bytes.HasPrefix(dds, []byte("DDS ")) {
		t.Fatal("output is not a DDS")
	}

	buf.Reset()
	writer = multipart.NewWriter(&buf)
	writer.WriteField("target_width", "4")
	writer.Close()
	req = httptest.NewRequest("POST", "/api/go/image-batch/convert-dds", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if body := expect(t, rec, 400); body["detail"] != "Provide both target_width and target_height or neither" {
		t.Fatalf("detail = %v", body)
	}
}

func TestImageBatchVanilla(t *testing.T) {
	game := fstest.MapFS{
		"gfx/interface/goals/goal_b.dds":   {Data: []byte("x")},
		"gfx/interface/goals/goal_a.DDS":   {Data: []byte("x")},
		"gfx/interface/goals/readme.txt":   {Data: []byte("x")},
		"gfx/interface/ideas/idea_one.dds": {Data: []byte("x")},
	}
	mux := newServer(t, newProject(t), game)
	body := expect(t, call(t, mux, "GET", "/api/go/image-batch/vanilla-root-status", ""), 200)
	dirs := body["asset_dirs"].([]any)
	if body["root_exists"] != true || body["configured_via_env"] != false || body["hint"] != nil || len(dirs) != 4 {
		t.Fatalf("status = %v", body)
	}
	counts := map[string]float64{}
	for _, d := range dirs {
		entry := d.(map[string]any)
		counts[entry["asset_type"].(string)] = entry["dds_count"].(float64)
	}
	if counts["focus_icons"] != 2 || counts["idea_pictures"] != 1 || counts["event_pictures"] != 0 {
		t.Fatalf("counts = %v", counts)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/image-batch/vanilla-extract/focus_icons", ""), 200)
	assets := body["assets"].([]any)
	if body["count"] != float64(2) || assets[0].(map[string]any)["filename"] != "goal_a.DDS" ||
		assets[1].(map[string]any)["relative_path"] != "gfx/interface/goals/goal_b.dds" ||
		assets[1].(map[string]any)["name"] != "goal_b" {
		t.Fatalf("extract = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/image-batch/vanilla-extract/bogus", ""), 400)
	if !strings.HasPrefix(body["detail"].(string), `Unknown asset_type "bogus"`) {
		t.Fatalf("detail = %v", body)
	}

	noGame := newServer(t, newProject(t), nil)
	body = expect(t, call(t, noGame, "GET", "/api/go/image-batch/vanilla-root-status", ""), 200)
	if body["root_exists"] != false || body["hint"] == nil || len(body["asset_dirs"].([]any)) != 0 {
		t.Fatalf("no game status = %v", body)
	}
	body = expect(t, call(t, noGame, "GET", "/api/go/image-batch/vanilla-extract/focus_icons", ""), 200)
	if body["count"] != float64(0) {
		t.Fatalf("no game extract = %v", body)
	}
}

func fakeGame() fstest.MapFS {
	return fstest.MapFS{
		"common/units/equipment/infantry.txt": {Data: []byte(`equipments = {
	infantry_equipment = { year = 1918 is_archetype = yes }
	infantry_equipment_1 = { year = 1918 archetype = infantry_equipment }
#	old_tank_equipment = { is_archetype = yes }
}`)},
		"common/units/equipment/artillery.txt": {Data: []byte("equipments = { artillery_equipment = { is_archetype = yes } }")},
		"common/units/infantry.txt": {Data: []byte(`sub_units = {
	infantry = {
		abbreviation = "INF"
		type = { infantry }
		group = infantry
		combat_width = 2
		max_strength = 25
		max_organisation = 60
		default_morale = 0.3
		manpower = 1000
		forest = { attack = 0.5 } # nested values are ignored
		soft_attack = -0.5
	}
	engineer = { type = { infantry support } group = support abbreviation = "ENG" }
	fighter = { type = { fighter } }
}`)},
		"dlc/dlc001_x/common/units/more.txt": {Data: []byte(`sub_units = {
	infantry = { type = { infantry } combat_width = 9 }
	marine = { type = { infantry } combat_width = 2 }
}`)},
		"events/Generic.txt": {Data: []byte(`add_namespace = generic
add_namespace = other

country_event = {
	id = generic.1
	title = generic.1.t
	picture = GFX_report_event_generic
	is_triggered_only = yes
	option = { name = a }
	option = { name = b }
}
news_event = { id = other.7
	title = "Quoted title"
	major = yes
	hidden = yes
	option = { name = a }
}
country_event = { id = foreign.2 fire_only_once = yes }
`)},
		"common/technologies/infantry.txt": {Data: []byte(`technologies = {
	infantry_weapons = { sub_technologies = { infantry_weapons_sub = { } } }
	@var = 5
	support_weapons = { }
}`)},
		"common/decisions/categories/cats.txt": {Data: []byte("# comment = { }\nTST_cat = { icon = x }\nother_cat = {\n}\n")},
		"common/dynamic_modifiers/mods.txt":    {Data: []byte("dyn_a = { attack = 1 }\n")},
		"common/national_focus/generic.txt": {Data: []byte(`focus_tree = { id = generic_focus country = { factor = 1 } }
shared_focus = { id = shared_one }`)},
	}
}

func TestVanillaGameFacts(t *testing.T) {
	mux := newServer(t, newProject(t), fakeGame())

	body := expect(t, call(t, mux, "GET", "/api/go/vanilla/equipment-archetypes", ""), 200)
	archetypes := body["archetypes"].([]any)
	if body["total"] != float64(2) || archetypes[0] != "artillery_equipment" || archetypes[1] != "infantry_equipment" {
		t.Fatalf("archetypes = %v", body)
	}

	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/subunits", ""), 200)
	subunits := body["subunits"].(map[string]any)
	if body["total"] != float64(3) || len(body["stat_keys"].([]any)) != 10 {
		t.Fatalf("subunits = %v", body)
	}
	infantry := subunits["infantry"].(map[string]any)
	if infantry["abbreviation"] != "INF" || infantry["combat_width"] != float64(2) || infantry["soft_attack"] != -0.5 ||
		infantry["is_support"] != false || infantry["supply_consumption"] != float64(0) {
		t.Fatalf("infantry = %v", infantry)
	}
	if engineer := subunits["engineer"].(map[string]any); engineer["is_support"] != true || engineer["group"] != "support" {
		t.Fatalf("engineer = %v", engineer)
	}
	if _, ok := subunits["fighter"]; ok {
		t.Fatal("air sub-unit listed")
	}
	if _, ok := subunits["marine"]; !ok {
		t.Fatal("DLC sub-unit missing")
	}

	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/id-collisions", ""), 200)
	want := map[string]string{
		"tech_ids":              `["infantry_weapons","infantry_weapons_sub","support_weapons"]`,
		"focus_tree_ids":        `["generic_focus"]`,
		"decision_category_ids": `["TST_cat","other_cat"]`,
		"dynamic_modifier_ids":  `["dyn_a"]`,
	}
	for key, ids := range want {
		got, _ := json.Marshal(body[key])
		if string(got) != ids {
			t.Fatalf("%s = %s, want %s", key, got, ids)
		}
	}
	if body["tech_total"] != float64(3) {
		t.Fatalf("tech_total = %v", body["tech_total"])
	}
}

func TestVanillaEvents(t *testing.T) {
	mux := newServer(t, newProject(t), fakeGame())
	body := expect(t, call(t, mux, "GET", "/api/go/vanilla/events", ""), 200)
	items := body["items"].([]any)
	meta := body["meta"].(map[string]any)
	if body["total_matching"] != float64(3) || len(items) != 3 || meta["total"] != float64(3) || meta["namespaces"] != float64(2) {
		t.Fatalf("events = %v", body)
	}
	first := items[0].(map[string]any)
	if first["id"] != "generic.1" || first["ns"] != "generic" || first["type"] != "country_event" ||
		first["title"] != "generic.1.t" || first["picture"] != "GFX_report_event_generic" ||
		first["is_triggered_only"] != true || first["option_count"] != float64(2) ||
		first["file"] != "Generic.txt" || first["line"] != float64(3) {
		t.Fatalf("first = %v", first)
	}
	second := items[1].(map[string]any)
	if second["title"] != "Quoted title" || second["major"] != true || second["hidden"] != true || second["line"] != float64(12) {
		t.Fatalf("second = %v", second)
	}

	if third := items[2].(map[string]any); third["ns"] != "generic" || third["fire_only_once"] != false || third["line"] != float64(18) {
		t.Fatalf("third = %v", third)
	}

	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/events?event_type=news_event", ""), 200)
	if body["total_matching"] != float64(1) {
		t.Fatalf("type filter = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/events?q=GFX_REPORT&namespace_prefix=GEN", ""), 200)
	if body["total_matching"] != float64(1) {
		t.Fatalf("q filter = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/events?limit=0&offset=2", ""), 200)
	if body["limit"] != float64(1) || len(body["items"].([]any)) != 1 || body["items"].([]any)[0].(map[string]any)["id"] != "foreign.2" {
		t.Fatalf("paging = %v", body)
	}
	expect(t, call(t, mux, "GET", "/api/go/vanilla/events?limit=x", ""), 422)

	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/events/generic/1", ""), 200)

	if !strings.HasPrefix(body["body"].(string), "\ncountry_event = {") || !strings.HasSuffix(body["body"].(string), "}\n") ||
		body["entry"].(map[string]any)["id"] != "generic.1" {
		t.Fatalf("detail = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/events/generic/9", ""), 404)
	if body["detail"] != "vanilla event generic.9 not found" {
		t.Fatalf("detail = %v", body)
	}
	expect(t, call(t, mux, "GET", "/api/go/vanilla/events/generic/x", ""), 422)

	noGame := newServer(t, newProject(t), nil)
	body = expect(t, call(t, noGame, "GET", "/api/go/vanilla/events", ""), 200)
	if body["total_matching"] != float64(0) {
		t.Fatalf("no game = %v", body)
	}
	body = expect(t, call(t, noGame, "GET", "/api/go/vanilla/subunits", ""), 200)
	if body["total"] != float64(0) {
		t.Fatalf("no game subunits = %v", body)
	}
}

func TestScriptNamesAndStructuralFields(t *testing.T) {
	mux := newServer(t, newProject(t), nil)
	body := expect(t, call(t, mux, "GET", "/api/go/vanilla/script-names", ""), 200)
	if len(body["triggers"].([]any)) == 0 || len(body["effects"].([]any)) == 0 ||
		body["math_function_total"] != float64(36) || body["extracted_at"] == nil {
		t.Fatalf("script-names = %v", body["math_function_total"])
	}
	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/structural-fields", ""), 200)
	fields := body["fields"].([]any)
	if len(fields) == 0 || len(body) != 2 || len(body["by_entity"].(map[string]any)) == 0 {
		t.Fatalf("structural-fields keys = %d", len(body))
	}

	has := map[string]bool{}
	for _, f := range fields {
		has[f.(string)] = true
	}
	if !has["is_triggered_only"] || !has["political_power_gain"] || !has["infantry_folder"] {
		t.Fatalf("structural fields miss a group: %d fields", len(fields))
	}

	t.Cleanup(gameindex.SetDefault(nil))
	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/script-names", ""), 200)
	if len(body["triggers"].([]any)) != 0 || body["math_function_total"] != float64(0) {
		t.Fatalf("no game script-names = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/vanilla/structural-fields", ""), 200)
	if len(body["fields"].([]any)) != 0 {
		t.Fatalf("no game structural-fields = %v", body)
	}
}

func TestVanillaRoutesWhileBuilding(t *testing.T) {
	block := make(chan struct{})
	routes := map[string]string{
		"/api/go/vanilla/equipment-archetypes": gameindex.Equipment,
		"/api/go/vanilla/country-aliases":      gameindex.CountryAliases,
		"/api/go/vanilla/state-content":        gameindex.StateContent,
		"/api/go/vanilla/dlc-states":           gameindex.DLCStates,
		"/api/go/vanilla/tech-sections":        gameindex.TechSections,
		"/api/go/vanilla/unit-names":           gameindex.UnitNames,
	}
	var slow []gameindex.Dataset
	for _, name := range routes {
		slow = append(slow, gameindex.Dataset{Name: name, Version: 1, Build: func(ctx context.Context, in *gameindex.Input) ([]byte, error) {
			select {
			case <-block:
			case <-ctx.Done():
			}
			return []byte(`{}`), nil
		}})
	}
	game := fakeGame()
	x, err := gameindex.New(func() (fs.FS, error) { return game, nil }, slow, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gameindex.SetDefault(x))
	defer close(block)
	mux := newServer(t, newProject(t), nil)
	for route := range routes {
		rec := call(t, mux, "GET", route, "")
		body := expect(t, rec, http.StatusServiceUnavailable)
		if body["code"] != "game_index_building" || body["detail"] == "" {
			t.Fatalf("%s building = %v", route, body)
		}
	}
}

func TestStaticRoutes(t *testing.T) {
	mux := newServer(t, newProject(t), nil)
	body := expect(t, call(t, mux, "GET", "/api/go/cwtool/whitelist", ""), 200)
	if body["version"] == nil || len(body["known_effects"].([]any)) == 0 {
		t.Fatal("cwtool whitelist incomplete")
	}
	body = expect(t, call(t, mux, "GET", "/api/go/qiuqi/modifiers", ""), 200)
	translations := body["translations"].(map[string]any)
	if body["version"] != float64(1) || body["count"] != float64(len(translations)) || len(translations) == 0 {
		t.Fatalf("qiuqi count = %v", body["count"])
	}
	var key string
	for key = range translations {
		break
	}
	expect(t, call(t, mux, "GET", "/api/go/qiuqi/modifier/"+strings.ToUpper(key), ""), 200)
	body = expect(t, call(t, mux, "GET", "/api/go/qiuqi/modifier/nope", ""), 404)
	if body["detail"] != `No QIUQI translation for modifier "nope"` {
		t.Fatalf("detail = %v", body)
	}

	body = expect(t, call(t, mux, "GET", "/api/go/workflow-templates?limit=50&offset=0", ""), 200)
	if body["total"] != float64(0) || len(body["templates"].([]any)) != 0 || body["scope"] != nil {
		t.Fatalf("library = %v", body)
	}
	body = expect(t, call(t, mux, "GET", "/api/go/workflow-templates?scope=marketplace", ""), 200)
	if body["scope"] != "marketplace" {
		t.Fatalf("marketplace = %v", body)
	}
	expect(t, call(t, mux, "GET", "/api/go/workflow-templates/abc", ""), 404)
}

func TestBasemodOffline(t *testing.T) {
	p := newProject(t)
	mux := newServer(t, p, nil)
	pid := strconv.FormatInt(p.ID(), 10)

	body := expect(t, call(t, mux, "GET", "/api/go/basemod/templates", ""), 200)
	if body["total"] != float64(0) || len(body["templates"].([]any)) != 0 || len(body["items"].([]any)) != 0 {
		t.Fatalf("templates = %v", body)
	}

	if _, err := p.UpdateRecord(func(r project.Row) { r["dependencies"] = []any{"The New Order: Last Days of Europe", 7, " "} }); err != nil {
		t.Fatal(err)
	}

	body = expect(t, call(t, mux, "GET", "/api/go/basemod/refs?project_id="+pid+"&type=character&q=x", ""), 200)
	if body["type"] != "character" || len(body["results"].([]any)) != 0 {
		t.Fatalf("refs = %v", body)
	}
	expect(t, call(t, mux, "GET", "/api/go/basemod/refs?project_id="+pid+"&type=nope", ""), 400)
	expect(t, call(t, mux, "GET", "/api/go/basemod/refs?project_id="+pid+"&type=focus&limit=500", ""), 422)
	expect(t, call(t, mux, "GET", "/api/go/basemod/refs?project_id=x&type=focus", ""), 400)

	body = expect(t, call(t, mux, "GET", "/api/go/basemod/ideology-registry?project_id="+pid, ""), 200)
	if len(body["groups"].([]any)) != 0 || len(body["sub_ideologies"].([]any)) != 0 {
		t.Fatalf("registry = %v", body)
	}
	expect(t, call(t, mux, "GET", "/api/go/basemod/ideology-registry", ""), 404)

	body = expect(t, call(t, mux, "GET", "/api/go/basemod/id-taken?project_id="+pid+"&type=focus&id=TST_x", ""), 200)
	if body["indexed"] != false || body["taken"] != false {
		t.Fatalf("id-taken = %v", body)
	}
	expect(t, call(t, mux, "GET", "/api/go/basemod/id-taken?project_id="+pid+"&type=state&id=1", ""), 400)
	expect(t, call(t, mux, "GET", "/api/go/basemod/id-taken?project_id="+pid+"&type=focus&id=", ""), 400)
	body = expect(t, call(t, mux, "POST", "/api/go/basemod/id-taken", `{"project_id":`+pid+`,"type":"idea","ids":["a","b"]}`), 200)
	if body["indexed"] != false || len(body["taken"].(map[string]any)) != 0 {
		t.Fatalf("id-taken batch = %v", body)
	}

	tno := "/api/go/projects/" + pid + "/basemod/tno"
	body = expect(t, call(t, mux, "GET", tno+"/gfx/GFX_x", ""), 404)
	if body["detail"] != "no base-mod folder for dependency 'tno'" {
		t.Fatalf("gfx = %v", body)
	}
	expect(t, call(t, mux, "GET", tno+"/art/gfx/leaders/a.dds", ""), 404)
	expect(t, call(t, mux, "GET", tno+"/flag/GER?size=medium", ""), 404)
	body = expect(t, call(t, mux, "GET", tno+"/flag/GER?size=huge", ""), 400)
	if body["detail"] != "invalid flag size: 'huge' (large|medium|small)" {
		t.Fatalf("flag = %v", body)
	}
	expect(t, call(t, mux, "GET", tno+"/flag/GER?ideology=bad-one", ""), 400)
}
