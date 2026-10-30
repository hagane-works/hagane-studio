package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestPerfPrimitivesParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "perf_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Text        string   `json:"text"`
			Iterators   []string `json:"iterators"`
			Nested      []string `json:"nested"`
			Days        bool     `json:"days"`
			Factor      bool     `json:"factor"`
			MonthsYears bool     `json:"months_years"`
			Fanout      []string `json:"fanout"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		iters := perfIteratorNames(c.Text)
		if iters == nil {
			iters = []string{}
		}
		want := c.Iterators
		if want == nil {
			want = []string{}
		}
		if !reflect.DeepEqual(iters, want) {
			t.Errorf("%q iterators %v want %v", c.Text, iters, want)
		}
		outer, inner, ok := perfNestedIterator(c.Text)
		if c.Nested == nil {
			if ok {
				t.Errorf("%q nested (%s,%s) want none", c.Text, outer, inner)
			}
		} else if !ok || outer != c.Nested[0] || inner != c.Nested[1] {
			t.Errorf("%q nested (%s,%s,%v) want %v", c.Text, outer, inner, ok, c.Nested)
		}
		if perfDaysRe.MatchString(c.Text) != c.Days ||
			perfFactorRe.MatchString(c.Text) != c.Factor ||
			perfMYRe.MatchString(c.Text) != c.MonthsYears {
			t.Errorf("%q mtth regexes mismatch", c.Text)
		}
		var fanout []string
		seen := map[string]bool{}
		for _, m := range perfFanoutRe.FindAllStringSubmatch(c.Text, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				fanout = append(fanout, m[1])
			}
		}
		sort.Strings(fanout)
		if fanout == nil {
			fanout = []string{}
		}
		wantF := c.Fanout
		if wantF == nil {
			wantF = []string{}
		}
		if !reflect.DeepEqual(fanout, wantF) {
			t.Errorf("%q fanout %v want %v", c.Text, fanout, wantF)
		}
	}
}
