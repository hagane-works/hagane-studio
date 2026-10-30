package decisions

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
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

type testEnv struct {
	t   *testing.T
	p   *project.Project
	mux *http.ServeMux
}

func newEnv(t *testing.T, game fs.FS) *testEnv {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) {
			if game == nil {
				return nil, fs.ErrNotExist
			}
			return game, nil
		},
	})
	return &testEnv{t: t, p: p, mux: mux}
}

func (e *testEnv) do(method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, "/api/go/projects/1"+path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) expect(method, path, body string, status int, fragments ...string) *httptest.ResponseRecorder {
	e.t.Helper()
	rec := e.do(method, path, body)
	if rec.Code != status {
		e.t.Fatalf("%s %s: status=%d want %d body=%s", method, path, rec.Code, status, rec.Body.String())
	}
	for _, fragment := range fragments {
		if !strings.Contains(rec.Body.String(), fragment) {
			e.t.Fatalf("%s %s: body %s lacks %s", method, path, rec.Body.String(), fragment)
		}
	}
	return rec
}

func decodeInto[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return out
}

func (e *testEnv) createCategory(categoryID string) DecisionCategory {
	e.t.Helper()
	rec := e.expect("POST", "/decisions/categories", `{"category_id":"`+categoryID+`","country_tag":"TST"}`, 201)
	return decodeInto[DecisionCategory](e.t, rec)
}

func (e *testEnv) createDecision(categoryID int64, decisionID string, extra string) Decision {
	e.t.Helper()
	body := `{"category_id":` + itoa(categoryID) + `,"decision_id":"` + decisionID + `"` + extra + `}`
	rec := e.expect("POST", "/decisions", body, 201)
	return decodeInto[Decision](e.t, rec)
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }

