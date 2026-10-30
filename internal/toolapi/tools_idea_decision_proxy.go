package toolapi

import (
	"context"
	"fmt"
)

type createDecisionCategoryIn struct {
	ProjectID  int64  `json:"project_id"`
	CategoryID string `json:"category_id" jsonschema:"snake_case string — the in-game category id"`
	NameEn     string `json:"name_en,omitempty"`
	NameZh     string `json:"name_zh,omitempty"`
	DescEn     string `json:"desc_en,omitempty"`
	CountryTag string `json:"country_tag,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	Visible    string `json:"visible,omitempty" jsonschema:"raw HOI4 trigger text — category-level visibility gate"`
}

type createDecisionCategoryOut struct {
	CategoryDBID int64    `json:"category_db_id"`
	CategoryID   string   `json:"category_id"`
	Warnings     []string `json:"warnings,omitempty"`
}

func registerCreateDecisionCategory(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_decision_category",
		Description: "Create a decision category (the in-game tab). category_id = snake_case string. " +
			"Returns category_db_id (numeric) to pass into create_decision. Duplicate → 409.",
	}, func(ctx context.Context, _ *CallToolRequest, in createDecisionCategoryIn) (*CallToolResult, createDecisionCategoryOut, error) {
		body := map[string]any{
			"category_id": in.CategoryID,
			"name_en":     in.NameEn,
			"name_zh":     in.NameZh,
			"desc_en":     in.DescEn,
			"country_tag": in.CountryTag,
			"priority":    in.Priority,
			"visible":     in.Visible,
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/decisions/categories", in.ProjectID), body)
		if err != nil {
			return nil, createDecisionCategoryOut{}, err
		}
		out := createDecisionCategoryOut{
			CategoryID: stringField(resp, "category_id"),
			Warnings:   stringSliceField(resp, "warnings"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.CategoryDBID = int64(id)
		}
		return nil, out, nil
	})
}

type createDecisionIn struct {
	ProjectID      int64  `json:"project_id"`
	CategoryID     int64  `json:"category_id" jsonschema:"numeric category_db_id from create_decision_category (NOT the string)"`
	DecisionID     string `json:"decision_id" jsonschema:"snake_case; auto-suffixes on collision"`
	NameEn         string `json:"name_en,omitempty"`
	DescEn         string `json:"desc_en,omitempty"`
	Cost           int    `json:"cost,omitempty"`
	Available      string `json:"available,omitempty" jsonschema:"raw HOI4 trigger text"`
	Visible        string `json:"visible,omitempty" jsonschema:"raw HOI4 trigger text"`
	CompleteEffect string `json:"complete_effect,omitempty" jsonschema:"raw HOI4 effect text"`
	FireOnlyOnce   bool   `json:"fire_only_once,omitempty"`
	DaysRemove     int    `json:"days_remove,omitempty"`
	CountryTag     string `json:"country_tag,omitempty" jsonschema:"blank creates a global decision; nonblank on a created decision synthesizes allowed = { original_tag = TAG }"`
	AIWillDo       string `json:"ai_will_do,omitempty" jsonschema:"raw HOI4 ai_will_do block"`
}

type createDecisionOut struct {
	DecisionDBID int64    `json:"decision_db_id"`
	DecisionID   string   `json:"decision_id"`
	CategoryID   int64    `json:"category_id"`
	Warnings     []string `json:"warnings,omitempty"`
}

func registerCreateDecision(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_decision",
		Description: "Create a decision in a category. category_id = NUMERIC category_db_id from " +
			"create_decision_category (NOT the string). decision_id auto-suffixes on collision. " +
			"A blank country_tag creates a global decision; a nonblank country_tag on a created decision " +
			"synthesizes allowed = { original_tag = TAG }.",
	}, func(ctx context.Context, _ *CallToolRequest, in createDecisionIn) (*CallToolResult, createDecisionOut, error) {
		body := map[string]any{
			"category_id":     in.CategoryID,
			"decision_id":     in.DecisionID,
			"name_en":         in.NameEn,
			"desc_en":         in.DescEn,
			"cost":            in.Cost,
			"available":       in.Available,
			"visible":         in.Visible,
			"complete_effect": in.CompleteEffect,
			"fire_only_once":  in.FireOnlyOnce,
			"days_remove":     in.DaysRemove,
			"country_tag":     in.CountryTag,
			"ai_will_do":      in.AIWillDo,
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/decisions", in.ProjectID), body)
		if err != nil {
			return nil, createDecisionOut{}, err
		}
		out := createDecisionOut{
			DecisionID: stringField(resp, "decision_id"),
			Warnings:   stringSliceField(resp, "warnings"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.DecisionDBID = int64(id)
		}
		if cid, ok := resp["category_id"].(float64); ok {
			out.CategoryID = int64(cid)
		}
		return nil, out, nil
	})
}

type updateDecisionIn struct {
	ProjectID      int64   `json:"project_id"`
	DecisionDBID   int64   `json:"decision_db_id"`
	NameEn         *string `json:"name_en,omitempty"`
	DescEn         *string `json:"desc_en,omitempty"`
	Cost           *int    `json:"cost,omitempty"`
	Allowed        *string `json:"allowed,omitempty" jsonschema:"raw HOI4 trigger text controlling which countries can use this decision; omitted preserves the saved value"`
	Available      *string `json:"available,omitempty"`
	Visible        *string `json:"visible,omitempty"`
	CompleteEffect *string `json:"complete_effect,omitempty"`
	FireOnlyOnce   *bool   `json:"fire_only_once,omitempty"`
	DaysRemove     *int    `json:"days_remove,omitempty"`
	AIWillDo       *string `json:"ai_will_do,omitempty"`
	CategoryID     *int64  `json:"category_id,omitempty" jsonschema:"numeric; moves the decision to another category"`
}

type updateDecisionOut struct {
	DecisionDBID int64  `json:"decision_db_id"`
	DecisionID   string `json:"decision_id"`
}

func registerUpdateDecision(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_decision",
		Description: "Patch a decision by numeric decision_db_id. Only supplied fields change. " +
			"category_id (numeric) moves it to another category.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateDecisionIn) (*CallToolResult, updateDecisionOut, error) {
		body := map[string]any{}
		if in.NameEn != nil {
			body["name_en"] = *in.NameEn
		}
		if in.DescEn != nil {
			body["desc_en"] = *in.DescEn
		}
		if in.Cost != nil {
			body["cost"] = *in.Cost
		}
		if in.Allowed != nil {
			body["allowed"] = *in.Allowed
		}
		if in.Available != nil {
			body["available"] = *in.Available
		}
		if in.Visible != nil {
			body["visible"] = *in.Visible
		}
		if in.CompleteEffect != nil {
			body["complete_effect"] = *in.CompleteEffect
		}
		if in.FireOnlyOnce != nil {
			body["fire_only_once"] = *in.FireOnlyOnce
		}
		if in.DaysRemove != nil {
			body["days_remove"] = *in.DaysRemove
		}
		if in.AIWillDo != nil {
			body["ai_will_do"] = *in.AIWillDo
		}
		if in.CategoryID != nil {
			body["category_id"] = *in.CategoryID
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/decisions/%d", in.ProjectID, in.DecisionDBID), body)
		if err != nil {
			return nil, updateDecisionOut{}, err
		}
		out := updateDecisionOut{
			DecisionID: stringField(resp, "decision_id"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.DecisionDBID = int64(id)
		}
		return nil, out, nil
	})
}

type createIdeaIn struct {
	ProjectID         int64             `json:"project_id"`
	Name              string            `json:"name,omitempty" jsonschema:"the spirit's display name; required unless idea_id is given (the program no longer invents a placeholder)"`
	IdeaType          string            `json:"idea_type,omitempty" jsonschema:"default country"`
	IdeaID            string            `json:"idea_id,omitempty" jsonschema:"blank → derived from name; collisions auto-suffix"`
	CountryTag        string            `json:"country_tag,omitempty"`
	Modifier          map[string]any    `json:"modifier,omitempty" jsonschema:"raw HOI4 modifier dict, e.g. {'political_power_gain': 0.1}"`
	Picture           string            `json:"picture,omitempty" jsonschema:"vanilla sprite name with the GFX_idea_ prefix dropped"`
	RemovalCost       *int              `json:"removal_cost,omitempty" jsonschema:"default -1 (cannot be removed manually); 0 = removable for free; >0 = PP cost"`
	Allowed           string            `json:"allowed,omitempty" jsonschema:"raw HOI4 trigger text"`
	NameEn            string            `json:"name_en,omitempty"`
	NameZh            string            `json:"name_zh,omitempty"`
	DescEn            string            `json:"desc_en,omitempty"`
	DescZh            string            `json:"desc_zh,omitempty"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs   map[string]string `json:"description_locs,omitempty"`
	AutoAttachOnStart *bool             `json:"auto_attach_on_start,omitempty" jsonschema:"default true; set false for a reward spirit that should only be granted by a focus/event"`
}

