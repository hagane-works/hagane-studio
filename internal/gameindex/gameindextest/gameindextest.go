package gameindextest

import (
	"embed"
	"io/fs"
	"path"
	"strings"
	"testing"

	"haganeworks/internal/gameindex"
)

//go:embed all:game
var gameFiles embed.FS

//go:embed datasets
var datasetFiles embed.FS

func Game() fs.FS {
	sub, err := fs.Sub(gameFiles, "game")
	if err != nil {
		panic(err)
	}
	return sub
}

func Datasets() map[string][]byte {
	out := map[string][]byte{}
	entries, _ := fs.ReadDir(datasetFiles, "datasets")
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := datasetFiles.ReadFile(path.Join("datasets", entry.Name()))
		if err != nil {
			panic(err)
		}
		out[strings.TrimSuffix(entry.Name(), ".json")] = raw
	}
	return out
}

func Use(t testing.TB) {
	t.Helper()
	t.Cleanup(gameindex.UseStatic(Datasets()))
}
