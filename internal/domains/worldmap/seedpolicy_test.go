package worldmap

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapsketch"
)

func TestPoliticalTerrainRequiresExplicitOceanAndNeutralLand(t *testing.T) {
	for _, tc := range []struct {
		name, unmatched string
		rules           []worldImageRuleIn
		areas           []worldImageAreaIn
	}{
		{"missing ocean", "", nil, nil},
		{"lake ocean misread", "plains", []worldImageRuleIn{{Colour: "#a1b9cd", Terrain: "lakes"}}, nil},
		{"political country becomes forest", "plains", []worldImageRuleIn{{Terrain: "ocean"}, {Terrain: "forest"}}, nil},
		{"nearest reinterprets country colours", "nearest", []worldImageRuleIn{{Terrain: "ocean"}}, nil},
		{"area bypass", "plains", []worldImageRuleIn{{Terrain: "ocean"}}, []worldImageAreaIn{{Terrain: "mountain"}}},
		{"negative tolerance cannot restore legacy 40", "plains", []worldImageRuleIn{{Terrain: "ocean", Tolerance: -1}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := normalizeWorldTerrainSource("political", tc.unmatched, tc.rules, tc.areas); err == nil {
				t.Fatal("unsupported political terrain request was silently accepted")
			}
		})
	}
	_, unmatched, err := normalizeWorldTerrainSource("political", "", []worldImageRuleIn{{Colour: "#a1b9cd", Terrain: "ocean"}, {Terrain: "ignore"}}, nil)
	if err != nil || unmatched != "plains" {
		t.Fatalf("political source did not enforce plains: %q %v", unmatched, err)
	}
	_, unmatched, err = normalizeWorldTerrainSource("", "nearest", []worldImageRuleIn{{Terrain: "forest"}}, nil)
	if err != nil || unmatched != "nearest" {
		t.Fatal("legacy terrain interpretation changed")
	}
}

