package gamefiles

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io/fs"
	"testing"
	"testing/fstest"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testGame(t *testing.T) fstest.MapFS {
	return fstest.MapFS{
		"launcher-settings.json":                   {Data: []byte(`{"rawVersion":"1.19.3.0"}`)},
		"common/ideas/base.txt":                    {Data: []byte("base")},
		"common/ideas/shared.txt":                  {Data: []byte("from base")},
		"interface/base.gfx":                       {Data: []byte(`spriteTypes = { spriteType = { name = "GFX_base" texturefile = "gfx\\interface\\base.dds" } spriteType = { name = "GFX_over" texturefile = "gfx/interface/old.dds" } }`)},
		"gfx/interface/base.png":                   {Data: pngBytes(t, 4, 2)},
		"dlc/dlc001_a/common/ideas/a.txt":          {Data: []byte("dlc a")},
		"dlc/dlc001_a/common/ideas/shared.txt":     {Data: []byte("from dlc")},
		"integrated_dlc/b/common/ideas/shared.txt": {Data: []byte("from integrated")},
		"integrated_dlc/b/interface/b.gfx":         {Data: []byte(`spriteTypes = { spriteType = { name = "GFX_over" texturefile = "gfx/interface/new.tga" } }`)},
		"integrated_dlc/b/gfx/interface/new.png":   {Data: pngBytes(t, 3, 3)},
	}
}

func TestRootsFollowTheGamesLoadOrder(t *testing.T) {
	var names []string
	for _, root := range Roots(testGame(t)) {
		names = append(names, root.Name)
	}
	want := []string{"", "dlc/dlc001_a", "integrated_dlc/b"}
	if len(names) != len(want) {
		t.Fatalf("roots = %q, want %q", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("roots = %q, want %q", names, want)
		}
	}
}

func TestReadFileTakesTheLastLayer(t *testing.T) {
	g := For(testGame(t))
	got, err := g.ReadFile(`common\ideas\shared.txt`)
	if err != nil || string(got) != "from integrated" {
		t.Fatalf("shared.txt = %q, %v", got, err)
	}
	if got, _ := g.ReadFile("common/ideas/a.txt"); string(got) != "dlc a" {
		t.Fatalf("a.txt = %q", got)
	}
	if _, err := g.ReadFile("common/ideas/none.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing file: %v", err)
	}
}

func TestFilesListsEveryLayerOnce(t *testing.T) {
	got := For(testGame(t)).Files("common/ideas", false, ".TXT")
	want := []string{"common/ideas/a.txt", "common/ideas/base.txt", "common/ideas/shared.txt"}
	if len(got) != len(want) {
		t.Fatalf("files = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files = %q, want %q", got, want)
		}
	}
}

func TestSpritesAndPictures(t *testing.T) {
	g := For(testGame(t))
	sprites := g.Sprites()
	if sprites["GFX_base"] != "gfx/interface/base.dds" {
		t.Fatalf("GFX_base = %q", sprites["GFX_base"])
	}
	if sprites["GFX_over"] != "gfx/interface/new.tga" {
		t.Fatalf("a later layer's sprite should win: GFX_over = %q", sprites["GFX_over"])
	}

	for sprite, width := range map[string]int{"GFX_base": 4, "GFX_over": 3} {
		data, err := g.SpritePNG(sprite)
		if err != nil {
			t.Fatalf("%s: %v", sprite, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil || img.Bounds().Dx() != width {
			t.Fatalf("%s: width %v, %v", sprite, img.Bounds(), err)
		}
	}
	if _, err := g.SpritePNG("GFX_none"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("unknown sprite: %v", err)
	}
}

func TestForSharesTheGameOfOneInstallation(t *testing.T) {
	game := testGame(t)
	if For(game) != For(game) {
		t.Fatal("the same installation should share one Game")
	}
}

func TestPNGCacheStaysWithinItsBounds(t *testing.T) {
	c := newPNGCache(2, 10)
	c.put("a", []byte("1234"))
	c.put("b", []byte("1234"))
	c.get("a")
	c.put("c", []byte("1234"))
	if _, ok := c.get("b"); ok {
		t.Fatal("b should have been dropped")
	}
	if _, ok := c.get("a"); !ok {
		t.Fatal("a was used recently and should stay")
	}
	c.put("big", make([]byte, 11))
	if _, ok := c.get("big"); ok {
		t.Fatal("an entry larger than the cache is not kept")
	}
}
