package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func TestRealGameBritainsScientists(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.ScientistsPayload("ENG", "1936.1.1.12")
	got := []string{}
	for _, row := range out["scientists"].([]map[string]any) {
		skills := []string{}
		for _, skill := range row["skills"].([]map[string]any) {
			skills = append(skills, fmt.Sprintf("%v %v", skill["label"], skill["level"]))
		}
		traits := []string{}
		for _, trait := range row["traits"].([]map[string]any) {
			traits = append(traits, trait["icon"].(string))
		}
		got = append(got, fmt.Sprintf("%v [%s] %s", row["name"], strings.Join(skills, ","), strings.Join(traits, ",")))
	}
	want := []string{
		"James Chadwick [Physics 2] GFX_scientist_trait_nuclear_warfare_expert",
		"Frank Whittle [Air 2] GFX_scientist_trait_jet_expert,GFX_scientist_trait_bright",
		"John Cockcroft [Physics 2]",
		"Charles Goodeve [Naval 2] GFX_scientist_trait_surface_ship_specialist",
	}
	if len(got) != 6 {
		t.Fatalf("scientists = %v", got)
	}
	for i, line := range want {
		if !strings.HasPrefix(got[i], line) {
			t.Errorf("row %d = %q, want %q", i, got[i], line)
		}
	}
	if out["title"] != "Scientists" {
		t.Errorf("title = %v", out["title"])
	}
}
