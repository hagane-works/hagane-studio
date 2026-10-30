package mapgen

import (
	"math"
	"strings"
	"testing"
)

func TestReterrainPaintsOneProvince(t *testing.T) {
	_, world := twoIslands(t)
	base := world.Base
	var target int
	for _, d := range base.Definitions {
		if d.Kind == "land" {
			target = d.ID
			break
		}
	}
	heights := append([]uint8(nil), base.Heights...)
	terrain := append([]uint8(nil), base.Terrain...)
	add, err := base.Reterrain(map[int]string{target: "mountain"}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(add.Pixels(target)) == 0 || add.Changed.Empty() {
		t.Fatalf("no pixels or area: %+v", add.Changed)
	}
	var before, after float64
	for _, p := range add.Pixels(target) {
		if base.Terrain[p] != 6 {
			t.Fatalf("pixel %d terrain index %d, want 6", p, base.Terrain[p])
		}
		if base.Heights[p] < LandHeight {
			t.Fatalf("pixel %d sank to %d", p, base.Heights[p])
		}
		before += float64(heights[p])
		after += float64(base.Heights[p])
	}
	if after <= before {
		t.Errorf("the mountain did not rise: mean %.1f → %.1f", before/float64(len(add.Pixels(target))), after/float64(len(add.Pixels(target))))
	}
	own := map[int32]bool{}
	for _, p := range add.Pixels(target) {
		own[p] = true
	}
	for p := range base.Province {
		if !own[int32(p)] && (base.Heights[p] != heights[p] || base.Terrain[p] != terrain[p]) {
			t.Fatalf("pixel %d outside the province changed", p)
		}
	}
	for _, d := range base.Definitions {
		if d.ID == target && d.Terrain != "mountain" {
			t.Errorf("definition terrain %q", d.Terrain)
		}
	}
	if !strings.Contains(string(base.DefinitionCSV()), "mountain") {
		t.Error("definition.csv does not carry it")
	}
}

func TestReterrainRefuses(t *testing.T) {
	_, world := twoIslands(t)
	base := world.Base
	var sea, land int
	for _, d := range base.Definitions {
		if d.Kind == "sea" && sea == 0 {
			sea = d.ID
		}
		if d.Kind == "land" && land == 0 {
			land = d.ID
		}
	}
	for name, terrains := range map[string]map[int]string{
		"water":   {sea: "plains"},
		"unknown": {land: "lava"},
		"missing": {999999: "plains"},
	} {
		if _, err := base.Reterrain(terrains, 1); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if idx, ok := TerrainIndex("urban"); !ok || idx != 13 {
		t.Errorf("urban index %d %v", idx, ok)
	}
}

func TestReterrainShiftsColoursKeepingGrain(t *testing.T) {
	_, world := twoIslands(t)
	base := world.Base
	dds := testColormap(base)
	w, h, pitch, err := colormapLayout(dds)
	if err != nil {
		t.Fatal(err)
	}
	terrains := base.terrainOf()
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			p := base.texel(w, h, cx, cy)
			grain := byte((cx*7 + cy*13) % 11)
			i := 128 + cy*pitch + cx*4
			switch terrains[base.Province[p]] {
			case "plains":
				dds[i], dds[i+1], dds[i+2] = 40+grain, 100+grain, 60+grain
			case "forest":
				dds[i], dds[i+1], dds[i+2] = 30+grain, 70+grain, 30+grain
			}
		}
	}
	var target int
	for _, d := range base.Definitions {
		if d.Kind == "land" && d.Terrain == "plains" {
			target = d.ID
			break
		}
	}
	add, err := base.Reterrain(map[int]string{target: "forest"}, 4)
	if err != nil {
		t.Fatal(err)
	}
	out, err := base.PaintColormap(dds, add, 4)
	if err != nil {
		t.Fatal(err)
	}
	var reds, before []float64
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			i := 128 + cy*pitch + cx*4
			if int(base.Province[base.texel(w, h, cx, cy)]) != target {
				if out[i] != dds[i] || out[i+1] != dds[i+1] || out[i+2] != dds[i+2] {
					t.Fatalf("texel %d,%d outside the province changed", cx, cy)
				}
				continue
			}
			if d := float64(out[i+2]) - float64(dds[i+2]); d < -36 || d > -24 {
				t.Fatalf("texel %d,%d red moved %.0f, want about -30", cx, cy, d)
			}
			reds = append(reds, float64(out[i+2]))
			before = append(before, float64(dds[i+2]))
		}
	}
	spread := func(v []float64) float64 {
		var sum, sq float64
		for _, x := range v {
			sum += x
			sq += x * x
		}
		n := float64(len(v))
		return math.Sqrt(sq/n - (sum/n)*(sum/n))
	}
	if len(reds) == 0 {
		t.Fatal("no texel of the province")
	}
	if spread(before) < 1 || math.Abs(spread(reds)-spread(before)) > 0.5 {
		t.Errorf("grain lost: spread %.2f → %.2f", spread(before), spread(reds))
	}
}
