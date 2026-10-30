package tools

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/build"
	"haganeworks/internal/gameindex/gameindextest"
)

func TestMain(m *testing.M) {
	restore := gameindex.UseStatic(gameindextest.Datasets())
	code := m.Run()
	restore()
	os.Exit(code)
}

func useGameIndex(t *testing.T, game fs.FS) {
	t.Helper()
	x, err := gameindex.New(func() (fs.FS, error) { return game, nil }, build.Datasets(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gameindex.SetDefault(x))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := x.Wait(ctx); err != nil {
		t.Fatal(err)
	}
}
