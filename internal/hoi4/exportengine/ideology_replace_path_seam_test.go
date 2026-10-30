package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIdeologyReplacePathSeamShipsExactlyThePlannedCompleteFile(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 52,
		Project: json.RawMessage(`{
			"id":52,"name":"Demo","country_tag":"GER","export_languages":["en"],
			"ideology_registry":{"groups":["democratic"],"sub_ideologies":[
				{"id":"demo_liberalism","group":"democratic","name":{"en":"Demo Liberalism"}}
			]}
		}`),
		Tables: map[string][]json.RawMessage{},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	descriptor := descriptorText(t, artifact)
	if strings.Count(descriptor, `replace_path="common/ideologies"`) != 1 {
		t.Fatalf("descriptor does not carry exactly one ideology replacement:\n%s", descriptor)
	}
	if artifact.Timeline == nil || !containsString(artifact.Timeline.Fingerprint.EffectiveReplacePaths, "common/ideologies") {
		t.Fatalf("timeline metadata does not match descriptor: %#v", artifact.Timeline)
	}
	count := 0
	for _, entry := range artifact.Entries {
		if entry.Path != "Demo/common/ideologies/00_ideologies.txt" {
			continue
		}
		count++
		text := string(entry.Bytes)
		if !strings.Contains(text, "demo_liberalism = {") {
			t.Fatalf("complete ideology file lost custom sub:\n%s", text)
		}
	}
	if count != 1 {
		t.Fatalf("planned complete ideology file count=%d", count)
	}
}

func TestIdeologyReplacePathNegativeControlsStayAdditive(t *testing.T) {
	project := `{
		"id":53,"name":"Demo","country_tag":"GER","export_languages":["en"],
		"ideology_registry":{"groups":["democratic"],"sub_ideologies":[
			{"id":"demo_liberalism","group":"democratic","name":{"en":"Demo Liberalism"}}
		]}`
	cases := map[string]Snapshot{
		"submod": {
			ProjectID: 53,
			Project:   json.RawMessage(project + `,"dependencies":[{"remote_file_id":"base"}]}`),
			Tables:    map[string][]json.RawMessage{},
		},
		"raw_override": {
			ProjectID: 54,
			Project:   json.RawMessage(project + `}`),
			Tables: map[string][]json.RawMessage{
				"project_raw_files": {json.RawMessage(`{
					"id":1,"dest_path":"common/ideologies/my_complete.txt",
					"content":"ideologies = { }","source":"created",
					"raw_kind":"verbatim_override","is_binary":false
				}`)},
			},
		},
	}
	for name, snapshot := range cases {
		t.Run(name, func(t *testing.T) {
			artifact, err := BuildNativeShadow(snapshot, nil, false)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if strings.Contains(descriptorText(t, artifact), `replace_path="common/ideologies"`) {
				t.Fatal("unsafe ideology replacement was added")
			}
			for _, entry := range artifact.Entries {
				if entry.Path == "Demo/common/ideologies/00_ideologies.txt" {
					t.Fatal("platform ideology file must not compete with submod/raw ownership")
				}
			}
		})
	}
}
