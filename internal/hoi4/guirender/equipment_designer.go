package guirender

import (
	"sort"
	"strconv"
	"strings"
)

type moduleSlot struct {
	Name       string
	Required   bool
	Categories []string

	Ref string
}

type designerStatRow struct {
	Key  string
	Stat string

	Format string
}

var tankDesignerColumns = map[string][]designerStatRow{
	"base": {
		{"STAT_COMMON_MAXIMUM_SPEED", "maximum_speed", "speed"},
		{"STAT_COMMON_RELIABILITY", "reliability", "percent"},
		{"STAT_ARMY_SUPPLY_CONSUMPTION", "supply_consumption", "rate"},
	},
	"combat": {
		{"STAT_ARMY_SOFT_ATTACK", "soft_attack", ""},
		{"STAT_ARMY_HARD_ATTACK", "hard_attack", ""},
		{"STAT_COMMON_PIERCING", "ap_attack", ""},
		{"STAT_ARMY_HARDNESS", "hardness", "percent"},
		{"STAT_COMMON_ARMOR", "armor_value", ""},
		{"STAT_ARMY_BREAKTHROUGH", "breakthrough", ""},
		{"STAT_ARMY_DEFENSE", "defense", ""},
		{"STAT_AIR_ATTACK", "air_attack", ""},
	},
	"misc": {
		{"STAT_ARMY_FUEL_CAPACITY", "fuel_capacity", ""},
		{"STAT_COMMON_FUEL_CONSUMPTION", "fuel_consumption", "rate"},
		{"STAT_ARMY_SUPRESSION", "suppression", ""},
		{"STAT_ARMY_RECON", "recon", ""},
		{"STAT_ARMY_ENTRENCHMENT", "entrenchment", ""},
	},
}

var planeDesignerColumns = map[string][]designerStatRow{
	"base": {
		{"STAT_COMMON_MAXIMUM_SPEED", "maximum_speed", "speed"},
		{"STAT_AIR_RANGE", "air_range", "km"},
		{"STAT_ARMY_SUPPLY_CONSUMPTION", "supply_consumption", "rate"},
		{"STAT_COMMON_WEIGHT", "weight", ""},
		{"STAT_COMMON_THRUST", "thrust", ""},
	},
	"combat": {
		{"STAT_AIR_DEFENCE", "air_defence", ""},
		{"STAT_AIR_ATTACK", "air_attack", ""},
		{"STAT_AIR_AGILITY", "air_agility", ""},
		{"STAT_AIR_SUPERIORITY", "air_superiority", "rate"},
		{"STAT_AIR_NAVAL_STRIKE_ATTACK", "naval_strike_attack", "rate"},
		{"STAT_AIR_NAVAL_STRIKE_TARGETTING", "naval_strike_targetting", "rate"},
		{"STAT_AIR_GROUND_ATTACK", "air_ground_attack", ""},
		{"STAT_AIR_BOMBING", "air_bombing", ""},
	},
	"misc": {
		{"STAT_COMMON_RELIABILITY", "reliability", "percent"},
		{"STAT_COMMON_FUEL_CONSUMPTION", "fuel_consumption", "rate"},
		{"STAT_NAVY_SURFACE_DETECTION", "surface_detection", ""},
		{"STAT_NAVY_SUB_DETECTION", "sub_detection", ""},
		{"STAT_COMMON_NIGHT_PENALTY", "night_penalty", "percent"},
		{"STAT_NAVY_MINES_PLANTING", "mines_planting", "rate"},
		{"STAT_NAVY_MINES_SWEEPING", "mines_sweeping", "rate"},
	},
}

