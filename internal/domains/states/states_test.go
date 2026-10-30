package states

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

	"golang.org/x/image/bmp"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

const alphaState = `state={
	id=1
	name="STATE_1"
	manpower=1000
	state_category = town
	resources={ steel=5 }
	history={
		owner = TST
		add_core_of = TST
		victory_points = { 1 5 }
		buildings = {
			infrastructure = 3
			1 = { naval_base = 2 }
		}
		1939.1.1 = { owner = GER }
	}
	provinces={ 1 2 }
	some_flag = yes
}
`

const betaState = `state={
	id=2
	name="STATE_2"
	manpower=500
	state_category = rural
	history={ owner = OTH }
	provinces={ 3 4 }
	local_supplies=2.5
	extra_key = { a = 1 }
}
`

func testGame(t *testing.T) fstest.MapFS {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	colors := map[int]color.RGBA{1: {255, 0, 0, 255}, 2: {0, 255, 0, 255}, 3: {0, 0, 255, 255}, 4: {255, 255, 0, 255}}
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			province := 1
			if y >= 2 {
				province = 2
			}
			if x >= 4 {
				province += 2
			}
			img.Set(x, y, colors[province])
		}
	}
	var picture bytes.Buffer
	if err := bmp.Encode(&picture, img); err != nil {
		t.Fatal(err)
	}
	var sprite bytes.Buffer
	if err := png.Encode(&sprite, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return fstest.MapFS{
		"launcher-settings.json":                                   {Data: []byte(`{"version":"Test v1.19.3.0","rawVersion":"1.19.3.0"}`)},
		"history/states/1-Alpha.txt":                               {Data: []byte(alphaState)},
		"history/states/2-Beta.txt":                                {Data: []byte(betaState)},
		"history/states/notes.md":                                  {Data: []byte("ignored")},
		"map/provinces.bmp":                                        {Data: picture.Bytes()},
		"map/definition.csv":                                       {Data: []byte("0;0;0;0;land;false;unknown;0\n1;255;0;0;land;false;plains;1\n2;0;255;0;land;false;plains;1\n3;0;0;255;land;false;plains;1\n4;255;255;0;land;false;plains;1\n")},
		"interface/bookmarks.gfx":                                  {Data: []byte(`spriteTypes = { spriteType = { name = "GFX_select_date_1936" texturefile = "gfx/interface/bookmarks/date_1936.png" } }`)},
		"gfx/interface/bookmarks/date_1936.png":                    {Data: sprite.Bytes()},
		"localisation/english/state_names_l_english.yml":           {Data: []byte("l_english:\n STATE_1:0 \"Alpha\"\n STATE_2:0 \"Beta\" # note\n GER_STATE_1:0 \"Not this\"\n")},
		"localisation/simp_chinese/state_names_l_simp_chinese.yml": {Data: []byte("l_simp_chinese:\n STATE_1:0 \"阿尔法\"\n")},
	}
}

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
	deps := domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) {
			if game == nil {
				return nil, fs.ErrNotExist
			}
			return game, nil
		},
	}
	mux := http.NewServeMux()
	Register(mux, deps)
	RegisterList(mux, deps)
	return &testEnv{t: t, p: p, mux: mux}
}

func (e *testEnv) request(method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, "/api/go/projects/1"+path, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) expect(method, path, body string, status int, fragments ...string) *httptest.ResponseRecorder {
	e.t.Helper()
	rec := e.request(method, path, body, nil)
	e.check(rec, method+" "+path, status, fragments...)
	return rec
}