type createIdeaOut struct {
	IdeaDBID          int64             `json:"idea_db_id"`
	IdeaID            string            `json:"idea_id"`
	Name              string            `json:"name"`
	Picture           string            `json:"picture"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs   map[string]string `json:"description_locs,omitempty"`
	AutoAttachOnStart bool              `json:"auto_attach_on_start"`
	Warnings          []string          `json:"warnings,omitempty"`
}

func registerCreateIdea(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_idea",
		Description: "Create a national spirit / idea. modifier = raw HOI4 modifier dict, e.g. " +
			"{'political_power_gain': 0.1}. Blank idea_id → derived from name; collisions auto-suffix. " +
			"Set name_zh/desc_zh or localisation maps for labels, picture for its sprite, and auto_attach_on_start=false for a focus/event reward spirit.",
	}, func(ctx context.Context, _ *CallToolRequest, in createIdeaIn) (*CallToolResult, createIdeaOut, error) {

		name := in.Name
		ideaType := in.IdeaType
		if ideaType == "" {
			ideaType = "country"
		}
		removalCost := -1
		if in.RemovalCost != nil {
			removalCost = *in.RemovalCost
		}
		modifier := in.Modifier
		if modifier == nil {
			modifier = map[string]any{}
		}

		body := map[string]any{
			"idea_id":      in.IdeaID,
			"name":         name,
			"idea_type":    ideaType,
			"country_tag":  in.CountryTag,
			"modifier":     modifier,
			"picture":      in.Picture,
			"removal_cost": removalCost,
			"allowed":      in.Allowed,
			"name_en":      in.NameEn,
			"name_zh":      in.NameZh,
			"desc_en":      in.DescEn,
			"desc_zh":      in.DescZh,
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}
		if in.AutoAttachOnStart != nil {
			body["auto_attach_on_start"] = *in.AutoAttachOnStart
		}
		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/ideas", in.ProjectID), body)
		if err != nil {
			return nil, createIdeaOut{}, err
		}
		out := createIdeaOut{
			IdeaID: stringField(resp, "idea_id"), Name: stringField(resp, "name"),
			Picture: stringField(resp, "picture"), NameEn: stringField(resp, "name_en"),
			NameZh: stringField(resp, "name_zh"), DescEn: stringField(resp, "desc_en"),
			DescZh:            stringField(resp, "desc_zh"),
			DisplayNameLocs:   nonNilStringMap(stringMapField(resp, "display_name_locs")),
			DescriptionLocs:   nonNilStringMap(stringMapField(resp, "description_locs")),
			AutoAttachOnStart: boolField(resp, "auto_attach_on_start"),
			Warnings:          stringSliceField(resp, "warnings"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.IdeaDBID = int64(id)
		}
		return nil, out, nil
	})
}

type updateIdeaIn struct {
	ProjectID         int64              `json:"project_id"`
	IdeaDBID          int64              `json:"idea_db_id"`
	Name              *string            `json:"name,omitempty"`
	IdeaID            *string            `json:"idea_id,omitempty"`
	IdeaType          *string            `json:"idea_type,omitempty"`
	Modifier          map[string]any     `json:"modifier,omitempty" jsonschema:"REPLACES the existing modifier dict when supplied"`
	Picture           *string            `json:"picture,omitempty"`
	RemovalCost       *int               `json:"removal_cost,omitempty"`
	Allowed           *string            `json:"allowed,omitempty"`
	NameEn            *string            `json:"name_en,omitempty"`
	NameZh            *string            `json:"name_zh,omitempty"`
	DescEn            *string            `json:"desc_en,omitempty"`
	DescZh            *string            `json:"desc_zh,omitempty"`
	DisplayNameLocs   *map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs   *map[string]string `json:"description_locs,omitempty"`
	AutoAttachOnStart *bool              `json:"auto_attach_on_start,omitempty"`
}

type updateIdeaOut struct {
	IdeaDBID          int64             `json:"idea_db_id"`
	IdeaID            string            `json:"idea_id"`
	Name              string            `json:"name"`
	Picture           string            `json:"picture"`
	NameEn            string            `json:"name_en"`
	NameZh            string            `json:"name_zh"`
	DescEn            string            `json:"desc_en"`
	DescZh            string            `json:"desc_zh"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs   map[string]string `json:"description_locs,omitempty"`
	AutoAttachOnStart bool              `json:"auto_attach_on_start"`
}