func TestCategoryCRUD(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("  war_plans ")
	if cat.CategoryID != "war_plans" || cat.ScopeKind != "own" || cat.Visibility != "visible" ||
		cat.CountryTag != "TST" || len(cat.Warnings) != 0 || cat.DecisionCount != 0 || cat.CreatedAt == "" {
		t.Fatalf("created category: %+v", cat)
	}
	if !regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d{6})?$`).MatchString(cat.CreatedAt) {
		t.Fatalf("created_at %q is not in the expected ISO form", cat.CreatedAt)
	}
	raw := e.expect("GET", "/decisions/categories/"+itoa(cat.ID), "", 200).Body.String()
	for _, want := range []string{`"raw_extras":{}`, `"shared_country_tags":null`, `"display_name_locs":{}`, `"warnings":[]`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("category response %s lacks %s", raw, want)
		}
	}
	e.expect("POST", "/decisions/categories", `{"category_id":"war_plans"}`, 409,
		"category_id 'war_plans' already exists in this project. Pick a unique id.")
	e.expect("POST", "/decisions/categories", `{"icon":"x"}`, 400, "category_id is required")
	e.expect("POST", "/decisions/categories", `not json`, 422, "Invalid category payload")

	other := e.createCategory("economy")
	e.createDecision(cat.ID, "mobilize", "")
	e.createDecision(cat.ID, "demobilize", "")
	list := decodeInto[[]DecisionCategory](t, e.expect("GET", "/decisions/categories", "", 200))
	if len(list) != 2 || list[0].DecisionCount != 2 || list[1].DecisionCount != 0 {
		t.Fatalf("list: %+v", list)
	}
	if got := decodeInto[[]DecisionCategory](t, e.expect("GET", "/decisions/categories?country_tag=GER", "", 200)); len(got) != 0 {
		t.Fatalf("country filter: %+v", got)
	}

	updated := decodeInto[DecisionCategory](t, e.expect("PUT", "/decisions/categories/"+itoa(cat.ID),
		`{"priority":7,"allowed":"tag = TST\\nhas_war = no","name_en":"War Plans"}`, 200))
	if updated.Priority != 7 || updated.Allowed != "tag = TST\nhas_war = no" || updated.NameEn != "War Plans" || updated.DecisionCount != 2 {
		t.Fatalf("updated: %+v", updated)
	}
	e.expect("PUT", "/decisions/categories/"+itoa(other.ID), `{"category_id":"war_plans"}`, 409, "already exists")
	e.expect("PUT", "/decisions/categories/"+itoa(cat.ID), `{"visible":"has_flag = \"a\\nb\""}`, 400, "请在第 1 行")
	e.expect("PUT", "/decisions/categories/999", `{"priority":1}`, 404, "Category not found")
	e.expect("GET", "/decisions/categories/abc", "", 404, "Category not found")
	e.expect("PATCH", "/decisions/categories/"+itoa(cat.ID), "", 405, "GET, PUT or DELETE only")
	e.expect("DELETE", "/decisions/categories", "", 404, "Not Found")

	e.expect("DELETE", "/decisions/categories/"+itoa(cat.ID), "", 204)
	e.expect("GET", "/decisions/categories/"+itoa(cat.ID), "", 404, "Category not found")
	if got := decodeInto[[]Decision](t, e.expect("GET", "/decisions", "", 200)); len(got) != 0 {
		t.Fatalf("decisions of a deleted category must go with it: %+v", got)
	}
	e.expect("DELETE", "/decisions/categories/"+itoa(cat.ID), "", 404, "Category not found")
}

func TestDecisionCRUD(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("war_plans")
	other := e.createCategory("economy")
	d := e.createDecision(cat.ID, " mobilize ", `,"cost":25,"targets":["GER"],"complete_effect":"add_stability = 0.1\\nadd_war_support = 0.1"`)
	if d.DecisionID != "mobilize" || d.Cost != 25 || d.Source != "created" || d.ScopeKind != "own" ||
		len(d.Targets) != 1 || d.CompleteEffect != "add_stability = 0.1\nadd_war_support = 0.1" || len(d.Warnings) != 0 {
		t.Fatalf("created: %+v", d)
	}
	raw := e.expect("GET", "/decisions/"+itoa(d.ID), "", 200).Body.String()
	for _, want := range []string{`"targeted_modifier":{}`, `"target_array":[]`, `"highlight_states":[]`, `"effect_tooltip_locs":{}`, `"shared_country_tags":null`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("decision response %s lacks %s", raw, want)
		}
	}

	dup := e.createDecision(cat.ID, "mobilize", "")
	if dup.DecisionID != "mobilize_1" || len(dup.Warnings) != 1 || dup.Warnings[0] != "decision_id 'mobilize' collided · auto-suffixed to 'mobilize_1'." {
		t.Fatalf("auto-suffix: %+v", dup)
	}
	e.createDecision(other.ID, "invest", "")

	e.expect("POST", "/decisions", `{"decision_id":"x"}`, 422, "category_id is required")
	e.expect("POST", "/decisions", `{"category_id":1}`, 422, "decision_id is required")
	e.expect("POST", "/decisions", `{"category_id":999,"decision_id":"x"}`, 400, "Category not found in this project")
	e.expect("POST", "/decisions", `{"category_id":"x"}`, 422, "Invalid decision payload")
	e.expect("POST", "/decisions", `{"category_id":1,"decision_id":"x","visible":"has_flag = \"a\\nb\""}`, 422, "请在第 1 行")
	e.expect("DELETE", "/decisions", "", 405, "GET or POST only")

	if got := decodeInto[[]Decision](t, e.expect("GET", "/decisions?category_id="+itoa(cat.ID), "", 200)); len(got) != 2 {
		t.Fatalf("category filter: %+v", got)
	}
	if got := decodeInto[[]Decision](t, e.expect("GET", "/decisions", "", 200)); len(got) != 3 {
		t.Fatalf("list: %+v", got)
	}
	if got := decodeInto[[]Decision](t, e.expect("GET", "/decisions?decision_id=invest", "", 200)); len(got) != 1 || got[0].CategoryID != other.ID {
		t.Fatalf("lookup: %+v", got)
	}
	e.expect("GET", "/decisions?decision_id=nothing", "", 200, "[]")

	updated := decodeInto[Decision](t, e.expect("PUT", "/decisions/"+itoa(d.ID),
		`{"cost":50,"category_id":`+itoa(other.ID)+`,"name_en":"Mobilise","targets":null,"id":99,"warnings":["x"]}`, 200))
	if updated.Cost != 50 || updated.CategoryID != other.ID || updated.NameEn != "Mobilise" || len(updated.Targets) != 0 || updated.ID != d.ID {
		t.Fatalf("updated: %+v", updated)
	}
	e.expect("PUT", "/decisions/"+itoa(d.ID), `{"decision_id":"mobilize_1"}`, 409, "decision_id 'mobilize_1' already exists in this project")
	e.expect("PUT", "/decisions/"+itoa(d.ID), `{"category_id":999}`, 400, "Category not found in this project")
	e.expect("PUT", "/decisions/"+itoa(d.ID), `{"decision_id":null}`, 400, "decision_id cannot be null")
	e.expect("PUT", "/decisions/"+itoa(d.ID), `{"cost":"lots"}`, 422, "Invalid decision payload")
	e.expect("PUT", "/decisions/"+itoa(d.ID), `{"modifier":"a = \"x\\ny\""}`, 400, "请在第 1 行")
	e.expect("PUT", "/decisions/999", `{"cost":1}`, 404, "Decision not found")
	e.expect("GET", "/decisions/abc", "", 404, "Decision not found")
	e.expect("GET", "/decisions/"+itoa(d.ID)+"/nothing", "", 404, "Not Found")
	e.expect("GET", "/decisions/1/2/3/4", "", 404, "Project not found")
	e.expect("PATCH", "/decisions/"+itoa(d.ID), "", 405, "GET, PUT or DELETE only")

	e.expect("DELETE", "/decisions/"+itoa(d.ID), "", 204)
	e.expect("DELETE", "/decisions/"+itoa(d.ID), "", 404, "Decision not found")
}

func TestUpdateFlipsImportedDecisionToCreated(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("hkg_flip_category")
	var ids []int64
	if err := e.p.Write([]string{decisionsTable}, func(tx *project.Tx) error {
		for _, name := range []string{"explicit", "implicit", "kept"} {
			row := tx.Insert(decisionsTable, project.Row{
				"category_id": cat.ID, "decision_id": name, "source": "imported", "scope_kind": "own",
				"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
			})
			ids = append(ids, project.Int64(row["id"]))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for i, tc := range []struct{ body, want string }{
		{`{"source":"imported","name_en":"flip-explicit"}`, "created"},
		{`{"name_en":"flip-implicit"}`, "created"},
		{`{"source":"ai_generated","name_en":"keep"}`, "ai_generated"},
	} {
		got := decodeInto[Decision](t, e.expect("PUT", "/decisions/"+itoa(ids[i]), tc.body, 200))
		if got.Source != tc.want {
			t.Fatalf("case %d: source=%q want %q", i, got.Source, tc.want)
		}
	}
}

func TestForkDecision(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("war_plans")
	d := e.createDecision(cat.ID, "mobilize", `,"cost":25,"icon":"GFX_decision_x"`)
	e.createDecision(cat.ID, "taken", "")
	e.expect("POST", "/decisions/"+itoa(d.ID)+"/fork", `{"new_decision_id":"taken"}`, 409, "decision_id 'taken' already exists in this project")
	e.expect("POST", "/decisions/"+itoa(d.ID)+"/fork", `{"new_decision_id":"mobilize"}`, 400, "new_decision_id must differ from the original")
	e.expect("POST", "/decisions/"+itoa(d.ID)+"/fork", `{}`, 422, "new_decision_id is required")
	e.expect("POST", "/decisions/"+itoa(d.ID)+"/fork", `{"new_decision_id":"   "}`, 400, "new_decision_id cannot be empty")
	e.expect("POST", "/decisions/999/fork", `{"new_decision_id":"x"}`, 404, "Source decision not found")
	e.expect("GET", "/decisions/"+itoa(d.ID)+"/fork", "", 404, "Not Found")
	fork := decodeInto[Decision](t, e.expect("POST", "/decisions/"+itoa(d.ID)+"/fork", `{"new_decision_id":" mobilize_copy "}`, 201))
	if fork.ID == d.ID || fork.DecisionID != "mobilize_copy" || fork.Source != "created" || fork.Cost != 25 ||
		fork.Icon != "GFX_decision_x" || fork.CategoryID != cat.ID {
		t.Fatalf("fork: %+v", fork)
	}
}

func TestDecisionPreviews(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("war_plans")
	d := e.createDecision(cat.ID, "mobilize", `,"cost":25`)
	category, _ := getCategory(e.p, cat.ID)
	decision, _ := getDecision(e.p, d.ID)
	want, err := emitDecisionsPreview([]DecisionCategory{category}, []Decision{decision})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(want, "war_plans = {") || !strings.Contains(want, "mobilize = {") {
		t.Fatalf("decision emitter output is incomplete: %q", want)
	}
	for _, path := range []string{"/decisions/categories/" + itoa(cat.ID) + "/preview", "/decisions/" + itoa(d.ID) + "/preview"} {
		got := decodeInto[map[string]string](t, e.expect("GET", path, "", 200))
		if got["script"] != want {
			t.Fatalf("%s preview mismatch\nwant=%q\n got=%q", path, want, got["script"])
		}
	}
	e.expect("GET", "/decisions/999/preview", "", 404, "Decision not found")
	e.expect("GET", "/decisions/categories/999/preview", "", 404, "Category not found")
	e.expect("GET", "/decisions/categories/nope/preview", "", 422, `"loc":["path","cat_id"]`)
	e.expect("GET", "/decisions/nope/preview", "", 422, `"loc":["path","decision_db_id"]`)
	e.expect("GET", "/decisions/0/preview", "", 404, "Decision not found")
	e.expect("POST", "/decisions/"+itoa(d.ID)+"/preview", `{}`, 405, "Method Not Allowed")
}

func TestDecisionParsePreview(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("war_plans")
	d := e.createDecision(cat.ID, "mobilize", "")
	path := "/decisions/" + itoa(d.ID) + "/parse-preview"
	script := `war_plans = {
    icon = GFX_war_plans
    mobilize_now = {
        cost = 25
        available = { has_war = no }
        custom_effect = { add_war_support = 0.1 }
    }
}`
	body, _ := json.Marshal(map[string]string{"script": script})
	got := decodeInto[parsePreviewResponse](t, e.expect("POST", path, string(body), 200))
	if got.Entity == nil || len(got.ParseErrors) != 0 {
		t.Fatalf("unexpected parse result: %+v", got)
	}
	if got.Entity["decision_id"] != "mobilize_now" || got.Entity["category_id"] != "war_plans" || got.Entity["cost"] != float64(25) {
		t.Fatalf("decision identity mismatch: %#v", got.Entity)
	}
	raw, ok := got.Entity["raw_extras"].(map[string]any)
	if !ok || !strings.Contains(raw["custom_effect"].(string), "add_war_support") {
		t.Fatalf("raw_extras must remain apply-able: %#v", got.Entity["raw_extras"])
	}
	if _, leaked := got.Extras["raw_extras"]; leaked {
		t.Fatalf("raw_extras classified as parser metadata: %#v", got.Extras)
	}

	empty := decodeInto[parsePreviewResponse](t, e.expect("POST", path, `{"script":"  \n"}`, 200))
	if empty.Entity != nil || len(empty.ParseErrors) != 1 || empty.ParseErrors[0].Severity != "info" || empty.ParseErrors[0].Msg != "Script is empty" {
		t.Fatalf("empty: %+v", empty)
	}
	multi, _ := json.Marshal(map[string]string{"script": "war_plans = {\n    first = { cost = 1 }\n    second = { cost = 2 }\n}"})
	many := decodeInto[parsePreviewResponse](t, e.expect("POST", path, string(multi), 200))
	want := "Script defined 2 decisions; only the first (first) was used. Apply this and edit additional decisions separately."
	if many.Entity["decision_id"] != "first" || len(many.Warnings) != 1 || many.Warnings[0] != want {
		t.Fatalf("multi: %+v", many)
	}
	e.expect("POST", path, `{}`, 422, `"loc":["body","script"]`)
	e.expect("POST", path, `{"script":null}`, 422, `"type":"string_type"`)
	e.expect("POST", path, ``, 422, `"type":"missing"`)
	e.expect("POST", "/decisions/999/parse-preview", `{"script":"x = {}"}`, 404, "Decision not found")
	e.expect("GET", path, "", 405, "Method Not Allowed")
}

func TestCanonicalBlockTextBoundaries(t *testing.T) {
	in := decisionCreateInput{
		Allowed: "tag = GER\\nhas_war = no", Modifier: "stability_factor = 0.1\\nwar_support_factor = 0.1",
		Activation: "has_war = no\\nhas_government = democratic", TimeoutEffect: "add_stability = -0.1\\nadd_war_support = -0.1",
		TargetRootTrigger: "is_ai = no\\nis_major = yes", CustomCostTrigger: "has_political_power > 10\\nhas_war = no",
		NameEn: "Player\\nFacing", DisplayNameLocs: map[string]string{"en": "Player\\nFacing"},
	}
	if err := canonicalDecisionCreate(&in); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"allowed": in.Allowed, "modifier": in.Modifier, "activation": in.Activation,
		"timeout_effect": in.TimeoutEffect, "target_root_trigger": in.TargetRootTrigger,
		"custom_cost_trigger": in.CustomCostTrigger,
	} {
		if strings.Contains(value, `\n`) || !strings.Contains(value, "\n") {
			t.Errorf("%s was not canonicalised: %q", name, value)
		}
	}
	if in.NameEn != `Player\nFacing` || in.DisplayNameLocs["en"] != `Player\nFacing` {
		t.Fatalf("ordinary/localised text changed: %#v", in)
	}

	category := categoryCreateInput{Allowed: "tag = GER\\nhas_war = no", OnMapArea: "state = 1\\nstate = 2"}
	if err := canonicalCategoryCreate(&category); err != nil {
		t.Fatal(err)
	}
	if category.Allowed != "tag = GER\nhas_war = no" || category.OnMapArea != "state = 1\nstate = 2" {
		t.Fatalf("category blocks not canonicalised: %#v", category)
	}

	opt := func(v string) workbench.Optional[string] { return workbench.Optional[string]{Set: true, Value: &v} }
	patch := decisionPatch{Allowed: opt("tag = GER\\nhas_war = no"), Visible: opt(`has_country_flag = "literal\nvalue"`)}
	before := *patch.Allowed.Value
	if err := canonicalDecisionPatch(&patch); err == nil {
		t.Fatal("an ambiguous quoted literal newline must be rejected")
	}
	if *patch.Allowed.Value != before {
		t.Fatalf("patch was partially changed before rejection: %q", *patch.Allowed.Value)
	}
	if out, _, err := canonicalBlockText("# keep \\n in comments\nx = 1"); err != nil || out != "# keep \\n in comments\nx = 1" {
		t.Fatalf("comment changed: %q %v", out, err)
	}
}

func testDDS(w, h int) []byte {
	pix := make([]byte, w*h*4)
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+1], pix[i+2], pix[i+3] = 200, 40, 40, 255
	}
	return imgcodec.EncodeDXT5DDS(pix, w, h)
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.NRGBA{10, 120, 200, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *testEnv) upload(path string, data []byte, fields map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if data != nil {
		part, _ := form.CreateFormFile("file", "icon.png")
		part.Write(data)
	}
	for k, v := range fields {
		form.WriteField(k, v)
	}
	form.Close()
	req := httptest.NewRequest("POST", "/api/go/projects/1"+path, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func assertPNG(t *testing.T, rec *httptest.ResponseRecorder, width, height int) {
	t.Helper()
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("status=%d type=%q body=%.200s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != width || b.Dy() != height {
		t.Fatalf("picture is %dx%d, want %dx%d", b.Dx(), b.Dy(), width, height)
	}
}

func TestUploadedIcons(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("War Plans")
	d := e.createDecision(cat.ID, "Mobilize Now", "")

	e.expect("GET", "/decisions/"+itoa(d.ID)+"/icon", "", 404, "Asset not uploaded")
	e.expect("GET", "/decisions/999/icon", "", 404, "Asset not uploaded")
	e.expect("GET", "/decisions/abc/icon", "", 404, "Asset target not found")
	e.expect("GET", "/decisions/categories/icon", "", 404, "Asset target not found")
	e.expect("GET", "/decisions/"+itoa(d.ID)+"/upload-icon", "", 405, "POST only")
	e.expect("POST", "/decisions/"+itoa(d.ID)+"/icon", "", 405, "GET only")
	if rec := e.upload("/decisions/999/upload-icon", testPNG(t), nil); rec.Code != 404 {
		t.Fatalf("missing decision upload: %d %s", rec.Code, rec.Body.String())
	}
	if rec := e.upload("/decisions/"+itoa(d.ID)+"/upload-icon", nil, nil); rec.Code != 422 || !strings.Contains(rec.Body.String(), "file is required") {
		t.Fatalf("no file: %d %s", rec.Code, rec.Body.String())
	}
	if rec := e.upload("/decisions/"+itoa(d.ID)+"/upload-icon", []byte("not an image"), nil); rec.Code != 400 || !strings.Contains(rec.Body.String(), "Failed to decode image") {
		t.Fatalf("bad image: %d %s", rec.Code, rec.Body.String())
	}
	if rec := e.upload("/decisions/"+itoa(d.ID)+"/upload-icon", testPNG(t), map[string]string{"crop_x": "x"}); rec.Code != 422 || !strings.Contains(rec.Body.String(), "invalid crop coordinates") {
		t.Fatalf("bad crop: %d %s", rec.Code, rec.Body.String())
	}

	rec := e.upload("/decisions/"+itoa(d.ID)+"/upload-icon", testPNG(t), map[string]string{"crop_x": "0", "crop_y": "0", "crop_width": "40", "crop_height": "40"})
	if rec.Code != 200 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	got := decodeInto[map[string]any](t, rec)
	if got["icon"] != "GFX_decision_mobilize_now" || got["filename"] != "decision_mobilize_now.png" ||
		got["preview_url"] != "/api/go/projects/1/decisions/"+itoa(d.ID)+"/icon" {
		t.Fatalf("upload response: %v", got)
	}
	stored, _ := getDecision(e.p, d.ID)
	if stored.Icon != "GFX_decision_mobilize_now" {
		t.Fatalf("decision icon not set: %+v", stored)
	}
	row, _ := findDecisionRow(e.p, d.ID)
	if row["has_obs"] != true {
		t.Fatalf("has_obs not set: %v", row["has_obs"])
	}
	paths, _ := e.p.Rows("entity_obs_paths")
	if len(paths) != 1 || paths[0]["entity_type"] != "decision_icon" || !strings.HasPrefix(text(paths[0]["obs_key"]), "local/p/u1/upload/decision/") {
		t.Fatalf("mapping: %v", paths)
	}
	assertPNG(t, e.do("GET", "/decisions/"+itoa(d.ID)+"/icon", ""), 33, 32)
	assertPNG(t, e.do("GET", "/decision-icons/GFX_decision_mobilize_now", ""), 33, 32)
	e.expect("GET", "/decision-icons", "", 200, `{"icons":["GFX_decision_mobilize_now"]}`)
	e.expect("GET", "/decision-icons/GFX_nothing", "", 404, "Decision icon not found")

	if rec := e.upload("/decisions/"+itoa(d.ID)+"/upload-icon", testPNG(t), nil); rec.Code != 200 {
		t.Fatalf("re-upload: %d %s", rec.Code, rec.Body.String())
	}
	if paths, _ := e.p.Rows("entity_obs_paths"); len(paths) != 1 {
		t.Fatalf("mappings after re-upload: %v", paths)
	}

	rec = e.upload("/decisions/categories/"+itoa(cat.ID)+"/upload-icon", testPNG(t), nil)
	if rec.Code != 200 {
		t.Fatalf("category upload: %d %s", rec.Code, rec.Body.String())
	}
	got = decodeInto[map[string]any](t, rec)
	if got["icon"] != "GFX_decision_category_war_plans" || got["preview_url"] != "/api/go/projects/1/decisions/categories/"+itoa(cat.ID)+"/icon" {
		t.Fatalf("category upload response: %v", got)
	}
	assertPNG(t, e.do("GET", "/decisions/categories/"+itoa(cat.ID)+"/icon", ""), 52, 40)
	assertPNG(t, e.do("GET", "/decision-icons/GFX_decision_category_war_plans", ""), 52, 40)
	if c, _ := getCategory(e.p, cat.ID); c.Icon != "GFX_decision_category_war_plans" {
		t.Fatalf("category icon not set: %+v", c)
	}
}

func TestImportedIconMappings(t *testing.T) {
	e := newEnv(t, nil)
	cat := e.createCategory("war_plans")
	d := e.createDecision(cat.ID, "mobilize", `,"icon":"GFX_decision_imported"`)
	e.expect("PUT", "/decisions/categories/"+itoa(cat.ID), `{"icon":"GFX_decision_category_imported"}`, 200)
	blobs := e.p.Blobs()
	put := func(key string) {
		data := testPNG(t)
		if err := blobs.Put(key, "image/png", int64(len(data)), bytes.NewReader(data)); err != nil {
			t.Fatal(err)
		}
	}
	put("local/p/u1/upload/import/a.png")
	put("local/p/u1/upload/import/b.png")
	if err := e.p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		tx.Insert("project_obs_assets", project.Row{"asset_type": "decision_icon", "gfx_name": "GFX_decision_imported", "obs_key": "local/p/u1/upload/import/a.png"})

		tx.Insert("project_obs_assets", project.Row{"asset_type": "decision_icon", "gfx_name": "GFX_decision_category_imported", "obs_key": "local/p/u1/upload/import/b.png"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assertPNG(t, e.do("GET", "/decisions/"+itoa(d.ID)+"/icon", ""), 40, 40)
	assertPNG(t, e.do("GET", "/decisions/categories/"+itoa(cat.ID)+"/icon", ""), 40, 40)
	assertPNG(t, e.do("GET", "/decision-icons/GFX_decision_imported", ""), 40, 40)
	e.expect("GET", "/decision-icons", "", 200, `"GFX_decision_category_imported"`, `"GFX_decision_imported"`)
}

func TestBaseGameIconsAndCategoryWarning(t *testing.T) {
	game := fstest.MapFS{
		"interface/decisions.gfx": {Data: []byte(`spriteTypes = {
	spriteType = {
		name = "GFX_decision_generic_industry"
		texturefile = "gfx/interface/decisions/decision_generic_industry.dds"
	}
	spriteType = {
		name = "GFX_decision_category_generic_industry"
		texturefile = "gfx/interface/decisions/decision_category_generic_industry.dds"
	}
}`)},
		"gfx/interface/decisions/decision_generic_industry.dds":          {Data: testDDS(32, 32)},
		"gfx/interface/decisions/decision_category_generic_industry.dds": {Data: testDDS(52, 40)},
		"common/decisions/categories/00_categories.txt": {Data: []byte(`political_actions = {
	icon = generic_political_actions
}
economy_decisions = { icon = generic_industry }
`)},
	}
	e := newEnv(t, game)
	rec := e.expect("POST", "/decisions/categories", `{"category_id":"political_actions","country_tag":"tst"}`, 201)
	cat := decodeInto[DecisionCategory](t, rec)
	if len(cat.Warnings) != 1 || !strings.Contains(cat.Warnings[0], "建议改名 'TST_political_actions'") {
		t.Fatalf("vanilla category warning: %+v", cat.Warnings)
	}
	if fresh := e.createCategory("TST_own"); len(fresh.Warnings) != 0 {
		t.Fatalf("own category warned: %+v", fresh.Warnings)
	}

	d := e.createDecision(cat.ID, "build", `,"icon":"generic_industry"`)
	assertPNG(t, e.do("GET", "/decisions/"+itoa(d.ID)+"/icon", ""), 32, 32)
	e.expect("PUT", "/decisions/categories/"+itoa(cat.ID), `{"icon":"generic_industry"}`, 200)
	assertPNG(t, e.do("GET", "/decisions/categories/"+itoa(cat.ID)+"/icon", ""), 52, 40)
	assertPNG(t, e.do("GET", "/decision-icons/generic_industry", ""), 32, 32)
	assertPNG(t, e.do("GET", "/decision-icons/GFX_decision_category_generic_industry", ""), 52, 40)
	e.expect("GET", "/decision-icons/GFX_decision_unknown", "", 404, "Decision icon not found")

	other := e.createDecision(cat.ID, "plain", `,"icon":"generic_unknown"`)
	e.expect("GET", "/decisions/"+itoa(other.ID)+"/icon", "", 404, "Asset not uploaded")
}
