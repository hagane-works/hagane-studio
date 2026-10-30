package ideas

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
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func newTestServer(t *testing.T) (*project.Project, *http.ServeMux) {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"}}, "t")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return fstest.MapFS{}, nil },
	})
	return p, mux
}

func call(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func callMultipart(t *testing.T, mux *http.ServeMux, path, filename string, file []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if file != nil {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(file)
	}
	for k, v := range fields {
		writer.WriteField(k, v)
	}
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status=%d want %d body=%s", rec.Code, status, rec.Body.String())
	}
}

const base = "/api/go/projects/1"

func TestIdeaCRUD(t *testing.T) {
	_, mux := newTestServer(t)

	rec := call(t, mux, "GET", base+"/ideas", "")
	wantStatus(t, rec, 200)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list = %s", rec.Body.String())
	}

	rec = call(t, mux, "POST", base+"/ideas", `{"name":"Great Depression","idea_type":"country","modifier":{"stability_factor":-0.1}}`)
	wantStatus(t, rec, 201)
	created := decodeBody[Idea](t, rec)
	if created.IdeaID != "great_depression" || created.CountryTag != "TST" || created.NameEn != "Great Depression" ||
		created.RemovalCost != -1 || !created.CancelIfInvalid || !created.AutoAttachOnStart ||
		created.Source != "created" || created.ScopeKind != "own" || created.SharedCountryTags != nil ||
		created.Modifier["stability_factor"] != -0.1 || len(created.Warnings) != 0 {
		t.Fatalf("created = %+v", created)
	}
	var raw map[string]any
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if raw["shared_country_tags"] != nil || raw["warnings"] == nil {
		t.Fatalf("null/empty contract: %s", rec.Body.String())
	}

	rec = call(t, mux, "POST", base+"/ideas", `{"name":"Tank Co","idea_type":"tank_manufacturer"}`)
	wantStatus(t, rec, 201)
	designer := decodeBody[Idea](t, rec)
	if !designer.IsDesigner || designer.CountryTag != "" {
		t.Fatalf("designer = %+v", designer)
	}

	rec = call(t, mux, "GET", base+"/ideas", "")
	items := decodeBody[[]Idea](t, rec)
	if len(items) != 2 || items[0].IdeaType != "country" || items[1].IdeaType != "tank_manufacturer" {
		t.Fatalf("list = %+v", items)
	}
	rec = call(t, mux, "GET", base+"/ideas?idea_type=tank_manufacturer", "")
	if items := decodeBody[[]Idea](t, rec); len(items) != 1 || items[0].ID != designer.ID {
		t.Fatalf("filtered list = %+v", items)
	}
	rec = call(t, mux, "GET", base+"/ideas?country_tag=TST", "")
	if items := decodeBody[[]Idea](t, rec); len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("country list = %+v", items)
	}

	rec = call(t, mux, "GET", base+"/ideas/1", "")
	wantStatus(t, rec, 200)
	rec = call(t, mux, "GET", base+"/ideas/99", "")
	wantStatus(t, rec, 404)
	rec = call(t, mux, "GET", base+"/ideas/abc", "")
	wantStatus(t, rec, 404)

	rec = call(t, mux, "PUT", base+"/ideas/1", `{"picture":"GFX_idea_x","removal_cost":0,"display_name_locs":{"english":"GD"}}`)
	wantStatus(t, rec, 200)
	updated := decodeBody[Idea](t, rec)
	if updated.Picture != "GFX_idea_x" || updated.RemovalCost != 0 || updated.DisplayNameLocs["english"] != "GD" ||
		updated.Name != "Great Depression" || updated.Modifier["stability_factor"] != -0.1 {
		t.Fatalf("updated = %+v", updated)
	}

	rec = call(t, mux, "DELETE", base+"/ideas/1", "")
	wantStatus(t, rec, 204)
	rec = call(t, mux, "DELETE", base+"/ideas/1", "")
	wantStatus(t, rec, 404)
	if !strings.Contains(rec.Body.String(), "Idea not found") {
		t.Fatalf("delete missing body = %s", rec.Body.String())
	}
}

