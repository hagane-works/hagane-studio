package exportengine

import (
	"strings"
	"testing"
)

func TestWorldLintSeamNativeShadowShipsLogsAndHeadlines(t *testing.T) {
	snapshot := loadLintWorldSnapshot(t)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("a lint must never fail the export: %v", err)
	}
	root := artifactRoot(artifact)
	for _, name := range []string{PoliticsLogName, StateOrphanTagLogName} {
		entry, ok := findEntry(artifact, root+"/"+name)
		if !ok {
			t.Fatalf("%s not shipped in the artifact (native.go no longer calls ApplyWorldLints?) entries=%d", name, len(artifact.Entries))
		}
		want := loadLintWorldExpected(t, "expected"+name)
		if string(entry.Bytes) != want {
			t.Fatalf("%s differs from expectation:\n--- got\n%s\n--- want\n%s", name, entry.Bytes, want)
		}
	}
	var politics, orphan string
	for _, w := range artifact.Warnings {
		if strings.HasPrefix(w, "Politics export: ") {
			politics = w
		}
		if strings.HasPrefix(w, "State owners: ") {
			orphan = w
		}
	}
	if politics == "" || orphan == "" {
		t.Fatalf("headlines not pushed to artifact.Warnings (export page consumer); warnings=%v", artifact.Warnings)
	}
	if !strings.Contains(politics, "warnings for ZZ1 (1)") {
		t.Fatalf("politics headline: %q", politics)
	}
	if !strings.Contains(orphan, "(1; tags: XQZ)") {
		t.Fatalf("orphan headline: %q", orphan)
	}
	joined := strings.Join(artifact.Coverage, ",")
	if !strings.Contains(joined, "politics_export_lint") || !strings.Contains(joined, "state_orphan_tag_lint") {
		t.Fatalf("coverage tags missing: %v", artifact.Coverage)
	}
}
