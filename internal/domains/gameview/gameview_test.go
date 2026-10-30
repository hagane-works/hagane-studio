package gameview

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

func TestMain(m *testing.M) {

	sizeCacheDir = func() string { return "" }
	os.Exit(m.Run())
}

const testGUI = `guiTypes = {
	containerWindowType = {
		name = "test_window"
		position = { x = 0 y = 0 }
		size = { width = 200 height = 100 }
		iconType = {
			name = "test_icon"
			spriteType = "GFX_test_icon"
			position = { x = 10 y = 10 }
		}
		instantTextboxType = {
			name = "test_text"
			text = "TEST_KEY"
			position = { x = 20 y = 20 }
		}
	}
}
`

const testGFX = `spriteTypes = {
	spriteType = {
		name = "GFX_test_icon"
		texturefile = "gfx/interface/test_icon.dds"
	}
}
`

func testDDS(w, h int) []byte {
	pix := make([]byte, w*h*4)
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+1], pix[i+2], pix[i+3] = 200, 30, 30, 255
	}
	return imgcodec.EncodeDXT5DDS(pix, w, h)
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{20, 120, 220, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeGame(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string][]byte{
		"launcher-settings.json":                                   []byte(`{"rawVersion":"1.2.3.0"}`),
		"common/.keep":                                             nil,
		"interface/test.gui":                                       []byte(testGUI),
		"interface/test.gfx":                                       []byte(testGFX),
		"gfx/interface/test_icon.dds":                              testDDS(8, 8),
		"gfx/cursors/normal.dds":                                   testDDS(4, 4),
		"gfx/models/units/GER_infantry_diffuse.dds":                testDDS(4, 4),
		"localisation/english/test_l_english.yml":                  []byte("\ufeffl_english:\n TEST_KEY:0 \"Hello\"\n"),
		"localisation/simp_chinese/test_l_simp_chinese.yml":        []byte("\ufeffl_simp_chinese:\n TEST_KEY:0 \"你好\"\n"),
		"dlc/dlc001_test/interface/dlc.gui":                        []byte(strings.ReplaceAll(testGUI, "test_window", "dlc_window")),
		"dlc/dlc001_test/localisation/english/dlc_l_english.yml":   []byte("\ufeffl_english:\n DLC_KEY:0 \"From the DLC\"\n"),
		"integrated_dlc/idlc_test/localisation/english/x.yml":      []byte("\ufeffl_english:\n TEST_KEY:0 \"Hello again\"\n"),
		"integrated_dlc/idlc_test/gfx/interface/idlc_picture.dds":  testDDS(16, 8),
		"integrated_dlc/idlc_test/interface/idlc_picture_gfx.gfx":  []byte(strings.ReplaceAll(strings.ReplaceAll(testGFX, "GFX_test_icon", "GFX_idlc_picture"), "test_icon.dds", "idlc_picture.dds")),
		"documentation/ignored.txt":                                []byte("not read"),
		"history/countries/TST - Testland.txt":                     []byte("capital = 1\n"),
		"interface/subfolder/nested.gfx":                           []byte("spriteTypes = { }\n"),
		"gfx/interface/subfolder/nested_unused.dds":                testDDS(4, 4),
		"gfx/leaders/TST/portrait_tst_leader.dds":                  testDDS(4, 4),
		"map/definition.csv":                                       []byte("0;0;0;0;land;false;unknown;0\n"),
		"map/unitstacks.txt":                                       []byte("not read either"),
		"common/country_tags/00_countries.txt":                     []byte("TST = \"countries/Testland.txt\"\n"),
		"common/countries/Testland.txt":                            []byte("color = { 1 2 3 }\n"),
		"tutorial/tutorial.txt":                                    []byte("tutorial = { }\n"),
		"gfx/fonts/hoi_16mbs.fnt":                                  []byte("info face=\"x\"\n"),
		"gfx/fonts/hoi_16mbs.dds":                                  testDDS(8, 8),
		"gfx/loadingscreens/load_1.dds":                            testDDS(16, 16),
		"integrated_dlc/idlc_test/gfx/interface/goals/goal_x.dds":  testDDS(4, 4),
		"integrated_dlc/idlc_test/common/national_focus/empty.txt": []byte("\n"),
	}
	for name, body := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if body != nil {
			if err := os.WriteFile(target, body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

type fixture struct {
	t   *testing.T
	p   *project.Project
	mux *http.ServeMux
	svc *Service
}

func newFixture(t *testing.T, game func() (fs.FS, error)) *fixture {
	t.Helper()
	store := project.NewStore(t.TempDir())
	record, err := workbench.CreateProject(store, workbench.CreateInput{Name: "Test Mod", CountryTag: "TST", CountryName: "Testland"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Get(project.Int64(record["id"]))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(domains.Deps{Open: func(*http.Request) (*project.Project, error) { return p, nil }, Game: game})
	mux := http.NewServeMux()
	RegisterWith(mux, svc)
	return &fixture{t: t, p: p, mux: mux, svc: svc}
}

func newGameFixture(t *testing.T) *fixture {
	dir := writeGame(t)
	return newFixture(t, func() (fs.FS, error) { return os.DirFS(dir), nil })
}

func (f *fixture) do(method, target string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	f.t.Helper()
	request := httptest.NewRequest(method, target, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, request)
	return rec
}

const prefix = "/api/go/projects/1/gui-render/"

func (f *fixture) get(route string, want int) map[string]any {
	f.t.Helper()
	rec := f.do(http.MethodGet, prefix+route, nil, "")
	return decode(f.t, rec, want)
}

func (f *fixture) post(route string, body any, want int) map[string]any {
	f.t.Helper()
	raw, _ := json.Marshal(body)
	rec := f.do(http.MethodPost, prefix+route, bytes.NewReader(raw), "application/json")
	return decode(f.t, rec, want)
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, want int) map[string]any {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, want %d: %s", rec.Code, want, rec.Body.String())
	}
	out := map[string]any{}
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode %q: %v", rec.Body.String(), err)
		}
	}
	return out
}

func windowNames(payload map[string]any) map[string]bool {
	out := map[string]bool{}
	rows, _ := payload["windows"].([]any)
	for _, row := range rows {
		if window, ok := row.(map[string]any); ok {
			out[str(window["name"])] = true
		}
	}
	return out
}

func TestStatusWindowsAndWindow(t *testing.T) {
	f := newGameFixture(t)
	status := f.get("status", 200)
	if status["resolved"] != true || status["vanilla_root"] != "1.2.3.0" || status["windows"].(float64) < 2 {
		t.Fatalf("status = %v", status)
	}
	names := windowNames(f.get("windows", 200))
	if !names["test_window"] || !names["dlc_window"] {
		t.Fatalf("windows = %v; want the base game's and the DLC's", names)
	}
	window := f.get("window?name=test_window", 200)
	if window["file"] != "interface/test.gui" || window["revision"] != "0" || len(str(window["scaffold_hash"])) != 64 {
		t.Fatalf("window envelope = file %v revision %v scaffold %v", window["file"], window["revision"], window["scaffold_hash"])
	}
	if got := f.get("window?name=test_window&file=interface/test.gui", 200); got["file"] != "interface/test.gui" {
		t.Fatalf("file-qualified window = %v", got["file"])
	}
	f.get("window?name=test_window&file=interface/dlc.gui", 404)
	f.get("window?name=nope", 404)
	f.get("window", 422)
	f.get("window?name=test_window&file=../x.gui", 422)
	f.get("nonsense", 404)
	if rec := f.do(http.MethodPost, prefix+"windows", nil, ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST windows = %d", rec.Code)
	}
}

func TestLocInTheGamesLanguages(t *testing.T) {
	f := newGameFixture(t)
	texts := f.get("loc?keys=TEST_KEY,DLC_KEY,MISSING", 200)["texts"].(map[string]any)

	if texts["TEST_KEY"] != "Hello again" || texts["DLC_KEY"] != "From the DLC" || texts["MISSING"] != nil {
		t.Fatalf("texts = %v", texts)
	}
	zh := f.get("loc?keys=TEST_KEY&lang=zh", 200)["texts"].(map[string]any)
	if zh["TEST_KEY"] != "你好" {
		t.Fatalf("zh texts = %v", zh)
	}
	f.get("loc?keys=bad%20key", 422)
	f.get("loc?keys=", 422)
}

func TestSpritesAndBaseGameTextures(t *testing.T) {
	f := newGameFixture(t)
	for _, target := range []string{
		prefix + "sprite?name=GFX_test_icon",
		prefix + "sprite?name=GFX_idlc_picture",
		"/api/go/gui-vanilla/gfx/interface/test_icon.png",
		"/api/go/gui-vanilla/gfx/cursors/normal.png",
		"/api/go/gui-vanilla/gfx/models/units/ger_infantry_diffuse.png",
	} {
		rec := f.do(http.MethodGet, target, nil, "")
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s: %d %s", target, rec.Code, rec.Body.String())
		}
		if _, err := png.DecodeConfig(bytes.NewReader(rec.Body.Bytes())); err != nil {
			t.Fatalf("%s: not a PNG: %v", target, err)
		}
	}
	for _, target := range []string{
		"/api/go/gui-vanilla/gfx/interface/missing.png",
		"/api/go/gui-vanilla/gfx/interface/test_icon.dds",
	} {
		if rec := f.do(http.MethodGet, target, nil, ""); rec.Code != 404 {
			t.Fatalf("%s: %d", target, rec.Code)
		}
	}
	f.get("sprite?name=GFX_missing", 404)
	f.get("sprite", 422)
	sprites := f.get("sprites?names=GFX_test_icon", 200)["sprites"].(map[string]any)
	sprite, _ := sprites["GFX_test_icon"].(map[string]any)
	if url := str(sprite["vanilla_url"]); url != "/api/go/gui-vanilla/gfx/interface/test_icon.png" {
		t.Fatalf("sprite payload = %v", sprite)
	}
}

func TestWithoutAGame(t *testing.T) {
	f := newFixture(t, func() (fs.FS, error) { return nil, errors.New("no game folder") })

	status := f.get("status", 200)
	if status["vanilla_root"] != nil || status["windows"].(float64) != 0 {
		t.Fatalf("status = %v", status)
	}
	if names := windowNames(f.get("windows", 200)); len(names) != 0 {
		t.Fatalf("windows = %v", names)
	}
	f.get("window?name=test_window", 404)
	if got := f.get("frontend", 200); got["version"] != "" {
		t.Fatalf("frontend = %v", got)
	}
	if f.svc.BaseIndex() != nil {
		t.Fatal("BaseIndex without a game")
	}
	if _, err := f.svc.ReadGameFile("interface/test.gui", 1<<20); err == nil {
		t.Fatal("ReadGameFile without a game")
	}
	if rec := f.do(http.MethodGet, "/api/go/gui-vanilla/gfx/interface/test_icon.png", nil, ""); rec.Code != 503 {
		t.Fatalf("texture without a game = %d", rec.Code)
	}
}

func TestDataRoutes(t *testing.T) {
	f := newGameFixture(t)
	if got := f.get("frontend", 200); got["version"] != "1.2.3.0" {
		t.Fatalf("frontend = %v", got)
	}
	f.get("bookmarks", 200)
	f.get("alerts", 200)
	f.get("diplomacy?tag=TST&date=1936.1.1", 200)
	f.get("diplomacy?tag=TST%0A", 422)
	f.get("unit-visual?tag=TST&unit=infantry", 404)
	f.get("focus-tree?id=nope", 404)
	f.get("event?id=nope.1", 404)
	choices := f.get("event-choices?q=", 200)
	if choices["scope"] != "all" || choices["offset"].(float64) != 0 {
		t.Fatalf("event-choices = %v", choices)
	}
	f.get("tech-canvas/layout", 422)
	layout := f.get("tech-canvas/layout?folder=infantry_folder", 200)
	if layout["folder"] != "infantry_folder" || layout["source"] != nil {
		t.Fatalf("layout = %v", layout)
	}
	f.get("states?tag="+strings.Repeat("x", 300), 422)
}

func TestTextRewritesShowAndGoBack(t *testing.T) {
	f := newGameFixture(t)
	before := f.get("data-version", 200)["version"]
	if got := f.get("text", 200)["overrides"].(map[string]any); len(got) != 0 {
		t.Fatalf("overrides = %v", got)
	}
	f.post("text", map[string]any{"key": "TEST_KEY", "value": "Rewritten"}, 200)
	if got := f.get("text", 200)["overrides"].(map[string]any); got["TEST_KEY"] != "Rewritten" {
		t.Fatalf("overrides = %v", got)
	}
	if after := f.get("data-version", 200)["version"]; after == before {
		t.Fatal("data-version did not change after a write")
	}

	if texts := f.get("loc?keys=TEST_KEY", 200)["texts"].(map[string]any); texts["TEST_KEY"] != "Rewritten" {
		t.Fatalf("after the rewrite TEST_KEY = %v", texts["TEST_KEY"])
	}
	f.post("text", map[string]any{"key": "bad key", "value": "x"}, 422)
	f.get("text?lang=xx", 422)
	rec := f.do(http.MethodDelete, prefix+"text?key=TEST_KEY", nil, "")
	if got := decode(t, rec, 200); got["removed"] != true {
		t.Fatalf("delete = %v", got)
	}
	if texts := f.get("loc?keys=TEST_KEY", 200)["texts"].(map[string]any); texts["TEST_KEY"] != "Hello again" {
		t.Fatalf("after the restore TEST_KEY = %v", texts["TEST_KEY"])
	}
	rows, _ := f.p.Rows("project_loc_entries")
	for _, row := range rows {
		if str(row["key"]) == "TEST_KEY" {
			t.Fatalf("restore left a row: %v", row)
		}
	}
}

func findNode(node map[string]any, name string) map[string]any {
	if node == nil {
		return nil
	}
	if str(node["name"]) == name {
		return node
	}
	children, _ := node["children"].([]any)
	for _, child := range children {
		if found := findNode(mapOf(child), name); found != nil {
			return found
		}
	}
	return nil
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func TestOverlayPreviewSaveAndValidate(t *testing.T) {
	f := newGameFixture(t)
	overlay := f.get("overlay?file=interface/test.gui", 200)
	if overlay["exists"] != false || overlay["revision"] != "0" || overlay["materializable"] != true || overlay["source"] != "vanilla" {
		t.Fatalf("overlay = %v", overlay)
	}
	hash := str(overlay["scaffold_hash"])
	edits := []map[string]any{{"op": "set_layout", "target": "test_window/test_icon", "payload": map[string]any{"x": 77, "y": 33}}}
	preview := f.post("overlay/preview", map[string]any{
		"file": "interface/test.gui", "window_name": "test_window", "expected_revision": "0",
		"expected_scaffold_hash": hash, "edits": edits,
	}, 200)
	icon := findNode(mapOf(mapOf(preview["window"])["node"]), "test_icon")
	if preview["write_performed"] != false || icon == nil || !strings.Contains(mustJSON(icon), "77") {
		t.Fatalf("preview = %v", mustJSON(preview["window"]))
	}
	f.post("overlay/preview", map[string]any{"file": "interface/test.gui", "window_name": "test_window", "expected_revision": "9"}, 409)

	write := map[string]any{"file": "interface/test.gui", "edits": edits, "scaffold_hash": hash, "expected_scaffold_hash": hash}
	f.post("overlay", write, 428)
	write["expected_revision"] = "2020-01-01T00:00:00Z"
	f.post("overlay", write, 409)
	write["expected_revision"] = "0"
	saved := f.post("overlay", write, 200)
	revision := str(saved["revision"])
	if revision == "0" || saved["source"] != "vanilla" {
		t.Fatalf("saved = %v", saved)
	}

	f.post("overlay", write, 409)
	read := f.get("overlay?file=test.gui", 200)
	if read["exists"] != true || read["revision"] != revision || read["scaffold_hash_pinned"] != true || read["scaffold_drift"] != false {
		t.Fatalf("read back = %v", read)
	}

	window := f.get("window?name=test_window", 200)
	if icon := findNode(mapOf(window["node"]), "test_icon"); !strings.Contains(mustJSON(icon), "77") || window["revision"] != revision {
		t.Fatalf("window after save = %v", mustJSON(icon))
	}
	validated := f.post("overlay/validate", map[string]any{"file": "interface/test.gui", "window_name": "test_window", "expected_revision": revision}, 200)
	if validated["valid"] != true || validated["saved"] != true {
		t.Fatalf("validate = %v", validated)
	}
	f.post("overlay/validate", map[string]any{"file": "interface/other.gui", "window_name": "x"}, 404)

	rows, _ := f.p.Rows("project_obs_assets")
	pinned := false
	for _, row := range rows {
		if str(row["asset_type"]) == vanillaScaffoldAssetType && str(row["gfx_name"]) == hash && f.p.Blobs().Exists(str(row["obs_key"])) {
			pinned = true
		}
	}
	if !pinned {
		t.Fatalf("no pinned scaffold in %v", rows)
	}

	write["expected_revision"] = revision
	if again := f.post("overlay", write, 200); again["revision"] == revision {
		t.Fatal("a save kept the old revision")
	}

	rec := f.do(http.MethodPost, prefix+"export?file=interface/test.gui", nil, "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("export = %d %s", rec.Code, rec.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var members []string
	for _, member := range archive.File {
		members = append(members, member.Name)
	}
	if !strings.Contains(strings.Join(members, ","), "interface/test.gui") {
		t.Fatalf("bundle members = %v", members)
	}
	if rec := f.do(http.MethodPost, prefix+"export?file=interface/none.gui", nil, ""); rec.Code != 409 {
		t.Fatalf("export without overlay = %d", rec.Code)
	}
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func multipartUpload(t *testing.T, fields map[string]string, picture []byte) (io.Reader, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}
	if picture != nil {
		part, err := writer.CreateFormFile("file", "upload.png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(picture)
	}
	_ = writer.Close()
	return &body, writer.FormDataContentType()
}

func rawFilesOf(t *testing.T, p *project.Project) map[string]project.Row {
	t.Helper()
	rows, err := p.Rows("project_raw_files")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]project.Row{}
	for _, row := range rows {
		out[str(row["dest_path"])] = row
	}
	return out
}

func TestSpriteReplaceAndRestore(t *testing.T) {
	f := newGameFixture(t)
	body, contentType := multipartUpload(t, map[string]string{"name": "GFX_test_icon"}, testPNG(t, 32, 32))
	got := decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 200)
	exported := mapOf(got["export"])
	if got["width"].(float64) != 8 || got["height"].(float64) != 8 || exported["gfx"] != "interface/test.gfx" {
		t.Fatalf("replace = %v", got)
	}
	texture := str(exported["texture"])
	files := rawFilesOf(t, f.p)
	gfx, dds := files["interface/test.gfx"], files[texture]
	if gfx == nil || !strings.Contains(str(gfx["content"]), texture) || dds == nil || dds["is_binary"] != true {
		t.Fatalf("raw files = %v", files)
	}
	if !f.p.Blobs().Exists(str(dds["content_obs_key"])) {
		t.Fatal("the .dds bytes were not stored")
	}

	rec := f.do(http.MethodGet, prefix+"sprite?name=GFX_test_icon", nil, "")
	config, err := png.DecodeConfig(bytes.NewReader(rec.Body.Bytes()))
	if rec.Code != 200 || err != nil || config.Width != 8 {
		t.Fatalf("sprite after replace: %d %v %v", rec.Code, err, config)
	}
	sprite := mapOf(f.get("sprites?names=GFX_test_icon", 200)["sprites"].(map[string]any)["GFX_test_icon"])
	if !strings.Contains(mustJSON(sprite), "gui-render/sprite?name=GFX_test_icon") {
		t.Fatalf("sprite payload after replace = %v", sprite)
	}

	body, contentType = multipartUpload(t, map[string]string{"name": "GFX_test_icon", "mode": "fit"}, testPNG(t, 16, 8))
	decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 200)

	restored := decode(t, f.do(http.MethodDelete, prefix+"sprite/override?name=GFX_test_icon", nil, ""), 200)
	if restored["resolved"] != true || restored["width"].(float64) != 8 {
		t.Fatalf("restore = %v", restored)
	}
	if files := rawFilesOf(t, f.p); len(files) != 0 {
		t.Fatalf("restore left raw files: %v", files)
	}
	if overrides := spriteOverrideRows(f.p); len(overrides) != 0 {
		t.Fatalf("restore left the pointer: %v", overrides)
	}

	decode(t, f.do(http.MethodDelete, prefix+"sprite/override?name=GFX_test_icon", nil, ""), 200)
}

func TestSpriteReplaceRefusals(t *testing.T) {
	f := newGameFixture(t)
	body, contentType := multipartUpload(t, map[string]string{"name": "GFX_unknown"}, testPNG(t, 8, 8))
	decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 404)
	body, contentType = multipartUpload(t, map[string]string{"name": "GFX_test_icon"}, nil)
	decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 422)
	body, contentType = multipartUpload(t, map[string]string{"name": "GFX_test_icon", "mode": "zoom"}, testPNG(t, 8, 8))
	decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 422)
	body, contentType = multipartUpload(t, map[string]string{"name": "GFX_test_icon"}, []byte("not a picture"))
	decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 400)
	if files := rawFilesOf(t, f.p); len(files) != 0 {
		t.Fatalf("a refused upload wrote raw files: %v", files)
	}

	body, contentType = multipartUpload(t, map[string]string{"name": "GFX_test_icon_own", "create": "1", "width": "12", "height": "6"}, testPNG(t, 12, 6))
	created := decode(t, f.do(http.MethodPost, prefix+"sprite/override", body, contentType), 200)
	if gfx := str(mapOf(created["export"])["gfx"]); !strings.HasPrefix(gfx, "interface/game_view_generated_") {
		t.Fatalf("created = %v", created)
	}
	decode(t, f.do(http.MethodDelete, prefix+"sprite/override?name=GFX_test_icon_own", nil, ""), 200)
	if files := rawFilesOf(t, f.p); len(files) != 0 {
		t.Fatalf("restoring a created sprite left raw files: %v", files)
	}
}

