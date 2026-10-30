package mapgen

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/bmp"
)

func testBase() *Base {
	const w, h = 256, 256
	b := &Base{Width: w, Height: h, Province: make([]int32, w*h), Heights: make([]uint8, w*h),
		Terrain: make([]uint8, w*h), Rivers: make([]uint8, w*h), TerrainPalette: GreyPalette(), RiverPalette: GreyPalette()}
	for p := range b.Province {
		x, y := p%w, p/w
		switch {
		case x >= 20 && x < 30 && y >= 40 && y < 50:
			b.Province[p] = 5
		case x < 60:
			b.Province[p] = 1
		case x < 130:
			b.Province[p] = 2
		case x < 252:
			b.Province[p] = 3
		default:
			b.Province[p] = 4
		}
		if b.Province[p] == 1 {
			b.Heights[p], b.Terrain[p], b.Rivers[p] = 100, 0, RiverLand
		} else {
			b.Heights[p], b.Terrain[p], b.Rivers[p] = SeaHeight, 15, RiverWater
		}
	}
	b.Definitions = []Definition{
		{ID: 1, RGB: 0x102030, Kind: "land", Coastal: true, Terrain: "plains", Continent: 2},
		{ID: 2, RGB: 0x405060, Kind: "sea", Coastal: true, Terrain: "ocean"},
		{ID: 3, RGB: 0x708090, Kind: "sea", Coastal: false, Terrain: "ocean"},
		{ID: 4, RGB: 0xa0b0c0, Kind: "sea", Coastal: false, Terrain: "ocean"},
		{ID: 5, RGB: 0xd0e0f0, Kind: "lake", Terrain: "lakes"},
	}
	return b
}

func ignoreFixtureBoxes(list []Finding) []Finding {
	var out []Finding
	for _, f := range list {
		if f.Code == "province_box" && f.Count <= 4 {
			continue
		}
		out = append(out, f)
	}
	return out
}

func sketchOf(w, h int, paint func(x, y int) Class) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := entryOf(paint(x, y)).Color
			img.SetRGBA(x, y, color.RGBA{c[0], c[1], c[2], 255})
		}
	}
	return img
}

func island(w, h int) image.Image {
	return sketchOf(w, h, func(x, y int) Class {
		dx, dy := (float64(x)-float64(w)/2)/(float64(w)*0.42), (float64(y)-float64(h)/2)/(float64(h)*0.4)
		if dx*dx+dy*dy > 1 {
			return Sea
		}
		if dx > 0.3 && dx < 0.6 {
			return Mountain
		}
		return Plains
	})
}

func mapHash(b *Base) string {
	sum := sha256.New()
	for _, id := range b.Province {
		_ = binary.Write(sum, binary.LittleEndian, id)
	}
	sum.Write(b.Heights)
	sum.Write(b.Terrain)
	return string(sum.Sum(nil))
}

func TestAddLandKeepsEveryBaseProvince(t *testing.T) {
	b := testBase()
	before := append([]int32(nil), b.Province...)
	add, err := b.AddLand(island(40, 20), AddOptions{Place: image.Rect(80, 25, 180, 75), Seed: 5, LandProvincePixels: 150})
	if err != nil {
		t.Fatal(err)
	}
	if add.Ignored != 0 {
		t.Errorf("ignored %d", add.Ignored)
	}
	count := map[int32]int{}
	for p, id := range b.Province {
		count[id]++
		if id <= 5 {
			if id != before[p] {
				t.Fatalf("pixel %d moved from %d to %d", p, before[p], id)
			}
			continue
		}
		if before[p] != 2 && before[p] != 3 {
			t.Fatalf("new land at %d was province %d, not sea", p, before[p])
		}
		if b.Heights[p] < LandHeight || b.Rivers[p] != RiverLand {
			t.Fatalf("new land at %d: height %d river %d", p, b.Heights[p], b.Rivers[p])
		}
	}
	for id := int32(1); id <= 5; id++ {
		if count[id] == 0 {
			t.Errorf("base province %d is gone", id)
		}
	}
	if add.Covered[2] == 0 || add.Covered[3] == 0 || add.Covered[4] != 0 {
		t.Errorf("covered %v", add.Covered)
	}
	if got := add.NowCoastal; len(got) != 1 || got[0] != 3 {
		t.Errorf("now coastal %v, want [3] (2 already was)", got)
	}
	land := 0
	inState := map[int]int{}
	for k, st := range add.States {
		found := false
		for _, id := range st.Provinces {
			inState[id]++
			found = found || id == st.Capital
		}
		if !found {
			t.Errorf("state %d's capital %d is not one of its provinces", k, st.Capital)
		}
	}
	first := add.Provinces[0].ID
	if first != 6 {
		t.Errorf("first new id %d, want 6", first)
	}
	mountain := false
	for i, pr := range add.Provinces {
		if pr.ID != first+i || count[int32(pr.ID)] != pr.Pixels || pr.Pixels == 0 {
			t.Errorf("province %+v (%d pixels on the map)", pr, count[int32(pr.ID)])
		}
		if pr.Kind == "land" {
			land++
			if inState[pr.ID] != 1 {
				t.Errorf("land province %d is in %d states", pr.ID, inState[pr.ID])
			}
			if pr.Continent != 2 {
				t.Errorf("province %d continent %d, want the nearest land's 2", pr.ID, pr.Continent)
			}
			mountain = mountain || pr.Terrain == "mountain"
		}
	}
	if land < 5 || len(add.States) < 1 || !mountain {
		t.Errorf("%d land provinces, %d states, mountain %t", land, len(add.States), mountain)
	}
	findings := ignoreFixtureBoxes(Check(b.CheckInput(nil)))
	if len(findings) != 0 {
		t.Errorf("findings: %+v", findings)
	}

	again := testBase()
	if _, err := again.AddLand(island(40, 20), AddOptions{Place: image.Rect(80, 25, 180, 75), Seed: 5, LandProvincePixels: 150}); err != nil {
		t.Fatal(err)
	}
	if mapHash(again) != mapHash(b) {
		t.Error("the same sketch, place and seed gave another map")
	}
}

