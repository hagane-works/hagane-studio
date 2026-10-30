package military

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"haganeworks/internal/apiutil"
	unitsparser "haganeworks/internal/hoi4/importsvc/units"
)

func TestValidateDeploymentIndexedMatchesWarningContract(t *testing.T) {
	location := 42
	index := deploymentValidationIndex{
		ownedByTag:     map[string]map[int]bool{"GER": {1: true}},
		templatesByOOB: map[string]map[string]bool{"GER_1936": {"Infantry": true}},
		allTemplates:   map[string]bool{"Infantry": true, "Fallback": true},
	}
	cases := []struct {
		name string
		row  DivisionDeployment
		want []string
	}{
		{"unowned and missing template preserve warning order",
			DivisionDeployment{OobFile: "GER_1936", CountryTag: "ger", Location: &location, DivisionTemplate: "Unknown"},
			[]string{"actionable:unowned_province|province=42|tag=GER", "info:missing_template|template=Unknown"}},
		{"global template fallback",
			DivisionDeployment{OobFile: "GER_1936", CountryTag: "GER", Location: &location, DivisionTemplate: "Fallback"},
			[]string{"actionable:unowned_province|province=42|tag=GER"}},
		{"missing location and template",
			DivisionDeployment{OobFile: "GER_1936"},
			[]string{"actionable:no_location", "info:no_template"}},
	}
	for _, tc := range cases {
		if got := validateDeploymentIndexed(tc.row, index); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: warnings=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDeploymentUnknownTagCriteria(t *testing.T) {
	loc, other := 7274, 999
	realProject := deploymentValidationIndex{
		ownedByTag:     map[string]map[int]bool{"GUN": {7274: true, 1: true}},
		knownTags:      map[string]bool{"GUN": true, "ROM": true},
		templatesByOOB: map[string]map[string]bool{}, allTemplates: map[string]bool{},
	}
	emptyStates := deploymentValidationIndex{
		ownedByTag: map[string]map[int]bool{}, knownTags: map[string]bool{"GUN": true},
		templatesByOOB: map[string]map[string]bool{}, allTemplates: map[string]bool{},
	}
	missing := "info:missing_template|template=x"
	cases := []struct {
		name  string
		index deploymentValidationIndex
		row   DivisionDeployment
		want  []string
	}{
		{"empty states project stays silent", emptyStates,
			DivisionDeployment{OobFile: "new_1978", Location: &loc, DivisionTemplate: "x"}, []string{missing}},
		{"empty states project silent even for a real tag", emptyStates,
			DivisionDeployment{OobFile: "GUN_1936", CountryTag: "GUN", Location: &other, DivisionTemplate: "x"}, []string{missing}},
		{"unresolvable tag reports question mark", realProject,
			DivisionDeployment{OobFile: "", CountryTag: "  ", Location: &loc, DivisionTemplate: "x"},
			[]string{"actionable:unknown_tag|tag=?", missing}},
		{"platform default oob_file yields unknown_tag NEW", realProject,
			DivisionDeployment{OobFile: "new_1978", Location: &loc, DivisionTemplate: "x"},
			[]string{"actionable:unknown_tag|tag=NEW", missing}},
		{"known tag with foreign province keeps unowned_province", realProject,
			DivisionDeployment{OobFile: "GUN_1936", CountryTag: "gun", Location: &other, DivisionTemplate: "x"},
			[]string{"actionable:unowned_province|province=999|tag=GUN", missing}},
		{"known but stateless country stays silent", realProject,
			DivisionDeployment{OobFile: "ROM_1936", CountryTag: "ROM", Location: &loc, DivisionTemplate: "x"}, []string{missing}},
		{"owned province stays silent", realProject,
			DivisionDeployment{OobFile: "GUN_1936", CountryTag: "GUN", Location: &loc, DivisionTemplate: "x"}, []string{missing}},
		{"no_location wins over tag checks", realProject,
			DivisionDeployment{OobFile: "new_1978", DivisionTemplate: "x"}, []string{"actionable:no_location", missing}},
	}
	for _, tc := range cases {
		if got := validateDeploymentIndexed(tc.row, tc.index); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: warnings=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestOOBOwnershipWarningWireFormat(t *testing.T) {
	if warnNoLocation != "actionable:no_location" {
		t.Fatalf("no_location wire format=%q", warnNoLocation)
	}
	for _, scope := range []oobOwnershipScope{provinceOwnership, stateOwnership} {
		if got := oobOwnershipWarning(scope, "NEW", 12, nil, false, true); got != "actionable:unknown_tag|tag=NEW" {
			t.Fatalf("unknown_tag wire format=%q", got)
		}
		if got := oobOwnershipWarning(scope, "", 12, nil, false, true); got != "actionable:unknown_tag|tag=?" {
			t.Fatalf("empty-tag wire format=%q", got)
		}
		if got := oobOwnershipWarning(scope, "NEW", 12, nil, false, false); got != "" {
			t.Fatalf("empty-states must stay silent, got %q", got)
		}
	}
	if got := oobOwnershipWarning(stateOwnership, "GUN", 90, map[int]bool{12: true}, true, true); got != "actionable:unowned_state|state=90|tag=GUN" {
		t.Fatalf("unowned_state wire format=%q", got)
	}
	if got := deploymentLocationWarning("GUN", 999, map[int]bool{7274: true}, true, true); got != "actionable:unowned_province|province=999|tag=GUN" {
		t.Fatalf("unowned_province wire format=%q", got)
	}
}

func TestMilitaryImportPolicyKeepsLegacyPythonSemantics(t *testing.T) {
	for _, tc := range []struct {
		policy    string
		overwrite bool
	}{
		{"", true}, {"overwrite", true}, {"OVERWRITE", true},
		{"skip", false}, {"SKIP", false},

		{" skip ", true}, {"merge", true},
	} {
		if got := militaryOverwrite(tc.policy); got != tc.overwrite {
			t.Fatalf("militaryOverwrite(%q)=%v want %v", tc.policy, got, tc.overwrite)
		}
	}
}

func TestMilitaryImportDuplicateCountersAndReturnedIDsMatchSequentialPython(t *testing.T) {
	keys := []string{"old", "new", "new", "other"}
	plan := planImport(keys, map[string]bool{"old": true}, true)
	if plan.Created != 2 || plan.Updated != 2 || plan.Skipped != 0 {
		t.Fatalf("plan = %#v", plan)
	}
	if got, want := successfulImportKeys(keys, plan.Actions), []string{"old", "new", "new", "other"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("returned IDs=%#v want %#v", got, want)
	}
	if got, want := selectedImportKeys(keys, plan.Selected), []string{"old", "new", "other"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("persisted IDs=%#v want %#v", got, want)
	}
	plan = planImport(keys, map[string]bool{"old": true}, false)
	if got, want := successfulImportKeys(keys, plan.Actions), []string{"new", "other"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("skip returned IDs=%#v want %#v", got, want)
	}
}

func TestNormalizeImportSourceFilePythonParity(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		bad         bool
	}{
		{"", "", false},
		{` /common\units\demo.TXT/ `, "common/units/demo.TXT", false},
		{"common/units/deep/demo.txt", "", true},
		{"common/units/../demo.txt", "", true},
		{"C:/common/units/demo.txt", "", true},
	} {
		got, err := normalizeImportSourceFile(tc.input)
		if (err != nil) != tc.bad || got != tc.want {
			t.Fatalf("normalize(%q)=(%q,%v), want (%q,bad=%v)", tc.input, got, err, tc.want, tc.bad)
		}
	}
}

const oobTransferFixture = `air_wings = {
	219 = {
		fighter_equipment_1 = { mission = escort owner = GER amount = 24 }
		name = "First Wing"
	}
}
units = {
	fleet = {
		name = "Home Fleet"
		naval_base = 123
		task_force = {
			name = "First Fleet"
			location = 123
			ship = {
				name = "KMS Test"
				definition = destroyer
				start_experience_factor = 0.25
			}
		}
	}
}`

func TestOOBImportTopNormalisationMatchesPython(t *testing.T) {
	source, oob, tag := normaliseOOBImportTop(oobImportInput{OobFile: "  ger_1936  ", CountryTag: " gerlongcountry "})
	if source != "history/units/ger_1936.txt" || oob != "ger_1936" || tag != "GERLONGCOU" {
		t.Fatalf("normalised = %q, %q, %q", source, oob, tag)
	}
	source, oob, tag = normaliseOOBImportTop(oobImportInput{SourceFile: `\history\units\USA_1936.txt`})
	if source != "history/units/USA_1936.txt" || oob != "USA_1936" || tag != "USA" {
		t.Fatalf("derived = %q, %q, %q", source, oob, tag)
	}
	source, oob, tag = normaliseOOBImportTop(oobImportInput{OobFile: "   ", CountryTag: "   "})
	if source != "history/units/oob.txt" || oob != "" || tag != "" {
		t.Fatalf("explicit whitespace = %q, %q, %q", source, oob, tag)
	}
	if got := purePosixStem("history/units/foo."); got != "foo" {
		t.Fatalf("trailing-dot stem = %q", got)
	}
	if got := purePosixStem("history/units/..."); got != "..." {
		t.Fatalf("all-dot stem = %q", got)
	}
}

func TestOOBImportResponseCountsEntireHierarchy(t *testing.T) {
	wire := unitsparser.AirNavalOOBJSON(oobTransferFixture)
	result := countOOBWire(oobWireSlice(wire["groups"]), oobWireSlice(wire["fleets"]))
	if result.AirWingGroups != 1 || result.AirWings != 1 || result.Fleets != 1 ||
		result.TaskForces != 1 || result.Ships != 1 || result.Skipped != 0 || len(result.Warnings) != 0 {
		t.Fatalf("counts = %#v", result)
	}
}

func TestOOBImportRejectsRecoveredUnclosedBlockLikePython(t *testing.T) {
	wire := unitsparser.AirNavalOOBJSON("units = {\n fleet = {\n name = \"Unterminated\"\n")
	if len(oobWireSlice(wire["groups"])) != 0 || len(oobWireSlice(wire["fleets"])) != 0 {
		t.Fatalf("partial hierarchy escaped strict import: %#v", wire)
	}
	warnings := oobWireWarnings(wire["warnings"])
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Expected RBRACE, got EOF") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestOOBPreviewMatchesFrozenPythonReference(t *testing.T) {
	snapshot := OOBSnapshot{
		AirWingGroups: []AirWingGroup{{ID: 1, ProjectID: 7, Location: "219", Layout: []map[string]any{}}},
		AirWings: []AirWing{{
			ID: 2, ProjectID: 7, GroupID: 1, EquipmentType: "fighter_equipment_1", Name: "First Wing",
			Owner: "GER", Amount: "24", BodyAst: []map[string]any{{
				"key": "mission", "op": "=", "value": map[string]any{"t": "str", "v": "escort"},
				"line": float64(1), "quoted": false, "qualifier": nil,
			}},
		}},
		Fleets:     []Fleet{{ID: 3, ProjectID: 7, Name: "Home Fleet", NavalBase: "123", Layout: []map[string]any{}}},
		TaskForces: []TaskForce{{ID: 4, ProjectID: 7, FleetID: 3, Name: "First Fleet", Location: "123", Layout: []map[string]any{}}},
		Ships:      []Ship{{ID: 5, ProjectID: 7, TaskForceID: 4, Name: "KMS Test", Definition: "destroyer", StartExperienceFactor: "0.25", BodyAst: []map[string]any{}}},
	}
	got, err := renderOOBPreview(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	want := "air_wings = {\n\t219 = {\n\t\tfighter_equipment_1 = {\n\t\t\tmission = escort\n\t\t\towner = GER\n\t\t\tamount = 24\n\t\t}\n\t\tname = \"First Wing\"\n\t}\n}\n\nunits = {\n\tfleet = {\n\t\tname = \"Home Fleet\"\n\t\tnaval_base = 123\n\t\ttask_force = {\n\t\t\tname = \"First Fleet\"\n\t\t\tlocation = 123\n\t\t\tship = {\n\t\t\t\tname = \"KMS Test\"\n\t\t\t\tdefinition = destroyer\n\t\t\t\tstart_experience_factor = 0.25\n\t\t\t}\n\t\t}\n\t}\n}\n"
	if got != want {
		t.Fatalf("preview drifted from Python reference\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestOOBPreviewPreservesPython422Context(t *testing.T) {
	_, err := renderOOBPreview(OOBSnapshot{AirWingGroups: []AirWingGroup{{ID: 91, Location: "two tokens"}}})
	var status *apiutil.StatusError
	if !errors.As(err, &status) || status.Status != 422 ||
		status.Body.(map[string]any)["detail"] != "Air wing group #91 requires a valid spawn state/location token" {
		t.Fatalf("error = %#v", err)
	}
}
