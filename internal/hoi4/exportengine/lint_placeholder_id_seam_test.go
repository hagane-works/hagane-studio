package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPlaceholderIDSeamNativeShadowShipsWarning(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"ideas": {
				json.RawMessage(`{"id":1,"idea_id":"new_idea_3","country_tag":"GER","idea_type":"country","name":"New Idea","source":"created"}`),
				json.RawMessage(`{"id":2,"idea_id":"GER_demo","country_tag":"GER","idea_type":"country","name":"Demo","name_en":"Demo","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("a placeholder-id notice must never fail the export: %v", err)
	}
	var hit string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "Workbench placeholder IDs") {
			hit = w
		}
	}
	if hit == "" {
		t.Fatalf("headline not pushed to artifact.Warnings (seam hunk in native.go not landed?); warnings=%v", artifact.Warnings)
	}
	want := "Workbench placeholder IDs (1): new_idea_3. The game has no name for these and shows the raw ID instead; " +
		"rename them in the workbench (ideas: 1)."
	if hit != want {
		t.Fatalf("unexpected headline:\n got %q\nwant %q", hit, want)
	}

	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "PLACEHOLDER") {
			t.Fatalf("this lint must not add a zip log entry: %s", entry.Path)
		}
	}
}
