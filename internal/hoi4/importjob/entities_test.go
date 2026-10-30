package importjob

import "testing"

func TestParseSimpleEntity(t *testing.T) {
	rows, err := parseSimpleEntity("common/dynamic_modifiers/a.txt", `x = { icon = GFX_x political_power_factor = 0.1 }`)
	if err != nil || len(rows) != 1 || rows[0].naturalKey != "x" || rows[0].kind != "dynamic_modifier" {
		t.Fatalf("rows=%#v err=%v", rows, err)
	}
}
