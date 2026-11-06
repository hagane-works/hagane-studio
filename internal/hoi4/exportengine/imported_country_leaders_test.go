package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// Imported characters keep their leader roles in their own definitions. They
// are not moved into a trait-overlay event that retires and re-creates
// leaders under the project's tag.
func TestImportedCountryLeadersStayInTheirDefinitions(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"DPR","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"characters": {
				json.RawMessage(`{"id":1,"tag":"AFA_sultan","country_tag":"AFA","role_type":"country_leader","ideology":"fds_095","traits":["rich_sultan"],"expire":"1944.1.1.1","source":"imported","source_file":"common/characters/AFA.txt","is_replacement":true,"replaces_vanilla_id":true}`),
				json.RawMessage(`{"id":2,"tag":"SPR_general","country_tag":"SPR","role_type":"general","ideology":"fds_090","expire":"1950.1.1","traits":["brilliant_strategist"],"source":"imported","source_file":"common/characters/SPR.txt","is_replacement":true,"replaces_vanilla_id":true}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[strings.TrimPrefix(entry.Path, "Demo/")] = string(entry.Bytes)
		if strings.Contains(entry.Path, "trait_overlay") || strings.Contains(string(entry.Bytes), "create_country_leader") {
			t.Errorf("%s re-creates imported leaders", entry.Path)
		}
	}
	afa := contents["common/characters/AFA.txt"]
	for _, want := range []string{"country_leader = {", "ideology = fds_095", "rich_sultan", `expire = "1944.1.1.1"`} {
		if !strings.Contains(afa, want) {
			t.Errorf("AFA_sultan lost %q:\n%s", want, afa)
		}
	}
	spr := contents["common/characters/SPR.txt"]
	if !strings.Contains(spr, "country_leader = {") || !strings.Contains(spr, "ideology = fds_090") {
		t.Errorf("the general who is also head of state lost his country_leader role:\n%s", spr)
	}
}
