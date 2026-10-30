package worldmap

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/mapgen"
)

func seaCutWorld(t *testing.T) (*landFileSet, *mapgen.Base) {
	t.Helper()
	region := func(id int, provinces string) string {
		return "strategic_region={\n\tid=" + strconv.Itoa(id) + "\n\tname=\"STRATEGICREGION_" + strconv.Itoa(id) + "\"\n\tprovinces={\n\t\t" + provinces + " \n\t}\n}\n"
	}
	index, err := guirender.NewIndex(map[string][]byte{
		"map/definition.csv":                   []byte("0;0;0;0;land;false;unknown;0\n1;1;1;1;land;true;plains;1\n2;2;2;2;sea;true;ocean;0\n3;3;3;3;sea;false;ocean;0\n4;4;4;4;sea;true;ocean;0\n6;6;6;6;sea;false;ocean;0\n"),
		"map/strategicregions/10-West.txt":     []byte(region(10, "1")),
		"map/strategicregions/20-Cut Sea.txt":  []byte(region(20, "2 3 4")),
		"map/strategicregions/40-East Sea.txt": []byte(region(40, "6")),
		"history/states/100-West.txt":          []byte("state={\n\tid=100\n\tname=\"STATE_100\"\n\tprovinces={ 1 }\n}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	east := "map/strategicregions/40-East Sea.txt"
	files := &landFileSet{
		index: index,
		raw: map[string]RawFile{
			strings.ToLower(east): {DestPath: east, Content: region(40, "6"), RawKind: "verbatim_override"},
		},
		ours: map[string]bool{},
	}
	const w, h = 100, 10
	b := &mapgen.Base{Width: w, Height: h, Province: make([]int32, w*h), Heights: make([]uint8, w*h),
		Terrain: make([]uint8, w*h), Rivers: make([]uint8, w*h), TerrainPalette: mapgen.GreyPalette(), RiverPalette: mapgen.GreyPalette()}
	for p := range b.Province {
		switch x := p % w; {
		case x < 20:
			b.Province[p] = 1
		case x < 40:
			b.Province[p] = 2
		case x < 60:
			b.Province[p] = 3
		case x < 70:
			b.Province[p] = 5
		case x < 85:
			b.Province[p] = 4
		default:
			b.Province[p] = 6
		}
	}
	for id, kind := range map[int]string{1: "land", 2: "sea", 3: "sea", 4: "sea", 5: "land", 6: "sea"} {
		b.Definitions = append(b.Definitions, mapgen.Definition{ID: id, RGB: uint32(id) * 0x010101, Kind: kind, Terrain: "ocean"})
	}
	return files, b
}

func TestLandThatCutsASeaRegionMendsIt(t *testing.T) {
	files, base := seaCutWorld(t)
	regions, regionFiles, err := files.projectRegions()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(regions[20], []int{2, 3, 4}) || regionFiles[40] != "map/strategicregions/40-East Sea.txt" {
		t.Fatalf("regions %v files %v", regions, regionFiles)
	}
	if got := files.stateOf(); got[1] != 100 {
		t.Fatalf("stateOf %v", got)
	}

	newRegion := files.index.MaxStrategicRegionID() + 1
	add := &mapgen.Addition{Provinces: []mapgen.Province{{ID: 5, Kind: "land"}}, CutFrom: map[int]int{}}
	moves, err := landSeaRegionMoves(files, base, add, newRegion)
	if err != nil {
		t.Fatal(err)
	}
	if want := []mapgen.RegionMove{{From: 20, To: 40, Provinces: []int{4}}}; !reflect.DeepEqual(moves, want) {
		t.Fatalf("moves %+v, want %+v", moves, want)
	}

	newFile := "map/strategicregions/41-New Land.txt"
	texts := map[string]string{newFile: mapgen.RegionFile(newRegion, []int{5}, "")}
	err = applySeaRegionMoves(files, moves, texts, func(id int) (string, bool) {
		if id == newRegion {
			return newFile, true
		}
		return files.index.StrategicRegionFile(id)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := mapgen.RegionProvinces(texts["map/strategicregions/20-Cut Sea.txt"]); !reflect.DeepEqual(got, []int{2, 3}) {
		t.Errorf("region 20 now %v", got)
	}
	if got := mapgen.RegionProvinces(texts["map/strategicregions/40-East Sea.txt"]); !reflect.DeepEqual(got, []int{6, 4}) {
		t.Errorf("region 40 now %v", got)
	}
	if got := mapgen.RegionProvinces(texts[newFile]); !reflect.DeepEqual(got, []int{5}) {
		t.Errorf("the new land's region now %v", got)
	}

	regions[20], regions[40] = []int{2, 3}, []int{6, 4}
	regions[newRegion] = []int{5}
	if again, err := base.MendSeaRegions(regions, files.stateOf()); err != nil || again != nil {
		t.Fatalf("after: %+v %v", again, err)
	}
}

func TestSeaRegionMoveIntoAnUnreadableFileIsRefused(t *testing.T) {
	files, _ := seaCutWorld(t)
	east := "map/strategicregions/40-East Sea.txt"
	files.raw[strings.ToLower(east)] = RawFile{DestPath: east, IsBinary: true, RawKind: "verbatim_override"}
	err := applySeaRegionMoves(files, []mapgen.RegionMove{{From: 20, To: 40, Provinces: []int{4}}}, map[string]string{}, files.index.StrategicRegionFile)
	if problem := mapProblemOf(err); problem == nil || problem.Code != problemOwnMap || !strings.Contains(problem.Detail, east) {
		t.Fatalf("err %v", err)
	}
}

func provinceEditWorld(t *testing.T) (*landFileSet, *mapgen.Base, []mapgen.Definition) {
	t.Helper()
	region := func(id int, provinces string) []byte {
		return []byte("strategic_region={\n\tid=" + strconv.Itoa(id) + "\n\tname=\"STRATEGICREGION_" + strconv.Itoa(id) + "\"\n\tprovinces={\n\t\t" + provinces + " \n\t}\n}\n")
	}
	index, err := guirender.NewIndex(map[string][]byte{
		"map/definition.csv":               []byte("0;0;0;0;land;false;unknown;0\n1;1;1;1;land;true;plains;1\n2;2;2;2;sea;true;ocean;0\n3;3;3;3;sea;false;ocean;0\n4;4;4;4;sea;true;ocean;0\n5;5;5;5;land;true;plains;1\n"),
		"map/strategicregions/10-West.txt": region(10, "1"),
		"map/strategicregions/20-Sea.txt":  region(20, "2 3 4"),
		"map/strategicregions/30-East.txt": region(30, "5"),
		"history/states/100-West.txt":      []byte("state={\n\tid=100\n\tname=\"STATE_100\"\n\tprovinces={ 1 }\n}\n"),
		"history/states/200-East.txt":      []byte("state={\n\tid=200\n\tname=\"STATE_200\"\n\tprovinces={ 5 }\n}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	files := &landFileSet{index: index, raw: map[string]RawFile{}, ours: map[string]bool{}}
	const w, h = 100, 10
	b := &mapgen.Base{Width: w, Height: h, Province: make([]int32, w*h), Heights: make([]uint8, w*h),
		Terrain: make([]uint8, w*h), Rivers: make([]uint8, w*h), TerrainPalette: mapgen.GreyPalette(), RiverPalette: mapgen.GreyPalette()}
	for p := range b.Province {
		b.Province[p] = int32(p%w/20 + 1)
	}
	var defs []mapgen.Definition
	for id, kind := range map[int]string{1: "land", 2: "sea", 3: "sea", 4: "sea", 5: "land"} {
		defs = append(defs, mapgen.Definition{ID: id, RGB: uint32(id) * 0x010101, Kind: kind, Terrain: "ocean"})
	}
	b.Definitions = defs
	return files, b, defs
}

func moveRegion(t *testing.T, files *landFileSet, province, to int) map[string]string {
	t.Helper()
	_, fromFile, _ := files.index.StrategicRegionOf(province)
	toFile, _ := files.index.StrategicRegionFile(to)
	from, _, err := files.regionText(fromFile)
	if err != nil {
		t.Fatal(err)
	}
	target, _, err := files.regionText(toFile)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]string{fromFile: mapgen.RemoveFromRegion(from, province), toFile: mapgen.AddToRegion(target, []int{province})}
}

func TestProvinceEditKeepsTheGamesLoadingRules(t *testing.T) {
	files, base, defs := provinceEditWorld(t)
	at := func(v int) *int { return &v }

	if problem, err := provinceEditLoadRules(files, base, defs, provinceEdit{Province: 1, State: at(200)}, nil); err != nil || !strings.Contains(problemDetail(problem), "同一个战略区") {
		t.Errorf("state 200 is in region 30, province 1 in 10: %q %v", problem, err)
	}
	if problem, err := provinceEditLoadRules(files, base, defs, provinceEdit{Province: 1, State: at(200), Region: at(30)}, moveRegion(t, files, 1, 30)); err != nil || problem != nil {
		t.Errorf("moved to the state's region along with it: %q %v", problem, err)
	}
	if problem, err := provinceEditLoadRules(files, base, defs, provinceEdit{Province: 3, Region: at(10)}, moveRegion(t, files, 3, 10)); err != nil || !strings.Contains(problemDetail(problem), "断成几块") {
		t.Errorf("sea 3 out of region 20 leaves 2 and 4 apart: %q %v", problem, err)
	}
	if problem, err := provinceEditLoadRules(files, base, defs, provinceEdit{Province: 4, Region: at(30)}, moveRegion(t, files, 4, 30)); err != nil || problem != nil {
		t.Errorf("sea 4 into region 30 beside land 5, region 20 still whole: %q %v", problem, err)
	}
}
