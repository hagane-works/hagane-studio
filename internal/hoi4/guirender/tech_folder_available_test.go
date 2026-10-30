package guirender

import (
	"reflect"
	"testing"

	"haganeworks/internal/hoi4/paradox"
)

func TestTechFoldersFollowTheCountrysAvailable(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/scripted_triggers/sg.txt": []byte(`SG_is_sanguo_country = { OR = { tag = WEI tag = SHU } }
SG_is_wei = { SG_is_sanguo_country = yes tag = WEI }`),
		"common/technology_tags/00_technology.txt": []byte(`technology_folders = {
	infantry_folder = { ledger = army }
	armour_folder = { ledger = army available = { NOT = { SG_is_sanguo_country = yes } } }
	wei_folder = { ledger = army available = { SG_is_wei = yes } }
	naval_folder = { ledger = navy available = { is_in_faction = yes } }
}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	names := func(tag string) []string {
		var out []string
		for _, folder := range idx.TechTreePayload(tag, "", "")["folders"].([]map[string]any) {
			out = append(out, folder["name"].(string))
		}
		return out
	}
	if got := names("WEI"); !reflect.DeepEqual(got, []string{"infantry_folder", "wei_folder", "naval_folder"}) {
		t.Errorf("WEI: %v", got)
	}
	if got := names("SHU"); !reflect.DeepEqual(got, []string{"infantry_folder", "naval_folder"}) {
		t.Errorf("SHU: %v", got)
	}
	if got := names("GER"); !reflect.DeepEqual(got, []string{"infantry_folder", "armour_folder", "naval_folder"}) {
		t.Errorf("GER: %v", got)
	}

	if got := len(idx.TechFolders()); got != 4 {
		t.Errorf("all folders: %d", got)
	}
}

func TestStartDecidesScriptedTriggers(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/scripted_triggers/sg.txt": []byte(`is_wei = { tag = WEI }
loops = { loops = yes }`),
	})
	if err != nil {
		t.Fatal(err)
	}
	state := idx.decisionStartOf("WEI", "")
	for _, c := range []struct {
		text             string
		holds, decidable bool
	}{
		{`is_wei = yes`, true, true},
		{`is_wei = no`, false, true},
		{`NOT = { is_wei = yes }`, false, true},
		{`loops = yes`, false, false},
		{`unknown_trigger = yes`, false, false},
	} {
		root, _, err := paradox.Parse(c.text)
		if err != nil {
			t.Fatal(err)
		}
		holds, decided := state.eval(root.Entries, false)
		if holds != c.holds || decided != c.decidable {
			t.Errorf("%s: holds %v decided %v, want %v %v", c.text, holds, decided, c.holds, c.decidable)
		}
	}
}
