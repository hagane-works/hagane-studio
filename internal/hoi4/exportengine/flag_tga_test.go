package exportengine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xdraw "golang.org/x/image/draw"
	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/importassets"
)

func encodeFlagTGALegacy(src image.Image, w, h int) []byte {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	header := make([]byte, 18)
	header[2] = 2
	binary.LittleEndian.PutUint16(header[12:14], uint16(w))
	binary.LittleEndian.PutUint16(header[14:16], uint16(h))
	header[16] = 32
	header[17] = 0x28
	out := make([]byte, 18+w*h*4)
	copy(out, header)
	at := 18
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := dst.NRGBAAt(x, y)
			out[at], out[at+1], out[at+2], out[at+3] = c.B, c.G, c.R, 255
			at += 4
		}
	}
	return out
}

func stripedFlag(colors ...color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 4, len(colors)))
	for y, c := range colors {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func tgaPixel(t *testing.T, data []byte, index int) color.NRGBA {
	t.Helper()
	at := 18 + index*4
	if at+4 > len(data) {
		t.Fatalf("pixel %d out of range (len=%d)", index, len(data))
	}
	return color.NRGBA{R: data[at+2], G: data[at+1], B: data[at], A: data[at+3]}
}

func TestEncodeFlagTGAWritesBottomLeftOrigin(t *testing.T) {
	top := color.NRGBA{R: 255, A: 255}
	mid := color.NRGBA{G: 255, A: 255}
	bottom := color.NRGBA{B: 255, A: 255}
	src := stripedFlag(top, mid, bottom)

	data := encodeFlagTGA(src, 4, 3)

	if got := data[17]; got != 0x08 {
		t.Fatalf("descriptor = 0x%02x, want 0x08 (bottom-left origin + 8 alpha bits)", got)
	}
	if got := data[2]; got != 2 {
		t.Fatalf("image type = %d, want 2 (uncompressed true-colour, no RLE)", got)
	}
	if got := data[16]; got != 32 {
		t.Fatalf("bits per pixel = %d, want 32", got)
	}
	if got := tgaPixel(t, data, 0); got != bottom {
		t.Fatalf("first stored pixel = %+v, want the BOTTOM row %+v", got, bottom)
	}
	if got := tgaPixel(t, data, 4*3-1); got != top {
		t.Fatalf("last stored pixel = %+v, want the TOP row %+v", got, top)
	}
	if got := tgaPixel(t, data, 4); got != mid {
		t.Fatalf("middle row pixel = %+v, want %+v", got, mid)
	}
}

func TestEncodeFlagTGAEqualsLegacyRowReversed(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 37, 23))
	for y := 0; y < 23; y++ {
		for x := 0; x < 37; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 7), G: uint8(y * 11), B: uint8(x ^ y), A: uint8(x + y)})
		}
	}
	for _, size := range []struct{ w, h int }{{82, 52}, {41, 26}, {10, 7}, {1, 1}} {
		got := encodeFlagTGA(src, size.w, size.h)
		legacy := encodeFlagTGALegacy(src, size.w, size.h)
		if len(got) != len(legacy) {
			t.Fatalf("%dx%d: length %d != legacy %d", size.w, size.h, len(got), len(legacy))
		}
		want := append([]byte(nil), legacy...)
		want[17] = 0x08
		stride := size.w * 4
		for y := 0; y < size.h; y++ {
			row := legacy[18+y*stride : 18+(y+1)*stride]
			dstRow := size.h - 1 - y
			copy(want[18+dstRow*stride:18+(dstRow+1)*stride], row)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%dx%d: output is not legacy-with-rows-reversed", size.w, size.h)
		}

		if size.h > 1 && bytes.Equal(got, legacy) {
			t.Fatalf("%dx%d: output is byte-identical to the buggy encoder", size.w, size.h)
		}
	}
}

func TestEncodeFlagTGALengthAcrossFlagSizes(t *testing.T) {
	src := stripedFlag(color.NRGBA{A: 255}, color.NRGBA{R: 255, A: 255})
	for _, size := range []struct{ w, h int }{{82, 52}, {41, 26}, {10, 7}} {
		data := encodeFlagTGA(src, size.w, size.h)
		if want := size.w*size.h*4 + 18; len(data) != want {
			t.Fatalf("%dx%d: len=%d want %d", size.w, size.h, len(data), want)
		}
		if data[17] != 0x08 {
			t.Fatalf("%dx%d: descriptor=0x%02x", size.w, size.h, data[17])
		}
		if w := binary.LittleEndian.Uint16(data[12:14]); int(w) != size.w {
			t.Fatalf("%dx%d: header width=%d", size.w, size.h, w)
		}
		if h := binary.LittleEndian.Uint16(data[14:16]); int(h) != size.h {
			t.Fatalf("%dx%d: header height=%d", size.w, size.h, h)
		}
	}
}

