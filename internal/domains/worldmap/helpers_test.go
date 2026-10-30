package worldmap

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/image/bmp"

	"haganeworks/internal/domains"
	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const testW, testH = 64, 32

func testProvinceAt(x, y int) int {
	switch {
	case x < 16 && y < 16:
		return 1
	case x < 16:
		return 2
	case x < 32:
		return 3
	}
	return 4
}

var testColours = map[int]color.RGBA{1: {10, 20, 30, 255}, 2: {40, 50, 60, 255}, 3: {70, 80, 90, 255}, 4: {100, 110, 120, 255}}

func encodeBMP(t *testing.T, img image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := bmp.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func testMapImages(t *testing.T) map[string][]byte {
	t.Helper()
	provinces := image.NewRGBA(image.Rect(0, 0, testW, testH))
	heights := image.NewPaletted(provinces.Rect, mapgen.GreyPalette())
	terrain := image.NewPaletted(provinces.Rect, mapgen.GreyPalette())
	rivers := image.NewPaletted(provinces.Rect, mapgen.GreyPalette())
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			id := testProvinceAt(x, y)
			provinces.SetRGBA(x, y, testColours[id])
			i := y*testW + x
			if id == 4 {
				heights.Pix[i], terrain.Pix[i], rivers.Pix[i] = mapgen.SeaHeight, mapgen.Legend[mapgen.Sea].Terrain, mapgen.RiverWater
			} else {
				heights.Pix[i], terrain.Pix[i], rivers.Pix[i] = 100, 0, mapgen.RiverLand
			}
		}
	}
	return map[string][]byte{
		"map/provinces.bmp": encodeBMP(t, provinces), "map/heightmap.bmp": encodeBMP(t, heights),
		"map/terrain.bmp": encodeBMP(t, terrain), "map/rivers.bmp": encodeBMP(t, rivers),
	}
}

const testDefinitions = "0;0;0;0;land;false;unknown;0\n1;10;20;30;land;false;plains;1\n2;40;50;60;land;false;plains;1\n3;70;80;90;land;true;plains;1\n4;100;110;120;sea;true;ocean;0\n"

const testWeather = `	weather = {
		period = {
			between = { 0.0 30.11 }
			temperature = { -5.0 10.0 }
			no_phenomenon = 0.5
			rain_light = 0.5
			rain_heavy = 0.0
			snow = 0.0
			blizzard = 0.0
			arctic_water = 0.0
			mud = 0.0
			sandstorm = 0.0
			min_snow_level = 0.0
		}
	}
`

