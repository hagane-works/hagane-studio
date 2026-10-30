package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"haganeworks/internal/gamedir"
	"haganeworks/internal/reftables"
	"haganeworks/internal/settings"
)

type testAPI struct {
	t   *testing.T
	mux *http.ServeMux
	dir string
}

func newTestAPI(t *testing.T) *testAPI { return newTestAPIWith(t, nil) }

func noInstalledGame() (gamedir.Install, error) { return gamedir.Install{}, gamedir.ErrNotFound }

const testPort = 7384

func newTestAPIWith(t *testing.T, tables map[string]string) *testAPI {
	t.Helper()
	dir := t.TempDir()
	for name, body := range tables {
		path := filepath.Join(dir, "tables", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if len(tables) > 0 {
		t.Cleanup(func() { reftables.Load("") })
	}
	store, err := settings.Open(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	projects := filepath.Join(dir, "projects")
	if _, err := store.Update(func(s *settings.Settings) { s.ProjectsDir = projects }); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Register(mux, Deps{Settings: store, GameIndexDir: filepath.Join(dir, "game-index"), Port: testPort, FindGame: noInstalledGame})
	return &testAPI{t: t, mux: mux, dir: projects}
}

func (a *testAPI) call(method, path string, body, out any) int {
	a.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			a.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	if out != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			a.t.Fatalf("%s %s: %v in %s", method, path, err, rec.Body.String())
		}
	}
	return rec.Code
}

func TestProjectLifecycle(t *testing.T) {
	a := newTestAPI(t)

	var created map[string]any
	if status := a.call("POST", "/api/go/projects", map[string]any{"name": "Test Mod", "country_tag": "TST", "country_name": "Testland"}, &created); status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, created)
	}
	id := int64(created["id"].(float64))
	if created["name"] != "Test Mod" || created["country_tag"] != "TST" {
		t.Fatalf("created = %v", created)
	}

	entries, err := os.ReadDir(a.dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("creation folder: %v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(a.dir, entries[0].Name(), "project.json")); err != nil {
		t.Fatal(err)
	}

	var list struct {
		Items []map[string]any `json:"items"`
	}
	if status := a.call("GET", "/api/go/projects", nil, &list); status != http.StatusOK || len(list.Items) != 1 {
		t.Fatalf("list: %d %v", status, list)
	}

	path := "/api/go/projects/" + jsonNumber(id)
	var updated map[string]any
	if status := a.call("PUT", path, map[string]any{"name": "Renamed"}, &updated); status != http.StatusOK || updated["name"] != "Renamed" {
		t.Fatalf("update: %d %v", status, updated)
	}
	var got map[string]any
	if status := a.call("GET", path, nil, &got); status != http.StatusOK || got["name"] != "Renamed" {
		t.Fatalf("get after update: %d %v", status, got)
	}

	if status := a.call("DELETE", path, nil, nil); status != http.StatusNoContent && status != http.StatusOK {
		t.Fatalf("delete: %d", status)
	}
	if status := a.call("GET", path, nil, nil); status != http.StatusNotFound {
		t.Fatalf("get after delete: %d", status)
	}
}

func TestCountriesOfAProject(t *testing.T) {
	a := newTestAPI(t)
	var created map[string]any
	a.call("POST", "/api/go/projects", map[string]any{"name": "Countries", "country_tag": "TST", "country_name": "Testland"}, &created)
	path := "/api/go/projects/" + jsonNumber(int64(created["id"].(float64)))

	var country map[string]any
	if status := a.call("POST", path+"/countries", map[string]any{"tag": "ABC", "name": "Abcland"}, &country); status != http.StatusCreated {
		t.Fatalf("add country: %d %v", status, country)
	}
	var countries struct {
		Countries []map[string]any `json:"countries"`
		Total     int              `json:"total"`
	}
	if status := a.call("GET", path+"/countries", nil, &countries); status != http.StatusOK || countries.Total != 2 {
		t.Fatalf("list countries: %d %v", status, countries)
	}
	tags := map[string]bool{}
	for _, c := range countries.Countries {
		tags[c["tag"].(string)] = true
	}
	if !tags["TST"] || !tags["ABC"] {
		t.Fatalf("countries = %v", tags)
	}

	var project map[string]any
	a.call("GET", path, nil, &project)
	listed := map[string]bool{}
	for _, tag := range project["country_tags"].([]any) {
		listed[tag.(string)] = true
	}
	if !listed["TST"] || !listed["ABC"] {
		t.Fatalf("project country_tags = %v", project["country_tags"])
	}

	if status := a.call("POST", path+"/countries", map[string]any{"tag": "ABC", "name": "Again"}, nil); status < 400 {
		t.Fatalf("duplicate tag accepted: %d", status)
	}

	if status := a.call("POST", path+"/countries", map[string]any{"tag": "a1", "name": "Bad"}, nil); status < 400 {
		t.Fatalf("invalid tag accepted: %d", status)
	}
}

