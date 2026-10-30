package worldmap

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/mapgen"
)

func TestMapProblemResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeMapProblem(rec, 422, mapProblemOf(mapgen.WaterTerrain(12)))
	var out struct {
		Code      string         `json:"code"`
		Detail    string         `json:"detail"`
		Values    map[string]any `json:"values"`
		Provinces []int          `json:"provinces"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 422 || out.Code != "water_terrain" || out.Detail != "省份 #12 是水面,只有陆地能换地形" ||
		out.Values["id"] != 12.0 || len(out.Provinces) != 1 || out.Provinces[0] != 12 {
		t.Errorf("%d %+v", rec.Code, out)
	}
}

func TestRequestProblem(t *testing.T) {
	for detail, code := range map[string]string{
		"一次改的省份太多":   problemTooLarge,
		"新土地的范围太大了":  problemTooLarge,
		"没有要改的省份":    problemNothing,
		"没有收到草图":     problemNothing,
		"读不懂这次改动":    problemBadRequest,
		"省份编号不对:abc": problemBadRequest,
	} {
		if got := requestProblem(detail); got.Code != code || got.Detail != detail {
			t.Errorf("%s: %+v", detail, got)
		}
	}

	if got := mapProblemOr(errString("boom"), problemUnreadable); got.Code != problemUnreadable || got.Detail != "boom" {
		t.Errorf("other: %+v", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestEveryMapProblemCodeHasWords(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "messages", "en", "mapStudio.json"))
	if err != nil {
		t.Skip("frontend messages not here")
	}
	var messages struct {
		Problem map[string]string `json:"problem"`
	}
	if err := json.Unmarshal(raw, &messages); err != nil {
		t.Fatal(err)
	}
	codes := map[string]string{}
	pattern := regexp.MustCompile(`(?:Code:\s+"|newMapProblem\("|problem\w+\s+=\s+")([a-z_]+)"`)
	for _, glob := range []string{"*.go", filepath.Join("..", "..", "hoi4", "mapgen", "*.go")} {
		files, _ := filepath.Glob(glob)
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range pattern.FindAllStringSubmatch(string(body), -1) {
				codes[m[1]] = file
			}
		}
	}
	if len(codes) < 20 {
		t.Fatalf("found only %d codes: %v", len(codes), codes)
	}

	listed := map[string]bool{}
	if ts, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "src", "app", "[locale]", "game-view", "map-studio", "mapProblems.ts")); err == nil {
		block := string(ts)
		if start := strings.Index(block, "MAP_PROBLEM_CODES = ["); start >= 0 {
			block = block[start:]
			block = block[:strings.Index(block, "] as const")]
			for _, m := range regexp.MustCompile(`"([a-z_]+)"`).FindAllStringSubmatch(block, -1) {
				listed[m[1]] = true
			}
		}
	}
	if len(listed) == 0 {
		t.Fatal("MAP_PROBLEM_CODES not found in mapProblems.ts")
	}
	for code, file := range codes {
		if messages.Problem[code] == "" {
			t.Errorf("%s (in %s) has no words in mapStudio.problem", code, file)
		}
		if !listed[code] {
			t.Errorf("%s (in %s) is not in mapProblems.ts MAP_PROBLEM_CODES: the studio would not say it", code, file)
		}
	}
}

func problemDetail(p *mapProblem) string {
	if p == nil {
		return ""
	}
	return p.Detail
}
