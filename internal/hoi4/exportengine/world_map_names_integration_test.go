package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/guirender"
)

func TestGeneratedWorldNamesOverridePlaceholdersInExportAndGameView(t *testing.T) {
	const (
		enRaw = "localisation/english/new_world_l_english.yml"
		zhRaw = "localisation/simp_chinese/new_world_l_simp_chinese.yml"
		enFix = "localisation/english/replace/zz_game_view_l_english.yml"
		zhFix = "localisation/simp_chinese/replace/zz_game_view_l_simp_chinese.yml"
	)
	snapshot := Snapshot{
		ProjectID: 42,
		Project: json.RawMessage(`{
			"id":42,"name":"Demo","country_tag":"AAA",
			"export_languages":["en","zh"]
		}`),
		Tables: map[string][]json.RawMessage{
			"states": {
				json.RawMessage(`{
					"id":1,"state_id":1,"name":"STATE_1","source":"created",
					"display_name_locs":{"en":"The North","zh":"北境"},
					"provinces":[101],"victory_points":[{"province_id":101,"value":1}],
					"buildings":{},"raw_extras":{}
				}`),
			},
			"project_loc_entries": {
				locRow(1, "l_english", "VICTORY_POINTS_101", "Winterfell", enFix, "created", nil, 0, 0),
				locRow(2, "l_simp_chinese", "VICTORY_POINTS_101", "临冬城", zhFix, "created", nil, 0, 0),
			},
			"project_raw_files": {
				json.RawMessage(`{
					"id":11,"dest_path":"` + enRaw + `","raw_kind":"verbatim_override",
					"source":"created","content":"\ufeffl_english:\n VICTORY_POINTS_101:0 \"Town 1\"\n"
				}`),
				json.RawMessage(`{
					"id":12,"dest_path":"` + zhRaw + `","raw_kind":"verbatim_override",
					"source":"created","content":"\ufeffl_simp_chinese:\n VICTORY_POINTS_101:0 \"城镇 1\"\n"
				}`),
			},
		},
	}

	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, entry := range artifact.Entries {
		files[strings.TrimPrefix(entry.Path, "Demo/")] = string(entry.Bytes)
	}
	if got := files[enRaw]; !strings.Contains(got, `VICTORY_POINTS_101:0 "Town 1"`) {
		t.Fatalf("generated raw localisation no longer proves the placeholder conflict:\n%s", got)
	}
	stateLocPath, stateLoc := "", ""
	for name, body := range files {
		if strings.HasSuffix(name, "_states_l_english.yml") {
			stateLocPath = name
			stateLoc = body
			break
		}
	}
	if !strings.Contains(strings.ToLower(stateLocPath), "/replace/") {
		t.Fatalf("structured state localisation is not an override file: %q", stateLocPath)
	}
	if !strings.Contains(stateLoc, `STATE_1: "The North"`) || strings.Contains(stateLoc, `"State 1"`) {
		t.Fatalf("structured state localisation did not replace State 1:\n%s", stateLoc)
	}
	if got := files[enFix]; !strings.Contains(got, `VICTORY_POINTS_101: "Winterfell"`) || strings.Contains(got, `"Town 1"`) {
		t.Fatalf("capital replace localisation did not carry the authored name:\n%s", got)
	}
	if got := files[zhFix]; !strings.Contains(got, `VICTORY_POINTS_101: "临冬城"`) || strings.Contains(got, `"城镇 1"`) {
		t.Fatalf("Chinese capital replace localisation did not carry the authored name:\n%s", got)
	}

	projection, err := BuildNativePreviewProjection(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	index, err := guirender.NewIndex(projection.TextFiles)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"STATE_1":            "The North",
		"VICTORY_POINTS_101": "Winterfell",
	} {
		if got, _ := index.Localise(key); got != want {
			t.Errorf("GameView English %s = %q, want %q", key, got, want)
		}
	}
	zh := index.InLanguage("zh", nil)
	for key, want := range map[string]string{
		"STATE_1":            "北境",
		"VICTORY_POINTS_101": "临冬城",
	} {
		if got, _ := zh.Localise(key); got != want {
			t.Errorf("GameView Chinese %s = %q, want %q", key, got, want)
		}
	}
}
