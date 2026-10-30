package guirender

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type subUnitRecord struct {
	Name string

	File         string
	Abbreviation string
	Sprite       string
	Group        string
	Categories   []string

	BattalionMults []battalionMult
	Types          []string
	Regimental     bool

	MapIcon      string
	AffectsSpeed bool

	Transport    string
	SpecialForce bool

	Priority float64

	Active bool

	ArmyHQ    bool
	NonArmyHQ bool

	Divisional bool

	Order int

	CombatWidth     float64
	MaxStrength     float64
	MaxOrganisation float64
	DefaultMorale   float64
	Manpower        float64
	TrainingTime    float64
	Weight          float64
	Supply          float64
	Suppression     float64
	Entrenchment    float64
	Recon           float64
	Initiative      float64

	Paradrop bool

	DeploymentCost  float64
	LeaderModifiers []leaderModifier
	Abilities       []string

	Terrain map[string]terrainMod

	Modifiers map[string]float64

	Need map[string]float64

	NeedEquipment  map[string]float64
	NeedModules    [][]string
	HitProfileMult float64

	NeedOrder []string
}

type equipmentRecord struct {
	Name        string
	Archetype   string
	Parent      string
	Year        int
	IsArchetype bool
	Priority    float64

	Group   string
	Picture string

	Types []string

	DefaultModules map[string]string

	ModuleSlots  []moduleSlot
	SlotsInherit bool
	Upgrades     []string

	DerivedVariantName string

	TypeOverride string
	AllowedTypes []string

	Copied    bool
	Stats     map[string]float64
	Resources map[string]float64

	Order int
}

var combatModifierKeys = map[string]struct{}{
	"soft_attack": {}, "hard_attack": {}, "air_attack": {},
	"defense": {}, "breakthrough": {}, "armor_value": {},
	"ap_attack": {}, "maximum_speed": {},
}

var equipmentStatKeys = []string{
	"maximum_speed", "hardness", "armor_value", "ap_attack",
	"soft_attack", "hard_attack", "defense", "breakthrough",
	"air_attack", "reliability", "build_cost_ic", "lend_lease_cost",
	"fuel_capacity", "fuel_consumption",

	"air_range", "weight", "thrust", "air_defence", "air_agility",
	"air_superiority", "naval_strike_attack", "naval_strike_targetting",
	"air_ground_attack", "air_bombing", "surface_detection", "sub_detection",
	"night_penalty", "mines_planting", "mines_sweeping",

	"lg_attack", "lg_armor_piercing", "hg_attack", "hg_armor_piercing",
	"torpedo_attack", "sub_attack", "anti_air_attack", "surface_visibility",
	"sub_visibility", "naval_speed", "naval_range", "max_strength", "manpower",
	"naval_dominance_factor", "naval_weather_penalty_factor", "carrier_size",
}

