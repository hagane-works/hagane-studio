package tools

import (
	"bytes"
	"image/png"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"haganeworks/internal/domains"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/project"
)

func tgaBytes(w, h int) []byte {
	header := []byte{0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(w), byte(w >> 8), byte(h), byte(h >> 8), 32, 8}
	return append(header, bytes.Repeat([]byte{40, 80, 120, 255}, w*h)...)
}

func pictureSize(t *testing.T, mux *http.ServeMux, path string) (int, int) {
	t.Helper()
	rec := call(t, mux, "GET", path, "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("%s: status %d (%s): %s", path, rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	config, err := png.DecodeConfig(rec.Body)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return config.Width, config.Height
}

func TestBasemodPicturesFromTheChosenFolder(t *testing.T) {
	modDir := t.TempDir()
	for name, data := range map[string][]byte{
		"descriptor.mod": []byte(`name="The New Order: Last Days of Europe"`),
		"interface/tno.gfx": []byte(`spriteTypes = {
			spriteType = { name = "GFX_focus_tno_test" texturefile = "gfx/interface/goals/focus_tno_test.dds" }
			spriteType = { name = "GFX_idea_from_game" texturefile = "gfx/interface/ideas/base_only.dds" }
			spriteType = { name = "GFX_idea_nowhere" texturefile = "gfx/interface/ideas/nowhere.dds" }
			spriteType = { name = "GFX_portrait_tno_test" texturefile = "gfx/leaders/TNO/Portrait_Test.dds" }
		}`),
		"gfx/interface/goals/focus_tno_test.png": pngBytes(t, 5, 4),
		"gfx/leaders/TNO/Portrait_Test.png":      pngBytes(t, 3, 5),
		"gfx/flags/TNO.tga":                      tgaBytes(4, 2),
		"gfx/flags/medium/TNO_fascism.tga":       tgaBytes(2, 1),
	} {
		file := filepath.Join(modDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	game := fstest.MapFS{"gfx/interface/ideas/base_only.png": {Data: pngBytes(t, 6, 6)}}
	p := newProject(t)

	setBaseMods(t, p, []any{"梦回", "The New Order - Last Days of Europe", "The New Order: Last Days of Europe", "Kaiserreich"}, map[string]string{
		"梦回":                                 t.TempDir(),
		"The New Order: Last Days of Europe": modDir,
		"Kaiserreich":                        filepath.Join(t.TempDir(), "gone"),
	})
	mux := basemodServer(t, p, game)
	check := func(path string, wantW, wantH int) {
		t.Helper()
		if w, h := pictureSize(t, mux, path); w != wantW || h != wantH {
			t.Fatalf("%s: %dx%d, want %dx%d", path, w, h, wantW, wantH)
		}
	}
	base := "/api/go/projects/" + strconv.FormatInt(p.ID(), 10) + "/basemod/"
	dep := base + "the_new_order_last_days_of_europe"

	check(dep+"/gfx/GFX_focus_tno_test", 5, 4)
	check(dep+"/gfx/GFX_idea_from_game", 6, 6)
	expect(t, call(t, mux, "GET", dep+"/gfx/GFX_idea_nowhere", ""), 404)
	body := expect(t, call(t, mux, "GET", dep+"/gfx/GFX_focus_unknown", ""), 404)
	if body["detail"] != "base-mod gfx name not found: GFX_focus_unknown" {
		t.Fatalf("detail = %v", body)
	}

	check(base+"the_new_order__last_days_of_europe/gfx/GFX_focus_tno_test", 5, 4)

	check(dep+"/art/gfx/leaders/TNO/Portrait_Test.dds", 3, 5)
	check(dep+"/art/portrait_test.png", 3, 5)
	expect(t, call(t, mux, "GET", dep+"/art/gfx/leaders/TNO/none.dds", ""), 404)
	expect(t, call(t, mux, "GET", dep+"/art/none.dds", ""), 404)
	expect(t, call(t, mux, "GET", dep+"/art/gfx/interface/ideas/base_only.dds", ""), 404)
	expect(t, call(t, mux, "GET", dep+"/art/gfx%5C..%5C..%5Csecret.dds", ""), 400)
	expect(t, call(t, mux, "GET", dep+"/art/..%2Fdescriptor.mod", ""), 400)

	check(dep+"/flag/TNO", 4, 2)
	check(dep+"/flag/TNO?ideology=fascism&size=medium", 2, 1)
	check(dep+"/flag/TNO?ideology=communism", 4, 2)
	expect(t, call(t, mux, "GET", dep+"/flag/TNO?size=small", ""), 404)
	expect(t, call(t, mux, "GET", dep+"/flag/GER", ""), 404)
	expect(t, call(t, mux, "GET", dep+"/flag/TNO?size=huge", ""), 400)

	body = expect(t, call(t, mux, "GET", base+"kaiserreich/gfx/GFX_focus_tno_test", ""), 404)
	if body["detail"] != "no base-mod folder for dependency 'kaiserreich'" {
		t.Fatalf("detail = %v", body)
	}
	expect(t, call(t, mux, "GET", base+"unknown/flag/TNO", ""), 404)
	expect(t, call(t, mux, "GET", "/api/go/projects/999/basemod/the_new_order_last_days_of_europe/gfx/GFX_focus_tno_test", ""), 404)

	if rec := call(t, mux, "GET", dep+"/gfx/GFX_focus_tno_test", ""); rec.Header().Get("Cache-Control") != "" {
		t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
	setBaseMods(t, p, []any{"The New Order: Last Days of Europe"}, nil)
	expect(t, call(t, mux, "GET", dep+"/gfx/GFX_focus_tno_test", ""), 404)
}

func setBaseMods(t *testing.T, p *project.Project, dependencies []any, folders map[string]string) {
	t.Helper()
	_, err := p.UpdateRecordAndBaseMods(func(record project.Row, baseMods map[string]string) {
		record["dependencies"] = dependencies
		clear(baseMods)
		maps.Copy(baseMods, folders)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func basemodServer(t *testing.T, p *project.Project, game fs.FS) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	registerBasemod(mux, domains.Deps{
		Open: func(r *http.Request) (*project.Project, error) {
			if r.PathValue("id") != strconv.FormatInt(p.ID(), 10) {
				return nil, project.ErrNotFound
			}
			return p, nil
		},
		Game: func() (fs.FS, error) { return game, nil },
	})
	return mux
}

func TestBasemodPicturesFromARealInstall(t *testing.T) {
	gameDir, modDir := os.Getenv("HOI4_GAME_DIR"), os.Getenv("HOI4_BASE_MOD_DIR")
	if gameDir == "" || modDir == "" {
		t.Skip("HOI4_GAME_DIR or HOI4_BASE_MOD_DIR not set")
	}
	files := gamefiles.Mod(os.DirFS(modDir))
	var sprite, texture string
	for _, name := range slices.Sorted(maps.Keys(files.Sprites())) {
		if !strings.HasPrefix(name, "GFX_focus_") {
			continue
		}
		if found, ok := files.Texture(files.Sprites()[name]); ok {
			sprite, texture = name, found
			break
		}
	}
	flags, _ := fs.Glob(os.DirFS(modDir), "gfx/flags/*.tga")
	if sprite == "" || len(flags) == 0 {
		t.Skipf("%s ships no focus icon or no flag", modDir)
	}
	p := newProject(t)
	setBaseMods(t, p, []any{"Base"}, map[string]string{"Base": modDir})
	mux := basemodServer(t, p, os.DirFS(gameDir))
	dep := "/api/go/projects/" + strconv.FormatInt(p.ID(), 10) + "/basemod/base"
	name := strings.TrimSuffix(path.Base(flags[0]), ".tga")
	flag := dep + "/flag/" + name
	if tag, ideology, ok := strings.Cut(name, "_"); ok {
		flag = dep + "/flag/" + tag + "?ideology=" + ideology
	}
	for _, route := range []string{dep + "/gfx/" + sprite, dep + "/art/" + texture, flag} {
		w, h := pictureSize(t, mux, route)
		t.Logf("%s: %s %dx%d", modDir, route, w, h)
	}
}
