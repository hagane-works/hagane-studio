package world

import (
	"strings"
	"testing"
)

func TestHistoryKeepsAutonomyStatesAndCountryLeaders(t *testing.T) {
	text := `capital = 1
set_autonomy = { target = SHX autonomy_state = fds_nominal freedom_level = 0.0 }
set_autonomy = { target = MAN autonomous_state = autonomy_puppet }
set_autonomy = { target = MEN autonomy_state = autonomy_puppet end_wars = no }
create_country_leader = {
	name = "Ahmed Ben Bella"
	picture = GFX_portrait_Ahmed_Ben_Bella
	ideology = fds_095_personalist_dictatorship
}
`
	items, _, err := ImportHistoryFiles(text, "ALG - Algeria.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	list, _ := items[0].MustGet("set_autonomy").([]any)
	if len(list) != 2 {
		t.Fatalf("set_autonomy = %v", list)
	}
	get := func(d any, key string) any {
		v, _ := d.(*PyDict).Get(key)
		return v
	}
	if get(list[0], "target") != "SHX" || get(list[0], "autonomous_state") != "fds_nominal" ||
		get(list[0], "state_key") != "autonomy_state" || get(list[0], "freedom_level") != "0.0" {
		t.Errorf("first set_autonomy: target %v state %v key %v level %v",
			get(list[0], "target"), get(list[0], "autonomous_state"), get(list[0], "state_key"), get(list[0], "freedom_level"))
	}
	if get(list[1], "target") != "MAN" || get(list[1], "autonomous_state") != "autonomy_puppet" ||
		get(list[1], "state_key") != nil || get(list[1], "freedom_level") != nil {
		t.Errorf("second set_autonomy: %v", list[1])
	}
	raw, _ := items[0].MustGet("raw_script").(string)
	for _, want := range []string{"end_wars = no", "create_country_leader", "Ahmed Ben Bella", "fds_095_personalist_dictatorship"} {
		if !strings.Contains(raw, want) {
			t.Errorf("raw_script lost %q:\n%s", want, raw)
		}
	}
}
