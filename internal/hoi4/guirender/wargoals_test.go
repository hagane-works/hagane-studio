package guirender

import "testing"

func TestRealGameJustifyWarGoalAgainstPoland(t *testing.T) {
	idx := realTopbarIndex(t, false)
	out := idx.WarGoalPayload("GER", "POL", "1936.1.1.12")
	types := out["types"].([]map[string]any)
	if len(types) != 1 || types[0]["id"] != "take_state" || types[0]["label"] != "Conquer" {
		t.Fatalf("types = %v", types)
	}
	conquer := types[0]
	if conquer["base_cost"] != 200.0 || conquer["per_state_cost"] != 50.0 || conquer["limit"] != 5.0 || conquer["threat"] != 2.0 {
		t.Errorf("numbers = %v", conquer)
	}
	states := conquer["states"].([]map[string]any)
	names := map[string]bool{}
	for _, state := range states {
		names[state["name"].(string)] = true
	}
	if len(states) != 18 || !names["Warszawa"] || !names["Wilno"] || !names["Poznań"] || names["Warsaw"] {
		t.Errorf("states = %v", states)
	}
	if out["daily_pp"] != 0.2 {
		t.Errorf("daily pp = %v", out["daily_pp"])
	}
	texts := out["texts"].(map[string]any)
	if texts["no_wargoal"] != "No war goal selected" || texts["select_type"] != "Select a war goal type" ||
		texts["cost"] != "War goal will cost you $DAILYCOST|Y$ Political Power and last $DAYS|Y$ days" {
		t.Errorf("texts = %v", texts)
	}

	if self := idx.WarGoalPayload("GER", "GER", "1936.1.1.12")["types"].([]map[string]any); len(self) != 0 {
		t.Errorf("against itself = %v", self)
	}
}
