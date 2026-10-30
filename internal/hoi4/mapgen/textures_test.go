package mapgen

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/bmp"
)

func testColormap(b *Base) []byte {
	w, h := b.Width/2, b.Height/2
	out := make([]byte, 128+w*h*4)
	copy(out, "DDS ")
	put := func(at int, v uint32) { binary.LittleEndian.PutUint32(out[at:], v) }
	put(4, 124)
	put(8, 0x100f)
	put(12, uint32(h))
	put(16, uint32(w))
	put(20, uint32(w*4))
	put(28, 1)
	put(76, 32)
	put(80, 0x41)
	put(88, 32)
	put(92, 0xff0000)
	put(96, 0xff00)
	put(100, 0xff)
	put(104, 0xff000000)
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			i := 128 + (cy*w+cx)*4
			if b.Province[2*cy*b.Width+2*cx] == 1 {
				out[i], out[i+1], out[i+2], out[i+3] = 50, 90, 100, 0
			} else {
				out[i], out[i+1], out[i+2], out[i+3] = 170, 80, 40, 0
			}
		}
	}
	return out
}

func TestPaintTexturesOverNewLand(t *testing.T) {
	b := testBase()
	b.Definitions[0].Terrain = "forest"
	forest := sketchOf(40, 20, func(x, y int) Class {
		dx, dy := (float64(x)-20)/17, (float64(y)-10)/8
		if dx*dx+dy*dy > 1 {
			return Sea
		}
		if x > 26 {
			return Mountain
		}
		return Forest
	})
	colormap := testColormap(b)

	trees := image.NewPaletted(image.Rect(0, 0, 64, 64), GreyPalette())
	for ty := 0; ty < 64; ty++ {
		for tx := 0; tx < 64; tx++ {
			if b.Province[(ty*4)*b.Width+tx*4] == 1 {
				trees.Pix[ty*trees.Stride+tx] = 6
			}
		}
	}
	treeBytes, _ := EncodeBMP(trees)
	flat := image.NewRGBA(image.Rect(0, 0, 128, 128))
	for i := 0; i < len(flat.Pix); i += 4 {
		flat.Pix[i], flat.Pix[i+1], flat.Pix[i+2], flat.Pix[i+3] = 128, 128, 255, 255
	}
	normalBytes, _ := EncodeBMP(flat)

	add, err := b.AddLand(forest, AddOptions{Place: image.Rect(80, 40, 200, 100), Seed: 4, LandProvincePixels: 150})
	if err != nil {
		t.Fatal(err)
	}
	fresh := b.newPixels(add)

	painted, err := b.PaintColormap(colormap, add, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(painted[:128], colormap[:128]) || len(painted) != len(colormap) {
		t.Fatal("header or size changed")
	}
	changed, landLike := 0, 0
	for cy := 0; cy < b.Height/2; cy++ {
		for cx := 0; cx < b.Width/2; cx++ {
			i := 128 + (cy*b.Width/2+cx)*4
			isNew := fresh[int32(2*cy*b.Width+2*cx)]
			if !isNew {
				if !bytes.Equal(painted[i:i+4], colormap[i:i+4]) {
					t.Fatalf("(%d,%d) not new land, changed", cx, cy)
				}
				continue
			}
			changed++
			bl, g, r := float64(painted[i]), float64(painted[i+1]), float64(painted[i+2])
			if math.Abs(r-100) <= 10 && math.Abs(g-90) <= 9 && math.Abs(bl-50) <= 6 && painted[i+3] == 0 {
				landLike++
			}
		}
	}
	if changed == 0 || landLike != changed {
		t.Errorf("%d of %d new land pixels coloured like the land", landLike, changed)
	}

	normals, err := b.PaintNormals(normalBytes, add)
	if err != nil {
		t.Fatal(err)
	}
	img, _ := bmp.Decode(bytes.NewReader(normals))
	tilted := 0
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), 255}
			inside := image.Pt(2*x, 2*y).In(add.Changed.Inset(-6))
			if !inside && c != (color.RGBA{128, 128, 255, 255}) {
				t.Fatalf("normal at (%d,%d) outside the new land changed: %v", x, y, c)
			}
			if c != (color.RGBA{128, 128, 255, 255}) {
				tilted++
			}
		}
	}
	if tilted == 0 {
		t.Error("the new hills and mountains left the normals flat")
	}

	treeOut, err := b.PaintTrees(treeBytes, add, 4)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := bmp.Decode(bytes.NewReader(treeOut))
	pal := decoded.(*image.Paletted)
	forestTrees, mountainTrees := 0, 0
	for ty := 0; ty < 64; ty++ {
		for tx := 0; tx < 64; tx++ {
			p := (ty*4)*b.Width + tx*4
			index := pal.Pix[ty*pal.Stride+tx]
			switch {
			case !fresh[int32(p)]:
				if index != trees.Pix[ty*trees.Stride+tx] {
					t.Fatalf("tree (%d,%d) outside the new land changed", tx, ty)
				}
			case newTerrain(b.Terrain[p]) == "forest" && index == 6:
				forestTrees++
			case index != 0:
				mountainTrees++
			}
		}
	}
	if forestTrees == 0 || mountainTrees != 0 {
		t.Errorf("trees on new forest %d, elsewhere %d", forestTrees, mountainTrees)
	}
}

