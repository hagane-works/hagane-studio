package workbench

import (
	"strings"
	"testing"

	"haganeworks/internal/reftables"
)

func TestRegistryValidationWithAnEmptyIdeologyTable(t *testing.T) {
	registry := map[string]any{
		"root_definitions": []any{map[string]any{"id": "technocracy", "body": "types = { }\ncolor = { 1 2 3 }"}},
		"sub_ideologies": []any{
			map[string]any{"id": "rexism", "group": "fascism"},
			map[string]any{"id": "neo_fascism", "group": "fascism"},
			map[string]any{"id": "cybernetics", "group": "technocracy"},
		},
	}
	if err := ValidateRegistry(registry, nil); err != nil {
		t.Fatalf("with the built-in table: %v", err)
	}
	if reason := ValidateRoot("democratic", "types = { }\ncolor = { 1 2 3 }"); reason == "" {
		t.Fatal("a base-game ROOT was accepted as the project's own")
	}

	defer reftables.Use(map[string][]byte{reftables.IdeologiesFile: []byte(`{"groups": [], "sub_ideologies": []}`)})()
	err := ValidateRegistry(registry, nil)
	if err == nil || !strings.Contains(err.Error(), "rexism (parent=fascism)") || !strings.Contains(err.Error(), "neo_fascism (parent=fascism)") ||
		strings.Contains(err.Error(), "cybernetics") {
		t.Fatalf("with an empty table: %v", err)
	}
	if reason := ValidateRoot("democratic", "types = { }\ncolor = { 1 2 3 }"); reason != "" {
		t.Fatalf("with an empty table a ROOT named democratic is refused: %s", reason)
	}
}
