package rawfiles

import (
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/hoi4/importsvc/characters"
	"haganeworks/internal/hoi4/importsvc/trees"
	"haganeworks/internal/hoi4/paradox"
)

func TestSingleImportPolicy(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		bad      bool
	}{{"", "overwrite", false}, {" SKIP ", "skip", false}, {"overwrite", "overwrite", false}, {"merge", "", true}} {
		got, err := singleImportPolicy(tc.in)
		if tc.bad != (err != nil) || got != tc.want {
			t.Fatalf("policy(%q) = %q, %v", tc.in, got, err)
		}
		if tc.bad {
			var typed apiutil.BadRequestError
			if !errors.As(err, &typed) {
				t.Fatalf("policy error type = %T", err)
			}
		}
	}
}

func TestSingleImportBatchPlanMatchesSequentialPythonUpsert(t *testing.T) {
	keys := []string{"old", "new", "new", "", "other"}
	overwrite := planSingleImportBatch(keys, map[string]bool{"old": true}, "overwrite")
	if overwrite.Created != 2 || overwrite.Updated != 2 || overwrite.Skipped != 0 ||
		!reflect.DeepEqual(overwrite.Selected, []int{0, 2, 4}) ||
		!reflect.DeepEqual(overwrite.Actions, []string{"updated", "created", "updated", "", "created"}) {
		t.Fatalf("overwrite plan = %#v", overwrite)
	}
	skip := planSingleImportBatch(keys, map[string]bool{"old": true}, "skip")
	if skip.Created != 2 || skip.Updated != 0 || skip.Skipped != 2 ||
		!reflect.DeepEqual(skip.Selected, []int{1, 4}) ||
		!reflect.DeepEqual(skip.Actions, []string{"skipped", "created", "skipped", "", "created"}) {
		t.Fatalf("skip plan = %#v", skip)
	}
}

func TestCharacterSingleImportSyntheticPythonContract(t *testing.T) {
	if _, _, err := characters.ImportCharactersStrict("characters = { X = { name = X"); err == nil ||
		!strings.Contains(err.Error(), "Expected RBRACE, got EOF") {
		t.Fatalf("strict standalone parse error = %v", err)
	}
	rows, _, err := characters.ImportCharacters(`characters = {
 GER_demo = {
  name = GER_demo_name
  portraits = { army = { large = "gfx/leaders/GER/demo.dds" } }
  corps_commander = { skill = 4 attack_skill = 3 }
  operative = { skill_level = 5 traits = { operative_demo } }
 }
 MTG_shared = { advisor = { slot = political_advisor cost = 90 } }
}`)
	if err != nil {
		t.Fatal(err)
	}
	native := characters.NativeEntities(rows)
	ger := characterImportRecord(native[0], "FRA")
	if ger["country_tag"] != "GER" || ger["skill"] != int64(4) || ger["large_portrait"] != "gfx/leaders/GER/demo.dds" {
		t.Fatalf("GER materialization drifted: %#v", ger)
	}
	if ger["operative_skill_level"] != int64(1) || len(ger["operative_traits"].([]any)) != 0 {
		t.Fatalf("operative legacy defaults drifted: %#v", ger)
	}
	mtg := characterImportRecord(native[1], "FRA")
	if mtg["country_tag"] != "FRA" {
		t.Fatalf("DLC prefix must not become a country owner: %#v", mtg)
	}
	snapshot := ger["_source_snapshot"].(map[string]any)
	if len(snapshot) != len(characterSnapshotKeys) || snapshot["tag"] != "GER_demo" || snapshot["is_replacement"] != true {
		t.Fatalf("character snapshot drifted: %#v", snapshot)
	}
}

func TestNativeTreeValueAndJSON(t *testing.T) {
	raw := trees.NewRawDict()
	raw.Set("unknown", paradox.Scalar{Kind: paradox.ScalarInt, Int: big.NewInt(7)})
	input := map[string]any{
		"raw_extras": raw,
		"path":       []map[string]any{{"leads_to_tech": "next", "research_cost_coeff": big.NewInt(2)}},
	}
	native := jsonValue(nativeTreeValue(input)).(map[string]any)
	if got := native["raw_extras"].(map[string]any)["unknown"]; got != json.Number("7") {
		t.Fatalf("raw extra = %#v", got)
	}
	if got := native["path"].([]any)[0].(map[string]any)["research_cost_coeff"]; got != json.Number("2") {
		t.Fatalf("path coeff = %#v", got)
	}
}

func TestMergeYearColumnsLastWinsStableSort(t *testing.T) {
	stored := []any{[]any{"@1936", json.Number("2")}, []any{"@1938", json.Number("4")},
		[]any{"@1940", json.Number("4")}, []any{"bad"}}
	got := mergeYearColumns(stored, []trees.YearColumn{
		{Name: "@1938", Row: big.NewInt(8)},
		{Name: "@1939", Row: big.NewInt(4)},
	})
	want := []orderedYear{{"@1936", 2}, {"@1940", 4}, {"@1939", 4}, {"@1938", 8}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("years = %#v, want %#v", got, want)
	}
}
