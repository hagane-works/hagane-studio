package events

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSlugifyIDMirrorsPython(t *testing.T) {
	cases := map[string]string{
		"france.100":  "france100",
		"My Event-ID": "my_event_id",
		"  GER.2  ":   "ger2",
		"":            "unnamed",
		"___":         "unnamed",
	}
	for in, want := range cases {
		if got := slugifyID(in); got != want {
			t.Fatalf("slugifyID(%q)=%q want %q", in, got, want)
		}
	}
	if got := eventPictureSpriteName("ger", "france.100"); got != "GFX_report_event_GER_france100" {
		t.Fatalf("sprite=%q", got)
	}
}

func TestNormalizeEventOptionsMirrorsPython(t *testing.T) {
	opts := []any{
		map[string]any{"name": "france.1.a", "display_text": "Accept", "name_zh": "接受"},
		"not-a-dict",
	}
	out := normalizeEventOptionsWithNameBackfill(opts)
	first := out[0].(map[string]any)
	locs := first["display_name_locs"].(map[string]any)
	if locs["en"] != "Accept" || locs["zh"] != "接受" {
		t.Fatalf("locs=%v", locs)
	}
	if first["name_en"] != "Accept" {
		t.Fatalf("name_en=%v", first["name_en"])
	}
	if out[1] != "not-a-dict" {
		t.Fatalf("non-dict must pass through")
	}
	out2 := normalizeEventOptionsWithNameBackfill([]any{map[string]any{"name": "Take it"}})
	if out2[0].(map[string]any)["name_en"] != "Take it" {
		t.Fatalf("bare name backfill broken: %v", out2[0])
	}
	out3 := normalizeEventOptionsWithNameBackfill([]any{map[string]any{"name": "germany.107.a"}})
	if got := out3[0].(map[string]any); got["name_en"] != nil || got["display_name_locs"] != nil {
		t.Fatalf("a key was taken for text: %v", got)
	}
}

func TestEventIDPattern(t *testing.T) {
	if m := eventIDRE.FindStringSubmatch("france.100"); m == nil || m[1] != "france" || m[2] != "100" {
		t.Fatalf("m=%v", m)
	}
	for _, bad := range []string{"france", "france.", ".100", "1france.2", "france.1.a"} {
		if eventIDRE.MatchString(bad) {
			t.Fatalf("%q should not match", bad)
		}
	}
}

func TestEventCreateInputNormalize(t *testing.T) {
	var in eventCreateInput
	if err := in.normalize(); err == nil {
		t.Fatal("missing namespace_id must fail")
	}
	ns, num := int64(4), 2
	eid := "france.2"
	in = eventCreateInput{NamespaceID: &ns, EventID: &eid, EventNumber: &num}
	if err := in.normalize(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in.Options, []any{}) || in.Title != "" {
		t.Fatalf("defaults: %+v", in)
	}
}

func TestEventRenameResolvesAutomaticPicture(t *testing.T) {
	newID := "france.101"
	explicitOldSprite := "GFX_report_event_GER_france100"
	patch := eventPatch{
		EventID: optional[string]{Set: true, Value: &newID},
		Picture: optional[string]{Set: true, Value: &explicitOldSprite},
	}
	cur := eventResponse{EventID: "france.100", CountryTag: "GER", Picture: explicitOldSprite}
	got, ok := eventRenamedAutoPicture(cur, patch, "GER")
	if !ok || got != "GFX_report_event_GER_france101" {
		t.Fatalf("automatic picture was not resolved: got=%q ok=%v", got, ok)
	}
	custom := "GFX_custom_event_picture"
	patch.Picture.Value = &custom
	if got, ok := eventRenamedAutoPicture(cur, patch, "GER"); ok {
		t.Fatalf("custom picture must not be overwritten: got=%q", got)
	}
}

