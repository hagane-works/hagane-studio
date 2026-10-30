package guirender

import (
	"regexp"
	"sort"
	"strings"

	"haganeworks/internal/hoi4/paradox"
)

var mioValueSlot = regexp.MustCompile(`\$VALUE[^$]*\$`)

var mioTraitSprites = []string{
	"GFX_country_spefific_org_trait_button",
	"GFX_design_team_icon",
	"GFX_industrial_manufacturer_icon",
	"GFX_organization_modifier_icon",
	"GFX_trait_line_vertical_tile_8x8",
	"GFX_trait_line_horizontal_tile_8x8",
	"GFX_trait_line_dotted_tile_8x8",
}

func traitPosition(traits map[string]mioTrait, token string, depth int) (x, y int) {
	trait, ok := traits[token]
	if !ok || depth > 16 {
		return 0, 0
	}
	for _, field := range trait.Block.Entries {
		if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, "position") {
			x = coordinate(field.Block.Entries, "x", trait.Vars)
			y = coordinate(field.Block.Entries, "y", trait.Vars)
		}
	}
	if base := ciScalar(trait.Block.Entries, "relative_position_id"); base != "" && base != token {
		bx, by := traitPosition(traits, base, depth+1)
		x, y = x+bx, y+by
	}
	return x, y
}

func scriptTokens(block *paradox.Block, key string) []string {
	if block == nil {
		return nil
	}
	for _, field := range block.Entries {
		if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, key) {
			return blockTokens(field.Block.Entries)
		}
	}
	return nil
}

func modifierSum(traits []*paradox.Block, name string) float64 {
	total := 0.0
	for _, trait := range traits {
		if trait == nil {
			continue
		}
		for _, field := range trait.Entries {
			if field == nil || field.Key == nil || field.Block == nil || !strings.EqualFold(*field.Key, "organization_modifier") {
				continue
			}
			if value, ok := toFloat(ciScalarValue(field.Block.Entries, name)); ok {
				total += value
			}
		}
	}
	return total
}

