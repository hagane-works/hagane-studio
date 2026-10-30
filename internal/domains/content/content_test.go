package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
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

func (e *testEnv) expect(method, path, body string, status int, fragments ...string) map[string]any {
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
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return out
}

func (e *testEnv) list(path string) []any {
	e.t.Helper()
	rec := e.do(http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		e.t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
	}
	var out any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		e.t.Fatal(err)
	}
	if items, ok := out.([]any); ok {
		return items
	}
	return out.(map[string]any)["items"].([]any)
}

func (e *testEnv) seed(table string, row project.Row) int64 {
	e.t.Helper()
	var id int64
	if err := e.p.Write([]string{table}, func(tx *project.Tx) error {
		id = project.Int64(tx.Insert(table, row)["id"])
		return nil
	}); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *testEnv) row(table string, id int64) project.Row {
	e.t.Helper()
	rows, err := e.p.Rows(table)
	if err != nil {
		e.t.Fatal(err)
	}
	return findRow(rows, id)
}

func idOf(t *testing.T, item map[string]any) string {
	t.Helper()
	id, ok := item["id"].(float64)
	if !ok {
		t.Fatalf("no id in %v", item)
	}
	return fmt.Sprintf("/%d", int64(id))
}

func jsonString(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func pngUpload(t *testing.T, path string) *http.Request {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for x := 0; x < 40; x++ {
		for y := 0; y < 40; y++ {
			img.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	var pic bytes.Buffer
	if err := png.Encode(&pic, img); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("file", "logo.png")
	_, _ = part.Write(pic.Bytes())
	_ = form.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/go/projects/1"+path, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

func TestRoutesAnswerTheWebVersionsMethodAndPathErrors(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPatch, "/dynamic-modifiers", "", http.StatusMethodNotAllowed, `{"error":"GET or POST only"}`)
	e.expect(http.MethodPost, "/dynamic-modifiers/1", "", http.StatusMethodNotAllowed, `"GET, PUT or DELETE only"`)
	e.expect(http.MethodGet, "/dynamic-modifiers/abc", "", http.StatusNotFound, `"DynamicModifier not found"`)
	e.expect(http.MethodGet, "/dynamic-modifiers/1/x", "", http.StatusNotFound, `"DynamicModifier not found"`)
	e.expect(http.MethodGet, "/ai-strategy-plans/1", "", http.StatusMethodNotAllowed, `"PUT or DELETE only"`)
	e.expect(http.MethodPatch, "/super-events", "", http.StatusMethodNotAllowed, `"error":"GET or POST only"`, `"detail":"GET or POST only"`)
	e.expect(http.MethodGet, "/super-events/1/nope", "", http.StatusNotFound, `"Not Found"`)
	e.expect(http.MethodGet, "/special-projects", "", http.StatusNotFound, `"Not Found"`)
	e.expect(http.MethodGet, "/special-projects/other", "", http.StatusNotFound, `"Not Found"`)
	e.expect(http.MethodPost, "/scripted-snippets", "not json", http.StatusUnprocessableEntity, `"Invalid payload"`)
}

func TestDynamicModifierCRUD(t *testing.T) {
	game := fstest.MapFS{
		"common/dynamic_modifiers/vanilla.txt": {Data: []byte("\xef\xbb\xbfGER_war_effort = {\n\tenable = { always = yes }\n}\n")},
	}
	e := newEnv(t, game)
	e.expect(http.MethodPost, "/dynamic-modifiers", `{"dm_id":"  "}`, http.StatusUnprocessableEntity, `"dm_id required"`)

	first := e.expect(http.MethodPost, "/dynamic-modifiers", `{"dm_id":" my_dm ","country_tag":"ger","modifier":{"stability_factor":0.1}}`, http.StatusCreated,
		`"dm_id":"my_dm"`, `"country_tag":"GER"`, `"source":"created"`, `"warnings":[]`, `"stability_factor":0.1`, `"display_name_locs":{}`)
	second := e.expect(http.MethodPost, "/dynamic-modifiers", `{"dm_id":"my_dm"}`, http.StatusCreated,
		`"dm_id":"my_dm_2"`, `dm_id 'my_dm' collided · auto-suffixed to 'my_dm_2'.`)
	e.expect(http.MethodPost, "/dynamic-modifiers", `{"dm_id":"GER_war_effort","country_tag":"GER"}`, http.StatusCreated,
		`跟 vanilla dynamic_modifier 同名`, `建议改名 'GER_war_effort'`)
	if row := e.row(dynamicModifiersTable, 1); row["description_locs"] == nil {
		t.Fatalf("description_locs default missing: %v", row)
	}

	body := e.expect(http.MethodGet, "/dynamic-modifiers", "", http.StatusOK, `"total":3`)
	if len(body["dynamic_modifiers"].([]any)) != 3 {
		t.Fatalf("envelope: %v", body)
	}
	e.expect(http.MethodGet, "/dynamic-modifiers"+idOf(t, first), "", http.StatusOK, `"dm_id":"my_dm"`)

	e.expect(http.MethodPut, "/dynamic-modifiers"+idOf(t, first), `{"name_en":"War","icon":null}`, http.StatusOK, `"name_en":"War"`, `"icon":""`)

	e.expect(http.MethodPut, "/dynamic-modifiers"+idOf(t, second), `{"dm_id":"my_dm"}`, http.StatusServiceUnavailable, `"store unavailable"`)
	e.expect(http.MethodPut, "/dynamic-modifiers"+idOf(t, second), `{"display_name_locs":null}`, http.StatusServiceUnavailable)

	imported := e.seed(dynamicModifiersTable, project.Row{"dm_id": "imp", "display_name_locs": map[string]any{}, "source": "imported"})
	path := "/dynamic-modifiers/" + jsonString(imported)
	e.expect(http.MethodPut, path, `{"source":"imported","enable":"always = yes"}`, http.StatusOK, `"source":"created"`)
	e.expect(http.MethodPut, path, `{"source":"custom"}`, http.StatusOK, `"source":"custom"`)

	e.expect(http.MethodDelete, path, "", http.StatusNoContent)
	e.expect(http.MethodDelete, path, "", http.StatusNotFound, `"DynamicModifier not found"`)
}

func TestAutonomyStateCRUD(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/autonomy-states", `{}`, http.StatusUnprocessableEntity, `"autonomy_id required"`)
	item := e.expect(http.MethodPost, "/autonomy-states", `{"autonomy_id":"my_dominion","is_puppet":true,"manpower_influence":0.5,"min_freedom_level":2}`, http.StatusCreated,
		`"is_puppet":true`, `"manpower_influence":0.5`, `"min_freedom_level":2`, `"ai_weights":{}`)
	e.expect(http.MethodPost, "/autonomy-states", `{"autonomy_id":"my_dominion","source":""}`, http.StatusCreated,
		`"autonomy_id":"my_dominion_2"`, `"source":""`, `autonomy_id 'my_dominion' collided`)
	e.expect(http.MethodPut, "/autonomy-states"+idOf(t, item), `{"manpower_influence":null,"rule":"x"}`, http.StatusOK,
		`"manpower_influence":null`, `"rule":"x"`)
	e.expect(http.MethodGet, "/autonomy-states", "", http.StatusOK, `"autonomy_states":[`, `"total":2`)
	e.expect(http.MethodDelete, "/autonomy-states"+idOf(t, item), "", http.StatusNoContent)
	e.expect(http.MethodGet, "/autonomy-states"+idOf(t, item), "", http.StatusNotFound, `"AutonomyState not found"`)
}

func TestScriptedSnippetsAndLocalisations(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/scripted-snippets", `{"name":"x","kind":"macro"}`, http.StatusBadRequest,
		`kind must be one of ('effect', 'trigger'), got 'macro'`)
	e.expect(http.MethodPost, "/scripted-snippets", `{"name":"  "}`, http.StatusBadRequest, `name is required (e.g. 'my_effect_name').`)
	effect := e.expect(http.MethodPost, "/scripted-snippets", `{"name":" zeta "}`, http.StatusCreated, `"kind":"effect"`, `"name":"zeta"`, `"raw_extras":null`)
	e.expect(http.MethodPost, "/scripted-snippets", `{"name":"alpha","kind":"trigger","raw_extras":{"a":1}}`, http.StatusCreated, `"raw_extras":{"a":1}`)
	e.expect(http.MethodPost, "/scripted-snippets", `{"name":"alpha","kind":"effect"}`, http.StatusCreated)
	e.expect(http.MethodPost, "/scripted-snippets", `{"name":"zeta"}`, http.StatusConflict,
		`A scripted effect named 'zeta' already exists in this project. Pick a unique name.`)

	items := e.list("/scripted-snippets")
	var order []string
	for _, item := range items {
		m := item.(map[string]any)
		order = append(order, m["kind"].(string)+":"+m["name"].(string))
	}
	if strings.Join(order, ",") != "effect:alpha,effect:zeta,trigger:alpha" {
		t.Fatalf("order: %v", order)
	}
	if got := e.list("/scripted-snippets?kind=trigger"); len(got) != 1 {
		t.Fatalf("kind filter: %v", got)
	}
	e.expect(http.MethodGet, "/scripted-snippets?kind=trigger", "", http.StatusOK, `"snippets":[`, `"total":1`)

	e.expect(http.MethodPut, "/scripted-snippets"+idOf(t, effect), `{"name":" "}`, http.StatusBadRequest, `"name cannot be empty"`)
	e.expect(http.MethodPut, "/scripted-snippets"+idOf(t, effect), `{"name":"alpha"}`, http.StatusConflict, `"kind + name conflict in this project"`)
	e.expect(http.MethodPut, "/scripted-snippets"+idOf(t, effect), `{"body":"add_stability = 0.1"}`, http.StatusOK, `"body":"add_stability = 0.1"`)

	e.expect(http.MethodPost, "/scripted-localisations", `{"name":""}`, http.StatusBadRequest, `name is required (e.g. 'GetMyDynamicText').`)
	loc := e.expect(http.MethodPost, "/scripted-localisations", `{"name":"GetB"}`, http.StatusCreated)
	e.expect(http.MethodPost, "/scripted-localisations", `{"name":"GetA"}`, http.StatusCreated)
	e.expect(http.MethodPost, "/scripted-localisations", `{"name":"GetA"}`, http.StatusConflict,
		`A scripted localisation named 'GetA' already exists in this project. Pick a unique name.`)
	e.expect(http.MethodPut, "/scripted-localisations"+idOf(t, loc), `{"name":"GetA"}`, http.StatusConflict, `"name conflict in this project"`)
	body := e.expect(http.MethodGet, "/scripted-localisations", "", http.StatusOK, `"total":2`)
	first := body["localisations"].([]any)[0].(map[string]any)
	if first["name"] != "GetA" {
		t.Fatalf("ordered by name: %v", body)
	}
}

func TestOpinionModifiers(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/opinion-modifiers", `{"value":10}`, http.StatusUnprocessableEntity, `"modifier_id is required"`)
	item := e.expect(http.MethodPost, "/opinion-modifiers", `{"modifier_id":"b_mod","value":10,"decay":1,"trade":true}`, http.StatusCreated,
		`"country_tag":"TST"`, `"value":10`, `"trade":true`, `"body":{`, `"trade":"yes"`, `"source":"created"`)
	e.expect(http.MethodPost, "/opinion-modifiers", `{"modifier_id":"a_mod","country_tag":"GER"}`, http.StatusCreated, `"trade":null`)
	e.expect(http.MethodPost, "/opinion-modifiers", `{"modifier_id":"a_mod"}`, http.StatusConflict,
		`Opinion modifier 'a_mod' already exists. Pick a unique modifier id.`)

	items := e.list("/opinion-modifiers")
	if len(items) != 2 || items[0].(map[string]any)["modifier_id"] != "a_mod" {
		t.Fatalf("bare list ordered by modifier_id: %v", items)
	}
	if got := e.list("/opinion-modifiers?country_tag=ger"); len(got) != 1 {
		t.Fatalf("country filter: %v", got)
	}
	e.expect(http.MethodPut, "/opinion-modifiers"+idOf(t, item), `{"modifier_id":"a_mod"}`, http.StatusConflict, `'a_mod' already exists`)
	updated := e.expect(http.MethodPut, "/opinion-modifiers"+idOf(t, item), `{"value":null,"trade":false,"months":3}`, http.StatusOK,
		`"trade":false`, `"months":3`, `"value":null`)
	if _, ok := updated["body"].(map[string]any)["value"]; ok {
		t.Fatalf("null must remove the body key: %v", updated["body"])
	}
	e.expect(http.MethodDelete, "/opinion-modifiers"+idOf(t, item), "", http.StatusNoContent)
	e.expect(http.MethodGet, "/opinion-modifiers"+idOf(t, item), "", http.StatusNotFound, `"Opinion modifier not found"`)
}

func TestModifierDefinitions(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/modifier-definitions", `{"modifier_id":"1bad"}`, http.StatusUnprocessableEntity, `modifier_id must start with a letter`)
	e.expect(http.MethodPost, "/modifier-definitions", `{"modifier_id":"ok","color_type":"pink"}`, http.StatusUnprocessableEntity, `color_type must be one of good/bad/neutral`)
	e.expect(http.MethodPost, "/modifier-definitions", `{"modifier_id":"ok","categories":["army","bogus"]}`, http.StatusUnprocessableEntity, `unknown modifier category 'bogus'`)
	item := e.expect(http.MethodPost, "/modifier-definitions", `{"modifier_id":"my_mod","categories":["army","army","naval"]}`, http.StatusCreated,
		`"color_type":"bad"`, `"value_type":"number"`, `"precision":2`, `"postfix":"none"`, `"categories":["army","naval"]`,
		`"explicit_fields":["color_type","value_type","precision","postfix","category"]`)
	e.expect(http.MethodPost, "/modifier-definitions", `{"modifier_id":"my_mod"}`, http.StatusConflict, `"Modifier definition ID already exists"`)

	imported := e.seed(modifierDefinitionsTable, project.Row{
		"modifier_id": "imp_mod", "color_type": "good", "value_type": "number", "precision": int64(1),
		"postfix": "none", "categories": []any{"all"}, "source_file": "x.txt",
		"explicit_fields": []any{"category"}, "source": "imported",
	})
	e.expect(http.MethodPut, "/modifier-definitions/"+jsonString(imported), `{"postfix":"days"}`, http.StatusOK,
		`"explicit_fields":["postfix","category"]`, `"source":"created"`, `"postfix":"days"`)
	e.expect(http.MethodPut, "/modifier-definitions/"+jsonString(imported), `{"modifier_id":"my_mod"}`, http.StatusConflict)
	e.expect(http.MethodPut, "/modifier-definitions"+idOf(t, item), `{"precision":9}`, http.StatusUnprocessableEntity, `precision must be between 0 and 3`)
	e.expect(http.MethodGet, "/modifier-definitions", "", http.StatusOK, `"modifier_definitions":[`, `"total":2`)
}

func TestWargoals(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/wargoals", `{"wargoal_key":"two words"}`, http.StatusUnprocessableEntity, `wargoal_key must be one HOI4 token`)
	e.expect(http.MethodPost, "/wargoals", `{"wargoal_key":"x","advanced_raw_script":"allowed = { }"}`, http.StatusUnprocessableEntity,
		`allowed belongs in the visual fields, not Advanced raw script`)
	item := e.expect(http.MethodPost, "/wargoals", `{"wargoal_key":"annex_all","safe_scalars":{"expire":" 5 "},"allowed":"always = yes","advanced_raw_script":"custom_key = { a = b }","source_file":"common\\wargoals\\x.txt"}`,
		http.StatusCreated, `"safe_scalars":{"expire":"5"}`, `"advanced_raw_script":"custom_key = {`, `"source_file":"common/wargoals/x.txt"`)
	e.expect(http.MethodPost, "/wargoals", `{"wargoal_key":"annex_all"}`, http.StatusConflict, `Wargoal 'annex_all' already exists. Pick a unique wargoal id.`)
	e.expect(http.MethodPut, "/wargoals"+idOf(t, item), `{"allowed":null}`, http.StatusUnprocessableEntity, `"Wargoal update fields cannot be null"`)
	e.expect(http.MethodPut, "/wargoals"+idOf(t, item), `{"advanced_raw_script":""}`, http.StatusOK, `"advanced_raw_script":""`)
	e.expect(http.MethodGet, "/wargoals", "", http.StatusOK, `"wargoals":[`, `"total":1`)
	e.expect(http.MethodDelete, "/wargoals"+idOf(t, item), "", http.StatusNoContent)
	e.expect(http.MethodGet, "/wargoals"+idOf(t, item), "", http.StatusNotFound, `"Wargoal not found"`)
}

func TestAIPlanning(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/ai-strategy-plans", `{"plan_id":"x","bogus":1}`, http.StatusUnprocessableEntity, `unknown field`)
	plan := e.expect(http.MethodPost, "/ai-strategy-plans", `{"plan_id":"  my_plan  ","name":"Plan"}`, http.StatusCreated,
		`"plan_id":"my_plan"`, `"weight":["{\n\tfactor = 1\n}"]`, `"field_order":["name","desc","allowed","enable","abort","weight"]`, `"source":"created"`)
	if created, _ := plan["created_at"].(string); !strings.HasSuffix(created, "Z") {
		t.Fatalf("created_at must be RFC 3339 UTC: %v", plan["created_at"])
	}
	e.expect(http.MethodPost, "/ai-strategy-plans", `{"plan_id":"my_plan"}`, http.StatusConflict, `"AI strategy plan ID already exists"`)
	e.expect(http.MethodPut, "/ai-strategy-plans"+idOf(t, plan), `{"weight":null}`, http.StatusUnprocessableEntity, `"AI strategy plan update fields cannot be null"`)
	e.expect(http.MethodPut, "/ai-strategy-plans"+idOf(t, plan), `{"enable":["{ has_war = yes }"]}`, http.StatusOK, `"enable":["{ has_war = yes }"]`)
	e.expect(http.MethodGet, "/ai-strategy-plans", "", http.StatusOK, `"ai_strategy_plans":[`, `"total":1`)

	imported := e.seed(aiStrategyPlansTable, project.Row{
		"plan_id": "imp", "name": "", "desc": "", "allowed": []any{"tag = GER"}, "enable": []any{}, "abort": []any{},
		"ai_national_focuses": []any{}, "focus_factors": []any{}, "research": []any{}, "ideas": []any{},
		"traits": []any{}, "ai_strategy": []any{}, "weight": []any{"factor = 1"}, "field_order": []any{"allowed"},
		"raw_block": "imp = { }", "source_file": "x.txt", "source": "imported",
	})
	e.expect(http.MethodGet, "/ai-strategy-plans", "", http.StatusOK, `"allowed":["{\ntag = GER\n}"]`)
	e.expect(http.MethodPut, "/ai-strategy-plans/"+jsonString(imported), `{"name":"Imp"}`, http.StatusOK, `"raw_block":""`, `"source":"created"`)

	template := e.expect(http.MethodPost, "/ai-templates", `{"template_id":"generic_role"}`, http.StatusCreated,
		`"role":"infantry"`, `"id":"infantry_default"`, `"field_order":["role","upgrade_prio","target:infantry_default"]`)
	e.expect(http.MethodPost, "/ai-templates", `{"template_id":"x","available_for":"GER"}`, http.StatusUnprocessableEntity, `"available_for must be one brace block"`)
	e.expect(http.MethodPost, "/ai-templates", `{"template_id":"generic_role"}`, http.StatusConflict, `"AI template ID already exists"`)
	e.expect(http.MethodPut, "/ai-templates"+idOf(t, template), `{"role":"armor"}`, http.StatusOK, `"role":"armor"`)
	e.expect(http.MethodGet, "/ai-templates", "", http.StatusOK, `"ai_templates":[`, `"total":1`)
	e.expect(http.MethodDelete, "/ai-templates"+idOf(t, template), "", http.StatusNoContent)
	e.expect(http.MethodDelete, "/ai-templates"+idOf(t, template), "", http.StatusNotFound, `"AI template not found"`)
}

func TestIntelligenceAgenciesAndLogo(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/intelligence-agencies", `{"agency_id":" "}`, http.StatusBadRequest, `agency_id is required (e.g. 'usa_oss')`)
	item := e.expect(http.MethodPost, "/intelligence-agencies", `{"agency_id":"tst_secret","names":["Secret Service"]}`, http.StatusCreated,
		`"scope_kind":"own"`, `"names":["Secret Service"]`, `"shared_country_tags":null`, `"upgrade_branches":[]`)
	e.expect(http.MethodPost, "/intelligence-agencies", `{"agency_id":"tst_secret"}`, http.StatusConflict,
		`agency_id 'tst_secret' already exists in this project. Pick a unique agency_id.`)
	path := "/intelligence-agencies" + idOf(t, item)
	e.expect(http.MethodPut, path, `{"agency_id":""}`, http.StatusBadRequest, `"agency_id cannot be empty"`)
	e.expect(http.MethodGet, "/intelligence-agencies", "", http.StatusOK, `"agencies":[`, `"total":1`)

	e.expect(http.MethodGet, path+"/logo", "", http.StatusNotFound, `"Asset not uploaded"`)
	e.expect(http.MethodGet, "/intelligence-agencies/99/upload-logo", "", http.StatusMethodNotAllowed, `"POST only"`)
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, pngUpload(t, "/intelligence-agencies/99/upload-logo"))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Asset target not found") {
		t.Fatalf("upload to a missing agency: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	e.mux.ServeHTTP(rec, pngUpload(t, path+"/upload-logo"))
	if rec.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	for _, fragment := range []string{`"picture":"GFX_intelligence_agency_logo_tst_secret"`, `"filename":"intel_agency_tst_secret.png"`, `/intelligence-agencies/1/logo"`} {
		if !strings.Contains(rec.Body.String(), fragment) {
			t.Fatalf("upload response %s lacks %s", rec.Body.String(), fragment)
		}
	}
	if e.row(agenciesTable, 1)["picture"] != "GFX_intelligence_agency_logo_tst_secret" {
		t.Fatal("agency picture not pointed at the logo")
	}
	logo := e.do(http.MethodGet, path+"/logo", "")
	if logo.Code != http.StatusOK || logo.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("logo: %d %s", logo.Code, logo.Header())
	}
	if img, err := png.Decode(logo.Body); err != nil || img.Bounds().Dx() != 117 {
		t.Fatalf("logo is not the 117x119 slot: %v %v", err, img)
	}
	e.expect(http.MethodDelete, path, "", http.StatusNoContent)
}

func TestSuperEventsAndPicture(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/super-events", `{"country_tag":"TST"}`, http.StatusUnprocessableEntity, `"number is required"`)
	item := e.expect(http.MethodPost, "/super-events", `{"number":1,"country_tag":"TST","quote_locs":{"en":" Hi "},"options":[{"display_text":"Go"}]}`, http.StatusCreated,
		`"quote_en":"Hi"`, `"quote_locs":{"en":"Hi"}`, `"display_name_locs":{"en":"Go"}`, `"name_en":"Go"`)
	e.expect(http.MethodPost, "/super-events", `{"number":1,"country_tag":"TST"}`, http.StatusConflict,
		`superevent number 1 already exists for this country in the project. Pick a unique 4-digit number.`)
	other := e.expect(http.MethodPost, "/super-events", `{"number":1,"country_tag":"GER"}`, http.StatusCreated)
	e.expect(http.MethodPut, "/super-events"+idOf(t, other), `{"country_tag":"TST"}`, http.StatusConflict, `"superevent number conflict"`)
	e.expect(http.MethodPut, "/super-events"+idOf(t, item), `{"btn_en":"OK"}`, http.StatusOK, `"button_locs":{"en":"OK"}`, `"btn_en":"OK"`)
	e.expect(http.MethodGet, "/super-events", "", http.StatusOK, `"super_events":[`, `"total":2`)

	path := "/super-events" + idOf(t, item)
	e.expect(http.MethodGet, path+"/picture", "", http.StatusNotFound, `"Asset not uploaded"`)
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, pngUpload(t, path+"/upload-picture"))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"picture_path":"super_events/SE_0001.png"`) {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	if e.row(superEventsTable, 1)["has_obs"] != true {
		t.Fatal("has_obs not set")
	}
	picture := e.do(http.MethodGet, path+"/picture", "")
	if img, err := png.Decode(picture.Body); err != nil || img.Bounds().Dx() != 460 {
		t.Fatalf("picture is not the 460x310 slot: %v %v", err, img)
	}
	e.expect(http.MethodDelete, path, "", http.StatusNoContent)
	e.expect(http.MethodGet, path, "", http.StatusNotFound, `"SuperEvent not found"`)
}

func TestSpecialProjects(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/special-projects/specializations", `{"specialization_id":" "}`, http.StatusBadRequest, `"specialization_id cannot be empty"`)
	spec := e.expect(http.MethodPost, "/special-projects/specializations", `{"specialization_id":"spec_nuclear"}`, http.StatusCreated,
		`"country_tag":"TST"`, `"name_en":"spec_nuclear"`, `"color":[]`, `"sub_specializations":[]`, `"raw_extras":{}`)
	e.expect(http.MethodPost, "/special-projects/specializations", `{"specialization_id":"spec_nuclear"}`, http.StatusConflict, `Specialization 'spec_nuclear' already exists`)
	e.expect(http.MethodGet, "/special-projects/specializations"+idOf(t, spec), "", http.StatusMethodNotAllowed, `"PUT or DELETE only"`)
	e.expect(http.MethodPut, "/special-projects/specializations"+idOf(t, spec), `{"country_tag":" ger ","color":[1,2,3]}`, http.StatusOK,
		`"country_tag":"GER"`, `"color":[1,2,3]`)
	if got := e.list("/special-projects/specializations?country_tag=ger"); len(got) != 1 {
		t.Fatalf("specialization filter: %v", got)
	}

	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_bad"}`, http.StatusBadRequest, `"specialization cannot be empty"`)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_bad","specialization":"spec_nuclear","complexity":"25 days"}`, http.StatusUnprocessableEntity, `"complexity must be one unquoted positive integer or script constant"`)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_bad","specialization":"spec_nuclear","prototype_time":"123bad"}`, http.StatusUnprocessableEntity, `"prototype_time must be one unquoted positive integer or script constant"`)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_bad","specialization":"spec_nuclear","resource_cost":"steel = 10"}`, http.StatusUnprocessableEntity, `"resource_cost must contain a resources = { ... } block"`)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_bad","specialization":"spec_nuclear","project_output":"123bad"}`, http.StatusUnprocessableEntity, `project_output contains an invalid token`)
	a := e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_a","specialization":"spec_nuclear","complexity":"sp_complexity.medium","prototype_time":"30","resource_cost":"resources = { steel = 10 }","project_output":"country_effects = { add_stability = 0.05 }","x":"3","y":4.0}`, http.StatusCreated,
		`"x":3`, `"y":4`, `"special_project_parent":[]`, `"name_en":"sp_a"`)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_b","specialization":"spec_nuclear","special_project_parent":["sp_a","sp_missing"]}`, http.StatusCreated)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_lone","specialization":"spec_nuclear"}`, http.StatusCreated)
	e.expect(http.MethodPost, "/special-projects/projects", `{"project_token":"sp_a","specialization":"spec_nuclear"}`, http.StatusConflict, `Special project 'sp_a' already exists`)
	if got := e.list("/special-projects/projects?specialization=spec_nuclear"); len(got) != 3 {
		t.Fatalf("specialization filter: %v", got)
	}
	e.expect(http.MethodPut, "/special-projects/projects"+idOf(t, a), `{"special_project_parent":["sp_b"]}`, http.StatusOK, `"special_project_parent":["sp_b"]`)
	e.expect(http.MethodPut, "/special-projects/projects"+idOf(t, a), `{"project_token":"sp_b"}`, http.StatusServiceUnavailable, `"special project store unavailable"`)

	result := e.expect(http.MethodGet, "/special-projects/validate", "", http.StatusOK,
		`"kind":"missing_parent"`, `parent 'sp_missing' does not exist`, `"kind":"cycle"`, `dependency cycle: sp_a → sp_b → sp_a`,
		`"kind":"orphan"`, `"project_token":"sp_lone"`, `"total":3`)
	if len(result["actionable"].([]any)) != 2 || len(result["info"].([]any)) != 1 {
		t.Fatalf("validate: %v", result)
	}
	e.expect(http.MethodPost, "/special-projects/validate", "", http.StatusNotFound, `"Not Found"`)
	e.expect(http.MethodDelete, "/special-projects/projects"+idOf(t, a), "", http.StatusNoContent)
	e.expect(http.MethodDelete, "/special-projects/projects"+idOf(t, a), "", http.StatusNotFound, `"Special project not found"`)
}

func TestUIPanels(t *testing.T) {
	e := newEnv(t, nil)
	e.expect(http.MethodPost, "/ui-panels", `{"token":" "}`, http.StatusUnprocessableEntity, `"token required"`)
	e.expect(http.MethodPost, "/ui-panels", `{"token":"x","context_type":"bogus"}`, http.StatusUnprocessableEntity, `context_type 'bogus' is not a valid scripted_gui context`)
	panel := e.expect(http.MethodPost, "/ui-panels", `{"token":"my panel","controls":[{"name":"btn","kind":"button"},{"name":"box","kind":"container","children":[{"name":"inner","kind":"text"}]}]}`, http.StatusCreated,
		`"context_type":"decision_category"`, `"gui_path":"interface/zz_uipanel_my_panel.gui"`, `"control_name":"my_panel_inner"`, `"warnings":[]`)
	e.expect(http.MethodPost, "/ui-panels", `{"token":"my panel"}`, http.StatusCreated, `"token":"my panel_2"`, `token 'my panel' collided · auto-suffixed to 'my panel_2'.`)

	path := "/ui-panels" + idOf(t, panel)
	stale := `{"expected_updated_at":"2000-01-01T00:00:00Z","name_en":"x"}`
	e.expect(http.MethodPut, path, stale, http.StatusConflict, `"code":"ui_panel_revision_conflict"`, `"expected_updated_at":"2000-01-01T00:00:00Z"`)
	e.expect(http.MethodPut, path, `{"expected_updated_at":"yesterday"}`, http.StatusUnprocessableEntity, `expected_updated_at must be an RFC3339 timestamp`)
	current := e.expect(http.MethodGet, path, "", http.StatusOK)
	updated := e.expect(http.MethodPut, path, `{"expected_updated_at":"`+current["updated_at"].(string)+`","name_en":"Mine"}`, http.StatusOK, `"name_en":"Mine"`)
	if updated["updated_at"] == current["updated_at"] {
		t.Fatal("updated_at not bumped")
	}
	e.expect(http.MethodPut, path, `{"token":"my panel_2"}`, http.StatusConflict, `token 'my panel_2' already exists in this project`)
	e.expect(http.MethodPut, "/ui-panels/99", `{"name_en":"x"}`, http.StatusNotFound, `"Panel not found"`)

	files := e.expect(http.MethodGet, path+"/files", "", http.StatusOK, `"window_name":"my_panel_panel"`, `"txt_path":"common/scripted_guis/zz_uipanel_my_panel.txt"`)
	if text, _ := files["gui_text"].(string); !strings.Contains(text, "containerWindowType") {
		t.Fatalf("gui_text: %v", files)
	}
	if paths, _ := files["loc_paths"].([]any); len(paths) == 0 || !strings.Contains(paths[0].(string), "TST_ui_panels_") {
		t.Fatalf("loc_paths: %v", files["loc_paths"])
	}
	e.expect(http.MethodPost, path+"/files", "", http.StatusMethodNotAllowed, `"GET only"`)
	e.expect(http.MethodGet, "/ui-panels", "", http.StatusOK, `"panels":[`, `"total":2`)
	e.expect(http.MethodDelete, path, "", http.StatusNoContent)
	e.expect(http.MethodGet, path+"/files", "", http.StatusNotFound, `"Panel not found"`)
}
