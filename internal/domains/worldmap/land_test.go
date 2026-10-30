package worldmap

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
)

func landRequestOf(t *testing.T, fields map[string]string) *http.Request {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for i := range picture.Pix {
		picture.Pix[i] = 200
	}
	picture.Set(0, 0, color.RGBA{160, 210, 100, 255})
	var sketch bytes.Buffer
	_ = png.Encode(&sketch, picture)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("sketch", "sketch.png")
	_, _ = part.Write(sketch.Bytes())
	for key, value := range fields {
		_ = form.WriteField(key, value)
	}
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/projects/7/gui-render/world-map/land", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}

func TestParseLandRequest(t *testing.T) {
	req, problem := parseLandRequest(httptest.NewRecorder(), landRequestOf(t, map[string]string{
		"x": "200", "y": "1100", "width": "120", "height": "60", "seed": "9", "land": "300",
		"name": " Atlantis ", "owner": "ger", "languages": "en, zh",
	}))
	if problem != "" {
		t.Fatal(problem)
	}
	if req.options.Place != image.Rect(200, 1100, 320, 1160) || req.options.Seed != 9 || req.options.LandProvincePixels != 300 ||
		req.name != "Atlantis" || req.owner != "GER" || strings.Join(req.languages, ",") != "en,zh" {
		t.Errorf("parsed %+v", req)
	}
	for fields, want := range map[string]map[string]string{
		"太大":     {"x": "0", "y": "0", "width": "4096", "height": "2048"},
		"国家标签":   {"x": "0", "y": "0", "width": "10", "height": "10", "owner": "GERMANY"},
		"不认识的语言": {"x": "0", "y": "0", "width": "10", "height": "10", "languages": "xx"},
	} {
		if _, problem := parseLandRequest(httptest.NewRecorder(), landRequestOf(t, want)); !strings.Contains(problem, fields) {
			t.Errorf("%v: %q, want it to mention %q", want, problem, fields)
		}
	}
	if _, problem := parseLandRequest(httptest.NewRecorder(), landRequestOf(t, map[string]string{"x": "0", "y": "0", "width": "10", "height": "10"})); problem != "" {
		t.Errorf("defaults: %s", problem)
	}
}

func TestNewFindingsAreTheOnesTheMapDidNotHave(t *testing.T) {
	after := []mapgen.Finding{
		{Code: "x_crossing", Severity: "warning", Count: 176},
		{Code: "disconnected_province", Severity: "warning", Count: 491},
		{Code: "small_province", Severity: "warning", Count: 2},
	}
	got := newFindings(after, map[string]int{"x_crossing": 174, "disconnected_province": 491})
	if len(got) != 2 || got[0].Code != "x_crossing" || got[0].Count != 2 || got[1].Code != "small_province" || got[1].Count != 2 {
		t.Errorf("new findings %+v", got)
	}
}

func TestLandMapPaths(t *testing.T) {
	for _, p := range []string{"map/provinces.bmp", "map/definition.csv", "map/unitstacks.txt", "map/buildings.txt",
		"map/world_normal.bmp", "map/trees.bmp", "map/terrain/colormap_rgb_cityemissivemask_a.dds"} {
		if !isLandMapPath(p) {
			t.Errorf("%s not a map file", p)
		}
	}
	for _, p := range []string{"map/strategicregions/1-x.txt", "map/continent.txt", "common/map/provinces.bmp", "gfx/provinces.bmp",
		"map/terrain/colormap_water_0.dds"} {
		if isLandMapPath(p) {
			t.Errorf("%s taken for a map file", p)
		}
	}
	if got := safeFileStem("Neu-Atlantis: 2"); got != "Neu-Atlantis 2" {
		t.Errorf("stem %q", got)
	}
	if got := safeFileStem("亚特兰蒂斯"); got != "New Land" {
		t.Errorf("stem %q", got)
	}
}

func TestSwitchWater(t *testing.T) {
	const w, h = 40, 10
	base := &mapgen.Base{Width: w, Height: h, Province: make([]int32, w*h), Terrain: make([]uint8, w*h)}
	for p := range base.Province {
		switch x := p % w; {
		case x < 10:
			base.Province[p] = 1
		case x < 20:
			base.Province[p] = 2
		default:
			base.Province[p] = 3
		}
	}
	base.Definitions = []mapgen.Definition{
		{ID: 1, Kind: "land", Coastal: true, Terrain: "plains", Continent: 1},
		{ID: 2, Kind: "sea", Coastal: true, Terrain: "ocean"},
		{ID: 3, Kind: "sea", Terrain: "ocean"},
	}
	if problem := switchWater(base, &base.Definitions[1], "lake"); !strings.Contains(problemDetail(problem), "#1") {
		t.Errorf("stranding province 1: %q", problem)
	}
	if base.Definitions[1].Kind != "sea" {
		t.Error("a refused switch changed the row")
	}
	if problem := switchWater(base, &base.Definitions[2], "lake"); problem != nil {
		t.Fatal(problem)
	}
	if d := base.Definitions[2]; d.Kind != "lake" || d.Terrain != "lakes" || d.Coastal || base.Terrain[30] != mapgen.Legend[mapgen.Lake].Terrain || base.Terrain[15] != 0 {
		t.Errorf("lake: %+v terrain %d / %d", d, base.Terrain[30], base.Terrain[15])
	}
	if problem := switchWater(base, &base.Definitions[2], "sea"); problem != nil {
		t.Fatal(problem)
	}
	if d := base.Definitions[2]; d.Kind != "sea" || d.Coastal || base.Terrain[30] != mapgen.Legend[mapgen.Sea].Terrain {
		t.Errorf("back to sea: %+v", d)
	}
}

func TestTheProjectsOwnMapFilesAreTheMapAnEditStartsFrom(t *testing.T) {
	own := "1;naval_base;3400.00;10.00;1000.00;0.00;5240\n"
	definition := "0;0;0;0;land;false;unknown;0\n1;10;20;30;land;false;plains;1\n"
	sha, key := "aa", "prefix/rawblobs/aa/aa"
	set, err := newLandFileSet(nil, nil, nil, []RawFile{
		{DestPath: "map/buildings.txt", Content: own, RawKind: "verbatim_override"},
		{DestPath: "map/unitstacks.txt", Content: "1;0;1.00;2.00;3.00;0.00;0.00\n"},
		{DestPath: "map/definition.csv", Content: definition, RawKind: "verbatim_override"},
		{DestPath: "map/provinces.bmp", IsBinary: true, RawKind: "verbatim_override", ContentSHA256: &sha, ContentOBSKey: &key},
	}, nil)
	if err != nil {
		t.Fatalf("a mod's own map refused: %v", err)
	}
	for name, want := range map[string]string{"map/buildings.txt": own, "map/definition.csv": definition} {
		body, err := set.file(name)
		if err != nil || string(body) != want {
			t.Fatalf("%s = %q, %v; want the project's own", name, body, err)
		}
	}
	if !set.ownMap() {
		t.Fatal("a project with map files of its own must not reuse the base game's cached checks")
	}

	for _, row := range []RawFile{
		{DestPath: "map/provinces.bmp", IsBinary: true, RawKind: "verbatim_override"},
		{DestPath: "map/buildings.txt", IsBinary: true, RawKind: "verbatim_override"},
		{DestPath: "map/unitstacks.txt", RawKind: "empty_shadow"},
	} {
		_, err := newLandFileSet(nil, nil, nil, []RawFile{row}, nil)
		var refusal *landRefusal
		if !errors.As(err, &refusal) || !strings.Contains(refusal.message, row.DestPath) {
			t.Errorf("%s (binary %v, %q): err = %v, want a refusal naming it", row.DestPath, row.IsBinary, row.RawKind, err)
		}
	}

	if _, err := newLandFileSet(nil, nil, nil, []RawFile{{DestPath: "map/provinces.bmp", IsBinary: true}},
		[]WorldMapAddition{{RawPaths: []string{"map/provinces.bmp"}}}); err != nil {
		t.Fatalf("an addition's own map refused: %v", err)
	}
	if empty, _ := newLandFileSet(nil, nil, nil, nil, nil); empty.ownMap() {
		t.Fatal("a project without map files of its own is on the base game's map")
	}
}
