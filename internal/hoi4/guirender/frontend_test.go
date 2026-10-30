package guirender

import "testing"

func TestFrontendPictures(t *testing.T) {
	manifest := []byte(`{"game_version":"1.19.3","textures":{
		"gfx/loadingscreens/load_1.dds":[1920,1440],
		"gfx/loadingscreens/load_2.dds":[1920,1440],
		"gfx/loadingscreens/load_taog_ast.dds":[1920,1440],
		"gfx/loadingscreens/load_bba.dds":[1920,1440]}}`)
	base, err := NewVanillaIndex(map[string][]byte{
		"common/frontend/backgrounds/base_backgrounds.txt": []byte(`
			load_taog_ast = { dlc_allowed = "Thunder at Our Gates" }
			load_missing = { dlc_allowed = "Nothing" }
			load_bba = { dlc_allowed = "By Blood Alone" locale = "simp_chinese" }
		`),
		VanillaManifestPath: manifest,
	}, "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	backgrounds := base.FrontendBackgrounds()
	if len(backgrounds) != 2 || backgrounds[0].Name != "load_taog_ast" || backgrounds[1].Name != "load_bba" {
		t.Fatalf("backgrounds, in file order, those with a picture: %+v", backgrounds)
	}
	if backgrounds[1].URL != "https://obs.example.com/vanilla/gfx/loadingscreens/load_bba.png" || backgrounds[1].Width != 1920 || backgrounds[1].Height != 1440 {
		t.Fatalf("a background's picture: %+v", backgrounds[1])
	}
	if backgrounds[0].Locale != "" || backgrounds[1].Locale != "simp_chinese" {
		t.Fatalf("locales: %q %q", backgrounds[0].Locale, backgrounds[1].Locale)
	}
	loading := base.LoadingPictures()
	names := []string{}
	for _, picture := range loading {
		names = append(names, picture.Name)
	}
	if len(names) != 4 || names[0] != "load_1" || names[1] != "load_2" || names[2] != "load_taog_ast" || names[3] != "load_bba" {
		t.Fatalf("loading pictures = %v", names)
	}

	if file, ok := base.ScreenPictureFile("GFX_load_taog_ast"); !ok || file != "gfx/loadingscreens/load_taog_ast.dds" {
		t.Fatalf("picture file = %q %v", file, ok)
	}
	for _, name := range []string{"GFX_load_missing", "load_1", "GFX_../load_1", "GFX_"} {
		if _, ok := base.ScreenPictureFile(name); ok {
			t.Fatalf("%s is not a picture", name)
		}
	}
	if url := base.VanillaSpriteURL("GFX_load_1"); url != "https://obs.example.com/vanilla/gfx/loadingscreens/load_1.png" {
		t.Fatalf("the picture by its sprite name: %q", url)
	}

	base.SetSpriteOverrides(map[string]SpriteOverride{"GFX_load_taog_ast": {Width: 800, Height: 600, URL: "/api/go/projects/2/gui-render/sprite?name=GFX_load_taog_ast&v=1"}})
	replaced := base.FrontendBackgrounds()[0]
	if !replaced.Overridden || replaced.URL != "/api/go/projects/2/gui-render/sprite?name=GFX_load_taog_ast&v=1" || replaced.Width != 800 {
		t.Fatalf("replaced background: %+v", replaced)
	}
	if payload := base.SpritesPayload([]string{"GFX_load_taog_ast"})["GFX_load_taog_ast"].(map[string]any); payload["overridden"] != true || payload["width"] != 800 {
		t.Fatalf("the replaced picture as a sprite: %v", payload)
	}
}

func TestVanillaTextureURLHook(t *testing.T) {
	manifest := []byte(`{"game_version":"1.19.3","textures":{
		"gfx/loadingscreens/load_1.dds":[1920,1440],
		"gfx/loadingscreens/load_2.dds":[1920,1440]}}`)
	base, err := NewVanillaIndex(map[string][]byte{VanillaManifestPath: manifest}, "/api/go/gui-vanilla")
	if err != nil {
		t.Fatal(err)
	}
	var asked []string
	base.SetVanillaTextureURL(func(rel string) string {
		asked = append(asked, rel)
		if rel == "gfx/loadingscreens/load_2.png" {
			return ""
		}
		return "https://assets.example.test/vanilla/" + rel + "?Signature=s"
	})
	if url := base.VanillaSpriteURL("GFX_load_1"); url != "https://assets.example.test/vanilla/gfx/loadingscreens/load_1.png?Signature=s" {
		t.Fatalf("signed address: %q", url)
	}
	if url := base.VanillaSpriteURL("GFX_load_2"); url != "/api/go/gui-vanilla/gfx/loadingscreens/load_2.png" {
		t.Fatalf("an unsigned texture falls back to the proxy: %q", url)
	}
	if len(asked) != 2 || asked[0] != "gfx/loadingscreens/load_1.png" {
		t.Fatalf("the hook gets the normalized .png path: %q", asked)
	}

	project, err := NewIndexWithFallback(map[string][]byte{}, base)
	if err != nil {
		t.Fatal(err)
	}
	for name, idx := range map[string]*Index{"project": project, "russian": project.InLanguage("russian", nil)} {
		if url := idx.VanillaSpriteURL("GFX_load_1"); url != "https://assets.example.test/vanilla/gfx/loadingscreens/load_1.png?Signature=s" {
			t.Fatalf("through the %s layer: %q", name, url)
		}
	}
}
