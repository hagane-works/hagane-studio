package exportjob

import (
	"encoding/json"
	"testing"

	"haganeworks/internal/hoi4/exportengine"
)

func TestBuildRequiresAbsoluteArtifactPath(t *testing.T) {
	_, err := Build(t.Context(), BuildInput{
		Snapshot:     exportengine.Snapshot{Schema: exportengine.SnapshotSchema, ProjectID: 7},
		ArtifactPath: "relative.zip",
	})
	if err == nil {
		t.Fatal("relative artifact path should fail")
	}
}

func TestArtifactFingerprintIsStable(t *testing.T) {
	first := artifactFingerprint("snapshot", "timeline")
	second := artifactFingerprint("snapshot", "timeline")
	if first == "" || first != second || !json.Valid(json.RawMessage(`{"ok":true}`)) {
		t.Fatalf("fingerprint=%q", first)
	}
}
