package techlifecycle

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func testState() State {
	return State{
		Projects: []int64{1, 2},
		Branches: []Branch{
			{
				ProjectID: 1, ID: "branch_b", ProfileID: "profile", BaselineSHA256: "abc", SchemaVersion: 1,
				Surface: "army", Folder: "infantry_folder", Slot: "slot_b",
				RootTechnologyID: "TST_b", TechnologyIDs: []string{"TST_b"},
			},
			{
				ProjectID: 1, ID: "branch_a", ProfileID: "profile", BaselineSHA256: "abc", SchemaVersion: 1,
				Surface: "army", Folder: "infantry_folder", Slot: "slot_a", GridboxID: "TST_a_tree",
				RootTechnologyID: "TST_a", TechnologyIDs: []string{"TST_a"},
			},
		},
		Technologies: []Technology{
			{
				DatabaseID: 11, ProjectID: 1, TechID: "TST_a", Source: "created",
				Name: "Alpha", NameEn: "Alpha", NameZh: "阿尔法", DescEn: "Original description",
				DisplayNameLocs:  map[string]string{"zh": "阿尔法", "en": "Alpha"},
				DescriptionLocs:  map[string]string{"en": "Original description"},
				EnableEquipments: []string{}, TechLine: "infantry", Folder: "infantry_folder",
				StartYear: 1936, PositionX: 0, PositionY: 2,
				FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 2}},
				Path:             []PathEdge{},
			},
			{
				DatabaseID: 12, ProjectID: 1, TechID: "TST_b", Source: "created",
				Name: "Bravo", NameEn: "Bravo", DescEn: "Bravo description",
				TechLine: "infantry", Folder: "infantry_folder", StartYear: 1936, PositionY: 2,
				FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 2}},
			},
		},
		Equipments: []Equipment{
			{ProjectID: 1, EquipmentID: "TST_rifle"},
			{ProjectID: 2, EquipmentID: "FOREIGN_rifle"},
		},
	}
}

func contractCode(t *testing.T, err error) string {
	t.Helper()
	var target ContractError
	if !errors.As(err, &target) {
		t.Fatalf("expected ContractError, got %T: %v", err, err)
	}
	return target.Code
}

