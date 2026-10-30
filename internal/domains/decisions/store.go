package decisions

import (
	"fmt"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/project"
	"haganeworks/internal/workbench"
)

const (
	categoriesTable = "decision_categories"
	decisionsTable  = "decisions"
)

var (
	errCategoryMissing = apiutil.NotFoundError("Category not found")
	errDecisionMissing = apiutil.NotFoundError("Decision not found")
	errSourceMissing   = apiutil.NotFoundError("Source decision not found")

	errCategoryNotInProject = inputError("Category not found in this project")
)

func categoryConflict(categoryID string) error {
	return apiutil.ConflictError(fmt.Sprintf("category_id '%s' already exists in this project. Pick a unique id.", categoryID))
}

func decisionIDTaken(decisionID string) error {
	return apiutil.ConflictError(fmt.Sprintf("decision_id '%s' already exists in this project", decisionID))
}

func nullColumnError(sets map[string]any, columns ...string) error {
	for _, column := range columns {
		if value, ok := sets[column]; ok && value == nil {
			return inputError(column + " cannot be null")
		}
	}
	return nil
}

func categoryFromRow(row project.Row) DecisionCategory {
	return DecisionCategory{
		ID:                project.Int64(row["id"]),
		ProjectID:         project.Int64(row["project_id"]),
		CategoryID:        text(row["category_id"]),
		Icon:              text(row["icon"]),
		Picture:           text(row["picture"]),
		Priority:          intOf(row["priority"]),
		Allowed:           text(row["allowed"]),
		Visible:           text(row["visible"]),
		OnMapArea:         text(row["on_map_area"]),
		RawExtras:         anyMapOf(row["raw_extras"]),
		CountryTag:        text(row["country_tag"]),
		ScopeKind:         textOr(row["scope_kind"], "own"),
		SharedCountryTags: stringsOf(row["shared_country_tags"]),
		NameEn:            text(row["name_en"]),
		NameZh:            text(row["name_zh"]),
		DescEn:            text(row["desc_en"]),
		DescZh:            text(row["desc_zh"]),
		DisplayNameLocs:   stringMapOf(row["display_name_locs"]),
		DescriptionLocs:   stringMapOf(row["description_locs"]),
		Visibility:        textOr(row["visibility"], "visible"),
		CreatedAt:         isoNaive(row["created_at"]),
		UpdatedAt:         isoNaive(row["updated_at"]),
		Warnings:          []string{},
	}
}

func listCategories(p *project.Project, countryTag string) ([]DecisionCategory, error) {
	rows, err := p.Rows(categoriesTable)
	if err != nil {
		return nil, err
	}
	out := []DecisionCategory{}
	for _, row := range rows {
		if countryTag != "" && text(row["country_tag"]) != countryTag {
			continue
		}
		out = append(out, categoryFromRow(row))
	}
	if len(out) > 0 {
		decisions, err := p.Rows(decisionsTable)
		if err != nil {
			return nil, err
		}
		byCategory := map[int64]int64{}
		for _, d := range decisions {
			byCategory[project.Int64(d["category_id"])]++
		}
		for i := range out {
			out[i].DecisionCount = byCategory[out[i].ID]
		}
	}
	return out, nil
}

func getCategory(p *project.Project, id int64) (DecisionCategory, error) {
	rows, err := p.Rows(categoriesTable)
	if err != nil {
		return DecisionCategory{}, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) != id {
			continue
		}
		c := categoryFromRow(row)
		decisions, err := p.Rows(decisionsTable)
		if err != nil {
			return DecisionCategory{}, err
		}
		for _, d := range decisions {
			if project.Int64(d["category_id"]) == id {
				c.DecisionCount++
			}
		}
		return c, nil
	}
	return DecisionCategory{}, errCategoryMissing
}

func categoryIDTaken(tx *project.Tx, categoryID string, except int64) bool {
	for _, row := range tx.Rows(categoriesTable) {
		if text(row["category_id"]) == categoryID && project.Int64(row["id"]) != except {
			return true
		}
	}
	return false
}

