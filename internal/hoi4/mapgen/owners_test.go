package mapgen

import (
	"slices"
	"testing"
)

func paintMask(w, h int, paint func(x, y int) uint8) []uint8 {
	mask := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mask[y*w+x] = paint(x, y)
		}
	}
	return mask
}

func westOfMiddle(world *World) map[int]bool {
	centre := map[int]float64{}
	for _, pr := range world.All.Provinces {
		centre[pr.ID] = pr.X
	}
	west := map[int]bool{}
	for k, st := range world.All.States {
		west[k+1] = centre[st.Capital] < float64(world.Base.Width)/2
	}
	return west
}

func TestOwnersFromMaskFollowsThePainting(t *testing.T) {
	_, world := twoIslands(t)
	tags := []string{"AAA", "BBB"}
	mask := paintMask(32, 16, func(x, _ int) uint8 {
		if x < 16 {
			return 2
		}
		return 1
	})
	owner, capital, missing := world.OwnersFromMask(tags, mask, 32, 16)
	if len(missing) != 0 {
		t.Fatalf("missing %v", missing)
	}
	west := westOfMiddle(world)
	for id := range west {
		want := "AAA"
		if west[id] {
			want = "BBB"
		}
		if owner[id] != want {
			t.Errorf("state %d (west %v) went to %q, want %s", id, west[id], owner[id], want)
		}
	}
	for _, tag := range tags {
		if owner[capital[tag]] != tag {
			t.Errorf("%s's capital %d is %q's", tag, capital[tag], owner[capital[tag]])
		}
	}
}

func TestOwnersFromMaskFillsUnpaintedLandAndNamesEmptyCountries(t *testing.T) {
	_, world := twoIslands(t)
	mask := paintMask(64, 32, func(x, _ int) uint8 {
		if x < 32 {
			return 1
		}
		return 0
	})
	owner, _, missing := world.OwnersFromMask([]string{"AAA", "BBB"}, mask, 64, 32)
	for k := range world.All.States {
		if owner[k+1] != "AAA" {
			t.Errorf("state %d went to %q, want AAA (the only country painted)", k+1, owner[k+1])
		}
	}
	if !slices.Equal(missing, []string{"BBB"}) {
		t.Errorf("missing %v, want [BBB]", missing)
	}
}

func TestOwnersFromMaskWithoutPaintIsWestToEast(t *testing.T) {
	_, world := twoIslands(t)
	tags := []string{"AAA", "BBB"}
	wantOwner, wantCapital := world.Owners(tags)
	for name, mask := range map[string][]uint8{
		"empty":         make([]uint8, 8*4),
		"past the tags": paintMask(8, 4, func(int, int) uint8 { return 9 }),
	} {
		owner, capital, missing := world.OwnersFromMask(tags, mask, 8, 4)
		if len(missing) != 0 {
			t.Errorf("%s: missing %v", name, missing)
		}
		for id, tag := range wantOwner {
			if owner[id] != tag {
				t.Errorf("%s: state %d %q, want %q", name, id, owner[id], tag)
			}
		}
		for tag, id := range wantCapital {
			if capital[tag] != id {
				t.Errorf("%s: %s capital %d, want %d", name, tag, capital[tag], id)
			}
		}
	}
}
