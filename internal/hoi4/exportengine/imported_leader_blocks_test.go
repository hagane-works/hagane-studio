package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/importsvc/characters"
)

// A character can lead its country under several ideologies, each with its
// own traits, and can hold another main role. Import then export writes
// every country_leader block as it was.
func TestImportedLeaderBlocksRoundTripAsWritten(t *testing.T) {
	source := `characters = {
	AFA_sultan = {
		name = AFA_sultan
		field_marshal = { skill = 3 attack_skill = 2 defense_skill = 2 planning_skill = 3 logistics_skill = 2 }
		country_leader = {
			ideology = fds_095_personalist_dictatorship
			traits = { rich_sultan }
			expire = "1944.1.1.1"
		}
		country_leader = {
			ideology = fds_090_absolute_monarchy
			traits = { pious }
		}
	}
}
`
	parsed, _, err := characters.ImportCharacters(source)
	if err != nil {
		t.Fatal(err)
	}
	rows := characters.NativeEntities(parsed)
	if len(rows) != 1 {
		t.Fatalf("characters = %d", len(rows))
	}
	row := rows[0]
	row["id"], row["country_tag"], row["source"], row["source_file"] = 1, "AFA", "imported", "common/characters/AFA.txt"
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	created := json.RawMessage(`{"id":2,"tag":"AFA_created","country_tag":"AFA","role_type":"country_leader","ideology":"fds_095","source":"created","raw_extras":{"country_leader":"ideology = stale"}}`)
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"DPR","export_languages":["en"]}`),
		Tables:    map[string][]json.RawMessage{"characters": {raw, created}},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	sultan, createdBlock := "", ""
	for _, entry := range artifact.Entries {
		if !strings.Contains(entry.Path, "/common/characters/") {
			continue
		}
		if strings.Contains(string(entry.Bytes), "AFA_sultan") {
			sultan = string(entry.Bytes)
		} else if strings.Contains(string(entry.Bytes), "AFA_created") {
			createdBlock = string(entry.Bytes)
		}
	}
	afa := sultan
	if got := strings.Count(sultan, "country_leader = {"); got != 2 {
		t.Errorf("AFA_sultan has %d country_leader blocks, want 2:\n%s", got, afa)
	}
	for _, want := range []string{"fds_095_personalist_dictatorship", "rich_sultan", "1944.1.1.1", "fds_090_absolute_monarchy", "pious"} {
		if !strings.Contains(sultan, want) {
			t.Errorf("AFA_sultan lost %q:\n%s", want, afa)
		}
	}
	if got := strings.Count(createdBlock, "country_leader = {"); got != 1 || strings.Contains(createdBlock, "stale") {
		t.Errorf("a character made in the app has %d leader blocks or used stale raw text:\n%s", got, createdBlock)
	}
}
