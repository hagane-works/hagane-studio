package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBookmarkCountryBlockDropsNewContentKeepsVersion(t *testing.T) {
	bookmark := map[string]any{
		"id": 9, "bookmark_id": "demo_start", "name_en": "Demo", "source": "created",

		"filters": []string{"new_content", "continent"},
		"selectable_countries": []map[string]any{{
			"tag": "GER", "ideology": "fascism", "minor": false,
			"raw_extras": map[string]any{
				"version":     "base_game",
				"new_content": "yes",
			},
		}},
	}
	row, err := json.Marshal(bookmark)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := BuildNativeShadow(Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables:    map[string][]json.RawMessage{"bookmarks": {row}},
	}, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	var script string
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, "common/bookmarks/demo_start.txt") {
			script = string(entry.Bytes)
		}
	}
	if script == "" {
		t.Fatal("no bookmark script was emitted")
	}

	countryBlock := script
	if at := strings.Index(script, `"GER" = {`); at >= 0 {
		countryBlock = script[at:]
	}
	if strings.Contains(countryBlock, "new_content") {
		t.Fatalf("country block still emits new_content — the engine rejects it:\n%s", script)
	}
	if !strings.Contains(countryBlock, "version = base_game") {
		t.Fatalf("the real field `version` must still ship:\n%s", script)
	}

	if !strings.Contains(script, "filters = { new_content continent }") {
		t.Fatalf("bookmark-level `filters = { new_content … }` is legal and must survive:\n%s", script)
	}
}