func (e *testEnv) check(rec *httptest.ResponseRecorder, label string, status int, fragments ...string) {
	e.t.Helper()
	if rec.Code != status {
		e.t.Fatalf("%s: status %d, want %d: %s", label, rec.Code, status, rec.Body.String())
	}
	for _, fragment := range fragments {
		if !strings.Contains(rec.Body.String(), fragment) {
			e.t.Fatalf("%s: body lacks %q: %s", label, fragment, rec.Body.String())
		}
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return out
}

func (e *testEnv) states() map[int]map[string]any {
	e.t.Helper()
	rows := decode[[]map[string]any](e.t, e.expect("GET", "/states", "", 200))
	out := map[int]map[string]any{}
	for _, row := range rows {
		out[int(row["state_id"].(float64))] = row
	}
	return out
}

func rowID(row map[string]any) string { return jsonText(row["id"]) }

func jsonText(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func TestSeedAndRead(t *testing.T) {
	e := newEnv(t, testGame(t))
	e.expect("POST", "/states/seed-from-vanilla", `{}`, 400, "Provide `all: true`")
	e.expect("POST", "/states/seed-from-vanilla", `[1]`, 422, "Invalid state seed payload")
	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 200, `"seeded":2`, `"skipped":0`, `"missing":[]`, `"warnings":[]`)
	e.expect("POST", "/states/seed-from-vanilla", `{"all":"yes"}`, 200, `"seeded":0`, `"skipped":2`)
	e.expect("POST", "/states/seed-from-vanilla", `{"state_ids":[1,"2",99]}`, 200, `"missing":[99]`, `"skipped":2`)

	states := e.states()
	alpha := states[1]
	if got := jsonText(alpha["display_name"]); got != `{"en":"Alpha","zh":"阿尔法"}` {
		t.Errorf("display_name %s", got)
	}
	if got := jsonText(alpha["province_buildings"]); got != `{"1":{"naval_base":2}}` {
		t.Errorf("province_buildings %s", got)
	}
	extras := alpha["raw_extras"].(map[string]any)
	if _, ok := extras["buildings"]; !ok {
		t.Errorf("seed dropped the province buildings: %v", extras)
	}
	if _, ok := extras["history.1939.1.1"]; !ok {
		t.Errorf("seed dropped the dated block: %v", extras)
	}
	if _, ok := extras["some_flag"]; ok {
		t.Errorf("seed kept a non-whitelisted raw_extras key: %v", extras)
	}
	if alpha["source"] != "imported" || alpha["source_file"] != "1-Alpha.txt" || alpha["owner"] != "TST" || alpha["controller"] != nil {
		t.Errorf("alpha row %v", alpha)
	}
	if got := jsonText(alpha["victory_points"]); got != `[{"province_id":1,"value":5}]` {
		t.Errorf("victory_points %s", got)
	}
	provenance := alpha["membership_provenance"].(map[string]any)
	if provenance["status"] != "known" || provenance["membership_changed"] != false {
		t.Errorf("provenance %v", provenance)
	}
	if got := jsonText(states[2]["display_name"]); got != `{"en":"Beta"}` {
		t.Errorf("beta display_name %s", got)
	}
	if states[2]["local_supplies"] != 2.5 {
		t.Errorf("local_supplies %v", states[2]["local_supplies"])
	}

	owned := decode[[]map[string]any](t, e.expect("GET", "/states/by-owner/tst", "", 200))
	if len(owned) != 1 || owned[0]["state_id"].(float64) != 1 {
		t.Errorf("by-owner %v", owned)
	}
	e.expect("GET", "/states/"+rowID(alpha), "", 200, `"display_name":{"en":"Alpha"`)
	e.expect("GET", "/states/999", "", 404, "State not found")
}

func TestSeedWithoutGame(t *testing.T) {
	e := newEnv(t, nil)
	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 503, `"code":"vanilla_root_unconfigured"`)

	e.expect("GET", "/states", "", 200, "[]")
}

