package mapgen

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func lakeStrip() *Base {
	return stripBase([]band{{20, 1}, {40, 2}, {60, 3}, {80, 4}, {100, 5}}, 0,
		map[int32]string{1: "land", 2: "land", 3: "lake", 4: "land", 5: "land"})
}

func TestPlanRestateLakesGoWithTheLand(t *testing.T) {
	b := lakeStrip()
	states := map[int]RestateState{
		10: {Provinces: []int{1, 2}, Owner: "AAA"},
		20: {Provinces: []int{3, 4}, Owner: "AAA"},
		30: {Provinces: []int{5}, Owner: "BBB"},
	}
	plan := PlanRestate(states, b.Neighbours(), map[int]string{4: "BBB"}, nil, b.IsLand())
	if !reflect.DeepEqual(plan.Owner, map[int]string{20: "BBB"}) || len(plan.Lists) != 0 || len(plan.New) != 0 {
		t.Fatalf("state 20 should go to BBB whole, lake and all: %+v", plan)
	}

	if plan := PlanRestate(states, b.Neighbours(), map[int]string{3: "BBB"}, nil, b.IsLand()); len(plan.Owner)+len(plan.Lists)+len(plan.New) != 0 {
		t.Fatalf("a painted lake moved: %+v", plan)
	}

	old := PlanRestate(states, b.Neighbours(), map[int]string{4: "BBB"}, nil, nil)
	if !reflect.DeepEqual(old.Lists[20], []int{3}) {
		t.Fatalf("without land the plan should have left the lake: %+v", old)
	}
}

func TestLandForLakeStatesTakesTheLandBeside(t *testing.T) {
	b := lakeStrip()
	states := map[int]RestateState{
		10: {Provinces: []int{1, 2}, Owner: "AAA"},
		20: {Provinces: []int{3}, Owner: "AAA"},
		30: {Provinces: []int{4, 5}, Owner: "BBB"},
	}
	regionOf := map[int]int{1: 1, 2: 1, 3: 2, 4: 2, 5: 2}
	plan := b.LandForLakeStates(states, regionOf)
	if !reflect.DeepEqual(plan.Lists, map[int][]int{20: {3, 4}, 30: {5}}) || !reflect.DeepEqual(plan.Owner, map[int]string{20: "BBB"}) {
		t.Fatalf("plan %+v", plan)
	}

	states[30] = RestateState{Provinces: []int{4}, Owner: "BBB"}
	states[40] = RestateState{Provinces: []int{5}, Owner: "BBB"}
	if plan := b.LandForLakeStates(states, regionOf); len(plan.Lists) != 0 {
		t.Fatalf("took a state's only land: %+v", plan)
	}
	if plan := b.LandForLakeStates(map[int]RestateState{10: {Provinces: []int{1, 2}}, 20: {Provinces: []int{3, 4}}}, regionOf); len(plan.Lists)+len(plan.Owner) != 0 {
		t.Fatalf("nothing to mend: %+v", plan)
	}
}

func TestRestateBuildingsPlacesSitesOnLand(t *testing.T) {
	b := lakeStrip()
	out, _, added := b.RestateBuildings("", nil, map[int][]int{7: {3, 4}}, 5)
	if added == 0 {
		t.Fatal("nothing placed")
	}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, ";")
		x, _ := strconv.ParseFloat(f[2], 64)
		z, _ := strconv.ParseFloat(f[4], 64)
		if p := b.Province[(b.Height-1-int(z+0.5))*b.Width+int(x+0.5)]; p != 4 {
			t.Fatalf("%s is in province %d, want the land 4", line, p)
		}
	}
	if out, _, added := b.RestateBuildings("", nil, map[int][]int{7: {3}}, 5); added != 0 || out != "" {
		t.Fatalf("a state of a lake alone got buildings on the lake: %q", out)
	}

	onLake := b.line(7, "air_base", 5*b.Width+50, 0, 0)
	elsewhere := b.line(8, "air_base", 5*b.Width+50, 0, 0)
	out, _, _ = b.RestateBuildings(onLake+"\n"+elsewhere, nil, map[int][]int{7: {3, 4}}, 5)
	bases := 0
	for _, line := range strings.Split(out, "\n") {
		if line == onLake {
			t.Fatal("the site on the lake stayed")
		}
		if strings.HasPrefix(line, "7;air_base;") {
			bases++
		}
	}
	if bases != 1 || !strings.Contains(out, elsewhere) {
		t.Fatalf("want one air base for 7 on land and 8's line untouched:\n%s", out)
	}
}

