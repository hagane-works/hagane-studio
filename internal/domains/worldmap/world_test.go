package worldmap

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
	"haganeworks/internal/hoi4/mapsketch"
)

func worldRequestOf(t *testing.T, fields map[string]string, sketch bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if sketch {
		request := landRequestOf(t, nil)
		file, _, err := request.FormFile("sketch")
		if err != nil {
			t.Fatal(err)
		}
		part, _ := form.CreateFormFile("sketch", "sketch.png")
		var picture bytes.Buffer
		_, _ = picture.ReadFrom(file)
		_, _ = part.Write(picture.Bytes())
	}
	for key, value := range fields {
		_ = form.WriteField(key, value)
	}
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/projects/7/gui-render/world-map/world", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}

func TestParseWorldRequest(t *testing.T) {
	req, problem := parseWorldRequest(httptest.NewRecorder(), worldRequestOf(t, map[string]string{
		"seed": "9", "land": "3000", "tags": "ger, nwb,GER", "languages": "en,zh",
	}, true))
	if problem != nil {
		t.Fatal(problem)
	}
	if req.source != "sketch" || req.sketch == nil || req.seed != 9 || req.land != 3000 || req.sea != 15000 ||
		strings.Join(req.tags, ",") != "GER,NWB" || strings.Join(req.languages, ",") != "en,zh" {
		t.Errorf("parsed %+v", req)
	}
	game, problem := parseWorldRequest(httptest.NewRecorder(), worldRequestOf(t, map[string]string{"source": "game", "tags": "GER", "per_state": "6"}, false))
	if problem != nil || game.source != "game" || game.sketch != nil || game.perState != 6 || game.languages[0] != "en" {
		t.Errorf("from the base game: %+v %q", game, problem)
	}
	for want, fields := range map[string]map[string]string{
		"至少选一个国家": {"source": "game"},
		"三个字母":    {"source": "game", "tags": "GERMANY"},
		"没有收到草图":  {"tags": "GER"},
		"省份大小":    {"source": "sketch", "tags": "GER", "land": "50"},
		"最多 8 个":  {"source": "game", "tags": "AAA,BBB,CCC,DDD,EEE,FFF,GGG,HHH,III"},
	} {
		_, problem := parseWorldRequest(httptest.NewRecorder(), worldRequestOf(t, fields, want == "省份大小"))
		if !strings.Contains(problemDetail(problem), want) {
			t.Errorf("%v: %q, want it to mention %q", fields, problem, want)
		}
	}
}

func TestWorldSummaryReturnsStateLayoutForNamingStep(t *testing.T) {
	built := &builtWorld{world: &mapgen.World{
		Base: &mapgen.Base{Width: 100, Height: 50},
		All: &mapgen.Addition{
			Provinces: []mapgen.Province{
				{ID: 11, Kind: "land", Pixels: 3, X: 20, Y: 10},
				{ID: 12, Kind: "land", Pixels: 1, X: 40, Y: 30},
				{ID: 21, Kind: "land", Pixels: 2, X: 80, Y: 25},
			},
			States: []mapgen.NewState{
				{Provinces: []int{11, 12}, Capital: 11},
				{Provinces: []int{21}, Capital: 21},
			},
		},
	}}
	req := worldRequest{
		source: "sketch", languages: []string{"en", "zh"}, tags: []string{"NTH", "STH"},
		picturePlacement: &mapsketch.Placement{
			Crop: mapsketch.Rect{X: 10, Y: 20, W: 100, H: 50}, Scale: 1,
			CanvasW: 100, CanvasH: 50,
		},
	}
	summary := worldSummary(built, req, map[int]string{1: "NTH", 2: "STH"}, 802, map[string]bool{"NTH": true})
	layout, ok := summary["state_layout"].([]map[string]any)
	if !ok || len(layout) != 2 {
		t.Fatalf("state_layout = %#v", summary["state_layout"])
	}
	first := layout[0]
	if first["state_id"] != 1 || first["owner"] != "NTH" || first["capital_province"] != 11 || first["province_count"] != 2 {
		t.Fatalf("first state = %#v", first)
	}
	if got := first["map_position"]; fmt.Sprint(got) != "map[x:25 y:15]" {
		t.Fatalf("map_position = %#v", got)
	}
	if got := first["image_position"]; fmt.Sprint(got) != "map[x:35 y:35]" {
		t.Fatalf("image_position = %#v", got)
	}
	status := summary["name_status"].(map[string]any)
	if status["complete"] != false || status["next_tool"] != "name_world_map" || status["total_states"] != 2 {
		t.Fatalf("name_status = %#v", status)
	}
}

func TestParseWorldUpload(t *testing.T) {
	png := func() []byte {
		request := landRequestOf(t, nil)
		file, _, err := request.FormFile("sketch")
		if err != nil {
			t.Fatal(err)
		}
		var picture bytes.Buffer
		_, _ = picture.ReadFrom(file)
		return picture.Bytes()
	}()
	jpeg := []byte{0xFF, 0xD8, 0xFF}
	form := func(files map[string][]byte) *http.Request {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("source", "upload")
		_ = w.WriteField("tags", "GER")
		for field, data := range files {
			part, _ := w.CreateFormFile(field, field+".png")
			_, _ = part.Write(data)
		}
		_ = w.Close()
		request := httptest.NewRequest(http.MethodPost, "/projects/7/gui-render/world-map/world", &body)
		request.Header.Set("Content-Type", w.FormDataContentType())
		return request
	}
	req, problem := parseWorldRequest(httptest.NewRecorder(), form(map[string][]byte{"provinces": png, "heightmap": png, "terrain": png, "definition": []byte("0;0;0;0;land;false;unknown;0\n")}))
	if problem != nil || req.source != "upload" || req.upload.Provinces == nil || req.upload.Rivers != nil || len(req.upload.Definitions) == 0 {
		t.Errorf("upload: %q %+v", problem, req.upload)
	}
	if _, problem := parseWorldRequest(httptest.NewRecorder(), form(map[string][]byte{"provinces": png, "heightmap": png})); !strings.Contains(problemDetail(problem), "地表图") {
		t.Errorf("no terrain map: %q", problem)
	}
	if _, problem := parseWorldRequest(httptest.NewRecorder(), form(map[string][]byte{"provinces": jpeg, "heightmap": png, "terrain": png})); !strings.Contains(problemDetail(problem), "省份图") {
		t.Errorf("an unreadable province map: %q", problem)
	}
}

func TestParseWorldOwners(t *testing.T) {
	owners := func(paint func(x, y int) color.NRGBA) []byte {
		img := image.NewNRGBA(image.Rect(0, 0, 4, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 4; x++ {
				img.SetNRGBA(x, y, paint(x, y))
			}
		}
		var out bytes.Buffer
		if err := png.Encode(&out, img); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	form := func(tags string, picture []byte) *http.Request {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("source", "game")
		_ = w.WriteField("tags", tags)
		if picture != nil {
			part, _ := w.CreateFormFile("owners", "owners.png")
			_, _ = part.Write(picture)
		}
		_ = w.Close()
		request := httptest.NewRequest(http.MethodPost, "/projects/7/gui-render/world-map/world", &body)
		request.Header.Set("Content-Type", w.FormDataContentType())
		return request
	}
	painted := owners(func(x, y int) color.NRGBA {
		switch {
		case y == 1 && x == 3:
			return color.NRGBA{2, 0, 0, 0}
		case x < 2:
			return color.NRGBA{1, 0, 0, 255}
		case x == 2:
			return color.NRGBA{2, 0, 0, 255}
		default:
			return color.NRGBA{0, 0, 0, 255}
		}
	})
	req, problem := parseWorldRequest(httptest.NewRecorder(), form("GER,ENG", painted))
	if problem != nil {
		t.Fatal(problem)
	}
	if req.ownersW != 4 || req.ownersH != 2 || string(req.owners) != string([]uint8{1, 1, 2, 0, 1, 1, 2, 0}) {
		t.Errorf("owners %dx%d %v", req.ownersW, req.ownersH, req.owners)
	}
	if req, problem := parseWorldRequest(httptest.NewRecorder(), form("GER", nil)); problem != nil || req.owners != nil {
		t.Errorf("no owners picture: %q %v", problem, req.owners)
	}
	if _, problem := parseWorldRequest(httptest.NewRecorder(), form("GER", painted)); !strings.Contains(problemDetail(problem), "第 2 个国家") {
		t.Errorf("a number past the tags: %q", problem)
	}
	if _, problem := parseWorldRequest(httptest.NewRecorder(), form("GER", []byte{0xFF, 0xD8, 0xFF})); !strings.Contains(problemDetail(problem), "PNG") {
		t.Errorf("not a PNG: %q", problem)
	}
}
