package mapgen

import (
	"strings"
	"testing"
)

func floodWorld(t *testing.T) (*Base, int) {
	_, world := twoIslands(t)
	b := world.Base
	for x := 70; x <= 190; x++ {
		v := uint8(RiverNarrow)
		if x == 70 {
			v = RiverSource
		}
		b.Rivers[128*b.Width+x] = v
	}
	for y := 100; y <= 127; y++ {
		v := uint8(RiverNarrow)
		if y == 127 {
			v = RiverJoin
		}
		b.Rivers[y*b.Width+150] = v
	}
	all := make([]int, len(b.Rivers))
	for p := range all {
		all[p] = p
	}
	if n := riverFaults(b.Rivers, b.Width, b.Height, all); n != 0 {
		t.Fatalf("the test rivers have %d faults", n)
	}
	return b, int(b.Province[128*b.Width+125])
}

func riverNetworks(b *Base) []int {
	seen := map[int]bool{}
	var out []int
	for p, v := range b.Rivers {
		if !IsRiver(v) || seen[p] {
			continue
		}
		sources := 0
		stack := []int{p}
		seen[p] = true
		for len(stack) > 0 {
			q := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if b.Rivers[q] == RiverSource {
				sources++
			}
			for _, n := range []int{q - 1, q + 1, q - b.Width, q + b.Width} {
				if n >= 0 && n < len(b.Rivers) && IsRiver(b.Rivers[n]) && !seen[n] {
					seen[n] = true
					stack = append(stack, n)
				}
			}
		}
		out = append(out, sources)
	}
	return out
}

func TestFloodMakesALake(t *testing.T) {
	b, target := floodWorld(t)
	heights := append([]uint8(nil), b.Heights...)
	terrain := append([]uint8(nil), b.Terrain...)
	add, err := b.Flood([]int{target})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range b.Definitions {
		if d.ID == target && (d.Kind != "lake" || d.Terrain != "lakes" || d.Coastal || d.Continent != 0) {
			t.Fatalf("row %+v", d)
		}
	}
	own := map[int32]bool{}
	for _, p := range add.Pixels(target) {
		own[p] = true
		if b.Heights[p] >= LandHeight || b.Terrain[p] != Legend[Lake].Terrain || b.Rivers[p] != RiverWater {
			t.Fatalf("pixel %d: height %d terrain %d river %d", p, b.Heights[p], b.Terrain[p], b.Rivers[p])
		}
	}
	for p := range b.Province {
		if !own[int32(p)] && (b.Heights[p] != heights[p] || b.Terrain[p] != terrain[p]) {
			t.Fatalf("pixel %d outside the lake changed", p)
		}
	}
	all := make([]int, len(b.Rivers))
	for p := range all {
		all[p] = p
	}
	if n := riverFaults(b.Rivers, b.Width, b.Height, all); n != 0 {
		t.Errorf("%d river faults after the flood", n)
	}
	networks := riverNetworks(b)
	if len(networks) < 2 {
		t.Fatalf("the river was not cut: %v", networks)
	}
	for _, sources := range networks {
		if sources != 1 {
			t.Errorf("a river network has %d sources: %v", sources, networks)
		}
	}

	x := 125
	for own[int32(128*b.Width+x)] {
		x++
	}
	if b.Rivers[128*b.Width+x] != RiverSource {
		t.Errorf("below the lake (%d, 128) is %d, want a source", x, b.Rivers[128*b.Width+x])
	}
}

func TestFloodMendsATributary(t *testing.T) {
	b, _ := floodWorld(t)
	target := int(b.Province[128*b.Width+150])
	if _, err := b.Flood([]int{target}); err != nil {
		t.Fatal(err)
	}
	all := make([]int, len(b.Rivers))
	for p := range all {
		all[p] = p
	}
	if n := riverFaults(b.Rivers, b.Width, b.Height, all); n != 0 {
		t.Errorf("%d river faults", n)
	}
	for _, sources := range riverNetworks(b) {
		if sources != 1 {
			t.Errorf("a network has %d sources", sources)
		}
	}
	if top := b.Rivers[100*b.Width+150]; IsRiver(top) && top != RiverSource {
		t.Errorf("the tributary's upper end is %d, want its source", top)
	}
}

func TestFloodRefuses(t *testing.T) {
	_, world := twoIslands(t)
	b := world.Base
	var sea int
	for _, d := range b.Definitions {
		if d.Kind == "sea" {
			sea = d.ID
			break
		}
	}
	for _, ids := range [][]int{{sea}, {999999}} {
		if _, err := b.Flood(ids); err == nil {
			t.Errorf("%v accepted", ids)
		}
	}
}

func TestFloodCleansPositionFiles(t *testing.T) {
	gone := map[int]bool{5: true}
	rail, cut := CutRailways("3 4 1 2 5 6 \r\n2 2 7 8 \r\n1 3 4 5 9 \r\n", gone)
	if cut != 2 || rail != "3 2 1 2 \r\n2 2 7 8 \r\n" {
		t.Errorf("railways (%d cut): %q", cut, rail)
	}
	nodes, n := DropProvinceLines("1 4 \n1 5 \n2 6 \n", " ", []int{1}, gone)
	if n != 1 || nodes != "1 4 \n2 6 \n" {
		t.Errorf("supply nodes: %q", nodes)
	}
	stacks, n := DropProvinceLines("4;0;1.0;2.0;3.0;0.0;0.1\n5;0;1.0;2.0;3.0;0.0;0.1\n5;1;1.0;2.0;3.0;0.0;0.1", ";", []int{0}, gone)
	if n != 2 || stacks != "4;0;1.0;2.0;3.0;0.0;0.1" {
		t.Errorf("unit stacks: %q", stacks)
	}
	adj, n := DropProvinceLines("From;To;Type;Through;a\n1;2;sea;5;x\n5;3;;-1;x\n4;6;;-1;x\n-1;-1;;-1;-1\n", ";", []int{0, 1, 3}, gone)
	if n != 2 || !strings.HasPrefix(adj, "From;To") || !strings.Contains(adj, "4;6;;-1;x") || !strings.HasSuffix(adj, "-1;-1;;-1;-1\n") {
		t.Errorf("adjacencies: %q", adj)
	}
}

func TestFloodDropsItsBuildings(t *testing.T) {
	b, target := floodWorld(t)
	inside := b.line(1, "arms_factory", 128*b.Width+125, 0, 0)
	var outside string
	for p, id := range b.Province {
		if int(id) != target && b.kinds()[id] == kindLand {
			outside = b.line(1, "arms_factory", p, 0, 0)
			break
		}
	}
	text, dropped := b.DropBuildingsIn(inside+"\n"+outside+"\n", map[int]bool{target: true})
	if dropped != 1 || text != outside+"\n" {
		t.Errorf("%d dropped: %q", dropped, text)
	}
}