var shipDesignerColumns = map[string][]designerStatRow{
	"base": {
		{"STAT_NAVY_MAXIMUM_SPEED", "naval_speed", "kn"},
		{"STAT_NAVY_RANGE", "naval_range", "km"},
		{"STAT_COMMON_MAX_ORG", "max_organisation", ""},
		{"STAT_COMMON_MAX_STRENGTH", "max_strength", ""},
		{"STAT_COMMON_RELIABILITY", "reliability", "percent"},
		{"STAT_ARMY_SUPPLY_CONSUMPTION", "supply_consumption", "rate"},
		{"DESIGNER_MANPOWER", "manpower", "int"},
	},
	"combat": {
		{"STAT_NAVY_LG_ATTACK", "lg_attack", ""},
		{"STAT_NAVY_LG_ARMOR_PIERCING", "lg_armor_piercing", ""},
		{"STAT_NAVY_HG_ATTACK", "hg_attack", ""},
		{"STAT_NAVY_HG_ARMOR_PIERCING", "hg_armor_piercing", ""},
		{"STAT_NAVY_TORPEDO_ATTACK", "torpedo_attack", ""},
		{"STAT_NAVY_SUB_ATTACK", "sub_attack", ""},
		{"STAT_COMMON_ARMOR", "armor_value", ""},
		{"STAT_NAVY_ANTI_AIR_ATTACK", "anti_air_attack", ""},
	},
	"misc": {
		{"STAT_COMMON_FUEL_CONSUMPTION", "fuel_consumption", "rate"},
		{"STAT_NAVY_SURFACE_VISIBILITY", "surface_visibility", ""},
		{"STAT_NAVY_SURFACE_DETECTION", "surface_detection", ""},
		{"STAT_NAVY_SUB_VISIBILITY", "sub_visibility", ""},
		{"STAT_NAVY_SUB_DETECTION", "sub_detection", ""},
		{"STAT_NAVY_MINES_PLANTING", "mines_planting", "rate"},
		{"STAT_NAVY_MINES_SWEEPING", "mines_sweeping", "rate"},
		{"STAT_NAVY_WEATHER_PENALTY", "naval_weather_penalty_factor", "pct0"},
		{"STAT_NAVY_HIT_PROFILE", "hit_profile", ""},
	},
}

func designerKind(archetype *equipmentRecord) string {
	if archetype == nil {
		return ""
	}
	if containsWord(archetype.Types, "armor") {
		return "tank"
	}
	switch archetype.Group {
	case "interface_category_air":
		return "plane"
	case "interface_category_capital_ships", "interface_category_screen_ships", "interface_category_other_ships":
		return "ship"
	}
	return ""
}

func (idx *Index) shipUnitFor(archetype string, modules map[string]string) *subUnitRecord {
	archetype = strings.ToLower(archetype)
	fitted := map[string]bool{}
	for _, module := range modules {
		fitted[module] = true
	}
	better := func(record, than *subUnitRecord) bool {
		return than == nil || record.Priority > than.Priority || (record.Priority == than.Priority && record.Name < than.Name)
	}
	var best, anyShip *subUnitRecord
	seen := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.subUnits {
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, built := record.NeedEquipment[archetype]; !built {
				continue
			}
			if better(record, anyShip) {
				anyShip = record
			}
			fits := true
			for _, group := range record.NeedModules {
				any := false
				for _, module := range group {
					any = any || fitted[module]
				}
				fits = fits && any
			}
			if fits && better(record, best) {
				best = record
			}
		}
	}
	if best == nil {
		return anyShip
	}
	return best
}

func (idx *Index) shipStats(stats map[string]float64, unit *subUnitRecord) map[string]float64 {
	out := map[string]float64{}
	for key, value := range stats {
		out[key] = value
	}
	mult := 1.0
	if unit != nil {
		out["max_organisation"] = unit.MaxOrganisation
		out["supply_consumption"] = unit.Supply * idx.defineNumber("NNavy.SUPPLY_NEED_FACTOR")
		if unit.HitProfileMult > 0 {
			mult = unit.HitProfileMult
		}
	}
	if speed := idx.defineNumber("NNavy.HIT_PROFILE_SPEED_BASE") + out["naval_speed"]*idx.defineNumber("NNavy.HIT_PROFILE_SPEED_FACTOR"); speed > 0 {
		out["hit_profile"] = out["surface_visibility"] * idx.defineNumber("NNavy.HIT_PROFILE_MULT") * mult / speed
	}
	if _, ok := stats["naval_weather_penalty_factor"]; !ok {
		out["naval_weather_penalty_factor"] = 1
	}
	return out
}

func (idx *Index) navalDominance(stats map[string]float64) float64 {
	rangeNeutral := idx.defineNumber("NNavy.DOMINANCE_PER_SHIP_PER_RANGE_NEUTRAL")
	speedNeutral := idx.defineNumber("NNavy.DOMINANCE_PER_SHIP_PER_SPEED_NEUTRAL")
	if rangeNeutral <= 0 || speedNeutral <= 0 {
		return 0
	}
	scale := (stats["naval_range"]/rangeNeutral + stats["naval_speed"]/speedNeutral) / 2
	scale += stats["hg_attack"] * idx.defineNumber("NNavy.DOMINANCE_PER_SHIP_PER_HEAVY_GUN_ATTACK")
	scale += stats["carrier_size"] * idx.defineNumber("NNavy.DOMINANCE_PER_SHIP_PER_CARRIER_SIZE")
	return stats["naval_dominance_factor"] * scale
}

