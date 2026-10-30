package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildNativeShadowWiresTraitOverlayArtifacts(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 92,
		Project:   json.RawMessage(`{"id":92,"name":"Demo","country_tag":"GER","export_languages":["en","zh"]}`),
		Tables: map[string][]json.RawMessage{
			"characters": {
				json.RawMessage(`{"id":71,"tag":"GER_overlay_leader","name":"Overlay Leader","name_en":"Overlay Leader","name_zh":"覆盖领袖","country_tag":"GER","role_type":"country_leader","ideology":"fascism","traits":["custom_overlay_trait"],"raw_extras":{},"source":"created","replaces_vanilla_id":true,"is_portrait_modified":true}`),
			},
			"custom_traits": {
				json.RawMessage(`{"id":81,"trait_id":"custom_overlay_trait","trait_type":"country_leader","country_tag":"GER","source":"created"}`),
			},
			"entity_obs_paths": {
				json.RawMessage(`{"id":91,"entity_type":"character_portrait","entity_id":71,"obs_key":"fixture/overlay.png"}`),
			},
		},
	}
	pngBytes := tolerancePNG(t)
	artifact, err := BuildNativeShadowWithBlobs(snapshot, nil, false, func(key string, _ int64) ([]byte, error) {
		if key == "fixture/overlay.png" {
			return pngBytes, nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, entry := range artifact.Entries {
		contents[entry.Path] = string(entry.Bytes)
	}
	wants := []string{
		"Demo/events/GER_trait_overlay.txt",
		"Demo/interface/zz_GER_replacement_portraits.gfx",
		"Demo/localisation/english/replace/GER_characters_replacement_overrides_l_english.yml",
		"Demo/localisation/simp_chinese/replace/GER_characters_replacement_overrides_l_simp_chinese.yml",
	}
	for _, path := range wants {
		if _, ok := contents[path]; !ok {
			t.Errorf("central seam did not ship %q", path)
		}
	}
	onActions := contents["Demo/common/on_actions/GER_on_actions.txt"]
	if !strings.Contains(onActions, "country_event = ger_trait_overlay.1") {
		t.Errorf("central seam did not bind the emitted overlay event:\n%s", onActions)
	}
}
