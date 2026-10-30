package gamefiles

import (
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

type countKey struct{}
type otherKey struct{}

func TestDerivedIsBuiltOncePerGameAndKey(t *testing.T) {
	g := For(testGame(t))
	var builds atomic.Int32
	build := func(g *Game) int {
		builds.Add(1)
		return len(g.Files("common/ideas", false, ".txt"))
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := Derived(g, countKey{}, build); got != 3 {
				t.Errorf("derived = %d, want 3", got)
			}
		}()
	}
	wg.Wait()
	if builds.Load() != 1 {
		t.Fatalf("built %d times, want once", builds.Load())
	}
	if got := Derived(g, otherKey{}, func(*Game) string { return "other" }); got != "other" {
		t.Fatalf("another key = %q", got)
	}

	h := For(fstest.MapFS{"common/ideas/x.txt": {Data: []byte("x")}})
	if got := Derived(h, countKey{}, build); got != 1 || builds.Load() != 2 {
		t.Fatalf("other game: derived = %d after %d builds", got, builds.Load())
	}
}