func syntheticBottomOriginFlag(w, h int, descriptor byte) []byte {
	out := make([]byte, 18, 18+w*h*4)
	out[2] = 2
	binary.LittleEndian.PutUint16(out[12:], uint16(w))
	binary.LittleEndian.PutUint16(out[14:], uint16(h))
	out[16] = 32
	out[17] = descriptor
	for row := 0; row < h; row++ {
		y := h - 1 - row
		for x := 0; x < w; x++ {
			r, g, b := byte(40), byte(x*255/max(w-1, 1)), byte(200)
			if y < h/2 {
				r, b = 220, 30
			}
			out = append(out, b, g, r, 255)
		}
	}
	return out
}

func maxChannelDelta(a, b image.Image, w, h int, flip bool) int {
	worst := 0
	for y := 0; y < h; y++ {
		by := y
		if flip {
			by = h - 1 - y
		}
		for x := 0; x < w; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, by).RGBA()
			for _, d := range []int{int(r1>>8) - int(r2>>8), int(g1>>8) - int(g2>>8), int(b1>>8) - int(b2>>8)} {
				if d < 0 {
					d = -d
				}
				if d > worst {
					worst = d
				}
			}
		}
	}
	return worst
}

func TestEncodeFlagTGARoundTripsVanillaFlag(t *testing.T) {
	type flagCase struct {
		name        string
		vanilla     []byte
		w, h        int
		vanillaDesc byte
	}
	cases := []flagCase{
		{"synthetic medium", syntheticBottomOriginFlag(41, 26, 0x08), 41, 26, 0x08},
		{"synthetic small", syntheticBottomOriginFlag(10, 7, 0x00), 10, 7, 0x00},
	}
	if dir := os.Getenv("HOI4_GAME_DIR"); dir != "" {
		for _, real := range []struct {
			rel  string
			w, h int
			desc byte
		}{
			{"gfx/flags/medium/SAX_neutrality.tga", 41, 26, 0x08},
			{"gfx/flags/small/SAX_neutrality.tga", 10, 7, 0x00},
		} {
			raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(real.rel)))
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, flagCase{real.rel, raw, real.w, real.h, real.desc})
		}
	}
	for _, tc := range cases {
		vanilla := tc.vanilla
		if vanilla[2] != 2 || vanilla[16] != 32 || vanilla[17] != tc.vanillaDesc {
			t.Fatalf("%s: fixture drifted: type=%d bpp=%d desc=0x%02x", tc.name, vanilla[2], vanilla[16], vanilla[17])
		}
		if vanilla[17]&0x20 != 0 {
			t.Fatalf("%s: fixture is not a bottom-origin flag", tc.name)
		}
		src, err := importassets.DecodeImage(vanilla, "SAX_neutrality.tga")
		if err != nil {
			t.Fatal(err)
		}
		if got := src.Bounds().Dx(); got != tc.w {
			t.Fatalf("%s: decoded width=%d want %d", tc.name, got, tc.w)
		}

		encoded := encodeFlagTGA(src, tc.w, tc.h)

		if encoded[2] != vanilla[2] || encoded[16] != vanilla[16] {
			t.Fatalf("%s: header drifted from vanilla", tc.name)
		}
		if encoded[17]&0x20 != 0 {
			t.Fatalf("%s: encoded descriptor 0x%02x claims top-left origin", tc.name, encoded[17])
		}
		if want := tc.w*tc.h*4 + 18; len(encoded) != want {
			t.Fatalf("%s: len=%d want %d", tc.name, len(encoded), want)
		}

		back, err := importassets.DecodeImage(encoded, "SAX_neutrality.tga")
		if err != nil {
			t.Fatal(err)
		}
		upright := maxChannelDelta(src, back, tc.w, tc.h, false)
		flipped := maxChannelDelta(src, back, tc.w, tc.h, true)
		if upright > 2 {
			t.Fatalf("%s: round-trip max channel delta = %d (want <= 2)", tc.name, upright)
		}
		if flipped < 32 {
			t.Fatalf("%s: fixture too symmetric to detect a flip (delta=%d)", tc.name, flipped)
		}

		vanRows := vanilla[18 : 18+tc.w*tc.h*4]
		ourRows := encoded[18:]
		mismatch := 0
		for i := 0; i < len(vanRows); i++ {
			if i%4 == 3 {
				continue
			}
			if vanRows[i] != ourRows[i] {
				mismatch++
			}
		}
		if mismatch != 0 {
			t.Fatalf("%s: %d/%d colour bytes differ from vanilla's own row order", tc.name, mismatch, len(vanRows))
		}
		t.Logf("%s: upright delta=%d flipped delta=%d colour bytes identical to vanilla=%d",
			tc.name, upright, flipped, len(vanRows)-len(vanRows)/4)
	}
}