func testGame(t *testing.T) fstest.MapFS {
	t.Helper()
	files := map[string]string{
		"launcher-settings.json":                         `{"rawVersion":"1.0.0.0"}`,
		"map/definition.csv":                             testDefinitions,
		"map/continent.txt":                              "continents = {\n\teurope\n}\n",
		"map/buildings.txt":                              "",
		"map/unitstacks.txt":                             "",
		"map/supply_nodes.txt":                           "",
		"map/weatherpositions.txt":                       "",
		"map/railways.txt":                               "",
		"map/adjacencies.csv":                            "From;To;Type;Through;start_x;start_y;stop_x;stop_y;adjacency_rule_name;Comment\n-1;-1;;-1;-1;-1;-1;-1;-1\n",
		"map/strategicregions/1-Land.txt":                "strategic_region = {\n\tid = 1\n\tname = \"STRATEGICREGION_1\"\n\tprovinces = {\n\t\t1 2 3\n\t}\n" + testWeather + "}\n",
		"map/strategicregions/2-Sea.txt":                 "strategic_region = {\n\tid = 2\n\tname = \"STRATEGICREGION_2\"\n\tprovinces = {\n\t\t4\n\t}\n\tnaval_terrain = water_deep_ocean\n" + testWeather + "}\n",
		"history/states/1-One.txt":                       "state = {\n\tid = 1\n\tname = \"STATE_1\"\n\tmanpower = 1000\n\tstate_category = rural\n\thistory = {\n\t\towner = TST\n\t\tvictory_points = { 1 5 }\n\t\tbuildings = { infrastructure = 2 }\n\t\tadd_core_of = TST\n\t}\n\tprovinces = { 1 2 }\n}\n",
		"history/states/2-Two.txt":                       "state = {\n\tid = 2\n\tname = \"STATE_2\"\n\tmanpower = 500\n\tstate_category = town\n\thistory = {\n\t\towner = OTH\n\t\tvictory_points = { 3 1 }\n\t\tadd_core_of = OTH\n\t}\n\tprovinces = { 3 }\n}\n",
		"common/country_tags/00_countries.txt":           "TST = \"countries/Testland.txt\"\nOTH = \"countries/Otherland.txt\"\n",
		"common/countries/Testland.txt":                  "color = { 200 50 50 }\n",
		"common/countries/Otherland.txt":                 "color = { 50 50 200 }\n",
		"history/countries/TST - Testland.txt":           "capital = 1\n",
		"history/countries/OTH - Otherland.txt":          "capital = 2\n",
		"localisation/english/state_names_l_english.yml": "\ufeffl_english:\n STATE_1:0 \"One\"\n STATE_2:0 \"Two\"\n",
		"tutorial/tutorial.txt":                          "tutorial = {\n}\n",
	}
	game := fstest.MapFS{}
	for name, body := range files {
		game[name] = &fstest.MapFile{Data: []byte(body)}
	}
	for name, body := range testMapImages(t) {
		game[name] = &fstest.MapFile{Data: body}
	}
	return game
}

type testEnv struct {
	t   *testing.T
	p   *project.Project
	mux *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	sweepIdle(0)
	vanillaFindings.Lock()
	vanillaFindings.counts = nil
	vanillaFindings.Unlock()
	store := project.NewStore(t.TempDir())
	record, err := workbench.CreateProject(store, workbench.CreateInput{Name: "Map Test", CountryTag: "TST", CountryName: "Testland"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Get(project.Int64(record["id"]))
	if err != nil {
		t.Fatal(err)
	}
	game := testGame(t)
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return game, nil },
	})
	return &testEnv{t: t, p: p, mux: mux}
}

const testPrefix = "/api/go/projects/1/gui-render/world-map"

func (e *testEnv) do(method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, testPrefix+path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) get(path string) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.do(http.MethodGet, path, nil, "")
}

func (e *testEnv) postJSON(path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.do(http.MethodPost, path, strings.NewReader(body), "application/json")
}

func (e *testEnv) postForm(path string, files map[string][]byte, fields map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for name, data := range files {
		part, _ := form.CreateFormFile(name, name+".png")
		_, _ = part.Write(data)
	}
	for key, value := range fields {
		_ = form.WriteField(key, value)
	}
	_ = form.Close()
	return e.do(http.MethodPost, path, &body, form.FormDataContentType())
}

func want(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d: %s", rec.Code, status, truncate(rec.Body.String()))
	}
	var out map[string]any
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("body %s: %v", truncate(rec.Body.String()), err)
		}
	}
	return out
}

func truncate(s string) string {
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

func (e *testEnv) rows(table string) []project.Row {
	e.t.Helper()
	rows, err := e.p.Rows(table)
	if err != nil {
		e.t.Fatal(err)
	}
	return rows
}

func (e *testEnv) stateRow(id int) project.Row {
	for _, row := range e.rows(statesTable) {
		if intOf(row["state_id"]) == id {
			return row
		}
	}
	return nil
}

func (e *testEnv) rawFile(dest string) project.Row {
	for _, row := range e.rows(rawFilesTable) {
		if strings.EqualFold(str(row["dest_path"]), dest) {
			return row
		}
	}
	return nil
}

func intsOf(v any) []int {
	list, _ := v.([]any)
	out := make([]int, 0, len(list))
	for _, item := range list {
		out = append(out, intOf(item))
	}
	return out
}

func pngOf(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func itoa(n int) string { return strconv.Itoa(n) }

func mustDecodePNG(t *testing.T, data []byte) *image.RGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	return out
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
