package mapsketch

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
)

func picture(w, h int, paint func(x, y int) RGB) *Picture {
	p := &Picture{W: w, H: h, Pix: make([]RGB, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p.Pix[y*w+x] = paint(x, y)
		}
	}
	return p
}

var (
	red   = RGB{200, 40, 40}
	green = RGB{90, 170, 70}
	blue  = RGB{40, 80, 170}
	black = RGB{10, 10, 10}
)

func TestPaletteFindsTheMainColoursAndAPlaceInsideEach(t *testing.T) {
	p := picture(400, 200, func(x, y int) RGB {
		if x < 300 {
			return green
		}
		return blue
	})
	swatches := Palette(p, 8)
	if len(swatches) != 2 {
		t.Fatalf("swatches = %+v", swatches)
	}
	if swatches[0].Hex != green.Hex() || swatches[0].Share < 0.7 || swatches[0].Sample.X >= 300 {
		t.Fatalf("first = %+v", swatches[0])
	}
	if swatches[1].Hex != blue.Hex() || swatches[1].NearestTerrain != "ocean" || swatches[1].Sample.X < 300 {
		t.Fatalf("second = %+v", swatches[1])
	}
}

func TestTerrainPlacesTheDrawingOnTheMapsShapeAndFillsLines(t *testing.T) {

	p := picture(200, 400, func(x, y int) RGB {
		switch {
		case x < 20 || x >= 180 || y < 20 || y >= 380:
			return blue
		case y == 200 || y == 201:
			return black
		}
		return green
	})
	result, err := Terrain(p, TerrainOptions{Rules: []Rule{{Colour: blue, Class: "ocean"}, {Colour: green, Class: "forest"}}})
	if err != nil {
		t.Fatal(err)
	}
	pl := result.Placement
	if pl.CanvasW != DefaultCanvasW || pl.CanvasH != DefaultCanvasH {
		t.Fatalf("canvas = %+v", pl)
	}

	if h := int(float64(400) * pl.Scale); h < 950 || h > 970 {
		t.Fatalf("scaled height %d", h)
	}
	if pl.OffsetX != (DefaultCanvasW-int(200*pl.Scale+0.5))/2 {
		t.Fatalf("offset = %+v", pl)
	}
	forest := mapgen.Legend[mapgen.Forest].Color
	sea := mapgen.Legend[mapgen.Sea].Color
	at := func(s Point) RGB {
		c, ok := pl.CanvasOf(s)
		if !ok {
			t.Fatalf("%v is off the drawing", s)
		}
		o := result.Sketch.PixOffset(c.X, c.Y)
		return RGB{result.Sketch.Pix[o], result.Sketch.Pix[o+1], result.Sketch.Pix[o+2]}
	}
	if got := at(Point{100, 100}); got != RGB(forest) {
		t.Fatalf("land = %v", got)
	}
	if got := at(Point{100, 200}); got != RGB(forest) {
		t.Fatalf("the line was not filled from around it: %v", got)
	}
	if got := at(Point{5, 5}); got != RGB(sea) {
		t.Fatalf("sea = %v", got)
	}

	o := result.Sketch.PixOffset(10, 10)
	if got := (RGB{result.Sketch.Pix[o], result.Sketch.Pix[o+1], result.Sketch.Pix[o+2]}); got != RGB(sea) {
		t.Fatalf("margin = %v", got)
	}
	if result.Filled <= 0 || result.Shares["forest"] < 0.5 {
		t.Fatalf("result = %+v", result)
	}
}

func TestTerrainRefusesAnUnknownClass(t *testing.T) {
	p := picture(50, 50, func(int, int) RGB { return green })
	if _, err := Terrain(p, TerrainOptions{Rules: []Rule{{Colour: green, Class: "tundra"}}}); err == nil {
		t.Fatal("tundra accepted")
	}
}

func TestRegionsFollowColoursAndSplitSameColourByDistance(t *testing.T) {

	p := picture(300, 100, func(x, y int) RGB {
		switch {
		case x == 100:
			return black
		case x < 200:
			return red
		}
		return blue
	})
	seeds := []Seed{{At: Point{30, 50}, Tag: "AAA"}, {At: Point{170, 50}, Tag: "bbb"}, {At: Point{260, 50}, Tag: "CCC"}}
	result, err := Regions(p, RegionOptions{Seeds: seeds})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(result.Tags, ",") != "AAA,BBB,CCC" {
		t.Fatalf("tags = %v", result.Tags)
	}
	if got := result.TagAt(Point{60, 50}); got != 1 {
		t.Fatalf("(60,50) = %d", got)
	}
	if got := result.TagAt(Point{150, 50}); got != 2 {
		t.Fatalf("(150,50) = %d", got)
	}

	if got := result.TagAt(Point{205, 50}); got != 3 {
		t.Fatalf("(205,50) = %d", got)
	}
	warned := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "AAA") && strings.Contains(w, "BBB") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("no same-colour warning: %v", result.Warnings)
	}
	for _, r := range result.Regions {
		if r.Parts != 1 || r.Share < 0.25 {
			t.Fatalf("region = %+v", r)
		}
	}
}

