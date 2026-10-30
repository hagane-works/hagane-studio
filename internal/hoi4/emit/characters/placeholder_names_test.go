package characters

import "testing"

func TestPlaceholderDisplayNamesMatchPython(t *testing.T) {
	want := []string{"new character", "new focus", "new idea", "new technology"}
	if len(placeholderDisplayNames) != len(want) {
		t.Fatalf("placeholderDisplayNames has %d entries, want %d: %v", len(placeholderDisplayNames), len(want), placeholderDisplayNames)
	}
	for _, name := range want {
		if !placeholderDisplayNames[name] {
			t.Fatalf("placeholder set lacks %q", name)
		}
	}
	for _, text := range []string{"New Character", "new character", "  NEW CHARACTER  "} {
		if !isPlaceholderDisplayName(text, "") {
			t.Fatalf("%q must be a placeholder display name", text)
		}
	}
	if isPlaceholderDisplayName("Erwin Rommel", "") {
		t.Fatal("a real name must not be a placeholder")
	}
}