func (idx *Index) indexSubUnitFile(path string, body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "sub_units") {
			continue
		}
		for _, unit := range entry.Block.Entries {
			if unit == nil || unit.Key == nil || unit.Block == nil {
				continue
			}
			record := &subUnitRecord{
				Name:         *unit.Key,
				File:         path,
				Abbreviation: strings.Trim(ciScalar(unit.Block.Entries, "abbreviation"), "\""),
				Sprite:       strings.Trim(ciScalar(unit.Block.Entries, "sprite"), "\""),
				Group:        strings.Trim(ciScalar(unit.Block.Entries, "group"), "\""),
				MapIcon:      strings.ToLower(strings.Trim(ciScalar(unit.Block.Entries, "map_icon_category"), "\"")),

				Regimental:    !strings.EqualFold(ciScalar(unit.Block.Entries, "regimental"), "no"),
				AffectsSpeed:  !strings.EqualFold(ciScalar(unit.Block.Entries, "affects_speed"), "no"),
				Transport:     strings.ToLower(strings.Trim(ciScalar(unit.Block.Entries, "transport"), `"`)),
				Active:        !strings.EqualFold(ciScalar(unit.Block.Entries, "active"), "no"),
				ArmyHQ:        strings.EqualFold(ciScalar(unit.Block.Entries, "allow_in_army_hq"), "yes"),
				NonArmyHQ:     !strings.EqualFold(ciScalar(unit.Block.Entries, "allow_in_non_army_hq"), "no"),
				Divisional:    !strings.EqualFold(ciScalar(unit.Block.Entries, "divisional"), "no"),
				Paradrop:      strings.EqualFold(ciScalar(unit.Block.Entries, "can_be_parachuted"), "yes"),
				Need:          map[string]float64{},
				NeedEquipment: map[string]float64{},
				Modifiers:     map[string]float64{},
			}
			for _, field := range unit.Block.Entries {
				if field == nil || field.Key == nil {
					continue
				}
				key := strings.ToLower(*field.Key)
				if field.Block != nil {
					switch key {
					case "battalion_mult":
						mult := battalionMult{
							Category: strings.ToLower(ciScalar(field.Block.Entries, "category")),
							Add:      strings.EqualFold(ciScalar(field.Block.Entries, "add"), "yes"),
							Stats:    numberBlock(field.Block),
						}
						if mult.Category != "" && len(mult.Stats) > 0 {
							record.BattalionMults = append(record.BattalionMults, mult)
						}
					case "categories":
						record.Categories = blockTokens(field.Block.Entries)
						for _, category := range record.Categories {
							if strings.EqualFold(category, "category_special_forces") {
								record.SpecialForce = true
							}
						}
					case "type":
						record.Types = blockTokens(field.Block.Entries)
					case "deployed_leader_modifiers":
						for _, item := range field.Block.Entries {
							if item == nil || item.Key == nil || item.Block != nil {
								continue
							}
							if amount, ok := toFloat(scalarValue(item.Scalar)); ok {
								record.LeaderModifiers = append(record.LeaderModifiers, leaderModifier{strings.ToLower(*item.Key), amount})
							}
						}
					case "enable_ability":
						record.Abilities = append(record.Abilities, blockTokens(field.Block.Entries)...)
					case "forest", "hills", "mountain", "plains", "urban", "jungle", "marsh", "desert", "amphibious", "river", "fort":
						mod := terrainMod{}
						for _, item := range field.Block.Entries {
							if item == nil || item.Key == nil || item.Block != nil {
								continue
							}
							amount, ok := toFloat(scalarValue(item.Scalar))
							if !ok {
								continue
							}
							switch strings.ToLower(*item.Key) {
							case "attack":
								mod.Attack = amount
							case "defence", "defense":
								mod.Defence = amount
							case "movement":
								mod.Movement = amount
							}
						}
						if record.Terrain == nil {
							record.Terrain = map[string]terrainMod{}
						}
						record.Terrain[key] = mod
					case "need_equipment":
						for key, amount := range numberBlock(field.Block) {
							record.NeedEquipment[key] = amount
						}
					case "need_equipment_modules":
						for _, item := range field.Block.Entries {
							if item == nil || item.Key == nil {
								continue
							}
							if item.Block != nil {
								group := []string{}
								for module := range numberBlock(item.Block) {
									group = append(group, module)
								}
								sort.Strings(group)
								record.NeedModules = append(record.NeedModules, group)
							} else {
								record.NeedModules = append(record.NeedModules, []string{strings.ToLower(*item.Key)})
							}
						}
					case "need":
						for _, item := range field.Block.Entries {
							if item == nil || item.Key == nil || item.Block != nil {
								continue
							}
							if amount, ok := toFloat(scalarValue(item.Scalar)); ok {
								key := strings.ToLower(*item.Key)
								if _, seen := record.Need[key]; !seen {
									record.NeedOrder = append(record.NeedOrder, key)
								}
								record.Need[key] = amount
							}
						}
					}
					continue
				}
				value, ok := toFloat(scalarValue(field.Scalar))
				if !ok {
					continue
				}
				switch key {
				case "priority":
					record.Priority = value
				case "combat_width":
					record.CombatWidth = value
				case "max_strength":
					record.MaxStrength = value
				case "max_organisation":
					record.MaxOrganisation = value
				case "default_morale":
					record.DefaultMorale = value
				case "manpower":
					record.Manpower = value
				case "training_time":
					record.TrainingTime = value
				case "weight":
					record.Weight = value
				case "supply_consumption":
					record.Supply = value
				case "suppression":
					record.Suppression = value
				case "entrenchment":
					record.Entrenchment = value
				case "deployment_cost":
					record.DeploymentCost = value
				case "recon":
					record.Recon = value
				case "initiative":
					record.Initiative = value
				case "hit_profile_mult":
					record.HitProfileMult = value
				default:
					if _, ok := combatModifierKeys[key]; ok {
						record.Modifiers[key] = value
					}
				}
			}
			idx.subUnitSeq++
			record.Order = idx.subUnitSeq
			idx.subUnits[strings.ToLower(record.Name)] = record
		}
	}
}

