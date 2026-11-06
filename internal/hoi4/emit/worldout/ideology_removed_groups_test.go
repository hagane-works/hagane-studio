package worldout

import (
	"strings"
	"testing"
)

// A mod that replaces the base game's ideology file without the base game's
// groups keeps them out of the full replacement; the other groups stay.
func TestFullIdeologiesFileLeavesOutRemovedBaseGroups(t *testing.T) {
	tables, err := testTables(t)
	if err != nil {
		t.Fatal(err)
	}
	registry := managedRegistry()
	registry.Set("removed_base_groups", []Value{"democratic", "fascism", "not_a_base_group"})

	full := generateFullIdeologiesFile(registry, tables.IdeologyBaseText)
	if full == "" {
		t.Fatal("no full replacement")
	}
	for root, want := range map[string]int{"democratic": 0, "fascism": 0, "communism": 1, "neutrality": 1, "technocracy": 1} {
		if got := strings.Count(full, "\t"+root+" = {"); got != want {
			t.Errorf("ROOT %q definitions = %d, want %d", root, got, want)
		}
	}
	if strings.Count(full, "{") != strings.Count(full, "}") {
		t.Fatalf("unbalanced braces:\n%s", full)
	}

	// A subtype under a removed group has nowhere to go.
	sub := NewOMap()
	sub.Set("id", "lost_subtype")
	sub.Set("group", "democratic")
	registry.Set("sub_ideologies", []Value{sub})
	if got := generateFullIdeologiesFile(registry, tables.IdeologyBaseText); got != "" {
		t.Fatalf("a subtype of a removed group produced a file:\n%s", got)
	}
}
