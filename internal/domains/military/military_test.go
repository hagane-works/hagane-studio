package military

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

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

type testEnv struct {
	t   *testing.T
	p   *project.Project
	mux *http.ServeMux
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return nil, fs.ErrNotExist },
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

func (e *testEnv) seed(table string, rows ...project.Row) {
	e.t.Helper()
	if err := e.p.Write([]string{table}, func(tx *project.Tx) error {
		for _, row := range rows {
			tx.Insert(table, row)
		}
		return nil
	}); err != nil {
		e.t.Fatal(err)
	}
}

func TestEquipmentCRUD(t *testing.T) {
	e := newEnv(t)
	created := decodeInto[Equipment](t, e.expect("POST", "/equipment", `{"equipment_id":" tank_a ","country_tag":" ger ","soft_attack":3.5,"year":1936}`, 201))
	if created.EquipmentID != "tank_a" || created.CountryTag != "GER" || created.Source != "created" ||
		created.SoftAttack == nil || *created.SoftAttack != 3.5 || created.HardAttack != nil || created.Year == nil || *created.Year != 1936 {
		t.Fatalf("created = %+v", created)
	}

	if second := decodeInto[Equipment](t, e.expect("POST", "/equipment", `{"equipment_id":"tank_a"}`, 201)); second.EquipmentID != "tank_a_1" {
		t.Fatalf("suffixed id = %q", second.EquipmentID)
	}
	if third := decodeInto[Equipment](t, e.expect("POST", "/equipment", `{}`, 201)); third.EquipmentID != "new_equipment" || third.Resources == nil {
		t.Fatalf("default = %+v", third)
	}
	list := decodeInto[[]Equipment](t, e.expect("GET", "/equipment?country_tag=GER", "", 200))
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("filtered list = %+v", list)
	}
	if all := decodeInto[[]Equipment](t, e.expect("GET", "/equipment", "", 200)); len(all) != 3 || all[0].EquipmentID != "new_equipment" {
		t.Fatalf("list order = %+v", all)
	}
	e.expect("PUT", "/equipment/1", `{"equipment_id":"tank_a_1"}`, 409, "equipment_id collision")
	e.expect("PUT", "/equipment/1", `{"soft_attack":"x"}`, 503, "equipment store unavailable")
	updated := decodeInto[Equipment](t, e.expect("PUT", "/equipment/1", `{"soft_attack":null,"name":"Tank"}`, 200))
	if updated.SoftAttack != nil || updated.Name != "Tank" {
		t.Fatalf("updated = %+v", updated)
	}
	e.expect("PUT", "/equipment/1", `[`, 422, "Invalid equipment payload")
	e.expect("GET", "/equipment/99", "", 404, "Equipment not found")
	e.expect("GET", "/equipment/x/y", "", 404, "Equipment not found")
	e.expect("PATCH", "/equipment", "", 405, `"error":"GET or POST only"`)
	e.expect("PATCH", "/equipment/1", "", 405, `"error":"GET, PUT or DELETE only"`)
	e.expect("GET", "/equipment/preview/1", "", 200, `"script":`, "tank_a")
	e.expect("GET", "/equipment/preview/abc", "", 422, "int_parsing", "equipment_id")
	e.expect("GET", "/equipment/preview/0", "", 404, "Equipment not found")
	e.expect("DELETE", "/equipment/1", "", 204)
	e.expect("DELETE", "/equipment/1", "", 404, "Equipment not found")
}

func TestEquipmentImportedRowFlipsOnEdit(t *testing.T) {
	e := newEnv(t)
	e.seed(equipmentTable, project.Row{"equipment_id": "x", "name": "", "is_archetype": false, "source": "imported",
		"display_name_locs": map[string]any{}, "description_locs": map[string]any{}})
	if got := decodeInto[Equipment](t, e.expect("PUT", "/equipment/1", `{"picture":"GFX_x"}`, 200)); got.Source != "created" {
		t.Fatalf("source = %q", got.Source)
	}
}

