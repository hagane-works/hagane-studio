package guirender

import (
	"regexp"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

type projectOutputStat struct {
	Stat string
	Key  string

	Format string
}

var projectOutputStats = []projectOutputStat{
	{"manpower", "SERVICE_MANPOWER", "int"},
	{"carrier_sub_detection", "STAT_NAVY_CARRIER_SUB_DETECTION", "one"},
	{"carrier_surface_detection", "STAT_NAVY_CARRIER_SURFACE_DETECTION", "one"},
	{"defense", "STAT_ARMY_DEFENSE", "one"},
	{"breakthrough", "STAT_ARMY_BREAKTHROUGH", "one"},
	{"hardness", "STAT_ARMY_HARDNESS", "pct0"},
	{"soft_attack", "STAT_ARMY_SOFT_ATTACK", "one"},
	{"hard_attack", "STAT_ARMY_HARD_ATTACK", "one"},
	{"surface_detection", "STAT_NAVY_SURFACE_DETECTION", "one"},
	{"sub_detection", "STAT_NAVY_SUB_DETECTION", "one"},
	{"surface_visibility", "STAT_NAVY_SURFACE_VISIBILITY", "one"},
	{"sub_visibility", "STAT_NAVY_SUB_VISIBILITY", "one"},
	{"maximum_speed", "STAT_COMMON_MAXIMUM_SPEED", "one"},
	{"naval_speed", "STAT_NAVY_MAXIMUM_SPEED", "kn"},
	{"carrier_size", "STAT_CARRIER_SIZE", "int"},
	{"naval_range", "STAT_NAVY_RANGE", "km"},
	{"air_range", "STAT_AIR_RANGE", "km"},
	{"naval_torpedo_damage_reduction_factor", "STAT_NAVY_TORPEDO_DAMAGE_REDUCTION", "one"},
	{"naval_torpedo_enemy_critical_chance_factor", "STAT_NAVY_TORPEDO_ENEMY_CRIT_CHANCE", "pct"},
	{"naval_weather_penalty_factor", "STAT_NAVY_WEATHER_PENALTY", "pct0"},
	{"max_strength", "STAT_COMMON_MAX_STRENGTH", "one"},
	{"armor_value", "STAT_COMMON_ARMOR", "one"},
	{"ap_attack", "STAT_COMMON_PIERCING", "one"},
	{"lg_attack", "STAT_NAVY_LG_ATTACK", "one"},
	{"lg_armor_piercing", "STAT_NAVY_LG_ARMOR_PIERCING", "one"},
	{"hg_attack", "STAT_NAVY_HG_ATTACK", "one"},
	{"hg_armor_piercing", "STAT_NAVY_HG_ARMOR_PIERCING", "one"},
	{"torpedo_attack", "STAT_NAVY_TORPEDO_ATTACK", "one"},
	{"sub_attack", "STAT_NAVY_SUB_ATTACK", "one"},
	{"anti_air_attack", "STAT_NAVY_ANTI_AIR_ATTACK", "one"},
	{"air_attack", "STAT_AIR_ATTACK", "one"},
	{"air_defence", "STAT_AIR_DEFENCE", "one"},
	{"air_agility", "STAT_AIR_AGILITY", "one"},
	{"air_superiority", "STAT_AIR_SUPERIORITY", "one"},
	{"air_ground_attack", "STAT_AIR_GROUND_ATTACK", "one"},
	{"air_bombing", "STAT_AIR_BOMBING", "one"},
	{"naval_strike_attack", "STAT_AIR_NAVAL_STRIKE_ATTACK", "one"},
	{"naval_strike_targetting", "STAT_AIR_NAVAL_STRIKE_TARGETTING", "one"},
	{"thrust", "STAT_COMMON_THRUST", "one"},
	{"weight", "STAT_COMMON_WEIGHT", "one"},
	{"mines_planting", "STAT_NAVY_MINES_PLANTING", "two"},
	{"mines_sweeping", "STAT_NAVY_MINES_SWEEPING", "two"},
	{"recon", "STAT_ARMY_RECON", "one"},
	{"entrenchment", "STAT_ARMY_ENTRENCHMENT", "one"},
	{"suppression", "STAT_ARMY_SUPRESSION", "one"},
	{"reliability", "STAT_COMMON_RELIABILITY", "pct"},
	{"fuel_consumption", "STAT_COMMON_FUEL_CONSUMPTION", "two"},
	{"build_cost_ic", "STAT_COMMON_BUILD_COST_IC", "two"},
	{"naval_dominance_factor", "NAVAL_DOMINANCE_TECH_TREE", "int"},
}

func outputNumber(value float64, format string, share bool) string {
	if share {
		text := strconv.FormatFloat(value*100, 'f', 1, 64) + "%"
		if value > 0 {
			text = "+" + text
		}
		return text
	}
	switch format {
	case "int":
		return strconv.FormatFloat(value, 'f', 0, 64)
	case "two":
		return strconv.FormatFloat(value, 'f', 2, 64)
	case "kn":
		return strconv.FormatFloat(value, 'f', 1, 64) + " kn"
	case "km":
		return strconv.FormatFloat(value, 'f', 0, 64) + " km"
	case "pct":
		return strconv.FormatFloat(value*100, 'f', 1, 64) + "%"
	case "pct0":
		return strconv.FormatFloat(value*100, 'f', 0, 64) + "%"
	}
	return strconv.FormatFloat(value, 'f', 1, 64)
}

func (idx *Index) outputStatLines(add, multiply map[string]float64) []map[string]any {
	out := []map[string]any{}
	for _, stat := range projectOutputStats {
		label := strings.TrimSpace(idx.localiseOr(stat.Key, stat.Stat))
		if value := add[stat.Stat]; value != 0 {
			out = append(out, map[string]any{"stat": stat.Stat, "label": label, "value": outputNumber(value, stat.Format, false)})
		}
		if value := multiply[stat.Stat]; value != 0 {
			out = append(out, map[string]any{"stat": stat.Stat, "label": label, "value": outputNumber(value, stat.Format, true)})
		}
	}
	return out
}

func (idx *Index) outputResources(resources map[string]float64) []map[string]any {
	frames := idx.ResourceFrames()
	out := []map[string]any{}
	for _, name := range idx.resourceOrder(resources) {
		if resources[name] <= 0 {
			continue
		}
		out = append(out, map[string]any{"resource": name, "amount": resources[name], "frame": frames[name]})
	}
	return out
}

var locSlot = regexp.MustCompile(`\$([A-Za-z_]+)(\|[^$]*)?\$`)

var texticonName = regexp.MustCompile(`£([A-Za-z0-9_]+)`)

func iconSpriteName(name string) string {
	if strings.HasPrefix(name, "GFX_") {
		return name
	}
	return "GFX_" + name
}

func fillOutputText(template string, vars map[string]string) string {
	filled := locSlot.ReplaceAllStringFunc(template, func(slot string) string {
		match := locSlot.FindStringSubmatch(slot)
		if value, ok := vars[strings.ToUpper(match[1])]; ok {
			return value
		}
		return slot
	})
	return strings.TrimSpace(stripColour(filled))
}

func (idx *Index) outputEffectLines(state decisionStart, entries []*paradox.Entry, bonuses *[]map[string]any) []string {
	lines := []string{}
	taken := false
	for _, entry := range entries {
		if entry == nil || entry.Key == nil {
			continue
		}
		key := strings.ToLower(*entry.Key)
		if entry.Block != nil && (key == "if" || key == "else_if" || key == "else") {
			if key == "if" {
				taken = false
			}
			if taken {
				continue
			}
			if key != "else" && !state.holds(childBlock(entry.Block.Entries, "limit")) {
				continue
			}
			lines = append(lines, idx.outputEffectLines(state, entry.Block.Entries, bonuses)...)
			taken = key != "else"
			continue
		}
		switch key {
		case "custom_effect_tooltip", "custom_override_tooltip":
			if entry.Block == nil {
				if text := idx.localiseOr(strings.Trim(toString(scalarValue(entry.Scalar)), `"`), ""); text != "" {
					lines = append(lines, strings.TrimSpace(stripColour(text)))
				}
				continue
			}
			name := firstNonEmpty(ciScalar(entry.Block.Entries, "localization_key"), ciScalar(entry.Block.Entries, "tooltip"))
			template, ok := idx.locValue(strings.Trim(name, `"`))
			if !ok {
				continue
			}
			vars := map[string]string{}
			for _, field := range entry.Block.Entries {
				if field == nil || field.Key == nil || field.Block != nil {
					continue
				}
				value := strings.Trim(toString(scalarValue(field.Scalar)), `"`)
				vars[strings.ToUpper(*field.Key)] = idx.localiseOr(value, value)
			}
			lines = append(lines, fillOutputText(template, vars))
		case "add_design_template_bonus":
			if entry.Block == nil {
				continue
			}
			template, ok := idx.locValue("EFFECT_ADD_DESIGN_TEMPLATE_BONUS")
			if !ok {
				continue
			}
			factor, _ := toFloat(ciScalarValue(entry.Block.Entries, "cost_factor"))
			uses, ok := toFloat(ciScalarValue(entry.Block.Entries, "uses"))
			if !ok {
				uses = 1
			}
			equipment := strings.Trim(ciScalar(entry.Block.Entries, "equipment"), `"`)
			lines = append(lines, fillOutputText(template, map[string]string{
				"DISCOUNT": strconv.FormatFloat(factor*100, 'f', 2, 64) + "%",
				"USES":     strconv.FormatFloat(uses, 'f', -1, 64),
				"NAME":     idx.localiseOr(equipment, equipment),
			}))
			if bonuses != nil {
				*bonuses = append(*bonuses, map[string]any{
					"name": strings.Trim(ciScalar(entry.Block.Entries, "name"), `"`), "equipment": equipment,
					"label": idx.localiseOr(equipment, equipment), "uses": uses, "cost_factor": factor,
				})
			}
		case "set_technology":
			if entry.Block == nil {
				continue
			}
			template, ok := idx.locValue("SP_UNLOCK_TECH")
			if !ok {
				continue
			}
			for _, field := range entry.Block.Entries {
				if field == nil || field.Key == nil || field.Block != nil || strings.EqualFold(*field.Key, "popup") {
					continue
				}
				lines = append(lines, fillOutputText(template, map[string]string{"TECH": idx.localiseOr(*field.Key, *field.Key)}))
			}
		}
	}
	return lines
}

func outputUnlocks(state decisionStart, output *paradox.Block, key string) []string {
	out := []string{}
	if output == nil {
		return out
	}
	for _, entry := range output.Entries {
		if entry == nil || entry.Key == nil || entry.Block == nil || !strings.EqualFold(*entry.Key, key) {
			continue
		}
		if !state.holds(childBlock(entry.Block.Entries, "limit")) {
			continue
		}
		for _, name := range blockTokens(entry.Block.Entries) {
			out = append(out, strings.ToLower(name))
		}
	}
	return out
}

func (idx *Index) SpecialProjectDetailPayload(tag, date, id string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	var project *specialProjectRecord
	for _, record := range idx.specialProjectList() {
		if strings.EqualFold(record.Name, strings.TrimSpace(id)) {
			project = record
		}
	}
	if project == nil {
		return map[string]any{"found": false, "id": id}
	}
	state := idx.decisionStartOf(tag, date)
	year := yearOf(date)
	pictures := []string{"GFX_resources_strip"}
	iconsIn := func(text string) {
		for _, match := range texticonName.FindAllStringSubmatch(text, -1) {
			pictures = append(pictures, iconSpriteName(match[1]))
		}
	}
	label, labelKey := idx.localiseOr(project.Name, project.Name), project.Name
	if own, ok := idx.Localise(tag + "_" + project.Name); ok {
		label, labelKey = own, tag+"_"+project.Name
	}
	desc, descKey := "", project.Name+"_desc"
	for _, key := range []string{labelKey + "_desc", project.Name + "_desc"} {
		if text, ok := idx.Localise(key); ok {
			desc, descKey = text, key
			break
		}
	}
	icon := project.Icon
	if icon == "" {
		if _, ok := idx.spriteRecordFor("GFX_" + project.Name); ok {
			icon = "GFX_" + project.Name
		}
	}
	if icon != "" {
		pictures = append(pictures, icon)
	}
	outputs := []string{}
	discounts := []map[string]any{}
	if project.Output != nil {
		if effects := childBlock(project.Output.Entries, "country_effects"); effects != nil {
			outputs = idx.outputEffectLines(state, effects.Entries, &discounts)
		}
	}
	modules := []map[string]any{}
	for _, name := range outputUnlocks(state, project.Output, "enable_equipment_modules") {
		module := idx.equipmentModuleFor(name)
		if module == nil {
			continue
		}
		item := map[string]any{
			"name": name, "label": idx.localiseOr(name, name),
			"stats":     idx.outputStatLines(module.Add, module.Multiply),
			"resources": idx.outputResources(module.Resources),
		}
		for _, candidate := range []string{"GFX_SMI_" + firstNonEmpty(module.Gfx, name), "GFX_EMI_" + name} {
			if _, ok := idx.spriteRecordFor(candidate); ok {
				item["icon"] = candidate
				pictures = append(pictures, candidate)
				break
			}
		}
		for _, stat := range item["stats"].([]map[string]any) {
			iconsIn(stat["label"].(string))
		}
		modules = append(modules, item)
	}
	equipment := []map[string]any{}
	unlockedBy := idx.enablingTech()
	for _, name := range outputUnlocks(state, project.Output, "enable_equipments") {
		record := idx.equipmentFor(name)
		if record == nil {
			continue
		}
		stats, resources := idx.designStats(record.Name, idx.defaultModulesOf(record), nil, "", year)
		item := map[string]any{
			"name": record.Name, "label": idx.localiseOr(record.Name, record.Name),
			"stats":     idx.outputStatLines(stats, nil),
			"resources": idx.outputResources(resources),
		}
		if archetype := idx.equipmentFor(record.Archetype); archetype != nil {
			if picture := idx.designIcon(tag, record, archetype, unlockedBy); picture != "" {
				item["icon"] = picture
				pictures = append(pictures, picture)
			}
		}
		for _, stat := range item["stats"].([]map[string]any) {
			iconsIn(stat["label"].(string))
		}
		equipment = append(equipment, item)
	}
	return map[string]any{
		"found": true, "id": project.Name, "label": label, "label_key": labelKey,
		"description": strings.TrimSpace(desc), "description_key": descKey, "icon": icon,
		"outputs": outputs, "modules": modules, "equipment": equipment,

		"discounts": discounts,
		"texts": map[string]any{
			"output_header":    idx.localiseOr("ONGOING_PROJECT_OUTPUT_HEADER", "Project Output"),
			"rewards_header":   idx.localiseOr("SPECIAL_PROJECT_REWARDS_HEADER_TEXT", "Output(s)"),
			"modules_header":   idx.localiseOr("SPECIAL_PROJECT_REWARD_EQUIPMENT_MODULE", "Equipment module(s)"),
			"equipment_header": idx.localiseOr("SPECIAL_PROJECT_REWARD_EQUIPMENT", "Equipment(s)"),
		},
		"sprites": idx.SpritesPayload(pictures),
	}
}
