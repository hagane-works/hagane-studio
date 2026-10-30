package mapgen

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"image"
	"image/color"
	"testing"
)

func legendColor(c Class) color.RGBA {
	e := Legend[c]
	return color.RGBA{e.Color[0], e.Color[1], e.Color[2], 255}
}

func testSketch() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	fill := func(c Class, inside func(x, y float64) bool) {
		for y := 0; y < 100; y++ {
			for x := 0; x < 200; x++ {
				if inside(float64(x), float64(y)) {
					img.SetRGBA(x, y, legendColor(c))
				}
			}
		}
	}
	ellipse := func(cx, cy, rx, ry float64) func(x, y float64) bool {
		return func(x, y float64) bool {
			dx, dy := (x-cx)/rx, (y-cy)/ry
			return dx*dx+dy*dy <= 1
		}
	}
	rect := func(x0, y0, x1, y1 float64) func(x, y float64) bool {
		return func(x, y float64) bool { return x >= x0 && x < x1 && y >= y0 && y < y1 }
	}
	fill(Sea, func(x, y float64) bool { return true })
	fill(Plains, ellipse(90, 50, 70, 35))
	fill(Forest, ellipse(60, 40, 15, 10))
	fill(Mountain, rect(100, 28, 120, 58))
	fill(Hills, ellipse(132, 62, 12, 9))
	fill(Desert, rect(126, 30, 146, 46))
	fill(Marsh, ellipse(52, 66, 10, 6))
	fill(Jungle, ellipse(84, 73, 10, 6))
	fill(Sea, ellipse(108, 72, 5, 4))
	fill(Plains, ellipse(180, 30, 4, 3))
	fill(Plains, ellipse(190, 44, 3, 3))
	img.SetRGBA(5, 95, legendColor(Plains))
	return img
}

var testOptions = Options{Width: 1024, Height: 512, Seed: 7, LakeMaxPixels: 5000}

