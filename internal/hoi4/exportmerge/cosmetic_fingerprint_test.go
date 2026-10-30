package exportmerge

import (
	"haganeworks/internal/hoi4/exportengine"
	"testing"
)

func TestCosmeticAppendFingerprintIncludesOwnership(t *testing.T) {
	entry := exportengine.Entry{Path: "Demo/common/countries/cosmetic.txt", Bytes: []byte("A = {} B = {}")}
	full := appendFingerprint(Archive{}, []exportengine.Entry{entry})
	entry.CosmeticColorTags = []string{"A", "B"}
	selected := appendFingerprint(Archive{}, []exportengine.Entry{entry})
	if selected == full {
		t.Fatal("whole-file and selected-tag merges have identical fingerprints")
	}
	entry.CosmeticColorTags = []string{"B", "A"}
	if appendFingerprint(Archive{}, []exportengine.Entry{entry}) != selected {
		t.Fatal("tag ordering changes policy fingerprint")
	}
}