func TestAddLandLakesAndRefusals(t *testing.T) {
	b := testBase()
	ring := sketchOf(50, 50, func(x, y int) Class {
		d := math.Hypot(float64(x)-24.5, float64(y)-24.5)
		if d >= 10 && d <= 22 {
			return Forest
		}
		return Sea
	})
	add, err := b.AddLand(ring, AddOptions{Place: image.Rect(70, 25, 120, 75), Seed: 2, LandProvincePixels: 120})
	if err != nil {
		t.Fatal(err)
	}
	lakes := 0
	for _, pr := range add.Provinces {
		if pr.Kind == "lake" {
			lakes++
			if add.CutFrom[pr.ID] != 2 {
				t.Errorf("lake %d cut from %d", pr.ID, add.CutFrom[pr.ID])
			}
			for _, p := range add.Pixels(pr.ID) {
				if b.Terrain[p] != entryOf(Lake).Terrain {
					t.Fatalf("lake pixel terrain %d", b.Terrain[p])
				}
			}
		}
	}
	if lakes != 1 {
		t.Errorf("%d lakes, want 1", lakes)
	}
	if errs := Errors(ignoreFixtureBoxes(Check(b.CheckInput(nil)))); len(errs) != 0 {
		t.Errorf("errors: %+v", errs)
	}

	over := testBase()
	add, err = over.AddLand(island(40, 20), AddOptions{Place: image.Rect(30, 20, 110, 80), Seed: 1, LandProvincePixels: 150})
	if err != nil {
		t.Fatal(err)
	}
	if add.Ignored == 0 {
		t.Error("land painted over province 1 was not counted as left out")
	}

	_, err = testBase().AddLand(sketchOf(10, 10, func(int, int) Class { return Plains }), AddOptions{Place: image.Rect(250, 0, 256, 256), Seed: 1})
	var refused *AddError
	if !errors.As(err, &refused) || !strings.Contains(refused.Message, "#4") {
		t.Errorf("swallowing sea province 4: %v", err)
	}
	if _, err := testBase().AddLand(sketchOf(10, 10, func(int, int) Class { return Sea }), AddOptions{Place: image.Rect(80, 20, 120, 60)}); !errors.Is(err, ErrNoLand) {
		t.Errorf("all-sea sketch: %v", err)
	}
	if _, err := testBase().AddLand(island(10, 10), AddOptions{Place: image.Rect(240, 20, 280, 60)}); !errors.As(err, &refused) {
		t.Errorf("off the map: %v", err)
	}
}

