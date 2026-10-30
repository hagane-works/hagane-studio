package worldmap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
)

func worldTestPicture(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{30, 90, 200, 255}
			if x >= w/8 && x < w*7/8 && y >= h/10 && y < h*9/10 {
				c = color.RGBA{60, 170, 60, 255}
				if y < h/2 {
					c = color.RGBA{210, 50, 50, 255}
				}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func jsonBody(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func imageIDOf(t *testing.T, out map[string]any) string {
	t.Helper()
	meta, _ := out["image"].(map[string]any)
	id, _ := meta["id"].(string)
	if id == "" {
		t.Fatalf("no image id in %v", out)
	}
	return id
}

func (e *testEnv) imageCall(name string, body any) (int, map[string]any) {
	e.t.Helper()
	rec := e.postJSON("/"+name, jsonBody(body))
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func worldImageShown(t *testing.T, env *testEnv, link any, contentType string) {
	t.Helper()
	url, _ := link.(string)
	id, ok := strings.CutPrefix(url, fmt.Sprintf("/api/go/projects/%d/gui-render/world-map/image?id=", env.p.ID()))
	if !ok || !worldImageIDPattern.MatchString(id) {
		t.Fatalf("preview_url %v", link)
	}
	rec := env.get("/image?id=" + id)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != contentType || rec.Body.Len() == 0 {
		t.Fatalf("preview %s: %d %s", id, rec.Code, rec.Header().Get("Content-Type"))
	}
}

func TestWorldImageStepsMakeATerrainSketchAndCountries(t *testing.T) {
	env := newTestEnv(t)
	data := worldTestPicture(80, 160)
	code, out := env.imageCall("image-import", map[string]any{"data_base64": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)})
	if code != http.StatusCreated || !strings.HasPrefix(out["preview"].(string), "data:image/jpeg;base64,") {
		t.Fatalf("import %d %v", code, out)
	}
	picture := imageIDOf(t, out)

	code, out = env.imageCall("image-inspect", map[string]any{"image_id": picture, "marks": []map[string]any{{"x": 40, "y": 40, "label": "AAA"}}})
	if code != http.StatusOK {
		t.Fatalf("inspect %d %v", code, out)
	}
	if colours := out["colours"].([]any); len(colours) != 3 || colours[0].(map[string]any)["nearest_terrain"] == "" {
		t.Fatalf("colours %v", colours)
	}

	code, out = env.imageCall("image-terrain", map[string]any{
		"image_id": picture, "unmatched": "plains",
		"rules": []map[string]any{{"at": map[string]int{"x": 2, "y": 2}, "terrain": "ocean"}},
	})
	if code != http.StatusCreated {
		t.Fatalf("terrain %d %v", code, out)
	}
	terrain := imageIDOf(t, out)
	worldImageShown(t, env, out["preview_url"], "image/jpeg")
	if shares := out["shares"].(map[string]any); shares["plains"].(float64) < 0.5 || shares["ocean"].(float64) < 0.2 {
		t.Fatalf("shares %v", shares)
	}

	code, out = env.imageCall("image-countries", map[string]any{
		"image_id": picture, "terrain_image_id": terrain,
		"seeds": []map[string]any{{"x": 40, "y": 40, "tag": "aaa"}, {"x": 40, "y": 120, "tag": "BBB"}},
	})
	if code != http.StatusCreated {
		t.Fatalf("countries %d %v", code, out)
	}
	ownersID := imageIDOf(t, out)
	worldImageShown(t, env, out["preview_url"], "image/jpeg")
	if tags := out["image"].(map[string]any)["tags"].([]any); len(tags) != 2 || tags[0] != "AAA" || tags[1] != "BBB" {
		t.Fatalf("tags %v", tags)
	}
	if warnings := fmt.Sprint(out["warnings"]); strings.Contains(warnings, "原版") {
		t.Fatalf("invented tags warned as base-game ones: %s", warnings)
	}

	code, out = env.imageCall("image-countries", map[string]any{
		"image_id": picture, "terrain_image_id": terrain,
		"seeds": []map[string]any{{"x": 40, "y": 40, "tag": "NOR"}, {"x": 40, "y": 120, "tag": "BBB"}},
	})
	if warnings := fmt.Sprint(out["warnings"]); code != http.StatusCreated || !strings.Contains(warnings, "NOR 是原版游戏的国家代码") || strings.Contains(warnings, "BBB 是") {
		t.Fatalf("base-game tag %d %s", code, warnings)
	}

	decode := func(id string) image.Image {
		t.Helper()
		rec := env.get("/image?id=" + id)
		img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		return img
	}
	owners, sketch := decode(ownersID), decode(terrain)
	if owners.Bounds() != sketch.Bounds() {
		t.Fatalf("owners %v sketch %v", owners.Bounds(), sketch.Bounds())
	}
	red := func(x, y int) uint8 { r, _, _, _ := owners.At(x, y).RGBA(); return uint8(r >> 8) }
	cx, h := owners.Bounds().Dx()/2, owners.Bounds().Dy()
	if red(cx, h*3/10) != 1 || red(cx, h*7/10) != 2 || red(5, 5) != 0 {
		t.Fatalf("owners: north %d south %d sea %d", red(cx, h*3/10), red(cx, h*7/10), red(5, 5))
	}
	if r, g, b, _ := sketch.At(cx, h/2).RGBA(); [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)} != mapgen.Legend[mapgen.Plains].Color {
		t.Fatalf("sketch centre %d %d %d", r>>8, g>>8, b>>8)
	}

	if len(env.rows(rawFilesTable)) != 0 {
		t.Fatal("a world image became a raw file")
	}
}

func TestWorldImageChecksItsPictures(t *testing.T) {
	env := newTestEnv(t)
	if code, out := env.imageCall("image-inspect", map[string]any{"image_id": "wimg_0123456789abcdef"}); code != http.StatusNotFound {
		t.Fatalf("unknown picture %d %v", code, out)
	}
	_, out := env.imageCall("image-import", map[string]any{"data_base64": base64.StdEncoding.EncodeToString(worldTestPicture(80, 160))})
	picture := imageIDOf(t, out)
	_, out = env.imageCall("image-terrain", map[string]any{"image_id": picture})
	terrain := imageIDOf(t, out)
	_, out = env.imageCall("image-import", map[string]any{"data_base64": base64.StdEncoding.EncodeToString(worldTestPicture(90, 160))})
	other := imageIDOf(t, out)
	if code, out := env.imageCall("image-countries", map[string]any{
		"image_id": other, "terrain_image_id": terrain, "seeds": []map[string]any{{"x": 40, "y": 40, "tag": "AAA"}},
	}); code != http.StatusUnprocessableEntity {
		t.Fatalf("mismatched picture %d %v", code, out)
	}
	if code, _ := env.imageCall("image-countries", map[string]any{
		"image_id": picture, "terrain_image_id": terrain, "seeds": []map[string]any{{"x": 40, "y": 40, "tag": "germany"}},
	}); code != http.StatusUnprocessableEntity {
		t.Fatalf("bad tag %d", code)
	}
	if code, out := env.imageCall("image-world-preview", map[string]any{"terrain_image_id": terrain}); code != http.StatusUnprocessableEntity || !strings.Contains(fmt.Sprint(out["detail"]), "owners_image_id") {
		t.Fatalf("world without countries %d %v", code, out)
	}
	if code, _ := env.imageCall("image-world-preview", map[string]any{"terrain_image_id": terrain, "owners_image_id": picture}); code != http.StatusUnprocessableEntity {
		t.Fatalf("picture as owners %d", code)
	}

	code, out := env.imageCall("image-import", map[string]any{"image_id": picture, "crop": map[string]int{"x": 0, "y": 0, "width": 40, "height": 40}})
	if code != http.StatusCreated || out["image"].(map[string]any)["width"] != 40.0 {
		t.Fatalf("crop %d %v", code, out)
	}

	if code, out := env.imageCall("image-import", map[string]any{}); code != http.StatusUnprocessableEntity || !strings.Contains(fmt.Sprint(out["detail"]), "data_base64") {
		t.Fatalf("empty import %d %v", code, out)
	}
	if code, _ := env.imageCall("image-redraw", map[string]any{"image_id": picture, "prompt": "redraw", "request_id": "world-redraw-1"}); code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		t.Fatalf("redraw route still answers %d", code)
	}
}

func TestWorldImageWorldPreview(t *testing.T) {
	env := newTestEnv(t)
	_, out := env.imageCall("image-import", map[string]any{"data_base64": base64.StdEncoding.EncodeToString(worldTestPicture(64, 32))})
	picture := imageIDOf(t, out)
	_, out = env.imageCall("image-terrain", map[string]any{
		"image_id": picture, "unmatched": "plains", "rules": []map[string]any{{"at": map[string]int{"x": 1, "y": 1}, "terrain": "ocean"}},
	})
	terrain := imageIDOf(t, out)
	code, out := env.imageCall("image-countries", map[string]any{
		"image_id": picture, "terrain_image_id": terrain,
		"seeds": []map[string]any{{"x": 32, "y": 10, "tag": "TST"}, {"x": 32, "y": 24, "tag": "OTH"}},
	})
	if code != http.StatusCreated {
		t.Fatalf("countries %d %v", code, out)
	}
	owners := imageIDOf(t, out)
	code, out = env.imageCall("image-world-preview", map[string]any{"terrain_image_id": terrain, "owners_image_id": owners, "land": 400, "per_state": 1})
	if code != http.StatusOK {
		t.Fatalf("world preview %d %v", code, out)
	}
	urls, _ := out["preview_urls"].(map[string]any)
	if len(urls) == 0 || out["state_layout"] == nil {
		t.Fatalf("preview %v", out)
	}
	for name, url := range urls {
		kind := "image/png"
		if name == "political_close" {
			kind = "image/jpeg"
		}
		worldImageShown(t, env, url, kind)
	}
}

func TestWorldImageKeepPreview(t *testing.T) {
	env := newTestEnv(t)
	objects := worldImageStore(env.p)
	url := worldImageKeepPreview(objects, env.p.ID(), "wimg_0000000000000001", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(worldTestPicture(40, 40)))
	worldImageShown(t, env, url, "image/png")
	id := url[strings.LastIndex(url, "=")+1:]
	if code, _ := env.imageCall("image-countries", map[string]any{
		"image_id": id, "terrain_image_id": id, "seeds": []map[string]any{{"x": 1, "y": 1, "tag": "AAA"}},
	}); code != http.StatusUnprocessableEntity {
		t.Fatalf("a preview taken as a terrain sketch: %d", code)
	}
	if got := worldImageKeepPreview(objects, env.p.ID(), "", "data:text/plain;base64,aGk="); got != "" {
		t.Fatalf("not a picture kept: %s", got)
	}
}

func worldImageCall(t *testing.T, env *testEnv, name, body string) (int, map[string]any) {
	t.Helper()
	rec := env.postJSON("/"+name, body)
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}
