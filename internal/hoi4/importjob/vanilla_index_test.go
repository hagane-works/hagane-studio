package importjob

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/gameindextest"
	"haganeworks/internal/project"
)

func useTestIndex(t *testing.T, reads *atomic.Int64, build func(ctx context.Context) ([]byte, error)) *gameindex.Index {
	t.Helper()
	game := fstest.MapFS{"launcher-settings.json": {Data: []byte(`{"rawVersion":"1.19.3.0"}`)}}
	fixtures := gameindextest.Datasets()
	dataset := func(name string) gameindex.Dataset {
		return gameindex.Dataset{Name: name, Version: 1, Build: func(ctx context.Context, _ *gameindex.Input) ([]byte, error) {
			if name == gameindex.CountryAliases && build != nil {
				return build(ctx)
			}
			return fixtures[name], nil
		}}
	}
	index, err := gameindex.New(func() (fs.FS, error) {
		reads.Add(1)
		return game, nil
	}, []gameindex.Dataset{dataset(gameindex.CountryAliases), dataset(gameindex.FocusTrees)}, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gameindex.SetDefault(index))
	return index
}

func TestEventsWorldReadsTheGameIndexOnce(t *testing.T) {
	var reads atomic.Int64
	index := useTestIndex(t, &reads, nil)
	if _, err := index.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	tags := []any{}
	for c := 0; c < 60; c++ {
		tags = append(tags, fmt.Sprintf("C%02d", c))
	}
	for f := 0; f < 20; f++ {
		var events strings.Builder
		fmt.Fprintf(&events, "add_namespace = mod_ns%d\n", f)
		for e := 0; e < 50; e++ {
			fmt.Fprintf(&events, "country_event = { id = mod_ns%d.%d option = { name = ok } }\n", f, e)
		}
		files[fmt.Sprintf("events/mod_%02d.txt", f)] = events.String()
	}
	files["events/germany.txt"] = "add_namespace = germany\ncountry_event = { id = germany.1 option = { name = ok } }\n"
	p, err := project.NewStore(t.TempDir()).Create(project.Row{"name": "t", "country_tag": "C00", "country_tags": tags}, "t")
	if err != nil {
		t.Fatal(err)
	}
	reads.Store(0)
	s := newSession(context.Background(), writeWorldTestMod(t, files), AllOptions(), p.Record())
	counts, err := s.stageEventsWorld(EventsWorldStageOptions{ProjectID: p.ID(), ImportEvents: true}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := reads.Load(); got > 4 {
		t.Errorf("the game folder was looked at %d times for 1,001 events and 60 countries", got)
	}

	if counts["event"] != 0 || counts["event_namespace"] != 0 {
		t.Errorf("counts = %v", counts)
	}
}

func TestImportWaitsForTheGameIndex(t *testing.T) {
	var reads atomic.Int64
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	useTestIndex(t, &reads, func(ctx context.Context) ([]byte, error) {
		select {
		case <-release:
			return gameindextest.Datasets()[gameindex.CountryAliases], nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	p := newWorldTestProject(t)
	m := NewManager()
	if _, err := m.Start(p, "waiting", writeWorldTestMod(t, worldTestMod), AllOptions()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		state, _ := m.Get(p.ID(), "waiting")
		if state.Phase == "game_index" {
			break
		}
		if state.Status != "running" || time.Now().After(deadline) {
			t.Fatalf("the import did not wait for the game index: %+v", state)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !m.Cancel(p.ID(), "waiting") {
		t.Fatal("cancel")
	}
	if state := waitForJob(t, m, p.ID(), "waiting"); state.Status != "cancelled" {
		t.Errorf("after cancel while waiting: %+v", state)
	}
	if events, _ := p.Rows("events"); len(events) != 0 {
		t.Errorf("a cancelled import wrote %d events", len(events))
	}

	close(release)
	if _, err := m.Start(p, "ready", writeWorldTestMod(t, worldTestMod), AllOptions()); err != nil {
		t.Fatal(err)
	}
	if state := waitForJob(t, m, p.ID(), "ready"); state.Status != "complete" {
		t.Fatalf("once the index is ready: %+v", state)
	}
	if events, _ := p.Rows("events"); len(events) == 0 {
		t.Error("no events imported")
	}
}