func TestListGetAndPartialUpdate(t *testing.T) {
	state := testState()
	items, err := List(state, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "branch_a" || items[1].ID != "branch_b" {
		t.Fatalf("list is not stable by branch id: %#v", items)
	}
	original, err := Get(state, 1, "branch_a")
	if err != nil {
		t.Fatal(err)
	}
	var request UpdateRequest
	if err := json.Unmarshal([]byte(`{"name_en":"Renamed","desc_zh":"新说明","enable_equipments":["TST_rifle"]}`), &request); err != nil {
		t.Fatal(err)
	}
	plan, err := Update(state, 1, "branch_a", request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.View.NameEn != "Renamed" || plan.View.DescZh != "新说明" || !reflect.DeepEqual(plan.View.EnableEquipments, []string{"TST_rifle"}) {
		t.Fatalf("mutable fields not applied: %#v", plan.View)
	}
	if plan.View.Name != original.Name || plan.View.DescEn != original.DescEn {
		t.Fatalf("omitted fields changed: before=%#v after=%#v", original, plan.View)
	}
	if plan.View.Folder != original.Folder || plan.View.StartYear != original.StartYear ||
		plan.View.PositionX != original.PositionX || plan.View.PositionY != original.PositionY ||
		!reflect.DeepEqual(plan.View.FolderPlacements, original.FolderPlacements) || !reflect.DeepEqual(plan.View.Path, original.Path) {
		t.Fatalf("profile geometry changed during descriptive update: before=%#v after=%#v", original, plan.View)
	}
	wantChanged := []string{"desc_zh", "enable_equipments", "name_en"}
	if !reflect.DeepEqual(plan.ChangedFields, wantChanged) {
		t.Fatalf("changed fields = %#v, want %#v", plan.ChangedFields, wantChanged)
	}
}

func TestExplicitClearSurvivesSaveAndReopen(t *testing.T) {
	state := testState()
	root := &state.Technologies[0]
	root.DescEn = "clear me"
	root.DescriptionLocs = map[string]string{"en": "clear me"}
	root.EnableEquipments = []string{"TST_rifle"}
	var request UpdateRequest
	if err := json.Unmarshal([]byte(`{"desc_en":null,"description_locs":null,"enable_equipments":null}`), &request); err != nil {
		t.Fatal(err)
	}
	plan, err := Update(state, 1, "branch_a", request)
	if err != nil {
		t.Fatal(err)
	}
	root.DescEn = plan.Mutable.DescEn
	root.DescriptionLocs = plan.Mutable.DescriptionLocs
	root.EnableEquipments = plan.Mutable.EnableEquipments
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var reopened State
	if err := json.Unmarshal(encoded, &reopened); err != nil {
		t.Fatal(err)
	}
	view, err := Get(reopened, 1, "branch_a")
	if err != nil {
		t.Fatal(err)
	}
	if view.DescEn != "" || len(view.DescriptionLocs) != 0 || len(view.EnableEquipments) != 0 {
		t.Fatalf("explicit clear did not survive reopen: %#v", view)
	}
	patchJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if string(patchJSON) != `{"desc_en":null,"description_locs":null,"enable_equipments":null}` {
		t.Fatalf("clear patch lost absent/null semantics: %s", patchJSON)
	}
}

func TestCrossProjectIdentityAndEquipmentOwnership(t *testing.T) {
	state := testState()
	if _, err := Get(state, 2, "branch_a"); contractCode(t, err) != CodeBranchProjectMismatch {
		t.Fatalf("cross-project branch code = %v", err)
	}
	var request UpdateRequest
	if err := json.Unmarshal([]byte(`{"enable_equipments":["FOREIGN_rifle"]}`), &request); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(state, 1, "branch_a", request); contractCode(t, err) != CodeEquipmentProjectMismatch {
		t.Fatalf("cross-project equipment code = %v", err)
	}
	if err := json.Unmarshal([]byte(`{"enable_equipments":["missing"]}`), &request); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(state, 1, "branch_a", request); contractCode(t, err) != CodeEquipmentMissing {
		t.Fatalf("missing equipment code = %v", err)
	}
}

func TestImmutableGeometryAndUnknownFieldsAreRejected(t *testing.T) {
	state := testState()
	for _, raw := range []string{
		`{"folder":"air_techs_folder"}`,
		`{"gridbox_id":"free_tree"}`,
		`{"start_year":1944}`,
		`{"position_x":99}`, `{"position_y":99}`,
		`{"folder_placements":[]}`,
		`{"path":[]}`,
	} {
		var request UpdateRequest
		if err := json.Unmarshal([]byte(raw), &request); err != nil {
			t.Fatal(err)
		}
		if _, err := Update(state, 1, "branch_a", request); contractCode(t, err) != CodeImmutableField {
			t.Fatalf("immutable payload %s returned %v", raw, err)
		}
	}
	var request UpdateRequest
	if err := json.Unmarshal([]byte(`{"free_coordinates":{"x":1}}`), &request); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(state, 1, "branch_a", request); contractCode(t, err) != CodeUnknownField {
		t.Fatalf("unknown field returned %v", err)
	}
}

func TestDeleteBlockersAreDeterministicAndNonCascading(t *testing.T) {
	state := testState()
	state.Branches[1].TechnologyIDs = []string{"TST_a", "TST_child"}
	state.Technologies[0].EnableEquipments = []string{"TST_rifle"}
	state.Technologies = append(state.Technologies, Technology{
		DatabaseID: 13, ProjectID: 1, TechID: "TST_child", Name: "Child",
		Folder: "infantry_folder", StartYear: 1939,
		FolderPlacements: []FolderPlacement{{Folder: "infantry_folder", X: 0, Y: 6}},
	})
	state.ExternalReferences = []ExternalReference{
		{ProjectID: 1, SourceKind: "focus", SourceID: "focus_z", Field: "completion_reward", TargetTechnologyID: "TST_a"},
		{ProjectID: 1, SourceKind: "technology", SourceID: "external_tech", Field: "dependencies", TargetTechnologyID: "TST_child"},
	}
	_, err := Delete(state, 1, "branch_a")
	var blocked DeleteBlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("expected DeleteBlockedError, got %T: %v", err, err)
	}
	wantKinds := []string{BlockerChildTechnology, BlockerEquipmentUnlock, BlockerExternalReference, BlockerExternalReference}
	gotKinds := make([]string, 0, len(blocked.Blockers))
	for _, blocker := range blocked.Blockers {
		gotKinds = append(gotKinds, blocker.Kind)
	}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("blocker order = %#v, want %#v", gotKinds, wantKinds)
	}
	if blocked.Blockers[0].SourceID != "TST_child" || blocked.Blockers[1].TargetID != "TST_rifle" {
		t.Fatalf("delete blockers lost exact identities: %#v", blocked.Blockers)
	}
	if blocked.Code != CodeDeleteBlocked {
		t.Fatalf("delete code = %q", blocked.Code)
	}
}

func TestDeleteReturnsOnlyExplicitEmptyBranchPlan(t *testing.T) {
	state := testState()
	plan, err := Delete(state, 1, "branch_a")
	if err != nil {
		t.Fatal(err)
	}
	if plan.ProjectID != 1 || plan.BranchID != "branch_a" || plan.TechnologyDatabaseID != 11 || plan.RootTechnologyID != "TST_a" {
		t.Fatalf("unexpected delete plan: %#v", plan)
	}
}

func TestBranchViewSerializationIsStable(t *testing.T) {
	view, err := Get(testState(), 1, "branch_a")
	if err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var reopened BranchView
	if err := json.Unmarshal(first, &reopened); err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(reopened)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("branch GET shape is not stable after reopen:\n%s\n%s", first, second)
	}
}
