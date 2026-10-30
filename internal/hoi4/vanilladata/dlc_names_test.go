package vanilladata

import "testing"

func TestDLCNamesRosterLoads(t *testing.T) {
	useFixture(t)
	names := DLCNames()
	if len(names) != 3 || DLCNamesGameVersion() != "1.19.3" || len(DLCNamesSources()) == 0 {
		t.Fatalf("roster = %+v (version %q, sources %v)", names, DLCNamesGameVersion(), DLCNamesSources())
	}
	if !IsKnownDLCName("Poland: United and Ready") || IsKnownDLCName("Wojtek Expansion Pack") {
		t.Fatal("membership is the roster's names")
	}
}

func TestDLCNamesExactMatchAndSuggestion(t *testing.T) {
	useFixture(t)
	if IsKnownDLCName("la resistance") {
		t.Fatal("has_dlc is case-sensitive; folded match must not count as known")
	}
	if got, ok := SuggestDLCName("la  resistance"); !ok || got != "La Resistance" {
		t.Fatalf("suggestion = %q %v", got, ok)
	}
	if _, ok := SuggestDLCName("La Resistance"); ok {
		t.Fatal("exact name must not suggest itself")
	}
	if _, ok := SuggestDLCName("Wojtek Expansion Pack"); ok {
		t.Fatal("no suggestion for a name with no folded match")
	}
	if d, ok := LookupDLCName("Man the Guns"); !ok || d.Prefix != "MTG" {
		t.Fatalf("lookup = %+v %v", d, ok)
	}
}
