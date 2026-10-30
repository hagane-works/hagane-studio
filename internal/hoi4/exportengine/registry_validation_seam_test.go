package exportengine

import (
	"strings"
	"testing"
)

func TestRegistryValidationFindingReachesPublishedNativeArtifact(t *testing.T) {
	snapshot := rawFileSnapshot(`{"id":1,"dest_path":"common/decisions/categories/non_ascii.txt","content":"中文分类 = { icon = generic }\n","source":"created","raw_kind":"verbatim_override","is_binary":false}`)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("registry findings must never fail export: %v", err)
	}
	if !containsString(artifact.Coverage, RegistryValidationCoverage) {
		t.Fatalf("production artifact skipped registry validation; coverage=%v", artifact.Coverage)
	}
	log, ok := findEntry(artifact, artifactRoot(artifact)+"/"+RegistryValidationLogName)
	if !ok || !strings.Contains(string(log.Bytes), "script_id.non_ascii") {
		t.Fatalf("registry finding did not reach shipped log: %q", log.Bytes)
	}
	if !strings.Contains(strings.Join(artifact.Warnings, "\n"), "Content source check:") {
		t.Fatalf("registry headline did not reach export UI: %v", artifact.Warnings)
	}
}