func (idx *Index) indexEquipmentFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	for _, entry := range root.Entries {
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "duplicate_archetypes") {
			idx.indexDuplicateArchetypes(entry.Block)
			continue
		}
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "equipment_modules") {
			idx.indexEquipmentModules(entry.Block)
			continue
		}
		if entry != nil && entry.Key != nil && entry.Block != nil && strings.EqualFold(*entry.Key, "upgrades") {
			idx.indexEquipmentUpgrades(entry.Block)
			continue
		}
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "equipments") {
			continue
		}
		for _, item := range entry.Block.Entries {
			if item == nil || item.Key == nil || item.Block == nil {
				continue
			}
			record := &equipmentRecord{
				Name:               *item.Key,
				Archetype:          strings.Trim(ciScalar(item.Block.Entries, "archetype"), "\""),
				Parent:             strings.Trim(ciScalar(item.Block.Entries, "parent"), "\""),
				IsArchetype:        strings.EqualFold(ciScalar(item.Block.Entries, "is_archetype"), "yes"),
				Group:              strings.ToLower(strings.Trim(ciScalar(item.Block.Entries, "interface_category"), "\"")),
				Picture:            strings.Trim(ciScalar(item.Block.Entries, "picture"), "\""),
				DerivedVariantName: strings.Trim(ciScalar(item.Block.Entries, "derived_variant_name"), "\""),
				TypeOverride:       strings.ToLower(strings.Trim(ciScalar(item.Block.Entries, "type_override"), "\"")),
				Stats:              map[string]float64{},
				Resources:          map[string]float64{},
			}
			idx.equipmentSeq++
			record.Order = idx.equipmentSeq
			if year, ok := toFloat(ciScalarValue(item.Block.Entries, "year")); ok {
				record.Year = int(year)
			}
			if priority, ok := toFloat(ciScalarValue(item.Block.Entries, "priority")); ok {
				record.Priority = priority
			}
			for _, key := range equipmentStatKeys {
				if value, ok := toFloat(ciScalarValue(item.Block.Entries, key)); ok {
					record.Stats[key] = value
				}
			}
			for _, field := range item.Block.Entries {
				if field == nil || field.Key == nil {
					continue
				}
				if strings.EqualFold(*field.Key, "type") {
					if field.Block != nil {
						for _, token := range blockTokens(field.Block.Entries) {
							record.Types = append(record.Types, strings.ToLower(token))
						}
					} else if token := strings.Trim(toString(scalarValue(field.Scalar)), "\""); token != "" {
						record.Types = []string{strings.ToLower(token)}
					}
					continue
				}
				if strings.EqualFold(*field.Key, "module_slots") {
					if field.Block == nil {
						record.SlotsInherit = strings.EqualFold(strings.Trim(toString(scalarValue(field.Scalar)), `"`), "inherit")
						continue
					}
					for _, slot := range field.Block.Entries {
						if slot == nil || slot.Key == nil {
							continue
						}

						if slot.Block == nil {
							record.ModuleSlots = append(record.ModuleSlots, moduleSlot{
								Name: strings.ToLower(*slot.Key),
								Ref:  strings.ToLower(strings.Trim(toString(scalarValue(slot.Scalar)), `"`)),
							})
							continue
						}
						entry := moduleSlot{
							Name:     strings.ToLower(*slot.Key),
							Required: strings.EqualFold(ciScalar(slot.Block.Entries, "required"), "yes"),
						}
						if allowed := childBlock(slot.Block.Entries, "allowed_module_categories"); allowed != nil {
							for _, category := range blockTokens(allowed.Entries) {
								entry.Categories = append(entry.Categories, strings.ToLower(category))
							}
						}
						record.ModuleSlots = append(record.ModuleSlots, entry)
					}
					continue
				}
				if field.Block != nil && strings.EqualFold(*field.Key, "allowed_types") {
					for _, token := range blockTokens(field.Block.Entries) {
						record.AllowedTypes = append(record.AllowedTypes, strings.ToLower(token))
					}
					continue
				}
				if field.Block != nil && strings.EqualFold(*field.Key, "upgrades") {
					for _, upgrade := range blockTokens(field.Block.Entries) {
						record.Upgrades = append(record.Upgrades, strings.ToLower(upgrade))
					}
					continue
				}
				if field.Block != nil && strings.EqualFold(*field.Key, "default_modules") {
					record.DefaultModules = map[string]string{}
					for _, slot := range field.Block.Entries {
						if slot != nil && slot.Key != nil && slot.Block == nil {
							record.DefaultModules[strings.ToLower(*slot.Key)] = strings.ToLower(strings.Trim(toString(scalarValue(slot.Scalar)), `"`))
						}
					}
					continue
				}
				if field.Block == nil || !strings.EqualFold(*field.Key, "resources") {
					continue
				}
				for _, resource := range field.Block.Entries {
					if resource == nil || resource.Key == nil || resource.Block != nil {
						continue
					}
					if amount, ok := toFloat(scalarValue(resource.Scalar)); ok {
						record.Resources[strings.ToLower(*resource.Key)] = amount
					}
				}
			}
			idx.equipment[strings.ToLower(record.Name)] = record
		}
	}
}

func blockTokens(entries []*paradox.Entry) []string {
	out := []string{}
	for _, entry := range entries {
		if entry == nil || entry.Block != nil {
			continue
		}
		if entry.Key != nil {
			out = append(out, *entry.Key)
			continue
		}
		if token := toString(scalarValue(entry.Scalar)); token != "" {
			out = append(out, strings.Trim(token, "\""))
		}
	}
	return out
}

func (idx *Index) subUnitFor(name string) *subUnitRecord {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.subUnits[strings.ToLower(name)]; ok {
			return record
		}
	}
	return nil
}

func (idx *Index) equipmentFor(name string) *equipmentRecord {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.equipment[strings.ToLower(name)]; ok {
			return record
		}
	}
	return nil
}

func (idx *Index) equipmentAt(archetype string, year int, fielded map[string]bool) (string, map[string]float64, map[string]float64) {
	base := idx.equipmentFor(archetype)
	if base == nil {
		return "", nil, nil
	}

	better := func(candidate, current *equipmentRecord) bool {
		if current.IsArchetype {
			return true
		}
		if candidate.Year != current.Year {
			return candidate.Year > current.Year
		}
		if candidate.Priority != current.Priority {
			return candidate.Priority > current.Priority
		}

		if strings.EqualFold(candidate.Parent, current.Name) {
			return true
		}
		if strings.EqualFold(current.Parent, candidate.Name) {
			return false
		}
		return candidate.Order > current.Order
	}
	best := base
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, record := range layer.equipment {
			if record.IsArchetype || !strings.EqualFold(record.Archetype, archetype) {
				continue
			}

			if len(fielded) > 0 {
				if !fielded[strings.ToLower(record.Name)] {
					continue
				}
			} else if record.Year > year {
				continue
			}
			if better(record, best) {
				best = record
			}
		}
	}
	stats := map[string]float64{}
	for key, value := range base.Stats {
		stats[key] = value
	}

	chain := []*equipmentRecord{}
	if best != base {
		chain = append(chain, best)
	}
	resources := map[string]float64{}
	for key, value := range base.Resources {
		resources[key] = value
	}
	for _, record := range chain {
		for key, value := range record.Stats {
			stats[key] = value
		}
		for key, value := range record.Resources {
			resources[key] = value
		}
	}
	return best.Name, stats, resources
}

