package worldout

import (
	"context"
	"testing"

	"haganeworks/internal/gameindex/gameindextest"
)

func testTables(t *testing.T) (*Tables, error) {
	t.Helper()
	gameindextest.Use(t)
	return LoadVanillaTables(context.Background())
}
