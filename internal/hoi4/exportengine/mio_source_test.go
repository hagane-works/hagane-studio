package exportengine

import (
	"encoding/json"
	"strings"
	"testing"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

func TestMIOsGoBackToTheFileTheyCameFrom(t *testing.T) {
	snapshot := Snapshot{
		Schema: SnapshotSchema, ProjectID: 42,
		Project: json.RawMessage(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"military_industrial_organizations": {
				json.RawMessage(`{"id":1,"mio_id":"GER_porsche_organization","include":"generic_heavy_tank_organization","source":"created",
					"allowed":"original_tag = GER","traits":[{"kind":"add","token":"GER_mio_trait_hoesch_facilities","name":"GER_mio_trait_hoesch_facilities","raw_extras":{"position":"x = 9\ny = 0"}}],
					"raw_extras":{"__source_file":"common/military_industrial_organization/organizations/GER_organization.txt"},
					"display_name_locs":{},"description_locs":{}}`),
				json.RawMessage(`{"id":2,"mio_id":"GER_new_organization","source":"created","allowed":"original_tag = GER",
					"traits":[{"kind":"initial","name":"GER_new_trait","raw_extras":{}}],"raw_extras":{},
					"display_name_locs":{"en":"New Works"},"description_locs":{}}`),
			},
		},
	}
	files, err := BuildNativeTextTree(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	game := string(files["common/military_industrial_organization/organizations/GER_organization.txt"])
	own := string(files["common/military_industrial_organization/organizations/GER_organizations.txt"])
	if !strings.Contains(game, "GER_porsche_organization = {") || strings.Contains(game, "GER_new_organization") {
		t.Errorf("the game's file:\n%s", game)
	}
	if strings.Contains(game, "__source_file") {
		t.Error("the editor's marker reached the game file")
	}
	if !strings.Contains(own, "GER_new_organization = {") || strings.Contains(own, "GER_porsche_organization") {
		t.Errorf("the project's file:\n%s", own)
	}
	loc := ""
	for name, body := range files {
		if strings.Contains(name, "_mio_l_english") {
			loc += string(body)
		}
	}
	if strings.Contains(loc, "GER_mio_trait_hoesch_facilities") {
		t.Errorf("a taken-in trait's name was written over:\n%s", loc)
	}
	if !strings.Contains(loc, "GER_new_trait") || !strings.Contains(loc, "New Works") {
		t.Errorf("the project's own company lost its names:\n%s", loc)
	}
}

func TestMIOSourceFileTakesOnlyTheOrganizationsFolder(t *testing.T) {
	for file, want := range map[string]string{
		"common/military_industrial_organization/organizations/GER_organization.txt": "common/military_industrial_organization/organizations/GER_organization.txt",
		"common/ideas/germany.txt": "",
		"common/military_industrial_organization/organizations/../../ideas/x.txt": "",
		"common/military_industrial_organization/organizations/sub/x.txt":         "",
		"common/military_industrial_organization/organizations/.txt":              "",
	} {
		row := emitmisc.NewNS().Set("raw_extras", emitmisc.NewOD().Set(MIOSourceKey, file))
		if got := MIOSourceFile(row); got != want {
			t.Errorf("MIOSourceFile(%q) = %q, want %q", file, got, want)
		}
	}
	if got := MIOSourceFile(emitmisc.NewNS()); got != "" {
		t.Errorf("no raw_extras = %q", got)
	}
}
