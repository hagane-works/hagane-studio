package mapgen

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"testing"
)

func TestNewWorldGroupsEveryProvince(t *testing.T) {
	m, world := twoIslands(t)
	kind := map[int]string{}
	for _, pr := range m.Provinces {
		kind[pr.ID] = pr.Kind
	}
	inState := map[int]int{}
	for k, st := range world.All.States {
		for _, id := range st.Provinces {
			if kind[id] != "land" {
				t.Errorf("state %d holds %s province %d", k+1, kind[id], id)
			}
			if inState[id] != 0 {
				t.Errorf("province %d in states %d and %d", id, inState[id], k+1)
			}
			inState[id] = k + 1
		}
	}
	inRegion := map[int]int{}
	for _, region := range world.Regions {
		statesHere := map[int]bool{}
		for _, id := range region.Provinces {
			if inRegion[id] != 0 {
				t.Errorf("province %d in regions %d and %d", id, inRegion[id], region.ID)
			}
			inRegion[id] = region.ID
			if region.Sea && kind[id] != "sea" {
				t.Errorf("sea region %d holds %s province %d", region.ID, kind[id], id)
			}
			if s := inState[id]; s > 0 {
				statesHere[s] = true
			}
		}
		for s := range statesHere {
			for _, id := range world.All.States[s-1].Provinces {
				if inRegion[id] != 0 && inRegion[id] != region.ID {
					t.Errorf("state %d split between regions %d and %d", s, inRegion[id], region.ID)
				}
			}
		}
	}
	for _, pr := range m.Provinces {
		if pr.Kind == "land" && inState[pr.ID] == 0 {
			t.Errorf("land province %d in no state", pr.ID)
		}
		if inRegion[pr.ID] == 0 {
			t.Errorf("%s province %d in no region", pr.Kind, pr.ID)
		}
	}
	if len(world.All.States) < 2 {
		t.Fatalf("%d states, want the two islands' land in several", len(world.All.States))
	}

	positions := world.Positions(3)
	stacked := map[int]bool{}
	for _, line := range positions.UnitStacks {
		id, _ := strconv.Atoi(strings.SplitN(line, ";", 2)[0])
		stacked[id] = true
	}
	for _, pr := range m.Provinces {
		if !stacked[pr.ID] {
			t.Errorf("no unit stacks for province %d", pr.ID)
		}
	}
	weather := map[string]int{}
	for _, line := range positions.Weather {
		weather[strings.SplitN(line, ";", 2)[0]]++
	}
	for _, region := range world.Regions {
		if weather[strconv.Itoa(region.ID)] != 2 {
			t.Errorf("region %d has %d weather points, want 2", region.ID, weather[strconv.Itoa(region.ID)])
		}
	}
	if len(positions.SupplyNodes) != len(world.All.States) || len(positions.Buildings) == 0 {
		t.Errorf("%d supply nodes for %d states, %d building lines", len(positions.SupplyNodes), len(world.All.States), len(positions.Buildings))
	}
}

func twoIslands(t *testing.T) (*Map, *World) {
	t.Helper()
	sketch := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			c := color.RGBA{0x28, 0x50, 0xaa, 255}
			if (x-16)*(x-16)+(y-16)*(y-16) < 90 {
				c = color.RGBA{0xa0, 0xd2, 0x64, 255}
			}
			if (x-46)*(x-46)/2+(y-14)*(y-14) < 60 {
				c = color.RGBA{0x1e, 0x6e, 0x28, 255}
			}
			sketch.Set(x, y, c)
		}
	}
	m, err := Generate(sketch, Options{Width: 512, Height: 256, Seed: 3, LandProvincePixels: 900, SeaProvincePixels: 4000})
	if err != nil {
		t.Fatal(err)
	}
	world, err := NewWorld(m, GreyPalette(), GreyPalette(), WorldOptions{Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	return m, world
}

func TestGenerateKeepsProvincesInTheGamesBox(t *testing.T) {
	sketch := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			c := color.RGBA{0x28, 0x50, 0xaa, 255}
			if x > 10 && x < 40 && y > 4 && y < 28 {
				c = color.RGBA{0xa0, 0xd2, 0x64, 255}
			}
			sketch.Set(x, y, c)
		}
	}
	m, err := Generate(sketch, Options{Width: 1536, Height: 768, Seed: 2, LandProvincePixels: 200000, SeaProvincePixels: 400000})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range Check(m.CheckInput()) {
		if f.Code == "province_box" {
			t.Errorf("%d provinces too big for the game: %v", f.Count, f.At)
		}
	}
}
