package exportengine

import (
	"encoding/json"
	"os"
	"testing"

	"haganeworks/internal/hoi4/emit/worldout"
)

func TestPB03CountryLeaderIsRecruitedToItsCountry(t *testing.T) {
	raw, err := os.ReadFile("testdata/p0b_pb03_country_leader_lifecycle.json")
	if err != nil {
		t.Fatalf("read PB-03 recruit fixture: %v", err)
	}
	var fixture struct {
		Create          json.RawMessage `json:"create"`
		CharacterCreate json.RawMessage `json:"character_create"`
		Expect          struct {
			CountryTag   string `json:"country_tag"`
			CharacterTag string `json:"character_tag"`
		} `json:"expect"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode PB-03 recruit fixture: %v", err)
	}
	recruit, replacements, err := BuildRecruitAndLeaderMaps(Snapshot{
		Schema: SnapshotSchema, ProjectID: 303, Project: fixture.Create,
		Tables: map[string][]json.RawMessage{"characters": {fixture.CharacterCreate}},
	}, false)
	if err != nil {
		t.Fatalf("build PB-03 recruit map: %v", err)
	}
	value, ok := recruit.Get(fixture.Expect.CountryTag)
	if !ok {
		t.Fatalf("PB-03 recruit map omitted country %s", fixture.Expect.CountryTag)
	}
	tags, ok := value.([]worldout.Value)
	if !ok || len(tags) != 1 || tags[0] != worldout.Value(fixture.Expect.CharacterTag) {
		t.Fatalf("PB-03 recruit map drift: %#v", value)
	}
	if replacements.Len() != 0 {
		t.Fatalf("PB-03 new leader was mistaken for vanilla replacement: %#v", replacements)
	}
}