func createCategory(p *project.Project, in categoryCreateInput, vanillaIDs func() map[string]bool) (DecisionCategory, error) {
	if err := canonicalCategoryCreate(&in); err != nil {
		return DecisionCategory{}, err
	}
	if in.CategoryID == nil {
		return DecisionCategory{}, inputError("category_id is required")
	}
	categoryID := strings.TrimSpace(*in.CategoryID)
	if in.DisplayNameLocs == nil {
		in.DisplayNameLocs = map[string]string{}
	}
	if in.DescriptionLocs == nil {
		in.DescriptionLocs = map[string]string{}
	}
	var shared any
	if in.SharedCountryTags != nil {
		shared = stored(in.SharedCountryTags)
	}
	var id int64
	err := p.Write([]string{categoriesTable}, func(tx *project.Tx) error {
		if categoryIDTaken(tx, categoryID, 0) {
			return categoryConflict(categoryID)
		}
		row := tx.Insert(categoriesTable, project.Row{
			"category_id": categoryID, "icon": in.Icon, "picture": in.Picture,
			"priority": int64(in.Priority), "allowed": in.Allowed, "visible": in.Visible,
			"on_map_area": in.OnMapArea, "country_tag": in.CountryTag,
			"scope_kind":          workbench.StrDefault(in.ScopeKind, "own"),
			"shared_country_tags": shared, "name_en": in.NameEn, "name_zh": in.NameZh,
			"desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs": stored(in.DisplayNameLocs), "description_locs": stored(in.DescriptionLocs),
			"visibility": workbench.StrDefault(in.Visibility, "visible"),
			"raw_extras": map[string]any{},
		})
		id = project.Int64(row["id"])
		return nil
	})
	if err != nil {
		return DecisionCategory{}, err
	}
	c, err := getCategory(p, id)
	if err != nil {
		return DecisionCategory{}, err
	}
	if w := vanillaCategoryWarning(categoryID, in.CountryTag, "created", vanillaIDs); w != "" {
		c.Warnings = []string{w}
	}
	return c, nil
}

func vanillaCategoryWarning(categoryID, countryTag, source string, vanillaIDs func() map[string]bool) string {
	if categoryID == "" || source == "imported" || vanillaIDs == nil || !vanillaIDs()[categoryID] {
		return ""
	}
	prefix := "MOD_"
	if countryTag != "" {
		prefix = strings.ToUpper(countryTag) + "_"
	}
	suggested := categoryID
	if !strings.HasPrefix(categoryID, prefix) {
		suggested = prefix + categoryID
	}
	return fmt.Sprintf("'%s' 跟 vanilla decision category 同名 · HOI4 mod 加载会 silent override vanilla · 建议改名 '%s'(同 country TAG prefix 防 collision)", categoryID, suggested)
}

func updateCategory(p *project.Project, id int64, patch categoryPatch) (DecisionCategory, error) {
	if err := canonicalCategoryPatch(&patch); err != nil {
		return DecisionCategory{}, err
	}
	if _, err := getCategory(p, id); err != nil {
		return DecisionCategory{}, err
	}
	if patch.CategoryID.Set && patch.CategoryID.Value != nil {
		v := strings.TrimSpace(*patch.CategoryID.Value)
		patch.CategoryID.Value = &v
	}
	sets := map[string]any{}
	setOptional(sets, "category_id", patch.CategoryID)
	setOptional(sets, "icon", patch.Icon)
	setOptional(sets, "picture", patch.Picture)
	setOptional(sets, "priority", patch.Priority)
	setOptional(sets, "allowed", patch.Allowed)
	setOptional(sets, "visible", patch.Visible)
	setOptional(sets, "on_map_area", patch.OnMapArea)
	setOptional(sets, "country_tag", patch.CountryTag)
	setOptional(sets, "scope_kind", patch.ScopeKind)
	setOptional(sets, "shared_country_tags", patch.SharedCountryTags)
	setOptional(sets, "name_en", patch.NameEn)
	setOptional(sets, "name_zh", patch.NameZh)
	setOptional(sets, "desc_en", patch.DescEn)
	setOptional(sets, "desc_zh", patch.DescZh)
	setOptional(sets, "display_name_locs", patch.DisplayNameLocs)
	setOptional(sets, "description_locs", patch.DescriptionLocs)
	setOptional(sets, "visibility", patch.Visibility)
	if err := nullColumnError(sets, "category_id", "scope_kind", "display_name_locs", "description_locs", "visibility"); err != nil {
		return DecisionCategory{}, err
	}
	if len(sets) > 0 {
		err := p.Write([]string{categoriesTable}, func(tx *project.Tx) error {
			if categoryID, ok := sets["category_id"].(string); ok && categoryIDTaken(tx, categoryID, id) {
				return categoryConflict(categoryID)
			}
			_, err := tx.Update(categoriesTable, id, apply(sets))
			return err
		})
		if err != nil {
			return DecisionCategory{}, err
		}
	}
	return getCategory(p, id)
}