func generateTest(t *testing.T, options Options) *Map {
	t.Helper()
	m, err := Generate(testSketch(), options)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func at(m *Map, sx, sy float64) int {
	x := int((sx + 0.5) * float64(m.Width) / 200)
	y := int((sy + 0.5) * float64(m.Height) / 100)
	return y*m.Width + x
}

func TestLegendColoursClassifyAsThemselves(t *testing.T) {
	for _, entry := range Legend {
		if got := nearestClass(entry.Color[0], entry.Color[1], entry.Color[2]); got != entry.Class {
			t.Errorf("%s classified as %v", entry.Key, got)
		}
	}

	if got := nearestClass(45, 120, 55); got != Forest {
		t.Errorf("off forest: %v", got)
	}
}

func TestGenerateMakesAMapThatBreaksNoRule(t *testing.T) {
	m := generateTest(t, testOptions)
	if findings := Check(m.CheckInput()); len(findings) > 0 {
		t.Fatalf("findings: %+v", findings)
	}
	summary := m.Summary()
	if summary["land"] < 50 || summary["sea"] < 10 || summary["lake"] < 1 {
		t.Errorf("provinces: %v", summary)
	}
	province := func(sx, sy float64) Province { return m.Provinces[m.Province[at(m, sx, sy)]] }

	if m.Classes[at(m, 5, 95)] != Sea {
		t.Error("the slip of the brush became land")
	}
	if island, coast := province(180, 30), province(90, 50); island.Kind != "land" || island.ID == coast.ID || !island.Coastal {
		t.Errorf("island %+v", island)
	}
	if lake := province(108, 72); lake.Kind != "lake" || lake.Terrain != "lakes" {
		t.Errorf("lake %+v", lake)
	}
	for _, want := range []struct {
		sx, sy  float64
		terrain string
	}{{60, 40, "forest"}, {110, 43, "mountain"}, {136, 38, "desert"}, {52, 66, "marsh"}, {84, 73, "jungle"}, {132, 64, "hills"}} {
		if got := province(want.sx, want.sy).Terrain; got != want.terrain {
			t.Errorf("(%v,%v) terrain %s, want %s", want.sx, want.sy, got, want.terrain)
		}
	}
	if inland := province(90, 50); inland.Coastal {
		t.Errorf("the middle of the continent is coastal: %+v", inland)
	}

	for p, class := range m.Classes {
		if class.Land() && m.Heights[p] < LandHeight || !class.Land() && m.Heights[p] != SeaHeight {
			t.Fatalf("pixel %d (%v) height %d", p, class, m.Heights[p])
		}
	}
	mountain, hills, plains := m.Heights[at(m, 110, 43)], m.Heights[at(m, 132, 64)], m.Heights[at(m, 90, 50)]
	if !(mountain > hills && hills > plains) {
		t.Errorf("heights mountain %d hills %d plains %d", mountain, hills, plains)
	}
}

func fingerprint(m *Map) string {
	sum := sha256.New()
	for _, k := range m.Province {
		c := m.Provinces[k].Color
		sum.Write(c[:])
	}
	sum.Write(m.Heights)
	sum.Write(m.DefinitionCSV())
	return hex.EncodeToString(sum.Sum(nil))
}

func TestGenerateIsTheSameForTheSameSeed(t *testing.T) {
	a, b := generateTest(t, testOptions), generateTest(t, testOptions)
	if fingerprint(a) != fingerprint(b) {
		t.Fatal("two runs with one seed differ")
	}
	other := testOptions
	other.Seed = 8
	if fingerprint(generateTest(t, other)) == fingerprint(a) {
		t.Fatal("another seed made the same map")
	}
}

const goldenFingerprint = "56b5e50d98549150aa1442da058e6d65799818db3806806fd5502db84db33b52"

func TestGenerateMatchesTheRecordedMap(t *testing.T) {
	got := fingerprint(generateTest(t, Options{Width: 512, Height: 256, Seed: 3, LakeMaxPixels: 2000}))
	if got != goldenFingerprint {
		t.Errorf("the generated map changed: %s", got)
	}
}

func TestCheckFindsBrokenMaps(t *testing.T) {

	const A, B, C, D = 0x100000, 0x200000, 0x300000, 0x400000
	base := func() []uint32 {
		return []uint32{A, A, B, B, A, A, B, B, A, A, B, B, A, A, B, B}
	}
	defs := []Definition{{1, A, "land", false, "plains", 1}, {2, B, "sea", true, "ocean", 0}, {3, C, "land", false, "plains", 1}, {4, D, "land", false, "plains", 1}}
	codes := func(rgb []uint32, mutate func(*CheckInput)) map[string]string {
		in := CheckInput{Width: 4, Height: 4, ProvinceRGB: rgb, Definitions: defs[:2]}
		if mutate != nil {
			mutate(&in)
		}
		out := map[string]string{}
		for _, f := range Check(in) {
			out[f.Code] = f.Severity
		}
		return out
	}
	if got := codes(base(), nil); len(got) != 1 || got["size_not_256"] != "warning" {

		t.Errorf("clean map: %v", got)
	}
	black := base()
	black[0] = 0
	if codes(black, nil)["black_pixel"] != "error" {
		t.Error("black pixel")
	}
	undefined := base()
	undefined[5] = 0x123456
	if codes(undefined, nil)["undefined_color"] != "error" {
		t.Error("undefined colour")
	}
	split := []uint32{A, B, A, B, A, B, B, B, A, A, B, B, A, A, B, B}
	if got := codes(split, nil); got["disconnected_province"] != "warning" {
		t.Errorf("split: %v", got)
	}
	corner := []uint32{A, B, B, B, B, A, B, B, A, A, B, B, A, A, B, B}
	if got := codes(corner, nil); got["x_crossing"] != "warning" {
		t.Errorf("touching at a corner: %v", got)
	}
	four := []uint32{A, A, B, B, A, C, D, B, A, A, B, B, A, A, B, B}
	if got := codes(four, func(in *CheckInput) { in.Definitions = defs }); got["x_crossing"] != "warning" || got["small_province"] != "warning" {
		t.Errorf("four at a point: %v", got)
	}
	if got := codes(base(), func(in *CheckInput) { in.HeightSize = [2]int{8, 4} }); got["size_mismatch"] != "error" {
		t.Errorf("sizes: %v", got)
	}
	dup := append([]Definition{}, defs[:2]...)
	dup = append(dup, Definition{3, A, "land", false, "plains", 1})
	if codes(base(), func(in *CheckInput) { in.Definitions = dup })["duplicate_color"] != "error" {
		t.Error("duplicate colour")
	}
}

func TestBMPsHaveTheGamesFormats(t *testing.T) {
	m := generateTest(t, Options{Width: 256, Height: 128, Seed: 1, LakeMaxPixels: 2000})
	bitsOf := func(img image.Image) uint16 {
		body, err := EncodeBMP(img)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(body, []byte("BM")) {
			t.Fatal("not a BMP")
		}
		return binary.LittleEndian.Uint16(body[28:30])
	}
	if bits := bitsOf(m.ProvincesImage()); bits != 24 {
		t.Errorf("provinces.bmp is %d-bit", bits)
	}
	if bits := bitsOf(m.HeightImage()); bits != 8 {
		t.Errorf("heightmap.bmp is %d-bit", bits)
	}
	if bits := bitsOf(m.TerrainImage(GreyPalette())); bits != 8 {
		t.Errorf("terrain.bmp is %d-bit", bits)
	}
	if bits := bitsOf(m.RiversImage(GreyPalette())); bits != 8 {
		t.Errorf("rivers.bmp is %d-bit", bits)
	}
	csv := string(m.DefinitionCSV())
	if !bytes.HasPrefix([]byte(csv), []byte("0;0;0;0;land;false;unknown;0\n1;")) {
		t.Errorf("definition.csv starts %q", csv[:40])
	}
}
