package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCentralAcceptanceOOBWiringModGUN(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 6073,
		Project:   []byte(`{"id":6073,"name":"mod_GUN","country_tag":"GUN","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"GUN","source":"created"}`),
				json.RawMessage(`{"id":2,"tag":"TOM","source":"created"}`),
			},
			"states": {
				json.RawMessage(`{"id":570,"state_id":570,"name":"TS 12","owner":"GUN","provinces":[1790,4840,4866,7274,7722,10555],"source":"created"}`),
				json.RawMessage(`{"id":578,"state_id":578,"name":"Siberia 5","owner":"TOM","provinces":[7789,7793],"source":"created"}`),
			},
			"history_files": {
				json.RawMessage(`{"id":1,"country_tag":"GUN","capital":570,"raw_script":"# 作战序列\nset_oob = \"TAG_1936\"\n","source":"created"}`),
				json.RawMessage(`{"id":2,"country_tag":"TOM","capital":578,"raw_script":"","source":"created"}`),
			},
			"division_templates": {
				json.RawMessage(`{"id":1,"oob_file":"new_1978","template_name":"New Division","country_tag":"","regiments":[{"token":"infantry","x":0,"y":0}],"support":[],"source":"created"}`),
			},
			"division_deployments": {
				json.RawMessage(`{"id":1,"oob_file":"new_1978","division_template":"New Division","location":7274,"count":11,"country_tag":"","source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, e := range artifact.Entries {
		contents[e.Path] = string(e.Bytes)
	}

	if _, ok := contents["mod_GUN/history/units/new_1978.txt"]; !ok {
		t.Fatalf("OOB 文件没发出来; paths=%v", keysOfContents(contents))
	}

	gun := contents["mod_GUN/history/countries/GUN - zmod_GUN.txt"]
	if !strings.Contains(gun, `set_oob = "new_1978"`) {
		t.Fatalf("GUN history 没有加载 new_1978:\n%s", gun)
	}

	if n := strings.Count(gun, `set_oob = "new_1978"`); n != 1 {
		t.Fatalf("set_oob 发了 %d 次,只允许 1 次", n)
	}
	if strings.Contains(gun, "set_air_oob") || strings.Contains(gun, "set_naval_oob") {
		t.Fatalf("混装文件不该再发 air/naval 引用:\n%s", gun)
	}

	tom := contents["mod_GUN/history/countries/TOM - zmod_GUN.txt"]
	if strings.Contains(tom, "set_oob") {
		t.Fatalf("TOM 被错误接线:\n%s", tom)
	}

	joined := strings.Join(artifact.Warnings, "\n")
	if !strings.Contains(joined, "TAG_1936") {
		t.Fatalf("悬空引用 TAG_1936 没有告警; warnings=%v", artifact.Warnings)
	}

	if !strings.Contains(joined, "new_1978") {
		t.Fatalf("自动接线没有告知作者; warnings=%v", artifact.Warnings)
	}
	t.Logf("GUN history:\n%s", gun)
	t.Logf("warnings:\n%s", joined)
}

func keysOfContents(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestCentralAcceptanceOOBWiringNavyBerthedInAnotherCountry(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 6073,
		Project:   []byte(`{"id":6073,"name":"mod_GUN","country_tag":"GUN","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"GUN","source":"created"}`),
				json.RawMessage(`{"id":2,"tag":"TOM","source":"created"}`),
			},
			"states": {
				json.RawMessage(`{"id":570,"state_id":570,"name":"TS 12","owner":"TOM","provinces":[1790,4840,4866,7274,7722,10555],"source":"created"}`),
			},
			"history_files": {
				json.RawMessage(`{"id":1,"country_tag":"GUN","capital":570,"raw_script":"","source":"created"}`),
			},

			"oob_fleets": {
				json.RawMessage(`{"id":20,"oob_file":"TOM_navy","country_tag":"","name":"TOM Fleet","naval_base":"7274","ordering":0,"layout":[],"source":"created"}`),
			},
			"oob_task_forces": {
				json.RawMessage(`{"id":21,"fleet_id":20,"name":"1st Task Force","location":"7274","ordering":0,"layout":[],"source":"created"}`),
			},
			"oob_ships": {
				json.RawMessage(`{"id":22,"task_force_id":21,"name":"TNS Tom","definition":"destroyer","equipment_type":"ship_hull_light_1","amount":1,"owner":"TOM","ordering":0,"layout":[],"source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, e := range artifact.Entries {
		contents[e.Path] = string(e.Bytes)
	}
	if _, ok := contents["mod_GUN/history/units/TOM_navy.txt"]; !ok {
		t.Fatalf("海军 OOB 文件没发出来; paths=%v", keysOfContents(contents))
	}
	tom := contents["mod_GUN/history/countries/TOM - zmod_GUN.txt"]
	gun := contents["mod_GUN/history/countries/GUN - zmod_GUN.txt"]
	if n := strings.Count(tom, `set_oob = "TOM_navy"`); n != 1 {
		t.Fatalf("TOM 的 history 加载 TOM_navy %d 次,要求恰好 1 次:\n%s", n, tom)
	}
	if strings.Contains(gun, "TOM_navy") {
		t.Fatalf("项目主 tag GUN 抢走了 TOM 的舰队(这正是 H-2 的病):\n%s", gun)
	}
	joined := strings.Join(artifact.Warnings, "\n")
	if !strings.Contains(joined, "TOM_navy") {
		t.Fatalf("自动接线没有告知作者; warnings=%v", artifact.Warnings)
	}
	t.Logf("TOM history:\n%s", tom)
	t.Logf("warnings:\n%s", joined)
}

func TestCentralAcceptanceOOBWiringNavyLocationFields(t *testing.T) {
	cases := []struct {
		name       string
		navalBase  string
		tfLocation string
	}{
		{"fleet_naval_base", "7274", ""},
		{"task_force_location", "", "7274"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := Snapshot{
				ProjectID: 6073,
				Project:   []byte(`{"id":6073,"name":"mod_GUN","country_tag":"GUN","export_languages":["en"]}`),
				Tables: map[string][]json.RawMessage{
					"countries": {
						json.RawMessage(`{"id":1,"tag":"GUN","source":"created"}`),
						json.RawMessage(`{"id":2,"tag":"TOM","source":"created"}`),
					},
					"states": {
						json.RawMessage(`{"id":570,"state_id":570,"name":"TS 12","owner":"TOM","provinces":[1790,7274],"source":"created"}`),
					},
					"oob_fleets": {
						json.RawMessage(`{"id":20,"oob_file":"new_1978","country_tag":"","name":"Fleet","naval_base":"` + tc.navalBase + `","ordering":0,"layout":[],"source":"created"}`),
					},
					"oob_task_forces": {
						json.RawMessage(`{"id":21,"fleet_id":20,"name":"1st Task Force","location":"` + tc.tfLocation + `","ordering":0,"layout":[],"source":"created"}`),
					},
					"oob_ships": {
						json.RawMessage(`{"id":22,"task_force_id":21,"name":"TNS Tom","definition":"destroyer","equipment_type":"ship_hull_light_1","amount":1,"owner":"TOM","ordering":0,"layout":[],"source":"created"}`),
					},
				},
			}
			artifact, err := BuildNativeShadow(snapshot, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			contents := map[string]string{}
			for _, e := range artifact.Entries {
				contents[e.Path] = string(e.Bytes)
			}
			tom := contents["mod_GUN/history/countries/TOM - zmod_GUN.txt"]
			gun := contents["mod_GUN/history/countries/GUN - zmod_GUN.txt"]
			if n := strings.Count(tom, `set_oob = "new_1978"`); n != 1 {
				t.Fatalf("%s 没能定出归属;TOM history:\n%s\nGUN history:\n%s",
					tc.name, tom, gun)
			}
			if strings.Contains(gun, "new_1978") {
				t.Fatalf("又接给了项目主 tag:\n%s", gun)
			}
			joined := strings.Join(artifact.Warnings, "\n")
			if !strings.Contains(joined, "naval base") {
				t.Fatalf("告警没说清是按泊位推的; warnings=%v", artifact.Warnings)
			}
			t.Logf("TOM history:\n%s", tom)
			t.Logf("warnings:\n%s", joined)
		})
	}
}

func TestCentralAcceptanceOOBWiringAirWingUsesStateOwner(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 6073,
		Project:   []byte(`{"id":6073,"name":"mod_GUN","country_tag":"GUN","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"countries": {
				json.RawMessage(`{"id":1,"tag":"GUN","source":"created"}`),
				json.RawMessage(`{"id":2,"tag":"TOM","source":"created"}`),
			},
			"states": {
				json.RawMessage(`{"id":1,"state_id":570,"name":"TS 12","owner":"TOM","provinces":[1790,7274],"source":"created"}`),
				json.RawMessage(`{"id":2,"state_id":578,"name":"Siberia 5","owner":"GUN","provinces":[570,7789],"source":"created"}`),
			},
			"oob_air_wing_groups": {
				json.RawMessage(`{"id":10,"oob_file":"new_1978","country_tag":"","location":"570","ordering":0,"layout":[],"source":"created"}`),
			},
			"oob_air_wings": {
				json.RawMessage(`{"id":11,"group_id":10,"name":"1. Fliegerkorps","equipment_type":"small_plane_airframe_0","amount":18,"owner":"TOM","ordering":0,"source":"created"}`),
			},
		},
	}
	artifact, err := BuildNativeShadow(snapshot, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, e := range artifact.Entries {
		contents[e.Path] = string(e.Bytes)
	}
	tom := contents["mod_GUN/history/countries/TOM - zmod_GUN.txt"]
	gun := contents["mod_GUN/history/countries/GUN - zmod_GUN.txt"]
	if n := strings.Count(tom, `set_oob = "new_1978"`); n != 1 {
		t.Fatalf("空军 OOB 没接给州 570 的 owner TOM;TOM history:\n%s\nGUN history:\n%s",
			tom, gun)
	}
	if strings.Contains(gun, "new_1978") {
		t.Fatalf("把 location 当省份读了(省份 570 属于 GUN 的州 578):\n%s", gun)
	}
	if !strings.Contains(strings.Join(artifact.Warnings, "\n"), "air wings are based in") {
		t.Fatalf("告警没说清是按机场所在州推的; warnings=%v", artifact.Warnings)
	}
}
