package worldmap

import (
	"testing"

	"haganeworks/internal/project"
	"haganeworks/internal/reftables"
)

func TestSeededLeadersWithAnEmptyIdeologyTable(t *testing.T) {
	seed := func() int {
		e := newTestEnv(t)
		var ids []int64
		err := e.p.Write([]string{characterTable}, func(tx *project.Tx) error {
			tx.Insert(characterTable, project.Row{"country_tag": "TST", "tag": "TST_leader", "role_type": "country_leader",
				"ideology": "despotism", "source": "created"})
			ids = seedWorldMapLeaders(tx, []string{"TST", "ABC"})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return len(ids)
	}
	if got := seed(); got != 1 {
		t.Fatalf("with the built-in table %d leaders were seeded, want ABC's only", got)
	}
	defer reftables.Use(map[string][]byte{reftables.IdeologiesFile: []byte(`{"groups": [], "sub_ideologies": []}`)})()
	if got := seed(); got != 2 {
		t.Fatalf("with an empty table %d leaders were seeded, want both countries'", got)
	}
}