func TestCreateIdeaRefusesToInventAName(t *testing.T) {
	_, mux := newTestServer(t)
	for _, body := range []string{`{}`, `{"name":"  "}`, `{"name":"","idea_id":"  "}`} {
		rec := call(t, mux, "POST", base+"/ideas", body)
		wantStatus(t, rec, 422)
		got := decodeBody[map[string]map[string]string](t, rec)
		if got["detail"]["error"] != "idea_name_or_id_required" || !strings.Contains(got["detail"]["message"], "new_idea_N") {
			t.Fatalf("%s: body = %s", body, rec.Body.String())
		}
	}
	rec := call(t, mux, "POST", base+"/ideas", `{"idea_id":"  ger_spirit  "}`)
	wantStatus(t, rec, 201)
	if it := decodeBody[Idea](t, rec); it.IdeaID != "ger_spirit" || it.Name != "" {
		t.Fatalf("idea_id-only create = %+v", it)
	}
	rec = call(t, mux, "POST", base+"/ideas", `not json`)
	wantStatus(t, rec, 422)
}

func TestIdeaCreateScriptID(t *testing.T) {
	cases := []struct{ name, ideaID, want string }{
		{"New Idea", "", "new_idea"},
		{"Great Depression", "", "great_depression"},
		{"", "ger_war_economy", "ger_war_economy"},
		{"   ", "  ger_spirit  ", "ger_spirit"},
		{"Great Depression", "GER_gd", "GER_gd"},
	}
	for _, c := range cases {
		got, err := ideaCreateScriptID(c.name, c.ideaID)
		if err != nil || got != c.want {
			t.Fatalf("name=%q idea_id=%q → %q, %v; want %q", c.name, c.ideaID, got, err, c.want)
		}
	}
}

func TestCreateIdeaAutoSuffixesCollisions(t *testing.T) {
	_, mux := newTestServer(t)
	call(t, mux, "POST", base+"/ideas", `{"name":"New Idea"}`)
	rec := call(t, mux, "POST", base+"/ideas", `{"name":"New Idea"}`)
	wantStatus(t, rec, 201)
	it := decodeBody[Idea](t, rec)
	if it.IdeaID != "new_idea_1" || len(it.Warnings) != 1 ||
		it.Warnings[0] != "idea_id 'new_idea' collided · auto-suffixed to 'new_idea_1'." {
		t.Fatalf("suffixed = %+v", it)
	}
}

func TestIdeaBlockTextBoundary(t *testing.T) {
	_, mux := newTestServer(t)
	rec := call(t, mux, "POST", base+"/ideas", `{"name":"A","on_add":"set_country_flag = demo\\nadd_political_power = 25","name_zh":"精神\\n名称"}`)
	wantStatus(t, rec, 201)
	it := decodeBody[Idea](t, rec)
	if it.OnAdd != "set_country_flag = demo\nadd_political_power = 25" || it.NameZh != `精神\n名称` {
		t.Fatalf("canonical create = %q / %q", it.OnAdd, it.NameZh)
	}
	rec = call(t, mux, "PUT", base+"/ideas/1", `{"on_remove":"custom_effect_tooltip = \"a\\nb\""}`)
	wantStatus(t, rec, 400)
	if !strings.Contains(rec.Body.String(), "后续内容不生效") {
		t.Fatalf("ambiguous body = %s", rec.Body.String())
	}
}