func (idx *Index) planeStats(stats map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for key, value := range stats {
		out[key] = value
	}
	if spare := out["thrust"] - out["weight"]; spare > 0 {
		out["air_agility"] += spare * idx.defineNumber("NAir.THRUST_WEIGHT_AGILITY_FACTOR")
	}
	supply := idx.defineNumber("NAir.SUPPLY_NEED_FACTOR")
	if own, ok := out["supply_consumption"]; ok && own > 0 {
		supply *= own
	}
	out["supply_consumption"] = supply
	return out
}

func (idx *Index) designName(tag string, record *equipmentRecord) string {
	if made := record.DerivedVariantName; made != "" {
		for _, key := range []string{tag + "_" + made + "_short", tag + "_" + made, made} {
			if name, ok := idx.Localise(key); ok && name != "" {
				return name
			}
		}
	}
	return firstNonEmpty(idx.localiseOr(tag+"_"+record.Name, ""), idx.localiseOr(record.Name, record.Name))
}

func (idx *Index) planeRole(archetype *equipmentRecord, modules map[string]string) string {
	module := idx.equipmentModuleFor(modules[archetype.TypeOverride])
	if archetype.TypeOverride == "" || module == nil {
		return idx.localiseOr("EQUIPMENT_DESIGNER_PLANE_ROLE_REQUIRE_WEAPON", "")
	}
	for _, role := range module.AddTypes {
		if len(archetype.AllowedTypes) == 0 || containsWord(archetype.AllowedTypes, role) {
			return idx.localiseOr("EQUIPMENT_"+strings.ToUpper(role), role)
		}
	}
	return ""
}

func (idx *Index) resourceOrder(resources map[string]float64) []string {
	frames := idx.ResourceFrames()
	out := []string{}
	for name := range resources {
		out = append(out, name)
	}
	sort.Slice(out, func(a, b int) bool {
		fa, oka := frames[out[a]]
		fb, okb := frames[out[b]]
		if oka != okb {
			return oka
		}
		if fa != fb {
			return fa < fb
		}
		return out[a] < out[b]
	})
	return out
}

func (idx *Index) designsOf(tag, date, archetype string) []map[string]any {
	out := []map[string]any{}
	seen := map[string]bool{}
	for _, variant := range idx.EquipmentVariantsOf(tag, date) {
		record := idx.equipmentFor(variant.Type)
		if record == nil || !strings.EqualFold(record.Archetype, archetype) || seen[variant.Name] {
			continue
		}
		seen[variant.Name] = true
		out = append(out, map[string]any{"name": variant.Name, "type": variant.Type, "type_label": idx.localiseOr(variant.Type, variant.Type)})
	}
	sort.Slice(out, func(a, b int) bool { return out[a]["name"].(string) < out[b]["name"].(string) })
	return out
}

func (idx *Index) battalionFor(archetype string) *subUnitRecord {
	archetype = strings.ToLower(archetype)
	var best *subUnitRecord
	seen := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, record := range layer.subUnits {
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, needs := record.Need[archetype]; !needs {
				continue
			}
			if best == nil || record.Priority > best.Priority || (record.Priority == best.Priority && record.Name < best.Name) {
				best = record
			}
		}
	}
	return best
}

func battalionStats(stats map[string]float64, unit *subUnitRecord) map[string]float64 {
	out := map[string]float64{}
	for key, value := range stats {
		out[key] = value
	}
	if unit == nil {
		return out
	}
	out["supply_consumption"] = unit.Supply
	out["suppression"] = unit.Suppression
	out["recon"] = unit.Recon
	out["entrenchment"] = unit.Entrenchment
	for key, factor := range unit.Modifiers {
		out[key] *= 1 + factor
	}
	return out
}

func (idx *Index) hasWindow(name string) bool {
	for layer := idx; layer != nil; layer = layer.fallback {
		if _, ok := layer.windows[name]; ok {
			return true
		}
	}
	return false
}

func (idx *Index) slotsOf(record *equipmentRecord) []moduleSlot {
	return idx.slotsAt(record, 0)
}

