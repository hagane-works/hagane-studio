package worldmap

import (
	"archive/zip"
	"bytes"
	"context"
	"image/color"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"haganeworks/internal/export"
)

func TestNewLandIsExported(t *testing.T) {
	env := newTestEnv(t)
	sketch := pngOf(t, 12, 12, color.RGBA{160, 210, 100, 255})
	fields := map[string]string{"x": "40", "y": "8", "width": "14", "height": "14", "seed": "1", "land": "60", "name": "Atlantis", "owner": "TST"}
	want(t, env.postForm("/land", map[string][]byte{"sketch": sketch}, fields), http.StatusCreated)
	stored, err := env.p.Blobs().Get(str(env.rawFile("map/provinces.bmp")["content_obs_key"]), 64<<20)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "mod.zip")
	if _, err := export.Build(context.Background(), env.p, export.Options{}, out); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(rc)
		rc.Close()

		name := f.Name
		for _, root := range []string{"map/", "history/", "localisation/"} {
			if at := strings.Index(name, "/"+root); at >= 0 {
				name = name[at+1:]
			}
		}
		files[name] = body
	}
	if !bytes.Equal(files["map/provinces.bmp"], stored) {
		t.Errorf("map/provinces.bmp not exported as stored (%d bytes, stored %d)", len(files["map/provinces.bmp"]), len(stored))
	}
	if !strings.Contains(string(files["map/definition.csv"]), "5;") {
		t.Errorf("definition.csv without the new provinces:\n%s", files["map/definition.csv"])
	}
	if !strings.Contains(string(files["map/strategicregions/3-Atlantis.txt"]), "id=3") && !strings.Contains(string(files["map/strategicregions/3-Atlantis.txt"]), "id = 3") {
		t.Errorf("new region file: %q", files["map/strategicregions/3-Atlantis.txt"])
	}
	state, region := false, false
	for name, body := range files {
		if strings.HasPrefix(name, "history/states/") && strings.Contains(string(body), "id = 3") {
			state = true
		}
		if strings.HasPrefix(name, "localisation/") && strings.Contains(string(body), "STRATEGICREGION_3") {
			region = true
		}
	}
	if !state || !region {
		names := make([]string, 0, len(files))
		for name := range files {
			names = append(names, name)
		}
		t.Fatalf("state file %v, region name %v in:\n%s", state, region, strings.Join(names, "\n"))
	}
}
