package mapgen

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func lineOfStates() (map[int]RestateState, map[int]map[int]bool) {
	states := map[int]RestateState{
		10: {Provinces: []int{1, 2, 3}, Owner: "AAA"},
		20: {Provinces: []int{4, 5}, Owner: "BBB"},
		30: {Provinces: []int{6, 7, 8}, Owner: "AAA"},
	}
	neighbours := map[int]map[int]bool{}
	for p := 1; p < 8; p++ {
		if neighbours[p] == nil {
			neighbours[p] = map[int]bool{}
		}
		if neighbours[p+1] == nil {
			neighbours[p+1] = map[int]bool{}
		}
		neighbours[p][p+1] = true
		neighbours[p+1][p] = true
	}
	return states, neighbours
}

func TestPlanRestateWholeStateChangesOnlyItsOwner(t *testing.T) {
	states, neighbours := lineOfStates()
	plan := PlanRestate(states, neighbours, map[int]string{4: "CCC", 5: "CCC", 1: "AAA"}, nil, nil)
	if !reflect.DeepEqual(plan.Owner, map[int]string{20: "CCC"}) || len(plan.Lists) != 0 || len(plan.New) != 0 {
		t.Fatalf("plan %+v", plan)
	}
}

func TestPlanRestatePiecesJoinTheNewOwnersStateBesideThem(t *testing.T) {
	states, neighbours := lineOfStates()

	plan := PlanRestate(states, neighbours, map[int]string{3: "BBB"}, nil, nil)
	want := map[int][]int{10: {1, 2}, 20: {3, 4, 5}}
	if len(plan.Owner) != 0 || !reflect.DeepEqual(plan.Lists, want) || len(plan.New) != 0 {
		t.Fatalf("plan %+v, want lists %v", plan, want)
	}
}

func TestPlanRestatePiecesBorderingNoneOfTheirOwnerBecomeOneNewState(t *testing.T) {
	states, neighbours := lineOfStates()

	plan := PlanRestate(states, neighbours, map[int]string{1: "CCC", 2: "CCC", 7: "CCC", 8: "CCC"}, nil, nil)
	want := []RestateNew{{Provinces: []int{1, 2}, Owner: "CCC", From: 10}, {Provinces: []int{7, 8}, Owner: "CCC", From: 30}}
	if !reflect.DeepEqual(plan.New, want) {
		t.Fatalf("new %+v, want %+v", plan.New, want)
	}
	if !reflect.DeepEqual(plan.Lists, map[int][]int{10: {3}, 30: {6}}) {
		t.Fatalf("lists %v", plan.Lists)
	}
}

func TestPlanRestateStateWithNoneOfItsOwnerLeftGoesToTheMost(t *testing.T) {
	states, neighbours := lineOfStates()

	plan := PlanRestate(states, neighbours, map[int]string{6: "BBB", 7: "CCC", 8: "CCC"}, nil, nil)
	if !reflect.DeepEqual(plan.Owner, map[int]string{30: "CCC"}) {
		t.Fatalf("owner %v", plan.Owner)
	}
	if !reflect.DeepEqual(plan.Lists, map[int][]int{20: {4, 5, 6}, 30: {7, 8}}) || len(plan.New) != 0 {
		t.Fatalf("plan %+v", plan)
	}
}

func TestPlanRestatePieceDoesNotJoinAStateBeingCutAway(t *testing.T) {
	states, neighbours := lineOfStates()

	plan := PlanRestate(states, neighbours, map[int]string{3: "BBB", 4: "AAA"}, nil, nil)
	if len(plan.New) != 2 {
		t.Fatalf("new %+v", plan.New)
	}
	for _, n := range plan.New {
		if (n.Owner == "BBB") != reflect.DeepEqual(n.Provinces, []int{3}) {
			t.Errorf("unexpected new state %+v", n)
		}
	}
}

func regionsOfLine() map[int]int {
	return map[int]int{1: 100, 2: 100, 3: 100, 4: 200, 5: 200, 6: 300, 7: 300, 8: 300}
}

func TestPlanRestateKeepsEveryStateInOneRegion(t *testing.T) {
	states, neighbours := lineOfStates()
	plan := PlanRestate(states, neighbours, map[int]string{3: "BBB"}, regionsOfLine(), nil)
	want := []RestateNew{{Provinces: []int{3}, Owner: "BBB", From: 10}}
	if !reflect.DeepEqual(plan.New, want) || !reflect.DeepEqual(plan.Lists, map[int][]int{10: {1, 2}}) {
		t.Fatalf("plan %+v, want new %+v", plan, want)
	}

	same := map[int]int{1: 100, 2: 100, 3: 100, 4: 100, 5: 100, 6: 300, 7: 300, 8: 300}
	plan = PlanRestate(states, neighbours, map[int]string{3: "BBB"}, same, nil)
	if !reflect.DeepEqual(plan.Lists, map[int][]int{10: {1, 2}, 20: {3, 4, 5}}) || len(plan.New) != 0 {
		t.Fatalf("same region: %+v", plan)
	}
}

