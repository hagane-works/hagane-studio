package guirender

import "strings"

const prideOfTheFleet = "pride_of_the_fleet_country"

func (idx *Index) startingStability(tag, date string, sums map[string]float64) (stability, warSupport float64) {
	start, _ := idx.countryStartAt(tag, date)
	stability = start.Stability + sums["stability_factor"]
	warSupport = start.WarSupport + sums["war_support_factor"]
	if politics := idx.countryPoliticsFor(tag); politics != nil {
		if ruling := idx.RulingPartyOf(tag); ruling != "" {
			stability += float64(politics.Popularities[ruling]) / 100 * idx.defineNumber("NCountry.BASE_STABILITY_PARTY_POPULARITY_FACTOR")
		}
	}
	return clamp01(stability), clamp01(warSupport)
}

func (idx *Index) stabilitySource(stability float64) (modifierSource, bool) {
	name, share := "stability_good_modifier", (stability-0.5)/0.5
	if stability < 0.5 {
		name, share = "stability_bad_modifier", (0.5-stability)/0.5
	}
	block := idx.staticModifierFor(name)
	if block == nil || share <= 0 {
		return modifierSource{}, false
	}
	values := map[string]float64{}
	for key, value := range numericEntries(block.Entries, nil) {
		values[key] = value * share
	}
	return modifierSource{"stability " + name, values}, true
}

func (idx *Index) buildingModifierSources(tag, date string) []modifierSource {
	out := []modifierSource{}
	for name, level := range idx.builtBuildings(tag, date) {
		if level <= 0 {
			continue
		}
		record := idx.buildingRecordFor(name)
		if record == nil || len(record.CountryModifiers) == 0 {
			continue
		}
		values := map[string]float64{}
		for key, value := range record.CountryModifiers {

			if strings.HasPrefix(key, "max_fuel") || strings.HasPrefix(key, "fuel_gain") {
				continue
			}
			values[key] = value * float64(level)
		}
		if len(values) > 0 {
			out = append(out, modifierSource{"building " + name, values})
		}
	}
	return out
}

func (idx *Index) hasPrideOfTheFleet(tag, date string) bool {
	for _, name := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(name)
		if !ok {
			continue
		}
		for _, force := range record.TaskForces {
			for _, ship := range force.Ships {
				if ship.Pride {
					return true
				}
			}
		}
	}
	return false
}
