package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc/characters"
)

// Import then export keeps what an imported character was written with:
// its portraits block, legacy_id, advisor ledger, ai_will_do and
// can_be_fired, a name that is not its own key, and no allowed block the
// mod did not write.
func TestImportedCharacterFieldsRoundTrip(t *testing.T) {
	source := `characters = {
	BUL_ivan_markov = {
		name = BUL_ivan_markov
		portraits = {
			army = {
				large = "GFX_portrait_bul_ivan_markov"
				small = GFX_portrait_BUL_ivan_markov_small
			}
		}
		corps_commander = {
			traits = { trait_reckless }
			skill = 3
			attack_skill = 4
			defense_skill = 2
			planning_skill = 3
			logistics_skill = 3
			legacy_id = 1205
		}
	}
	BUL_adviser = {
		name = "Petar Petrov"
		portraits = {
			civilian = {
				small = "GFX_idea_generic_political"
			}
		}
		advisor = {
			slot = political_advisor
			idea_token = BUL_adviser
			ledger = civilian
			can_be_fired = no
			traits = { silent_workhorse }
			ai_will_do = {
				factor = 1.000
				modifier = {
					factor = 0
					has_country_flag = BUL_no_advisers
				}
			}
		}
	}
	BUL_other_key = {
		name = BUL_named_elsewhere
		advisor = {
			slot = theorist
			idea_token = BUL_other_key
			allowed = { original_tag = BUL }
			traits = { military_theorist }
		}
	}
}
`
	parsed, _, err := characters.ImportCharacters(source)
	if err != nil {
		t.Fatal(err)
	}
	var rows []json.RawMessage
	for i, row := range characters.NativeEntities(parsed) {
		// What import stores (importjob charsExtrasMergeCharacters).
		row["id"], row["country_tag"], row["source"], row["source_file"] = i+1, "BUL", "imported", "common/characters/BUL.txt"
		raw, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, raw)
	}
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"DPR","export_languages":["en"]}`),
		Tables:    map[string][]json.RawMessage{"characters": rows},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	out := ""
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, "common/characters/BUL.txt") {
			out = string(entry.Bytes)
		}
	}
	if out == "" {
		t.Fatal("no common/characters/BUL.txt")
	}
	for _, want := range []string{
		"legacy_id = 1205",
		"ledger = civilian",
		"can_be_fired = no",
		"has_country_flag = BUL_no_advisers",
		`name = "Petar Petrov"`,
		"name = BUL_named_elsewhere",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lost %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"legacy_id = -1", "civilian = {\n\t\t\t\tlarge", "army = {\n\t\t\t\tsmall = \"GFX_idea_generic_political\""} {
		if strings.Contains(out, unwanted) {
			t.Errorf("export added %q:\n%s", unwanted, out)
		}
	}
	if got := strings.Count(out, "original_tag = BUL"); got != 1 {
		t.Errorf("allowed original_tag written %d times, want only the one the mod wrote:\n%s", got, out)
	}
	if got := strings.Count(out, "portraits = {"); got != 2 {
		t.Errorf("portraits blocks = %d, want the two the mod wrote:\n%s", got, out)
	}
}
