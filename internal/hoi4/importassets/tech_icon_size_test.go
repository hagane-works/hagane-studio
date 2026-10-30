package importassets

import (
	"testing"

	"haganeworks/internal/gameindex"
)

func TestTechIconSizeFollowsTheBaseGameTech(t *testing.T) {
	t.Cleanup(gameindex.UseStatic(map[string][]byte{gameindex.TechIconSizes: []byte(`{"sizes": {
		"infantry_weapons": [120, 24], "basic_battleship": [285, 56], "early_fighter": [82, 40], "CAS1": [130, 46]}}`)}))
	for _, tc := range []struct {
		id   string
		want Size
		base bool
	}{
		{"infantry_weapons", Size{120, 24}, true},
		{"basic_battleship", Size{285, 56}, true},
		{"early_fighter", Size{82, 40}, true},
		{"cas1", Size{130, 46}, true},
		{" CAS1 ", Size{130, 46}, true},
		{"radio", Size{64, 64}, false},
		{"my_new_tech", Size{64, 64}, false},
	} {
		got, base := TechIconSize(tc.id)
		if got != tc.want || base != tc.base {
			t.Fatalf("TechIconSize(%q) = %v base=%v, want %v base=%v", tc.id, got, base, tc.want, tc.base)
		}
	}

	t.Cleanup(gameindex.SetDefault(nil))
	if got, base := TechIconSize("infantry_weapons"); got != (Size{64, 64}) || base {
		t.Fatalf("no game folder: %v base=%v", got, base)
	}
}
