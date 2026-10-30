package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBookmarkLanguageFindingReachesPublishedArtifact(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 51,
		Project: json.RawMessage(`{
			"id":51,"name":"Demo","country_tag":"GER","export_languages":["zh"]
		}`),
		Tables: map[string][]json.RawMessage{
			"bookmarks": {json.RawMessage(`{
				"id":1,"bookmark_id":"iron_curtain","date":"1964.5.1.12",
				"display_name_locs":{"zh":"铁幕"},"source":"created"
			}`)},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	for _, path := range []string{
		"Demo/localisation/english/replace/iron_curtain_l_english.yml",
		"Demo/localisation/simp_chinese/replace/iron_curtain_l_simp_chinese.yml",
	} {
		if !strings.Contains(contents[path], `IRON_CURTAIN_NAME: "铁幕"`) {
			t.Fatalf("published artifact has a dangling bookmark NAME in %s", path)
		}
	}
	if !strings.Contains(strings.Join(artifact.Warnings, "\n"),
		"the bookmark has no name in en, so its zh name was used") {
		t.Fatalf("bookmark language finding died before publish: %v", artifact.Warnings)
	}
}
