package trees

import (
	"reflect"
	"strings"
	"testing"
)

const initialPositionRepairWarning = "focus.initial_position.invalid_shape: imported opening-camera data requires repair"

func importInitialPosition(t *testing.T, body string) (map[string]any, string, []string) {
	t.Helper()
	source := "focus_tree = { id = test initial_show_position = { " + body + " } }"
	bundles, summary, err := ImportFocusFile(source, "common/national_focus/test.txt")
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if len(bundles) != 1 {
		t.Fatalf("bundles=%d", len(bundles))
	}
	position, ok := bundles[0].TreeData["initial_show_position"].(map[string]any)
	if !ok {
		t.Fatalf("position type=%T", bundles[0].TreeData["initial_show_position"])
	}
	legacy, _ := bundles[0].TreeData["initial_show_focus"].(string)
	return position, legacy, summary.Warnings
}

func TestInitialShowPositionValidShapes(t *testing.T) {
	focus, legacy, warnings := importInitialPosition(t, "focus = GER_opening")
	if !reflect.DeepEqual(focus, map[string]any{
		"kind": "focus", "focus_id": "GER_opening",
	}) {
		t.Fatalf("focus=%#v", focus)
	}
	if legacy != "GER_opening" || len(warnings) != 0 {
		t.Fatalf("legacy=%q warnings=%#v", legacy, warnings)
	}

	coordinates, legacy, warnings := importInitialPosition(t, `x = 12 y = -3
		offset = {
			x = 4 y = 5
			trigger = { OR = { tag = GER tag = FRA } }
		}`)
	if coordinates["kind"] != "coordinates" || legacy != "" || len(warnings) != 0 {
		t.Fatalf("coordinates=%#v legacy=%q warnings=%#v", coordinates, legacy, warnings)
	}
	offsets, ok := coordinates["offsets"].([]map[string]any)
	if !ok || len(offsets) != 1 {
		t.Fatalf("offsets=%#v", coordinates["offsets"])
	}
	trigger, _ := offsets[0]["trigger"].(string)
	if !strings.Contains(trigger, "OR = {") || !strings.Contains(trigger, "tag = GER") {
		t.Fatalf("trigger=%q", trigger)
	}
}

func TestInitialShowPositionInvalidShapesArePreserved(t *testing.T) {
	cases := map[string]string{
		"mixed":             "focus = GER_opening x = 1 y = 2",
		"duplicate focus":   "focus = GER_opening focus = GER_other",
		"duplicate axis":    "x = 1 x = 2 y = 3",
		"unknown":           "x = 1 y = 2 mystery = yes",
		"bare":              "x = 1 y = 2 orphan",
		"scalar offset":     "x = 1 y = 2 offset = nope",
		"offset bare":       "x = 1 y = 2 offset = { x = 0 y = 0 trigger = { tag = GER } orphan }",
		"offset duplicate":  "x = 1 y = 2 offset = { x = 0 x = 1 y = 0 trigger = { tag = GER } }",
		"trigger duplicate": "x = 1 y = 2 offset = { x = 0 y = 0 trigger = { tag = GER } trigger = { tag = FRA } }",
		"missing trigger":   "x = 1 y = 2 offset = { x = 0 y = 0 }",
		"empty trigger":     "x = 1 y = 2 offset = { x = 0 y = 0 trigger = { } }",
		"float axis":        "x = 1.0 y = 2",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			position, legacy, warnings := importInitialPosition(t, body)
			if position["kind"] != "invalid" {
				t.Fatalf("position=%#v", position)
			}
			raw, rawOK := position["raw"].(string)
			rawText, rawTextOK := position["raw_text"].(string)
			if !rawOK || !rawTextOK || raw == "" || raw != rawText {
				t.Fatalf("raw=%#v raw_text=%#v", position["raw"], position["raw_text"])
			}
			if legacy != "" || !reflect.DeepEqual(warnings, []string{initialPositionRepairWarning}) {
				t.Fatalf("legacy=%q warnings=%#v", legacy, warnings)
			}
		})
	}
}

func TestInitialShowPositionJSONPreservesRepairPayload(t *testing.T) {
	payload := map[string]any{
		"kind": "invalid", "raw": "x = 1\n", "raw_text": "x = 1\n",
	}
	if got := initialShowPositionJSON(payload); !reflect.DeepEqual(got, payload) {
		t.Fatalf("got=%#v want=%#v", got, payload)
	}
}
