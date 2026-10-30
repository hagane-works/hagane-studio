package focus

import (
	"bytes"
	"image/png"
	"net/http"
	"os"
	"testing"

	"haganeworks/internal/gamefiles"
)

func TestRealGameIcons(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("HOI4_GAME_DIR not set")
	}
	game := os.DirFS(dir)
	h := newHarness(t, game)
	library := h.expect(h.do("GET", "/api/go/projects/1/focus-icons/library?source=vanilla&limit=5000", nil), http.StatusOK)
	total := int(library["total"].(float64))
	if total < 2000 {
		t.Fatalf("only %d base-game focus icons", total)
	}
	t.Logf("%d base-game focus icons", total)
	for _, name := range []string{"GFX_goal_unknown", "GFX_goal_generic_construct_civ_factory", "GFX_focus_generic_industry_1"} {
		rec := h.do("GET", "/api/go/vanilla/focus-icon/"+name, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
		img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s: %dx%d", name, img.Bounds().Dx(), img.Bounds().Dy())
	}
	h.expect(h.do("GET", "/api/go/lint/tree-validation/1", nil), http.StatusOK)
	data := loadGameLintData(gamefiles.For(game))
	if !data.categories["construction_tech"] || !data.sprites["GFX_goal_unknown"] {
		t.Fatalf("game lint data: %d categories, %d sprites", len(data.categories), len(data.sprites))
	}
}
