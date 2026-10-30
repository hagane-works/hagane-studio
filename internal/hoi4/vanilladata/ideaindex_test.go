package vanilladata

import (
	"slices"
	"strings"
	"testing"

	"haganeworks/internal/gameindex"
)

func TestIsVanillaIdeaPictureAcceptsDeclaredSprites(t *testing.T) {
	useFixture(t)
	if got := VanillaIdeaPictureCount(); got != 7 {
		t.Fatalf("legal set = %d, want 6 sprites + 1 orphan", got)
	}
	for _, name := range []string{
		"GFX_idea_generic_fascism_drift_2", "generic_fascism_drift_2",
		"  GFX_idea_generic_fascism_drift_2  ",
		"new_economic_policy",
		"war_economy", "civilian_economy", "BEL_rexist_legion",
		"GFX_idea_war_economy",
		"idea_PRC_sinochem",
	} {
		if !IsVanillaIdeaPicture(name) {
			t.Errorf("%q must be a vanilla idea picture", name)
		}
	}
	for _, name := range []string{
		"", "GFX_idea_", "GFX_idea_totally_made_up_2026",
		"gfx_idea_generic_fascism_drift_2",
		"zzz_not_a_sprite_xyz",
	} {
		if IsVanillaIdeaPicture(name) {
			t.Errorf("%q must NOT be a vanilla idea picture (case-sensitive, non-empty)", name)
		}
	}
}

func TestIdeaPicturesWithoutGame(t *testing.T) {
	t.Cleanup(gameindex.SetDefault(nil))
	if IsVanillaIdeaPicture("war_economy") || VanillaIdeaPictureCount() != 0 || SuggestIdeaPictures("war_ecnomy", 3) != nil {
		t.Fatal("no game folder must mean an empty set")
	}
}

func TestSuggestIdeaPictures(t *testing.T) {
	useFixture(t)

	got := SuggestIdeaPictures("war_ecnomy", 3)
	if !slices.Contains(got, "war_economy") {
		t.Errorf("SuggestIdeaPictures(\"war_ecnomy\", 3) = %v · must offer war_economy", got)
	}
	if len(got) > 3 {
		t.Errorf("n=3 honoured? got %d results: %v", len(got), got)
	}

	got = SuggestIdeaPictures("GFX_idea_war_ecnomy", 5)
	if !slices.Contains(got, "war_economy") {
		t.Errorf("prefixed query lost the suggestion: %v", got)
	}
	for _, s := range got {
		if strings.HasPrefix(s, "GFX_idea_") {
			t.Errorf("suggestion %q must be bare (the form an author types)", s)
		}
		if !IsVanillaIdeaPicture(s) {
			t.Errorf("suggestion %q is not itself a legal picture", s)
		}
	}

	if got = SuggestIdeaPictures("war_economy", 3); len(got) == 0 || got[0] != "war_economy" {
		t.Errorf("exact name must rank first, got %v", got)
	}

	if a, b := SuggestIdeaPictures("civilian_ecnomy", 5), SuggestIdeaPictures("civilian_ecnomy", 5); !slices.Equal(a, b) {
		t.Errorf("non-deterministic suggestions: %v vs %v", a, b)
	}

	for _, tc := range []struct {
		name string
		n    int
	}{
		{"", 3}, {"   ", 3}, {"GFX_idea_", 3}, {"war_economy", 0}, {"war_economy", -1},
	} {
		if out := SuggestIdeaPictures(tc.name, tc.n); len(out) != 0 {
			t.Errorf("SuggestIdeaPictures(%q, %d) = %v · want empty", tc.name, tc.n, out)
		}
	}
}
