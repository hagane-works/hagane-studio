package mapgen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func shoreProvince(t *testing.T) (*Base, int) {
	b, _ := floodWorld(t)
	kinds := b.kinds()
	for _, d := range b.Definitions {
		if d.Kind != "land" || !d.Coastal {
			continue
		}
		for other := range b.Borders([]int{d.ID})[d.ID] {
			if kinds[other] == kindSea {
				return b, d.ID
			}
		}
	}
	t.Fatal("no coastal land province")
	return nil, 0
}

func TestSinkMakesASea(t *testing.T) {
	b, target := shoreProvince(t)
	neighbours := b.Borders([]int{target})[target]
	add, err := b.Sink([]int{target})
	if err != nil {
		t.Fatal(err)
	}
	rows := map[int]Definition{}
	for _, d := range b.Definitions {
		rows[d.ID] = d
	}
	if d := rows[target]; d.Kind != "sea" || d.Terrain != "ocean" || d.Continent != 0 {
		t.Fatalf("row %+v", d)
	}
	for _, p := range add.Pixels(target) {
		if b.Heights[p] > SeaHeight || b.Terrain[p] != Legend[Sea].Terrain || b.Rivers[p] != RiverWater {
			t.Fatalf("pixel %d: height %d terrain %d river %d", p, b.Heights[p], b.Terrain[p], b.Rivers[p])
		}
	}
	landBeside := 0
	for id := range neighbours {
		if rows[id].Kind == "land" {
			landBeside++
			if !rows[id].Coastal {
				t.Errorf("land #%d beside the new sea is not coastal", id)
			}
		}
	}
	if landBeside == 0 {
		t.Fatal("the test province has no land beside it")
	}
	if !rows[target].Coastal {
		t.Error("the new sea touches land but is not coastal")
	}
	for _, id := range add.NowCoastal {
		if rows[id].Kind != "land" {
			t.Errorf("now-coastal #%d is %s", id, rows[id].Kind)
		}
	}

	kinds := b.kinds()
	var seas []int
	for _, d := range b.Definitions {
		if d.Kind == "sea" {
			seas = append(seas, d.ID)
		}
	}
	borders := b.Borders(seas)
	for _, id := range seas {
		touches := false
		for other := range borders[id] {
			if kinds[other] == kindLand {
				touches = true
			}
		}
		if rows[id].Coastal != touches {
			t.Errorf("sea #%d coastal %v, touches land %v", id, rows[id].Coastal, touches)
		}
	}
	if _, err := b.Sink([]int{target}); err == nil || !strings.Contains(err.Error(), "已经是水面") {
		t.Errorf("sinking a sea again: %v", err)
	}
}

func TestBordersAreSymmetric(t *testing.T) {
	b, target := shoreProvince(t)
	mine := b.Borders([]int{target})[target]
	if len(mine) == 0 {
		t.Fatal("no borders")
	}
	for other, n := range mine {
		if back := b.Borders([]int{other})[other][target]; back != n {
			t.Errorf("#%d→#%d %d, back %d", target, other, n, back)
		}
	}
}

func TestSinkPositions(t *testing.T) {
	b, target := shoreProvince(t)
	add, err := b.Sink([]int{target})
	if err != nil {
		t.Fatal(err)
	}
	stacks := b.UnitStacks([]int{target})
	if len(stacks) != len(unitStackTypes) || !strings.HasPrefix(stacks[0], strconv.Itoa(target)+";") {
		t.Fatalf("unit stacks %v", stacks)
	}
	if len(add.NowCoastal) == 0 {
		t.Skip("the new sea made no land coastal on this map")
	}
	stateOf := map[int]int{add.NowCoastal[0]: 5}
	text, added := b.AddCoastalBuildings("", add.NowCoastal[:1], stateOf, 1)
	if added != len(coastalBuildings) || !strings.Contains(text, "5;naval_base_spawn;") {
		t.Fatalf("added %d:\n%s", added, text)
	}
	for _, line := range strings.Split(text, "\n") {
		if fields := strings.Split(line, ";"); len(fields) == 7 && fields[1] == "naval_base_spawn" && fields[6] == "0" {
			t.Errorf("naval base without its sea: %s", line)
		}
	}
	again, more := b.AddCoastalBuildings(text, add.NowCoastal[:1], stateOf, 1)
	if more != 0 || again != text {
		t.Errorf("added again: %d", more)
	}
}

func TestSinkOnTheBaseGameMap(t *testing.T) {
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
	codes := func() map[string]bool {
		out := map[string]bool{}
		for _, finding := range Check(b.CheckInput(nil)) {
			out[finding.Severity+" "+finding.Code] = true
		}
		return out
	}
	before := codes()
	all := make([]int, len(b.Rivers))
	for p := range all {
		all[p] = p
	}
	faultsBefore := riverFaults(b.Rivers, b.Width, b.Height, all)
	kinds := b.kinds()
	var coastal []int
	for _, d := range b.Definitions {
		if d.Kind == "land" && d.Coastal {
			coastal = append(coastal, d.ID)
		}
	}
	borders := b.Borders(coastal)
	var chosen []int
	for _, id := range coastal {
		touches := false
		for other := range borders[id] {
			if kinds[other] == kindSea {
				touches = true
			}
		}
		if touches && len(chosen) < 5 && id%400 < 4 {
			chosen = append(chosen, id)
		}
	}
	if len(chosen) == 0 {
		t.Fatal("no coastal province chosen")
	}
	add, err := b.Sink(chosen)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sank %v; now coastal %v", chosen, add.NowCoastal)
	for code := range codes() {
		if !before[code] {
			t.Errorf("new finding after sinking: %s", code)
		}
	}
	kinds = b.kinds()
	coastalOf := map[int]bool{}
	for _, d := range b.Definitions {
		if d.Kind == "sea" {
			coastalOf[d.ID] = d.Coastal
		}
	}
	seaBorders := b.Borders(chosen)
	for _, id := range chosen {
		touches := false
		for other := range seaBorders[id] {
			if kinds[other] == kindLand {
				touches = true
			}
		}
		if coastalOf[id] != touches {
			t.Errorf("new sea #%d coastal %v, touches land %v", id, coastalOf[id], touches)
		}
	}
	if n := riverFaults(b.Rivers, b.Width, b.Height, all); n > faultsBefore {
		t.Errorf("river faults %d → %d", faultsBefore, n)
	}
}
