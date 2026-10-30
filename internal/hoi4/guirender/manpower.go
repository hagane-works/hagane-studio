package guirender

import "math"

func (idx *Index) fieldedManpower(tag, date string) int {
	templates := map[string]DivisionTemplate{}
	for _, template := range idx.DivisionTemplatesOf(tag, date) {
		templates[template.Name] = template
	}
	total := 0.0
	for _, name := range idx.OOBOf(tag, date) {
		record, ok := idx.oobRecordFor(name)
		if !ok {
			continue
		}
		for templateName, count := range record.Deployed {
			template, known := templates[templateName]
			if !known {
				continue
			}
			for _, cell := range template.AllCells() {
				if unit := idx.subUnitFor(cell.Type); unit != nil {
					total += unit.Manpower * float64(count)
				}
			}
		}
		for _, wing := range record.AirWings {
			total += idx.equipmentManpower(wing.Type) * wing.Amount
		}
	}
	return int(math.Round(total))
}

func (idx *Index) equipmentManpower(model string) float64 {
	record := idx.equipmentFor(model)
	if record == nil {
		return 0
	}
	if crew := record.Stats["manpower"]; crew > 0 || record.Archetype == "" {
		return crew
	}
	if archetype := idx.equipmentFor(record.Archetype); archetype != nil {
		return archetype.Stats["manpower"]
	}
	return 0
}

func (idx *Index) dailyPoliticalPower(sums map[string]float64) float64 {
	return (idx.defineNumber("NPolitics.BASE_POLITICAL_POWER_INCREASE") + sums["political_power_gain"]) *
		(1 + sums["political_power_factor"])
}
