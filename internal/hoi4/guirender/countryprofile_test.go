package guirender

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRealGameCountryProfile(t *testing.T) {
	below := realTopbarIndex(t, false)

	files := map[string][]byte{}
	dir := filepath.Join(os.Getenv("HOI4_GAME_ROOT"), "common", "countries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if body, err := os.ReadFile(filepath.Join(dir, entry.Name())); err == nil {
			files["common/countries/"+entry.Name()] = body
		}
	}
	idx, err := NewIndexWithFallback(files, below)
	if err != nil {
		t.Fatal(err)
	}
	profile := idx.CountryProfilePayload("GER", "1936.1.1.12")
	if profile["name"] != "Germany" || profile["adjective"] != "German" || profile["shown"] != "German Reich" {
		t.Errorf("names = %v / %v / %v", profile["name"], profile["adjective"], profile["shown"])
	}
	names := profile["names"].(map[string]any)
	if names["fascism"] != "German Reich" || names["democratic"] == nil {
		t.Errorf("ideology names = %v", names)
	}
	if profile["graphical_culture"] != "western_european_gfx" || profile["ruling"] != "fascism" {
		t.Errorf("culture %v, ruling %v", profile["graphical_culture"], profile["ruling"])
	}
	if capital := profile["capital"].(map[string]any); capital["id"] != 64 || capital["name"] != "Brandenburg" {
		t.Errorf("capital = %v", capital)
	}
	if len(profile["groups"].([]map[string]any)) != 4 {
		t.Errorf("groups %v", profile["groups"])
	}
}

func TestUploadedFlagsFollowTheEngineOrder(t *testing.T) {
	idx, err := NewIndexWithFallback(map[string][]byte{
		"gfx/flags/GER.png":         pngOf(t, 82, 52),
		"gfx/flags/GER_fascism.png": pngOf(t, 82, 52),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if flag := idx.flagPayload("GER", "fascism", ""); flag == nil || flag["project"] != nil {
		t.Fatalf("no uploads: %v", flag)
	}
	idx.SetProjectFlags(map[string]string{"GER": "/up/plain"})
	if flag := idx.flagPayload("GER", "fascism", ""); flag["url"] != "/up/plain" || flag["project"] != true {
		t.Errorf("the exported plain upload did not cover the game's fascist flag: %v", flag)
	}
	if flag := idx.flagPayload("GER", "democratic", ""); flag["url"] != "/up/plain" || flag["width"] != 82 {
		t.Errorf("democratic Germany flies %v", flag)
	}
	idx.SetProjectFlags(map[string]string{"GER": "/up/plain", "GER_fascism": "/up/fascist"})
	if flag := idx.flagPayload("GER", "fascism", ""); flag["url"] != "/up/fascist" {
		t.Errorf("fascist Germany flies %v", flag)
	}
	idx.SetProjectFlags(map[string]string{"ZZZ": "/up/z"})
	if flag := idx.flagPayload("ZZZ", "", "small"); flag["url"] != "/up/z" || flag["width"] != 10 || flag["height"] != 7 {
		t.Errorf("a new country's small flag = %v", flag)
	}

	below, err := NewIndex(nil)
	if err != nil {
		t.Fatal(err)
	}
	below.SetProjectFlags(map[string]string{"GER_fascism": "/lower/fascist"})
	top, err := NewIndexWithFallback(nil, below)
	if err != nil {
		t.Fatal(err)
	}
	top.SetProjectFlags(map[string]string{"GER": "/top/plain"})
	if flag := top.flagPayload("GER", "fascism", ""); flag["url"] != "/top/plain" {
		t.Errorf("a lower project variant shadowed the top exported base flag: %v", flag)
	}
}
