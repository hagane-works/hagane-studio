package exportengine

import (
	"strings"
	"testing"
)

func TestDuplicatePrivateAssetsCatchesTwoEntitiesOnePicture(t *testing.T) {
	shared := []byte("\x89PNG\r\n\x1a\n-one-portrait")
	warnings := DuplicatePrivateAssets([]Entry{
		{Path: "29742/gfx/leaders/AHM/portrait_AHM_Cerydra.png", Bytes: shared,
			Origin: "private_asset", SourceEntity: "character_portrait:425356"},
		{Path: "29742/gfx/leaders/AHM/portrait_AHM_Phainon.png", Bytes: shared,
			Origin: "private_asset", SourceEntity: "character_portrait:425382"},
	}, english)
	if len(warnings) != 1 {
		t.Fatalf("two entities sharing one picture must warn exactly once, got %d: %v",
			len(warnings), warnings)
	}
	for _, want := range []string{"portrait_AHM_Cerydra.png", "portrait_AHM_Phainon.png",
		"character_portrait:425356", "character_portrait:425382"} {
		if !strings.Contains(warnings[0], want) {
			t.Fatalf("warning must name both artifacts and both entities, missing %q: %s",
				want, warnings[0])
		}
	}
}

func TestDuplicatePrivateAssetsIgnoresSameEntityTwice(t *testing.T) {
	shared := []byte("same-entity-bytes")
	warnings := DuplicatePrivateAssets([]Entry{
		{Path: "m/gfx/leaders/AHM/portrait_AHM_Phainon.png", Bytes: shared,
			Origin: "private_asset", SourceEntity: "character_portrait:425382"},
		{Path: "m/gfx/leaders/AHM/army_AHM_Phainon.png", Bytes: shared,
			Origin: "private_asset", SourceEntity: "character_portrait:425382"},
	}, english)
	if len(warnings) != 0 {
		t.Fatalf("one entity written to two paths must stay silent, got %v", warnings)
	}
}

func TestDuplicatePrivateAssetsIgnoresGeneratedAndUnattributed(t *testing.T) {
	same := []byte("characters = {\n}\n")
	warnings := DuplicatePrivateAssets([]Entry{
		{Path: "m/common/characters/A.txt", Bytes: same, Origin: "generated", SourceEntity: "characters"},
		{Path: "m/common/characters/B.txt", Bytes: same, Origin: "generated", SourceEntity: "characters"},

		{Path: "m/gfx/x.png", Bytes: []byte("blob"), Origin: "private_asset"},
		{Path: "m/gfx/y.png", Bytes: []byte("blob"), Origin: "private_asset"},
	}, english)
	if len(warnings) != 0 {
		t.Fatalf("generated/unattributed entries must not warn, got %v", warnings)
	}
}
