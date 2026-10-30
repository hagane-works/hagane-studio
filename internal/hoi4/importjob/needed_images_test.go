package importjob

import (
	"context"
	"sort"
	"strings"
	"testing"
)

func TestNeededImagesResolveBareSpriteNames(t *testing.T) {
	s := newSession(context.Background(), t.TempDir(), AllOptions(), nil)
	s.gfx["GFX_idea_TST_spirit"] = "gfx/interface/ideas/idea_TST_spirit.dds"
	s.gfx["GFX_decision_tst_build"] = "gfx/interface/decisions/tst_build.dds"
	s.gfx["GFX_decision_category_tst"] = "gfx/interface/decisions/category_tst.dds"
	s.entities = []entityRow{
		{kind: "idea", naturalKey: "TST_spirit", payload: []byte(`{"picture":"TST_spirit"}`)},
	}
	s.stageCharsExtrasNeededImages()
	s.world.rows = []worldRow{
		{kind: "decision", key: "tst_build", data: payload{"icon": "tst_build"}},
		{kind: "decision_category", key: "tst", data: payload{"icon": "tst"}},
	}
	s.recordWorldImages()
	var images []string
	for path := range s.neededImages {
		images = append(images, path)
	}
	sort.Strings(images)
	want := "gfx/interface/decisions/category_tst.dds|gfx/interface/decisions/tst_build.dds|gfx/interface/ideas/idea_TST_spirit.dds"
	if got := strings.Join(images, "|"); got != want {
		t.Fatalf("needed images = %s, want %s", got, want)
	}
}
