package build

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"haganeworks/internal/gameindex"
)

func buildAll(t *testing.T, fsys fs.FS, root string) (map[string][]byte, time.Duration) {
	t.Helper()
	x, err := gameindex.New(func() (fs.FS, error) { return fsys, nil }, Datasets(), root)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	out := map[string][]byte{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	for _, d := range Datasets() {
		data, err := x.Get(ctx, d.Name)
		if err != nil {
			t.Errorf("%s: %v", d.Name, err)
			continue
		}
		out[d.Name] = data
	}
	status, err := x.Wait(ctx)
	elapsed := time.Since(start)
	if err != nil || status.State != "ready" {
		t.Errorf("status after build = %+v (%v)", status, err)
	}
	if raw, err := os.ReadFile(filepath.Join(root, firstDir(t, root), "manifest.json")); err == nil {
		t.Logf("manifest: %s", raw)
	}
	return out, elapsed
}

func TestRealInstall(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("HOI4_GAME_DIR not set")
	}
	fsys := os.DirFS(dir)
	root := t.TempDir()
	data, cold := buildAll(t, fsys, root)
	t.Logf("cold build: %d datasets in %v", len(data), cold)
	_, warm := buildAll(t, fsys, root)
	t.Logf("warm start (stored files reused): %v", warm)

	if out := os.Getenv("HOI4_INDEX_DUMP_DIR"); out != "" {
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, raw := range data {
			if err := os.WriteFile(filepath.Join(out, name+".json"), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	checkSanity(t, data)
	ref := os.Getenv("HOI4_INDEX_REFERENCE_DIR")
	if ref == "" {

		ref, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	compareWithReference(t, data, ref)
}

func firstDir(t *testing.T, root string) string {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() {
			return e.Name()
		}
	}
	return ""
}