type battalionMult struct {
	Category string
	Add      bool
	Stats    map[string]float64
}

type DivisionStats map[string]float64

func (idx *Index) divisionStats(template DivisionTemplate, year int, researched map[string]map[string]float64, fielded map[string]bool, designs ...designSet) (DivisionStats, map[string]float64, map[string]float64) {
	var drawn designSet
	if len(designs) > 0 {
		drawn = designs[0]
	}
	stats := DivisionStats{}
	need := map[string]float64{}
	resources := map[string]float64{}

	battalions := 0.0
	organisation, morale := 0.0, 0.0
	hardness, hardnessHP, strength, widthOfAll := 0.0, 0.0, 0.0, 0.0
	reliability, reliabilityOf := 0.0, 0.0
	speed := math.Inf(1)
	armour, piercing := []float64{}, []float64{}

	all := template.allCounts()

	mults := []battalionMult{}
	for _, count := range all {
		if unit := idx.subUnitFor(count.Type); unit != nil {
			mults = append(mults, unit.BattalionMults...)
		}
	}
	for _, count := range all {
		unit := idx.subUnitFor(count.Type)
		if unit == nil {
			continue
		}
		copies := float64(count.Count)

		bonus := map[string]float64{}
		for _, target := range append([]string{unit.Name}, unit.Categories...) {
			for stat, value := range researched[strings.ToLower(target)] {
				bonus[stat] += value
			}
		}
		times := map[string]float64{}
		for _, mult := range mults {
			if !containsFold(unit.Categories, mult.Category) {
				continue
			}
			for stat, value := range mult.Stats {
				if mult.Add {
					bonus[stat] += value
				} else if times[stat] == 0 {
					times[stat] = 1 + value
				} else {
					times[stat] *= 1 + value
				}
			}
		}
		organisationOf := unit.MaxOrganisation + bonus["max_organisation"]
		strengthOf := unit.MaxStrength + bonus["max_strength"]

		battalions += copies
		organisation += organisationOf * copies
		morale += (unit.DefaultMorale + bonus["default_morale"]) * copies
		strength += strengthOf * copies

		stats["combat_width"] += (unit.CombatWidth + bonus["combat_width"]) * copies
		stats["max_strength"] += strengthOf * copies
		stats["manpower"] += unit.Manpower * copies
		stats["supply_use"] += (unit.Supply + bonus["supply_consumption"]) * copies
		stats["weight"] += unit.Weight * copies
		stats["suppression"] += (unit.Suppression + bonus["suppression"]) * copies
		stats["entrenchment"] += (unit.Entrenchment + bonus["entrenchment"]) * copies
		stats["initiative"] += unit.Initiative * copies
		if unit.Recon > stats["recon"] {
			stats["recon"] = unit.Recon
		}
		if unit.TrainingTime > stats["training_time"] {
			stats["training_time"] = unit.TrainingTime
		}
		scale := func(key string) float64 {
			factor := 1 + unit.Modifiers[key] + bonus[key]
			if times[key] != 0 {
				factor *= times[key]
			}
			return factor
		}
		carried := 0.0

		armourOf, piercingOf, hardnessOf, reliableOf, kinds := 0.0, 0.0, 0.0, 0.0, 0.0

		together := unit.ArmyHQ && !unit.NonArmyHQ && unit.Transport == ""
		summed := 0.0

		for archetype, amount := range unit.Need {
			need[archetype] += amount * copies
			name, values, costs := idx.equipmentAt(archetype, year, fielded)

			if set := drawn[strings.ToLower(archetype)]; set != nil && set.Chosen != nil {
				name, values, costs = set.Chosen.Type, set.Chosen.Stats, set.Chosen.Resources
			}
			if name == "" {
				continue
			}
			for _, key := range []string{"soft_attack", "hard_attack", "defense", "breakthrough", "air_attack"} {
				stats[key] += values[key] * scale(key) * copies
			}
			armourOf += values["armor_value"] * scale("armor_value")
			piercingOf += values["ap_attack"] * scale("ap_attack")

			capacity := values["fuel_capacity"]
			if capacity == 0 {
				capacity = values["fuel_consumption"] * idx.defineNumber("NMilitary.ARMY_FUEL_COST_MULT") *
					idx.defineNumber("NMilitary.FUEL_CAPACITY_DEFAULT_HOURS")
			}
			stats["fuel_capacity"] += capacity * copies
			stats["fuel_use"] += values["fuel_consumption"] * copies
			hardnessOf += values["hardness"]
			reliableOf += values["reliability"]
			kinds++

			if moving := values["maximum_speed"] * scale("maximum_speed"); moving > carried {
				carried = moving
			}
			summed += values["maximum_speed"] * scale("maximum_speed")
			stats["production_cost"] += values["build_cost_ic"] * amount * copies
			for resource, quantity := range costs {
				resources[resource] += quantity * amount * copies / 100
			}
		}

		if together {
			carried = summed
		}
		if kinds > 0 {
			for range int(copies) {
				armour = append(armour, armourOf)
				piercing = append(piercing, piercingOf)
			}
			widthOf := unit.CombatWidth + bonus["combat_width"]
			hardness += hardnessOf * widthOf * copies
			widthOfAll += widthOf * copies
			hardnessHP += hardnessOf * strengthOf * copies
			reliability += reliableOf / kinds * copies
			reliabilityOf += copies
		}
		if unit.AffectsSpeed && carried > 0 && carried < speed {
			speed = carried
		}
	}
	if battalions > 0 {
		stats["max_organisation"] = round(organisation/battalions, 1)
		stats["recovery_rate"] = round(morale/battalions, 2)
	}

	if widthOfAll > 0 {
		stats["hardness"] = thousandths(hardness / widthOfAll)
	} else if strength > 0 {
		stats["hardness"] = thousandths(hardnessHP / strength)
	}
	if reliabilityOf > 0 {
		stats["reliability"] = thousandths(reliability / reliabilityOf)
	}
	if !math.IsInf(speed, 1) {
		stats["speed"] = speed
	}
	stats["armor_value"] = blendOf(armour, 0.4)
	stats["ap_attack"] = blendOf(piercing, 0.4)

	for key, value := range stats {
		stats[key] = thousandths(value)
	}
	return stats, need, resources
}