func deleteCategory(p *project.Project, id int64) error {
	return p.Write([]string{categoriesTable, decisionsTable}, func(tx *project.Tx) error {
		if tx.Delete(categoriesTable, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return errCategoryMissing
		}
		tx.Delete(decisionsTable, func(row project.Row) bool { return project.Int64(row["category_id"]) == id })
		return nil
	})
}

func decisionFromRow(row project.Row) Decision {
	return Decision{
		ID:                 project.Int64(row["id"]),
		ProjectID:          project.Int64(row["project_id"]),
		CategoryID:         project.Int64(row["category_id"]),
		DecisionID:         text(row["decision_id"]),
		Icon:               text(row["icon"]),
		Allowed:            text(row["allowed"]),
		Available:          text(row["available"]),
		Visible:            text(row["visible"]),
		Cost:               intOf(row["cost"]),
		FireOnlyOnce:       boolOf(row["fire_only_once"]),
		CancelIfNotVisible: boolOf(row["cancel_if_not_visible"]),
		DaysRemove:         intOf(row["days_remove"]),
		DaysReEnable:       intOf(row["days_re_enable"]),
		CompleteEffect:     text(row["complete_effect"]),
		RemoveEffect:       text(row["remove_effect"]),
		CancelTrigger:      text(row["cancel_trigger"]),
		CancelEffect:       text(row["cancel_effect"]),
		Modifier:           text(row["modifier"]),
		Activation:         text(row["activation"]),
		TimeoutEffect:      text(row["timeout_effect"]),
		DaysMissionTimeout: intOf(row["days_mission_timeout"]),
		SelectableMission:  boolOf(row["selectable_mission"]),
		IsGood:             boolOf(row["is_good"]),
		TargetedModifier:   anyMapOf(row["targeted_modifier"]),
		Targets:            stringsOrEmpty(row["targets"]),
		TargetArray:        stringsOrEmpty(row["target_array"]),
		TargetRootTrigger:  text(row["target_root_trigger"]),
		TargetTrigger:      text(row["target_trigger"]),
		StateTarget:        text(row["state_target"]),
		HighlightStates:    stringsOrEmpty(row["highlight_states"]),
		OnMapMode:          text(row["on_map_mode"]),
		Priority:           intOf(row["priority"]),
		CustomCostTrigger:  text(row["custom_cost_trigger"]),
		CustomCostText:     text(row["custom_cost_text"]),
		WarWithOnAdd:       text(row["war_with_on_add"]),
		WarWithOnRemove:    text(row["war_with_on_remove"]),
		AIWillDo:           text(row["ai_will_do"]),
		RawExtras:          anyMapOf(row["raw_extras"]),
		CountryTag:         text(row["country_tag"]),
		ScopeKind:          textOr(row["scope_kind"], "own"),
		SharedCountryTags:  stringsOf(row["shared_country_tags"]),
		NameEn:             text(row["name_en"]),
		NameZh:             text(row["name_zh"]),
		DescEn:             text(row["desc_en"]),
		DescZh:             text(row["desc_zh"]),
		DisplayNameLocs:    stringMapOf(row["display_name_locs"]),
		DescriptionLocs:    stringMapOf(row["description_locs"]),
		EffectTooltipLocs:  stringMapOf(row["effect_tooltip_locs"]),
		Source:             textOr(row["source"], "created"),
		CreatedAt:          isoNaive(row["created_at"]),
		UpdatedAt:          isoNaive(row["updated_at"]),
		Warnings:           []string{},
	}
}

