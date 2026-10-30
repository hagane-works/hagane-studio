package guirender

import "testing"

func TestLogisticsOnTheRealInstall(t *testing.T) {
	idx := realTopbarIndex(t, false)
	usage := idx.factoryUsage("GER", "1936.1.1.12")
	want := map[string]int{"military_used": 20, "military_total": 28, "naval_used": 10, "naval_total": 10}
	for key, value := range want {
		if usage[key] != value {
			t.Errorf("%s = %d, want %d (%v)", key, usage[key], value, usage)
		}
	}
	if lines := idx.ProductionLinesOf("GER", "1936.1.1.12"); len(lines) != 21 || lines[0].Equipment != "infantry_equipment_1" || lines[0].Factories != 10 {
		t.Errorf("lines %d, first %+v", len(lines), lines[0])
	}
}
