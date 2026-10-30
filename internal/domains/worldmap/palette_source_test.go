package worldmap

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"testing"
)

func TestPoliticalMapInspectionSamplesMakeFirstCountrySplitSucceed(t *testing.T) {
	path := os.Getenv("HOI4_WORLD_SOURCE_FIXTURE")
	if path == "" {
		t.Skip("set HOI4_WORLD_SOURCE_FIXTURE to the existing private fantasy_world.png")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	env := newTestEnv(t)
	code, imported := worldImageCall(t, env, "image-import", jsonBody(map[string]any{"data_base64": base64.StdEncoding.EncodeToString(raw)}))
	if code != http.StatusCreated {
		t.Fatalf("fixture import failed: %d", code)
	}
	imageID := imageIDOf(t, imported)
	code, inspection := worldImageCall(t, env, "image-inspect", jsonBody(map[string]any{"image_id": imageID, "max_colours": 40}))
	if code != http.StatusOK {
		t.Fatalf("first inspection failed: %d", code)
	}
	samples := map[string]map[string]any{}
	for _, value := range inspection["colours"].([]any) {
		swatch := value.(map[string]any)
		if source, ok := swatch["sample_source_colour"].(string); ok && samples[source] == nil {
			samples[source] = swatch
		}
	}
	ocean := samples["#a2bace"]
	if ocean == nil {
		t.Fatal("source ocean sample missing")
	}
	code, terrain := worldImageCall(t, env, "image-terrain", jsonBody(map[string]any{
		"image_id": imageID, "source_kind": "political", "unmatched": "plains", "dark": -1, "smooth": -1,
		"rules": []any{map[string]any{"colour": ocean["sample_source_colour"], "at": ocean["sample"], "terrain": "ocean"}},
	}))
	if code != http.StatusCreated {
		t.Fatalf("first terrain sketch failed: %d", code)
	}
	countryColours := []string{"#d27846", "#5476a8", "#467882", "#b24840", "#a0966e", "#608c5c", "#6e8246", "#8e5a96", "#c4a048", "#c8b478", "#b46e8c", "#785046", "#5a64a0", "#82a0b4"}
	seeds := make([]any, 0, len(countryColours))
	for index, colour := range countryColours {
		swatch := samples[colour]
		if swatch == nil {
			t.Fatalf("country %d has no verified source sample", index+1)
		}
		seeds = append(seeds, map[string]any{"tag": fmt.Sprintf("C%02d", index+1), "colour": swatch["sample_source_colour"], "at": swatch["sample"]})
	}
	code, split := worldImageCall(t, env, "image-countries", jsonBody(map[string]any{
		"image_id": imageID, "terrain_image_id": imageIDOf(t, terrain), "seeds": seeds, "dark": 45,
	}))
	if code != http.StatusCreated {
		t.Fatalf("first country split failed: %d detail=%v", code, split["detail"])
	}
	matches := split["seed_matches"].([]any)
	if len(matches) != len(countryColours) {
		t.Fatalf("country sample count = %d", len(matches))
	}
	for index, value := range matches {
		match := value.(map[string]any)
		if match["requested_colour"] != countryColours[index] || match["source_colour"] != countryColours[index] {
			t.Fatalf("country %d source colour changed: %v", index+1, match)
		}
		selection := match["selection"].(map[string]any)
		if selection["selected_region_pixels"].(float64) < 1000 {
			t.Fatalf("country %d selected lettering instead of its source region", index+1)
		}
	}
}