func blendOf(values []float64, best float64) float64 {
	if len(values) == 0 {
		return 0
	}
	top, sum := 0.0, 0.0
	for _, value := range values {
		top = math.Max(top, value)
		sum += value
	}
	mean := thousandths(sum / float64(len(values)))
	return thousandths(best*top) + thousandths((1-best)*mean)
}

func thousandths(value float64) float64 {
	if value < 0 {
		return -thousandths(-value)
	}
	return math.Floor(value*1000+1e-6) / 1000
}

func round(value float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
}

func yearOf(date string) int {
	head := strings.SplitN(strings.TrimSpace(date), ".", 2)[0]
	if year, err := strconv.Atoi(head); err == nil && year > 0 {
		return year
	}
	return 1936
}

func (idx *Index) DesignerPayload(tag, date, name string) map[string]any {
	return idx.DesignerPayloadFor(tag, date, name, "")
}

func (idx *Index) DesignerPayloadFor(tag, date, name, cells string) map[string]any {
	templates := idx.DivisionTemplatesOf(tag, date)
	chosen := DivisionTemplate{Priority: defaultTemplatePriority}
	fresh := name == "-"
	if fresh && cells == "" {
		cells = "-"
	}
	found := false
	for _, template := range templates {
		if fresh {
			break
		}
		if name == "" || strings.EqualFold(template.Name, name) {
			chosen, found = template, true
			break
		}
	}

	if !fresh && !found && name != "" {
		for _, template := range idx.ArmyHQTemplatesOf(tag, date) {
			if strings.EqualFold(template.Name, name) {
				chosen = template
				templates = idx.ArmyHQTemplatesOf(tag, date)
				break
			}
		}
	}
	if cells != "" {
		chosen = withDesignerCells(chosen, cells)
		if fresh {
			chosen.Name = idx.localiseOr("DESIGNER_NEW_DIVISION", "New Division")
		}
	}
	year := yearOf(date)
	fielded := idx.FieldedEquipment(tag, date)
	modifiers := idx.ResearchedModifiers(tag, date)
	designs := idx.countryDesigns(tag, date, year)
	stats, need, resources := idx.divisionStats(chosen, year, modifiers, fielded, designs)

	var header map[string]float64
	if len(templateUnits(chosen)) == 0 && len(templates) > 0 {
		header, _, _ = idx.divisionStats(templates[0], year, modifiers, fielded, designs)
	}

	equipment := []map[string]any{}
	for _, archetype := range idx.inEquipmentOrder(sortedKeys(need)) {
		item := map[string]any{"archetype": archetype, "amount": round(need[archetype], 0)}

		if short, ok := idx.Localise(archetype + "_short"); ok {
			item["short_label"] = short
		} else if name, ok := idx.Localise(archetype); ok {
			item["short_label"] = name
		}
		if set := designs[strings.ToLower(archetype)]; set != nil && set.Chosen != nil {
			item["model"] = set.Chosen.Type
			item["stats"] = set.Chosen.Stats
			item["label"] = idx.localiseOr(set.Chosen.Name, set.Chosen.Name)
		} else if model, values, _ := idx.equipmentAt(archetype, year, fielded); model != "" {
			item["model"] = model
			item["stats"] = values
			if label, ok := idx.Localise(model); ok {
				item["label"] = label
			}
		}
		if _, ok := item["label"]; !ok {
			if label, ok := idx.Localise(archetype); ok {
				item["label"] = label
			}
		}
		equipment = append(equipment, item)
	}

	frames := idx.ResourceFrames()
	cost := []map[string]any{}
	for _, resource := range sortedKeys(resources) {
		item := map[string]any{"resource": resource, "amount": round(resources[resource], 1)}
		if frame, ok := frames[resource]; ok {
			item["frame"] = frame
		}
		cost = append(cost, item)
	}
	sort.SliceStable(cost, func(a, b int) bool {
		return cost[a]["amount"].(float64) > cost[b]["amount"].(float64)
	})
	cheapest, dearest := idx.productionRange(need, year, fielded, designs)

	payload := map[string]any{
		"tag":                strings.ToUpper(tag),
		"name":               chosen.Name,
		"templates":          templateNames(templates),
		"regiments":          idx.cellPayload(chosen.RegimentCells),
		"support":            idx.cellPayload(chosen.SupportCells),
		"regimental_support": idx.cellPayload(chosen.RegimentalSupportCells),
		"stats":              stats,
		"equipment":          equipment,
		"resources":          cost,

		"production_cost": map[string]float64{"min": round(cheapest, 3), "max": round(dearest, 3)},
		"year":            year,
		"is_special":      specialForces(idx, chosen),

		"icon":                 idx.templateIcon(chosen),
		"names_group":          idx.divisionNamesGroup(chosen.NamesGroup),
		"division_names_group": chosen.NamesGroup,
		"priority":             chosen.Priority,

		"new":          fresh,
		"header_stats": header,

		"is_army_hq": chosen.IsArmyHQ,

		"terrain": idx.designerTerrain(chosen),

		"equipment_types": idx.designerEquipmentTypes(tag, date, chosen),

		"template_icons": templateIcons(idx, templates),

		"names_groups": idx.NamesGroupsFor(tag, date),
		"available":    idx.designerAvailable(tag, date, chosen.IsArmyHQ),
		"group_titles": idx.designerGroupTitles(),

		"army_experience": idx.designerExperience(tag, date),
		"costs": map[string]float64{
			"group":   idx.defineNumber("NMilitary.BASE_DIVISION_BRIGADE_GROUP_COST"),
			"change":  idx.defineNumber("NMilitary.BASE_DIVISION_BRIGADE_CHANGE_COST"),
			"support": idx.defineNumber("NMilitary.BASE_DIVISION_SUPPORT_SLOT_COST"),
		},
	}
	if chosen.IsArmyHQ {
		power, days := idx.armyHQDeployment(chosen)
		payload["commander_bonuses"] = idx.commanderBonuses(chosen)
		payload["command_power"] = power
		payload["deploy_days"] = days
	}
	return payload
}

