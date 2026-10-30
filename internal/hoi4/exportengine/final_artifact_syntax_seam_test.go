package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFinalArtifactSyntaxFindingReachesPublishedArtifact(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 104,
		Project:   json.RawMessage(`{"id":104,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {
				json.RawMessage(`{"id":1,"dest_path":"common/ideas/seam_bad.txt","content":"ideas = {\n GER_bad = {\n","source":"created","raw_kind":"verbatim_override","is_binary":false}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !containsString(artifact.Coverage, FinalArtifactSyntaxCoverage) {
		t.Fatalf("production artifact skipped final syntax scan; coverage=%v", artifact.Coverage)
	}
	warnings := strings.Join(artifact.Warnings, "\n")
	for _, want := range []string{
		"common/ideas/seam_bad.txt:1", "clausewitz.unclosed_block",
		"nearby:", "player impact:", "fix in:",
	} {
		if !strings.Contains(warnings, want) {
			t.Errorf("published finding lost %q: %s", want, warnings)
		}
	}
}
