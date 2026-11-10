package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// An imported mod gives its starting ideas in its own history (add_ideas); an
// idea definition is not a starting state. Export must not add every imported
// idea at game start.
func TestImportedIdeasAreNotGrantedAtStart(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"ideas": {
				json.RawMessage(`{"id":1,"idea_id":"AFG_imported","idea_type":"country","country_tag":"AFG","auto_attach_on_start":false,"source":"imported"}`),
				// Imported before the import stopped storing true: no value.
				json.RawMessage(`{"id":2,"idea_id":"AFG_unset","idea_type":"country","country_tag":"AFG","source":"imported"}`),
				json.RawMessage(`{"id":3,"idea_id":"GER_created","idea_type":"country","country_tag":"GER","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/common/on_actions/") && strings.Contains(string(entry.Bytes), "AFG_") {
			t.Fatalf("%s grants imported ideas:\n%s", entry.Path, entry.Bytes)
		}
	}
	if !strings.Contains(onActionsFile(t, artifact, "GER"), "GER_created") {
		t.Fatal("an idea made in the app is no longer granted at start")
	}
}
