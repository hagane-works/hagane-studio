package clausewitz

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func jsonNorm(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestClausewitzParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "clausewitz_parity.json"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	var fx struct {
		Cases []struct {
			Name       string          `json:"name"`
			Text       string          `json:"text"`
			RootPath   []string        `json:"root_path"`
			SourceName *string         `json:"source_name"`
			Result     json.RawMessage `json:"result"`
		} `json:"cases"`
		Spans []struct {
			Text  string          `json:"text"`
			Start int             `json:"start"`
			End   int             `json:"end"`
			Span  json.RawMessage `json:"span"`
		} `json:"spans"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}

	for i, sc := range fx.Spans {
		got := jsonNorm(t, Span([]rune(sc.Text), sc.Start, sc.End))
		var want any
		if err := json.Unmarshal(sc.Span, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("span case %d (%q %d..%d)\n got %s\nwant %s",
				i, sc.Text, sc.Start, sc.End, gb, sc.Span)
		}
	}

	for _, c := range fx.Cases {
		got := jsonNorm(t, AnalyzeAssignments(c.Text, c.RootPath, c.SourceName).ToMap())
		var want any
		if err := json.Unmarshal(c.Result, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			gb, _ := json.Marshal(got)
			t.Errorf("case %s mismatch\n got %s\nwant %s", c.Name, gb, c.Result)
		}
	}
}

func TestOffsetsAreCodepointsNotBytes(t *testing.T) {
	src := `name = "德意志国" id = a`
	res := AnalyzeAssignments(src, nil, nil)
	if len(res.Assignments) != 2 {
		t.Fatalf("assignments=%d want 2", len(res.Assignments))
	}
	runes := []rune(src)
	for _, a := range res.Assignments {
		if a.Excerpt != string(runes[a.Span.StartOffset:a.Span.EndOffset]) {
			t.Errorf("key %q excerpt %q 与 span %d..%d 切出来的不一致(偏移不是码点?)",
				a.Key, a.Excerpt, a.Span.StartOffset, a.Span.EndOffset)
		}
	}

	second := res.Assignments[1]
	if second.Key != "id" || string(runes[second.Span.StartOffset:second.Span.StartOffset+2]) != "id" {
		t.Errorf("第二条赋值起点错位:key=%q start=%d", second.Key, second.Span.StartOffset)
	}
}

func TestPyFloatRepr(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{10, "10.0"}, {1.5, "1.5"}, {0.25, "0.25"}, {-3, "-3.0"},
		{1000, "1000.0"}, {1e16, "1e+16"}, {1e-05, "1e-05"}, {0.0001, "0.0001"},
	} {
		if got := pyFloatRepr(tc.in); got != tc.want {
			t.Errorf("pyFloatRepr(%v) = %q want %q", tc.in, got, tc.want)
		}
	}
}