func registerUpdateIdea(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_idea",
		Description: "Patch a national spirit / idea by numeric idea_db_id. Only supplied fields change. modifier and localisation maps REPLACE; picture changes the sprite. auto_attach_on_start=false keeps a reward spirit out of the starting setup.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateIdeaIn) (*CallToolResult, updateIdeaOut, error) {
		body := map[string]any{}
		if in.Name != nil {
			body["name"] = *in.Name
		}
		if in.IdeaID != nil {
			body["idea_id"] = *in.IdeaID
		}
		if in.IdeaType != nil {
			body["idea_type"] = *in.IdeaType
		}
		if in.Modifier != nil {
			body["modifier"] = in.Modifier
		}
		if in.Picture != nil {
			body["picture"] = *in.Picture
		}
		if in.RemovalCost != nil {
			body["removal_cost"] = *in.RemovalCost
		}
		if in.Allowed != nil {
			body["allowed"] = *in.Allowed
		}
		if in.NameEn != nil {
			body["name_en"] = *in.NameEn
		}
		if in.NameZh != nil {
			body["name_zh"] = *in.NameZh
		}
		if in.DescEn != nil {
			body["desc_en"] = *in.DescEn
		}
		if in.DescZh != nil {
			body["desc_zh"] = *in.DescZh
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = *in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = *in.DescriptionLocs
		}
		if in.AutoAttachOnStart != nil {
			body["auto_attach_on_start"] = *in.AutoAttachOnStart
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/ideas/%d", in.ProjectID, in.IdeaDBID), body)
		if err != nil {
			return nil, updateIdeaOut{}, err
		}
		out := updateIdeaOut{
			IdeaID: stringField(resp, "idea_id"), Name: stringField(resp, "name"),
			Picture: stringField(resp, "picture"), NameEn: stringField(resp, "name_en"),
			NameZh: stringField(resp, "name_zh"), DescEn: stringField(resp, "desc_en"),
			DescZh:            stringField(resp, "desc_zh"),
			DisplayNameLocs:   nonNilStringMap(stringMapField(resp, "display_name_locs")),
			DescriptionLocs:   nonNilStringMap(stringMapField(resp, "description_locs")),
			AutoAttachOnStart: boolField(resp, "auto_attach_on_start"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.IdeaDBID = int64(id)
		}
		return nil, out, nil
	})
}