func TestPaintingTheBaseGameLeavesNoStateOfLakes(t *testing.T) {
	root := os.Getenv("HOI4_GAME_ROOT")
	if root == "" {
		t.Skip("HOI4_GAME_ROOT not set")
	}
	read := func(name string) []byte {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(root, "map", name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	b, err := LoadBase(read("provinces.bmp"), read("heightmap.bmp"), read("terrain.bmp"), read("rivers.bmp"), read("definition.csv"))
	if err != nil {
		t.Fatal(err)
	}
	regions := regionsIn(t, filepath.Join(root, "map", "strategicregions"))
	regionOf := map[int]int{}
	for id, provinces := range regions {
		for _, p := range provinces {
			regionOf[p] = id
		}
	}
	lists := statesIn(t, filepath.Join(root, "history", "states"))
	owners := ownersIn(t, filepath.Join(root, "history", "states"))
	isLand := b.IsLand()
	states := map[int]RestateState{}
	assign := map[int]string{}
	withLakes := 0
	for id, provinces := range lists {
		states[id] = RestateState{Provinces: provinces, Owner: owners[id]}
		lake := false
		for _, p := range provinces {
			lake = lake || !isLand(p)
		}
		if !lake {
			continue
		}
		withLakes++
		for _, p := range provinces {
			if isLand(p) {
				assign[p] = "ZZZ"
			}
		}
	}
	if withLakes < 5 {
		t.Fatalf("only %d states with lakes", withLakes)
	}
	after := func(base map[int]RestateState, plan RestatePlan) map[int]RestateState {
		out := map[int]RestateState{}
		for id, st := range base {
			out[id] = st
		}
		for id, provinces := range plan.Lists {
			st := out[id]
			st.Provinces = provinces
			out[id] = st
		}
		for id, tag := range plan.Owner {
			st := out[id]
			st.Owner = tag
			out[id] = st
		}
		next := len(base)
		for _, n := range plan.New {
			next++
			out[next] = RestateState{Provinces: n.Provinces, Owner: n.Owner}
		}
		return out
	}
	lists2 := func(states map[int]RestateState) map[int][]int {
		out := map[int][]int{}
		for id, st := range states {
			out[id] = st.Provinces
		}
		return out
	}
	codes := func(findings []Finding) []string {
		var out []string
		for _, f := range findings {
			out = append(out, f.Code)
		}
		sort.Strings(out)
		return out
	}
	neighbours := b.Neighbours()
	if got := b.LoadRules(regions, lists2(after(states, PlanRestate(states, neighbours, assign, regionOf, isLand)))); len(got) != 0 {
		t.Fatalf("painted %d states with lakes: %v", withLakes, codes(got))
	}
	old := after(states, PlanRestate(states, neighbours, assign, regionOf, nil))
	if got := codes(b.LoadRules(regions, lists2(old))); !reflect.DeepEqual(got, []string{"state_no_land"}) {
		t.Fatalf("painting as it was should leave states of lakes alone: %v", got)
	}
	mended := after(old, b.LandForLakeStates(old, regionOf))
	if got := b.LoadRules(regions, lists2(mended)); len(got) != 0 {
		t.Fatalf("after the repair: %v", codes(got))
	}
}