func TestUpdateState(t *testing.T) {
	e := newEnv(t, testGame(t))
	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 200)
	states := e.states()
	alpha := "/states/" + rowID(states[1])
	rec := e.expect("PUT", alpha, `{"owner":" OTH ","province_buildings":{"2":{"bunker":3},"1":{"naval_base":4}},"force_link_ownership_to":2}`, 200)
	row := decode[map[string]any](t, rec)
	extras := row["raw_extras"].(map[string]any)
	if extras["buildings"] != "2 = {\n\tbunker = 3\n}\n1 = {\n\tnaval_base = 4\n}" {
		t.Errorf("buildings %q", extras["buildings"])
	}
	if extras["force_link_ownership_to"] != "2" || row["force_link_ownership_to"].(float64) != 2 {
		t.Errorf("force link column %v copy %v", row["force_link_ownership_to"], extras["force_link_ownership_to"])
	}
	if row["owner"] != "OTH" || row["source"] != "created" {
		t.Errorf("owner/source %v %v", row["owner"], row["source"])
	}
	if row["membership_provenance"].(map[string]any)["owner_changed"] != true {
		t.Errorf("provenance %v", row["membership_provenance"])
	}
	e.expect("PUT", alpha, `{"provinces":[1,1]}`, 422, "duplicate")
	e.expect("PUT", alpha, `{"force_link_ownership_to":1.5}`, 422, "integer state id")
	e.expect("PUT", alpha, `{"manpower":"lots"}`, 422, "Invalid state payload")
	e.expect("PUT", alpha, `not json`, 422, "Invalid state payload")
	row = decode[map[string]any](t, e.expect("PUT", alpha, `{"force_link_ownership_to":null,"owner":null,"display_name_locs":{"fr":"Alpha FR"}}`, 200))
	if _, has := row["raw_extras"].(map[string]any)["force_link_ownership_to"]; has || row["owner"] != "" {
		t.Errorf("cleared force link / owner: %v", row)
	}
	if row["display_name"].(map[string]any)["fr"] != "Alpha FR" {
		t.Errorf("display_name %v", row["display_name"])
	}
	e.expect("PUT", "/states/999", `{}`, 404, "State not found")

	beta := "/states/" + rowID(states[2])
	if err := e.p.Write([]string{statesTable}, func(tx *project.Tx) error {
		_, err := tx.Update(statesTable, project.Int64(states[2]["id"].(float64)), func(r project.Row) {
			r["source_snapshot"] = nil
			r["source"] = "created"
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	row = decode[map[string]any](t, e.expect("PUT", beta, `{"provinces":[4,3,5]}`, 200))
	if got := jsonText(row["membership_provenance"].(map[string]any)["added"]); got != "[5]" {
		t.Errorf("added %s (%v)", got, row["membership_provenance"])
	}
}

func TestBulkOwnerCoreAndRestore(t *testing.T) {
	e := newEnv(t, testGame(t))
	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 200)
	states := e.states()
	ids := "[" + rowID(states[1]) + "," + rowID(states[2]) + "]"
	body := `{"state_db_ids":` + ids + `,"owner":"NEW","cores":{"mode":"replace_all","tags":["NEW"]},"dry_run":true}`
	e.expect("POST", "/states/bulk-owner-core", body, 200, `"dry_run":true`, `"changed":2`, `"removed_cores":["TST"]`)
	if after := e.states(); after[1]["owner"] != "TST" || after[1]["source"] != "imported" {
		t.Fatalf("dry run wrote: %v", after[1])
	}
	result := decode[stateBulkResult](t, e.expect("POST", "/states/bulk-owner-core", strings.Replace(body, `"dry_run":true`, `"dry_run":false`, 1), 200))
	if result.Changed != 2 || result.Details[0].BeforeSource != "imported" || result.Details[0].BeforeOwner != "TST" {
		t.Fatalf("bulk result %+v", result)
	}
	after := e.states()
	if after[1]["owner"] != "NEW" || jsonText(after[1]["add_core_of"]) != `["NEW"]` || after[1]["source"] != "created" {
		t.Fatalf("bulk wrote: %v", after[1])
	}
	e.expect("POST", "/states/bulk-owner-core", strings.Replace(body, `"dry_run":true`, `"dry_run":false`, 1), 200, `"skipped":2`, `"skipped_reason":"already_target"`)
	e.expect("POST", "/states/bulk-owner-core", `{"state_db_ids":[999],"owner":"X"}`, 404, "State not found")
	e.expect("POST", "/states/bulk-owner-core", `{"state_db_ids":[1]}`, 422, "nothing to change")

	entries := []map[string]any{}
	for _, d := range result.Details {
		entries = append(entries, map[string]any{"id": d.ID, "owner": d.BeforeOwner, "add_core_of": d.BeforeCores, "source": d.BeforeSource})
	}
	raw, _ := json.Marshal(map[string]any{"entries": entries})
	e.expect("POST", "/states/bulk-restore", string(raw), 200, `"changed":2`)
	restored := e.states()
	if restored[1]["owner"] != "TST" || jsonText(restored[1]["add_core_of"]) != `["TST"]` || restored[1]["source"] != "imported" {
		t.Fatalf("restore: %v", restored[1])
	}
	e.expect("POST", "/states/bulk-restore", `{"entries":[]}`, 422, "entries must not be empty")
}

func TestDeleteStates(t *testing.T) {
	e := newEnv(t, testGame(t))
	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 200)
	states := e.states()
	alpha, beta := rowID(states[1]), rowID(states[2])
	e.expect("PUT", "/states/"+alpha, `{"manpower":7}`, 200, `"source":"created"`)
	e.expect("POST", "/states/bulk-delete", `{"state_ids":[]}`, 422, "state_ids must not be empty")
	e.expect("POST", "/states/bulk-delete", `{"state_ids":[`+alpha+`,`+beta+`,999]}`, 200, `"deleted":1`, `"skipped_created":[`+alpha+`]`)
	e.expect("DELETE", "/states/"+alpha, "", 204)
	e.expect("DELETE", "/states/"+alpha, "", 404, "State not found")
	e.expect("GET", "/states", "", 200, "[]")
}

func TestRestoreVanillaContent(t *testing.T) {
	e := newEnv(t, testGame(t))
	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 200)
	if err := e.p.Write([]string{statesTable}, func(tx *project.Tx) error {
		tx.Insert(statesTable, project.Row{"state_id": int64(50), "name": "STATE_50", "display_name_locs": map[string]any{}, "provinces": []any{}, "source": "created"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	e.expect("POST", "/states/restore-vanilla-content", `{}`, 422, "state_db_ids must not be empty")
	e.expect("POST", "/states/restore-vanilla-content", `{"all":true,"state_db_ids":[1]}`, 422, "not both")
	dry := decode[restoreResult](t, e.expect("POST", "/states/restore-vanilla-content", `{"all":true,"dry_run":true}`, 200))
	if dry.Restored != 1 || dry.Skipped != 1 || dry.Missing != 1 || dry.MissingStateIDs[0] != 50 || dry.SkippedStateIDs[0] != 1 {
		t.Fatalf("dry run %+v", dry)
	}
	if !dry.VanillaVersionKnown || dry.VanillaGameVersion != "Test v1.19.3.0" || dry.VanillaFiles != 2 || dry.VanillaStates != 2 {
		t.Fatalf("source facts %+v", dry)
	}
	if extras := e.states()[2]["raw_extras"].(map[string]any); len(extras) != 0 {
		t.Fatalf("dry run wrote raw_extras %v", extras)
	}
	res := decode[restoreResult](t, e.expect("POST", "/states/restore-vanilla-content", `{"all":true}`, 200))
	if res.Restored != 1 || res.Details[1].Reason != "" || jsonText(res.Details[1].Keys) != `["extra_key"]` {
		t.Fatalf("restore %+v", res)
	}
	beta := e.states()[2]
	if beta["source"] != "imported" || beta["raw_extras"].(map[string]any)["extra_key"] == nil {
		t.Fatalf("restored row %v", beta)
	}
	e.expect("POST", "/states/restore-vanilla-content", `{"all":true}`, 200, `"restored":0`, `"skipped":2`, `"reason":"already_has_content"`, `"reason":"not_in_vanilla"`)

	none := newEnv(t, nil)
	none.expect("POST", "/states/restore-vanilla-content", `{"all":true}`, 503, `"code":"vanilla_root_unconfigured"`)
	empty := newEnv(t, fstest.MapFS{"map/definition.csv": {Data: []byte("")}})
	empty.expect("POST", "/states/restore-vanilla-content", `{"all":true}`, 503, `"code":"vanilla_states_dir_missing"`)
}

func TestStatesGeometry(t *testing.T) {
	none := newEnv(t, nil)
	none.expect("GET", "/states-geometry", "", 200, `"geometry":null`, `"available":false`, `"source":"vanilla"`)

	e := newEnv(t, testGame(t))
	type shape struct {
		BBox [4]int `json:"bbox"`
	}
	type response struct {
		Available bool
		Source    string
		Geometry  struct {
			Format      string           `json:"format"`
			GameVersion string           `json:"game_version"`
			Factor      int              `json:"factor"`
			Width       int              `json:"width"`
			Height      int              `json:"height"`
			StateCount  int              `json:"state_count"`
			States      map[string]shape `json:"states"`
		}
	}
	base := decode[response](t, e.expect("GET", "/states-geometry", "", 200))
	if !base.Available || base.Geometry.Format != "hoi4-states-geo" || base.Geometry.GameVersion != "1.19.3" ||
		base.Geometry.Factor != 1 || base.Geometry.Width != 8 || base.Geometry.Height != 4 || base.Geometry.StateCount != 2 {
		t.Fatalf("base geometry %+v", base)
	}
	if base.Geometry.States["1"].BBox != [4]int{0, 0, 4, 4} || base.Geometry.States["2"].BBox != [4]int{4, 0, 8, 4} {
		t.Fatalf("base boxes %+v", base.Geometry.States)
	}

	e.expect("POST", "/states/seed-from-vanilla", `{"all":true}`, 200)
	states := e.states()
	e.expect("PUT", "/states/"+rowID(states[1]), `{"provinces":[1,2,3]}`, 200)
	e.expect("PUT", "/states/"+rowID(states[2]), `{"provinces":[4]}`, 200)
	moved := decode[response](t, e.expect("GET", "/states-geometry", "", 200))
	if moved.Geometry.States["1"].BBox != [4]int{0, 0, 8, 4} || moved.Geometry.States["2"].BBox != [4]int{4, 2, 8, 4} {
		t.Fatalf("project outlines %+v", moved.Geometry.States)
	}

	if _, err := e.p.UpdateRecord(func(r project.Row) { r["dependencies"] = []any{"Some Base Mod"} }); err != nil {
		t.Fatal(err)
	}
	e.expect("GET", "/states-geometry", "", 200, `"geometry":null`, `"source":"Some Base Mod"`, `"available":false`)
}

func TestHistoryFiles(t *testing.T) {
	e := newEnv(t, nil)
	if err := e.p.Write([]string{locEntriesTable}, func(tx *project.Tx) error {
		tx.Insert(locEntriesTable, project.Row{"locale": "l_english", "key": "TST_alt", "value": "Alt Name", "source": "imported",
			"source_file": "localisation/english/mod_l_english.yml", "replace_priority": int64(0), "source_order": int64(0), "entry_order": int64(0), "is_replace": false})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	e.expect("POST", "/history-files", `{}`, 400, "country_tag is required")
	e.expect("POST", "/history-files", `[]`, 422, "Invalid payload")
	e.expect("POST", "/history-files", `{"country_tag":"TST","set_cosmetic_tag":"bad tag"}`, 422, "invalid character")
	rec := e.expect("POST", "/history-files", `{"country_tag":" tst ","set_cosmetic_tag":"TST_alt","set_popularities":{"democratic":60,"technocracy":40},"capital":1}`, 201,
		`"country_tag":"TST"`, `"set_popularities_democratic":60`, `"set_stability":-1`, `"date_blocks":[]`, `"source":"created"`)
	created := decode[map[string]any](t, rec)
	path := "/history-files/" + rowID(created)
	tags, _ := e.p.Rows(cosmeticTagsTable)
	names, _ := e.p.Rows(cosmeticNamesTable)
	if len(tags) != 1 || tags[0]["tag"] != "TST_alt" || project.Int64(tags[0]["revision"]) != 2 || len(names) != 1 || names[0]["value"] != "Alt Name" {
		t.Fatalf("cosmetic tag %v names %v", tags, names)
	}
	e.expect("POST", "/history-files", `{"country_tag":"TST"}`, 409, "already exists")
	e.expect("GET", "/history-files", "", 200, `"country_tag":"TST"`)
	e.expect("PUT", path, `{"set_popularities_fascism":10}`, 422, "canonical set_popularities map")
	e.expect("PUT", path, `{"date_blocks":[{"date":"bad","script":""}]}`, 422, "date block 0: date must match")
	e.expect("PUT", path, `{"date_blocks":[{"date":"1939.1.1","script":"x = y"}],"add_to_faction_tags":["GER"],"add_to_faction":"OLD"}`, 200,
		`"add_to_faction":""`, `"add_to_faction_tags":["GER"]`, `"date":"1939.1.1"`)
	e.expect("PUT", path, `{"capital":1.5}`, 422, "Invalid payload")
	e.expect("POST", "/history-files", `{"country_tag":"OTH","source":"imported"}`, 201)
	e.expect("PUT", path, `{"country_tag":"oth"}`, 409, "country_tag conflict")
	others := decode[[]map[string]any](t, e.expect("GET", "/history-files", "", 200))
	if len(others) != 2 || others[0]["country_tag"] != "OTH" {
		t.Fatalf("list order %v", others)
	}
	e.expect("PUT", "/history-files/"+rowID(others[0]), `{"capital":3}`, 200, `"source":"created"`, `"capital":3`)
	e.expect("DELETE", path, "", 204)
	e.expect("DELETE", path, "", 404, "HistoryFile not found")
	e.expect("GET", path, "", 404, "HistoryFile not found")
}

func TestBookmarks(t *testing.T) {
	e := newEnv(t, testGame(t))
	e.expect("POST", "/bookmarks", `{}`, 400, "bookmark_id is required")
	rec := e.expect("POST", "/bookmarks", `{"bookmark_id":" bm1 ","name_en":"One","default_flag":true,"default_country":"TST","selectable_countries":[{"tag":"TST","history_en":"Hi"}]}`, 201,
		`"edit_revision":"1"`, `"bookmark_id":"bm1"`, `"date":"1936.1.1.12"`, `"picture":"GFX_select_date_1936"`,
		`"history_locs":{"en":"Hi"}`, `"ideology":""`, `"code":"bookmark.selectable_ideology_missing"`, `"change":"introduced"`)
	if got := rec.Header().Get("X-Edit-Revisions"); got != `{"bookmark:1":"1"}` {
		t.Fatalf("create revision header %q", got)
	}
	e.expect("POST", "/bookmarks", `{"bookmark_id":"bm1"}`, 409, "already exists")

	stale := map[string]string{"X-Edit-Revisions": `{"bookmark:1":"5"}`}
	e.check(e.request("PUT", "/bookmarks/1", `{"name_en":"Uno"}`, stale), "stale PUT", 409, `"code":"save_conflict"`, `"current_revision":"1"`)
	e.check(e.request("PUT", "/bookmarks/1", `{}`, map[string]string{"X-Edit-Revisions": `not json`}), "bad header", 422, "invalid_edit_revisions")
	rec = e.request("PUT", "/bookmarks/1", `{"name_en":"Uno","filters":["label"],"default_flag":null}`, map[string]string{"X-Edit-Revisions": `{"bookmark:1":"1"}`})
	e.check(rec, "PUT", 200, `"edit_revision":"2"`, `"name_en":"Uno"`, `"filters":["label"]`, `"default_flag":false`)
	if got := rec.Header().Get("X-Edit-Revisions"); got != `{"bookmark:1":"2"}` {
		t.Fatalf("update revision header %q", got)
	}
	e.expect("PUT", "/bookmarks/1", `{"bookmark_id":"  "}`, 400, "bookmark_id cannot be empty")
	e.expect("POST", "/bookmarks", `{"bookmark_id":"bm0","source":"imported"}`, 201)
	e.expect("PUT", "/bookmarks/1", `{"bookmark_id":"bm0"}`, 409, "bookmark_id conflict")
	list := decode[[]map[string]any](t, e.expect("GET", "/bookmarks", "", 200, `"diagnostics":[`))
	if len(list) != 2 || list[0]["bookmark_id"] != "bm0" {
		t.Fatalf("list order %v", list)
	}
	e.expect("PUT", "/bookmarks/2", `{"name_en":"Zero"}`, 200, `"source":"created"`)
	e.expect("GET", "/bookmarks/1", "", 200, `"edit_revision":"2"`)
	e.expect("GET", "/bookmarks/9", "", 404, "Bookmark not found")

	vanillaPicture := e.expect("GET", "/bookmarks/1/picture", "", 200)
	if vanillaPicture.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("vanilla picture type %q", vanillaPicture.Header().Get("Content-Type"))
	}
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, _ := writer.CreateFormFile("file", "pic.png")
	upload := image.NewRGBA(image.Rect(0, 0, 360, 208))
	for i := range upload.Pix {
		upload.Pix[i] = 200
	}
	_ = png.Encode(part, upload)
	_ = writer.Close()
	req := httptest.NewRequest("POST", "/api/go/projects/1/bookmarks/1/upload-picture", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Edit-Revisions", `{"bookmark:1":"2"}`)
	up := httptest.NewRecorder()
	e.mux.ServeHTTP(up, req)
	e.check(up, "upload", 200, `"picture":"GFX_hagane_select_date_bm1"`, `"filename":"bookmark_bm1.png"`, `"preview_url":"/api/go/projects/1/bookmarks/1/picture"`)
	if got := up.Header().Get("X-Edit-Revisions"); got != `{"bookmark:1":"3"}` {
		t.Fatalf("upload revision header %q", got)
	}
	served := e.expect("GET", "/bookmarks/1/picture", "", 200)
	img, err := png.Decode(served.Body)
	if err != nil || img.Bounds().Dx() != 180 || img.Bounds().Dy() != 104 {
		t.Fatalf("served picture %v %v", img.Bounds(), err)
	}
	e.expect("GET", "/bookmarks/1", "", 200, `"picture":"GFX_hagane_select_date_bm1"`, `"edit_revision":"3"`)

	e.check(e.request("DELETE", "/bookmarks/1", "", stale), "stale DELETE", 409, "save_conflict")
	e.expect("DELETE", "/bookmarks/1", "", 204)
	e.expect("DELETE", "/bookmarks/1", "", 404, "save_target_not_found")
	e.expect("GET", "/bookmarks/1/picture", "", 404, "Asset not uploaded")
}
