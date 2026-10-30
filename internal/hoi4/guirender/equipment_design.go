package guirender

import (
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type equipmentModule struct {
	Name     string
	Category string

	AddTypes []string

	Gfx       string
	Add       map[string]float64
	Multiply  map[string]float64
	Average   map[string]float64
	Resources map[string]float64
}

type equipmentUpgrade struct {
	Name     string
	Add      map[string]float64
	Multiply map[string]float64

	MaxLevel int
}

func numberBlock(block *paradox.Block) map[string]float64 {
	out := map[string]float64{}
	if block == nil {
		return out
	}
	for _, entry := range block.Entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		if value, ok := toFloat(scalarValue(entry.Scalar)); ok {
			out[strings.ToLower(*entry.Key)] = value
		}
	}
	return out
}

func childBlock(entries []*paradox.Entry, key string) *paradox.Block {
	for _, entry := range entries {
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, key) {
			return entry.Block
		}
	}
	return nil
}

func (idx *Index) indexEquipmentModules(block *paradox.Block) {
	if idx.equipmentModules == nil {
		idx.equipmentModules = map[string]*equipmentModule{}
	}
	for _, entry := range block.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || strings.EqualFold(*entry.Key, "limit") {
			continue
		}
		types := []string{}
		for _, field := range entry.Block.Entries {
			if field == nil || field.Key == nil || !strings.EqualFold(*field.Key, "add_equipment_type") {
				continue
			}
			if field.Block != nil {
				for _, token := range blockTokens(field.Block.Entries) {
					types = append(types, strings.ToLower(token))
				}
			} else if token := strings.Trim(toString(scalarValue(field.Scalar)), `"`); token != "" {
				types = append(types, strings.ToLower(token))
			}
		}
		idx.equipmentModules[strings.ToLower(*entry.Key)] = &equipmentModule{
			Name:      *entry.Key,
			AddTypes:  types,
			Gfx:       strings.ToLower(strings.Trim(ciScalar(entry.Block.Entries, "gfx"), `"`)),
			Category:  strings.ToLower(ciScalar(entry.Block.Entries, "category")),
			Add:       numberBlock(childBlock(entry.Block.Entries, "add_stats")),
			Multiply:  numberBlock(childBlock(entry.Block.Entries, "multiply_stats")),
			Average:   numberBlock(childBlock(entry.Block.Entries, "add_average_stats")),
			Resources: numberBlock(childBlock(entry.Block.Entries, "build_cost_resources")),
		}
	}
}

var upgradeFactorKeys = map[string]bool{}

func init() {
	for _, key := range equipmentStatKeys {
		upgradeFactorKeys[key] = true
	}
}

func (idx *Index) indexEquipmentUpgrades(block *paradox.Block) {
	if idx.equipmentUpgrades == nil {
		idx.equipmentUpgrades = map[string]*equipmentUpgrade{}
	}
	for _, entry := range block.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		factors := map[string]float64{}
		for key, value := range numberBlock(entry.Block) {
			if upgradeFactorKeys[key] {
				factors[key] = value
			}
		}
		maxLevel, _ := toFloat(ciScalarValue(entry.Block.Entries, "max_level"))
		idx.equipmentUpgrades[strings.ToLower(*entry.Key)] = &equipmentUpgrade{
			Name:     *entry.Key,
			Add:      numberBlock(childBlock(entry.Block.Entries, "add_stats")),
			Multiply: factors,
			MaxLevel: int(maxLevel),
		}
	}
}

func (idx *Index) equipmentModuleFor(name string) *equipmentModule {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.equipmentModules[strings.ToLower(name)]; ok {
			return record
		}
	}
	return nil
}

func (idx *Index) moduleGfx(name string) string {
	if module := idx.equipmentModuleFor(name); module != nil {
		return module.Gfx
	}
	return ""
}

func (idx *Index) equipmentUpgradeFor(name string) *equipmentUpgrade {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.equipmentUpgrades[strings.ToLower(name)]; ok {
			return record
		}
	}
	return nil
}

type equipmentDesign struct {
	Name      string
	Type      string
	Obsolete  bool
	Year      int
	Order     int
	Stats     map[string]float64
	Resources map[string]float64
}

