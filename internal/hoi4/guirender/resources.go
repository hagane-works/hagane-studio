package guirender

import (
	"math"
	"strings"
)

const energyResource = "coal"

func (idx *Index) extractedByOwner(date string) map[string]map[string]int {
	infraBonus := idx.defineNumber("NBuildings.INFRASTRUCTURE_RESOURCE_BONUS")
	routeBonus := idx.defineNumber("NBuildings.SUPPLY_ROUTE_RESOURCE_BONUS")
	raw := map[string]map[string]float64{}
	for _, state := range idx.effectiveStates() {
		owner := idx.ownerOfState(state, date)
		if owner == "" || len(state.Resources) == 0 {
			continue
		}
		into := raw[owner]
		if into == nil {
			into = map[string]float64{}
			raw[owner] = into
		}
		infra := float64(state.buildingsAt(date)["infrastructure"])
		for name, amount := range state.Resources {
			into[name] += float64(amount) * (1 + infraBonus*infra) * (1 + routeBonus)
		}
	}
	out := map[string]map[string]int{}
	for owner, resources := range raw {
		out[owner] = map[string]int{}
		for name, amount := range resources {
			out[owner][name] = int(math.Round(amount))
		}
	}
	return out
}

func (idx *Index) energyCoal(factories int) int {
	perFactory := idx.defineNumber("NProduction.BASE_ENERGY_COST") +
		idx.defineNumber("NProduction.ENERGY_SCALING_COST_BY_FACTORY_COUNT")*float64(factories)
	if most := idx.defineNumber("NProduction.ENERGY_COST_CAP"); most > 0 && perFactory > most {
		perFactory = most
	}
	need := float64(factories)*perFactory - idx.defineNumber("NProduction.BASE_COUNTRY_ENERGY_PRODUCTION")
	coalEnergy := idx.defineNumber("NProduction.RESOURCE_TO_ENERGY_COEFFICIENT")
	if need <= 0 || coalEnergy <= 0 {
		return 0
	}
	return int(math.Round(need / coalEnergy))
}

func (idx *Index) resourceBalance(tag, date string) map[string]map[string]int {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	extracted := idx.extractedByOwner(date)[tag]
	_, share := idx.TradeLawOf(tag, date)
	production := map[string]float64{}
	if queue, ok := idx.ProductionPayload(tag, date)["queue"].(map[string]any); ok {
		lines, _ := queue["lines"].([]map[string]any)
		for _, line := range lines {
			if resources, ok := line["resources"].(map[string]float64); ok {
				for name, amount := range resources {
					production[name] += amount
				}
			}
		}
	}
	built := idx.builtBuildings(tag, date)
	production[energyResource] += float64(idx.energyCoal(built["industrial_complex"] + built["arms_factory"] + built["dockyard"]))

	out := map[string]map[string]int{
		"extracted": {}, "imported": {}, "exported": {}, "projects": {}, "production": {}, "surplus": {},
	}
	for name := range idx.ResourceFrames() {
		dug := extracted[name]
		sold := int(math.Floor(float64(dug) * share))
		used := int(math.Round(production[name]))
		out["extracted"][name] = dug
		out["imported"][name] = 0
		out["exported"][name] = sold
		out["projects"][name] = 0
		out["production"][name] = used
		out["surplus"][name] = dug - sold - used
	}
	return out
}
