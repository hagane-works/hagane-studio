package emitmisc

import (
	"strconv"
	"strings"
)

type SpritePair struct {
	Name string
	Path string

	Frames int
}

func EmitSpriteBlock(entries []SpritePair) string {
	if len(entries) == 0 {
		return ""
	}
	lines := []string{"spriteTypes = {"}
	for _, e := range entries {
		lines = append(lines, "\tspriteType = {")
		lines = append(lines, "\t\tname = "+e.Name)
		lines = append(lines, "\t\ttexturefile = \""+e.Path+"\"")
		if e.Frames > 1 {
			lines = append(lines, "\t\tnoOfFrames = "+strconv.Itoa(e.Frames))
		}
		lines = append(lines, "\t}")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

const shineOverlayPath = "gfx/interface/goals/shine_overlay.dds"

func EmitFocusShineBlock(entries []SpritePair) string {
	if len(entries) == 0 {
		return ""
	}
	lines := []string{"spriteTypes = {"}
	for _, e := range entries {
		lines = append(lines, "\tspriteType = {")
		lines = append(lines, "\t\tname = "+e.Name+"_shine")
		lines = append(lines, "\t\ttexturefile = \""+e.Path+"\"")
		lines = append(lines, "\t\teffectFile = \"gfx/FX/buttonstate.lua\"")
		for _, rotation := range []string{"-90.0", "90.0"} {
			lines = append(lines, "\t\tanimation = {")
			lines = append(lines, "\t\t\tanimationmaskfile = \""+e.Path+"\"")
			lines = append(lines, "\t\t\tanimationtexturefile = \""+shineOverlayPath+"\"")
			lines = append(lines, "\t\t\tanimationrotation = "+rotation)
			lines = append(lines, "\t\t\tanimationlooping = no")
			lines = append(lines, "\t\t\tanimationtime = 0.75")
			lines = append(lines, "\t\t\tanimationdelay = 0")
			lines = append(lines, "\t\t\tanimationblendmode = \"add\"")
			lines = append(lines, "\t\t\tanimationtype = \"scrolling\"")
			lines = append(lines, "\t\t\tanimationrotationoffset = { x = 0.0 y = 0.0 }")
			lines = append(lines, "\t\t\tanimationtexturescale = { x = 1.0 y = 1.0 }")
			lines = append(lines, "\t\t}")
		}
		lines = append(lines, "\t\tlegacy_lazy_load = no")
		lines = append(lines, "\t}")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}
