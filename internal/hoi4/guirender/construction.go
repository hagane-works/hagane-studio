package guirender

import (
	"math"
	"strings"
)

func (idx *Index) ConstructionPayload(tag, date string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	civilian, military := idx.factoriesOwned(tag, date)
	sums := idx.startingModifiers(tag, date)
	consumer := idx.consumerGoodsFactories(civilian+military, sums["consumer_goods_expected_value"], sums["consumer_goods_factor"])
	projects := int(math.Round(sums["civilian_factory_use"]))
	rows := []map[string]any{}
	if consumer > 0 {
		rows = append(rows, map[string]any{"kind": "consumer_goods", "factories": consumer})
	}
	if projects > 0 {
		rows = append(rows, map[string]any{"kind": "national_projects", "factories": projects})
	}
	return map[string]any{
		"tag":      tag,
		"civilian": civilian,
		"military": military,
		"used":     consumer + projects,
		"consumer": consumer,
		"projects": projects,

		"from_trade": 0,
		"speed":      sums["production_speed_buildings_factor"],
		"rows":       rows,

		"energy": map[string]any{
			"resource": energyResource,
			"frame":    idx.ResourceFrames()[energyResource],
			"amount":   idx.resourceBalance(tag, date)["surplus"][energyResource],
		},
	}
}

func (idx *Index) factoriesOwned(tag, date string) (civilian, military int) {
	for _, state := range idx.effectiveStates() {
		if idx.ownerOfState(state, date) != tag {
			continue
		}
		buildings := state.buildingsAt(date)
		civilian += buildings["industrial_complex"]
		military += buildings["arms_factory"]
	}
	return civilian, military
}

func (idx *Index) consumerGoodsFactories(factories int, expected, factor float64) int {
	share := expected * (1 + factor)
	if share < 0 {
		share = 0
	}
	count := int(math.Floor(float64(factories)*share + 1e-9))
	floor := int(math.Floor(float64(factories)*idx.defineNumber("NProduction.MINIMUM_NUMBER_OF_FACTORIES_TAKEN_BY_CONSUMER_GOODS_PERCENT") + 1e-9))
	if least := int(idx.defineNumber("NProduction.MINIMUM_NUMBER_OF_FACTORIES_TAKEN_BY_CONSUMER_GOODS_VALUE")); least > floor {
		floor = least
	}
	if factories > 0 && count < floor {
		count = floor
	}
	return count
}