func TestPoliticalOceanDefaultDoesNotSwallowNearbyCountryColour(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 80, 160))
	for y := 0; y < 160; y++ {
		for x := 0; x < 80; x++ {
			c := color.NRGBA{162, 186, 206, 255}
			if x >= 20 && x < 60 && y >= 20 && y < 140 {
				c = color.NRGBA{146, 172, 194, 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}

	distance := mapsketch.Distance(mapsketch.RGB{162, 186, 206}, mapsketch.RGB{146, 172, 194})
	if distance <= 12 || distance >= 40 {
		t.Fatal("fixture no longer exercises near-ocean colours")
	}
	for _, tc := range []struct {
		name, kind string
		explicit   float64
		effective  float64
		want       int
	}{
		{"political conservative default", "political", 0, 12, http.StatusCreated},
		{"political explicit value retained and verified", "political", 40, 40, http.StatusUnprocessableEntity},
		{"legacy default preserved", "", 0, 40, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			_, out := worldImageCall(t, env, "image-import", jsonBody(map[string]any{"data_base64": base64.StdEncoding.EncodeToString(data.Bytes())}))
			imageID := imageIDOf(t, out)
			rule := map[string]any{"colour": "#a2bace", "terrain": "ocean"}
			if tc.explicit != 0 {
				rule["tolerance"] = tc.explicit
			}
			code, out := worldImageCall(t, env, "image-terrain", jsonBody(map[string]any{"image_id": imageID, "source_kind": tc.kind, "unmatched": "plains", "dark": -1, "smooth": -1, "rules": []any{rule}}))
			if code != http.StatusCreated {
				t.Fatalf("terrain %d %v", code, out)
			}
			if got := out["rules"].([]any)[0].(map[string]any)["tolerance"]; got != tc.effective {
				t.Fatalf("effective ocean tolerance %v want %v", got, tc.effective)
			}
			terrainID := imageIDOf(t, out)
			code, out = worldImageCall(t, env, "image-countries", jsonBody(map[string]any{"image_id": imageID, "terrain_image_id": terrainID, "seeds": []any{map[string]any{"tag": "GRY", "colour": "#92acc2", "colour_tolerance": 3}}}))
			if code != tc.want {
				t.Fatalf("near-blue country: code=%d want=%d result=%v", code, tc.want, out)
			}
			if tc.want == http.StatusUnprocessableEntity && !strings.Contains(out["detail"].(string), "terrain marks them all as water") {
				t.Fatalf("water loss not actionable: %v", out)
			}
		})
	}
}

func TestPoliticalTerrainAndSourceColourCountrySeedsThroughHTTP(t *testing.T) {
	env := newTestEnv(t)
	_, out := worldImageCall(t, env, "image-import", jsonBody(map[string]any{"data_base64": base64.StdEncoding.EncodeToString(worldTestPicture(80, 160))}))
	picture := imageIDOf(t, out)
	code, out := worldImageCall(t, env, "image-terrain", jsonBody(map[string]any{
		"image_id": picture, "source_kind": "political", "rules": []any{map[string]any{"colour": "#1e5ac8", "terrain": "ocean", "tolerance": 10}},
	}))
	if code != http.StatusCreated || out["source_kind"] != "political" || out["unmatched_terrain"] != "plains" {
		t.Fatalf("political terrain %d %v", code, out)
	}
	terrain := imageIDOf(t, out)
	code, out = worldImageCall(t, env, "image-countries", jsonBody(map[string]any{
		"image_id": picture, "terrain_image_id": terrain,
		"seeds": []any{map[string]any{"tag": "NTH", "colour": "#d23232"}, map[string]any{"tag": "STH", "colour": "#3caa3c"}},
	}))
	if code != http.StatusCreated {
		t.Fatalf("colour-only seeds %d %v", code, out)
	}
	matches := out["seed_matches"].([]any)
	if len(matches) != 2 {
		t.Fatal("selection evidence omitted")
	}
	for _, value := range matches {
		match := value.(map[string]any)
		if match["requested_colour"] != match["source_colour"] || match["match_share"].(float64) <= 0 || match["actual_seed"] == nil {
			t.Fatalf("source pixel evidence wrong: %v", match)
		}
	}
}

func TestCountrySeedLegacyZeroAndMissingPointContract(t *testing.T) {
	p := mapsketch.FromImage(image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	zero := 0
	seeds, _, err := resolveWorldCountrySeeds(p, []worldImageSeedIn{{Tag: "AAA", X: &zero, Y: &zero}}, nil)
	if err != nil || seeds[0].At != (mapsketch.Point{}) {
		t.Fatal("legacy explicit 0,0 coordinate changed")
	}
	if _, _, err := resolveWorldCountrySeeds(p, []worldImageSeedIn{{Tag: "AAA"}}, nil); err == nil {
		t.Fatal("missing colour/point silently became 0,0")
	}
}

func TestPoliticalSourcePrivateFixtureColourSeeds(t *testing.T) {
	path := os.Getenv("HOI4_WORLD_SOURCE_FIXTURE")
	if path == "" {
		t.Skip("set HOI4_WORLD_SOURCE_FIXTURE to the existing private fantasy_world.png")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	p := mapsketch.FromImage(img)
	colours := []string{"#d27846", "#5476a8", "#467882", "#b24840", "#a0966e", "#608c5c", "#c4a048", "#c8b478", "#6e8246", "#8e5a96", "#b46e8c", "#785046", "#5a64a0", "#82a0b4"}
	for _, raw := range colours {
		colour, _ := mapsketch.ParseHex(raw)
		result, err := mapsketch.FindColourSeed(p, mapsketch.ColourSeedOptions{Colour: colour})
		if err != nil || !strings.EqualFold(result.SourceColour, raw) {
			t.Fatalf("private source %s: seed selection not exact source colour: %+v %v", raw, result, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	env := newTestEnv(t)
	_, out := worldImageCall(t, env, "image-import", jsonBody(map[string]any{"data_base64": base64.StdEncoding.EncodeToString(data)}))
	imageID := imageIDOf(t, out)
	code, out := worldImageCall(t, env, "image-terrain", jsonBody(map[string]any{
		"image_id": imageID, "source_kind": "political", "rules": []any{map[string]any{"colour": "#a1b9cd", "terrain": "ocean"}},
	}))
	if code != http.StatusCreated {
		t.Fatalf("private political terrain status %d", code)
	}
	terrainID := imageIDOf(t, out)
	tags := []string{"SLN", "OSK", "DRA", "VAL", "ESH", "TYR", "MER", "LUM", "VOR", "KAE", "QES", "NVA", "ARD", "ZEP"}
	seeds := []any{}
	for i, raw := range colours {
		seeds = append(seeds, map[string]any{"tag": tags[i], "colour": raw})
	}
	code, out = worldImageCall(t, env, "image-countries", jsonBody(map[string]any{"image_id": imageID, "terrain_image_id": terrainID, "seeds": seeds}))
	if code != http.StatusCreated {
		t.Fatalf("private source split %d: %v", code, out)
	}
	for _, value := range out["countries"].([]any) {
		row := value.(map[string]any)
		if row["share"].(float64) <= 0 {
			t.Fatalf("private source lost a required country: %v", row)
		}
	}
	if len(out["seed_matches"].([]any)) != 14 || len(out["countries"].([]any)) != 14 {
		t.Fatal("private source country/seed coverage incomplete")
	}
	t.Logf("all %d real political source colours resolve to matching actual source pixels without manual coordinates", len(colours))
}
