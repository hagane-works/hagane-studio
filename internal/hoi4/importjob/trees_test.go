package importjob

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
)

func TestGuessTechLineLongestAndFallback(t *testing.T) {
	if line, folder := guessTechLine("my_mtg_naval_support_patch.txt"); line != "naval_support" || folder != "mtgnavalsupportfolder" {
		t.Fatalf("got %q %q", line, folder)
	}
	if line, folder := guessTechLine("custom_rocket.txt"); line != "custom_rocket" || folder != "custom_rocket_folder" {
		t.Fatalf("got %q %q", line, folder)
	}
}

func TestJSONValueNormalisesParserTypes(t *testing.T) {
	raw := trees.NewRawDict()
	raw.Set("number", paradox.Scalar{Kind: paradox.ScalarInt, Int: big.NewInt(7)})
	raw.Set("repeat", []any{paradox.Scalar{Kind: paradox.ScalarStr, Str: "x"}, big.NewInt(8)})
	b, err := json.Marshal(jsonValue(raw))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"number":7,"repeat":["x",8]}` {
		t.Fatalf("unexpected %s", b)
	}
}

func TestFocusBundlePayloadRoutesOwnAndForeign(t *testing.T) {
	text := `focus_tree = { id = GER_test country = { factor = 0 modifier = { add = 10 tag = GER } } focus = { id = GER_a x = 1 y = 2 } }`
	bundles, _, err := trees.ImportFocusFile(text, "common/national_focus/test.txt")
	if err != nil || len(bundles) != 1 {
		t.Fatalf("parse: %v %#v", err, bundles)
	}
	payload, keep, err := focusBundlePayload(bundles[0], "common/national_focus/test.txt", "GER", map[string]bool{"GER": true})
	if err != nil || !keep {
		t.Fatalf("route: %v keep=%v", err, keep)
	}
	var got map[string]any
	if err = json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got["country_tag"] != "GER" || got["scope_kind"] != "own" {
		t.Fatalf("payload %#v", got)
	}
	if _, keep, _ = focusBundlePayload(bundles[0], "x", "FRA", map[string]bool{"FRA": true}); keep {
		t.Fatal("foreign own tree was retained")
	}
}

func TestFocusBundlePayloadPreservesInvalidInitialPosition(t *testing.T) {
	text := `focus_tree = {
		id = test
		initial_show_position = { focus = GER_opening x = 1 y = 2 }
	}`
	bundles, summary, err := trees.ImportFocusFile(
		text, "common/national_focus/test.txt")
	if err != nil || len(bundles) != 1 {
		t.Fatalf("parse: %v %#v", err, bundles)
	}
	if len(summary.Warnings) != 1 {
		t.Fatalf("warnings=%#v", summary.Warnings)
	}
	payload, keep, err := focusBundlePayload(
		bundles[0], "common/national_focus/test.txt", "", nil)
	if err != nil || !keep {
		t.Fatalf("payload: %v keep=%v", err, keep)
	}
	var got map[string]any
	if err = json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	position, ok := got["initial_show_position"].(map[string]any)
	if !ok || position["kind"] != "invalid" || position["raw"] == "" ||
		position["raw"] != position["raw_text"] {
		t.Fatalf("initial_show_position=%#v", got["initial_show_position"])
	}
}

func TestTreeStageOptionsExplicitCountryTag(t *testing.T) {
	opts := TreeStageOptions{ProjectID: 7, CountryTag: " fra ", ImportFocus: true}
	if got := strings.ToUpper(strings.TrimSpace(opts.CountryTag)); got != "FRA" {
		t.Fatalf("got %q", got)
	}
	if !opts.ImportFocus || opts.ImportTech {
		t.Fatalf("flag contract changed: %#v", opts)
	}
}

func TestFocusFileInSubfolderIsSkipped(t *testing.T) {
	tree := func(tree, focus string) string {
		return "focus_tree = { id = " + tree + " country = { factor = 0 modifier = { add = 10 tag = TST } } focus = { id = " + focus + " x = 1 y = 0 cost = 10 } }\n"
	}
	root := writeWorldTestMod(t, map[string]string{
		"common/national_focus/TST.txt":                       tree("tst_focus", "TST_a"),
		"common/national_focus/FOR HOTFIX/TST_notes.txt":      tree("tst_notes", "TST_a"),
		"common/national_focus/Old Trees/Nested/TST_0_35.txt": tree("tst_old", "TST_b"),
	})
	p := newWorldTestProject(t)
	m := NewManager()
	if _, err := m.Start(p, "subfolders", root, AllOptions()); err != nil {
		t.Fatal(err)
	}
	state := waitForJob(t, m, p.ID(), "subfolders")
	if state.Status != "complete" {
		t.Fatalf("import %s: %s", state.Status, state.Error)
	}
	trees, _ := p.Rows("focus_trees")
	if len(trees) != 1 || trees[0]["tree_id"] != "tst_focus" {
		t.Errorf("focus trees = %v", trees)
	}
	warnings, _ := state.Result["warnings"].([]string)
	want := map[string]bool{
		`§§not_loaded_by_game§§{"file": "common/national_focus/FOR HOTFIX/TST_notes.txt", "folder": "common/national_focus"}§§`:      true,
		`§§not_loaded_by_game§§{"file": "common/national_focus/Old Trees/Nested/TST_0_35.txt", "folder": "common/national_focus"}§§`: true,
	}
	for _, warning := range warnings {
		for prefix := range want {
			if strings.HasPrefix(warning, prefix) {
				delete(want, prefix)
			}
		}
	}
	if len(want) != 0 {
		t.Errorf("missing warnings %v in %v", want, warnings)
	}
}