func TestEquipmentEnabledBy(t *testing.T) {
	e := newEnv(t)
	e.seed("technologies",
		project.Row{"tech_id": "t1", "name": "Tech 1", "tech_line": "infantry", "enable_equipments": []any{"rifle_1"}},
		project.Row{"tech_id": "t2", "name": "Tech 2", "tech_line": "armor", "enable_equipments": []any{"tank_1"}},
	)
	got := decodeInto[map[string]any](t, e.expect("GET", "/equipment/enabled-by/rifle_1", "", 200))
	techs := got["techs"].([]any)
	if got["equipment_id"] != "rifle_1" || len(techs) != 1 || techs[0].(map[string]any)["tech_id"] != "t1" {
		t.Fatalf("enabled-by = %v", got)
	}
	e.expect("POST", "/equipment/enabled-by/rifle_1", "", 405)
}

const equipmentFile = `equipments = {
	rifle_1 = {
		year = 1936
		is_archetype = yes
		soft_attack = 3
		resources = { steel = 2 }
		custom_key = 5
	}
	rifle_2 = { archetype = rifle_1 year = 1939 }
}`

func TestEquipmentImport(t *testing.T) {
	e := newEnv(t)
	body := func(policy string) string {
		raw, _ := json.Marshal(map[string]any{"text": equipmentFile, "source_file": "common/units/equipment/x.txt", "policy": policy})
		return string(raw)
	}
	result := decodeInto[EquipmentImportResult](t, e.expect("POST", "/equipment/import", body(""), 200))
	if result.Created != 2 || result.Updated != 0 || len(result.EquipmentIDs) != 2 {
		t.Fatalf("import = %+v", result)
	}
	rows, _ := e.p.Rows(equipmentTable)
	first := equipmentFromRow(rows[0])
	if !first.IsArchetype || first.SoftAttack == nil || *first.SoftAttack != 3 || first.Source != "created" ||
		first.SourceFile != "common/units/equipment/x.txt" || first.Resources["steel"] == nil || first.RawExtras["custom_key"] == nil {
		t.Fatalf("imported row = %+v", first)
	}
	if again := decodeInto[EquipmentImportResult](t, e.expect("POST", "/equipment/import", body("skip"), 200)); again.Skipped != 2 || len(again.EquipmentIDs) != 0 {
		t.Fatalf("skip import = %+v", again)
	}
	if again := decodeInto[EquipmentImportResult](t, e.expect("POST", "/equipment/import", body("overwrite"), 200)); again.Updated != 2 {
		t.Fatalf("overwrite import = %+v", again)
	}
	if rows, _ := e.p.Rows(equipmentTable); len(rows) != 2 {
		t.Fatalf("rows after overwrite = %d", len(rows))
	}
	e.expect("POST", "/equipment/import", `{"source_file":"x"}`, 422, "text is required")
	e.expect("POST", "/equipment/import", `{"text":""} {}`, 422, "Invalid military import payload")
	e.expect("GET", "/equipment/import", "", 405, "Method Not Allowed")
}

