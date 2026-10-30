package content

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/project"
)

func TestWargoalAdvancedParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "wargoal_advanced_parity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Parse []struct {
			Input   string          `json:"input"`
			Records []wargoalRecord `json:"records"`
			Error   *string         `json:"error"`
		} `json:"parse"`
		Replace []struct {
			Existing []wargoalRecord `json:"existing"`
			Advanced []wargoalRecord `json:"advanced"`
			Result   []wargoalRecord `json:"result"`
		} `json:"replace"`
		FromRecords []struct {
			Records []wargoalRecord `json:"records"`
			Script  string          `json:"script"`
		} `json:"from_records"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Parse) == 0 || len(fx.Replace) == 0 || len(fx.FromRecords) == 0 {
		t.Fatal("empty fixture sections")
	}
	for _, c := range fx.Parse {
		records, err := parseWargoalAdvancedEntries(c.Input)
		if c.Error != nil {
			if err == nil || err.Error() != *c.Error {
				t.Errorf("parse(%q): err = %v, python = %q", c.Input, err, *c.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("parse(%q): unexpected error %v", c.Input, err)
			continue
		}
		if !reflect.DeepEqual(records, c.Records) {
			t.Errorf("parse(%q) drift:\n got: %#v\nwant: %#v", c.Input, records, c.Records)
		}
	}
	for i, c := range fx.Replace {
		want := c.Result
		if want == nil {
			want = []wargoalRecord{}
		}
		if got := wargoalReplaceAdvancedRecords(c.Existing, c.Advanced); !reflect.DeepEqual(got, want) {
			t.Errorf("replace[%d] drift:\n got: %#v\nwant: %#v", i, got, want)
		}
	}
	for i, c := range fx.FromRecords {
		if got := wargoalAdvancedScriptFromRecords(c.Records); got != c.Script {
			t.Errorf("from_records[%d] = %q, python %q", i, got, c.Script)
		}
	}
}

func TestWargoalValidators(t *testing.T) {
	if _, err := wargoalValidateToken("  annex_everything ", "wargoal_key"); err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := wargoalValidateToken("two words", "wargoal_key"); err == nil || err.Error() != "wargoal_key must be one HOI4 token" {
		t.Fatalf("token space: %v", err)
	}
	if _, err := wargoalValidateToken("", "wargoal_key"); err == nil || err.Error() != "wargoal_key is required" {
		t.Fatalf("token empty: %v", err)
	}
	if v, err := wargoalValidateScalar(` "War of X" `, "war_name"); err != nil || v != `"War of X"` {
		t.Fatalf("scalar quoted: %v %q", err, v)
	}
	if _, err := wargoalValidateScalar("no quote spaces", "war_name"); err == nil {
		t.Fatal("scalar unquoted spaces must fail")
	}
	if _, err := wargoalValidateBlock("oops = {", "allowed"); err == nil || err.Error() != "allowed contains malformed HOI4 script" {
		t.Fatalf("block malformed: %v", err)
	}
	if _, err := wargoalValidateSafeScalars(map[string]string{"expire": "5", "bogus": "1"}); err == nil ||
		err.Error() != "unsupported safe scalar keys: bogus" {
		t.Fatalf("safe scalars unknown: %v", err)
	}
	if v, err := wargoalValidateSafeScalars(map[string]string{"expire": " 5 ", "threat": "  "}); err != nil || v["expire"] != "5" || len(v) != 1 {
		t.Fatalf("safe scalars clean: %v %v", err, v)
	}
}

func TestAIStrategyPlanCreateDefaultsAndValidation(t *testing.T) {
	var input aiStrategyPlanCreate
	if err := json.Unmarshal([]byte(`{"plan_id":"  plan_one  "}`), &input); err != nil {
		t.Fatal(err)
	}
	if err := input.normalize(); err != nil {
		t.Fatal(err)
	}
	if input.PlanID != "plan_one" || len(input.Allowed) != 1 || input.Allowed[0] != "{\n}" ||
		len(input.Weight) != 1 || len(input.FieldOrder) != 6 || input.AINationalFocuses == nil {
		t.Fatalf("Python defaults drifted: %+v", input)
	}
	for _, payload := range []string{`{"plan_id":"x","name":null}`, `{"plan_id":"x","unknown":1}`} {
		if err := json.Unmarshal([]byte(payload), &input); err == nil {
			t.Fatalf("payload must fail: %s", payload)
		}
	}
	if _, err := aiValidateScriptID("has'quote", "plan_id"); err == nil {
		t.Fatal("single quote must be rejected like SAFE_ID_RE")
	}
	if _, err := aiValidateBlock("factor = 1", "weight"); err == nil || err.Error() != "weight must be one brace block" {
		t.Fatalf("unwrapped block error drifted: %v", err)
	}
	if _, err := aiValidateBlock("{ factor = { }", "weight"); err == nil || err.Error() != "weight must be one parseable brace block" {
		t.Fatalf("malformed block error drifted: %v", err)
	}
}

func TestAIStrategyPlanPatchRejectsNullAndUnknownOrder(t *testing.T) {
	var patch aiStrategyPlanPatch
	if err := json.Unmarshal([]byte(`{"field_order":["name","bogus"]}`), &patch); err != nil {
		t.Fatal(err)
	}
	if err := patch.normalize(); err == nil || err.Error() != "field_order contains an unknown plan field" {
		t.Fatalf("field_order error drifted: %v", err)
	}
}

func TestAITemplateCreateDefaultsAndValidation(t *testing.T) {
	var input aiTemplateCreate
	if err := json.Unmarshal([]byte(`{"template_id":"  generic_role  "}`), &input); err != nil {
		t.Fatal(err)
	}
	if err := input.normalize(); err != nil {
		t.Fatal(err)
	}
	if input.TemplateID != "generic_role" || input.Role != "infantry" || len(input.TargetTemplates) != 1 ||
		input.TargetTemplates[0].ID != "infantry_default" || len(input.FieldOrder) != 3 {
		t.Fatalf("Python defaults drifted: %+v", input)
	}
	for _, payload := range []string{
		`{"template_id":"x","target_templates":[]}`,
		`{"template_id":"x","target_templates":[{"id":"same","body":"{}"},{"id":"same","body":"{}"}]}`,
		`{"template_id":"x","target_templates":[{"id":"x","body":"{}","extra":1}]}`,
	} {
		var candidate aiTemplateCreate
		err := json.Unmarshal([]byte(payload), &candidate)
		if err == nil {
			err = candidate.normalize()
		}
		if err == nil {
			t.Fatalf("invalid template payload accepted: %s", payload)
		}
	}
}

func TestAINormalizeLegacyTemplateBlockReadCompatibility(t *testing.T) {
	legacy := "factor = 1\nmodifier = { factor = 2 }"
	if got, err := aiNormalizeLegacyBlock(legacy, "upgrade_prio"); err != nil || got != "{\n"+legacy+"\n}" {
		t.Fatalf("legacy block not repaired: got=%q err=%v", got, err)
	}
	if _, err := aiNormalizeLegacyBlock("{ factor = { }", "available_for"); err == nil {
		t.Fatal("damaged leading-brace block must not be wrapped again")
	}
	targets, err := aiNormalizeLegacyTargets([]aiTargetTemplate{{ID: "infantry_default", Body: ""}})
	if err != nil || targets[0].Body != "{\n}" {
		t.Fatalf("empty legacy target body was not restored as an empty block: got=%+v err=%v", targets, err)
	}
	if got, err := aiNormalizeRequiredLegacyBlock("\n", "weight"); err != nil || got != "{\n}" {
		t.Fatalf("empty legacy strategy-plan body was not restored: got=%q err=%v", got, err)
	}
}

func TestAITemplatePatchRejectsNullAndValidatesMarkers(t *testing.T) {
	var patch aiTemplatePatch
	if err := json.Unmarshal([]byte(`{"role":null}`), &patch); err != nil {
		t.Fatal(err)
	}
	if err := patch.normalize(); err == nil || err.Error() != "AI template update fields cannot be null" {
		t.Fatalf("explicit null error drifted: %v", err)
	}
	patch = aiTemplatePatch{}
	if err := json.Unmarshal([]byte(`{"field_order":["target:two words"]}`), &patch); err != nil {
		t.Fatal(err)
	}
	if err := patch.normalize(); err == nil || err.Error() != "field_order contains an unknown template field" {
		t.Fatalf("marker error drifted: %v", err)
	}
}

func aiTemplateRow(role, targets, order string) project.Row {
	var targetValue, orderValue any
	_ = json.Unmarshal([]byte(targets), &targetValue)
	_ = json.Unmarshal([]byte(order), &orderValue)
	return project.Row{
		"id": json.Number("1"), "project_id": json.Number("1"), "template_id": "generic_suppression",
		"role": role, "blocked_for": "", "available_for": "", "upgrade_prio": "", "enable": "",
		"front_role_override": "", "target_templates": targetValue, "field_order": orderValue,
		"raw_block": "", "source_file": "common/ai_templates/generic.txt", "source": "imported",
	}
}

func TestAITemplateReadsImportedBareTargetIDs(t *testing.T) {
	item, err := aiTemplateFromRow(aiTemplateRow("suppression",
		`[{"id":"generic_suppression_template","body":"{ target_template = x }"}]`,
		`["role","generic_suppression_template"]`))
	if err != nil {
		t.Fatalf("imported row with a bare target id in field_order: %v", err)
	}
	if got := strings.Join(item.FieldOrder, ","); got != "role,target:generic_suppression_template" {
		t.Fatalf("field_order = %s", got)
	}
	if _, err := aiTemplateFromRow(aiTemplateRow("suppression", `[{"id":"a","body":"{ }"}]`, `["role","not_a_target"]`)); err == nil {
		t.Fatal("a bare name that is not one of the row's targets must still be refused")
	}
	dup, err := aiTemplateFromRow(aiTemplateRow("infantry",
		`[{"id":"infantry_default","body":"{ }"},{"id":"infantry_default","body":"{ }"}]`,
		`["role","infantry_default","infantry_default"]`))
	if err != nil || len(dup.TargetTemplates) != 2 {
		t.Fatalf("imported duplicate target ids: item=%+v err=%v", dup, err)
	}
	if _, err := aiNormalizeTargets(dup.TargetTemplates); err == nil {
		t.Fatal("saving duplicate target ids must still be refused")
	}
}

func TestPanelNamesParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "panel_names_parity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Derive []struct {
			Token   string            `json:"token"`
			Control *string           `json:"control"`
			Names   map[string]string `json:"names"`
		} `json:"derive"`
		Walk []struct {
			Name     string           `json:"name"`
			Controls []map[string]any `json:"controls"`
			Order    []string         `json:"order"`
		} `json:"walk"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Derive) == 0 {
		t.Fatal("empty fixture")
	}
	for _, c := range fx.Derive {
		ctrl, with := "", false
		if c.Control != nil {
			ctrl, with = *c.Control, true
		}
		if got := derivePanelNames(c.Token, ctrl, with); !reflect.DeepEqual(got, c.Names) {
			t.Errorf("derive(%q,%v)\n got %v\nwant %v", c.Token, c.Control, got, c.Names)
		}
	}
	for _, c := range fx.Walk {
		order := []string{}
		for _, ctrl := range walkPanelControls(c.Controls) {
			order = append(order, anyToString(ctrl["name"]))
		}
		want := c.Order
		if want == nil {
			want = []string{}
		}
		if !reflect.DeepEqual(order, want) {
			t.Errorf("walk %s\n got %v\nwant %v", c.Name, order, want)
		}
	}
}

func TestWalkOnlyRecursesContainers(t *testing.T) {
	got := walkPanelControls([]map[string]any{
		{"name": "a", "kind": "text", "children": []any{map[string]any{"name": "hidden"}}},
		{"name": "b", "kind": "CONTAINER", "children": []any{map[string]any{"name": "shown"}}},
	})
	names := []string{}
	for _, c := range got {
		names = append(names, anyToString(c["name"]))
	}
	if strings.Join(names, ",") != "a,b,shown" {
		t.Errorf("got %v", names)
	}
}
