package cosmetic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"haganeworks/internal/gameindex"
)

func TestColorFileMergePreservesUnrelatedSourceBytes(t *testing.T) {
	keep := "# literal { } and TARGET = { fake }\r\n@factor = 0.7\r\nKEEP = { color = hsv { @factor 0.5 1 } note = \"quoted } \\\" {\" } # keep\r\n"
	base := "\ufeff" + keep + "TARGET = { color = rgb { 1 2 3 } }\r\n# between\r\nTARGET = { color = rgb { 4 5 6 } }\r\nTAIL = { color_ui = rgb { 9 9 9 } }\r\n"
	patch := "TARGET = { color = rgb { 156 72 220 } color_ui = rgb { 240 180 45 } }\nNEW = { color = rgb { 9 8 7 } }\nKEEP = { color = rgb { 0 0 0 } }"
	got, err := MergeColorFiles([]byte(base), []byte(patch), []string{"TARGET", "NEW"})
	if err != nil {
		t.Fatal(err)
	}
	want := "\ufeff" + keep + "TARGET = { color = rgb { 156 72 220 } color_ui = rgb { 240 180 45 } }\r\n# between\r\n\r\nTAIL = { color_ui = rgb { 9 9 9 } }\r\n\nNEW = { color = rgb { 9 8 7 } }\n"
	if string(got) != want {
		t.Fatalf("source bytes changed outside selected blocks:\n%q\nwant %q", got, want)
	}
	if strings.Count(string(got), "TARGET = {") != 2 {
		t.Fatal("expected one real target plus unchanged comment")
	}
}

func TestColorFileMergeMalformedNeverSilentlyDropsSource(t *testing.T) {
	for _, bad := range []string{"X = {", "}", "X = { note = \"unterminated }", "{ color = rgb { 1 2 3 } }"} {
		if _, err := MergeColorFiles([]byte(bad), []byte("X = { color = rgb { 1 2 3 } }"), []string{"X"}); err == nil {
			t.Errorf("accepted malformed base %q", bad)
		}
	}
	if _, err := MergeColorFiles(nil, []byte("X = {} X = {}"), []string{"X"}); err == nil {
		t.Fatal("duplicate replacement accepted")
	}
	if _, err := MergeColorFiles(nil, []byte("X = {}"), []string{"MISSING"}); err == nil {
		t.Fatal("missing replacement accepted")
	}
}

func TestVanillaColorBaselineReadAndPreserved(t *testing.T) {
	fixture := "\ufeff# base\r\nGER_X = { color = rgb { 1 2 3 } }\r\n"
	raw, _ := json.Marshal(map[string]any{"path": "common/countries/cosmetic.txt", "bytes": []byte(fixture)})
	t.Cleanup(gameindex.UseStatic(map[string][]byte{gameindex.Cosmetic: raw}))
	base, err := VanillaColorFile(context.Background())
	if err != nil || string(base) != fixture {
		t.Fatalf("baseline = %q, %v", base, err)
	}
	patch := []byte("MVP_TEST = { color = rgb { 156 72 220 } }")
	merged, err := MergeColorFiles(base, patch, []string{"MVP_TEST"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(merged, base) {
		t.Fatal("adding project tag changed existing vanilla bytes")
	}
	base[0] = '!'
	if again, _ := VanillaColorFile(context.Background()); again[0] == '!' {
		t.Fatal("caller mutated the index's baseline")
	}

	t.Cleanup(gameindex.SetDefault(nil))
	if _, err := VanillaColorFile(context.Background()); !errors.Is(err, gameindex.ErrNoGame) {
		t.Fatalf("no game folder = %v", err)
	}
}