func TestPlanRestateNewStatesStayInOneRegion(t *testing.T) {
	states, neighbours := lineOfStates()
	plan := PlanRestate(states, neighbours, map[int]string{3: "CCC", 4: "CCC"}, regionsOfLine(), nil)
	want := []RestateNew{{Provinces: []int{3}, Owner: "CCC", From: 10}, {Provinces: []int{4}, Owner: "CCC", From: 20}}
	if !reflect.DeepEqual(plan.New, want) {
		t.Fatalf("new %+v, want %+v", plan.New, want)
	}
	if plan = PlanRestate(states, neighbours, map[int]string{3: "CCC", 4: "CCC"}, nil, nil); len(plan.New) != 1 {
		t.Fatalf("without regions the touching pieces are one state: %+v", plan.New)
	}
}

func TestSplitStatesByRegionCutsAcrossTheEdge(t *testing.T) {
	states, neighbours := lineOfStates()
	states[10] = RestateState{Provinces: []int{1, 2, 3, 4}, Owner: "AAA"}
	states[20] = RestateState{Provinces: []int{5}, Owner: "BBB"}
	plan := SplitStatesByRegion(states, neighbours, regionsOfLine(), nil)
	if !reflect.DeepEqual(plan.Lists, map[int][]int{10: {1, 2, 3}}) || len(plan.Owner) != 0 {
		t.Fatalf("plan %+v", plan)
	}
	if want := []RestateNew{{Provinces: []int{4}, Owner: "AAA", From: 10}}; !reflect.DeepEqual(plan.New, want) {
		t.Fatalf("new %+v, want %+v", plan.New, want)
	}
	if plan := SplitStatesByRegion(map[int]RestateState{10: {Provinces: []int{1, 2, 3}, Owner: "AAA"}}, neighbours, regionsOfLine(), nil); len(plan.Lists)+len(plan.New) != 0 {
		t.Fatalf("a state in one region: %+v", plan)
	}
}

func TestSplitStatesByRegionKeepsTheCapitalsPart(t *testing.T) {
	_, neighbours := lineOfStates()
	states := map[int]RestateState{20: {Provinces: []int{3, 4}, Owner: "BBB"}}
	if plan := SplitStatesByRegion(states, neighbours, regionsOfLine(), nil); !reflect.DeepEqual(plan.Lists[20], []int{3}) {
		t.Fatalf("no capital, the lower region stays: %+v", plan)
	}
	plan := SplitStatesByRegion(states, neighbours, regionsOfLine(), map[int]int{20: 4})
	if !reflect.DeepEqual(plan.Lists[20], []int{4}) || len(plan.New) != 1 || !reflect.DeepEqual(plan.New[0].Provinces, []int{3}) {
		t.Fatalf("capital 4: %+v", plan)
	}
}

func TestRestateStem(t *testing.T) {
	for name, want := range map[string]struct {
		stem string
		n    int
	}{
		"Alsace-Lorraine 2": {"Alsace-Lorraine", 2}, "Alsace-Lorraine": {"Alsace-Lorraine", 0},
		"阿尔萨斯-洛林 3": {"阿尔萨斯-洛林", 3}, "Sector 0": {"Sector 0", 0}, "2": {"2", 0},
	} {
		if stem, n := RestateStem(name); stem != want.stem || n != want.n {
			t.Errorf("%q: %q %d, want %q %d", name, stem, n, want.stem, want.n)
		}
	}
}

func TestRestateBuildingsRenamesAndFillsIn(t *testing.T) {
	_, world := twoIslands(t)
	base := world.Base
	stateIDs := make([]int, len(world.All.States))
	for k := range stateIDs {
		stateIDs[k] = k + 1
	}
	text := strings.Join(base.Positions(world.All, stateIDs, 1, 5).Buildings, "\n")

	first := world.All.States[0]
	stateOf := map[int]int{}
	for _, p := range first.Provinces {
		stateOf[p] = 99
	}
	out, renamed, added := base.RestateBuildings(text, stateOf, map[int][]int{99: first.Provinces, 1: first.Provinces}, 5)
	if renamed == 0 {
		t.Fatal("no line renamed")
	}
	count := map[int]map[string]int{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, ";")
		id, _ := strconv.Atoi(f[0])
		if count[id] == nil {
			count[id] = map[string]int{}
		}
		count[id][f[1]]++
	}
	for _, b := range stateBuildings {
		if count[99][b.kind] < b.count {
			t.Errorf("state 99 has %d %s, want %d", count[99][b.kind], b.kind, b.count)
		}
	}
	if count[1] != nil && count[1]["supply_node"] > 0 {
		t.Errorf("state 1 kept province lines: %v", count[1])
	}
	if added == 0 {
		t.Error("state 1 got no state-wide buildings back")
	}

	if got, _, _ := base.RestateBuildings(text+"\n", nil, nil, 5); !strings.HasSuffix(got, "\n") {
		t.Error("lost the last newline")
	}
	if got, _, _ := base.RestateBuildings(text, nil, nil, 5); got != text {
		t.Error("an untouched file changed")
	}
}