func decodeCountryRow(t *testing.T, raw string) *worldout.OMap {
	t.Helper()
	row := worldout.NewOMap()
	if err := row.UnmarshalJSON([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestCountryFlagHasSomethingOnRecord(t *testing.T) {
	cases := []struct {
		row  string
		want bool
	}{
		{`{"tag":"CHI"}`, false},
		{`{"tag":"CHI","flag_image_path":null,"flag_paths_by_ideology":{},"has_obs":false}`, false},
		{`{"tag":"CHI","flag_image_path":"","flag_paths_by_ideology":{},"has_obs":false}`, false},
		{`{"tag":"CHI","flag_image_path":"   "}`, false},
		{`{"tag":"CHI","flag_paths_by_ideology":[]}`, false},
		{`{"tag":"CHI","flag_image_path":"flags/CHI.png"}`, true},
		{`{"tag":"CHI","flag_paths_by_ideology":{"fascism":"flags/CHI_f.png"}}`, true},
		{`{"tag":"CHI","has_obs":true}`, true},
	}
	for _, tc := range cases {
		if got := countryFlagHasSomethingOnRecord(decodeCountryRow(t, tc.row)); got != tc.want {
			t.Fatalf("%s -> %v want %v", tc.row, got, tc.want)
		}
	}
	if countryFlagHasSomethingOnRecord(nil) {
		t.Fatal("nil row must be silent")
	}
}

func TestFlagsMissingWarningWording(t *testing.T) {
	if got := flagsMissingWarning(nil, english); got != "" {
		t.Fatalf("empty input must stay silent, got %q", got)
	}
	if got := flagsMissingWarning([]string{"", "   "}, english); got != "" {
		t.Fatalf("blank tags must stay silent, got %q", got)
	}
	one := flagsMissingWarning([]string{"chi"}, english)
	if !strings.Contains(one, "could not be exported (1): CHI.") {
		t.Fatalf("one country: %q", one)
	}
	three := flagsMissingWarning([]string{"CHI", "YUN", "WUU", "chi"}, english)
	if !strings.Contains(three, "(3): CHI, WUU, YUN.") {
		t.Fatalf("tag list should be counted, sorted and deduped: %q", three)
	}
	if !strings.Contains(three, "show the base game's flag or a blank one") {
		t.Fatalf("consequence must be spelled out: %q", three)
	}
	many := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		many = append(many, fmt.Sprintf("T%02d", i))
	}
	long := flagsMissingWarning(many, english)
	if !strings.Contains(long, "and 4 more") {
		t.Fatalf("overflow suffix: %q", long)
	}
	if strings.Contains(long, "T08") {
		t.Fatalf("only the first 8 tags may be listed: %q", long)
	}
}

func flagWarnings(artifact NativeArtifact) []string {
	var out []string
	for _, w := range artifact.Warnings {
		if strings.HasPrefix(w, "Countries whose flag could not be exported") {
			out = append(out, w)
		}
	}
	return out
}

func flagSnapshot(countries ...string) Snapshot {
	rows := make([]json.RawMessage, 0, len(countries))
	for _, c := range countries {
		rows = append(rows, json.RawMessage(c))
	}
	return Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","start_year":1936}`),
		Tables:    map[string][]json.RawMessage{"countries": rows},
	}
}

func TestBuildNativeShadowAggregatesLostCountryFlags(t *testing.T) {
	snapshot := flagSnapshot(
		`{"id":1,"tag":"CHI","name":"China","flag_image_path":"flags/CHI.png"}`,
		`{"id":2,"tag":"WUU","name":"Wu","flag_paths_by_ideology":{"fascism":"flags/WUU_f.png"}}`,
		`{"id":3,"tag":"YUN","name":"Yunnan","has_obs":true}`,
		`{"id":4,"tag":"SHX","name":"Shanxi"}`,
	)
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	got := flagWarnings(artifact)
	if len(got) != 1 {
		t.Fatalf("want exactly 1 aggregated warning, got %d: %#v", len(got), got)
	}
	if !strings.Contains(got[0], "(3): CHI, WUU, YUN.") {
		t.Fatalf("warning = %q", got[0])
	}
	if strings.Contains(got[0], "SHX") {
		t.Fatalf("a country that never had a flag must not be named: %q", got[0])
	}
	t.Logf("warning: %s", got[0])
}

func TestBuildNativeShadowSilentWhenNoFlagWasEverUploaded(t *testing.T) {
	rows := make([]string, 0, 496)
	for i := 0; i < 496; i++ {
		rows = append(rows, fmt.Sprintf(
			`{"id":%d,"tag":"T%02d","name":"C%d","flag_image_path":null,"flag_paths_by_ideology":{},"has_obs":false}`,
			i+1, i%100, i))
	}
	artifact, err := BuildNativeShadow(flagSnapshot(rows...), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := flagWarnings(artifact); len(got) != 0 {
		t.Fatalf("want silence, got %#v", got)
	}
}