func TestAddLandPositions(t *testing.T) {
	b := testBase()
	add, err := b.AddLand(island(40, 20), AddOptions{Place: image.Rect(80, 25, 180, 75), Seed: 5, LandProvincePixels: 150})
	if err != nil {
		t.Fatal(err)
	}
	stateIDs := make([]int, len(add.States))
	stateOf := map[int]int{}
	for k, st := range add.States {
		stateIDs[k] = 900 + k
		for _, id := range st.Provinces {
			stateOf[id] = 900 + k
		}
	}
	pos := b.Positions(add, stateIDs, 400, 5)
	provinceAt := func(fields []string) int {
		x, _ := strconv.ParseFloat(fields[0], 64)
		z, _ := strconv.ParseFloat(fields[2], 64)
		return int(b.Province[(b.Height-1-int(z))*b.Width+int(x)])
	}
	kinds := map[string]int{}
	for _, line := range pos.Buildings {
		f := strings.Split(line, ";")
		state, _ := strconv.Atoi(f[0])
		sea, _ := strconv.Atoi(f[6])
		kinds[f[1]]++
		at := provinceAt(f[2:5])
		if f[1] == "floating_harbor" {
			if at != sea {
				t.Errorf("%s at province %d, names sea %d", line, at, sea)
			}
			continue
		}
		if stateOf[at] != state {
			t.Errorf("%s: at province %d of state %d", line, at, stateOf[at])
		}
		if f[1] == "naval_base_spawn" && (sea != 2 && sea != 3) {
			t.Errorf("%s names sea %d", line, sea)
		}
	}
	land := 0
	for _, pr := range add.Provinces {
		if pr.Kind == "land" {
			land++
		}
	}
	if kinds["arms_factory"] != 6*len(add.States) || kinds["supply_node"] != land || kinds["naval_base_spawn"] == 0 || kinds["dockyard"] == 0 {
		t.Errorf("buildings %v for %d states, %d land provinces", kinds, len(add.States), land)
	}
	if len(pos.UnitStacks) != len(unitStackTypes)*len(add.Provinces) {
		t.Errorf("%d unit stacks", len(pos.UnitStacks))
	}
	for _, line := range pos.UnitStacks {
		f := strings.Split(line, ";")
		id, _ := strconv.Atoi(f[0])
		if at := provinceAt(f[2:5]); at != id {
			t.Errorf("%s at province %d", line, at)
		}
	}
	if len(pos.SupplyNodes) != len(add.States) || len(pos.Weather) != 2 || !strings.HasPrefix(pos.Weather[0], "400;") {
		t.Errorf("supply %v weather %v", pos.SupplyNodes, pos.Weather)
	}

	centre := add.Pixels(add.Provinces[0].ID)[0]
	x, z := int(centre)%b.Width, b.Height-1-int(centre)/b.Width
	text := "2;0;" + strconv.Itoa(x) + ".00;8.90;" + strconv.Itoa(z) + ".00;0.00;0.54\n1;0;10.00;10.00;50.00;0.00;0.30\n"
	moved, n := b.MoveCovered(text, add, true)
	if n != 1 {
		t.Fatalf("moved %d: %s", n, moved)
	}
	f := strings.Split(strings.Split(moved, "\n")[0], ";")
	if at := provinceAt(f[2:5]); at != 2 || !strings.HasSuffix(strings.Split(moved, "\n")[1], "0.30") {
		t.Errorf("moved to province %d: %q", at, moved)
	}
}

func TestRegionFiles(t *testing.T) {
	text := "strategic_region={\n\tid=95\n\tname=\"STRATEGICREGION_95\"\n\tprovinces={\n\t\t54 262 459 \n\t}\n\tnaval_terrain=water_deep_ocean\n\tweather={\n\t\tperiod={\n\t\t\tbetween={ 0.0 30.0 }\n\t\t}\n\t}\n}\n"
	if got := RegionProvinces(text); len(got) != 3 || got[2] != 459 {
		t.Errorf("provinces %v", got)
	}
	added := AddToRegion(text, []int{13413, 13414})
	if got := RegionProvinces(added); len(got) != 5 || got[4] != 13414 {
		t.Errorf("after adding: %v\n%s", got, added)
	}
	weather := WeatherBlock(text)
	if !strings.HasPrefix(weather, "weather={") || !strings.HasSuffix(weather, "}") || strings.Count(weather, "{") != strings.Count(weather, "}") {
		t.Errorf("weather %q", weather)
	}
	region := RegionFile(305, []int{13415, 13413}, weather)
	if RegionProvinces(region)[0] != 13413 || !strings.Contains(region, "id=305") || !strings.Contains(region, "between={ 0.0 30.0 }") || strings.Count(region, "{") != strings.Count(region, "}") {
		t.Errorf("region file:\n%s", region)
	}
	if got := AppendLines("a\r\nb", []string{"c"}); got != "a\r\nb\r\nc" {
		t.Errorf("append %q, want no last newline, as the text had none", got)
	}
	if got := AppendLines("a\n", []string{"b", "c"}); got != "a\nb\nc\n" {
		t.Errorf("append %q, want the last newline kept", got)
	}
}

