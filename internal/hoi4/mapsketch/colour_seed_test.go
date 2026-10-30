package mapsketch

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func seedTestPicture() *Picture {
	img := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			c := color.NRGBA{162, 186, 206, 255}
			if x >= 5 && x < 35 && y >= 5 && y < 35 {
				c = color.NRGBA{84, 118, 168, 255}
			}
			if x >= 45 && x < 75 && y >= 5 && y < 35 {
				c = color.NRGBA{90, 100, 160, 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return FromImage(img)
}

func TestColourSeedDoesNotMergeNearbyPoliticalColours(t *testing.T) {
	p := seedTestPicture()
	for _, tc := range []struct {
		colour      RGB
		left, right int
	}{{RGB{84, 118, 168}, 5, 35}, {RGB{90, 100, 160}, 45, 75}} {
		result, err := FindColourSeed(p, ColourSeedOptions{Colour: tc.colour, Tolerance: 3})
		if err != nil || result.ActualSeed.X < tc.left || result.ActualSeed.X >= tc.right || result.SourceColour != tc.colour.Hex() {
			t.Fatalf("distinct source colour jumped to another country: %+v err=%v", result, err)
		}
		if result.MatchedLandPixels != 900 || result.SelectedPixels != 900 || result.MatchShare != 900.0/3200 {
			t.Fatalf("match evidence wrong: %+v", result)
		}
	}
}

func TestColourSeedAmbiguityRequiresBoxOrLocationHint(t *testing.T) {
	p := seedTestPicture()
	opt := ColourSeedOptions{Colour: RGB{86, 110, 164}, Tolerance: 20}
	result, err := FindColourSeed(p, opt)
	if err == nil || !strings.Contains(err.Error(), "disconnected") || len(result.Candidates) != 2 {
		t.Fatalf("merged-looking palette colour silently picked a continent: %+v %v", result, err)
	}
	opt.Box = &Rect{X: 40, Y: 0, W: 40, H: 40}
	result, err = FindColourSeed(p, opt)
	if err != nil || result.ActualSeed.X < 45 || result.SourceColour != "#5a64a0" {
		t.Fatalf("source box failed to isolate Ardenfall: %+v %v", result, err)
	}
	opt.Box, opt.At = nil, &Point{X: 70, Y: 30}
	result, err = FindColourSeed(p, opt)
	if err != nil || result.ActualSeed.X < 45 {
		t.Fatalf("at hint failed to select the intended same-colour region: %+v %v", result, err)
	}
}

func TestColourSeedAvoidsWhiteLabelsAndDarkBorders(t *testing.T) {
	p := seedTestPicture()
	for y := 15; y < 25; y++ {
		for x := 15; x < 25; x++ {
			p.Pix[y*p.W+x] = RGB{250, 246, 236}
		}
	}
	result, err := FindColourSeed(p, ColourSeedOptions{Colour: RGB{84, 118, 168}, At: &Point{X: 20, Y: 20}, Tolerance: 3})
	if err != nil || result.SourceColour != "#5476a8" || (result.ActualSeed.X >= 15 && result.ActualSeed.X < 25 && result.ActualSeed.Y >= 15 && result.ActualSeed.Y < 25) {
		t.Fatalf("source colour selector chose white lettering: %+v %v", result, err)
	}
}

func TestColourSeedRejectsColourErasedByTerrainAndMissingColour(t *testing.T) {
	p := seedTestPicture()
	result, err := FindColourSeed(p, ColourSeedOptions{Colour: RGB{84, 118, 168}, Tolerance: 3, Land: func(Point) bool { return false }})
	if err == nil || !strings.Contains(err.Error(), "terrain") || result.SourceMatchPixels != 900 || result.MatchedLandPixels != 0 {
		t.Fatalf("water-erased country was silently moved elsewhere: %+v %v", result, err)
	}
	if _, err := FindColourSeed(p, ColourSeedOptions{Colour: RGB{255, 0, 255}, Tolerance: 3}); err == nil {
		t.Fatal("nonexistent source colour accepted")
	}
	if _, err := FindColourSeed(p, ColourSeedOptions{Colour: RGB{84, 118, 168}, Box: &Rect{X: 70, W: 20, H: 20}}); err == nil {
		t.Fatal("out-of-source box accepted")
	}
}
