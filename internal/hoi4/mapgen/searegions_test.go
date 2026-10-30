package mapgen

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

func stripBase(bands []struct {
	until int
	id    int32
}, island int32, kinds map[int32]string) *Base {
	const w, h = 100, 10
	b := &Base{Width: w, Height: h, Province: make([]int32, w*h), Heights: make([]uint8, w*h),
		Terrain: make([]uint8, w*h), Rivers: make([]uint8, w*h), TerrainPalette: GreyPalette(), RiverPalette: GreyPalette()}
	for p := range b.Province {
		x, y := p%w, p/w
		for _, band := range bands {
			if x < band.until {
				b.Province[p] = band.id
				break
			}
		}
		if island != 0 && x >= 75 && x < 79 && y >= 3 && y < 6 {
			b.Province[p] = island
		}
	}
	for id := int32(1); id <= 12; id++ {
		if kind, ok := kinds[id]; ok {
			b.Definitions = append(b.Definitions, Definition{ID: int(id), RGB: uint32(id) * 0x010101, Kind: kind, Terrain: "ocean"})
		}
	}
	return b
}

type band = struct {
	until int
	id    int32
}

func cutRegionBase(east int32, eastKind string, island int32) (*Base, map[int][]int) {
	kinds := map[int32]string{1: "land", 2: "sea", 3: "sea", 4: "sea", 5: "land", east: eastKind}
	if island != 0 {
		kinds[island] = "land"
	}
	b := stripBase([]band{{20, 1}, {40, 2}, {60, 3}, {70, 5}, {85, 4}, {100, east}}, island, kinds)
	regions := map[int][]int{10: {1}, 20: {2, 3, 4}, 30: {5}, 40: {int(east)}}
	if island != 0 {
		regions[20] = append(regions[20], int(island))
	}
	return b, regions
}

func TestMendSeaRegionsMovesTheCutOffPart(t *testing.T) {
	b, regions := cutRegionBase(6, "sea", 0)
	moves, err := b.MendSeaRegions(regions, map[int]int{1: 100, 5: 101})
	if err != nil {
		t.Fatal(err)
	}
	want := []RegionMove{{From: 20, To: 40, Provinces: []int{4}}}
	if !reflect.DeepEqual(moves, want) {
		t.Fatalf("moves %+v, want %+v", moves, want)
	}
}

func TestMendSeaRegionsTakesAWholeStateAlong(t *testing.T) {
	b, regions := cutRegionBase(6, "sea", 7)
	moves, err := b.MendSeaRegions(regions, map[int]int{1: 100, 5: 101, 7: 102})
	if err != nil {
		t.Fatal(err)
	}
	want := []RegionMove{{From: 20, To: 40, Provinces: []int{4, 7}}}
	if !reflect.DeepEqual(moves, want) {
		t.Fatalf("moves %+v, want %+v", moves, want)
	}
}

func TestMendSeaRegionsRefusesSplittingAState(t *testing.T) {
	b, regions := cutRegionBase(6, "sea", 7)
	_, err := b.MendSeaRegions(regions, map[int]int{1: 100, 5: 101, 7: 100})
	var refused *AddError
	if !errors.As(err, &refused) || refused.Code != "sea_region_state" || refused.Values["state"] != 100 {
		t.Fatalf("err %v", err)
	}
	if !reflect.DeepEqual(refused.Provinces, []int{4, 7}) {
		t.Errorf("provinces %v", refused.Provinces)
	}
}

func TestMendSeaRegionsRefusesAPartWithNoSeaBeside(t *testing.T) {
	b, regions := cutRegionBase(8, "land", 0)
	_, err := b.MendSeaRegions(regions, map[int]int{1: 100, 5: 101, 8: 102})
	var refused *AddError
	if !errors.As(err, &refused) || refused.Code != "sea_region_alone" || refused.Values["region"] != 20 {
		t.Fatalf("err %v", err)
	}
}

func TestMendSeaRegionsLeavesWholeRegions(t *testing.T) {
	b, regions := cutRegionBase(6, "sea", 0)
	regions[20] = []int{2, 3}
	regions[40] = []int{4, 6}
	if moves, err := b.MendSeaRegions(regions, nil); err != nil || moves != nil {
		t.Fatalf("whole: %+v %v", moves, err)
	}

	b2, _ := cutRegionBase(6, "sea", 7)
	if moves, err := b2.MendSeaRegions(map[int][]int{10: {1}, 20: {2, 3, 7}, 30: {5}, 40: {4, 6}}, nil); err != nil || moves != nil {
		t.Fatalf("island: %+v %v", moves, err)
	}
}

func regionsIn(t *testing.T, dirs ...string) map[int][]int {
	t.Helper()
	files := map[string]string{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			files[e.Name()] = filepath.Join(dir, e.Name())
		}
	}
	idPattern := regexp.MustCompile(`\bid\s*=\s*(\d+)`)
	out := map[int][]int{}
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m := idPattern.FindSubmatch(body)
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(string(m[1]))
		out[id] = RegionProvinces(string(body))
	}
	return out
}

func TestBaseGameSeaRegionsAreWhole(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	read := func(name string) []byte {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(root, "map", name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	b, err := LoadBase(read("provinces.bmp"), read("heightmap.bmp"), read("terrain.bmp"), read("rivers.bmp"), read("definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	regions := regionsIn(t, filepath.Join(root, "map", "strategicregions"))
	if len(regions) < 300 {
		t.Fatalf("only %d regions", len(regions))
	}
	if moves, err := b.MendSeaRegions(regions, nil); err != nil || moves != nil {
		t.Fatalf("moves %+v err %v", moves, err)
	}
}