func (idx *Index) slotsAt(record *equipmentRecord, depth int) []moduleSlot {
	if record == nil || depth > 16 {
		return nil
	}
	archetype := idx.equipmentFor(record.Archetype)
	if len(record.ModuleSlots) == 0 || record.SlotsInherit {
		if archetype != nil && archetype != record {
			return idx.slotsAt(archetype, depth+1)
		}
		return nil
	}
	var inherited []moduleSlot
	source := func() []moduleSlot {
		if inherited == nil {
			if parent := idx.equipmentFor(record.Parent); parent != nil && parent != record {
				inherited = idx.slotsAt(parent, depth+1)
			} else if archetype != nil && archetype != record {
				inherited = idx.slotsAt(archetype, depth+1)
			}
		}
		return inherited
	}
	find := func(slots []moduleSlot, name string) (moduleSlot, bool) {
		for _, slot := range slots {
			if slot.Name == name {
				return slot, true
			}
		}
		return moduleSlot{}, false
	}
	out := []moduleSlot{}
	for _, slot := range record.ModuleSlots {
		if slot.Ref == "" {
			out = append(out, slot)
			continue
		}
		from := slot.Name
		if slot.Ref != "inherit" {
			from = slot.Ref
			if copied, ok := find(out, from); ok {
				out = append(out, moduleSlot{Name: slot.Name, Required: copied.Required, Categories: copied.Categories})
				continue
			}
		}
		if copied, ok := find(source(), from); ok {
			out = append(out, moduleSlot{Name: slot.Name, Required: copied.Required, Categories: copied.Categories})
		}
	}
	return out
}

func (idx *Index) upgradesOf(record *equipmentRecord) []string {
	seen := map[string]bool{}
	for at := record; at != nil && !seen[strings.ToLower(at.Name)]; at = idx.equipmentFor(at.Parent) {
		seen[strings.ToLower(at.Name)] = true
		if len(at.Upgrades) > 0 {
			return at.Upgrades
		}
		if at.Parent == "" {
			break
		}
	}
	if archetype := idx.equipmentFor(record.Archetype); archetype != nil {
		return archetype.Upgrades
	}
	return nil
}

func (idx *Index) EquipmentDesignerPayload(tag, date, name string) map[string]any {
	return idx.EquipmentDesignerDraftPayload(tag, date, name, DesignDraft{})
}

type DesignDraft struct {
	Modules  map[string]string
	Upgrades map[string]float64
	Options  bool
}

func ParseDesignDraft(modules, upgrades string, options bool) DesignDraft {
	draft := DesignDraft{Options: options}
	for _, pair := range strings.Split(modules, ";") {
		slot, module, ok := strings.Cut(strings.TrimSpace(pair), ":")
		if !ok || slot == "" {
			continue
		}
		if draft.Modules == nil {
			draft.Modules = map[string]string{}
		}
		draft.Modules[strings.ToLower(slot)] = strings.ToLower(strings.TrimSpace(module))
	}
	for _, pair := range strings.Split(upgrades, ";") {
		name, level, ok := strings.Cut(strings.TrimSpace(pair), ":")
		if !ok || name == "" {
			continue
		}
		if value, err := strconv.ParseFloat(strings.TrimSpace(level), 64); err == nil {
			if draft.Upgrades == nil {
				draft.Upgrades = map[string]float64{}
			}
			draft.Upgrades[strings.ToLower(name)] = value
		}
	}
	return draft
}

func (idx *Index) DesignSlotNames(typeName string) ([]string, bool) {
	record := idx.equipmentFor(typeName)
	if record == nil || record.IsArchetype {
		return nil, false
	}
	names := []string{}
	for _, slot := range idx.slotsOf(record) {
		names = append(names, slot.Name)
	}
	return names, len(names) > 0
}

func (idx *Index) KnownEquipmentModule(name string) bool {
	return idx.equipmentModuleFor(name) != nil
}

func (idx *Index) modulesFor(slot moduleSlot) []*equipmentModule {
	order := map[string]int{}
	for at, category := range slot.Categories {
		order[category] = at + 1
	}
	out := []*equipmentModule{}
	seen := map[string]bool{}
	for layer := idx; layer != nil; layer = layer.fallback {
		for key, module := range layer.equipmentModules {
			if seen[key] || order[module.Category] == 0 {
				continue
			}
			seen[key] = true
			out = append(out, module)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if order[out[a].Category] != order[out[b].Category] {
			return order[out[a].Category] < order[out[b].Category]
		}
		return strings.ToLower(out[a].Name) < strings.ToLower(out[b].Name)
	})
	return out
}

