package exportengine

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	xdraw "golang.org/x/image/draw"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

func TestAgencyLogoBecomesTheGamesTwoFrameStrip(t *testing.T) {
	logo := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	red := color.NRGBA{200, 30, 30, 255}
	xdraw.Draw(logo, image.Rect(32, 32, 96, 96), image.NewUniform(red), image.Point{}, xdraw.Src)
	var raw bytes.Buffer
	if err := png.Encode(&raw, logo); err != nil {
		t.Fatal(err)
	}
	strip, err := agencyLogoStrip(raw.Bytes(), "agency test")
	if err != nil {
		t.Fatal(err)
	}
	if strip.Bounds().Dx() != 234 || strip.Bounds().Dy() != 119 {
		t.Fatalf("strip is %v, want 234x119", strip.Bounds())
	}
	centre := func(x int) color.NRGBA { return strip.NRGBAAt(x, 59) }
	if centre(58) != centre(58+117) || centre(58).A != 255 || centre(58).R < 180 {
		t.Fatalf("the logo differs between frames: %v vs %v", centre(58), centre(58+117))
	}

	edge := 25
	if a := strip.NRGBAAt(edge, 59).A; a != 0 {
		t.Fatalf("frame 1 outside the logo has alpha %d", a)
	}
	if g := strip.NRGBAAt(edge+117, 59); g.A < 60 || g.R < 200 || g.B > 80 {
		t.Fatalf("frame 2 outside the logo = %v, want a warm glow", g)
	}
	if a := strip.NRGBAAt(1+117, 1).A; a > 10 {
		t.Fatalf("frame 2 corner alpha %d: the glow should stay near the logo", a)
	}
}

func TestSpriteBlockDeclaresTheFrameCount(t *testing.T) {
	block := emitmisc.EmitSpriteBlock([]emitmisc.SpritePair{
		{Name: "GFX_intelligence_agency_logo_abw", Path: "gfx/interface/intelligence_agencies/intel_agency_abw.dds", Frames: 2},
		{Name: "GFX_idea_mio_x", Path: "gfx/interface/ideas/mio_x.dds"},
	})
	if strings.Count(block, "noOfFrames") != 1 || !strings.Contains(block, "intel_agency_abw.dds\"\n\t\tnoOfFrames = 2\n\t}") {
		t.Fatalf("block =\n%s", block)
	}
}

func TestAgencyLogoStripAcceptsThePreviewStandIn(t *testing.T) {
	dds, err := agencyLogoStripDDS(manifestOnlyBlob, "agency preview")
	if err != nil || len(dds) == 0 {
		t.Fatalf("strip from the preview stand-in: %v", err)
	}
}
