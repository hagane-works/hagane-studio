package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"haganeworks/internal/gameindex"
	"haganeworks/internal/gameindex/gameindextest"
)

func writeSyntheticGame(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Hearts of Iron IV")
	err := fs.WalkDir(gameindextest.Game(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := fs.ReadFile(gameindextest.Game(), name)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "gfx"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func (a *testAPI) waitReady() gameindex.Status {
	a.t.Helper()
	deadline := time.Now().Add(time.Minute)
	for {
		var status gameindex.Status
		if code := a.call("GET", "/api/app/game-index", nil, &status); code != http.StatusOK {
			a.t.Fatalf("game-index status %d", code)
		}
		switch status.State {
		case "ready":
			return status
		case "failed", "missing":
			a.t.Fatalf("game index %+v", status)
		}
		if time.Now().After(deadline) {
			a.t.Fatalf("game index still %+v", status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestGameIndexRoutes(t *testing.T) {
	a := newTestAPI(t)
	t.Cleanup(func() { gameindex.SetDefault(nil) })

	var status gameindex.Status
	if code := a.call("GET", "/api/app/game-index", nil, &status); code != http.StatusOK || status.State != "missing" {
		t.Fatalf("no game folder: %d %+v", code, status)
	}
	var tags map[string]any
	if code := a.call("GET", "/api/go/vanilla/country-tags", nil, &tags); code != http.StatusOK || tags["total"] != float64(0) {
		t.Fatalf("country-tags without a game: %d %v", code, tags)
	}

	for route, want := range map[string]string{
		"/api/go/vanilla/country-aliases": `{"aliases":{},"game_version":""}`,
		"/api/go/vanilla/state-content":   `{"game_version":"","states":{}}`,
		"/api/go/vanilla/dlc-states":      `{"dlc_names":[],"game_version":"","states":{}}`,
		"/api/go/vanilla/tech-sections":   `{"folders":{},"game_version":""}`,
		"/api/go/vanilla/unit-names":      `{"equipment":{},"game_version":"","subunits":{}}`,
	} {
		var body json.RawMessage
		if code := a.call("GET", route, nil, &body); code != http.StatusOK || compactJSON(t, body) != want {
			t.Fatalf("%s without a game: %d %s", route, code, body)
		}
	}

	game := writeSyntheticGame(t)
	if code := a.call("PUT", "/api/app/settings", map[string]any{"game_dir": game}, nil); code != http.StatusOK {
		t.Fatalf("settings %d", code)
	}
	status = a.waitReady()
	if status.GameVersion != "1.19.3.0" || status.Done != status.Total || status.Total == 0 || status.BuiltAt == "" || status.Error != "" {
		t.Fatalf("ready status %+v", status)
	}

	if code := a.call("GET", "/api/go/vanilla/country-tags", nil, &tags); code != http.StatusOK || tags["total"].(float64) < 10 {
		t.Fatalf("country-tags %d %v", code, tags["total"])
	}
	first := tags["tags"].([]any)[0].(map[string]any)
	for _, key := range []string{"tag", "name_en", "name_zh", "i18n_key", "ideology_default", "flag_emoji", "continent", "names"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("country tag misses %s: %v", key, first)
		}
	}
	var ideas map[string]any
	if code := a.call("GET", "/api/go/vanilla/ideas/index", nil, &ideas); code != http.StatusOK || len(ideas["all_ids"].([]any)) == 0 {
		t.Fatalf("ideas index %d %v", code, ideas)
	}
	for _, key := range []string{"meta", "by_group", "by_country", "all_ids"} {
		if _, ok := ideas[key]; !ok {
			t.Fatalf("ideas index misses %s", key)
		}
	}
	var picker map[string]any
	if code := a.call("GET", "/api/go/tags", nil, &picker); code != http.StatusOK || picker["total"].(float64) == 0 {
		t.Fatalf("tags %d %v", code, picker)
	}
	checkRound2Routes(t, a)

	if code := a.call("POST", "/api/app/game-index/rebuild", nil, &status); code != http.StatusOK || (status.State != "building" && status.State != "ready") {
		t.Fatalf("rebuild %d %+v", code, status)
	}
	a.waitReady()

	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(a.dir), "game-index", "*", "manifest.json"))
	if len(matches) != 1 {
		t.Fatalf("manifests %v", matches)
	}
}

func compactJSON(t *testing.T, raw []byte) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func checkRound2Routes(t *testing.T, a *testAPI) {
	t.Helper()
	get := func(route string) map[string]json.RawMessage {
		t.Helper()
		var body map[string]json.RawMessage
		if code := a.call("GET", route, nil, &body); code != http.StatusOK {
			t.Fatalf("%s: %d", route, code)
		}
		if compactJSON(t, body["game_version"]) != `"1.19.3.0"` {
			t.Fatalf("%s game_version %s", route, body["game_version"])
		}
		return body
	}
	expect := func(route, key, want string) {
		t.Helper()
		if got := compactJSON(t, get(route)[key]); got != want {
			t.Fatalf("%s %s:\n got %s\nwant %s", route, key, got, want)
		}
	}
	aliases := get("/api/go/vanilla/country-aliases")
	var byTag map[string][]string
	if err := json.Unmarshal(aliases["aliases"], &byTag); err != nil || len(byTag["GER"]) != 1 || byTag["GER"][0] != "germany" ||
		len(byTag["SOV"]) != 3 {
		t.Fatalf("country aliases %s (%v)", aliases["aliases"], err)
	}
	expect("/api/go/vanilla/state-content", "states", `{"1":{"coastalBunkers":1,"navalBases":1},`+
		`"2":{"bunkers":1,"hasDlc":1,"landmarks":["landmark_test_tower"]},"3":{"datedBlocks":1},"4":{"hasDlc":2},`+
		`"5":{"dams":1,"otherContent":1},"6":{"resistance":1},"7":{"forceLink":true}}`)
	expect("/api/go/vanilla/dlc-states", "dlc_names", `["La Resistance","Man the Guns"]`)
	expect("/api/go/vanilla/dlc-states", "states", `{"2":["La Resistance"],"4":["Man the Guns","La Resistance"]}`)
	expect("/api/go/vanilla/tech-sections", "folders", `{"electronics_folder":{"count":0,"sub_sections":[]},`+
		`"infantry_folder":{"count":2,"sub_sections":[`+
		`{"name":"infantry_subtitle_weapons","names":{"en":"Weapons and Equipment","zh":"武器和装备"},"sub_key":"weapons","text_key":"INFANTRY_TITLE_WEAPONS","y":100},`+
		`{"name":"infantry_subtitle_motorized","names":{"en":"Motorised"},"sub_key":"motorized","text_key":"INFANTRY_TITLE_MOTORISED","y":510}]},`+
		`"support_folder":{"count":1,"sub_sections":[`+
		`{"name":"support_subtitle_trains","names":{"en":"Trains"},"sub_key":"trains","text_key":"SUPPORT_TITLE_TRAINS","y":300}]}}`)
	expect("/api/go/vanilla/unit-names", "subunits", `{"fighter":{"en":"Fighter"},"infantry":{"en":"Infantry","zh":"步兵"}}`)
	expect("/api/go/vanilla/unit-names", "equipment", `{"artillery_equipment":{},`+
		`"infantry_equipment":{"en":"Infantry Equipment","zh":"步兵装备"},`+
		`"infantry_equipment_1":{"en":"Infantry Equipment I","zh":"步兵装备 I"},`+
		`"small_plane_cas_airframe":{"en":"Fighter (CAS)"}}`)
}