func (idx *Index) EquipmentDesignerDraftPayload(tag, date, name string, draft DesignDraft) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	year := yearOf(date)
	typeName, designName := "", ""
	modules := map[string]string{}
	upgradeLevels := map[string]float64{}
	team := ""

	variantIcon := ""
	for _, variant := range idx.EquipmentVariantsOf(tag, date) {
		if !strings.EqualFold(variant.Name, name) {
			continue
		}
		typeName, designName = variant.Type, variant.Name
		if block := childBlock(variant.Block.Entries, "modules"); block != nil {
			for _, slot := range block.Entries {
				if slot != nil && slot.Key != nil && slot.Block == nil {
					modules[strings.ToLower(*slot.Key)] = strings.ToLower(strings.Trim(toString(scalarValue(slot.Scalar)), `"`))
				}
			}
		}
		upgradeLevels = numberBlock(childBlock(variant.Block.Entries, "upgrades"))
		team = ciScalar(variant.Block.Entries, "design_team")
		variantIcon = strings.Trim(ciScalar(variant.Block.Entries, "icon"), `"`)
	}
	fromHistory := typeName != ""
	record := idx.equipmentFor(firstNonEmpty(typeName, name))
	if record == nil {
		return map[string]any{"found": false, "name": name}
	}
	archetypeRecord := idx.equipmentFor(record.Archetype)
	kind := designerKind(archetypeRecord)
	version := ""
	versionOf := func(same int) string {
		if same < 0 || same >= 26 {
			return ""
		}
		return string(rune('A' + same))
	}
	typeNameIsNew := typeName == ""
	if typeNameIsNew {
		typeName = record.Name
		designName = idx.designName(tag, record)
		modules = idx.defaultModulesOf(record)

		same := 0
		for _, variant := range idx.EquipmentVariantsOf(tag, date) {
			if strings.EqualFold(variant.Name, designName) {
				same++
			}
		}
		if same > 0 {
			version = versionOf(same)
		}
	}
	sprites := map[string]any{}
	sprite := func(name string) bool {
		if name == "" {
			return false
		}
		if _, done := sprites[name]; done {
			return true
		}
		payload := idx.spritePayload(name)
		if resolved, _ := payload["resolved"].(bool); !resolved {
			return false
		}
		sprites[name] = payload
		return true
	}
	for slot, module := range draft.Modules {
		modules[slot] = module
	}
	for upgrade, level := range draft.Upgrades {
		upgradeLevels[upgrade] = level
	}
	sprite("GFX_EMSI_empty")
	sprite("GFX_EMSI_locked")
	slots := []map[string]any{}

	rows := map[string]int{}
	for _, slot := range idx.slotsOf(record) {
		module := modules[slot.Name]
		if module == "empty" {
			module = ""
		}
		row := "custom"
		if (kind == "plane" || kind == "ship") && strings.HasPrefix(slot.Name, "fixed_") {
			row = "fixed"
		}
		entry := map[string]any{"slot": slot.Name, "row": row, "index": rows[row], "required": slot.Required, "module": module}
		rows[row]++

		if empty := "GFX_EMSI_" + slot.Name; module == "" && sprite(empty) {
			entry["empty_icon"] = empty
		}
		if module != "" {
			entry["label"] = idx.localiseOr(module, module)
			if definition := idx.equipmentModuleFor(module); definition != nil {
				entry["category"] = definition.Category
			}

			for _, icon := range []string{"GFX_EMI_" + module, "GFX_SMI_" + firstNonEmpty(idx.moduleGfx(module), module)} {
				if sprite(icon) {
					entry["icon"] = icon
					break
				}
			}
		}

		if draft.Options {
			allowed := []map[string]any{}
			for _, option := range idx.modulesFor(slot) {
				key := strings.ToLower(option.Name)
				allowed = append(allowed, map[string]any{"name": key, "label": idx.localiseOr(key, key), "category": option.Category})
			}
			entry["allowed"] = allowed

			entry["slot_label"] = idx.localiseOr("EQ_MOD_SLOT_"+slot.Name+"_TITLE", slot.Name)
		}
		slots = append(slots, entry)
	}
	upgrades := []map[string]any{}
	for _, upgrade := range idx.upgradesOf(record) {
		entry := map[string]any{
			"name": upgrade, "label": idx.localiseOr(upgrade, upgrade), "level": int(upgradeLevels[upgrade]),
		}
		if definition := idx.equipmentUpgradeFor(upgrade); definition != nil && definition.MaxLevel > 0 {
			entry["max"] = definition.MaxLevel
		}
		upgrades = append(upgrades, entry)
	}
	stats, resources := idx.designStats(typeName, modules, upgradeLevels, team, year)
	var unit *subUnitRecord
	var shown map[string]float64
	layout := tankDesignerColumns
	dominance := 0.0
	if kind == "plane" {
		shown = idx.planeStats(stats)
		layout = planeDesignerColumns
	} else if kind == "ship" {
		unit = idx.shipUnitFor(record.Archetype, modules)
		shown = idx.shipStats(stats, unit)
		layout = shipDesignerColumns
		dominance = idx.navalDominance(shown)
	} else {
		unit = idx.battalionFor(record.Archetype)
		shown = battalionStats(stats, unit)
	}
	columns := map[string]any{}
	for column, lines := range layout {
		items := []map[string]any{}
		for _, row := range lines {
			items = append(items, map[string]any{
				"key": row.Key, "stat": row.Stat, "format": row.Format, "value": shown[row.Stat],
				"label": idx.localiseOr(row.Key, row.Key),
			})
		}
		columns[column] = items
	}

	blueprint := ""
	lower := strings.ToLower(tag)
	for _, candidate := range []string{
		"equipment_designer_" + typeName + "_" + lower,
		"equipment_designer_" + record.Archetype + "_" + lower,
		"equipment_designer_" + typeName,
		"equipment_designer_" + record.Archetype,
	} {
		if idx.hasWindow(candidate) {
			blueprint = candidate
			break
		}
	}
	costFormat, _ := idx.locValue("DIVISION_PRODUCTION_COST_LAND")
	saveFormat, _ := idx.locValue("EQUIPMENT_DESIGNER_SAVE")
	start, _ := idx.countryStartAt(tag, date)

	icon := ""
	if variantIcon != "" && sprite(variantIcon) {
		icon = variantIcon
	} else if archetypeRecord != nil {
		icon = idx.designIcon(tag, record, archetypeRecord, idx.enablingTech())
		sprite(icon)
	}
	role := ""
	if unit != nil {

		role = idx.localiseOr(unit.Name, unit.Name)
	}
	title := idx.localiseOr("TANK_DESIGNER_TITLE", "Tank Designer")

	experience, experienceIcon := start.ArmyXP, "army_experience"
	names := ""
	if kind == "plane" {
		role = idx.planeRole(archetypeRecord, modules)
		title = idx.localiseOr("PLANE_DESIGNER_TITLE", "Plane Designer")
		experience, experienceIcon = start.AirXP, "air_experience"
	}
	if kind == "ship" {
		role = ""
		title = idx.localiseOr("SHIP_DESIGNER_TITLE", "Ship Designer")
		experience, experienceIcon = start.NavyXP, "navy_experience"

		names = idx.localiseOr("MODIFIER_NONE", "None")

		if typeNameIsNew && version == "" {
			version = versionOf(0)
		}
	}
	return map[string]any{
		"kind": kind,
		"texts": map[string]any{
			"title": title,

			"cost":         costFormat,
			"base_label":   idx.localiseOr("DESIGNER_BASE_STATS_LABEL", ""),
			"combat_label": idx.localiseOr("DESIGNER_COMBAT_STATS_LABEL", ""),
			"misc_label":   idx.localiseOr("FE_MISC_STATS", ""),

			"save": saveFormat,
		},

		"experience":      experience,
		"experience_icon": experienceIcon,
		"version":         version,
		"resource_order":  idx.resourceOrder(resources),
		"names":           names,
		"dominance":       dominance,

		"cost_decimals": func() int {
			if kind == "ship" {
				return 0
			}
			return 2
		}(),
		"icon":  icon,
		"role":  role,
		"found": true, "name": designName, "type": typeName, "archetype": record.Archetype,

		"variant":    fromHistory,
		"designs":    idx.designsOf(tag, date, record.Archetype),
		"type_label": idx.localiseOr(typeName, typeName), "year": record.Year,
		"slots": slots, "upgrades": upgrades, "columns": columns,
		"cost": stats["build_cost_ic"], "resources": resources, "blueprint": blueprint,
		"battalion": func() string {
			if unit == nil {
				return ""
			}
			return unit.Name
		}(),
		"sprites": sprites,
	}
}
