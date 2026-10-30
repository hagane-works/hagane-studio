package focus

import (
	"os"
	"testing"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/gameindextest"
)

func TestMain(m *testing.M) {
	restore := gameindex.UseStatic(gameindextest.Datasets())
	code := m.Run()
	restore()
	os.Exit(code)
}