func (idx *Index) subUnitIcon(unit *subUnitRecord) string {
	if unit == nil {
		return ""
	}
	byName := "GFX_unit_" + unit.Name + "_icon_medium"
	if _, ok := idx.spriteRecordFor(byName); ok {
		return byName
	}
	if unit.Sprite != "" {
		bySprite := "GFX_unit_" + unit.Sprite + "_icon_medium"
		if _, ok := idx.spriteRecordFor(bySprite); ok {
			return bySprite
		}
	}
	return byName
}

func templateNames(templates []DivisionTemplate) []string {
	out := []string{}
	for _, template := range templates {
		out = append(out, template.Name)
	}
	return out
}

func specialForces(idx *Index, template DivisionTemplate) bool {
	for _, count := range template.Regiments {
		if unit := idx.subUnitFor(count.Type); unit != nil && unit.SpecialForce {
			return true
		}
	}
	return false
}

func (idx *Index) cellPayload(cells []RegimentCell) []map[string]any {
	out := []map[string]any{}
	for _, cell := range cells {
		unit := idx.subUnitFor(cell.Type)
		abbreviation := ""
		if unit != nil {
			abbreviation = unit.Abbreviation
		}
		label, _ := idx.Localise(cell.Type)
		item := map[string]any{
			"type":         cell.Type,
			"label":        label,
			"abbreviation": abbreviation,
			"x":            cell.X,
			"y":            cell.Y,
		}
		if icon := idx.subUnitIcon(unit); icon != "" {
			item["icon"] = icon
		}
		out = append(out, item)
	}
	return out
}

func sortedKeys(values map[string]float64) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (idx *Index) variantsOf(archetype string, fielded map[string]bool) []*equipmentRecord {
	seen := map[string]bool{}
	out := []*equipmentRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.equipment {
			if seen[key] || record.IsArchetype || !strings.EqualFold(record.Archetype, archetype) {
				continue
			}
			if len(fielded) > 0 && !fielded[key] {
				continue
			}
			seen[key] = true
			out = append(out, record)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Year != out[b].Year {
			return out[a].Year < out[b].Year
		}
		return out[a].Name < out[b].Name
	})
	return out
}

func templateUnits(template DivisionTemplate) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, count := range template.allCounts() {
		key := strings.ToLower(count.Type)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, count.Type)
	}
	return out
}

var productionTabs = map[string]string{
	"interface_category_land":          "infantry",
	"interface_category_armor":         "armor",
	"interface_category_air":           "air",
	"interface_category_capital_ships": "capital_ships",
	"interface_category_screen_ships":  "screen_ships",
	"interface_category_other_ships":   "other_ships",
}

