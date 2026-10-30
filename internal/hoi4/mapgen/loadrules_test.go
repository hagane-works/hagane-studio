package mapgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestLoadRulesFindWhatTheGameQuitsOn(t *testing.T) {
	b, regions := cutRegionBase(6, "sea", 0)
	whole := map[int][]int{10: {1}, 20: {2, 3}, 30: {5}, 40: {4, 6}}
	states := map[int][]int{1: {1}, 2: {5}}
	if got := b.LoadRules(whole, states); len(got) != 0 {
		t.Fatalf("a whole map: %+v", got)
	}
	for _, c := range []struct {
		code            string
		regions, states map[int][]int
	}{
		{"sea_region_split", regions, states},
		{"province_no_state", whole, map[int][]int{1: {1}}},
		{"province_no_region", map[int][]int{10: {1}, 20: {2, 3}, 40: {4, 6}}, states},
		{"state_ids_gap", whole, map[int][]int{1: {1}, 3: {5}}},
		{"state_two_regions", whole, map[int][]int{1: {1, 5}}},
		{"state_no_land", whole, map[int][]int{1: {1}, 2: {5}, 3: {6}}},
	} {
		got := b.LoadRules(c.regions, c.states)
		found := false
		for _, f := range got {
			if f.Severity != "error" {
				t.Errorf("%s: %s is %s, the game quits on it", c.code, f.Code, f.Severity)
			}
			found = found || f.Code == c.code
		}
		if !found {
			t.Errorf("%s not found: %+v", c.code, got)
		}
	}
}

func TestNewWorldsPassTheGamesLoadingChecks(t *testing.T) {
	_, islands := twoIslands(t)
	if got := islands.LoadRules(); len(got) != 0 {
		t.Fatalf("two islands: %+v", got)
	}
	m := generateTest(t, testOptions)
	world, err := NewWorld(m, GreyPalette(), GreyPalette(), WorldOptions{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	if got := world.LoadRules(); len(got) != 0 {
		t.Fatalf("test sketch: %+v", got)
	}
}

func statesIn(t *testing.T, dir string) map[int][]int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	idPattern := regexp.MustCompile(`\bid\s*=\s*(\d+)`)
	listPattern := regexp.MustCompile(`provinces\s*=\s*\{([^}]*)\}`)
	numbers := regexp.MustCompile(`\d+`)
	comments := regexp.MustCompile(`#[^\n]*`)
	out := map[int][]int{}
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := comments.ReplaceAllString(string(body), "")
		id := idPattern.FindStringSubmatch(text)
		list := listPattern.FindStringSubmatch(text)
		if id == nil || list == nil {
			continue
		}
		n, _ := strconv.Atoi(id[1])
		for _, s := range numbers.FindAllString(list[1], -1) {
			p, _ := strconv.Atoi(s)
			out[n] = append(out[n], p)
		}
	}
	return out
}

func TestBaseGamePassesTheGamesLoadingChecks(t *testing.T) {
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
	states := statesIn(t, filepath.Join(root, "history", "states"))
	if got := b.LoadRules(regions, states); len(got) != 0 {
		t.Fatalf("the base game: %+v", got)
	}
	if got := WorldOfBase(b, WorldOptions{Seed: 1}).LoadRules(); len(got) != 0 {
		t.Fatalf("the base game as a new world: %+v", got)
	}
}

func TestRestateOnTheBaseGameKeepsTheLoadingChecks(t *testing.T) {
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
	states := map[int]RestateState{}
	stateOf := map[int]int{}
	for id, provinces := range lists {
		states[id] = RestateState{Provinces: provinces, Owner: owners[id]}
		for _, p := range provinces {
			stateOf[p] = id
		}
	}
	neighbours := b.Neighbours()
	assign := map[int]string{}
	for p, state := range stateOf {
		for q := range neighbours[p] {
			if other, ok := stateOf[q]; ok && owners[other] != "" && owners[other] != owners[state] {
				assign[p] = owners[other]
				break
			}
		}
	}
	if len(assign) < 500 {
		t.Fatalf("only %d border provinces", len(assign))
	}
	after := func(plan RestatePlan) map[int][]int {
		out := map[int][]int{}
		for id, provinces := range lists {
			out[id] = provinces
		}
		for id, provinces := range plan.Lists {
			out[id] = provinces
		}
		next := len(lists)
		for _, n := range plan.New {
			next++
			out[next] = n.Provinces
		}
		return out
	}
	if got := b.LoadRules(regions, after(PlanRestate(states, neighbours, assign, regionOf, b.IsLand()))); len(got) != 0 {
		t.Fatalf("painted %d provinces, regions kept to: %+v", len(assign), got)
	}
	found := false
	for _, f := range b.LoadRules(regions, after(PlanRestate(states, neighbours, assign, nil, nil))) {
		found = found || f.Code == "state_two_regions"
	}
	if !found {
		t.Fatal("without regions the painting should have put states across regions")
	}
}

func ownersIn(t *testing.T, dir string) map[int]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	idPattern := regexp.MustCompile(`\bid\s*=\s*(\d+)`)
	ownerPattern := regexp.MustCompile(`\bowner\s*=\s*([A-Za-z0-9]{3})\b`)
	comments := regexp.MustCompile(`#[^\n]*`)
	out := map[int]string{}
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := comments.ReplaceAllString(string(body), "")
		id := idPattern.FindStringSubmatch(text)
		owner := ownerPattern.FindStringSubmatch(text)
		if id == nil || owner == nil {
			continue
		}
		n, _ := strconv.Atoi(id[1])
		out[n] = owner[1]
	}
	return out
}
