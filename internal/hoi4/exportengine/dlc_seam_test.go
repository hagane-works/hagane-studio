package exportengine

import (
	"strings"
	"testing"
)

func TestDLCConditionalSeamNativeShadowShipsLogAndWarning(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"history/states/1-X.txt","content":"state = {\n\tid = 1\n\thistory = {\n\t\towner = HAN\n\t\tif = {\n\t\t\tlimit = { has_dlc = \"La Resistance\" }\n\t\t\ttransfer_state_to = COG\n\t\t}\n\t}\n}\n","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("a DLC-conditional report must never fail the export: %v", err)
	}
	root := artifactRoot(artifact)
	logEntry, ok := findEntry(artifact, root+"/"+DLCConditionalLogName)
	if !ok {
		t.Fatalf("%s not shipped in the artifact (seam hunk in native.go not landed?) entries=%d", DLCConditionalLogName, len(artifact.Entries))
	}
	if !strings.Contains(string(logEntry.Bytes), `history/states/1-X.txt:6  has_dlc "La Resistance"  if  -> transfer_state_to`) {
		t.Fatalf("log must carry the guarded-effect line:\n%s", logEntry.Bytes)
	}
	var hit string
	for _, w := range artifact.Warnings {
		if strings.Contains(w, "Blocks that need a DLC") {
			hit = w
		}
	}
	if hit == "" {
		t.Fatalf("headline not pushed to artifact.Warnings (export page consumer); warnings=%v", artifact.Warnings)
	}
	want := "Blocks that need a DLC (has_dlc): 1; DLCs involved: 1 (La Resistance). " +
		"They run only for players who own that DLC — check _DLC_CONDITIONAL_LINT.log (history/states: 1)."
	if hit != want {
		t.Fatalf("unexpected headline:\n got %q\nwant %q", hit, want)
	}
	found := false
	for _, c := range artifact.Coverage {
		if c == "dlc_conditional_lint" {
			found = true
		}
	}
	if !found {
		t.Fatalf("coverage tag dlc_conditional_lint missing: %v", artifact.Coverage)
	}
}