func (idx *Index) ProductionPayload(tag, date string) map[string]any {
	fielded := idx.FieldedEquipment(tag, date)
	unlockedBy := idx.enablingTech()
	year := yearOf(date)
	seen := map[string]bool{}
	items := []map[string]any{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.equipment {
			if seen[key] || record.IsArchetype || record.Archetype == "" || record.Copied {
				continue
			}
			if len(fielded) > 0 && !fielded[key] {
				continue
			}
			seen[key] = true
			archetype := idx.equipmentFor(record.Archetype)
			if archetype == nil {
				continue
			}
			_, stats, resources := idx.equipmentAt(record.Archetype, year, map[string]bool{key: true})

			kind := designerKind(archetype)
			if kind != "" && len(idx.slotsOf(record)) > 0 {
				if design, designResources := idx.designStats(record.Name, idx.defaultModulesOf(record), nil, "", year); design != nil {
					stats, resources = design, designResources
				}
			} else {
				kind = ""
			}
			item := map[string]any{
				"name":      record.Name,
				"archetype": record.Archetype,
				"category":  archetype.Group,
				"group":     productionTabs[archetype.Group],
				"year":      record.Year,
				"cost":      round(stats["build_cost_ic"], 2),
				"picture":   archetype.Picture,
				"icon":      idx.designIcon(tag, record, archetype, unlockedBy),
				"resources": resources,
			}
			if kind != "" {
				item["designer"] = kind
			}

			general, hasGeneral := idx.Localise(record.Name)
			if label, ok := idx.Localise(strings.ToUpper(tag) + "_" + record.Name); ok {
				item["label"] = label
				if hasGeneral && general != label {
					item["general_name"] = general
				}
			} else if hasGeneral {
				item["label"] = general
			}
			items = append(items, item)
		}
	}

	successors := map[string]bool{}
	for _, item := range items {
		if record := idx.equipmentFor(item["name"].(string)); record != nil && record.Parent != "" {
			successors[strings.ToLower(record.Parent)] = true
		}
	}
	for _, item := range items {
		if successors[strings.ToLower(item["name"].(string))] {
			item["obsolete"] = true
		}
	}

	designs := map[string]equipmentVariant{}
	names := []string{}
	for _, variant := range idx.EquipmentVariantsOf(tag, date) {
		key := variant.Name + "|" + strings.ToLower(variant.Type)
		if _, seen := designs[key]; !seen {
			names = append(names, key)
		}
		designs[key] = variant
	}
	for at, key := range names {
		variant := designs[key]
		name := variant.Name
		record := idx.equipmentFor(variant.Type)
		if record == nil {
			continue
		}
		archetype := idx.equipmentFor(record.Archetype)
		design := idx.variantDesign(variant, year, at)
		if archetype == nil || design == nil {
			continue
		}
		icon := strings.Trim(ciScalar(variant.Block.Entries, "icon"), `"`)
		if _, ok := idx.spriteRecordFor(icon); icon == "" || !ok {
			icon = idx.designIcon(tag, record, archetype, unlockedBy)
		}
		item := map[string]any{
			"name":         name,
			"label":        name,
			"general_name": idx.localiseOr(record.Name, record.Name),
			"design":       true,
			"type":         record.Name,
			"obsolete":     design.Obsolete,
			"archetype":    record.Archetype,
			"category":     archetype.Group,
			"group":        productionTabs[archetype.Group],
			"year":         record.Year,
			"order":        at,
			"cost":         round(design.Stats["build_cost_ic"], 2),
			"picture":      archetype.Picture,
			"icon":         icon,
			"resources":    design.Resources,
		}
		if kind := designerKind(archetype); kind != "" && len(idx.slotsOf(record)) > 0 {
			item["designer"] = kind
		}
		items = append(items, item)
	}

	order := map[string]int{
		"infantry": 0, "armor": 1, "air": 2,
		"capital_ships": 3, "screen_ships": 4, "other_ships": 5,
	}
	sort.Slice(items, func(a, b int) bool {
		ga, _ := items[a]["group"].(string)
		gb, _ := items[b]["group"].(string)
		if order[ga] != order[gb] {
			return order[ga] < order[gb]
		}
		if ga != gb {
			return ga < gb
		}

		da, _ := items[a]["design"].(bool)
		db, _ := items[b]["design"].(bool)
		if da != db {
			return da
		}
		if da {
			oa, _ := items[a]["order"].(int)
			ob, _ := items[b]["order"].(int)
			return oa < ob
		}
		ya, _ := items[a]["year"].(int)
		yb, _ := items[b]["year"].(int)
		if ya != yb {
			return ya < yb
		}
		na, _ := items[a]["name"].(string)
		nb, _ := items[b]["name"].(string)
		return na < nb
	})
	return map[string]any{
		"tag": strings.ToUpper(tag), "equipment": items,
		"resource_frames": idx.ResourceFrames(),

		"queue": idx.productionQueue(tag, date, items),
	}
}

func (idx *Index) enablingTech() map[string]string {
	out := map[string]string{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, record := range layer.technologies {
			for _, model := range record.Enables {
				key := strings.ToLower(model)
				if _, seen := out[key]; !seen {
					out[key] = record.Name
				}
			}
		}
	}
	return out
}