func listDecisions(p *project.Project, categoryID *int64, countryTag string) ([]Decision, error) {
	rows, err := p.Rows(decisionsTable)
	if err != nil {
		return nil, err
	}
	out := []Decision{}
	for _, row := range rows {
		if categoryID != nil && project.Int64(row["category_id"]) != *categoryID {
			continue
		}
		if countryTag != "" && text(row["country_tag"]) != countryTag {
			continue
		}
		out = append(out, decisionFromRow(row))
	}
	return out, nil
}

func findDecisionRow(p *project.Project, id int64) (project.Row, error) {
	rows, err := p.Rows(decisionsTable)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if project.Int64(row["id"]) == id {
			return row, nil
		}
	}
	return nil, errDecisionMissing
}

func getDecision(p *project.Project, id int64) (Decision, error) {
	row, err := findDecisionRow(p, id)
	if err != nil {
		return Decision{}, err
	}
	return decisionFromRow(row), nil
}

func decisionByScriptID(p *project.Project, decisionID string) (Decision, error) {
	decisionID = strings.TrimSpace(decisionID)
	rows, err := p.Rows(decisionsTable)
	if err != nil {
		return Decision{}, err
	}
	for _, row := range rows {
		if text(row["decision_id"]) == decisionID {
			return decisionFromRow(row), nil
		}
	}
	return Decision{}, errDecisionMissing
}

func categoryExists(tx *project.Tx, id int64) bool { return tx.Find(categoriesTable, id) != nil }

func decisionIDExists(tx *project.Tx, decisionID string, except int64) bool {
	for _, row := range tx.Rows(decisionsTable) {
		if text(row["decision_id"]) == decisionID && project.Int64(row["id"]) != except {
			return true
		}
	}
	return false
}

