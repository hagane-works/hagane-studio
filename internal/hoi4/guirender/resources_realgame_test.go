package guirender

import "testing"

func TestEnergyCoal(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/defines/00_defines.lua": []byte("NDefines = {\nNProduction = {\n" +
			"\tRESOURCE_TO_ENERGY_COEFFICIENT = 9.0,\n\tBASE_COUNTRY_ENERGY_PRODUCTION = 10.0,\n" +
			"\tENERGY_SCALING_COST_BY_FACTORY_COUNT = 0.0225,\n\tBASE_ENERGY_COST = 0.25,\n\tENERGY_COST_CAP = 6.6,\n},\n}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.energyCoal(73); got != 14 {
		t.Errorf("73 factories burn %d coal, want 14", got)
	}
	if got := idx.energyCoal(5); got != 0 {
		t.Errorf("5 factories burn %d coal, want none (the country's own 10 covers them)", got)
	}
}

func TestResourcesOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	balance := idx.resourceBalance("GER", "1936.1.1.12")
	want := map[string]map[string]int{
		"extracted":  {"oil": 4, "tungsten": 5, "steel": 301, "chromium": 5, "rubber": 0},
		"exported":   {"oil": 1, "tungsten": 1, "steel": 75, "chromium": 1},
		"production": {"oil": 0, "aluminium": 11, "rubber": 5, "tungsten": 1, "steel": 49, "chromium": 3, "coal": 14},
		"surplus":    {"oil": 3, "rubber": -5, "tungsten": 3, "steel": 177, "chromium": 1},
	}
	for row, values := range want {
		for resource, amount := range values {
			if got := balance[row][resource]; got != amount {
				t.Errorf("%s %s = %d, want %d", row, resource, got, amount)
			}
		}
	}
	stats := idx.CountryStatsPayload("GER", "1936.1.1.12")
	if manpower := stats["manpower"].(int); manpower < 1325000 || manpower >= 1335000 {
		t.Errorf("manpower %d, want 1.33M", manpower)
	}
	if pp := stats["political_power"].(float64); pp < 2 || pp >= 3 {
		t.Errorf("political power %v, want 2", pp)
	}
	fuel := stats["fuel"].(map[string]any)
	if days, _ := fuel["days"].(int); days < 456 || days > 493 {
		t.Errorf("fuel fills in %v days, want 1.3 years", fuel["days"])
	}
}
