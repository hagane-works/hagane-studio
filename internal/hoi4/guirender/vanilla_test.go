package guirender

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func vanillaPack(t *testing.T) map[string][]byte {
	t.Helper()
	manifest, err := json.Marshal(VanillaManifest{
		GameVersion: "1.19.3",
		Textures: map[string][]int{
			"gfx/interface/topbar/flag.dds":        {64, 48},
			"gfx/interface/goals/goal_generic.dds": {94, 76},
			"gfx/interface/bad_entry.dds":          {0, 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{
		"interface/topbar.gui": []byte(`guiTypes={containerWindowType={name=topbar size={width=1920 height=100} ` +
			`iconType={name=flag quadTextureSprite=GFX_player_flag} ` +
			`instantTextBoxType={name=date text=DATE_LABEL font=hoi_16mbs} ` +
			`iconType={name=goal quadTextureSprite=GFX_goal_generic}}}`),
		"interface/core.gfx":                      []byte(`spriteTypes={spriteType={name=GFX_player_flag texturefile="gfx/interface/topbar/flag.dds"}}`),
		"localisation/english/core_l_english.yml": []byte("l_english:\n DATE_LABEL:0 \"1 Sep 1939\"\n"),
		"gfx/fonts/hoi_16mbs.fnt":                 []byte("info face=\"hoi\" size=16 outline=1\ncommon lineHeight=20 base=13\n"),
		VanillaManifestPath:                       manifest,
	}
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{1, 2, 3, 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func layered(t *testing.T, project map[string][]byte) *Index {
	t.Helper()
	base, err := NewVanillaIndex(vanillaPack(t), "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(project, base)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestVanillaWindowsVisibleToProjectWithoutInterface(t *testing.T) {
	idx := layered(t, map[string][]byte{})

	rows := idx.ListWindows()
	if len(rows) != 1 || rows[0]["name"] != "topbar" || rows[0]["source"] != "vanilla" {
		t.Fatalf("window list: %#v", rows)
	}
	window, ok := idx.Window("topbar")
	if !ok {
		t.Fatal("vanilla window not resolvable")
	}
	if window["source"] != "vanilla" || window["file"] != "topbar.gui" {
		t.Fatalf("window labels: %#v", window["source"])
	}

	text := window["texts"].(map[string]any)["DATE_LABEL"].(map[string]any)
	if text["display"] != "1 Sep 1939" || text["kind"] != "loc" {
		t.Fatalf("vanilla loc not resolved: %#v", text)
	}
	font := window["fonts"].(map[string]any)["hoi_16mbs"].(map[string]any)
	if font["outline"] != true || font["line_ratio"] != 1.25 {
		t.Fatalf("vanilla font metadata: %#v", font)
	}
}

func TestManifestSuppliesSizeForTexturesKeptInObjectStorage(t *testing.T) {
	idx := layered(t, map[string][]byte{})
	window, _ := idx.Window("topbar")
	sprites := window["sprites"].(map[string]any)

	declared := sprites["GFX_player_flag"].(map[string]any)
	if declared["width"] != 64 || declared["height"] != 48 {
		t.Fatalf("declared sprite size: %#v", declared)
	}
	if declared["resolved"] != true || declared["vanilla"] != true || declared["missing_file"] != false {
		t.Fatalf("declared sprite flags: %#v", declared)
	}

	loose := sprites["GFX_goal_generic"].(map[string]any)
	if loose["width"] != 94 || loose["height"] != 76 || loose["vanilla"] != true {
		t.Fatalf("loose sprite: %#v", loose)
	}

	if declared["vanilla_url"] != "https://obs.example.com/vanilla/gfx/interface/topbar/flag.png" {
		t.Fatalf("declared sprite URL: %#v", declared["vanilla_url"])
	}
	if loose["vanilla_url"] != "https://obs.example.com/vanilla/gfx/interface/goals/goal_generic.png" {
		t.Fatalf("loose sprite URL: %#v", loose["vanilla_url"])
	}
}

func TestMissingBaseURLStillSizesVanillaSprites(t *testing.T) {
	base, err := NewVanillaIndex(vanillaPack(t), "")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndexWithFallback(map[string][]byte{}, base)
	if err != nil {
		t.Fatal(err)
	}
	window, _ := idx.Window("topbar")
	sprite := window["sprites"].(map[string]any)["GFX_player_flag"].(map[string]any)
	if sprite["width"] != 64 || sprite["vanilla"] != true {
		t.Fatalf("sizing lost without a base URL: %#v", sprite)
	}
	if _, present := sprite["vanilla_url"]; present {
		t.Fatalf("no base URL configured, yet a URL was emitted: %#v", sprite)
	}
	if got := idx.VanillaSpriteURL("GFX_player_flag"); got != "" {
		t.Fatalf("expected no URL, got %s", got)
	}
}

func TestProjectTextureOverridesVanillaTexture(t *testing.T) {
	idx := layered(t, map[string][]byte{

		"gfx/interface/topbar/flag.png": pngBytes(t, 7, 5),
	})
	window, _ := idx.Window("topbar")
	sprite := window["sprites"].(map[string]any)["GFX_player_flag"].(map[string]any)
	if sprite["width"] != 7 || sprite["height"] != 5 {
		t.Fatalf("project texture did not win: %#v", sprite)
	}
	if _, claimed := sprite["vanilla"]; claimed {
		t.Fatalf("locally served texture must not be flagged vanilla: %#v", sprite)
	}
	if _, external := idx.ExternalSpriteTexture("GFX_player_flag"); external {
		t.Fatal("sprite with local bytes must not be redirected to object storage")
	}
}

func TestModWindowShadowsVanillaWindow(t *testing.T) {
	idx := layered(t, map[string][]byte{
		"interface/topbar.gui": []byte(`guiTypes={containerWindowType={name=topbar size={width=800 height=60}}}`),
	})
	rows := idx.ListWindows()
	if len(rows) != 1 {
		t.Fatalf("shadowed window listed twice: %#v", rows)
	}
	if rows[0]["source"] != "mod" {
		t.Fatalf("mod window must win: %#v", rows[0])
	}
	window, _ := idx.Window("topbar")
	node := window["node"].(map[string]any)
	if node["size"].(map[string]any)["width"] != 800 {
		t.Fatalf("vanilla node served instead of mod node: %#v", node["size"])
	}
}

func TestExternalSpriteTextureAndURL(t *testing.T) {
	idx := layered(t, map[string][]byte{})

	rel, ok := idx.ExternalSpriteTexture("GFX_player_flag")
	if !ok || rel != "gfx/interface/topbar/flag.dds" {
		t.Fatalf("external texture: %q %v", rel, ok)
	}

	if got := VanillaTextureURL("https://obs.example.com/vanilla/", rel); got != "https://obs.example.com/vanilla/gfx/interface/topbar/flag.png" {
		t.Fatalf("URL: %s", got)
	}

	if got := VanillaTextureURL("", rel); got != "" {
		t.Fatalf("expected empty URL, got %s", got)
	}
	if _, ok := idx.ExternalSpriteTexture("GFX_does_not_exist"); ok {
		t.Fatal("unknown sprite must not resolve")
	}
}

func TestTexticonSpritesResolveThroughTheManifest(t *testing.T) {
	pack := vanillaPack(t)
	pack["interface/texticons.gfx"] = []byte(`spriteTypes={spriteType={name=GFX_organization_texticon texturefile="gfx/texticons/organization_texticon.dds"}}`)
	bare, err := NewVanillaIndex(pack, "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	if got := bare.spritePayload("GFX_organization_texticon"); got["missing_file"] != true {
		t.Fatalf("unlisted texticon = %v, want missing_file", got)
	}

	manifest, err := json.Marshal(VanillaManifest{GameVersion: "1.19.3", Textures: map[string][]int{
		"gfx/texticons/organization_texticon.dds": {14, 14},
	}})
	if err != nil {
		t.Fatal(err)
	}
	pack[VanillaManifestPath] = manifest
	listed, err := NewVanillaIndex(pack, "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	got := listed.spritePayload("GFX_organization_texticon")
	if got["missing_file"] != false || got["width"] != 14 ||
		got["vanilla_url"] != "https://obs.example.com/vanilla/gfx/texticons/organization_texticon.png" {
		t.Errorf("listed texticon = %v", got)
	}
}

func TestManifestIgnoresUnusableSizes(t *testing.T) {
	base, err := NewVanillaIndex(vanillaPack(t), "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := base.external["gfx/interface/bad_entry.dds"]; ok {
		t.Fatal("zero-sized manifest entry was kept")
	}
	if base.GameVersion() != "1.19.3" {
		t.Fatalf("game version: %q", base.GameVersion())
	}
}

func TestVanillaPackWithoutManifestIsRejected(t *testing.T) {
	if _, err := NewVanillaIndex(map[string][]byte{"interface/topbar.gui": []byte("guiTypes={}")}, ""); err == nil {
		t.Fatal("expected an error for a pack with no texture manifest")
	}
}

func TestNilFallbackMatchesPlainIndex(t *testing.T) {
	files := map[string][]byte{
		"interface/mod.gui": []byte(`guiTypes={containerWindowType={name=only_mine size={width=10 height=10}}}`),
	}
	idx, err := NewIndexWithFallback(files, nil)
	if err != nil {
		t.Fatal(err)
	}
	if idx.WindowCount() != 1 {
		t.Fatalf("window count %d", idx.WindowCount())
	}
	if _, ok := idx.Window("topbar"); ok {
		t.Fatal("no fallback configured, yet a vanilla window resolved")
	}
	rows := idx.ListWindows()
	if len(rows) != 1 || rows[0]["source"] != "mod" {
		t.Fatalf("rows: %#v", rows)
	}
}
