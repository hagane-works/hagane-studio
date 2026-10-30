package mapgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateOutlinesOfTwoNeighbours(t *testing.T) {

	rows := []string{
		"111222",
		"112222",
		"111222",
		"111222",
	}
	w, h := 6, 4
	province := make([]int32, w*h)
	for y, row := range rows {
		for x, c := range row {
			province[y*w+x] = int32(c - '0')
		}
	}
	shapes := StateOutlines(w, h, province, 1, map[int][]int{10: {1}, 20: {2}, 30: {9}})
	if _, ok := shapes[30]; ok {
		t.Error("state 30 has no pixel: want it left out")
	}
	left, right := shapes[10], shapes[20]
	if len(left.Rings) != 1 || len(right.Rings) != 1 {
		t.Fatalf("rings: %v / %v", left.Rings, right.Rings)
	}
	if left.BBox != [4]int{0, 0, 3, 4} || right.BBox != [4]int{2, 0, 6, 4} {
		t.Errorf("boxes %v %v", left.BBox, right.BBox)
	}

	if left.Centroid != [2]int{1, 2} {
		t.Errorf("left centroid %v", left.Centroid)
	}
	corners := func(ring [][2]int) map[[2]int]bool {
		out := map[[2]int]bool{}
		for _, p := range ring {
			out[p] = true
		}
		return out
	}
	l, r := corners(left.Rings[0]), corners(right.Rings[0])

	for _, p := range [][2]int{{3, 0}, {3, 1}, {2, 1}, {2, 2}, {3, 2}, {3, 4}} {
		if !l[p] || !r[p] {
			t.Errorf("corner %v: left %v right %v", p, l[p], r[p])
		}
	}
}

func TestStateOutlinesMatchTheShippedBaseGameOutlines(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	provinces, err := os.ReadFile(filepath.Join(root, "map", "provinces.bmp"))
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := os.ReadFile(filepath.Join(root, "map", "definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	w, h, province, err := LoadProvinces(provinces, definitions)
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		States []struct {
			StateID   int   `json:"state_id"`
			Provinces []int `json:"provinces"`
		} `json:"states"`
	}

	raw, err := os.ReadFile(filepath.Join("..", "projectapi", "vanilladata", "vanilla_states.json"))
	if err != nil {
		t.Skip("vanilla_states.json is not available")
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	states := map[int][]int{}
	for _, s := range catalog.States {
		states[s.StateID] = s.Provinces
	}
	var shipped struct {
		Factor int                   `json:"factor"`
		States map[string]StateShape `json:"states"`
	}
	raw, err = os.ReadFile(filepath.Join("..", "projectapi", "vanilladata", "vanilla_states_geo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &shipped); err != nil {
		t.Fatal(err)
	}
	traced := StateOutlines(w, h, province, shipped.Factor, states)
	differ := 0
	for id, want := range shipped.States {
		var n int
		if err := json.Unmarshal([]byte(id), &n); err != nil {
			t.Fatal(err)
		}
		got, ok := traced[n]
		if !ok || !reflect.DeepEqual(got, want) {
			differ++
			if differ <= 3 {
				t.Errorf("state %s: traced %d rings box %v centroid %v; shipped %d rings box %v centroid %v",
					id, len(got.Rings), got.BBox, got.Centroid, len(want.Rings), want.BBox, want.Centroid)
			}
		}
	}
	if differ > 0 || len(traced) != len(shipped.States) {
		t.Errorf("%d of %d states differ; traced %d", differ, len(shipped.States), len(traced))
	}
}
