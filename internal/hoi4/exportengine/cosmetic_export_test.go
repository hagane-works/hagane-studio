package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func cosmeticFixture() Snapshot {
	return Snapshot{ProjectID: 42,
		Project: json.RawMessage(`{"id":42,"name":"Cosmetic fixture","country_tag":"ROM","export_languages":["en","zh"]}`),
		Tables: map[string][]json.RawMessage{
			"history_files": {json.RawMessage(`{"id":1,"country_tag":"ROM","set_cosmetic_tag":"ROM_greater_kingdom","source":"created"}`)},
		}}
}

func TestCosmeticImportedNamesSurviveNativeArtifact(t *testing.T) {
	snapshot := cosmeticFixture()
	snapshot.Tables["project_loc_entries"] = []json.RawMessage{
		json.RawMessage(`{"id":1,"locale":"l_english","key":"ROM_greater_kingdom","value":"Creator's Kingdom","version":0,"source_file":"localisation/english/creator_l_english.yml","source":"imported"}`),
		json.RawMessage(`{"id":2,"locale":"l_simp_chinese","key":"ROM_greater_kingdom_ADJ","value":"大王国","version":0,"source_file":"localisation/simp_chinese/creator_l_simp_chinese.yml","source":"imported"}`),
		json.RawMessage(`{"id":3,"locale":"l_english","key":"ROM_greater_kingdom_DEF","value":"","source_file":"localisation/english/creator_l_english.yml","source":"imported"}`),
	}
	artifact, err := BuildNativeShadow(snapshot, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	text := ""
	english := ""
	for _, entry := range artifact.Entries {
		if strings.HasSuffix(entry.Path, ".yml") {
			text += string(entry.Bytes)
			if strings.Contains(string(entry.Bytes), "l_english:") {
				english += string(entry.Bytes)
			}
		}
	}
	for _, want := range []string{`ROM_greater_kingdom:0 "Creator's Kingdom"`, `ROM_greater_kingdom_ADJ:0 "大王国"`, `ROM_greater_kingdom_DEF: ""`} {
		if !strings.Contains(text, want) {
			t.Errorf("creator value missing: %s", want)
		}
	}
	if strings.Contains(english, `ROM_greater_kingdom:0 "ROM Greater Kingdom"`) {
		t.Error("machine guess still competes with preserved creator base name")
	}
}

func TestCosmeticRejectedPreservedRowsDoNotSuppressFallback(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		include    bool
	}{
		{"excluded_import", "localisation/english/original_l_english.yml", false},
		{"invalid_path", "../../escape.yml", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := cosmeticFixture()
			row, _ := json.Marshal(map[string]any{"id": 1, "locale": "l_english", "key": "ROM_greater_kingdom", "value": "Original", "source_file": tc.path, "source": "imported"})
			snapshot.Tables["project_loc_entries"] = []json.RawMessage{row}
			artifact, err := BuildNativeShadow(snapshot, nil, tc.include)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, entry := range artifact.Entries {
				if strings.Contains(string(entry.Bytes), `ROM_greater_kingdom:0 "ROM Greater Kingdom"`) {
					found = true
				}
			}
			if !found {
				t.Fatal("row that cannot replay suppressed fallback")
			}
		})
	}
}
