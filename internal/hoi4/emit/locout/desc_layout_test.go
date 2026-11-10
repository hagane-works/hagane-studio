package locout

import (
	"strings"
	"testing"
)

// Blank lines around a description are layout (a banner picture sits below
// them); the generated line must keep them.
func TestIdeaDescriptionKeepsItsLeadingAndTrailingLineBreaks(t *testing.T) {
	desc := locObj("en", "\n\n£GFX_banner\n\nThe economy is failing.\n")
	idea := locObj("idea_id", "MOD_crisis", "name_en", "Crisis", "description_locs", Value{Kind: KObj, Obj: desc})
	out := GenerateIdeasLocalisation([]*Obj{idea}, "en")
	if want := `MOD_crisis_desc: "\n\n£GFX_banner\n\nThe economy is failing.\n"`; !strings.Contains(out, want) {
		t.Fatalf("want %s in:\n%s", want, out)
	}
}
