package guirender

import (
	"encoding/json"
	"testing"
)

const testFnt = `info face="Orator Std HoI Medium" size=36 bold=0 italic=0 charset="" unicode=1 stretchH=100 smooth=1 aa=2 padding=3,3,3,3 spacing=1,1 outline=0
common lineHeight=36 base=28 scaleW=512 scaleH=256 pages=1 packed=0 alphaChnl=0 redChnl=4 greenChnl=4 blueChnl=4
page id=0 file="hoi_36header_0.dds"
chars count=2
char id=80   x=10    y=20    width=14    height=18    xoffset=1     yoffset=12    xadvance=15    page=0  chnl=15
char id=111  x=30    y=20    width=12    height=14    xoffset=0     yoffset=16    xadvance=13    page=0  chnl=15
kernings count=1
kerning first=80  second=111 amount=-2
`

const testFntOutlined = `info face="Ubuntu" size=16 bold=1
common lineHeight=16 base=13 scaleW=256 scaleH=256 pages=1 packed=0 alphaChnl=1 redChnl=0 greenChnl=0 blueChnl=0
page id=0 file="garamond_14_bold.dds"
char id=65   x=1     y=1     width=9     height=11    xoffset=0     yoffset=2     xadvance=10    page=0  chnl=15
`

func fontPack(t *testing.T) map[string][]byte {
	t.Helper()
	manifest, err := json.Marshal(VanillaManifest{
		GameVersion: "1.19.3",
		Textures: map[string][]int{

			"gfx/fonts/hoi_36header.dds":      {512, 256},
			"gfx/fonts/garamond_14_bold.dds":  {256, 256},
			"gfx/fonts/chinese/hoi_36_cn.dds": {2048, 2048},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{
		"interface/panel.gui": []byte(`guiTypes={containerWindowType={name=panel size={width=550 height=500} ` +
			`instantTextBoxType={name=title text=POL_TITLE font=hoi_36header maxWidth=400 maxHeight=20} ` +
			`instantTextBoxType={name=button_label text=POL_TITLE font=Main_14_black}}}`),

		"interface/core.gfx": []byte(`bitmapfonts={
			bitmapfont={ name="hoi_36header" fontfiles={ "gfx/fonts/hoi_36header" "gfx/fonts/hoi_36header_cryllic" } color=0xffffffff languages={ "l_english" "l_french" } }
			bitmapfont={ name="Main_14_black" path="gfx/fonts/garamond_14_bold" color=0xff000000 }
		}`),
		"interface/core_chinese.gfx": []byte(`bitmapfonts={
			bitmapfont={ name="hoi_36header" fontfiles={ "gfx/fonts/chinese/hoi_36_cn" } languages={ "l_simp_chinese" } }
		}`),
		"gfx/fonts/hoi_36header.fnt":           []byte(testFnt),
		"gfx/fonts/garamond_14_bold.fnt":       []byte(testFntOutlined),
		"localisation/english/x_l_english.yml": []byte("l_english:\n POL_TITLE:0 \"Political\"\n"),
		VanillaManifestPath:                    manifest,
	}
}

func bitmapOf(t *testing.T, font string) map[string]any {
	t.Helper()
	base, err := NewVanillaIndex(fontPack(t), "https://obs.example.com/vanilla")
	if err != nil {
		t.Fatal(err)
	}
	window, ok := base.Window("panel")
	if !ok {
		t.Fatal("window missing")
	}
	meta := window["fonts"].(map[string]any)[font].(map[string]any)
	bitmap, ok := meta["bitmap"].(map[string]any)
	if !ok {
		t.Fatalf("%s: no bitmap payload in %#v", font, meta)
	}
	return bitmap
}

func TestHeaderFontShipsTheGamesGlyphTable(t *testing.T) {
	bitmap := bitmapOf(t, "hoi_36header")
	if bitmap["line"] != 36 || bitmap["base"] != 28 || bitmap["atlas_w"] != 512 || bitmap["atlas_h"] != 256 {
		t.Fatalf("metrics: %#v", bitmap)
	}

	if bitmap["atlas"] != "https://obs.example.com/vanilla/gfx/fonts/hoi_36header.png" {
		t.Fatalf("atlas: %v", bitmap["atlas"])
	}
	chars := bitmap["chars"].(map[string][7]int)
	if chars["80"] != [7]int{10, 20, 14, 18, 1, 12, 15} {
		t.Fatalf("glyph P: %v", chars["80"])
	}
	if bitmap["kerning"].(map[string]int)["80,111"] != -2 {
		t.Fatalf("kerning: %#v", bitmap["kerning"])
	}

	if bitmap["tint"] != "alpha" {
		t.Fatalf("tint mode: %v", bitmap["tint"])
	}
	if _, tinted := bitmap["color"]; tinted {
		t.Fatalf("white font must not carry a tint colour: %#v", bitmap["color"])
	}
}

func TestChineseFontDefinitionDoesNotReplaceLatinGlyphs(t *testing.T) {
	bitmap := bitmapOf(t, "hoi_36header")
	if bitmap["atlas_w"] == 2048 {
		t.Fatal("the l_simp_chinese definition overrode the English font")
	}
}

func TestBlackButtonFontCarriesItsColour(t *testing.T) {
	bitmap := bitmapOf(t, "Main_14_black")
	if bitmap["color"] != "#000000" {
		t.Fatalf("colour: %v", bitmap["color"])
	}

	if bitmap["tint"] != "luminance" {
		t.Fatalf("tint mode: %v", bitmap["tint"])
	}
}

func TestFontOpacityIsTheColoursAlpha(t *testing.T) {

	if got := fontOpacity("0x44ffffff"); got < 0.266 || got > 0.267 {
		t.Fatalf("0x44ffffff: %v", got)
	}
	for _, opaque := range []string{"0xffffffff", "0xff000000", "", "white"} {
		if got := fontOpacity(opaque); got != 0 {
			t.Fatalf("%q: %v, want 0 (not sent)", opaque, got)
		}
	}
}

func TestNoObjectStoreMeansNoBitmapPayload(t *testing.T) {
	base, err := NewVanillaIndex(fontPack(t), "")
	if err != nil {
		t.Fatal(err)
	}
	window, _ := base.Window("panel")
	meta := window["fonts"].(map[string]any)["hoi_36header"].(map[string]any)
	if _, present := meta["bitmap"]; present {
		t.Fatal("without a reachable atlas the browser must fall back to web fonts")
	}
}
