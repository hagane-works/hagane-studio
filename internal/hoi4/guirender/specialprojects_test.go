package guirender

import (
	"fmt"
	"strings"
	"testing"
)

func TestRealGameExperimentalFacilities(t *testing.T) {
	idx := realTopbarIndex(t, false)
	for _, want := range []struct {
		tag, bands, facility string
	}{
		{"ENG", "Physics 6 0 0|Naval 19 0 0.6|Air 14 0 0.5|Land 9 0 0", "Sussex naval_facility Naval Engineering Facility 18"},
		{"GER", "Physics 6 0 0|Naval 19 0 0.8|Air 15 0 0.3|Land 9 1 0", "Brandenburg land_facility Land Warfare Facility 19"},
	} {
		out := idx.SpecialProjectsPayload(want.tag, "1936.1.1.12")
		bands := []string{}
		for _, band := range out["specializations"].([]map[string]any) {
			bands = append(bands, fmt.Sprintf("%v %v %v %.2g", band["label"], band["projects"], band["points"], band["progress"]))
		}
		if strings.Join(bands, "|") != want.bands {
			t.Errorf("%s bands = %s", want.tag, strings.Join(bands, "|"))
		}
		facilities := out["facilities"].([]map[string]any)
		got := []string{}
		for _, facility := range facilities {
			got = append(got, fmt.Sprintf("%v %v %v %v", facility["state_name"], facility["building"], facility["label"], facility["icon_frame"]))
		}
		if len(got) != 1 || got[0] != want.facility {
			t.Errorf("%s facilities = %v", want.tag, got)
		}
	}
}

func TestRealGameNavalProjectPicker(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.SpecialProjectsPayload("ENG", "1936.1.1.12")
	open, locked := []string{}, []string{}
	for _, project := range out["projects"].([]map[string]any) {
		if project["specialization"] != "specialization_naval" {
			continue
		}
		if project["available"] == true {
			open = append(open, project["label"].(string))
		} else {
			locked = append(locked, project["label"].(string))
		}
	}
	want := "Cruiser Submarine|Midget Submarine|Torpedo Cruiser|Jellicoe Class Battleship|Underway Replenishment|Support Ships|Escort Carrier"
	if strings.Join(open, "|") != want {
		t.Errorf("open = %s", strings.Join(open, "|"))
	}
	if len(open)+len(locked) != 19 {
		t.Errorf("locked = %v", locked)
	}
}