func TestUpdateIdeaRules(t *testing.T) {
	p, mux := newTestServer(t)
	p.Write([]string{table}, func(tx *project.Tx) error {
		tx.Insert(table, project.Row{"idea_id": "imp", "idea_type": "country", "source": "imported", "scope_kind": "own",
			"ai_will_do": "", "cancel_if_invalid": true, "auto_attach_on_start": true,
			"display_name_locs": map[string]any{}, "description_locs": map[string]any{}})
		tx.Insert(table, project.Row{"idea_id": "other", "idea_type": "country", "source": "created", "scope_kind": "own",
			"ai_will_do": "", "cancel_if_invalid": true, "auto_attach_on_start": true,
			"display_name_locs": map[string]any{}, "description_locs": map[string]any{}})
		return nil
	})

	rec := call(t, mux, "PUT", base+"/ideas/1", `{}`)
	wantStatus(t, rec, 200)
	if it := decodeBody[Idea](t, rec); it.Source != "created" {
		t.Fatalf("imported idea must flip to created: %+v", it)
	}
	rec = call(t, mux, "PUT", base+"/ideas/1", `{"source":"imported","idea_type":"naval_manufacturer","is_designer":false,"idea_id":"  imp2 "}`)
	wantStatus(t, rec, 200)
	if it := decodeBody[Idea](t, rec); it.Source != "created" || !it.IsDesigner || it.IdeaID != "imp2" {
		t.Fatalf("designer/source rules: %+v", it)
	}
	rec = call(t, mux, "PUT", base+"/ideas/1", `{"idea_id":"other"}`)
	wantStatus(t, rec, 503)
	rec = call(t, mux, "PUT", base+"/ideas/1", `{"scope_kind":null}`)
	wantStatus(t, rec, 503)
	rec = call(t, mux, "PUT", base+"/ideas/1", `{"country_tag":null}`)
	wantStatus(t, rec, 200)
	rec = call(t, mux, "PUT", base+"/ideas/1", `{"removal_cost":"x"}`)
	wantStatus(t, rec, 422)
	rec = call(t, mux, "PUT", base+"/ideas/9", `{}`)
	wantStatus(t, rec, 404)
}

func TestForkIdea(t *testing.T) {
	p, mux := newTestServer(t)
	call(t, mux, "POST", base+"/ideas", `{"name":"Orig","picture":"GFX_x","modifier":{"a":1}}`)
	p.Write([]string{table}, func(tx *project.Tx) error {
		_, err := tx.Update(table, 1, func(row project.Row) { row["source"] = "imported" })
		return err
	})

	rec := call(t, mux, "POST", base+"/ideas/1/fork", `{"new_idea_id":" copy "}`)
	wantStatus(t, rec, 201)
	it := decodeBody[Idea](t, rec)
	if it.ID != 2 || it.IdeaID != "copy" || it.Source != "created" || it.Picture != "GFX_x" || it.Modifier["a"] != 1.0 {
		t.Fatalf("fork = %+v", it)
	}
	if orig := decodeBody[Idea](t, call(t, mux, "GET", base+"/ideas/1", "")); orig.Source != "imported" {
		t.Fatalf("original changed: %+v", orig)
	}
	rec = call(t, mux, "POST", base+"/ideas/1/fork", `{"new_idea_id":"copy"}`)
	wantStatus(t, rec, 409)
	if !strings.Contains(rec.Body.String(), "idea_id 'copy' already exists in this project") {
		t.Fatalf("conflict body = %s", rec.Body.String())
	}
	wantStatus(t, call(t, mux, "POST", base+"/ideas/9/fork", `{"new_idea_id":"x"}`), 404)
	wantStatus(t, call(t, mux, "POST", base+"/ideas/1/fork", `{}`), 422)
	wantStatus(t, call(t, mux, "POST", base+"/ideas/1/fork", `{"new_idea_id":"  "}`), 400)
}

