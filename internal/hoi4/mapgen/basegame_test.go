package mapgen

import (
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"golang.org/x/image/bmp"
)

func TestCheckBaseGameMapHasNoErrors(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	read := func(name string) image.Image {
		t.Helper()
		file, err := os.Open(filepath.Join(root, "map", name))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		img, err := bmp.Decode(file)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return img
	}
	provinces := read("provinces.bmp")
	bounds := provinces.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	rgb := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := provinces.At(x, y).RGBA()
			rgb[y*w+x] = uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(b>>8)
		}
	}
	heights := make([]uint8, w*h)
	heightImg := read("heightmap.bmp")
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := heightImg.At(x, y).RGBA()
			heights[y*w+x] = uint8(r >> 8)
		}
	}
	terrainImg, ok := read("terrain.bmp").(*image.Paletted)
	if !ok {
		t.Fatal("terrain.bmp is not indexed")
	}
	definitions, err := os.ReadFile(filepath.Join(root, "map", "definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	terrainTxt, err := os.ReadFile(filepath.Join(root, "common", "terrain", "00_terrain.txt"))
	if err != nil {
		t.Fatal(err)
	}
	indexes := map[uint8]bool{}
	for _, m := range regexp.MustCompile(`color\s*=\s*\{\s*(\d+)\s*\}`).FindAllSubmatch(terrainTxt, -1) {
		if n, err := strconv.Atoi(string(m[1])); err == nil && n < 256 {
			indexes[uint8(n)] = true
		}
	}
	size := [2]int{w, h}
	findings := Check(CheckInput{
		Width: w, Height: h, ProvinceRGB: rgb, Definitions: ParseDefinitions(definitions),
		HeightSize: size, TerrainSize: [2]int{terrainImg.Rect.Dx(), terrainImg.Rect.Dy()}, RiverSize: size,
		Heights: heights, Terrain: terrainImg.Pix, TerrainIndexes: indexes,
	})
	for _, finding := range findings {
		t.Logf("%s %s ×%d %s %v", finding.Severity, finding.Code, finding.Count, finding.Message, finding.At[:min(3, len(finding.At))])
	}
	if errs := Errors(findings); len(errs) > 0 {
		t.Errorf("the base game's map has %d kinds of error", len(errs))
	}
}
