package worldmap

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseRestate(t *testing.T) {
	req, assign, problem := parseRestate([]byte(`{"provinces": {"12": "ger", "13": "ENG"}, "add_cores": true}`))
	if problem != "" || !req.AddCores || req.Date != "1936.1.1" || len(req.Languages) != 1 ||
		!reflect.DeepEqual(assign, map[int]string{12: "GER", 13: "ENG"}) {
		t.Fatalf("parsed %+v %v %q", req, assign, problem)
	}
	for want, body := range map[string]string{
		"没有要改的省份": `{"provinces": {}}`,
		"省份编号不对":  `{"provinces": {"x": "GER"}}`,
		"三个字母":    `{"provinces": {"5": "GERMANY"}}`,
		"读不懂这次改动": `{`,
	} {
		if _, _, problem := parseRestate([]byte(body)); !strings.Contains(problem, want) {
			t.Errorf("%s: %q, want %q", body, problem, want)
		}
	}
}

func TestProvinceBlocks(t *testing.T) {
	text := "6449 = {\n\tnaval_base = 3\n\tcoastal_bunker = 1\n}\n" +
		"# Paris landmark\n9523 = {\n\tlandmark_eiffel_tower = {\n\t\tlevel = 1\n\t\tallowed = { has_dlc = \"Gotterdammerung {x}\" }\n\t}\n}\n" +
		"some_flag = yes\n" +
		"485 = { naval_base = 1 } # a note\n"
	blocks, rest := provinceBlocks(text)
	if len(blocks) != 3 {
		t.Fatalf("blocks %v", blocks)
	}
	if blocks[6449] != "6449 = {\n\tnaval_base = 3\n\tcoastal_bunker = 1\n}" {
		t.Errorf("6449: %q", blocks[6449])
	}
	if !strings.HasPrefix(blocks[9523], "# Paris landmark\n9523 = {") || !strings.HasSuffix(blocks[9523], "\t}\n}") {
		t.Errorf("9523: %q", blocks[9523])
	}
	if blocks[485] != "485 = { naval_base = 1 }" {
		t.Errorf("485: %q", blocks[485])
	}
	if !reflect.DeepEqual(rest, []string{"some_flag = yes", "# a note"}) {
		t.Errorf("rest %q", rest)
	}
	if b, r := provinceBlocks(""); len(b) != 0 || len(r) != 0 {
		t.Errorf("empty: %v %v", b, r)
	}
}

func TestParseTerrain(t *testing.T) {
	req, terrains, problem := parseTerrain([]byte(`{"provinces": {"12": "Mountain", "13": "forest"}, "look": true}`))
	if problem != "" || !req.Look || !reflect.DeepEqual(terrains, map[int]string{12: "mountain", 13: "forest"}) {
		t.Fatalf("parsed %+v %v %q", req, terrains, problem)
	}
	for want, body := range map[string]string{
		"没有要改的省份": `{"provinces": {}}`,
		"省份编号不对":  `{"provinces": {"-3": "plains"}}`,
		"不认识的地形":  `{"provinces": {"5": "ocean"}}`,
	} {
		if _, _, problem := parseTerrain([]byte(body)); !strings.Contains(problem, want) {
			t.Errorf("%s: %q, want %q", body, problem, want)
		}
	}
}

func TestParseRivers(t *testing.T) {
	pixels, problem := parseRivers([]byte(`{"pixels":[[10,0],[11,3],[12,11],[13,255]]}`))
	if problem != "" || len(pixels) != 4 || pixels[10] != 0 || pixels[12] != 11 || pixels[13] != 255 {
		t.Fatalf("%v %q", pixels, problem)
	}
	for body, says := range map[string]string{
		`{"pixels":[]}`:        "没有要改的河流",
		`{"pixels":[[1,12]]}`:  "没有这个值",
		`{"pixels":[[1,254]]}`: "没有这个值",
		`{"pixels":[[-1,3]]}`:  "位置不对",
		`nope`:                 "读不懂",
	} {
		if _, problem := parseRivers([]byte(body)); !strings.Contains(problem, says) {
			t.Errorf("%s: %q", body, problem)
		}
	}
	if got := riverSteps([]int{5, 7, 20}); got[0] != 5 || got[1] != 2 || got[2] != 13 {
		t.Errorf("steps %v", got)
	}
}

func TestParseFlood(t *testing.T) {
	req, problem := parseFlood([]byte(`{"provinces":[30,10,30,20]}`))
	if problem != "" || len(req.Provinces) != 3 || req.Provinces[0] != 10 || req.Date != "1936.1.1" {
		t.Fatalf("%+v %q", req, problem)
	}
	for body, says := range map[string]string{`{"provinces":[]}`: "没有要变成湖", `{"provinces":[0]}`: "编号不对", `x`: "读不懂"} {
		if _, problem := parseFlood([]byte(body)); !strings.Contains(problem, says) {
			t.Errorf("%s: %q", body, problem)
		}
	}
}

func TestFloodPlan(t *testing.T) {
	provinces := [][]any{
		{1, 0, 0, 0, "plains", 1, 7, 1},
		{2, 0, 0, 0, "plains", 1, 7, 1},
		{3, 0, 0, 0, "plains", 1, 8, 1},
		{4, 0, 1, 0, "ocean", 0, 0, 2},
	}
	states := [][]any{{7, "Seven", "AAA"}, {8, "Eight", "AAA"}}
	gone, touched, problem := floodPlan(provinces, states, []int{1})
	if problem != nil || !gone[1] || len(touched) != 1 || touched[0] != 7 {
		t.Fatalf("%v %v %v", gone, touched, problem)
	}
	for _, c := range []struct {
		ids       []int
		says      string
		code      string
		provinces []int
	}{
		{[]int{3}, "「Eight」会整个被淹掉", problemStateDrowned, []int{3}},
		{[]int{1, 2}, "「Seven」会整个被淹掉", problemStateDrowned, []int{1, 2}},
		{[]int{4}, "已经是水面", "already_water", []int{4}},
		{[]int{9}, "没有省份 #9", "no_province", nil},
	} {
		_, _, problem := floodPlan(provinces, states, c.ids)
		if problem == nil || !strings.Contains(problem.Detail, c.says) || problem.Code != c.code ||
			fmt.Sprint(problem.Provinces) != fmt.Sprint(c.provinces) {
			t.Errorf("%v: %+v", c.ids, problem)
		}
	}
}

func TestKeepBlocks(t *testing.T) {
	text := "11 = { naval_base = 3 }\n# bunker\n12 = { bunker = 1 }\nextra = 1"
	if got := keepBlocks(text, map[int]bool{99: true}); got != text {
		t.Errorf("untouched: %q", got)
	}
	got := keepBlocks(text, map[int]bool{11: true})
	if strings.Contains(got, "naval_base") || !strings.Contains(got, "12 = { bunker = 1 }") || !strings.Contains(got, "extra = 1") {
		t.Errorf("trimmed: %q", got)
	}
}