func TestSubUnitCRUDAndValidation(t *testing.T) {
	e := newEnv(t)
	e.expect("POST", "/sub-units", `{"sub_unit_id":"x","unit_types":["cavalry"]}`, 422, "unit_types contains boolean field 'cavalry'; set cavalry=yes instead")
	e.expect("POST", "/sub-units", `{"sub_unit_id":"x","categories":["bad token"]}`, 422, "invalid sub-unit category token 'bad token'")
	created := decodeInto[SubUnit](t, e.expect("POST", "/sub-units", `{"sub_unit_id":"inf","country_tag":"ger","unit_types":[" infantry "],"categories":["category_front_line"],"priority":600}`, 201))
	if created.SubUnitID != "inf" || created.CountryTag != "GER" || created.Source != "created" || created.SourceFile != "" ||
		len(created.UnitTypes) != 1 || created.UnitTypes[0] != "infantry" || created.Priority == nil || *created.Priority != 600 {
		t.Fatalf("created = %+v", created)
	}
	if second := decodeInto[SubUnit](t, e.expect("POST", "/sub-units", `{"sub_unit_id":"inf"}`, 201)); second.SubUnitID != "inf_1" {
		t.Fatalf("suffix = %q", second.SubUnitID)
	}
	e.expect("PUT", "/sub-units/1", `{"source":"imported"}`, 422, "source is managed by the import/create workflow")
	e.expect("PUT", "/sub-units/1", `{"source_file":"common/units/x.txt"}`, 422, "source_file is managed")
	e.expect("PUT", "/sub-units/1", `{"sub_unit_id":"inf_1"}`, 409, "sub_unit_id collision")
	updated := decodeInto[SubUnit](t, e.expect("PUT", "/sub-units/1", `{"abbreviation":"INF","active":true}`, 200))
	if updated.Abbreviation != "INF" || updated.Active == nil || !*updated.Active {
		t.Fatalf("updated = %+v", updated)
	}
	if list := decodeInto[[]SubUnit](t, e.expect("GET", "/sub-units?country_tag=ger", "", 200)); len(list) != 1 {
		t.Fatalf("filtered = %+v", list)
	}
	e.expect("GET", "/sub-units/preview/1", "", 200, "inf = {", "type = { infantry }")
	e.expect("DELETE", "/sub-units/2", "", 204)
	e.expect("GET", "/sub-units/2", "", 404, "Sub-unit not found")
}

func TestSubUnitPreviewRefusesLegacyClassification(t *testing.T) {
	e := newEnv(t)
	e.seed(subUnitTable, project.Row{"sub_unit_id": "old", "source": "created", "ordering": 0, "name": "",
		"display_name_locs": map[string]any{}, "description_locs": map[string]any{}, "unit_types": []any{"marines"}})
	e.expect("GET", "/sub-units/preview/1", "", 422, "unit_types contains boolean field 'marines'")
}

func TestSubUnitImport(t *testing.T) {
	e := newEnv(t)
	text := "sub_units = {\n\tlight_armor = {\n\t\tsprite = light_armor\n\t\ttype = { armor }\n\t\tpriority = 2500\n\t\tmaximum_speed = 1.2\n\t\tforest = { attack = -0.2 }\n\t}\n}"
	body, _ := json.Marshal(map[string]any{"text": text, "source_file": `common\units\light_armor.txt`})
	result := decodeInto[SubUnitImportResult](t, e.expect("POST", "/sub-units/import", string(body), 200))
	if result.Created != 1 || result.SubUnitIDs[0] != "light_armor" {
		t.Fatalf("import = %+v", result)
	}
	got := decodeInto[SubUnit](t, e.expect("GET", "/sub-units/1", "", 200))
	if got.SourceFile != "common/units/light_armor.txt" || got.Sprite != "light_armor" || got.Priority == nil || *got.Priority != 2500 ||
		got.Stats["maximum_speed"] == nil || got.TerrainModifiers["forest"] == nil || got.UnitTypes[0] != "armor" {
		t.Fatalf("imported = %+v", got)
	}
	body, _ = json.Marshal(map[string]any{"text": text, "source_file": "common/units/deep/x.txt"})
	e.expect("POST", "/sub-units/import", string(body), 422, "source_file must be a mod-relative common/units/*.txt path")
}

