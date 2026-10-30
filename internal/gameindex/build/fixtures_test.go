package build

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/gameindextest"
)

func TestSyntheticFixtures(t *testing.T) {
	game := gameindextest.Game()
	x, err := gameindex.New(func() (fs.FS, error) { return game, nil }, Datasets(), "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	status, err := x.Wait(ctx)
	if err != nil || status.State != "ready" {
		t.Fatalf("synthetic build: %+v %v", status, err)
	}
	stored := gameindextest.Datasets()
	regen := os.Getenv("HOI4_REGEN_FIXTURES") == "1"
	dir := filepath.Join("..", "gameindextest", "datasets")
	for _, d := range Datasets() {
		data, err := x.Get(ctx, d.Name)
		if err != nil {
			t.Fatalf("%s: %v", d.Name, err)
		}
		if regen {
			if err := os.WriteFile(filepath.Join(dir, d.Name+".json"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if !bytes.Equal(stable(t, data), stable(t, stored[d.Name])) {
			t.Errorf("%s: gameindextest/datasets is out of date (HOI4_REGEN_FIXTURES=1 rewrites it)", d.Name)
		}
	}
}

func stable(t *testing.T, raw []byte) []byte {
	t.Helper()
	if raw == nil {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	var strip func(any)
	strip = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			delete(v, "generated_at")
			delete(v, "extracted_at")
			for _, child := range v {
				strip(child)
			}
		case []any:
			for _, child := range v {
				strip(child)
			}
		}
	}
	strip(v)
	out, _ := json.Marshal(v)
	return out
}
