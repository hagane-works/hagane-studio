package vanilladata

import (
	"slices"
	"testing"
)

func TestTokenSetsReadTheIndex(t *testing.T) {
	useFixture(t)
	if VanillaEffectCount() != 3 || VanillaTriggerCount() != 3 || StructuralFieldCount() != 3 || VanillaYesNoKeyCount() != 3 {
		t.Fatalf("sizes %d %d %d %d", VanillaEffectCount(), VanillaTriggerCount(), StructuralFieldCount(), VanillaYesNoKeyCount())
	}
	for _, name := range []string{"instant_build", "keep_completed", "designer"} {
		if !IsVanillaYesNoKey(name) {
			t.Fatalf("%q must be a vanilla yes/no key", name)
		}
	}
	if IsVanillaYesNoKey("my_custom_effect") || IsVanillaYesNoKey("Instant_Build") {
		t.Fatal("yes/no-key membership must be exact and case-sensitive")
	}
	if !slices.Equal(VanillaYesNoKeys(), []string{"designer", "instant_build", "keep_completed"}) {
		t.Fatalf("sorted keys = %v", VanillaYesNoKeys())
	}
	for _, name := range []string{"add_ideas", "country_event", "set_country_flag"} {
		if !IsVanillaEffect(name) {
			t.Fatalf("%q must be a vanilla effect", name)
		}
	}
	for _, name := range []string{"has_idea", "has_war", "is_ai"} {
		if !IsVanillaTrigger(name) {
			t.Fatalf("%q must be a vanilla trigger", name)
		}
	}
	for _, name := range []string{"cancel_if_invalid", "is_triggered_only", "available_if_capitulated"} {
		if !IsStructuralField(name) {
			t.Fatalf("%q must be a structural field", name)
		}
	}

	if IsVanillaEffect("Add_Ideas") || IsVanillaTrigger("HAS_IDEA") || IsStructuralField("") {
		t.Fatal("membership must be exact and case-sensitive")
	}
}
