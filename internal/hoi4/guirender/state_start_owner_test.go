package guirender

import "testing"

const testYananFile = `
state = {
	id = 1032
	name = "STATE_1032"
	manpower = 4670800
	state_category = town
	history = {
		owner = PRC
		add_core_of = PRC
		1936.1.1 = {
			IF = {
				limit = { has_dlc = "No Compromise, No Surrender" }
				transfer_state_to = XIC
				add_core_of = XIC
			}
		}
		1938.10.25 = {
			owner = PRC
			controller = PRC
		}
	}
	provinces = { 517 5007 }
}
`

const testGansuFile = `
state = {
	id = 283
	name = "STATE_283"
	history = {
		1938.10.25 = { owner = CHI }
		1936.1.1 = {
			IF = {
				limit = { has_dlc = "No Compromise, No Surrender" }
				XIC = { transfer_state = PREV }
			}
		}
		owner = XSM
	}
	provinces = { 531 556 }
}
`

const testNoDLCFile = `
state = {
	id = 744
	name = "STATE_744"
	history = {
		owner = CHI
		1936.1.1 = {
			IF = {
				limit = { NOT = { has_dlc = "No Compromise, No Surrender" } }
				transfer_state_to = SHX
			}
		}
	}
	provinces = { 600 }
}
`

func TestStateOwnerCountsGameStartTransfers(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/states/1032 - Yanan.txt": []byte(testYananFile),
		"history/states/283-China.txt":    []byte(testGansuFile),
		"history/states/744-Xian.txt":     []byte(testNoDLCFile),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	owners := map[int]map[string]string{}
	for _, date := range []string{"", "1936.1.1", "1938.10.25"} {
		for _, state := range idx.effectiveStates() {
			if owners[state.ID] == nil {
				owners[state.ID] = map[string]string{}
			}
			owners[state.ID][date] = idx.ownerOfState(state, date)
		}
	}
	want := map[int]map[string]string{
		1032: {"": "XIC", "1936.1.1": "XIC", "1938.10.25": "PRC"},
		283:  {"": "XIC", "1936.1.1": "XIC", "1938.10.25": "CHI"},
		744:  {"": "CHI", "1936.1.1": "CHI", "1938.10.25": "CHI"},
	}
	for id, dates := range want {
		for date, owner := range dates {
			if got := owners[id][date]; got != owner {
				t.Errorf("state %d at %q: owner %q, want %q", id, date, got, owner)
			}
		}
	}

	payload := idx.WorldMap("1936.1.1")
	for _, row := range payload.States {
		if id, _ := row[0].(int); id == 1032 {
			if owner, _ := row[2].(string); owner != "XIC" {
				t.Errorf("world map: Yan'an drawn as %q, want XIC", owner)
			}
		}
	}
}
