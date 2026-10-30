package mapgen

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"testing"
)

func fakeGame() *Base {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(255 - i), uint8(i * 7), 255}
	}
	palette[5] = palette[0]
	g := &Base{Width: 4, Height: 1, TerrainPalette: palette, RiverPalette: GreyPalette(),
		Definitions: []Definition{
			{ID: 1, Kind: "land", Terrain: "plains"}, {ID: 2, Kind: "land", Terrain: "forest"},
			{ID: 3, Kind: "sea", Terrain: "ocean"}, {ID: 4, Kind: "lake", Terrain: "lakes"},
		},
		Province: []int32{1, 2, 3, 4}, Terrain: []uint8{0, 1, 15, 14},
	}
	return g
}

func uploadImages(game *Base) UploadInput {
	const w, h = 80, 40
	provinces := image.NewRGBA(image.Rect(0, 0, w, h))
	terrain := image.NewRGBA(image.Rect(0, 0, w, h))
	heights := image.NewGray(image.Rect(0, 0, w, h))
	colours := []color.RGBA{{200, 10, 10, 255}, {10, 200, 10, 255}, {10, 10, 200, 255}, {200, 200, 10, 255}}
	index := []int{0, 1, 15, 1}
	height := []uint8{110, 120, 89, 89}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			k := 2
			switch {
			case x >= 10 && x < 12 && y >= 30 && y < 32:
				k = 3
			case x < 40 && y < 20:
				k = 0
			case x < 40:
				k = 1
			}
			provinces.Set(x, y, colours[k])
			terrain.Set(x, y, game.TerrainPalette[index[k]])
			heights.SetGray(x, y, color.Gray{Y: height[k]})
		}
	}
	return UploadInput{Provinces: provinces, Heightmap: heights, Terrain: terrain}
}

func TestReadUploadMakesDefinitions(t *testing.T) {
	game := fakeGame()
	up, err := ReadUpload(uploadImages(game), game)
	if err != nil {
		t.Fatal(err)
	}
	if !up.MadeDefinitions || !up.MadeRivers {
		t.Errorf("made definitions %t, rivers %t", up.MadeDefinitions, up.MadeRivers)
	}
	want := []string{"1 land plains coastal=true", "2 sea ocean coastal=true", "3 land forest coastal=true", "4 lake lakes coastal=false"}
	var got []string
	for _, d := range up.Base.Definitions {
		got = append(got, strings.Join([]string{strconv.Itoa(d.ID), d.Kind, d.Terrain, "coastal=" + strconv.FormatBool(d.Coastal)}, " "))
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("definitions got %v, want %v", got, want)
	}
	lake := 30*80 + 10
	if up.Base.Province[0] != 1 || up.Base.Province[lake] != 4 || up.Base.Terrain[0] != 0 || up.Base.Terrain[79] != 15 || up.Base.Heights[lake] != 89 {
		t.Errorf("pixels: province %d/%d terrain %d/%d height %d", up.Base.Province[0], up.Base.Province[lake], up.Base.Terrain[0], up.Base.Terrain[79], up.Base.Heights[lake])
	}
	if up.Base.Rivers[0] != RiverLand || up.Base.Rivers[79] != RiverWater || up.Base.Rivers[lake] != RiverWater {
		t.Errorf("rivers %d %d %d", up.Base.Rivers[0], up.Base.Rivers[79], up.Base.Rivers[lake])
	}
	if errs := Errors(Check(up.Check)); len(errs) > 0 {
		t.Errorf("findings on a good map: %+v", errs)
	}
	if _, err := LoadBase(mustBMP(t, up.Base.ProvincesImage()), mustBMP(t, up.Base.HeightImage()), mustBMP(t, up.Base.TerrainImage()), mustBMP(t, up.Base.RiversImage()), up.Base.DefinitionCSV()); err != nil {
		t.Errorf("the written files do not read back: %v", err)
	}
}

func TestReadUploadTakesDefinitionsAndRefusesSizes(t *testing.T) {
	game := fakeGame()
	in := uploadImages(game)
	in.Definitions = []byte("0;0;0;0;land;false;unknown;0\n7;200;10;10;land;false;hills;2\n8;10;10;200;sea;false;ocean;0\n")
	up, err := ReadUpload(in, game)
	if err != nil {
		t.Fatal(err)
	}
	if up.MadeDefinitions || len(up.Base.Definitions) != 2 || up.Base.Definitions[0].ID != 7 || up.Base.Province[0] != 7 {
		t.Errorf("definitions %+v, first pixel %d", up.Base.Definitions, up.Base.Province[0])
	}
	undefined := false
	for _, finding := range Check(up.Check) {
		undefined = undefined || finding.Code == "undefined_color"
	}
	if !undefined {
		t.Error("colours missing from definition.csv were not found")
	}
	in.Heightmap = image.NewGray(image.Rect(0, 0, 5, 2))
	var refused *UploadError
	if _, err := ReadUpload(in, game); err == nil || !asUploadError(err, &refused) || !strings.Contains(refused.Message, "不一样大") {
		t.Errorf("a heightmap of another size: %v", err)
	}
}

func asUploadError(err error, target **UploadError) bool {
	e, ok := err.(*UploadError)
	if ok {
		*target = e
	}
	return ok
}

func mustBMP(t *testing.T, img image.Image) []byte {
	t.Helper()
	data, err := EncodeBMP(img)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReadUploadFillsStrayPixels(t *testing.T) {
	game := fakeGame()
	in := uploadImages(game)
	provinces := in.Provinces.(*image.RGBA)
	for y := 5; y < 8; y++ {
		for x := 5; x < 8; x++ {
			provinces.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	provinces.Set(50, 10, color.RGBA{1, 2, 3, 255})
	in.Definitions = []byte("0;0;0;0;land;false;unknown;0\n1;200;10;10;land;false;plains;1\n2;10;200;10;land;false;forest;1\n3;10;10;200;sea;false;ocean;0\n4;200;200;10;lake;false;lakes;0\n")
	errorsOf := func(up *Upload) map[string]bool {
		out := map[string]bool{}
		for _, finding := range Check(up.Check) {
			if finding.Severity == "error" {
				out[finding.Code] = true
			}
		}
		return out
	}
	up, err := ReadUpload(in, game)
	if err != nil {
		t.Fatal(err)
	}
	if got := errorsOf(up); !got["black_pixel"] || !got["undefined_color"] || up.FilledPixels != 0 {
		t.Fatalf("without the fix: errors %v, filled %d", got, up.FilledPixels)
	}
	in.FillStrayPixels = true
	up, err = ReadUpload(in, game)
	if err != nil {
		t.Fatal(err)
	}
	if got := errorsOf(up); len(got) != 0 || up.FilledPixels != 10 {
		t.Fatalf("with the fix: errors %v, filled %d", got, up.FilledPixels)
	}
	w := up.Base.Width
	if up.Base.Province[6*w+6] != 1 || up.Base.Province[10*w+50] != 3 {
		t.Fatalf("the black patch is province %d, the stray pixel %d", up.Base.Province[6*w+6], up.Base.Province[10*w+50])
	}
	for p, id := range up.Base.Province {
		if id == 0 {
			t.Fatalf("pixel %d is still no province's", p)
		}
	}
}
