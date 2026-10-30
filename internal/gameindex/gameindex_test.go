package gameindex

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

func testDatasets(builds *atomic.Int64, versionB int) []Dataset {
	return []Dataset{
		{Name: "b", Version: versionB, Deps: []string{"a"}, Build: func(ctx context.Context, in *Input) ([]byte, error) {
			builds.Add(1)
			return append([]byte(`{"from_a":`), append(in.Dataset("a"), '}')...), nil
		}},
		{Name: "a", Version: 1, Build: func(ctx context.Context, in *Input) ([]byte, error) {
			builds.Add(1)
			raw, err := fs.ReadFile(in.FS, "value.txt")
			if err != nil {
				return nil, err
			}
			return []byte(`"` + string(raw) + `"`), nil
		}},
	}
}

func testGame() fstest.MapFS {
	return fstest.MapFS{
		"value.txt":              {Data: []byte("v1")},
		"launcher-settings.json": {Data: []byte(`{"rawVersion": "1.19.3.0"}`)},
	}
}

func wait(t *testing.T, x *Index) Status {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	status, err := x.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func TestBuildStoreAndReuse(t *testing.T) {
	root := t.TempDir()
	game := testGame()
	var builds atomic.Int64
	x, err := New(func() (fs.FS, error) { return game, nil }, testDatasets(&builds, 1), root)
	if err != nil {
		t.Fatal(err)
	}
	status := wait(t, x)
	if status.State != "ready" || status.GameVersion != "1.19.3.0" || status.Done != 2 || status.Total != 2 || status.BuiltAt == "" {
		t.Fatalf("status %+v", status)
	}
	got, err := x.Get(context.Background(), "b")
	if err != nil || string(got) != `{"from_a":"v1"}` {
		t.Fatalf("b = %s, %v", got, err)
	}
	if builds.Load() != 2 {
		t.Fatalf("builds = %d", builds.Load())
	}
	dirs, _ := os.ReadDir(root)
	if len(dirs) != 1 {
		t.Fatalf("cache folders %v", dirs)
	}
	for _, name := range []string{"manifest.json", "a.json", "b.json"} {
		if _, err := os.Stat(filepath.Join(root, dirs[0].Name(), name)); err != nil {
			t.Fatal(err)
		}
	}

	builds.Store(0)
	again, _ := New(func() (fs.FS, error) { return game, nil }, testDatasets(&builds, 1), root)
	if s := wait(t, again); s.State != "ready" || s.BuiltAt != status.BuiltAt {
		t.Fatalf("warm status %+v", s)
	}
	if got, _ := again.Get(context.Background(), "b"); string(got) != `{"from_a":"v1"}` || builds.Load() != 0 {
		t.Fatalf("warm b = %s after %d builds", got, builds.Load())
	}

	newer, _ := New(func() (fs.FS, error) { return game, nil }, testDatasets(&builds, 2), root)
	wait(t, newer)
	if builds.Load() != 1 {
		t.Fatalf("version bump rebuilt %d datasets", builds.Load())
	}

	builds.Store(0)
	game["launcher-settings.json"] = &fstest.MapFile{Data: []byte(`{"rawVersion": "1.19.4.0"}`), ModTime: time.Now().Add(time.Hour)}
	game["value.txt"] = &fstest.MapFile{Data: []byte("v2")}
	if got, _ := newer.Get(context.Background(), "b"); string(got) != `{"from_a":"v2"}` || builds.Load() != 2 {
		t.Fatalf("after update b = %s after %d builds", got, builds.Load())
	}
	if s := newer.Status(); s.GameVersion != "1.19.4.0" {
		t.Fatalf("version %+v", s)
	}

	builds.Store(0)
	newer.Rebuild()
	wait(t, newer)
	if builds.Load() != 2 {
		t.Fatalf("rebuild built %d", builds.Load())
	}
}

func TestFailureAndDependents(t *testing.T) {
	var builds atomic.Int64
	game := fstest.MapFS{}
	x, _ := New(func() (fs.FS, error) { return game, nil }, testDatasets(&builds, 1), "")
	status := wait(t, x)
	if status.State != "failed" || status.Error == "" {
		t.Fatalf("status %+v", status)
	}
	if _, err := x.Get(context.Background(), "a"); !errors.Is(err, ErrFailed) {
		t.Fatalf("a: %v", err)
	}
	if _, err := x.Get(context.Background(), "b"); !errors.Is(err, ErrFailed) {
		t.Fatalf("b: %v", err)
	}
	if _, err := x.Get(context.Background(), "nope"); err == nil {
		t.Fatal("unknown dataset")
	}
}

func TestNoGameAndPeek(t *testing.T) {
	var x *Index
	if _, err := x.Get(context.Background(), "a"); !errors.Is(err, ErrNoGame) {
		t.Fatalf("nil index: %v", err)
	}
	none, _ := New(func() (fs.FS, error) { return nil, errors.New("not set") }, nil, "")
	if s := none.Status(); s.State != "missing" {
		t.Fatalf("missing %+v", s)
	}
	if _, err := none.Peek("a"); !errors.Is(err, ErrNoGame) {
		t.Fatalf("peek: %v", err)
	}

	release := make(chan struct{})
	slow := []Dataset{{Name: "a", Version: 1, Build: func(ctx context.Context, in *Input) ([]byte, error) {
		<-release
		return []byte(`1`), nil
	}}}
	game := testGame()
	building, _ := New(func() (fs.FS, error) { return game, nil }, slow, "")
	if _, err := building.Peek("a"); !errors.Is(err, ErrBuilding) {
		t.Fatalf("peek while building: %v", err)
	}
	if s := building.Status(); s.State != "building" || s.Step != "a" {
		t.Fatalf("building %+v", s)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := building.Get(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get with a cancelled context: %v", err)
	}
	close(release)
	wait(t, building)
	if raw, err := building.Peek("a"); err != nil || string(raw) != "1" {
		t.Fatalf("peek when ready: %s %v", raw, err)
	}
}

func TestValueDecodesOncePerBuild(t *testing.T) {
	restore := UseStatic(map[string][]byte{"a": []byte(`3`)})
	defer restore()
	type key struct{}
	decodes := 0
	for i := 0; i < 3; i++ {
		v, err := Load(context.Background(), key{}, "a", func(raw []byte) (int, error) {
			decodes++
			return len(raw), nil
		})
		if err != nil || v != 1 {
			t.Fatalf("v = %d %v", v, err)
		}
	}
	if decodes != 1 {
		t.Fatalf("decoded %d times", decodes)
	}
	if _, err := Get(context.Background(), "other"); !errors.Is(err, ErrNoGame) {
		t.Fatalf("static unknown dataset: %v", err)
	}
	if err := CheckReady("a"); err != nil {
		t.Fatal(err)
	}
}