func TestRegionsLeaveWaterUnownedAndGiveIslandsToTheNearestCountry(t *testing.T) {

	p := picture(300, 100, func(x, y int) RGB {
		if x < 150 || (x >= 220 && x < 240 && y >= 40 && y < 60) {
			return green
		}
		return blue
	})
	land := func(at Point) bool { return p.Pix[at.Y*p.W+at.X] == green }
	result, err := Regions(p, RegionOptions{Seeds: []Seed{{At: Point{20, 50}, Tag: "AAA"}, {At: Point{120, 10}, Tag: "BBB"}}, Land: land})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.TagAt(Point{200, 50}); got != 0 {
		t.Fatalf("sea owned by %d", got)
	}
	if got := result.TagAt(Point{230, 50}); got == 0 {
		t.Fatal("island unowned")
	}
	pl := Place(Rect{W: 300, H: 100}, 600, 200, 0)
	owners := result.Owners(pl)
	o := owners.PixOffset(460, 100)
	if owners.Pix[o] == 0 {
		t.Fatal("owners picture misses the island")
	}
}

func TestPlacementRoundTrip(t *testing.T) {
	pl := Place(Rect{X: 10, Y: 20, W: 300, H: 600}, DefaultCanvasW, DefaultCanvasH, 0.03)
	for _, s := range []Point{{10, 20}, {150, 300}, {309, 619}} {
		c, ok := pl.CanvasOf(s)
		if !ok {
			t.Fatalf("%v off", s)
		}
		back, ok := pl.SourceOf(c.X, c.Y)
		if !ok || abs(back.X-s.X) > 1 || abs(back.Y-s.Y) > 1 {
			t.Fatalf("%v → %v → %v", s, c, back)
		}
	}
	if _, ok := pl.SourceOf(0, 0); ok {
		t.Fatal("corner of the canvas is in the drawing")
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestGridPreviewKeepsTheLongSide(t *testing.T) {
	p := picture(2000, 1000, func(int, int) RGB { return green })
	out := GridPreview(p, 1024, []Mark{{At: Point{1000, 500}, Label: "GER", Colour: red}})
	if out.Bounds().Dx() != 1024 || out.Bounds().Dy() != 512 {
		t.Fatalf("size = %v", out.Bounds())
	}
}

func TestRegionsWarnOnlyOfSameColourNeighbours(t *testing.T) {
	p := picture(300, 100, func(x, y int) RGB {
		if x >= 100 && x < 200 {
			return blue
		}
		return red
	})
	result, err := Regions(p, RegionOptions{Seeds: []Seed{{At: Point{30, 50}, Tag: "AAA"}, {At: Point{150, 50}, Tag: "BBB"}, {At: Point{270, 50}, Tag: "CCC"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings %v", result.Warnings)
	}
}

func TestRegionsMoveASeedOnWaterToTheNearestLand(t *testing.T) {

	p := picture(300, 100, func(x, y int) RGB {
		switch {
		case x < 150:
			return red
		case x < 250:
			return green
		}
		return blue
	})
	land := func(at Point) bool { return p.Pix[at.Y*p.W+at.X] != blue }
	result, err := Regions(p, RegionOptions{Seeds: []Seed{{At: Point{50, 50}, Tag: "AAA"}, {At: Point{256, 50}, Tag: "BBB"}}, Land: land})
	if err != nil {
		t.Fatal(err)
	}
	if at := result.Seeds[1].At; at.X >= 250 || at.X < 240 {
		t.Fatalf("seed not moved to the coast: %v", at)
	}
	if got := result.Regions[1].Colours; len(got) != 1 || got[0] != green.Hex() {
		t.Fatalf("BBB's colour %v, want the land's %s", got, green.Hex())
	}
	if got := result.TagAt(Point{200, 50}); got != 2 {
		t.Fatalf("green land owned by %d", got)
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "挪到了最近的陆地") {
		t.Fatalf("warnings %v", result.Warnings)
	}

	if result.Seeds[0].At != (Point{50, 50}) {
		t.Fatalf("land seed moved: %v", result.Seeds[0].At)
	}
}

func TestRegionsMoveASeedOffAnIslandOntoThatIsland(t *testing.T) {

	p := picture(300, 100, func(x, y int) RGB {
		switch {
		case x < 190:
			return red
		case x >= 200 && x < 210 && y >= 45 && y < 55:
			return green
		}
		return blue
	})
	land := func(at Point) bool { return p.Pix[at.Y*p.W+at.X] != blue }
	result, err := Regions(p, RegionOptions{Seeds: []Seed{{At: Point{50, 50}, Tag: "AAA"}, {At: Point{196, 50}, Tag: "BBB"}}, Land: land})
	if err != nil {
		t.Fatal(err)
	}
	if at := result.Seeds[1].At; at.X < 200 || at.X >= 210 || at.Y < 45 || at.Y >= 55 {
		t.Fatalf("island seed moved to %v", at)
	}
	if got := result.TagAt(Point{205, 50}); got != 2 {
		t.Fatalf("island owned by %d", got)
	}
}