func TestDivisionTemplates(t *testing.T) {
	e := newEnv(t)
	first := decodeInto[DivisionTemplate](t, e.expect("POST", "/division-templates", `{}`, 201))
	if first.TemplateName != "New Division" || first.OobFile != "untitled" || first.CountryTag != "UNTITLED" || first.Source != "created" {
		t.Fatalf("defaults = %+v", first)
	}
	if second := decodeInto[DivisionTemplate](t, e.expect("POST", "/division-templates", `{}`, 201)); second.TemplateName != "New Division 1" {
		t.Fatalf("suffix = %q", second.TemplateName)
	}
	ger := decodeInto[DivisionTemplate](t, e.expect("POST", "/division-templates",
		`{"oob_file":"GER_1936","template_name":"Infanterie","regiments":[{"token":"infantry","x":0,"y":0}],"priority":2}`, 201))
	if ger.CountryTag != "GER" || len(ger.Regiments) != 1 || ger.Priority == nil {
		t.Fatalf("ger = %+v", ger)
	}
	e.expect("PUT", "/division-templates/2", `{"template_name":" New Division "}`, 409, "division template collision")
	e.expect("PUT", "/division-templates/2", `{"ordering":null}`, 503, "division store unavailable")
	renamed := decodeInto[DivisionTemplate](t, e.expect("PUT", "/division-templates/2", `{"template_name":" Other "}`, 200))
	if renamed.TemplateName != "Other" {
		t.Fatalf("renamed = %+v", renamed)
	}
	if list := decodeInto[[]DivisionTemplate](t, e.expect("GET", "/division-templates?oob_file=GER_1936", "", 200)); len(list) != 1 {
		t.Fatalf("filtered = %+v", list)
	}
	e.expect("GET", "/division-templates/preview/3", "", 200, "division_template", "Infanterie", "infantry")
	e.expect("DELETE", "/division-templates/3", "", 204)
	e.expect("GET", "/division-templates/3", "", 404, "Division template not found")
}

func TestDivisionTemplateImport(t *testing.T) {
	e := newEnv(t)
	text := "division_template = {\n\tname = \"Infantry\"\n\tregiments = { infantry = { x = 0 y = 0 } }\n\tsupport = { engineer = { x = 0 y = 0 } }\n}\nunits = { }\n"
	body, _ := json.Marshal(map[string]any{"text": text, "oob_file": "GER_1936"})
	result := decodeInto[DivisionTemplateImportResult](t, e.expect("POST", "/division-templates/import", string(body), 200))
	if result.Created != 1 || result.TemplateNames[0] != "Infantry" {
		t.Fatalf("import = %+v", result)
	}
	got := decodeInto[DivisionTemplate](t, e.expect("GET", "/division-templates/1", "", 200))
	if got.CountryTag != "GER" || got.OobFile != "GER_1936" || len(got.Support) != 1 || !strings.Contains(got.OobUnitsTail, "units") {
		t.Fatalf("imported = %+v", got)
	}
	e.expect("POST", "/division-templates/import", `{"text":""}`, 422, "oob_file is required")
	e.expect("POST", "/division-templates/import", `{"text":"","oob_file":"  "}`, 422, "oob_file is required")
}

func TestDivisionDeployments(t *testing.T) {
	e := newEnv(t)
	e.seed("states", project.Row{"state_id": 1, "owner": "GER", "provinces": []any{100, "101"}, "source": "created", "display_name_locs": map[string]any{}})
	e.seed("countries", project.Row{"tag": "ROM", "name": "Romania"})
	e.seed(divisionTemplateTable, project.Row{"oob_file": "GER_1936", "template_name": "Infantry", "ordering": 0, "source": "created"})
	e.expect("POST", "/division-deployments", `{"count":0}`, 422, "count must be between 1 and 1000")
	created := decodeInto[DivisionDeployment](t, e.expect("POST", "/division-deployments",
		`{"oob_file":"GER_1936","division_template":" Infantry ","location":100}`, 201))
	if created.CountryTag != "GER" || created.Count != 1 || created.DivisionTemplate != "Infantry" || len(created.Warnings) != 0 {
		t.Fatalf("created = %+v", created)
	}
	foreign := decodeInto[DivisionDeployment](t, e.expect("POST", "/division-deployments",
		`{"oob_file":"new_1936","division_template":"Panzer","location":5}`, 201))
	want := []string{"actionable:unknown_tag|tag=NEW", "info:missing_template|template=Panzer"}
	if strings.Join(foreign.Warnings, ",") != strings.Join(want, ",") {
		t.Fatalf("warnings = %v", foreign.Warnings)
	}
	updated := decodeInto[DivisionDeployment](t, e.expect("PUT", "/division-deployments/1", `{"location":999,"count":0}`, 200))
	if updated.Count != 1 || len(updated.Warnings) != 1 || updated.Warnings[0] != "actionable:unowned_province|province=999|tag=GER" {
		t.Fatalf("updated = %+v", updated)
	}
	e.expect("PUT", "/division-deployments/1", `{"count":1001}`, 422, "count must be between 1 and 1000")
	list := decodeInto[[]DivisionDeployment](t, e.expect("GET", "/division-deployments?oob_file=GER_1936", "", 200))
	if len(list) != 1 || len(list[0].Warnings) != 1 {
		t.Fatalf("list = %+v", list)
	}
	e.expect("DELETE", "/division-deployments/2", "", 204)
	e.expect("GET", "/division-deployments/2", "", 404, "Division deployment not found")
}