func TestExportAndInstall(t *testing.T) {
	a := newTestAPI(t)
	var created map[string]any
	a.call("POST", "/api/go/projects", map[string]any{"name": "Export Mod", "country_tag": "TST", "country_name": "Testland"}, &created)
	id := int64(created["id"].(float64))

	modDir := filepath.Join(t.TempDir(), "mod")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if status := a.call("PUT", "/api/app/settings", map[string]any{"mod_dir": filepath.ToSlash(modDir)}, nil); status != http.StatusOK {
		t.Fatalf("settings: %d", status)
	}
	var installed map[string]any
	if status := a.call("POST", "/api/app/projects/"+jsonNumber(id)+"/install", map[string]any{}, &installed); status != http.StatusOK {
		t.Fatalf("install: %d %v", status, installed)
	}
	descriptor, ok := installed["descriptor"].(string)
	if !ok {
		t.Fatalf("install result = %v", installed)
	}
	if _, err := os.Stat(descriptor); err != nil {
		t.Fatalf("descriptor not written: %v", err)
	}
}

func jsonNumber(id int64) string {
	raw, _ := json.Marshal(id)
	return string(raw)
}

func TestBaseModFolders(t *testing.T) {
	a := newTestAPI(t)
	var created map[string]any
	a.call("POST", "/api/go/projects", map[string]any{"name": "Submod", "country_tag": "TST", "country_name": "Testland", "dependencies": []string{"Old Base"}}, &created)
	id := jsonNumber(int64(created["id"].(float64)))
	path := "/api/app/projects/" + id + "/base-mods"
	modDir := filepath.Join(t.TempDir(), "820260968")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "descriptor.mod"), []byte("\ufeffversion=\"1.0\"\nname=\"The Road to 56\"\nsupported_version=\"1.16.*\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	type baseMods struct {
		Dependency string `json:"dependency"`
		BaseMods   []struct {
			Name   string `json:"name"`
			Folder string `json:"folder"`
			Found  bool   `json:"found"`
		} `json:"base_mods"`
	}
	describe := func(got baseMods) string {
		out := ""
		for _, mod := range got.BaseMods {
			out += fmt.Sprintf("[%s|%s|%v]", mod.Name, mod.Folder, mod.Found)
		}
		return out
	}

	for range 2 {
		var got baseMods
		if status := a.call("PUT", path, map[string]any{"folder": modDir}, &got); status != http.StatusOK || got.Dependency != "The Road to 56" {
			t.Fatalf("set: %d %+v", status, got)
		}
		if want := "[Old Base||false][The Road to 56|" + modDir + "|true]"; describe(got) != want {
			t.Fatalf("base mods = %s, want %s", describe(got), want)
		}
	}
	var project map[string]any
	a.call("GET", "/api/go/projects/"+id, nil, &project)
	if fmt.Sprint(project["dependencies"]) != "[Old Base The Road to 56]" || project["base_mods"] != nil {
		t.Fatalf("project = %v", project)
	}

	var refused map[string]any
	if status := a.call("PUT", path, map[string]any{"folder": t.TempDir()}, &refused); status != http.StatusUnprocessableEntity || refused["code"] != "not_a_mod" {
		t.Fatalf("not a mod: %d %v", status, refused)
	}
	if status := a.call("PUT", path, map[string]any{"dependency": "Kaiserreich", "folder": modDir}, nil); status != http.StatusUnprocessableEntity {
		t.Fatalf("unknown dependency: %d", status)
	}

	var got baseMods
	a.call("PUT", path, map[string]any{"dependency": "Old Base", "folder": modDir}, &got)
	if want := "[Old Base|" + modDir + "|true][The Road to 56|" + modDir + "|true]"; describe(got) != want {
		t.Fatalf("change: %s", describe(got))
	}
	a.call("PUT", path, map[string]any{"dependency": "Old Base", "folder": ""}, &got)
	if want := "[Old Base||false][The Road to 56|" + modDir + "|true]"; describe(got) != want {
		t.Fatalf("clear: %s", describe(got))
	}

	moved := modDir + "-moved"
	if err := os.Rename(modDir, moved); err != nil {
		t.Fatal(err)
	}
	a.call("GET", path, nil, &got)
	if want := "[Old Base||false][The Road to 56|" + modDir + "|false]"; describe(got) != want {
		t.Fatalf("gone: %s", describe(got))
	}

	a.call("PUT", "/api/go/projects/"+id, map[string]any{"dependencies": []string{"Old Base"}}, nil)
	a.call("GET", path, nil, &got)
	if describe(got) != "[Old Base||false]" {
		t.Fatalf("after removing: %s", describe(got))
	}
	entries, _ := os.ReadDir(a.dir)
	raw, err := os.ReadFile(filepath.Join(a.dir, entries[0].Name(), "project.json"))
	if err != nil || bytes.Contains(raw, []byte("base_mods")) {
		t.Fatalf("project.json keeps a base_mods entry: %v\n%s", err, raw)
	}
}
