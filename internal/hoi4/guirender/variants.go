package guirender

import (
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type equipmentVariant struct {
	DLC   dlcGuard
	Date  string
	Name  string
	Type  string
	Block *paradox.Block
}

func collectVariants(entries []*paradox.Entry, date string) []equipmentVariant {
	return collectVariantsGuarded(entries, date, dlcGuard{})
}

func collectVariantsGuarded(entries []*paradox.Entry, date string, guard dlcGuard) []equipmentVariant {
	out := []equipmentVariant{}
	branch := branchState{}
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		switch {
		case key == "if" || key == "else_if":
			if inner, ok := branch.enterIf(entry.Block.Entries, guard); ok {
				out = append(out, collectVariantsGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "else":
			if inner, ok := branch.enterElse(); ok {
				out = append(out, collectVariantsGuarded(entry.Block.Entries, date, inner)...)
			}
		case key == "create_equipment_variant":
			out = append(out, equipmentVariant{DLC: guard,
				Date:  date,
				Name:  strings.Trim(ciScalar(entry.Block.Entries, "name"), `"`),
				Type:  strings.ToLower(ciScalar(entry.Block.Entries, "type")),
				Block: entry.Block,
			})
		case isGameDate(key):
			out = append(out, collectVariantsGuarded(entry.Block.Entries, key, guard)...)
		}
	}
	return out
}

func (idx *Index) EquipmentVariantsOf(tag, date string) []equipmentVariant {
	politics := idx.countryPoliticsFor(tag)
	if politics == nil {
		return nil
	}
	if date == "" {
		date = "1936.1.1"
	}
	out := []equipmentVariant{}
	for _, variant := range politics.Variants {
		if dateReached(variant.Date, date) && variant.DLC.holds(idx.ownedDLC) {
			out = append(out, variant)
		}
	}
	return out
}

func (idx *Index) designerEquipmentTypes(tag, date string, template DivisionTemplate) []map[string]any {
	archetypes := []string{}
	for _, count := range template.allCounts() {
		unit := idx.subUnitFor(count.Type)
		if unit == nil {
			continue
		}
		for _, archetype := range unit.NeedOrder {
			if !containsFold(archetypes, archetype) {
				archetypes = append(archetypes, archetype)
			}
		}
	}
	designs := map[string]int{}
	for _, variant := range idx.EquipmentVariantsOf(tag, date) {
		designs[variant.Type]++
	}
	fielded := idx.FieldedEquipment(tag, date)
	out := []map[string]any{}
	for _, archetype := range archetypes {
		types := []*equipmentRecord{}
		for name := range fielded {
			if record := idx.equipmentFor(name); record != nil && !record.IsArchetype && strings.EqualFold(record.Archetype, archetype) {
				types = append(types, record)
			}
		}
		if len(types) == 0 {
			continue
		}
		sort.Slice(types, func(a, b int) bool {
			if types[a].Year != types[b].Year {
				return types[a].Year > types[b].Year
			}
			return types[a].Order > types[b].Order
		})
		rows := []map[string]any{}
		for _, record := range types {
			total := designs[strings.ToLower(record.Name)]
			if total == 0 {
				total = 1
			}
			rows = append(rows, map[string]any{
				"type":    record.Name,
				"label":   idx.localiseOr(record.Name, record.Name),
				"allowed": total,
				"total":   total,
			})
		}
		label := idx.localiseOr(archetype+"_short", idx.localiseOr(archetype, archetype))
		out = append(out, map[string]any{"archetype": archetype, "label": label, "types": rows})
	}
	return out
}