func TestPaintTexturesOnTheBaseGameMap(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(root, "map", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	b, err := LoadBase(read("provinces.bmp"), read("heightmap.bmp"), read("terrain.bmp"), read("rivers.bmp"), read("definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	island := sketchOf(60, 30, func(x, y int) Class {
		dx, dy := (float64(x)-30)/25, (float64(y)-15)/12
		if dx*dx+dy*dy > 1 {
			return Sea
		}
		if x < 30 {
			return Forest
		}
		return Hills
	})
	add, err := b.AddLand(island, AddOptions{Place: image.Rect(200, 1100, 320, 1160), Seed: 9})
	if err != nil {
		t.Fatal(err)
	}
	colormap := read("terrain/colormap_rgb_cityemissivemask_a.dds")
	painted, err := b.PaintColormap(colormap, add, 9)
	if err != nil || len(painted) != len(colormap) {
		t.Fatalf("colour map: %v (%d bytes of %d)", err, len(painted), len(colormap))
	}

	var before, after [3]float64
	n := 0.0
	for p := range b.newPixels(add) {
		x, y := int(p)%b.Width/2, int(p)/b.Width/2
		i := 128 + (y*b.Width/2+x)*4
		for c := 0; c < 3; c++ {
			before[c] += float64(colormap[i+c])
			after[c] += float64(painted[i+c])
		}
		n++
	}
	t.Logf("BGR before %.0f %.0f %.0f, after %.0f %.0f %.0f", before[0]/n, before[1]/n, before[2]/n, after[0]/n, after[1]/n, after[2]/n)
	if !(before[0] > before[2]) || !(after[2] > after[0]) {
		t.Error("the new land is still sea-coloured")
	}
	normals, err := b.PaintNormals(read("world_normal.bmp"), add)
	if err != nil {
		t.Fatalf("normals: %v", err)
	}
	if size, err := bmp.DecodeConfig(bytes.NewReader(normals)); err != nil || size.Width != 2816 || size.Height != 1024 {
		t.Fatalf("normals %+v %v", size, err)
	}
	trees, err := b.PaintTrees(read("trees.bmp"), add, 9)
	if err != nil {
		t.Fatalf("trees: %v", err)
	}
	decoded, err := bmp.Decode(bytes.NewReader(trees))
	if err != nil || decoded.Bounds().Dx() != 1650 {
		t.Fatalf("trees decode: %v", err)
	}
	planted := 0
	original, _ := bmp.Decode(bytes.NewReader(read("trees.bmp")))
	op, np := original.(*image.Paletted), decoded.(*image.Paletted)
	for i := range op.Pix {
		if op.Pix[i] != np.Pix[i] && np.Pix[i] != 0 {
			planted++
		}
	}
	t.Logf("trees planted on %d tree-map pixels", planted)
	if planted == 0 {
		t.Error("no trees on the new forest")
	}
}
