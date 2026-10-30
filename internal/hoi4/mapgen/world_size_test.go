package mapgen

import (
	"bytes"
	"image"
	"testing"

	"golang.org/x/image/bmp"

	"haganeworks/internal/hoi4/imgcodec"
)

func TestWorldTexturesTakeTheMapsOwnSize(t *testing.T) {
	game := halfWorld(352, 128, true)
	world := halfWorld(320, 160, false)
	w, h := world.Width/2, world.Height/2

	colormap, err := world.PaintWorldColormap(game, testColormap(game), 7)
	if err != nil {
		t.Fatal(err)
	}
	if cw, ch, _, err := colormapLayout(colormap); err != nil || cw != w || ch != h {
		t.Fatalf("colour map %dx%d %v", cw, ch, err)
	}

	normals, err := world.WorldNormals()
	if err != nil {
		t.Fatal(err)
	}
	if img, err := bmp.Decode(bytes.NewReader(normals)); err != nil || img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		t.Fatalf("normals %v %v", img.Bounds(), err)
	}

	source, err := EncodeBMP(image.NewPaletted(image.Rect(0, 0, 88, 32), GreyPalette()))
	if err != nil {
		t.Fatal(err)
	}
	trees, err := world.PaintWorldTrees(game, source, 7)
	if err != nil {
		t.Fatal(err)
	}
	if img, err := bmp.Decode(bytes.NewReader(trees)); err != nil || img.Bounds().Dx() != 80 || img.Bounds().Dy() != 40 {
		t.Fatalf("trees %v %v", img.Bounds(), err)
	}

	levels, err := world.PaintWorldWater(game, paintedHalves(game, [4]uint8{40, 160, 40, 255}, [4]uint8{20, 40, 200, 255}), 7)
	if err != nil {
		t.Fatal(err)
	}
	for i, level := range levels {
		if _, lw, lh, ok := imgcodec.DecodeDDS(level); !ok || lw != w>>i || lh != h>>i {
			t.Fatalf("water level %d: %dx%d", i, lw, lh)
		}
	}

	fog, err := world.PaintWorldFog(game, paintedHalves(game, [4]uint8{180, 180, 180, 30}, [4]uint8{70, 70, 70, 120}), 7)
	if err != nil {
		t.Fatal(err)
	}
	if _, fw, fh, ok := imgcodec.DecodeDDS(fog); !ok || fw != w || fh != h {
		t.Fatalf("fog %dx%d", fw, fh)
	}
}