func writeSnapshot(t *testing.T, p *project.Project, members map[string][]byte) {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, body := range members {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(body)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	key := "local/p/" + p.Blobs().ProjectUUID() + "/gui-source/snapshot.zip"
	if err := p.Blobs().Put(key, "application/zip", int64(buf.Len()), bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if err := p.Write([]string{"project_obs_assets"}, func(tx *project.Tx) error {
		upsertAsset(tx, "gui_source", "snapshot.zip", key, map[string]any{})

		tx.SetRecord(func(record project.Row) { record["is_loaded_existing_mod"] = true })
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestImportedMod(t *testing.T) {
	f := newGameFixture(t)
	modGUI := strings.ReplaceAll(strings.ReplaceAll(testGUI, "test_window", "mod_window"), "GFX_test_icon", "GFX_mod_icon")
	writeSnapshot(t, f.p, map[string][]byte{
		"interface/mod.gui":                              []byte(modGUI),
		"interface/mod.gfx":                              []byte(strings.ReplaceAll(strings.ReplaceAll(testGFX, "GFX_test_icon", "GFX_mod_icon"), "test_icon.dds", "Mod_Icon.dds")),
		"gfx/interface/Mod_Icon.dds":                     testDDS(16, 16),
		"gfx/fonts/AldrichTNOV13.ttf":                    []byte("font bytes"),
		"common/scripted_guis/panels.txt":                []byte("scripted_gui = { }\n"),
		"snapshot_meta/game_data.txt":                    []byte("1\n"),
		"localisation/english/mod_l_english.yml":         []byte("\ufeffl_english:\n MOD_KEY:0 \"From the mod\"\n"),
		"common/scripted_effects/nested/ignored_one.txt": []byte("x = { }\n"),
	})
	names := windowNames(f.get("windows", 200))
	if !names["mod_window"] || !names["test_window"] {
		t.Fatalf("windows = %v; want the mod's over the base game's", names)
	}

	imported, found, err := f.svc.ImportedIndex(context.Background(), f.p)
	if err != nil || !found {
		t.Fatal(found, err)
	}
	if text, ok := imported.Localise("MOD_KEY"); !ok || text != "From the mod" {
		t.Fatalf("imported MOD_KEY = %q", text)
	}

	rec := f.do(http.MethodGet, prefix+"sprite?name=GFX_mod_icon", nil, "")
	config, err := png.DecodeConfig(bytes.NewReader(rec.Body.Bytes()))
	if rec.Code != 200 || err != nil || config.Width != 16 {
		t.Fatalf("mod sprite: %d %v", rec.Code, rec.Body.String())
	}

	if overlay := f.get("overlay?file=interface/mod.gui", 200); overlay["source"] != "import" || overlay["materializable"] != true {
		t.Fatalf("mod overlay = %v", overlay)
	}
	if rec := f.do(http.MethodGet, prefix+"font?family=aldrich", nil, ""); rec.Code != 200 || rec.Body.String() != "font bytes" {
		t.Fatalf("font = %d %s", rec.Code, rec.Body.String())
	}
	f.get("font?family=bomb", 404)
	f.get("font?family=Aldrich", 404)
	f.get("font", 422)
	points := f.get("scripted-gui-injection-points", 200)
	if _, ok := points["injection_points"].([]any); !ok {
		t.Fatalf("injection points = %v", points)
	}
	f.get("refresh", 404)
	if status := f.post("refresh", nil, 200); status["resolved"] != true {
		t.Fatalf("refresh = %v", status)
	}
}

func TestInjectionPointsWithoutAnImport(t *testing.T) {
	f := newGameFixture(t)
	points := f.get("scripted-gui-injection-points", 200)
	if list, ok := points["injection_points"].([]any); !ok || len(list) != 0 {
		t.Fatalf("injection points = %v", points)
	}
	f.get("font?family=aldrich", 404)
}

func TestProjectPicturesAreServed(t *testing.T) {
	f := newGameFixture(t)
	picture := testPNG(t, 94, 86)
	key := "local/p/" + f.p.Blobs().ProjectUUID() + "/focus_icon/GFX_focus_test.png"
	if err := f.p.Blobs().Put(key, "image/png", int64(len(picture)), bytes.NewReader(picture)); err != nil {
		t.Fatal(err)
	}
	if err := f.p.Write([]string{"project_obs_icons"}, func(tx *project.Tx) error {
		tx.Insert("project_obs_icons", project.Row{"gfx_name": "GFX_focus_test", "obs_key": key})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rec := f.do(http.MethodGet, prefix+"sprite?name=GFX_focus_test", nil, "")
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), picture) {
		t.Fatalf("project picture: %d", rec.Code)
	}
	sizes, versions := projectSpriteSizes(f.p)
	if sizes["GFX_focus_test"] == ([2]int{}) || versions["GFX_focus_test"] == "" {
		t.Fatalf("sizes %v versions %v", sizes, versions)
	}
}
