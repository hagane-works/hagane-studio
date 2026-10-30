package exportengine

import (
	"strings"
	"testing"
)

func TestLocFallbackSeamNativeShadowShipsLogAndWarning(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"localisation/english/seam_l_english.yml","content":"l_english:\n seam_focus: \"Rally the Army\"\n seam_num: \"12\"\n","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
		`{"id":2,"dest_path":"localisation/german/seam_l_german.yml","content":"l_german:\n seam_focus: \"Rally the Army\"\n seam_num: \"12\"\n","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("an English-fallback report must never fail the export: %v", err)
	}
	root := artifactRoot(artifact)
	logEntry, ok := findEntry(artifact, root+"/"+LocFallbackLogName)
	if !ok {
		t.Fatalf("%s not shipped in the artifact (native.go no longer runs ScanLocFallback?) entries=%d", LocFallbackLogName, len(artifact.Entries))
	}
	if !strings.Contains(string(logEntry.Bytes), "seam_focus") || strings.Contains(string(logEntry.Bytes), "seam_num") {
		t.Fatalf("log must list seam_focus and not seam_num:\n%s", logEntry.Bytes)
	}
	var hit string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "still show the English text") {
			hit = w
		}
	}
	if hit == "" {
		t.Fatalf("headline not pushed to artifact.Warnings (export page consumer); warnings=%v", artifact.Warnings)
	}
	if hit != "Untranslated text: 1 of 2 keys (50.0%) still show the English text in german. Translate them to clear the list in _LOC_EN_FALLBACK.log." {
		t.Fatalf("unexpected headline: %q", hit)
	}
	found := false
	for _, c := range artifact.Coverage {
		if c == "loc_fallback_lint" {
			found = true
		}
	}
	if !found {
		t.Fatalf("coverage tag loc_fallback_lint missing: %v", artifact.Coverage)
	}
}
