package guirender

import "testing"

const testTradeStates = `
state = {
	id = 1
	manpower = 100
	resources = { steel = 100 rubber = 40 }
	history = { owner = TAG }
	provinces = { 1 }
}
`

const testTradeStatesTwo = `
state = {
	id = 2
	manpower = 100
	resources = { rubber = 200 }
	history = { owner = SUB }
	provinces = { 2 }
}
`

const testTradeStatesThree = `
state = {
	id = 3
	manpower = 100
	resources = { steel = 80 }
	history = { owner = FRE }
	provinces = { 3 }
}
`

const testTradeIdeas = `
ideas = {
	trade_laws = {
		law = yes
		free_trade = {
			default = yes
			modifier = { min_export = 0.5 }
		}
		limited_exports = {
			modifier = { min_export = 0.25 }
		}
	}
}
`

const testAutonomy = `
autonomy_state = {
	id = autonomy_puppet
	default = yes
	modifier = {
		extra_trade_to_overlord_factor = 1.0
	}
}
`

const testAutonomyDominion = `
autonomy_state = {
	id = autonomy_dominion
	modifier = {
		extra_trade_to_overlord_factor = 0.25
	}
}
`

const testTradeHistoryTag = `
capital = 1
add_ideas = { limited_exports }
puppet = SUB
`

const testTradeHistoryFree = `
capital = 3
`

func tradeIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex(map[string][]byte{
		"history/states/1.txt":                    []byte(testTradeStates),
		"history/states/2.txt":                    []byte(testTradeStatesTwo),
		"history/states/3.txt":                    []byte(testTradeStatesThree),
		"common/ideas/_economic.txt":              []byte(testTradeIdeas),
		"common/autonomous_states/puppet.txt":     []byte(testAutonomy),
		"common/autonomous_states/dominion.txt":   []byte(testAutonomyDominion),
		"common/resources/00_resources.txt":       []byte(testResources),
		"history/countries/TAG - Tagland.txt":     []byte(testTradeHistoryTag),
		"history/countries/FRE - Freeland.txt":    []byte(testTradeHistoryFree),
		"history/countries/SUB - Subjectland.txt": []byte("capital = 2\n"),
		"common/defines/00_defines.lua": []byte("NDefines.NBuildings.INFRASTRUCTURE_RESOURCE_BONUS = 0.15\n" +
			"NDefines.NBuildings.SUPPLY_ROUTE_RESOURCE_BONUS = 0.2\n"),
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestTradeLawComesFromHistoryOrTheGroupsDefault(t *testing.T) {
	idx := tradeIndex(t)
	if law, share := idx.TradeLawOf("TAG", "1936.1.1"); law != "limited_exports" || share != 0.25 {
		t.Errorf("TAG = %v/%v, want the law its history adds", law, share)
	}

	if law, share := idx.TradeLawOf("FRE", "1936.1.1"); law != "free_trade" || share != 0.5 {
		t.Errorf("FRE = %v/%v, want the group's default", law, share)
	}
}

func TestTradeOfferIsOutputTimesTheLaw(t *testing.T) {
	payload := tradeIndex(t).TradePayload("TAG", "1936.1.1")
	columns, _ := payload["resources"].([]map[string]any)
	byName := map[string]map[string]any{}
	for _, column := range columns {
		byName[column["resource"].(string)] = column
	}

	if byName["steel"]["produced"] != 120 || byName["steel"]["exported"] != 30 || byName["steel"]["surplus"] != 90 {
		t.Errorf("steel column = %v", byName["steel"])
	}

	if byName["oil"]["produced"] != 0 {
		t.Errorf("oil column = %v", byName["oil"])
	}
}

func TestSubjectsTradeThroughTheirOverlord(t *testing.T) {
	payload := tradeIndex(t).TradePayload("FRE", "1936.1.1")
	sellers, _ := payload["sellers"].([]map[string]any)
	offers := map[string]map[string]int{}
	for _, seller := range sellers {
		available, _ := seller["available"].(map[string]int)
		offers[seller["tag"].(string)] = available
	}

	if _, listed := offers["SUB"]; listed {
		t.Errorf("a puppet that gives away all of its trade is still listed: %v", offers["SUB"])
	}
	if offers["TAG"]["rubber"] != 110 {

		t.Errorf("TAG rubber offer = %v, want its own 10 plus the puppet's 100", offers["TAG"]["rubber"])
	}
}
