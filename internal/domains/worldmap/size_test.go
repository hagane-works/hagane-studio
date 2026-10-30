package worldmap

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
)

func TestWorldSizeProblemTakesAModsOwnSize(t *testing.T) {
	game := &mapgen.Base{Width: 5632, Height: 2048}
	mod := &mapgen.Base{Width: 5120, Height: 2560}
	if problem := worldSizeProblem("mod", mod, game); problem != nil {
		t.Fatalf("a MOD's own size refused: %+v", problem)
	}
	problem := worldSizeProblem("upload", mod, game)
	if problem == nil || problem.Code != problemWorldSize || problem.Values["got_width"] != 5120 || problem.Values["width"] != 5632 {
		t.Fatalf("a player's own pictures at another size: %+v", problem)
	}
	if problem := worldSizeProblem("upload", &mapgen.Base{Width: 5632, Height: 2048}, game); problem != nil {
		t.Fatalf("the base game's size refused: %+v", problem)
	}
}

func TestOneClickFixesMatchTheStudio(t *testing.T) {
	ts, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "src", "app", "[locale]", "game-view", "map-studio", "mapProblems.ts"))
	if err != nil {
		t.Skip("frontend not checked out:", err)
	}
	line := regexp.MustCompile(`ONE_CLICK_FIXES = new Set<string>\(\[([^\]]*)\]\)`).FindSubmatch(ts)
	if line == nil {
		t.Fatal("ONE_CLICK_FIXES not found in mapProblems.ts")
	}
	studio := map[string]bool{}
	for _, code := range regexp.MustCompile(`"([a-z_]+)"`).FindAllSubmatch(line[1], -1) {
		studio[string(code[1])] = true
	}
	if !reflect.DeepEqual(studio, worldFixes) {
		t.Fatalf("studio %v, server %v", studio, worldFixes)
	}
}