func createDecision(p *project.Project, in decisionCreateInput) (Decision, error) {
	var id int64
	var warnings []string
	err := p.Write([]string{categoriesTable, decisionsTable}, func(tx *project.Tx) error {
		if !categoryExists(tx, *in.CategoryID) {
			return errCategoryNotInProject
		}
		taken := map[string]bool{}
		for _, row := range tx.Rows(decisionsTable) {
			taken[text(row["decision_id"])] = true
		}
		original := strings.TrimSpace(*in.DecisionID)
		decisionID := original
		suffix := 0
		for taken[decisionID] {
			suffix++
			if suffix > 1000 {
				return fmt.Errorf("decision_id auto-suffix exhausted 1000 attempts starting at '%s' · pick manually.", original)
			}
			decisionID = fmt.Sprintf("%s_%d", original, suffix)
		}
		warnings = []string{}
		if suffix > 0 {
			warnings = append(warnings, fmt.Sprintf("decision_id '%s' collided · auto-suffixed to '%s'.", original, decisionID))
		}
		var shared any
		if in.SharedCountryTags != nil {
			shared = stored(in.SharedCountryTags)
		}
		row := tx.Insert(decisionsTable, project.Row{
			"category_id": *in.CategoryID, "decision_id": decisionID, "icon": in.Icon,
			"allowed": in.Allowed, "available": in.Available, "visible": in.Visible,
			"cost": int64(in.Cost), "fire_only_once": in.FireOnlyOnce,
			"cancel_if_not_visible": in.CancelIfNotVisible, "days_remove": int64(in.DaysRemove),
			"days_re_enable": int64(in.DaysReEnable), "complete_effect": in.CompleteEffect,
			"remove_effect": in.RemoveEffect, "cancel_trigger": in.CancelTrigger,
			"cancel_effect": in.CancelEffect, "modifier": in.Modifier, "activation": in.Activation,
			"timeout_effect": in.TimeoutEffect, "days_mission_timeout": int64(in.DaysMissionTimeout),
			"selectable_mission": in.SelectableMission, "is_good": in.IsGood,
			"targeted_modifier": in.TargetedModifier, "targets": stored(in.Targets),
			"target_array": stored(in.TargetArray), "target_root_trigger": in.TargetRootTrigger,
			"target_trigger": in.TargetTrigger, "state_target": in.StateTarget,
			"highlight_states": stored(in.HighlightStates), "on_map_mode": in.OnMapMode,
			"priority": int64(in.Priority), "custom_cost_trigger": in.CustomCostTrigger,
			"custom_cost_text": in.CustomCostText, "war_with_on_add": in.WarWithOnAdd,
			"war_with_on_remove": in.WarWithOnRemove, "ai_will_do": in.AIWillDo,
			"country_tag": in.CountryTag, "scope_kind": workbench.StrDefault(in.ScopeKind, "own"),
			"shared_country_tags": shared, "name_en": in.NameEn, "name_zh": in.NameZh,
			"desc_en": in.DescEn, "desc_zh": in.DescZh,
			"display_name_locs":   stored(in.DisplayNameLocs),
			"description_locs":    stored(in.DescriptionLocs),
			"effect_tooltip_locs": stored(in.EffectTooltipLocs),
			"source":              workbench.StrDefault(in.Source, "created"),
			"raw_extras":          map[string]any{}, "source_snapshot": nil, "source_file": "",
			"has_obs": false,
		})
		id = project.Int64(row["id"])
		return nil
	})
	if err != nil {
		return Decision{}, err
	}
	d, err := getDecision(p, id)
	if err != nil {
		return Decision{}, err
	}
	d.Warnings = warnings
	return d, nil
}

