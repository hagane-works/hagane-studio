package guirender

import (
	"sort"
	"strings"
)

type StateFact struct {
	ID        int
	Name      string
	Provinces []int
	Owner     string
	Cores     []string
	Manpower  int
	Category  string

	VictoryPoints map[int]int

	Buildings map[string]int
	Resources map[string]int
}

var stateWideBuildings = map[string]bool{
	"infrastructure": true, "industrial_complex": true, "arms_factory": true, "dockyard": true,
	"air_base": true, "anti_air_building": true, "synthetic_refinery": true, "fuel_silo": true,
	"radar_station": true, "rocket_site": true, "nuclear_reactor": true,
}

func (idx *Index) StateFacts(date string) []StateFact {
	records := idx.effectiveStates()
	out := make([]StateFact, 0, len(records))
	for _, record := range records {
		fact := StateFact{
			ID: record.ID, Name: record.Name, Provinces: append([]int(nil), record.Provinces...),
			Owner: idx.ownerOfState(record, date), Manpower: record.Manpower, Category: record.Category,
			VictoryPoints: map[int]int{}, Buildings: map[string]int{}, Resources: map[string]int{},
		}
		for province, value := range record.VictoryPoints {
			fact.VictoryPoints[province] = value
		}
		for name, level := range record.buildingsAt(date) {
			if stateWideBuildings[name] && level > 0 {
				fact.Buildings[name] = level
			}
		}
		for name, amount := range record.Resources {
			fact.Resources[name] = amount
		}
		seen := map[string]bool{}
		for _, tag := range tokensByOwned(record.Cores, date, idx.ownedDLC) {
			tag = strings.ToUpper(tag)
			if !seen[tag] {
				seen[tag] = true
				fact.Cores = append(fact.Cores, tag)
			}
		}
		sort.Strings(fact.Cores)
		out = append(out, fact)
	}
	return out
}
