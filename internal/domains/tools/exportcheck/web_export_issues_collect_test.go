package exportcheck

import (
	"errors"
	"testing"
)

func TestExportGateDependenciesWired(t *testing.T) {
	s := &Store{}

	defer func() { _ = recover() }()
	if _, err := s.collectLocalisationReferenceIssues(nil, 1, false); errors.Is(err, ErrExportGateIncomplete) {
		t.Error("本地化引用扫描已移完,不该再返回 ErrExportGateIncomplete")
	}
	if _, err := s.closureGroups(nil, 1, false); errors.Is(err, ErrExportGateIncomplete) {
		t.Error("闭包分组已移完,不该再返回 ErrExportGateIncomplete")
	}
}

func TestAirWingIssuesDistinguishMissingFromInvalid(t *testing.T) {
	groups := []AirWingGroup{{ID: 1, Location: ""}, {ID: 2, Location: "not a token"}}
	wings := []AirWing{{ID: 10, GroupID: 1, EquipmentType: ""},
		{ID: 11, GroupID: 2, EquipmentType: "bad token"},
		{ID: 12, GroupID: 2, EquipmentType: "fighter_equipment_1"}}
	got := airWingLocationIssues(groups, wings)

	codes := map[string]int{}
	for _, issue := range got {
		codes[issue["code"].(string)]++
		if issue["severity"] != "blocker" {
			t.Errorf("%v 应当是 blocker", issue["code"])
		}
	}
	for code, want := range map[string]int{
		"oob_air_location_missing": 1, "oob_air_location_invalid": 1,
		"oob_air_equipment_type_missing": 1, "oob_air_equipment_type_invalid": 1,
	} {
		if codes[code] != want {
			t.Errorf("%s = %d want %d(全部:%v)", code, codes[code], want, codes)
		}
	}

	if len(got) != 4 {
		t.Errorf("共应 4 条,得到 %d", len(got))
	}
}
