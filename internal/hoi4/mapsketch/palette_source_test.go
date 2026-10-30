package mapsketch

import (
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"strings"
	"testing"
)

func TestPaletteSourceColourSelectsTheSampleRegionWithoutClusterAmbiguity(t *testing.T) {
	p := seedTestPicture()
	verifiedCluster := false
	for _, swatch := range Palette(p, 16) {
		actual := p.Pix[swatch.Sample.Y*p.W+swatch.Sample.X].Hex()
		if swatch.SampleSourceColour != actual {
			t.Fatalf("source colour does not describe the returned sample: %+v actual=%s", swatch, actual)
		}
		if swatch.Hex == actual {
			continue
		}
		cluster, _ := ParseHex(swatch.Hex)
		if _, err := FindColourSeed(p, ColourSeedOptions{Colour: cluster}); err == nil || !strings.Contains(err.Error(), "disconnected") {
			t.Fatalf("fixture must expose the historical merged-colour ambiguity: %+v err=%v", swatch, err)
		}
		colour, _ := ParseHex(swatch.SampleSourceColour)
		result, err := FindColourSeed(p, ColourSeedOptions{Colour: colour, At: &swatch.Sample})
		if err != nil || result.SourceColour != actual || result.SelectedPixels != 900 {
			t.Fatalf("exact colour and its explicit location must succeed on the first selection: %+v err=%v", result, err)
		}
		if (swatch.Sample.X < 40) != (result.ActualSeed.X < 40) {
			t.Fatalf("sample selected the other country: sample=%+v seed=%+v", swatch.Sample, result.ActualSeed)
		}
		verifiedCluster = true
	}
	if !verifiedCluster {
		t.Fatal("fixture did not exercise a display cluster differing from its source pixel")
	}
}

func TestPaletteDoesNotPublishATransparentSampleAsSourceColour(t *testing.T) {
	p := picture(640, 320, func(int, int) RGB { return green })
	before := Palette(p, 1)[0]
	p.Clear = make([]bool, len(p.Pix))
	p.Clear[before.Sample.Y*p.W+before.Sample.X] = true
	swatch := Palette(p, 1)[0]
	if swatch.Sample != before.Sample || swatch.Hex != before.Hex || swatch.Share != before.Share {
		t.Fatalf("adding source evidence must not replace legacy palette fields: before=%+v after=%+v", before, swatch)
	}
	raw, _ := json.Marshal(swatch)
	if swatch.SampleSourceColour != "" || strings.Contains(string(raw), "sample_source_colour") {
		t.Fatalf("transparent original pixel was advertised as a usable seed: %s", raw)
	}
}

func TestPaletteSourceColourAgainstExistingPoliticalMap(t *testing.T) {
	path := os.Getenv("HOI4_WORLD_SOURCE_FIXTURE")
	if path == "" {
		t.Skip("set HOI4_WORLD_SOURCE_FIXTURE to the existing private fantasy_world.png")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	p := FromImage(im)
	want := map[string]string{"#566ea4": "#5476a8", "#506184": "#5a64a0"}
	for _, swatch := range Palette(p, 40) {
		expected, matches := want[swatch.Hex]
		if !matches {
			continue
		}
		actual := p.Pix[swatch.Sample.Y*p.W+swatch.Sample.X].Hex()
		if swatch.SampleSourceColour != expected || actual != expected {
			t.Fatalf("historical display/source colour mismatch is not represented truthfully: %+v actual=%s", swatch, actual)
		}
		colour, _ := ParseHex(swatch.SampleSourceColour)
		result, err := FindColourSeed(p, ColourSeedOptions{Colour: colour, At: &swatch.Sample})
		if err != nil || result.SourceColour != expected || result.SelectedPixels < 10_000 {
			t.Fatalf("source sample selected lettering or another country: %+v err=%v", result, err)
		}
		delete(want, swatch.Hex)
	}
	if len(want) != 0 {
		t.Fatalf("historical palette/source regression was not exercised: %v", want)
	}
}
