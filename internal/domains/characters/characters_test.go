package characters

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
	"strconv"
	"strings"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
)

type testEnv struct {
	t   *testing.T
	p   *project.Project
	mux *http.ServeMux
}

func newEnv(t *testing.T, game fs.FS) *testEnv {
	t.Helper()
	store := project.NewStore(t.TempDir())
	p, err := store.Create(project.Row{
		"name": "t", "uuid": "u1", "country_tag": "TST", "country_tags": []any{"TST"},
		"export_languages": []any{"en", "zh"},
	}, "t")
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

func (e *testEnv) do(method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, "/api/go/projects/1"+path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) json(method, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var raw []byte
	switch v := body.(type) {
	case nil:
	case string:
		raw = []byte(v)
	default:
		raw, _ = json.Marshal(v)
	}
	return e.do(method, path, "application/json", raw)
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

func expect(t *testing.T, rec *httptest.ResponseRecorder, status int, contains string) {
	t.Helper()
	if rec.Code != status || !strings.Contains(rec.Body.String(), contains) {
		t.Fatalf("status=%d want %d (containing %q) body=%s", rec.Code, status, contains, rec.Body.String())
	}
}

func (e *testEnv) createCharacter(body map[string]any) Character {
	e.t.Helper()
	rec := e.json("POST", "/characters", body)
	expect(e.t, rec, http.StatusCreated, "")
	return decode[Character](e.t, rec)
}

func TestCharacterCreateDefaultsAndList(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter(map[string]any{"name": "Boris Smith", "description": "A general"})
	if c.Tag != "TST_boris_smith" || c.CountryTag != "TST" || c.RoleType != "general" ||
		c.Skill != 1 || c.AdvisorCost != 150 || c.AdvisorRemovalCost != -1 || !c.CanBeCaptured ||
		c.Source != "created" || c.NameEn != "Boris Smith" || c.DescEn != "A general" ||
		c.ProjectID != e.p.ID() || len(c.Warnings) != 0 || c.Traits == nil || c.ScientistData == nil {
		t.Fatalf("defaults: %+v", c)
	}

	dup := e.createCharacter(map[string]any{"name": "Boris Smith"})
	if dup.Tag != "TST_boris_smith_1" || len(dup.Warnings) != 1 ||
		dup.Warnings[0] != "tag 'TST_boris_smith' collided · auto-suffixed to 'TST_boris_smith_1'." {
		t.Fatalf("collision: %+v", dup)
	}
	other := e.createCharacter(map[string]any{"name": "Anna", "tag": " GER_anna ", "country_tag": "ger"})
	if other.Tag != "GER_anna" || other.CountryTag != "GER" {
		t.Fatalf("explicit tag/country: %+v", other)
	}

	list := decode[[]Character](t, e.json("GET", "/characters", nil))
	if len(list) != 3 || list[0].ID != c.ID || list[2].ID != other.ID {
		t.Fatalf("list: %+v", list)
	}
	filtered := decode[[]Character](t, e.json("GET", "/characters?country_tag=GER", nil))
	if len(filtered) != 1 || filtered[0].Tag != "GER_anna" {
		t.Fatalf("filtered: %+v", filtered)
	}
	got := decode[Character](t, e.json("GET", "/characters/"+itoa(c.ID), nil))
	if got.Tag != c.Tag {
		t.Fatalf("get: %+v", got)
	}
	expect(t, e.json("GET", "/characters/999", nil), 404, "Character not found")
	expect(t, e.json("GET", "/characters/abc", nil), 404, "Character not found")
}

func TestCharacterCreateValidation(t *testing.T) {
	e := newEnv(t, nil)
	expect(t, e.json("POST", "/characters", map[string]any{}), 422, "name is required")
	expect(t, e.json("POST", "/characters", map[string]any{"name": "x", "localisation_key": "bad key"}), 422, "localisation_key may contain only")
	expect(t, e.json("POST", "/characters", "{not json"), 422, "Invalid character payload")
	expect(t, e.json("POST", "/characters", map[string]any{"name": "x", "skill": "high"}), 422, "Invalid character payload")
}

func TestCharacterUpdate(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createCharacter(map[string]any{"name": "A", "source": "imported"})
	b := e.createCharacter(map[string]any{"name": "B"})
	if a.Source != "imported" {
		t.Fatalf("source: %+v", a)
	}
	rec := e.json("PUT", "/characters/"+itoa(a.ID), map[string]any{
		"tag": "  TST_alpha ", "country_tag": " ger ", "skill": 4,
		"traits": []string{"old_guard"}, "scientist_data": map[string]any{"allow_multiple": true},
		"description": "Now described",
	})
	expect(t, rec, 200, "")
	got := decode[Character](t, rec)

	if got.Tag != "TST_alpha" || got.CountryTag != "GER" || got.Skill != 4 || got.Traits[0] != "old_guard" ||
		got.ScientistData["allow_multiple"] != true || got.DescEn != "Now described" || got.Source != "created" {
		t.Fatalf("update: %+v", got)
	}

	expect(t, e.json("PUT", "/characters/"+itoa(b.ID), map[string]any{"tag": "TST_alpha"}), 409,
		"tag 'TST_alpha' already exists in this project. Pick a unique tag.")

	expect(t, e.json("PUT", "/characters/"+itoa(b.ID), map[string]any{"name": nil}), 503, "character store unavailable")

	cleared := decode[Character](t, e.json("PUT", "/characters/"+itoa(a.ID), map[string]any{"traits": nil, "source_file": nil}))
	if len(cleared.Traits) != 0 || cleared.Traits == nil || cleared.SourceFile != "" {
		t.Fatalf("null traits: %+v", cleared)
	}
	expect(t, e.json("PUT", "/characters/"+itoa(a.ID), map[string]any{"localisation_key": "a b"}), 422, "localisation_key")
	expect(t, e.json("PUT", "/characters/999", map[string]any{"name": "x"}), 404, "Character not found")
}

func TestCharacterDeleteAndFork(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createCharacter(map[string]any{"name": "A", "source": "imported", "traits": []string{"t1"}})
	e.createCharacter(map[string]any{"name": "B"})
	rec := e.json("POST", "/characters/"+itoa(a.ID)+"/fork", map[string]any{"new_tag": " TST_a_fork "})
	expect(t, rec, 201, "")
	fork := decode[Character](t, rec)
	if fork.ID == a.ID || fork.Tag != "TST_a_fork" || fork.Source != "created" || fork.Name != "A" || fork.Traits[0] != "t1" {
		t.Fatalf("fork: %+v", fork)
	}
	expect(t, e.json("POST", "/characters/"+itoa(a.ID)+"/fork", map[string]any{"new_tag": "TST_b"}), 409, "tag 'TST_b' already exists in this project")
	expect(t, e.json("POST", "/characters/"+itoa(a.ID)+"/fork", map[string]any{"new_tag": "TST_a"}), 400, "new_tag must differ from the original tag")
	expect(t, e.json("POST", "/characters/"+itoa(a.ID)+"/fork", map[string]any{}), 422, "new_tag is required")
	expect(t, e.json("POST", "/characters/"+itoa(a.ID)+"/fork", map[string]any{"new_tag": "   "}), 400, "new_tag cannot be empty")
	expect(t, e.json("POST", "/characters/999/fork", map[string]any{"new_tag": "x"}), 404, "Source character not found")

	rec = e.json("DELETE", "/characters/"+itoa(a.ID), nil)
	if rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	expect(t, e.json("DELETE", "/characters/"+itoa(a.ID), nil), 404, "Character not found")
	if list := decode[[]Character](t, e.json("GET", "/characters", nil)); len(list) != 2 {
		t.Fatalf("after delete: %+v", list)
	}
}

func TestCharacterPreviewAndPreviewFiles(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter(map[string]any{
		"name": "Boris", "tag": "TST_boris", "role_type": "general", "skill": 3, "attack_skill": 2,
		"traits": []string{"old_guard"},
	})
	rec := e.json("GET", "/characters/"+itoa(c.ID)+"/preview", nil)
	expect(t, rec, 200, "")
	script := decode[map[string]string](t, rec)["script"]
	for _, want := range []string{"characters = {", "TST_boris = {", "corps_commander = {", "traits = { old_guard }", "skill = 3", "attack_skill = 2"} {
		if !strings.Contains(script, want) {
			t.Fatalf("preview missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "portraits = {") {
		t.Fatalf("unresolved portraits must be gated off:\n%s", script)
	}
	expect(t, e.json("GET", "/characters/999/preview", nil), 404, "Character not found")

	rec = e.json("GET", "/characters/"+itoa(c.ID)+"/preview-files", nil)
	expect(t, rec, 200, "")
	files := decode[struct {
		Files map[string]previewFile `json:"files"`
	}](t, rec).Files
	if files["script"].Path != "common/characters/TST_TST.txt" || !strings.Contains(files["script"].Content, "TST_boris = {") {
		t.Fatalf("script file: %+v", files["script"])
	}
	if files["loc_en"].Path != "localisation/english/replace/TST_characters_l_english.yml" ||
		!strings.Contains(files["loc_en"].Content, "l_english:") || !strings.Contains(files["loc_en"].Content, "Boris") {
		t.Fatalf("loc_en: %+v", files["loc_en"])
	}
	if _, ok := files["loc_zh"]; !ok {
		t.Fatalf("every export language gets a file: %v", files)
	}
	if files["gfx"].Path != "interface/TST_characters.gfx" {
		t.Fatalf("gfx: %+v", files["gfx"])
	}
	expect(t, e.json("POST", "/characters/"+itoa(c.ID)+"/preview-files", nil), 405, "Method Not Allowed")
}

func TestCharacterParsePreview(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter(map[string]any{"name": "Boris"})
	script := `characters = {
    GER_alex = {
        name = GER_alex
        corps_commander = {
            skill = 4
            attack_skill = 3
            defense_skill = 2
            planning_skill = 4
            logistics_skill = 1
        }
        portraits = { civilian = { large = "gfx/leaders/GER/alex.dds" } }
        custom_root = yes
    }
}`
	rec := e.json("POST", "/characters/"+itoa(c.ID)+"/parse-preview", map[string]string{"script": script})
	expect(t, rec, 200, "")
	got := decode[parsePreviewResponse](t, rec)
	if got.Entity == nil || len(got.ParseErrors) != 0 || got.Entity["tag"] != "GER_alex" || got.Entity["skill"] != float64(4) {
		t.Fatalf("entity: %+v", got)
	}
	if raw, ok := got.Entity["raw_extras"].(map[string]any); !ok || raw["custom_root"] != "yes" {
		t.Fatalf("raw_extras must remain apply-able: %#v", got.Entity["raw_extras"])
	}
	if got.Extras["large_portrait"] != "gfx/leaders/GER/alex.dds" {
		t.Fatalf("portrait stays in extras: %#v", got.Extras)
	}

	cases := []struct{ name, body, contains, severity string }{
		{"empty", `{"script":"  \n"}`, "Script is empty", "info"},
		{"missing wrapper", `{"script":"not_characters = 1"}`, "No 'characters = {", "error"},
		{"field value", `{"script":"characters = { GER_x = { corps_commander = { skill = defense_skill } } }"}`, "Field value error: invalid literal for int()", "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := e.json("POST", "/characters/"+itoa(c.ID)+"/parse-preview", tc.body)
			got := decode[parsePreviewResponse](t, rec)
			if rec.Code != 200 || got.Entity != nil || len(got.ParseErrors) != 1 ||
				got.ParseErrors[0].Severity != tc.severity || !strings.Contains(got.ParseErrors[0].Msg, tc.contains) {
				t.Fatalf("status=%d result=%+v", rec.Code, got)
			}
		})
	}
	expect(t, e.json("POST", "/characters/"+itoa(c.ID)+"/parse-preview", `{}`), 422, `"loc":["body","script"]`)
	expect(t, e.json("POST", "/characters/999/parse-preview", `{"script":"x = 1"}`), 404, "Character not found")
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 80, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func multipartBody(t *testing.T, files map[string][]byte, fields map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, data := range files {
		part, err := mw.CreateFormFile(name, name+".bin")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(data)
	}
	for name, value := range fields {
		mw.WriteField(name, value)
	}
	mw.Close()
	return buf.Bytes(), mw.FormDataContentType()
}

func imageSize(t *testing.T, data []byte) (int, int) {
	t.Helper()
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	return cfg.Width, cfg.Height
}

func TestPortraitUploadServeAndReset(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter(map[string]any{"name": "Boris", "tag": "TST_boris", "role_type": "advisor", "source": "imported"})
	cid := itoa(c.ID)
	expect(t, e.json("GET", "/characters/"+cid+"/assets/portrait_TST_boris.png", nil), 404, "Asset not uploaded")

	body, ctype := multipartBody(t, map[string][]byte{"file": testPNG(t, 300, 400)}, nil)
	rec := e.do("POST", "/characters/"+cid+"/upload-portrait", ctype, body)
	expect(t, rec, 200, "")
	resp := decode[map[string]any](t, rec)
	wantBase := "/api/go/projects/" + itoa(e.p.ID()) + "/characters/" + cid + "/assets/"
	if resp["filename"] != "portrait_TST_boris.png" || resp["portrait_url"] != wantBase+"portrait_TST_boris.png" ||
		resp["advisor_icon_url"] != wantBase+"advisor_TST_boris.png" || resp["trait_icon_url"] != wantBase+"trait_boris.png" {
		t.Fatalf("upload response: %v", resp)
	}
	got := decode[Character](t, e.json("GET", "/characters/"+cid, nil))
	if got.LargePortrait != "portrait_TST_boris.png" || got.SmallPortrait != "advisor_TST_boris.png" ||
		!got.IsPortraitModified || got.Source != "created" {
		t.Fatalf("character after upload: %+v", got)
	}
	for name, size := range map[string][2]int{
		"portrait_TST_boris.png": {156, 210}, "advisor_TST_boris.png": {65, 67}, "trait_boris.png": {23, 33},
	} {
		rec := e.json("GET", "/characters/"+cid+"/assets/"+name, nil)
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Header().Get("Content-Type"))
		}
		if w, h := imageSize(t, rec.Body.Bytes()); w != size[0] || h != size[1] {
			t.Fatalf("%s is %dx%d", name, w, h)
		}
	}

	script := decode[map[string]string](t, e.json("GET", "/characters/"+cid+"/preview", nil))["script"]
	if !strings.Contains(script, "portraits = {") {
		t.Fatalf("uploaded portrait must be referenced:\n%s", script)
	}

	rec = e.json("DELETE", "/characters/"+cid+"/portrait", nil)
	expect(t, rec, 200, `"ok":true`)
	reset := decode[struct {
		Deleted            []string `json:"deleted"`
		IsPortraitModified bool     `json:"is_portrait_modified"`
	}](t, rec)
	if len(reset.Deleted) != 3 || reset.IsPortraitModified || !strings.HasPrefix(reset.Deleted[0], "obs:local/p/u1/upload/character/") {
		t.Fatalf("reset: %+v", reset)
	}
	for _, key := range reset.Deleted {
		if e.p.Blobs().Exists(strings.TrimPrefix(key, "obs:")) {
			t.Fatalf("%s still stored", key)
		}
	}
	got = decode[Character](t, e.json("GET", "/characters/"+cid, nil))
	if got.LargePortrait != "" || got.SmallPortrait != "" || got.IsPortraitModified {
		t.Fatalf("character after reset: %+v", got)
	}
	expect(t, e.json("GET", "/characters/"+cid+"/assets/portrait_TST_boris.png", nil), 404, "Asset not uploaded")

	expect(t, e.json("DELETE", "/characters/"+cid+"/portrait", nil), 200, `"deleted":[]`)
	expect(t, e.json("DELETE", "/characters/999/portrait", nil), 404, "Character not found")
}

func TestPortraitUploadErrors(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter(map[string]any{"name": "Boris"})
	cid := itoa(c.ID)
	body, ctype := multipartBody(t, map[string][]byte{"file": testPNG(t, 10, 10)}, nil)
	expect(t, e.do("POST", "/characters/999/upload-portrait", ctype, body), 404, "Asset target not found")
	expect(t, e.do("POST", "/characters/"+cid+"/upload-portrait", "application/json", []byte(`{}`)), 422, "invalid multipart form")
	none, ctype2 := multipartBody(t, nil, map[string]string{"x": "y"})
	expect(t, e.do("POST", "/characters/"+cid+"/upload-portrait", ctype2, none), 422, "file is required")
	bad, ctype3 := multipartBody(t, map[string][]byte{"file": []byte("not an image")}, nil)
	expect(t, e.do("POST", "/characters/"+cid+"/upload-portrait", ctype3, bad), 400, "Failed to decode image")
	crop, ctype4 := multipartBody(t, map[string][]byte{"file": testPNG(t, 10, 10)}, map[string]string{"crop_x": "abc"})
	expect(t, e.do("POST", "/characters/"+cid+"/upload-portrait", ctype4, crop), 422, "invalid crop coordinates")
}

func TestAttachAnimatedPortrait(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter(map[string]any{"name": "Boris", "tag": "TST_boris"})
	cid := itoa(c.ID)
	dds := append([]byte("DDS "), make([]byte, 124)...)
	previewPNG := testPNG(t, 20, 20)
	files := map[string][]byte{"large_dds": dds, "small_dds": dds, "preview_png": previewPNG}
	fields := map[string]string{"gfx_name": "GFX_portrait_boris_anim", "small_gfx_name": "GFX_portrait_boris_small", "frame_count": "8", "fps": "10"}
	body, ctype := multipartBody(t, files, fields)
	rec := e.do("POST", "/characters/"+cid+"/attach-animated-portrait", ctype, body)
	expect(t, rec, 200, `"gfx_name":"GFX_portrait_boris_anim"`)
	got := decode[Character](t, e.json("GET", "/characters/"+cid, nil))
	if got.LargePortrait != "GFX_portrait_boris_anim" || got.SmallPortrait != "GFX_portrait_boris_small" || !got.IsPortraitModified {
		t.Fatalf("character: %+v", got)
	}
	assets, _ := e.p.Rows("project_obs_assets")
	if len(assets) != 3 {
		t.Fatalf("project_obs_assets: %v", assets)
	}
	for _, row := range assets {
		if row["gfx_name"] != "GFX_portrait_boris_anim" || !e.p.Blobs().Exists(str(row["obs_key"])) {
			t.Fatalf("asset row: %v", row)
		}
	}

	rec = e.json("GET", "/characters/"+cid+"/assets/GFX_portrait_boris_anim", nil)
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), previewPNG) {
		t.Fatalf("animated preview: %d", rec.Code)
	}

	fields["fps"] = "30"
	body, ctype = multipartBody(t, files, fields)
	expect(t, e.do("POST", "/characters/"+cid+"/attach-animated-portrait", ctype, body), 422, "frame_count must be 2..48")
	fields["fps"] = "10"
	files["large_dds"] = []byte("PNG not dds")
	body, ctype = multipartBody(t, files, fields)
	expect(t, e.do("POST", "/characters/"+cid+"/attach-animated-portrait", ctype, body), 400, "portrait textures must be DDS files")
}

func TestPortraitUploadRemovesGeneratedWorldLeader(t *testing.T) {
	e := newEnv(t, nil)
	history := "capital = 1\n\ncreate_country_leader = {\n\tname = \"Leader of TST\"\n\tdesc = \"\"\n" +
		"\tpicture = GFX_portrait_Europe_Generic_1\n\tideology = despotism\n\ttraits = { }\n}\nadd_political_power = 5\n"
	err := e.p.Write([]string{"project_raw_files"}, func(tx *project.Tx) error {
		tx.Insert("project_raw_files", project.Row{
			"dest_path": "history/countries/TST - New World.txt", "content": history,
			"generate_replace_path": false, "note": worldHistoryNote, "is_binary": false, "source": "imported",
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	c := e.createCharacter(map[string]any{"name": "Leader", "role_type": "country_leader", "ideology": "despotism"})
	body, ctype := multipartBody(t, map[string][]byte{"file": testPNG(t, 50, 60)}, nil)
	expect(t, e.do("POST", "/characters/"+itoa(c.ID)+"/upload-portrait", ctype, body), 200, "")
	raws, _ := e.p.Rows("project_raw_files")
	if content := str(raws[0]["content"]); content != "capital = 1\n\nadd_political_power = 5\n" || raws[0]["source"] != "created" {
		t.Fatalf("history after portrait: %q source=%v", content, raws[0]["source"])
	}
}

func TestPortraitUploadWithAnEmptyIdeologyTable(t *testing.T) {
	defer reftables.Use(map[string][]byte{reftables.IdeologiesFile: []byte(`{"groups": [], "sub_ideologies": []}`)})()
	e := newEnv(t, nil)
	history := "capital = 1\n\ncreate_country_leader = {\n\tname = \"Leader of TST\"\n\tideology = despotism\n}\n"
	err := e.p.Write([]string{"project_raw_files"}, func(tx *project.Tx) error {
		tx.Insert("project_raw_files", project.Row{
			"dest_path": "history/countries/TST - New World.txt", "content": history,
			"generate_replace_path": false, "note": worldHistoryNote, "is_binary": false, "source": "imported",
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	c := e.createCharacter(map[string]any{"name": "Leader", "role_type": "country_leader", "ideology": "despotism"})
	body, ctype := multipartBody(t, map[string][]byte{"file": testPNG(t, 50, 60)}, nil)
	expect(t, e.do("POST", "/characters/"+itoa(c.ID)+"/upload-portrait", ctype, body), 200, "")
	raws, _ := e.p.Rows("project_raw_files")
	if content := str(raws[0]["content"]); content != history {
		t.Fatalf("history after portrait: %q", content)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestCharacterCreateInputDefaults(t *testing.T) {
	var in CharacterCreateInput
	if err := json.Unmarshal([]byte(`{"name":"Boris"}`), &in); err != nil {
		t.Fatal(err)
	}
	if err := in.Normalize(); err != nil {
		t.Fatal(err)
	}
	if strDefault(in.RoleType, "general") != "general" || intDefault(in.AdvisorCost, 150) != 150 ||
		intDefault(in.AdvisorRemovalCost, -1) != -1 || !boolDefault(in.CanBeCaptured, true) ||
		strDefault(in.Source, "created") != "created" || intDefault(in.Skill, 1) != 1 {
		t.Fatalf("defaults broken: %+v", in)
	}
	if in.Traits == nil || in.ScientistData == nil || in.DisplayNameLocs == nil {
		t.Fatalf("containers not defaulted")
	}
	var missing CharacterCreateInput
	if err := json.Unmarshal([]byte(`{}`), &missing); err != nil {
		t.Fatal(err)
	}
	if err := missing.Normalize(); err == nil {
		t.Fatal("missing name must fail")
	}
}

func TestLocalisationKeyValidator(t *testing.T) {
	if _, err := validateLocalisationKey("GER_boris.v2's-key"); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
	if v, err := validateLocalisationKey("  "); err != nil || v != "" {
		t.Fatalf("blank should pass as empty: %q %v", v, err)
	}
	if _, err := validateLocalisationKey("空格 不行"); err == nil {
		t.Fatal("non-ASCII must fail")
	}
	var p CharacterPatch
	if err := json.Unmarshal([]byte(`{"localisation_key":"bad key"}`), &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err == nil {
		t.Fatal("patch validator must reject")
	}
}

func TestCharacterFromSparseRow(t *testing.T) {
	c := characterFromRow(project.Row{"id": json.Number("3"), "name": "x", "tag": "T_x"})
	if c.ID != 3 || c.Skill != 1 || c.AdvisorCost != 150 || c.AdvisorRemovalCost != -1 || !c.CanBeCaptured ||
		c.OperativeSkillLevel != 1 || c.Traits == nil || c.RawExtras == nil || c.DisplayNameLocs == nil ||
		c.AlwaysShowOnActionsTooltip != nil || c.Warnings == nil {
		t.Fatalf("sparse row: %+v", c)
	}
}