func TestResolveCollisionFreeEventID(t *testing.T) {
	taken := map[string]bool{"france.1": true, "france.2": true, "custom": true, "custom_1": true}
	number := 1
	id, n, warning, err := resolveCollisionFreeEventID(taken, "france.1", &number)
	if err != nil || id != "france.3" || n == nil || *n != 3 || warning == nil {
		t.Fatalf("namespaced: %q %v %v %v", id, n, warning, err)
	}
	id, n, warning, err = resolveCollisionFreeEventID(taken, "custom", &number)
	if err != nil || id != "custom_2" || n != &number || warning == nil {
		t.Fatalf("plain: %q %v %v %v", id, n, warning, err)
	}
	id, _, warning, err = resolveCollisionFreeEventID(taken, "france.9", &number)
	if err != nil || id != "france.9" || warning != nil {
		t.Fatalf("free: %q %v %v", id, warning, err)
	}
}

func TestBlockTextPolicyNormalizesUnquotedLiteralNewline(t *testing.T) {
	got, issues, err := canonicalBlockText("add_stability = 0.1\\nset_country_flag = ready")
	if err != nil {
		t.Fatal(err)
	}
	if got != "add_stability = 0.1\nset_country_flag = ready" {
		t.Fatalf("unexpected canonical block: %q", got)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "后续内容不生效") {
		t.Fatalf("missing creator-facing issue: %#v", issues)
	}
}

func TestBlockTextPolicyPreservesComments(t *testing.T) {
	src := "add_stability = 0.1 # 文档示例 \\n 不应改写\nset_country_flag = ready"
	result := applyBlockTextPolicy(src)
	if result.Value != src || result.Normalized != 0 || result.Rejected || len(result.Issues) != 0 {
		t.Fatalf("comment text changed: %#v", result)
	}
}

func TestBlockTextPolicyRejectsQuotedAndEscapedAmbiguity(t *testing.T) {
	for _, src := range []string{`custom_effect_tooltip = "a\nb"`, `path = foo\\\nbar`} {
		got, issues, err := canonicalBlockText(src)
		if !errors.Is(err, errAmbiguousBlockText) {
			t.Fatalf("%q: expected reject, got %v", src, err)
		}
		if got != src || len(issues) != 1 {
			t.Fatalf("%q: rejected input must remain unchanged: %q %#v", src, got, issues)
		}
	}
}

func TestCanonicalBlockTextFieldsNamesRepairLocation(t *testing.T) {
	fields, issues, err := canonicalBlockTextFields(map[string]string{
		"trigger":   "tag = GER",
		"immediate": "set_country_flag = a\\nadd_stability = 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fields["immediate"] != "set_country_flag = a\nadd_stability = 1" {
		t.Fatalf("field was not normalized: %#v", fields)
	}
	if len(issues) != 1 || !strings.HasPrefix(issues[0].Repair, "immediate:") {
		t.Fatalf("repair does not name the field: %#v", issues)
	}
}

func TestCanonicalBlockTextPayloadHandlesNestedEventOptionsOnly(t *testing.T) {
	payload := map[string]any{
		"immediate":        "a = yes\\nb = yes",
		"options":          []any{map[string]any{"effects": "c = yes\\nd = yes", "name": "x\\ny"}},
		"description_locs": map[string]any{"en": "first\\nsecond"},
	}
	canonical, issues, err := canonicalBlockTextPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	got := canonical.(map[string]any)
	if got["immediate"] != "a = yes\nb = yes" || len(issues) != 2 {
		t.Fatalf("top-level block was not normalized: %#v %#v", got, issues)
	}
	option := got["options"].([]any)[0].(map[string]any)
	if option["effects"] != "c = yes\nd = yes" || option["name"] != "x\\ny" {
		t.Fatalf("nested policy crossed into scalar text: %#v", option)
	}
	if got["description_locs"].(map[string]any)["en"] != "first\\nsecond" {
		t.Fatalf("localisation was rewritten: %#v", got)
	}
}
