package guirender

import (
	"slices"
	"testing"
)

func TestWithoutVanillaFoldersUnloadsOnlyTheBaseGame(t *testing.T) {
	vanilla, err := NewVanillaIndex(map[string][]byte{
		VanillaManifestPath:                       []byte(`{"game_version":"1.19.3","textures":{}}`),
		"common/country_tags/00_countries.txt":    []byte("GER = \"countries/Germany.txt\"\nENG = \"countries/United Kingdom.txt\"\n"),
		"common/country_tags/sub/kept.txt":        []byte("ITA = \"countries/Italy.txt\"\n"),
		"common/countries/Germany.txt":            []byte("color = { 1 2 3 }\n"),
		"history/countries/GER - Germany.txt":     []byte("capital = 64\n"),
		"localisation/english/kept_l_english.yml": []byte("l_english:\n KEPT:0 \"Kept\"\n"),
	}, "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := NewIndexWithFallback(map[string][]byte{
		"common/country_tags/01_world.txt": []byte("NWB = \"countries/NWB.txt\"\n"),
	}, vanilla)
	if err != nil {
		t.Fatal(err)
	}
	before := mod.ListCountryTags()
	unloaded, err := mod.WithoutVanillaFolders([]string{"common/country_tags", "history/countries/"})
	if err != nil {
		t.Fatal(err)
	}
	tags := unloaded.ListCountryTags()
	if !slices.Contains(tags, "NWB") || slices.Contains(tags, "GER") || slices.Contains(tags, "ENG") {
		t.Errorf("tags without the base game's: %v", tags)
	}
	if !slices.Contains(tags, "ITA") {
		t.Errorf("a subfolder of the unloaded folder went too: %v", tags)
	}
	base := unloaded.Fallback()
	if base.HasOwnFile("history/countries/GER - Germany.txt") || !base.HasOwnFile("common/countries/Germany.txt") || base.GameVersion() != "1.19.3" {
		t.Error("the base game lost the wrong files, or its version")
	}
	if got := mod.ListCountryTags(); !slices.Equal(got, before) || !slices.Contains(got, "GER") {
		t.Errorf("the index asked of changed: %v", got)
	}
	again, err := mod.WithoutVanillaFolders([]string{"history/countries", "common/country_tags"})
	if err != nil || again.Fallback() != base {
		t.Errorf("the base game without the same folders built again (%v)", err)
	}
	if same, _ := mod.WithoutVanillaFolders(nil); same != mod {
		t.Error("no folders: a different index")
	}
}

func TestWithoutVanillaFoldersRemovesManifestOnlyTextures(t *testing.T) {
	base, err := NewVanillaIndex(vanillaPack(t), "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	unloaded, err := base.WithoutVanillaFolders([]string{"GFX/interface/goals/"})
	if err != nil {
		t.Fatal(err)
	}
	if unloaded == base {
		t.Fatal("manifest-only removal returned the unchanged vanilla index")
	}
	if _, ok := unloaded.ExternalSpriteTexture("GFX_goal_generic"); ok {
		t.Fatal("replace_path left a vanilla loose texture addressable")
	}
	if _, ok := unloaded.externalLoose["goal_generic"]; ok {
		t.Fatal("removed vanilla texture survived in externalLoose")
	}
	if rel, ok := unloaded.ExternalSpriteTexture("GFX_player_flag"); !ok || rel != "gfx/interface/topbar/flag.dds" {
		t.Fatalf("unrelated vanilla texture was lost: %q %v", rel, ok)
	}
	if _, ok := base.ExternalSpriteTexture("GFX_goal_generic"); !ok {
		t.Fatal("replace_path mutated the original vanilla manifest")
	}
}
