package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func animatedPortraitSnapshot(largePortrait, smallPortrait string, withAssets bool) Snapshot {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["english"]}`),
		Tables: map[string][]json.RawMessage{
			"characters": {
				json.RawMessage(`{"id":11,"tag":"GER_leader","country_tag":"GER","name":"Leader","source":"created","replaces_vanilla_id":false,` +
					`"large_portrait":"` + largePortrait + `","small_portrait":"` + smallPortrait + `"}`),
			},
		},
	}
	if withAssets {
		snapshot.Tables["project_obs_assets"] = []json.RawMessage{
			json.RawMessage(`{"asset_type":"animated_portrait","gfx_name":"GFX_portrait_GER_hero_large","obs_key":"p/large.dds"}`),
			json.RawMessage(`{"asset_type":"animated_portrait_small","gfx_name":"GFX_portrait_GER_hero_large","obs_key":"p/small.dds"}`),
			json.RawMessage(`{"asset_type":"animated_portrait_gfx","gfx_name":"GFX_portrait_GER_hero_large","obs_key":"p/sprite.gfx"}`),
		}
	}
	return snapshot
}

func animatedPortraitBlobs(t *testing.T) BlobResolver {
	t.Helper()
	gfxText := BuildAnimatedPortraitGFX("GFX_portrait_GER_hero_large", "GFX_portrait_GER_hero_small", 8, 8)
	blobs := map[string][]byte{
		"p/large.dds":  []byte("DDS large-sheet-bytes"),
		"p/small.dds":  []byte("DDS small-bytes"),
		"p/sprite.gfx": []byte(gfxText),
	}
	return func(key string, _ int64) ([]byte, error) {
		if data, ok := blobs[key]; ok {
			return data, nil
		}
		return nil, errBlobMissing
	}
}

func TestNativeShadowKeepsGFXPortraitReference(t *testing.T) {
	snapshot := animatedPortraitSnapshot("GFX_portrait_GER_hero_large", "GFX_portrait_GER_hero_small", false)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return nil, errBlobMissing })
	if err != nil {
		t.Fatal(err)
	}
	script := ""
	for _, entry := range artifact.Entries {
		if strings.Contains(entry.Path, "/common/characters/") {
			script = string(entry.Bytes)
		}
	}
	if script == "" {
		t.Fatal("character script missing")
	}
	if !strings.Contains(script, "GFX_portrait_GER_hero_large") {
		t.Fatalf("GFX portrait reference was dropped from the portraits block:\n%s", script)
	}
}

func TestAnimatedPortraitsShipSheetAdvisorAndGFX(t *testing.T) {
	snapshot := animatedPortraitSnapshot("GFX_portrait_GER_hero_large", "GFX_portrait_GER_hero_small", true)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, animatedPortraitBlobs(t))
	if err != nil {
		t.Fatal(err)
	}
	gfxEntry, ok := findEntry(artifact, "Demo/"+AnimatedPortraitGFXPath("GFX_portrait_GER_hero_large"))
	if !ok {
		var paths []string
		for _, e := range artifact.Entries {
			paths = append(paths, e.Path)
		}
		t.Fatalf("animated portrait gfx missing; entries=%v", paths)
	}
	text := string(gfxEntry.Bytes)
	if !strings.Contains(text, "frameAnimatedSpriteType") || !strings.Contains(text, "noOfFrames = 8") {
		t.Fatalf("gfx text not shipped verbatim:\n%s", text)
	}

	large, ok := findEntry(artifact, "Demo/"+AnimatedPortraitLargePath("GFX_portrait_GER_hero_large"))
	if !ok || !strings.Contains(text, AnimatedPortraitLargePath("GFX_portrait_GER_hero_large")) {
		t.Fatal("frame-sheet DDS missing or not referenced by the gfx text")
	}
	if string(large.Bytes) != "DDS large-sheet-bytes" {
		t.Fatal("frame-sheet bytes altered")
	}
	if _, ok = findEntry(artifact, "Demo/"+AnimatedPortraitSmallPath("GFX_portrait_GER_hero_large")); !ok {
		t.Fatal("advisor DDS missing")
	}
	coverage := strings.Join(artifact.Coverage, ",")
	if !strings.Contains(coverage, "animated_portraits") {
		t.Fatalf("coverage missing animated_portraits: %s", coverage)
	}
}

func TestUnreferencedAnimatedPortraitStaysOut(t *testing.T) {

	snapshot := animatedPortraitSnapshot("portrait_GER_leader.png", "", true)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, animatedPortraitBlobs(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findEntry(artifact, "Demo/"+AnimatedPortraitGFXPath("GFX_portrait_GER_hero_large")); ok {
		t.Fatal("unreferenced animated portrait must not ship")
	}
}

func TestUnfetchableAnimatedPortraitSkipsOnlyThatSprite(t *testing.T) {
	snapshot := animatedPortraitSnapshot("GFX_portrait_GER_hero_large", "GFX_portrait_GER_hero_small", true)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(string, int64) ([]byte, error) { return nil, errBlobMissing })
	if err != nil {
		t.Fatalf("one unfetchable animated portrait must not fail the export: %v", err)
	}
	if _, ok := findEntry(artifact, "Demo/"+AnimatedPortraitGFXPath("GFX_portrait_GER_hero_large")); ok {
		t.Fatal("sprite with missing blobs must be skipped entirely")
	}
	if !warnsAbout(artifact, "animated portrait") {
		t.Fatalf("no warning recorded; warnings=%v", artifact.Warnings)
	}
}
