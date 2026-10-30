package gamefiles

import (
	"os"
	"testing"
	"time"
)

func TestRealGame(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("HOI4_GAME_DIR not set")
	}
	g := For(os.DirFS(dir))
	start := time.Now()
	sprites := g.Sprites()
	t.Logf("%d layers, %d sprites in %v", len(g.Roots()), len(sprites), time.Since(start))
	for _, sprite := range []string{"GFX_goal_unknown", "GFX_idea_generic_infantry_bonus", "GFX_decision_generic_industry"} {
		start = time.Now()
		png, err := g.SpritePNG(sprite)
		if err != nil {
			t.Fatalf("%s: %v", sprite, err)
		}
		t.Logf("%s: %d bytes PNG in %v", sprite, len(png), time.Since(start))
	}
}
