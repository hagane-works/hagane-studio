package worldmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"haganeworks/internal/domains"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

func heapMB() float64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.HeapAlloc) / (1 << 20)
}

func TestRealGame(t *testing.T) {
	dir := os.Getenv("HOI4_GAME_DIR")
	if dir == "" {
		t.Skip("set HOI4_GAME_DIR to run against a real installation")
	}
	store := project.NewStore(t.TempDir())
	record, err := workbench.CreateProject(store, workbench.CreateInput{Name: "Real Map", CountryTag: "TST", CountryName: "Testland"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Get(project.Int64(record["id"]))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, domains.Deps{
		Open: func(*http.Request) (*project.Project, error) { return p, nil },
		Game: func() (fs.FS, error) { return os.DirFS(dir), nil },
	})
	serve := func(req *http.Request) (*httptest.ResponseRecorder, time.Duration) {
		started := time.Now()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec, time.Since(started)
	}
	t.Logf("heap before: %.0f MB", heapMB())
	for round := 1; round <= 2; round++ {
		for _, target := range []string{"?date=1936.1.1", "/provinces.png", "/rivers", "/check", "/land"} {
			rec, took := serve(httptest.NewRequest(http.MethodGet, testPrefix+target, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("round %d %s: %d %s", round, target, rec.Code, truncate(rec.Body.String()))
			}
			t.Logf("round %d %-16s %3d %9d bytes %v", round, target, rec.Code, rec.Body.Len(), took.Round(time.Millisecond))
		}
		t.Logf("heap after round %d: %.0f MB", round, heapMB())
	}

	island := image.NewRGBA(image.Rect(0, 0, 60, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 60; x++ {
			c := color.RGBA{40, 80, 170, 255}
			if dx, dy := float64(x-30)/26, float64(y-15)/12; dx*dx+dy*dy <= 1 {
				c = color.RGBA{160, 210, 100, 255}
			}
			island.SetRGBA(x, y, c)
		}
	}
	var sketch bytes.Buffer
	if err := png.Encode(&sketch, island); err != nil {
		t.Fatal(err)
	}
	for round := 1; round <= 2; round++ {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, _ := form.CreateFormFile("sketch", "island.png")
		_, _ = part.Write(sketch.Bytes())
		for key, value := range map[string]string{"x": "200", "y": "1100", "width": "120", "height": "60", "seed": "9"} {
			_ = form.WriteField(key, value)
		}
		_ = form.Close()
		req := httptest.NewRequest(http.MethodPost, testPrefix+"/land-preview", &body)
		req.Header.Set("Content-Type", form.FormDataContentType())
		rec, took := serve(req)
		if rec.Code != http.StatusOK {
			t.Fatalf("land preview: %d %s", rec.Code, truncate(rec.Body.String()))
		}
		t.Logf("land-preview round %d: %v, %d bytes, %s", round, took.Round(time.Millisecond), rec.Body.Len(), truncate(rec.Body.String()[:min(200, rec.Body.Len())]))
		t.Logf("heap after land preview %d: %.0f MB", round, heapMB())
	}
	if rows, _ := p.Rows(additionsTable); len(rows) != 0 {
		t.Fatal("a preview saved a record")
	}

	rec, _ := serve(httptest.NewRequest(http.MethodGet, testPrefix+"?date=1936.1.1", nil))
	var payload struct {
		Provinces [][]any `json:"provinces"`
		States    [][]any `json:"states"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	owner := map[int]string{}
	for _, row := range payload.States {
		owner[int(row[0].(float64))], _ = row[2].(string)
	}
	members := map[int][]int{}
	for _, row := range payload.Provinces {
		if state := int(row[6].(float64)); owner[state] == "FRA" && row[2].(float64) == 0 {
			members[state] = append(members[state], int(row[0].(float64)))
		}
	}
	painted := 0
	for _, provinces := range members {
		if len(provinces) >= 2 {
			painted = provinces[len(provinces)-1]
			break
		}
	}
	if painted == 0 {
		t.Fatal("no French state of two land provinces")
	}
	edit := httptest.NewRequest(http.MethodPost, testPrefix+"/restate", strings.NewReader(fmt.Sprintf(`{"provinces":{"%d":"GER"}}`, painted)))
	rec, took := serve(edit)
	if rec.Code != http.StatusOK {
		t.Fatalf("restate %d: %d %s", painted, rec.Code, truncate(rec.Body.String()))
	}
	states, _ := p.Rows(statesTable)
	t.Logf("restate province %d: %v, %s; %d state rows", painted, took.Round(time.Millisecond), truncate(rec.Body.String()), len(states))
	rec, took = serve(httptest.NewRequest(http.MethodDelete, testPrefix+"/land?latest=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("undo: %d %s", rec.Code, rec.Body.String())
	}
	if states, _ := p.Rows(statesTable); len(states) != 0 {
		t.Fatalf("undo left %d state rows", len(states))
	}
	t.Logf("undo: %v; heap %.0f MB", took.Round(time.Millisecond), heapMB())

	sweepIdle(0)
	t.Logf("heap after letting the map's caches go: %.0f MB", heapMB())
}