func (idx *Index) designIcon(tag string, record, archetype *equipmentRecord, unlockedBy map[string]string) string {
	if designerKind(archetype) == "plane" && record.DerivedVariantName != "" {
		if tech := unlockedBy[strings.ToLower(record.DerivedVariantName)]; tech != "" {
			for _, candidate := range []string{"GFX_" + strings.ToUpper(tag) + "_" + tech + "_medium", "GFX_" + tech + "_medium"} {
				if _, ok := idx.spriteRecordFor(candidate); ok {
					return candidate
				}
			}
		}
	}
	return idx.equipmentIcon(tag, record.Name, record.Archetype, archetype.Picture, unlockedBy[strings.ToLower(record.Name)])
}

func (idx *Index) defaultModulesOf(record *equipmentRecord) map[string]string {
	layers := []map[string]string{record.DefaultModules}
	seen := map[string]bool{strings.ToLower(record.Name): true}
	for at := idx.equipmentFor(record.Parent); at != nil && !seen[strings.ToLower(at.Name)]; at = idx.equipmentFor(at.Parent) {
		seen[strings.ToLower(at.Name)] = true
		layers = append(layers, at.DefaultModules)
		if at.Parent == "" {
			break
		}
	}
	if archetype := idx.equipmentFor(record.Archetype); archetype != nil && !seen[strings.ToLower(archetype.Name)] {
		layers = append(layers, archetype.DefaultModules)
	}
	merged := map[string]string{}
	for at := len(layers) - 1; at >= 0; at-- {
		for slot, module := range layers[at] {
			merged[slot] = module
		}
	}
	out := map[string]string{}
	for _, slot := range idx.slotsOf(record) {
		if module, ok := merged[slot.Name]; ok {
			out[slot.Name] = module
		}
	}
	return out
}

func (idx *Index) equipmentIcon(tag, name, archetype, picture, tech string) string {
	tag = strings.ToUpper(tag)
	candidates := []string{}
	if picture != "" {
		candidates = append(candidates, "GFX_"+picture+"_medium")
	}
	if tech != "" {
		candidates = append(candidates, "GFX_"+tag+"_"+tech+"_medium")
	}
	candidates = append(candidates, "GFX_"+tag+"_"+name+"_medium", "GFX_"+name+"_medium")
	if tech != "" {
		candidates = append(candidates, "GFX_"+tech+"_medium")
	}
	candidates = append(candidates, "GFX_"+archetype+"_medium")
	for _, candidate := range candidates {
		if _, ok := idx.spriteRecordFor(candidate); ok {
			return candidate
		}
	}
	return ""
}

func (idx *Index) indexResourceFile(body []byte) {
	root, ok := parseScript(body)
	if !ok {
		return
	}
	expandConstants(root)
	for _, entry := range root.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil ||
			!strings.EqualFold(*entry.Key, "resources") {
			continue
		}
		for _, resource := range entry.Block.Entries {
			if resource == nil || resource.Key == nil || resource.Block == nil ||
				strings.HasPrefix(*resource.Key, "@") {
				continue
			}
			name := strings.ToLower(*resource.Key)
			if _, seen := idx.resourceFrames[name]; seen {
				continue
			}
			if frame, ok := strictInt(ciScalarValue(resource.Block.Entries, "icon_frame")); ok {
				idx.resourceFrames[name] = frame
			}
		}
	}
}

func (idx *Index) ResourceFrames() map[string]int {
	out := map[string]int{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for name, frame := range layer.resourceFrames {
			if _, seen := out[name]; !seen {
				out[name] = frame
			}
		}
	}
	return out
}

func containsWord(words []string, word string) bool {
	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}

var subUnitEditStats = []string{
	"combat_width", "max_strength", "max_organisation", "default_morale", "manpower",
	"training_time", "suppression", "weight", "supply_consumption",
}

func (record *subUnitRecord) editStat(key string) float64 {
	switch key {
	case "combat_width":
		return record.CombatWidth
	case "max_strength":
		return record.MaxStrength
	case "max_organisation":
		return record.MaxOrganisation
	case "default_morale":
		return record.DefaultMorale
	case "manpower":
		return record.Manpower
	case "training_time":
		return record.TrainingTime
	case "suppression":
		return record.Suppression
	case "weight":
		return record.Weight
	case "supply_consumption":
		return record.Supply
	}
	return 0
}

func (idx *Index) SubUnitSource(id string) (string, []byte, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.subUnits[id]; ok && record.File != "" {
			return record.File, layer.files[record.File], true
		}
	}
	return "", nil, false
}

func (idx *Index) SubUnitEditPayload(id string) map[string]any {
	id = strings.ToLower(strings.TrimSpace(id))
	record := idx.subUnitFor(id)
	if record == nil {
		return map[string]any{"id": id, "found": false}
	}
	file, _, _ := idx.SubUnitSource(id)
	stats := map[string]any{}
	for _, key := range subUnitEditStats {
		stats[key] = record.editStat(key)
	}
	need := []map[string]any{}
	for _, equipment := range record.NeedOrder {
		need = append(need, map[string]any{"equipment": equipment, "name": idx.localiseOr(equipment, equipment), "amount": record.Need[equipment]})
	}
	return map[string]any{
		"id": id, "found": true, "name": idx.localiseOr(id, id), "file": file,
		"stat_order": subUnitEditStats, "stats": stats, "need": need,
	}
}
