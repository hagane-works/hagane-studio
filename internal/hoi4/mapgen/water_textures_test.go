package mapgen

import (
	"encoding/binary"
	"testing"

	"haganeworks/internal/hoi4/imgcodec"
)

func halfWorld(w, h int, landLeft bool) *Base {
	b := &Base{
		Width: w, Height: h,
		Province: make([]int32, w*h), Heights: make([]uint8, w*h),
		Terrain: make([]uint8, w*h), Rivers: make([]uint8, w*h),
		Definitions: []Definition{{ID: 1, Kind: "land", Terrain: "plains"}, {ID: 2, Kind: "sea", Terrain: "ocean"}},
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if (x < w/2) == landLeft {
				b.Province[p], b.Heights[p], b.Rivers[p] = 1, 120, RiverLand
			} else {
				b.Province[p], b.Heights[p], b.Rivers[p] = 2, 60, RiverWater
			}
		}
	}
	return b
}

func paintedHalves(b *Base, land, water [4]uint8) []byte {
	w, h := b.Width/2, b.Height/2
	pix := make([]byte, w*h*4)
	kinds := b.kinds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := water
			if kinds[b.Province[b.texel(w, h, x, y)]] == kindLand {
				c = land
			}
			i := (y*w + x) * 4
			copy(pix[i:i+4], c[:])
			pix[i] += uint8((x + y) % 3)
		}
	}
	return imgcodec.EncodeDXT5DDS(pix, w, h)
}

func meanOver(pix []byte, w, x0, x1, y0, y1 int) [4]float64 {
	var sum [4]float64
	n := 0.0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			for c := 0; c < 4; c++ {
				sum[c] += float64(pix[(y*w+x)*4+c])
			}
			n++
		}
	}
	for c := range sum {
		sum[c] /= n
	}
	return sum
}

func TestPaintWorldWaterFollowsTheNewWorldNotTheBaseGame(t *testing.T) {
	game := halfWorld(256, 128, true)
	world := halfWorld(256, 128, false)
	levels, err := world.PaintWorldWater(game, paintedHalves(game, [4]uint8{40, 160, 40, 255}, [4]uint8{20, 40, 200, 255}), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != len(WaterColormapFiles) {
		t.Fatalf("%d levels", len(levels))
	}
	for i, level := range levels {
		pix, w, h, ok := imgcodec.DecodeDDS(level)
		if !ok || w != 128>>i || h != 64>>i || string(level[84:88]) != "DXT5" {
			t.Fatalf("level %d: %dx%d ok=%v", i, w, h, ok)
		}
		left := meanOver(pix, w, 0, w/2-1, 0, h)
		right := meanOver(pix, w, w/2+1, w, 0, h)
		if left[2] < 150 || left[1] > 80 || right[1] < 120 || right[2] > 90 || left[3] < 250 {
			t.Fatalf("level %d: the new sea (left) %v, the new land (right) %v", i, left, right)
		}
	}
}

func TestPaintWorldFogFollowsTheNewWorldNotTheBaseGame(t *testing.T) {
	game := halfWorld(256, 128, true)
	world := halfWorld(256, 128, false)
	dds, err := world.PaintWorldFog(game, paintedHalves(game, [4]uint8{180, 180, 180, 30}, [4]uint8{70, 70, 70, 120}), 7)
	if err != nil {
		t.Fatal(err)
	}
	pix, w, h, ok := imgcodec.DecodeDDS(dds)
	if !ok || w != 128 || h != 64 {
		t.Fatalf("%dx%d ok=%v", w, h, ok)
	}
	sea := meanOver(pix, w, 0, w/2-1, 0, h)
	land := meanOver(pix, w, w/2+1, w, 0, h)
	if sea[1] > 90 || sea[3] < 100 || land[1] < 150 || land[3] > 60 {
		t.Fatalf("the new sea (left) %v, the new land (right) %v", sea, land)
	}
	mips := int(binary.LittleEndian.Uint32(dds[28:]))
	if mips != 8 {
		t.Fatalf("mipmaps %d", mips)
	}
	size := 128
	for w, h := 128, 64; ; w, h = max(1, w/2), max(1, h/2) {
		size += ((w + 3) / 4) * ((h + 3) / 4) * 16
		if w == 1 && h == 1 {
			break
		}
	}
	if len(dds) != size {
		t.Fatalf("%d bytes, want %d for the whole mipmap chain", len(dds), size)
	}
}

func TestCoastDistance(t *testing.T) {
	const w, h = 40, 3
	surface := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 20; x < w; x++ {
			surface[y*w+x] = 1
		}
	}
	d := coastDistance(surface, w, h)
	for _, c := range []struct {
		x    int
		want float32
	}{{19, 1}, {20, 1}, {15, 5}, {24, 5}, {0, 20}, {39, 20}} {
		if got := d[w+c.x]; got != c.want {
			t.Errorf("x=%d: %v, want %v", c.x, got, c.want)
		}
	}
	if b := coastBand(1); b != 0 {
		t.Errorf("band at the coast %v", b)
	}
	if b := coastBand(1000); b != coastBands-1 {
		t.Errorf("band far out %v", b)
	}
}
