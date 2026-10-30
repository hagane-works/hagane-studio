package exportengine

import (
	"strings"
	"testing"

	"haganeworks/internal/gameindex"
)

func TestVanillaCollisionSurfacedButNeverBlocks(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"common/bookmarks/blitzkrieg.txt","content":"bookmarks = { }","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
		`{"id":2,"dest_path":"common/ideas/harmless.txt","content":"ideas = { }","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatalf("a vanilla collision must never fail the export: %v", err)
	}
	var hit string
	for _, warning := range artifact.Warnings {
		if strings.Contains(warning, "blitzkrieg") {
			hit = warning
		}
	}
	if hit == "" {
		t.Fatalf("collision not surfaced; warnings=%v", artifact.Warnings)
	}
	if !strings.Contains(hit, "replaces a base-game bookmark") {
		t.Fatalf("unexpected wording: %q", hit)
	}

	for _, banned := range []string{"ERROR", "FAIL", "INVALID", "must not"} {
		if strings.Contains(strings.ToUpper(hit), strings.ToUpper(banned)) {
			t.Fatalf("informational lint must not be worded as a blocker: %q", hit)
		}
	}

	for _, warning := range artifact.Warnings {
		if strings.Contains(warning, "harmless.txt") {
			t.Fatalf("false positive on a non-vanilla path: %q", warning)
		}
	}
	entry, ok := findEntry(artifact, "Demo/_VANILLA_COLLISION_LINT.log")
	if !ok {
		t.Fatal("no _VANILLA_COLLISION_LINT.log entry — the Python channel writes one")
	}
	text := string(entry.Bytes)
	if !strings.HasPrefix(text, "# Files that replace base-game files\n") ||
		!strings.Contains(text, "This never blocks the export.") {
		t.Fatalf("lint header drifted:\n%s", text)
	}
	if !strings.Contains(text, "common/bookmarks/blitzkrieg.txt") {
		t.Fatalf("lint log missing the collision:\n%s", text)
	}
}

func TestNoVanillaCollisionLogWhenClean(t *testing.T) {
	snapshot := rawFileSnapshot(
		`{"id":1,"dest_path":"common/ideas/harmless.txt","content":"ideas = { }","source":"created","raw_kind":"verbatim_override","is_binary":false}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findEntry(artifact, "Demo/_VANILLA_COLLISION_LINT.log"); ok {
		t.Fatal("lint log written with no collisions")
	}
}

func TestVanillaCollisionTableLoaded(t *testing.T) {
	table := loadVanillaCollisionTable()
	if !table.icons["gfx/interface/goals/goal_generic_test.dds"] || !table.icons["gfx/interface/technologies/test_infantry_weapons.dds"] {
		t.Fatalf("vanilla icon table misses the game's focus / tech icons: %d entries", len(table.icons))
	}
	if !table.bookmarks["common/bookmarks/blitzkrieg.txt"] {
		t.Fatal("expected a known vanilla bookmark path in the table")
	}
	t.Cleanup(gameindex.SetDefault(nil))
	if got := VanillaCollisions([]Entry{{Path: "Demo/common/bookmarks/blitzkrieg.txt", Bytes: []byte("x")}}, english); len(got) != 0 {
		t.Fatalf("no game folder must report nothing: %v", got)
	}
}
