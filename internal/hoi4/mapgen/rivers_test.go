package mapgen

import (
	"strings"
	"testing"
)

func riverLine(b *Base, y, x0, x1 int, first uint8) map[int]uint8 {
	out := map[int]uint8{}
	for x := x0; x <= x1; x++ {
		v := uint8(RiverNarrow)
		if x == x0 {
			v = first
		}
		out[y*b.Width+x] = v
	}
	return out
}

func TestEditRiversDrawsAndErases(t *testing.T) {
	_, world := twoIslands(t)
	b := world.Base
	before := append([]uint8(nil), b.Rivers...)
	main := riverLine(b, 128, 110, 140, RiverSource)
	if n, err := b.EditRivers(main); err != nil || n != len(main) {
		t.Fatalf("main river: %d, %v", n, err)
	}

	tributary := map[int]uint8{}
	for y := 118; y <= 127; y++ {
		v := uint8(RiverNarrow)
		if y == 127 {
			v = RiverJoin
		}
		tributary[y*b.Width+125] = v
	}
	if _, err := b.EditRivers(tributary); err != nil {
		t.Fatalf("tributary: %v", err)
	}
	at, values := RiverPixels(b.Rivers)
	if len(at) != len(main)+len(tributary) || len(values) != len(at) {
		t.Fatalf("%d river pixels", len(at))
	}
	erase := map[int]uint8{}
	for _, p := range at {
		erase[p] = RiverLand
	}
	if _, err := b.EditRivers(erase); err != nil {
		t.Fatalf("erase: %v", err)
	}
	for p := range before {
		if b.Rivers[p] != before[p] {
			t.Fatalf("pixel %d is %d, was %d", p, b.Rivers[p], before[p])
		}
	}
}

func TestEditRiversRefuses(t *testing.T) {
	_, world := twoIslands(t)
	b := world.Base
	if _, err := b.EditRivers(riverLine(b, 128, 110, 140, RiverSource)); err != nil {
		t.Fatal(err)
	}
	before := append([]uint8(nil), b.Rivers...)
	var sea int
	for p, v := range b.Rivers {
		if v == RiverWater {
			sea = p
			break
		}
	}
	middle := map[int]uint8{128*b.Width + 120: RiverSource}
	block := map[int]uint8{127*b.Width + 120: RiverNarrow, 127*b.Width + 121: RiverNarrow}

	unjoined := map[int]uint8{126*b.Width + 130: RiverSource, 127*b.Width + 130: RiverNarrow}
	for name, c := range map[string]struct {
		pixels map[int]uint8
		says   string
	}{
		"water":    {map[int]uint8{sea: RiverNarrow}, "水面上"},
		"value":    {map[int]uint8{128*b.Width + 100: 100}, "没有 100"},
		"off map":  {map[int]uint8{len(b.Rivers): RiverNarrow}, "不在地图上"},
		"middle":   {middle, "接错"},
		"block":    {block, "接错"},
		"unjoined": {unjoined, "接错"},
		"dry sea":  {map[int]uint8{sea: RiverLand}, "不是河"},
	} {
		_, err := b.EditRivers(c.pixels)
		if err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s: %v", name, err)
		}
		for p := range before {
			if b.Rivers[p] != before[p] {
				t.Fatalf("%s: pixel %d changed", name, p)
			}
		}
	}
}

func TestEditRiversOnlyRefusesNewFaults(t *testing.T) {
	_, world := twoIslands(t)
	b := world.Base

	for x := 110; x <= 120; x++ {
		b.Rivers[128*b.Width+x] = RiverNarrow
	}
	b.Rivers[128*b.Width+115] = RiverSource

	if _, err := b.EditRivers(map[int]uint8{128*b.Width + 121: RiverNarrow}); err != nil {
		t.Fatal(err)
	}
}