func (idx *Index) designStats(typeName string, modules map[string]string, upgrades map[string]float64, team string, year int) (map[string]float64, map[string]float64) {
	record := idx.equipmentFor(typeName)
	if record == nil {
		return nil, nil
	}
	_, base, resources := idx.equipmentAt(record.Archetype, year, map[string]bool{strings.ToLower(typeName): true})
	add := map[string]float64{}
	factor := map[string]float64{}
	averages := map[string][]float64{}
	for _, name := range modules {
		module := idx.equipmentModuleFor(name)
		if module == nil {
			continue
		}
		for key, value := range module.Add {
			add[key] += value
		}
		for key, value := range module.Multiply {
			factor[key] += value
		}
		for key, value := range module.Average {
			averages[key] = append(averages[key], value)
		}
		for key, value := range module.Resources {
			resources[key] += value
		}
	}
	for key, level := range upgrades {
		upgrade := idx.equipmentUpgradeFor(key)
		if upgrade == nil {
			continue
		}
		for stat, value := range upgrade.Add {
			add[stat] += value * level
		}
		for stat, value := range upgrade.Multiply {
			factor[stat] += value * level
		}
	}
	if team != "" {
		if org := idx.resolveIndustrialOrg(strings.TrimPrefix(team, "mio:")); org != nil && org.Initial != nil {
			for stat, value := range numberBlock(childBlock(org.Initial.Entries, "equipment_bonus")) {
				factor[stat] += value
			}
		}
	}
	for key, values := range averages {
		sum := 0.0
		for _, value := range values {
			sum += value
		}
		add[key] += sum / float64(len(values))
	}
	stats := map[string]float64{}
	for key, value := range base {
		stats[key] = value
	}
	for key, value := range add {
		stats[key] += value
	}
	for key, value := range factor {
		stats[key] *= 1 + value
	}
	return stats, resources
}

func (idx *Index) variantDesign(variant equipmentVariant, year int, order int) *equipmentDesign {
	modules := map[string]string{}
	if block := childBlock(variant.Block.Entries, "modules"); block != nil {
		for _, slot := range block.Entries {
			if slot != nil && slot.Key != nil && slot.Block == nil {
				modules[strings.ToLower(*slot.Key)] = strings.ToLower(strings.Trim(toString(scalarValue(slot.Scalar)), `"`))
			}
		}
	}
	upgrades := numberBlock(childBlock(variant.Block.Entries, "upgrades"))
	stats, resources := idx.designStats(variant.Type, modules, upgrades, ciScalar(variant.Block.Entries, "design_team"), year)
	if stats == nil {
		return nil
	}
	record := idx.equipmentFor(variant.Type)
	return &equipmentDesign{
		Name:      variant.Name,
		Type:      record.Name,
		Obsolete:  strings.EqualFold(ciScalar(variant.Block.Entries, "obsolete"), "yes"),
		Year:      record.Year,
		Order:     order,
		Stats:     stats,
		Resources: resources,
	}
}

func (idx *Index) defaultDesign(record *equipmentRecord, year int) *equipmentDesign {
	modules := idx.defaultModulesOf(record)
	if len(modules) == 0 {
		return nil
	}
	stats, resources := idx.designStats(record.Name, modules, nil, "", year)
	if stats == nil {
		return nil
	}
	return &equipmentDesign{Name: record.Name, Type: record.Name, Year: record.Year, Stats: stats, Resources: resources}
}

type designSet map[string]*archetypeDesigns

type archetypeDesigns struct {
	Chosen *equipmentDesign
	All    []*equipmentDesign
}

func (idx *Index) countryDesigns(tag, date string, year int) designSet {
	out := designSet{}
	put := func(design *equipmentDesign) {
		record := idx.equipmentFor(design.Type)
		if record == nil {
			return
		}
		archetype := strings.ToLower(record.Archetype)
		if out[archetype] == nil {
			out[archetype] = &archetypeDesigns{}
		}
		out[archetype].All = append(out[archetype].All, design)
	}
	for order, variant := range idx.EquipmentVariantsOf(tag, date) {
		if design := idx.variantDesign(variant, year, order+1); design != nil {
			put(design)
		}
	}
	for name := range idx.FieldedEquipment(tag, date) {
		record := idx.equipmentFor(name)
		if record == nil || record.IsArchetype {
			continue
		}
		archetype := idx.equipmentFor(record.Archetype)
		if record.DefaultModules == nil && (archetype == nil || archetype.DefaultModules == nil) {
			continue
		}
		if design := idx.defaultDesign(record, year); design != nil {
			put(design)
		}
	}
	better := func(a, b *equipmentDesign) bool {
		if a.Obsolete != b.Obsolete {
			return !a.Obsolete
		}
		if a.Year != b.Year {
			return a.Year > b.Year
		}
		return a.Order > b.Order
	}
	for _, set := range out {

		for _, drawn := range []bool{true, false} {
			for _, design := range set.All {
				if (design.Order > 0) == drawn && (set.Chosen == nil || better(design, set.Chosen)) {
					set.Chosen = design
				}
			}
			if set.Chosen != nil {
				break
			}
		}
	}
	return out
}