func (idx *Index) IndustrialOrgPayload(tag, date, token string) map[string]any {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	token = strings.TrimSpace(token)
	org := idx.resolveIndustrialOrg(token)
	if org == nil {
		return map[string]any{"tag": tag, "token": token, "traits": []any{}}
	}
	state := idx.decisionStartOf(tag, date)
	state.from = tag
	localise := func(key string) string {
		if text, ok := idx.LocaliseFor(key, tag, date); ok {
			return text
		}
		return key
	}
	category := idx.industrialOrgCategory(org.EquipmentGroups)
	out := map[string]any{
		"tag":      tag,
		"token":    token,
		"name":     localise(token),
		"category": category,
		"size":     0,
		"points":   0,
	}
	if org.Icon != "" {
		out["icon"] = org.Icon
		out["icon_sprite"] = idx.spritePayload(org.Icon)
	}

	background := "GFX_MIO_details_background_" + category
	if _, ok := idx.spriteRecordFor(background); !ok {
		background = "GFX_MIO_details_background"
	}
	out["background"] = background
	sprites := map[string]any{background: idx.spritePayload(background)}
	for _, name := range mioTraitSprites {
		sprites[name] = idx.spritePayload(name)
	}

	headers := []map[string]any{}
	for _, header := range org.Headers {
		headers = append(headers, map[string]any{"text": localise(header.Text), "x": header.X})
	}
	out["headers"] = headers

	byToken := map[string]mioTrait{}
	for _, trait := range org.Traits {
		byToken[trait.token()] = trait
	}
	shown := map[string]bool{}
	for _, trait := range org.Traits {
		if state.holds(traitBlock(trait.Block, "visible")) {
			shown[trait.token()] = true
		}
	}
	keep := func(tokens []string) []string {
		out := []string{}
		for _, token := range tokens {
			if shown[token] {
				out = append(out, token)
			}
		}
		return out
	}
	traits := []map[string]any{}
	for _, trait := range org.Traits {
		key := trait.token()
		if !shown[key] {
			continue
		}
		x, y := traitPosition(byToken, key, 0)
		entry := map[string]any{
			"token":      key,
			"label":      localise(firstNonEmpty(ciScalar(trait.Block.Entries, "name"), key)),
			"x":          x,
			"y":          y,
			"special":    strings.EqualFold(ciScalar(trait.Block.Entries, "special_trait_background"), "yes"),
			"parents":    keep(scriptTokens(trait.Block, "all_parents")),
			"any_parent": keep(scriptTokens(trait.Block, "any_parent")),
			"exclusive":  keep(scriptTokens(trait.Block, "mutually_exclusive")),
		}
		if icon := ciScalar(trait.Block.Entries, "icon"); icon != "" {
			entry["icon"] = icon
			sprites[icon] = idx.spritePayload(icon)
		}
		bonuses := []string{}
		if hasScriptBlock(trait.Block, "equipment_bonus") {
			bonuses = append(bonuses, "design")
		}
		if hasScriptBlock(trait.Block, "production_bonus") {
			bonuses = append(bonuses, "production")
		}
		if hasScriptBlock(trait.Block, "organization_modifier") {
			bonuses = append(bonuses, "organization")
		}
		entry["bonuses"] = bonuses

		values := map[string]any{}
		for _, field := range []string{"equipment_bonus", "production_bonus", "organization_modifier"} {
			if block := traitBlock(trait.Block, field); block != nil {
				values[field] = orderedNumbers(block)
			}
		}
		entry["values"] = values
		if base := ciScalar(trait.Block.Entries, "relative_position_id"); base != "" && base != key {
			entry["relative_to"] = base
		}
		traits = append(traits, entry)
	}

	sort.SliceStable(traits, func(a, b int) bool {
		if traits[a]["y"].(int) != traits[b]["y"].(int) {
			return traits[a]["y"].(int) < traits[b]["y"].(int)
		}
		return traits[a]["x"].(int) < traits[b]["x"].(int)
	})
	out["traits"] = traits
	out["sprites"] = sprites

	owned := []*paradox.Block{org.Initial}
	research := idx.defineNumber("NIndustrialOrganisation.DESIGN_TEAM_RESEARCH_BONUS") +
		modifierSum(owned, "military_industrial_organization_research_bonus")
	cost := idx.defineNumber("NIndustrialOrganisation.ASSIGN_DESIGN_TEAM_PP_COST_PER_DAY") +
		modifierSum(owned, "military_industrial_organization_design_team_assign_cost")
	out["research_bonus"] = round(research, 4)
	out["assign_cost"] = round(cost, 4)

	points, _ := idx.locValue("INDUSTRIAL_ORG_DETAIL_TRAIT_POINTS")
	points = mioValueSlot.ReplaceAllString(points, "§Y0§!")
	out["texts"] = map[string]any{
		"trait_points":      points,
		"attributes":        localise("INDUSTRIAL_ORG_DETAIL_MIO_MODIFIERS"),
		"equipment_bonuses": localise("INDUSTRIAL_ORG_DETAIL_EQUIPMENT_BONUSES"),
		"research_bonus":    localise("MODIFIER_MIO_RESEARCH_BONUS_MIO_CONTEXT"),
		"assign_cost":       localise("MODIFIER_MIO_DESIGN_TEAM_ASSIGN_COST"),
	}

	initial := map[string]any{}
	if org.Initial != nil {
		if key := ciScalar(org.Initial.Entries, "name"); key != "" {
			initial["label"] = localise(key)
		}
	}
	design := hasScriptBlock(org.Initial, "equipment_bonus")
	production := hasScriptBlock(org.Initial, "production_bonus")
	equipment := []map[string]any{}
	for _, group := range org.EquipmentGroups {
		icon := idx.equipmentGroupIcon(group)
		item := map[string]any{"group": group, "label": localise(group), "icon": icon, "icon_sprite": idx.spritePayload(icon), "design": 0, "production": 0}
		if idx.traitAppliesTo(org.Initial, group) {
			if design {
				item["design"] = 1
			}
			if production {
				item["production"] = 1
			}
		}
		equipment = append(equipment, item)
	}
	initial["equipment"] = equipment
	out["initial"] = initial
	return out
}

func orderedNumbers(block *paradox.Block) []map[string]any {
	out := []map[string]any{}
	for _, field := range block.Entries {
		if field == nil || field.Key == nil || field.Block != nil {
			continue
		}
		if value, ok := toFloat(scalarValue(field.Scalar)); ok {
			out = append(out, map[string]any{"key": *field.Key, "value": value})
		}
	}
	return out
}

func traitBlock(block *paradox.Block, key string) *paradox.Block {
	if block == nil {
		return nil
	}
	for _, field := range block.Entries {
		if field != nil && field.Key != nil && field.Block != nil && strings.EqualFold(*field.Key, key) {
			return field.Block
		}
	}
	return nil
}