func updateDecision(p *project.Project, id int64, patch decisionPatch) (Decision, error) {
	if err := canonicalDecisionPatch(&patch); err != nil {
		return Decision{}, err
	}
	current, err := findDecisionRow(p, id)
	if err != nil {
		return Decision{}, err
	}
	if patch.DecisionID.Set && patch.DecisionID.Value != nil {
		v := strings.TrimSpace(*patch.DecisionID.Value)
		patch.DecisionID.Value = &v
	}
	sets := map[string]any{}
	setOptional(sets, "category_id", patch.CategoryID)
	setOptional(sets, "decision_id", patch.DecisionID)
	setOptional(sets, "icon", patch.Icon)
	setOptional(sets, "allowed", patch.Allowed)
	setOptional(sets, "available", patch.Available)
	setOptional(sets, "visible", patch.Visible)
	setOptional(sets, "cost", patch.Cost)
	setOptional(sets, "fire_only_once", patch.FireOnlyOnce)
	setOptional(sets, "cancel_if_not_visible", patch.CancelIfNotVisible)
	setOptional(sets, "days_remove", patch.DaysRemove)
	setOptional(sets, "days_re_enable", patch.DaysReEnable)
	setOptional(sets, "complete_effect", patch.CompleteEffect)
	setOptional(sets, "remove_effect", patch.RemoveEffect)
	setOptional(sets, "cancel_trigger", patch.CancelTrigger)
	setOptional(sets, "cancel_effect", patch.CancelEffect)
	setOptional(sets, "modifier", patch.Modifier)
	setOptional(sets, "activation", patch.Activation)
	setOptional(sets, "timeout_effect", patch.TimeoutEffect)
	setOptional(sets, "days_mission_timeout", patch.DaysMissionTimeout)
	setOptional(sets, "selectable_mission", patch.SelectableMission)
	setOptional(sets, "is_good", patch.IsGood)
	setOptional(sets, "targeted_modifier", patch.TargetedModifier)
	setOptional(sets, "targets", patch.Targets)
	setOptional(sets, "target_array", patch.TargetArray)
	setOptional(sets, "target_root_trigger", patch.TargetRootTrigger)
	setOptional(sets, "target_trigger", patch.TargetTrigger)
	setOptional(sets, "state_target", patch.StateTarget)
	setOptional(sets, "highlight_states", patch.HighlightStates)
	setOptional(sets, "on_map_mode", patch.OnMapMode)
	setOptional(sets, "priority", patch.Priority)
	setOptional(sets, "custom_cost_trigger", patch.CustomCostTrigger)
	setOptional(sets, "custom_cost_text", patch.CustomCostText)
	setOptional(sets, "war_with_on_add", patch.WarWithOnAdd)
	setOptional(sets, "war_with_on_remove", patch.WarWithOnRemove)
	setOptional(sets, "ai_will_do", patch.AIWillDo)
	setOptional(sets, "country_tag", patch.CountryTag)
	setOptional(sets, "scope_kind", patch.ScopeKind)
	setOptional(sets, "shared_country_tags", patch.SharedCountryTags)
	setOptional(sets, "name_en", patch.NameEn)
	setOptional(sets, "name_zh", patch.NameZh)
	setOptional(sets, "desc_en", patch.DescEn)
	setOptional(sets, "desc_zh", patch.DescZh)
	setOptional(sets, "display_name_locs", patch.DisplayNameLocs)
	setOptional(sets, "description_locs", patch.DescriptionLocs)
	setOptional(sets, "effect_tooltip_locs", patch.EffectTooltipLocs)
	setOptional(sets, "raw_extras", patch.RawExtras)

	flipToCreated := text(current["source"]) == "imported"
	if patch.Source.Set && patch.Source.Value != nil {
		flipToCreated = *patch.Source.Value == "imported"
	}
	if !flipToCreated {
		setOptional(sets, "source", patch.Source)
	} else {
		sets["source"] = "created"
	}
	if err := nullColumnError(sets, "category_id", "decision_id", "scope_kind", "source",
		"display_name_locs", "description_locs", "effect_tooltip_locs"); err != nil {
		return Decision{}, err
	}
	err = p.Write([]string{categoriesTable, decisionsTable}, func(tx *project.Tx) error {
		if decisionID, ok := sets["decision_id"].(string); ok && decisionIDExists(tx, decisionID, id) {
			return decisionIDTaken(decisionID)
		}
		if categoryID, ok := sets["category_id"].(int64); ok && !categoryExists(tx, categoryID) {
			return errCategoryNotInProject
		}
		if len(sets) == 0 {
			return nil
		}
		_, err := tx.Update(decisionsTable, id, apply(sets))
		return err
	})
	if err != nil {
		return Decision{}, err
	}
	return getDecision(p, id)
}

func deleteDecision(p *project.Project, id int64) error {
	return p.Write([]string{decisionsTable}, func(tx *project.Tx) error {
		if tx.Delete(decisionsTable, func(row project.Row) bool { return project.Int64(row["id"]) == id }) == 0 {
			return errDecisionMissing
		}
		return nil
	})
}

func forkDecision(p *project.Project, id int64, newDecisionID string) (Decision, error) {
	var newID int64
	err := p.Write([]string{decisionsTable}, func(tx *project.Tx) error {
		source := tx.Find(decisionsTable, id)
		if source == nil {
			return errSourceMissing
		}
		if newDecisionID == text(source["decision_id"]) {
			return inputError("new_decision_id must differ from the original")
		}
		if decisionIDExists(tx, newDecisionID, 0) {
			return decisionIDTaken(newDecisionID)
		}
		copied := project.Row{}
		for column, value := range source {
			switch column {
			case "id", "created_at", "updated_at":
				continue
			}
			copied[column] = value
		}
		copied["decision_id"] = newDecisionID
		copied["source"] = "created"
		newID = project.Int64(tx.Insert(decisionsTable, copied)["id"])
		return nil
	})
	if err != nil {
		return Decision{}, err
	}
	return getDecision(p, newID)
}
