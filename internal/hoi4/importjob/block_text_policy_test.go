package importjob

import (
	"strings"
	"testing"
)

func TestNormalizeImportedBlockPayloadRecursesIntoEventOptions(t *testing.T) {
	payload := map[string]any{
		"event_id":  "sov_prc.1",
		"immediate": "set_country_flag = a\\nadd_stability = 1",
		"options": []any{map[string]any{
			"name":    "sov_prc.1.a",
			"effects": "add_political_power = 5\\nset_country_flag = b",
		}},
	}
	result, err := NormalizeImportedBlockPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if result.Normalized != 2 {
		t.Fatalf("want 2 normalized boundaries, got %#v", result)
	}
	got := result.Value.(map[string]any)
	if !strings.Contains(got["immediate"].(string), "\nadd_stability") {
		t.Fatalf("immediate was not normalized: %#v", got)
	}
	option := got["options"].([]any)[0].(map[string]any)
	if !strings.Contains(option["effects"].(string), "\nset_country_flag") {
		t.Fatalf("option effects were not normalized: %#v", option)
	}
}

func TestNormalizeImportedBlockPayloadDoesNotTouchLocalisationOrScalar(t *testing.T) {
	payload := map[string]any{
		"description_locs": map[string]any{"en": "first\\nsecond"},
		"name":             "literal\\nidentifier",
		"trigger":          "tag = GER\ncountry_exists = GER",
	}
	result, err := NormalizeImportedBlockPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	got := result.Value.(map[string]any)
	if got["name"] != payload["name"] || got["description_locs"].(map[string]any)["en"] != "first\\nsecond" {
		t.Fatalf("non-block values changed: %#v", got)
	}
	if result.Normalized != 0 {
		t.Fatalf("real line breaks need no repair: %#v", result)
	}
}

func TestNormalizeImportedBlockPayloadRejectsAmbiguousQuotedText(t *testing.T) {
	payload := map[string]any{"immediate": `custom_effect_tooltip = "first\nsecond"`}
	result, err := NormalizeImportedBlockPayload(payload)
	if err == nil {
		t.Fatalf("ambiguous quoted text must reject: %#v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Path != "immediate" ||
		!strings.Contains(result.Issues[0].Message, "后续内容不生效") {
		t.Fatalf("missing actionable import issue: %#v", result.Issues)
	}
}