func insertPreviewFixture(t *testing.T, p *project.Project) {
	t.Helper()
	err := p.Write([]string{table}, func(tx *project.Tx) error {
		tx.Insert(table, project.Row{
			"id": int64(6), "idea_id": "war_economy", "idea_type": "country", "name": "War Economy",
			"removal_cost": int64(0), "cancel_if_invalid": true, "scope_kind": "", "source": "",
			"auto_attach_on_start": false, "ai_will_do": "",
			"display_name_locs": map[string]any{}, "description_locs": map[string]any{},
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdeaPreviewUsesParityEmitter(t *testing.T) {
	p, mux := newTestServer(t)
	insertPreviewFixture(t, p)

	rec := call(t, mux, "GET", base+"/ideas/preview?idea_id=6", "")
	wantStatus(t, rec, 200)
	if got := decodeBody[map[string]string](t, rec)["script"]; got != "        war_economy = {\n        }" {
		t.Fatalf("single preview = %q", got)
	}
	rec = call(t, mux, "GET", base+"/ideas/preview", "")
	wantStatus(t, rec, 200)
	want := "ideas = {\n\n\tcountry = {\n\t\twar_economy = {\n\t\t}\n\n\t}\n}"
	if got := decodeBody[map[string]string](t, rec)["script"]; got != want {
		t.Fatalf("all preview = %q", got)
	}
}

func TestIdeaParsePreviewHappyPathAndExtrasSplit(t *testing.T) {
	p, mux := newTestServer(t)
	insertPreviewFixture(t, p)
	script := `ideas = {
    country = {
        USA_test_idea = {
            picture = generic_morale_bonus
            modifier = {
                stability_factor = 0.05
                war_support_factor = 0.05
            }
            allowed = { tag = USA }
            custom_effect = { add_stability = 0.1 }
        }
    }
}`
	body, _ := json.Marshal(map[string]string{"script": script})
	rec := call(t, mux, "POST", base+"/ideas/6/parse-preview", string(body))
	wantStatus(t, rec, 200)
	got := decodeBody[parsePreviewResponse](t, rec)
	if len(got.ParseErrors) != 0 || got.Entity == nil {
		t.Fatalf("unexpected parse result: %+v", got)
	}
	if got.Entity["idea_id"] != "USA_test_idea" || got.Entity["idea_type"] != "country" {
		t.Fatalf("identity mismatch: %#v", got.Entity)
	}
	if mods, ok := got.Entity["modifier"].(map[string]any); !ok || mods["stability_factor"] != 0.05 {
		t.Fatalf("modifier mismatch: %#v", got.Entity["modifier"])
	}
	if raw, ok := got.Entity["raw_extras"].(map[string]any); !ok || raw["custom_effect"] == nil {
		t.Fatalf("raw_extras must remain apply-able: %#v", got.Entity["raw_extras"])
	}
	if _, leaked := got.Extras["raw_extras"]; leaked {
		t.Fatalf("raw_extras classified as metadata: %#v", got.Extras)
	}
	if verdict, ok := got.Extras["_scope_verdict"].(map[string]any); !ok || verdict["kind"] != "own" {
		t.Fatalf("scope verdict extras mismatch: %#v", got.Extras)
	}
}

func TestIdeaParsePreviewSentinelsAndMultiWarning(t *testing.T) {
	p, mux := newTestServer(t)
	insertPreviewFixture(t, p)

	rec := call(t, mux, "POST", base+"/ideas/6/parse-preview", `{"script":"  \n"}`)
	wantStatus(t, rec, 200)
	got := decodeBody[parsePreviewResponse](t, rec)
	if got.Entity != nil || len(got.ParseErrors) != 1 || got.ParseErrors[0].Severity != "info" || got.ParseErrors[0].Msg != "Script is empty" {
		t.Fatalf("empty contract: %+v", got)
	}

	rec = call(t, mux, "POST", base+"/ideas/6/parse-preview", `{"script":"not_ideas_block = 1"}`)
	got = decodeBody[parsePreviewResponse](t, rec)
	if rec.Code != 200 || got.Entity != nil || len(got.ParseErrors) != 1 || !strings.Contains(got.ParseErrors[0].Msg, "No 'ideas = {") {
		t.Fatalf("missing wrapper: %d %+v", rec.Code, got)
	}

	script := `ideas = { country = {
            idea_one = { picture = x }
            idea_two = { picture = y }
        } }`
	body, _ := json.Marshal(map[string]string{"script": script})
	rec = call(t, mux, "POST", base+"/ideas/6/parse-preview", string(body))
	got = decodeBody[parsePreviewResponse](t, rec)
	want := "Script defined 2 ideas; only the first (idea_one) was used. Apply this and edit additional ideas separately."
	if rec.Code != 200 || got.Entity["idea_id"] != "idea_one" || len(got.Warnings) != 1 || got.Warnings[0] != want {
		t.Fatalf("multi: %d %+v", rec.Code, got)
	}
}

func TestIdeaPreviewErrorContracts(t *testing.T) {
	p, mux := newTestServer(t)
	insertPreviewFixture(t, p)
	cases := []struct {
		name, method, path, body string
		want                     int
		contains                 string
	}{
		{"invalid preview query", "GET", base + "/ideas/preview?idea_id=nope", "", 422, `"loc":["query","idea_id"]`},
		{"zero preview row", "GET", base + "/ideas/preview?idea_id=0", "", 404, "Idea not found"},
		{"missing preview row", "GET", base + "/ideas/preview?idea_id=7", "", 404, "Idea not found"},
		{"missing parse body field", "POST", base + "/ideas/6/parse-preview", `{}`, 422, `"loc":["body","script"]`},
		{"trailing parse body", "POST", base + "/ideas/6/parse-preview", `{"script":""} {}`, 422, `"loc":["body","script"]`},
		{"missing parse row", "POST", base + "/ideas/7/parse-preview", `{"script":"ideas = {}"}`, 404, "Idea not found"},
		{"picture not uploaded", "GET", base + "/ideas/6/picture", "", 404, "Asset not uploaded"},
		{"picture of missing idea", "GET", base + "/ideas/7/picture", "", 404, "Asset not uploaded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(t, mux, tc.method, tc.path, tc.body)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), 200, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUploadAndServeIdeaPicture(t *testing.T) {
	p, mux := newTestServer(t)
	call(t, mux, "POST", base+"/ideas", `{"name":"War Economy","idea_type":"country","country_tag":"ger"}`)

	rec := callMultipart(t, mux, base+"/ideas/1/upload-picture", "pic.png", testPNG(t, 120, 90), map[string]string{
		"crop_x": "10", "crop_y": "0", "crop_width": "80", "crop_height": "82",
	})
	wantStatus(t, rec, 200)
	got := decodeBody[map[string]any](t, rec)
	if got["picture"] != "GFX_idea_GER_war_economy" || got["filename"] != "idea_GER_war_economy.png" ||
		got["preview_url"] != "/api/go/projects/1/ideas/1/picture" {
		t.Fatalf("upload response = %v", got)
	}
	if w, ok := got["warnings"].([]any); !ok || len(w) != 0 {
		t.Fatalf("warnings = %v", got["warnings"])
	}
	it := decodeBody[Idea](t, call(t, mux, "GET", base+"/ideas/1", ""))
	if it.Picture != "GFX_idea_GER_war_economy" {
		t.Fatalf("picture not written back: %+v", it)
	}
	rows, _ := p.Rows(table)
	if rows[0]["has_obs"] != true {
		t.Fatalf("has_obs = %v", rows[0]["has_obs"])
	}
	paths, _ := p.Rows("entity_obs_paths")
	if len(paths) != 1 || paths[0]["entity_type"] != "idea_picture" || project.Int64(paths[0]["entity_id"]) != 1 ||
		!strings.HasPrefix(paths[0]["obs_key"].(string), "local/p/u1/upload/idea/1/idea_picture/") {
		t.Fatalf("entity_obs_paths = %v", paths)
	}

	rec = call(t, mux, "GET", base+"/ideas/1/picture", "")
	wantStatus(t, rec, 200)
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content type = %q", rec.Header().Get("Content-Type"))
	}
	img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil || img.Bounds().Dx() != 65 || img.Bounds().Dy() != 67 {
		t.Fatalf("served picture: %v %v", err, img)
	}

	wantStatus(t, callMultipart(t, mux, base+"/ideas/1/upload-picture", "b.png", testPNG(t, 40, 40), nil), 200)
	if paths, _ := p.Rows("entity_obs_paths"); len(paths) != 1 {
		t.Fatalf("mapping not upserted: %v", paths)
	}

	wantStatus(t, callMultipart(t, mux, base+"/ideas/9/upload-picture", "a.png", testPNG(t, 4, 4), nil), 404)
	wantStatus(t, callMultipart(t, mux, base+"/ideas/1/upload-picture", "", nil, nil), 422)
	wantStatus(t, callMultipart(t, mux, base+"/ideas/1/upload-picture", "a.png", []byte{}, nil), 400)
	wantStatus(t, callMultipart(t, mux, base+"/ideas/1/upload-picture", "a.png", []byte("not an image"), nil), 400)
	wantStatus(t, callMultipart(t, mux, base+"/ideas/1/upload-picture", "a.png", testPNG(t, 4, 4), map[string]string{"crop_x": "x"}), 422)
}

func TestIdeaPictureFallsBackToImportedAsset(t *testing.T) {
	p, mux := newTestServer(t)
	call(t, mux, "POST", base+"/ideas", `{"name":"A","picture":"GFX_idea_imported"}`)
	data := testPNG(t, 5, 5)
	key := "local/p/u1/raw/gfx/idea.png"
	if err := p.Blobs().Put(key, "image/png", int64(len(data)), bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		tx.Insert("project_obs_assets", project.Row{"asset_type": "idea_picture", "gfx_name": "GFX_idea_imported", "obs_key": key})
		return nil
	})
	rec := call(t, mux, "GET", base+"/ideas/1/picture", "")
	wantStatus(t, rec, 200)
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Fatal("served bytes differ from the imported picture")
	}
}

func TestIdeaPictureFindsImportedSpriteName(t *testing.T) {
	p, mux := newTestServer(t)
	call(t, mux, "POST", base+"/ideas", `{"name":"A","picture":"TST_spirit"}`)
	data := testPNG(t, 5, 5)
	key := "local/p/u1/upload/idea/GFX_TST_spirit.png"
	if err := p.Blobs().Put(key, "image/png", int64(len(data)), bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		tx.Insert("project_obs_assets", project.Row{"asset_type": "idea_picture", "gfx_name": "GFX_TST_spirit", "obs_key": key})
		return nil
	})
	rec := call(t, mux, "GET", base+"/ideas/1/picture", "")
	wantStatus(t, rec, 200)
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Fatal("served bytes differ from the imported picture")
	}
}

const importScript = `ideas = {
	country = {
		TST_one = {
			picture = generic_morale_bonus
			modifier = { stability_factor = 0.05 }
			on_add = { add_political_power = 10 }
		}
		TST_two = { removal_cost = 50 }
	}
}`

func TestImportIdeasFile(t *testing.T) {
	p, mux := newTestServer(t)
	rec := callMultipart(t, mux, base+"/import/ideas", "ideas.txt", []byte(importScript), nil)
	wantStatus(t, rec, 200)
	res := decodeBody[importResult](t, rec)
	if res.Total != 2 || res.Created != 2 || res.Updated != 0 || res.ConflictPolicy != "overwrite" {
		t.Fatalf("first import = %+v", res)
	}
	items := decodeBody[[]Idea](t, call(t, mux, "GET", base+"/ideas", ""))
	if len(items) != 2 || items[0].IdeaID != "TST_one" || items[0].Source != "created" || items[0].CountryTag != "TST" ||
		items[0].Picture != "generic_morale_bonus" || items[0].Name != "TST_one" || items[0].NameEn != "TST_one" ||
		items[0].Modifier["stability_factor"] != 0.05 || items[0].OnAdd != "" || items[1].RemovalCost != 50 {
		t.Fatalf("imported rows = %+v", items)
	}
	rows, _ := p.Rows(table)
	if snap, ok := rows[0]["source_snapshot"].(map[string]any); !ok || snap["idea_id"] != "TST_one" {
		t.Fatalf("source_snapshot = %v", rows[0]["source_snapshot"])
	}

	rec = callMultipart(t, mux, base+"/import/ideas?country_tag=GER", "ideas.txt", []byte(importScript), nil)
	res = decodeBody[importResult](t, rec)
	if res.Created != 0 || res.Updated != 2 {
		t.Fatalf("overwrite import = %+v", res)
	}
	items = decodeBody[[]Idea](t, call(t, mux, "GET", base+"/ideas", ""))
	if len(items) != 2 || items[0].CountryTag != "GER" || items[0].ID == 1 {
		t.Fatalf("overwritten rows = %+v", items)
	}

	rec = callMultipart(t, mux, base+"/import/ideas", "ideas.txt", []byte(importScript), map[string]string{"conflict_policy": "skip"})
	res = decodeBody[importResult](t, rec)
	if res.Skipped != 2 || res.Created != 0 {
		t.Fatalf("skip import = %+v", res)
	}
	wantStatus(t, callMultipart(t, mux, base+"/import/ideas", "ideas.txt", []byte(importScript), map[string]string{"conflict_policy": "merge"}), 400)
	wantStatus(t, callMultipart(t, mux, base+"/import/ideas", "ideas.txt", []byte(strings.Repeat("a = { ", 1100)), nil), 422)

	rec = callMultipart(t, mux, base+"/import/ideas", "empty.txt", []byte("foo = bar"), nil)
	wantStatus(t, rec, 200)
	if res := decodeBody[importResult](t, rec); res.Total != 0 || len(res.Warnings) == 0 {
		t.Fatalf("empty import = %+v", res)
	}
}

func TestVanillaIdeaIndex(t *testing.T) {
	_, mux := newTestServer(t)
	rec := call(t, mux, "GET", "/api/go/vanilla/ideas?category=country&has_picture=true&limit=2", "")
	wantStatus(t, rec, 200)
	type page struct {
		TotalMatching int              `json:"total_matching"`
		Limit         int              `json:"limit"`
		Items         []map[string]any `json:"items"`
		Meta          map[string]any   `json:"meta"`
	}
	got := decodeBody[page](t, rec)
	if got.TotalMatching == 0 || got.Limit != 2 || len(got.Items) != 2 || got.Meta["categories"] == nil {
		t.Fatalf("vanilla list = %+v", got)
	}
	first := got.Items[0]
	if first["category"] != "country" || first["picture"] == nil || first["picture"] == "" {
		t.Fatalf("filter not applied: %v", first)
	}
	rec = call(t, mux, "GET", "/api/go/vanilla/ideas/country/"+first["idea_id"].(string), "")
	wantStatus(t, rec, 200)
	if entry := decodeBody[map[string]map[string]any](t, rec)["entry"]; entry["idea_id"] != first["idea_id"] {
		t.Fatalf("detail = %v", entry)
	}
	wantStatus(t, call(t, mux, "GET", "/api/go/vanilla/ideas/country/no_such_idea", ""), 404)
	wantStatus(t, call(t, mux, "GET", "/api/go/vanilla/ideas?limit=x", ""), 422)
	wantStatus(t, call(t, mux, "GET", "/api/go/vanilla/ideas?has_picture=maybe", ""), 422)
}
