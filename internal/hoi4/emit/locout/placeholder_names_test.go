package locout

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
		if !IsPlaceholderDisplayName(text, Value{}) {
			t.Fatalf("%q must be a placeholder display name", text)
		}
	}
	if IsPlaceholderDisplayName("Erwin Rommel", Value{}) {
		t.Fatal("a real name must not be a placeholder")
	}
}

func TestPlaceholderDisplayNamesSortedIncludesCharacter(t *testing.T) {
	got := PlaceholderDisplayNamesSorted()
	want := []string{"new character", "new focus", "new idea", "new technology"}
	if len(got) != len(want) {
		t.Fatalf("sorted=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted[%d]=%q want %q (%v)", i, got[i], want[i], got)
		}
	}
}
