package worldmap

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseDrownSaysWhat(t *testing.T) {
	if _, problem := parseDrown([]byte(`{"provinces":[]}`), "海"); !strings.Contains(problem, "没有要变成海") {
		t.Errorf("sea: %q", problem)
	}
	if _, problem := parseFlood([]byte(`{"provinces":[]}`)); !strings.Contains(problem, "没有要变成湖") {
		t.Errorf("lake: %q", problem)
	}
}

func sinkWorld() sinkFacts {
	return sinkFacts{
		kind:      map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 10: 1, 11: 1, 12: 2},
		state:     map[int]int{1: 7, 2: 7, 3: 8, 4: 8, 5: 8},
		region:    map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 10: 20, 11: 21, 12: 1},
		stateName: map[int]string{7: "Seven", 8: "Eight"},
		points:    map[int]map[int]int{7: {1: 1, 2: 10}},
		capitals:  map[int][]string{7: {"AAA"}},
		divisions: map[int]int{},
		borders: map[int]map[int]int{
			1: {2: 5, 10: 4},
			2: {1: 5},
			3: {4: 3, 10: 2, 11: 9},
			4: {3: 3, 5: 3},
			5: {4: 3, 12: 6},
		},
	}
}

func TestSinkPlanGroupsAndPicksTheRegion(t *testing.T) {
	pieces, problem := sinkPlan([]int{1, 3, 4, 5}, sinkWorld())
	if problem != nil {
		t.Fatal(problem)
	}
	if len(pieces) != 2 {
		t.Fatalf("%+v", pieces)
	}
	if got := pieces[0]; len(got.provinces) != 1 || got.provinces[0] != 1 || got.region != 20 {
		t.Errorf("first piece %+v", got)
	}

	if got := pieces[1]; fmt.Sprint(got.provinces) != "[3 4 5]" || got.region != 21 {
		t.Errorf("second piece %+v", got)
	}
}

func TestSinkPlanRefuses(t *testing.T) {
	world := sinkWorld()

	check := func(name string, ids []int, says, code string, values map[string]any, marked []int) {
		t.Helper()
		_, problem := sinkPlan(ids, world)
		if problem == nil || !strings.Contains(problem.Detail, says) || problem.Code != code ||
			fmt.Sprint(problem.Values) != fmt.Sprint(values) || fmt.Sprint(problem.Provinces) != fmt.Sprint(marked) {
			t.Errorf("%s: %+v", name, problem)
		}
	}
	check("lake only", []int{5}, "不挨着海", problemSeaUnreached, map[string]any{"id": 5}, []int{5})
	check("inland", []int{4}, "省份 #4 不挨着海", problemSeaUnreached, map[string]any{"id": 4}, []int{4})
	check("capital", []int{1, 2}, "AAA 的首都", problemSeaCapital, map[string]any{"id": 2, "tags": "AAA", "state": "Seven"}, []int{2})
	world.divisions[3] = 2
	check("divisions", []int{3}, "开局有 2 个师驻在省份 #3", problemSeaDivisions, map[string]any{"id": 3, "count": 2}, []int{3})
}

func TestCapitalProvince(t *testing.T) {
	world := sinkWorld()
	if got := world.capitalProvince(7); got != 2 {
		t.Errorf("state 7: %d", got)
	}
	if got := world.capitalProvince(8); got != 0 {
		t.Errorf("state 8: %d", got)
	}
	if got := capitalFirst([]int{4, 5, 6}, 5); len(got) != 3 || got[0] != 5 {
		t.Errorf("capital first: %v", got)
	}
	if got := capitalFirst([]int{4, 6}, 5); len(got) != 2 || got[0] != 4 {
		t.Errorf("capital gone: %v", got)
	}
}
