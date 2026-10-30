package exportengine

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestSnapshotFingerprintStableAcrossMapOrderAndCaptureTime(t *testing.T) {
	a := Snapshot{Schema: SnapshotSchema, ProjectID: 7, Project: json.RawMessage(`{"id":7}`), Tables: map[string][]json.RawMessage{
		"ideas": {json.RawMessage(`{"id":2}`)}, "characters": {json.RawMessage(`{"id":1}`)},
	}}
	b := Snapshot{Schema: SnapshotSchema, ProjectID: 7, Project: json.RawMessage(`{"id":7}`), Tables: map[string][]json.RawMessage{
		"characters": {json.RawMessage(`{"id":1}`)}, "ideas": {json.RawMessage(`{"id":2}`)},
	}}
	fa, err := SnapshotFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := SnapshotFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if fa != fb {
		t.Fatalf("fingerprints differ: %s %s", fa, fb)
	}
}

func TestSnapshotFingerprintRejectsInvalidJSON(t *testing.T) {
	_, err := SnapshotFingerprint(Snapshot{Schema: SnapshotSchema, Project: json.RawMessage(`{}`), Tables: map[string][]json.RawMessage{"ideas": {json.RawMessage(`{`)}}})
	if err == nil {
		t.Fatal("invalid JSON accepted")
	}
}

func TestCoreSnapshotCoversExportDomains(t *testing.T) {
	names := make([]string, 0, len(coreSnapshotSources))
	for _, source := range coreSnapshotSources {
		names = append(names, source.name)
	}
	for _, required := range []string{
		"characters", "focus_file_policies", "equipments", "oob_ships",
		"special_projects", "scripted_localisations", "gui_overlays",
		"project_raw_files", "entity_obs_paths", "project_obs_assets",
		"ai_strategy_plans", "ai_templates", "ui_panels",
	} {
		if !slices.Contains(names, required) {
			t.Fatalf("export snapshot is missing %q", required)
		}
	}
}