func TestAddLandOnTheBaseGameMap(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(root, "map", name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	started := time.Now()
	b, err := LoadBase(read("provinces.bmp"), read("heightmap.bmp"), read("terrain.bmp"), read("rivers.bmp"), read("definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("load %.1fs", time.Since(started).Seconds())

	original, err := bmp.Decode(bytes.NewReader(read("provinces.bmp")))
	if err != nil {
		t.Fatal(err)
	}
	written := b.ProvincesImage()
	for y := 0; y < b.Height; y += 3 {
		for x := 0; x < b.Width; x++ {
			r, g, bl, _ := original.At(x, y).RGBA()
			if c := written.RGBAAt(x, y); c.R != uint8(r>>8) || c.G != uint8(g>>8) || c.B != uint8(bl>>8) {
				t.Fatalf("(%d,%d) written %v, game %d,%d,%d", x, y, c, r>>8, g>>8, bl>>8)
			}
		}
	}
	if again := ParseDefinitions(b.DefinitionCSV()); len(again) != len(b.Definitions) {
		t.Fatalf("definition.csv round trip: %d rows of %d", len(again), len(b.Definitions))
	}
	baseFindings := map[string]int{}
	for _, f := range Check(b.CheckInput(nil)) {
		baseFindings[f.Code] = f.Count
	}
	firstNew := b.MaxProvinceID() + 1
	if len(b.Definitions) < 13000 || string(b.DefinitionCSV()) == "" {
		t.Fatalf("%d definitions", len(b.Definitions))
	}
	started = time.Now()
	add, err := b.AddLand(island(60, 30), AddOptions{Place: image.Rect(200, 1100, 320, 1160), Seed: 9})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("add %.1fs: %d provinces, %d states, covered %v, ignored %d", time.Since(started).Seconds(), len(add.Provinces), len(add.States), add.Covered, add.Ignored)
	if add.Provinces[0].ID != firstNew || add.Ignored != 0 {
		t.Errorf("first id %d, ignored %d", add.Provinces[0].ID, add.Ignored)
	}
	for _, f := range Check(b.CheckInput(nil)) {
		if f.Severity == "error" || f.Count > baseFindings[f.Code] {
			t.Errorf("%s %s: %d (base %d) %v", f.Severity, f.Code, f.Count, baseFindings[f.Code], f.At[:min(5, len(f.At))])
		}
	}
	for _, img := range []image.Image{b.ProvincesImage(), b.HeightImage(), b.TerrainImage(), b.RiversImage()} {
		if _, err := EncodeBMP(img); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRemoveFromRegion(t *testing.T) {
	text := "strategic_region={\n\tid=95\n\tprovinces={\n\t\t54 262 2620 459 \n\t}\n}\n"
	got := RemoveFromRegion(text, 262)
	if ids := RegionProvinces(got); len(ids) != 3 || ids[0] != 54 || ids[1] != 2620 || ids[2] != 459 {
		t.Errorf("after removing 262: %v\n%s", ids, got)
	}
	if got := RemoveFromRegion(text, 26); got != text {
		t.Errorf("a number inside another changed the file:\n%s", got)
	}
	if ids := RegionProvinces(RemoveFromRegion(text, 459)); len(ids) != 3 || ids[2] != 2620 {
		t.Errorf("last one: %v", ids)
	}
	rows := []Definition{{ID: 2, RGB: 0x010203, Kind: "sea", Terrain: "ocean"}, {ID: 1, RGB: 0x040506, Kind: "land", Coastal: true, Terrain: "plains", Continent: 1}}
	if got := string(DefinitionsCSV(rows)); got != "0;0;0;0;land;false;unknown;0\n1;4;5;6;land;true;plains;1\n2;1;2;3;sea;false;ocean;0\n" {
		t.Errorf("definitions:\n%s", got)
	}
}

func TestRegionListsWithComments(t *testing.T) {
	text := "strategic_region={\n\tid=13\n\tprovinces={\n\t\t11 123 # was 1014, see { this }\n\t\t456 #789\n\t}\n\tweather={ period={ between={ 0.0 30.0 } } }\n}\n"
	if got := RegionProvinces(text); len(got) != 3 || got[0] != 11 || got[1] != 123 || got[2] != 456 {
		t.Errorf("provinces %v", got)
	}
	added := AddToRegion(text, []int{13420})
	if got := RegionProvinces(added); len(got) != 4 || got[3] != 13420 {
		t.Errorf("after adding %v\n%s", got, added)
	}
	removed := RemoveFromRegion(added, 456)
	if got := RegionProvinces(removed); len(got) != 3 || got[2] != 13420 || !strings.Contains(removed, "#789") {
		t.Errorf("after removing 456 %v\n%s", got, removed)
	}
	if RemoveFromRegion(text, 789) != text {
		t.Error("a number in a comment was removed")
	}
	if !strings.HasPrefix(WeatherBlock(text), "weather={") {
		t.Errorf("weather %q", WeatherBlock(text))
	}
}