func TestDoctrines(t *testing.T) {
	e := newEnv(t)
	e.expect("POST", "/doctrines", `{"entity_key":"land","entity_type":"bogus"}`, 400, "entity_type must be one of")
	e.expect("POST", "/doctrines", `{"entity_key":" "}`, 400, "entity_key is required")
	e.expect("POST", "/doctrines", `{"entity_key":"x","source_file":"../x.txt"}`, 422, "path traversal")
	created := decodeInto[Doctrine](t, e.expect("POST", "/doctrines",
		`{"entity_key":"mobile","entity_type":"grand_doctrine","source_file":"\\grand_doctrines\\land.txt","raw_extras":{"xor":"a","keep":1}}`, 201))
	if created.SourceFile != "grand_doctrines/land.txt" || created.EntityType != "grand_doctrine" {
		t.Fatalf("created = %+v", created)
	}
	if !regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d{6})?$`).MatchString(created.CreatedAt) {
		t.Fatalf("created_at = %q", created.CreatedAt)
	}
	e.expect("POST", "/doctrines", `{"entity_key":"mobile"}`, 409, "A grand_doctrine with key 'mobile' already exists")
	decodeInto[Doctrine](t, e.expect("POST", "/doctrines", `{"entity_key":"land","entity_type":"folder"}`, 201))
	updated := decodeInto[Doctrine](t, e.expect("PUT", "/doctrines/1", `{"xor":"a b"}`, 200))
	if updated.Xor == nil || *updated.Xor != "a b" || updated.RawExtras["xor"] != nil || updated.RawExtras["keep"] == nil {
		t.Fatalf("purge = %+v", updated)
	}
	e.expect("PUT", "/doctrines/1", `{"entity_key":"land","entity_type":"folder"}`, 409, "entity_type + entity_key conflict in this project")
	e.expect("PUT", "/doctrines/1", `{"entity_key":""}`, 400, "entity_key cannot be empty")
	e.expect("PUT", "/doctrines/99", `{"entity_type":"bad"}`, 404, "Doctrine not found")
	list := decodeInto[map[string]any](t, e.expect("GET", "/doctrines", "", 200))
	if list["total"].(float64) != 2 || list["doctrines"].([]any)[0].(map[string]any)["entity_type"] != "folder" {
		t.Fatalf("list = %v", list)
	}
	if filtered := decodeInto[map[string]any](t, e.expect("GET", "/doctrines?entity_type=folder", "", 200)); filtered["total"].(float64) != 1 {
		t.Fatalf("filtered = %v", filtered)
	}
	e.expect("DELETE", "/doctrines/1", "", 204)
	e.expect("GET", "/doctrines/1", "", 404, "Doctrine not found")
}

func TestMIOs(t *testing.T) {
	e := newEnv(t)
	e.expect("POST", "/mios", `{"mio_id":" "}`, 400, "mio_id is required")
	created := decodeInto[MIO](t, e.expect("POST", "/mios", `{"mio_id":" GER_rheinmetall ","traits":[{"kind":"initial","token":"t1"}]}`, 201))
	if created.MioID != "GER_rheinmetall" || created.ScopeKind != "own" || created.SharedCountryTags != nil || len(created.Traits) != 1 {
		t.Fatalf("created = %+v", created)
	}
	if !strings.Contains(e.do("GET", "/mios/1", "").Body.String(), `"shared_country_tags":null`) {
		t.Fatal("shared_country_tags must stay null")
	}
	e.expect("POST", "/mios", `{"mio_id":"GER_rheinmetall"}`, 409, "mio_id 'GER_rheinmetall' already exists in this project")
	decodeInto[MIO](t, e.expect("POST", "/mios", `{"mio_id":"A_first"}`, 201))
	e.expect("PUT", "/mios/1", `{"mio_id":"A_first"}`, 409, "mio_id conflict in this project")
	e.expect("PUT", "/mios/1", `{"mio_id":" "}`, 400, "mio_id cannot be empty")
	updated := decodeInto[MIO](t, e.expect("PUT", "/mios/1", `{"equipment_type":["infantry_equipment"],"shared_country_tags":["ITA"]}`, 200))
	if len(updated.EquipmentType) != 1 || len(updated.SharedCountryTags) != 1 {
		t.Fatalf("updated = %+v", updated)
	}
	list := decodeInto[map[string]any](t, e.expect("GET", "/mios", "", 200))
	if list["total"].(float64) != 2 || list["mios"].([]any)[0].(map[string]any)["mio_id"] != "A_first" {
		t.Fatalf("list = %v", list)
	}
	e.expect("DELETE", "/mios/2", "", 204)
	e.expect("DELETE", "/mios/2", "", 404, "MIO not found")
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *testEnv) upload(path string) *httptest.ResponseRecorder {
	e.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("file", "icon.png")
	part.Write(pngBytes(e.t))
	form.Close()
	req := httptest.NewRequest("POST", "/api/go/projects/1"+path, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func TestMIOIcons(t *testing.T) {
	e := newEnv(t)
	decodeInto[MIO](t, e.expect("POST", "/mios", `{"mio_id":"GER_krupp","traits":[{"token":"heavy armor"}]}`, 201))
	e.expect("GET", "/mios/1/icon", "", 404, "Asset not uploaded")
	rec := e.upload("/mios/1/upload-icon")
	if rec.Code != 200 {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body.String())
	}
	got := decodeInto[map[string]any](t, rec)
	if got["icon"] != "GFX_idea_mio_ger_krupp" || got["filename"] != "mio_ger_krupp.png" || got["preview_url"] != "/api/go/projects/1/mios/1/icon" {
		t.Fatalf("upload response = %v", got)
	}
	if mio := decodeInto[MIO](t, e.expect("GET", "/mios/1", "", 200)); mio.Icon != "GFX_idea_mio_ger_krupp" {
		t.Fatalf("icon not written back: %+v", mio)
	}
	if icon := e.do("GET", "/mios/1/icon", ""); icon.Code != 200 || icon.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("icon read = %d %s", icon.Code, icon.Header().Get("Content-Type"))
	}
	if missing := e.upload("/mios/9/upload-icon"); missing.Code != 404 || !strings.Contains(missing.Body.String(), "Asset target not found") {
		t.Fatalf("missing target = %d %s", missing.Code, missing.Body.String())
	}
	e.expect("GET", "/mios/1/upload-icon", "", 405, "POST only")

	e.expect("GET", "/mios/1/traits/heavy%20armor/icon", "", 404, "MIO trait icon not uploaded")
	if rec := e.upload("/mios/1/traits/missing/upload-icon"); rec.Code != 404 || !strings.Contains(rec.Body.String(), "MIO trait not found") {
		t.Fatalf("missing trait = %d %s", rec.Code, rec.Body.String())
	}
	rec = e.upload("/mios/1/traits/heavy%20armor/upload-icon")
	if rec.Code != 200 {
		t.Fatalf("trait upload = %d %s", rec.Code, rec.Body.String())
	}
	got = decodeInto[map[string]any](t, rec)
	if got["icon"] != "GFX_idea_mio_trait_ger_krupp_heavy_armor" || got["preview_url"] != "/api/go/projects/1/mios/1/traits/heavy%20armor/icon" {
		t.Fatalf("trait upload response = %v", got)
	}
	mio := decodeInto[MIO](t, e.expect("GET", "/mios/1", "", 200))
	if mio.Traits[0]["icon"] != "GFX_idea_mio_trait_ger_krupp_heavy_armor" {
		t.Fatalf("trait icon not written back: %+v", mio.Traits)
	}
	if icon := e.do("GET", "/mios/1/traits/heavy%20armor/icon", ""); icon.Code != 200 {
		t.Fatalf("trait icon read = %d %s", icon.Code, icon.Body.String())
	}
	e.expect("GET", "/mios/1/traits/heavy%20armor/other", "", 404, "MIO trait asset not found")
}

func TestOOBUnitsCRUD(t *testing.T) {
	e := newEnv(t)
	e.seed("states", project.Row{"state_id": 64, "owner": "GER", "provinces": []any{123}, "source": "created", "display_name_locs": map[string]any{}})
	e.expect("POST", "/oob-units/air-wing-groups", `{"oob_file":"GER_1936"}`, 422, "location is required")
	e.expect("POST", "/oob-units/air-wing-groups", `{"oob_file":"GER_1936","location":"two tokens"}`, 422)
	group := decodeInto[AirWingGroup](t, e.expect("POST", "/oob-units/air-wing-groups",
		`{"oob_file":"GER_1936","source_file":"history\\units\\GER_1936.txt","country_tag":"ger","location":" 99 "}`, 201))
	if group.Location != "99" || group.CountryTag != "GER" || group.SourceFile != "history/units/GER_1936.txt" ||
		len(group.Warnings) != 1 || group.Warnings[0] != "actionable:unowned_state|state=99|tag=GER" {
		t.Fatalf("group = %+v", group)
	}
	e.expect("POST", "/oob-units/air-wings", `{"group_id":9,"equipment_type":"fighter_equipment_1"}`, 404, "Air wing group not found")
	e.expect("POST", "/oob-units/air-wings", `{"group_id":1}`, 422, "equipment_type is required")
	wing := decodeInto[AirWing](t, e.expect("POST", "/oob-units/air-wings",
		`{"group_id":1,"equipment_type":"fighter_equipment_1","name":"Wing","amount":"24","owner":"GER"}`, 201))
	if wing.GroupID != 1 || wing.Amount != "24" || wing.BodyAst == nil {
		t.Fatalf("wing = %+v", wing)
	}
	fleet := decodeInto[Fleet](t, e.expect("POST", "/oob-units/fleets", `{"oob_file":"GER_1936","name":"Home","naval_base":"123"}`, 201))
	if fleet.CountryTag != "" || len(fleet.Warnings) != 0 {
		t.Fatalf("fleet = %+v", fleet)
	}
	force := decodeInto[TaskForce](t, e.expect("POST", "/oob-units/task-forces", `{"fleet_id":1,"name":"TF","location":"777"}`, 201))
	if len(force.Warnings) != 1 || force.Warnings[0] != "actionable:unowned_province|province=777|tag=GER" {
		t.Fatalf("task force = %+v", force)
	}
	decodeInto[Ship](t, e.expect("POST", "/oob-units/ships", `{"task_force_id":1,"name":"KMS Test","definition":"destroyer","start_experience_factor":"0.25"}`, 201))
	e.expect("POST", "/oob-units/ships", `{"task_force_id":1,"definition":"two words"}`, 422)

	updated := decodeInto[AirWingGroup](t, e.expect("PUT", "/oob-units/air-wing-groups/1", `{"location":"64"}`, 200))
	if len(updated.Warnings) != 0 {
		t.Fatalf("updated group = %+v", updated)
	}
	e.expect("PUT", "/oob-units/air-wings/1", `{"group_id":5}`, 404, "Air wing group not found")
	e.expect("PUT", "/oob-units/air-wings/1", `{"ordering":null}`, 500, "update air-wings failed")
	e.expect("PUT", "/oob-units/ships/1", `[`, 422, "Invalid OOB payload")

	snap := decodeInto[OOBSnapshot](t, e.expect("GET", "/oob-units/snapshot?oob_file=GER_1936", "", 200))
	if len(snap.AirWingGroups) != 1 || len(snap.AirWings) != 1 || len(snap.Fleets) != 1 || len(snap.TaskForces) != 1 || len(snap.Ships) != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
	preview := decodeInto[map[string]string](t, e.expect("GET", "/oob-units/preview", "", 200))
	if !strings.Contains(preview["script"], "air_wings = {") || !strings.Contains(preview["script"], `name = "KMS Test"`) {
		t.Fatalf("preview = %s", preview["script"])
	}
	if list := decodeInto[[]Ship](t, e.expect("GET", "/oob-units/ships?task_force_id=1", "", 200)); len(list) != 1 {
		t.Fatalf("ships = %+v", list)
	}

	e.expect("DELETE", "/oob-units/fleets/1", "", 204)
	e.expect("GET", "/oob-units/task-forces/1", "", 404, "Task force not found")
	e.expect("GET", "/oob-units/ships/1", "", 404, "Ship not found")
	e.expect("DELETE", "/oob-units/air-wing-groups/1", "", 204)
	e.expect("GET", "/oob-units/air-wings/1", "", 404, "Air wing not found")
	e.expect("GET", "/oob-units/unknown", "", 404, "Not Found")
	e.expect("GET", "/oob-units", "", 404, "Not Found")
	e.expect("PATCH", "/oob-units/fleets", "", 405, "GET or POST only")
}

func TestOOBImport(t *testing.T) {
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{"oob_file": "GER_1936", "text": oobTransferFixture})
	result := decodeInto[OOBImportResult](t, e.expect("POST", "/oob-units/import", string(body), 200))
	if result.AirWingGroups != 1 || result.AirWings != 1 || result.Fleets != 1 || result.TaskForces != 1 || result.Ships != 1 {
		t.Fatalf("import = %+v", result)
	}
	snap := decodeInto[OOBSnapshot](t, e.expect("GET", "/oob-units/snapshot?source_file=history/units/GER_1936.txt", "", 200))
	if len(snap.AirWings) != 1 || snap.AirWings[0].Owner != "GER" || snap.Ships[0].StartExperienceFactor != "0.25" ||
		snap.Fleets[0].CountryTag != "GER" || snap.Fleets[0].Source != "created" {
		t.Fatalf("snapshot = %+v", snap)
	}
	preview := decodeInto[map[string]string](t, e.expect("GET", "/oob-units/preview?oob_file=GER_1936", "", 200))
	if !strings.Contains(preview["script"], "fighter_equipment_1 = {") || !strings.Contains(preview["script"], "mission = escort") {
		t.Fatalf("preview = %s", preview["script"])
	}

	skip, _ := json.Marshal(map[string]any{"oob_file": "GER_1936", "text": oobTransferFixture, "policy": "skip"})
	if skipped := decodeInto[OOBImportResult](t, e.expect("POST", "/oob-units/import", string(skip), 200)); skipped.Skipped != 2 || skipped.Fleets != 0 {
		t.Fatalf("skip = %+v", skipped)
	}

	e.expect("POST", "/oob-units/import", string(body), 200)
	for _, table := range []string{airWingGroupTable, airWingTable, fleetTable, taskForceTable, shipTable} {
		if rows, _ := e.p.Rows(table); len(rows) != 1 {
			t.Fatalf("%s rows after re-import = %d", table, len(rows))
		}
	}

	broken, _ := json.Marshal(map[string]any{"oob_file": "GER_1936", "text": "units = {\n fleet = {\n"})
	if warned := decodeInto[OOBImportResult](t, e.expect("POST", "/oob-units/import", string(broken), 200)); len(warned.Warnings) != 1 {
		t.Fatalf("warned = %+v", warned)
	}
	if rows, _ := e.p.Rows(fleetTable); len(rows) != 1 {
		t.Fatal("warning-only import must keep the hierarchy")
	}
	e.expect("POST", "/oob-units/import", `{"text":"x","policy":"merge"}`, 422, "Invalid OOB import payload")
	e.expect("POST", "/oob-units/import", `{"oob_file":null,"text":"x"}`, 422, "Invalid OOB import payload")
	e.expect("GET", "/oob-units/import", "", 405, "Method Not Allowed")
}
