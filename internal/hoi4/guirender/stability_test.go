package guirender

import (
	"math"
	"testing"
)

func TestStabilityAddsWhatTheGameAdds(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"history/countries/TAG - Tagland.txt": []byte("capital = 1\nset_oob = \"TAG_1936\"\nset_stability = 0.7\nset_war_support = 0.3\n" +
			"set_politics = { ruling_party = fascism }\nset_popularities = { fascism = 40 democratic = 60 }\n"),
		"common/ideologies/00_ideologies.txt": []byte("ideologies = { fascism = { types = { nazism = { } } } democratic = { types = { liberalism = { } } } }\n"),
		"common/buildings/01_landmark_buildings.txt": []byte("buildings = { landmark_hall = { is_buildable = no level_cap = { province_max = 1 }\n" +
			" country_modifiers = { modifiers = { stability_factor = 0.05 } } } }\n"),
		"history/states/1-Home.txt": []byte("state = { id = 1 provinces = { 10 } history = { owner = TAG buildings = { 10 = { landmark_hall = { level = 1 allowed = { has_dlc = \"X\" } } } } } }\n"),
		"history/units/TAG_1936.txt": []byte("units = { fleet = { name = \"F\" task_force = { name = \"T\" location = 10\n" +
			" ship = { name = \"Pride\" pride_of_the_fleet = yes definition = heavy_cruiser equipment = { ship_hull_cruiser_1 = { amount = 1 owner = TAG } } } } } }\n"),
		"common/modifiers/00_static_modifiers.txt": []byte("stability_good_modifier = { industrial_capacity_factory = 0.20 consumer_goods_factor = -0.20 }\n" +
			"stability_bad_modifier = { industrial_capacity_factory = -0.50 }\npride_of_the_fleet_country = { war_support_factor = 0.05 }\n"),
		"common/defines/00_defines.lua": []byte("NDefines.NCountry.BASE_STABILITY_PARTY_POPULARITY_FACTOR = 0.15\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	sums := idx.startingModifiers("TAG", "1936.1.1")
	stability, warSupport := idx.startingStability("TAG", "1936.1.1", sums)
	if math.Abs(stability-0.81) > 1e-9 || math.Abs(warSupport-0.35) > 1e-9 {
		t.Errorf("stability/war support = %v/%v, want 0.81/0.35", stability, warSupport)
	}

	if math.Abs(sums["industrial_capacity_factory"]-0.124) > 1e-9 || math.Abs(sums["consumer_goods_factor"]+0.124) > 1e-9 {
		t.Errorf("factory output %v, consumer goods %v, want +0.124 / -0.124", sums["industrial_capacity_factory"], sums["consumer_goods_factor"])
	}
	stats := idx.CountryStatsPayload("TAG", "1936.1.1")
	if math.Abs(stats["stability"].(float64)-0.81) > 1e-9 {
		t.Errorf("topbar stability %v", stats["stability"])
	}
}

func TestLowStabilityCostsOutput(t *testing.T) {
	idx, err := NewIndex(map[string][]byte{
		"common/modifiers/00_static_modifiers.txt": []byte("stability_good_modifier = { industrial_capacity_factory = 0.20 }\nstability_bad_modifier = { industrial_capacity_factory = -0.50 }\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	source, ok := idx.stabilitySource(0.3)
	if !ok || math.Abs(source.Values["industrial_capacity_factory"]+0.2) > 1e-9 {
		t.Errorf("30%% stability: %v %v, want -0.2 output", ok, source)
	}
	if _, ok := idx.stabilitySource(0.5); ok {
		t.Error("50% stability applied a modifier")
	}
}
