package exportengine

import (
	"context"
	"os"
	"testing"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/gameindextest"
	"haganeworks/internal/hoi4/emit/worldout"
)

func TestMain(m *testing.M) {
	restore := gameindex.UseStatic(gameindextest.Datasets())
	code := m.Run()
	restore()
	os.Exit(code)
}

func testWorldTables(t *testing.T) (*worldout.Tables, error) {
	t.Helper()
	gameindextest.Use(t)
	return worldout.LoadVanillaTables(context.Background())
}
