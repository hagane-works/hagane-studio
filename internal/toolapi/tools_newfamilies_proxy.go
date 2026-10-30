package toolapi

import (
	"context"
	"fmt"
)

type createOpinionModifierIn struct {
	ProjectID  int64    `json:"project_id"`
	ModifierID string   `json:"modifier_id" jsonschema:"unique within the project, e.g. war_ally_support"`
	CountryTag string   `json:"country_tag,omitempty" jsonschema:"3-letter tag this modifier applies FROM; blank -> project's primary country"`
	Value      *float64 `json:"value,omitempty" jsonschema:"opinion point delta"`
	Decay      *float64 `json:"decay,omitempty" jsonschema:"per-day decay of the current opinion value"`
	Days       *int     `json:"days,omitempty" jsonschema:"duration in days (pick one of days/months/years)"`
	Months     *int     `json:"months,omitempty" jsonschema:"duration in months"`
	Years      *int     `json:"years,omitempty" jsonschema:"duration in years"`
	MinTrust   *float64 `json:"min_trust,omitempty"`
	MaxTrust   *float64 `json:"max_trust,omitempty"`
	Trade      *bool    `json:"trade,omitempty" jsonschema:"whether this opinion modifier requires an active trade agreement"`
}

type updateOpinionModifierIn struct {
	ProjectID         int64    `json:"project_id"`
	OpinionModifierID int64    `json:"opinion_modifier_id" jsonschema:"id returned by create_opinion_modifier"`
	ModifierID        *string  `json:"modifier_id,omitempty"`
	CountryTag        *string  `json:"country_tag,omitempty"`
	Value             *float64 `json:"value,omitempty"`
	Decay             *float64 `json:"decay,omitempty"`
	Days              *int     `json:"days,omitempty"`
	Months            *int     `json:"months,omitempty"`
	Years             *int     `json:"years,omitempty"`
	MinTrust          *float64 `json:"min_trust,omitempty"`
	MaxTrust          *float64 `json:"max_trust,omitempty"`
	Trade             *bool    `json:"trade,omitempty"`
}

type opinionModifierOut struct {
	OpinionModifierID int64  `json:"opinion_modifier_id"`
	ModifierID        string `json:"modifier_id"`
	CountryTag        string `json:"country_tag"`
}

func registerCreateOpinionModifier(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_opinion_modifier",
		Description: "Create a country-to-country opinion modifier (a durable/decaying opinion bonus or " +
			"penalty). country_tag = who the modifier applies FROM (target country is set via the modifier's " +
			"own country context in the web UI's active-country selector). value = opinion point delta; " +
			"decay = per-day change toward 0; days/months/years = duration (use at most one).",
	}, func(ctx context.Context, _ *CallToolRequest, in createOpinionModifierIn) (*CallToolResult, opinionModifierOut, error) {
		body := map[string]any{"modifier_id": in.ModifierID, "country_tag": in.CountryTag}
		if in.Value != nil {
			body["value"] = *in.Value
		}
		if in.Decay != nil {
			body["decay"] = *in.Decay
		}
		if in.Days != nil {
			body["days"] = *in.Days
		}
		if in.Months != nil {
			body["months"] = *in.Months
		}
		if in.Years != nil {
			body["years"] = *in.Years
		}
		if in.MinTrust != nil {
			body["min_trust"] = *in.MinTrust
		}
		if in.MaxTrust != nil {
			body["max_trust"] = *in.MaxTrust
		}
		if in.Trade != nil {
			body["trade"] = *in.Trade
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/opinion-modifiers", in.ProjectID), body)
		if err != nil {
			return nil, opinionModifierOut{}, err
		}
		out := opinionModifierOut{
			OpinionModifierID: int64Field(resp, "id"),
			ModifierID:        stringField(resp, "modifier_id"),
			CountryTag:        stringField(resp, "country_tag"),
		}
		return nil, out, nil
	})
}

func registerUpdateOpinionModifier(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_opinion_modifier",
		Description: "Edit an opinion modifier in place. Only supplied fields change.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateOpinionModifierIn) (*CallToolResult, opinionModifierOut, error) {
		body := map[string]any{}
		if in.ModifierID != nil {
			body["modifier_id"] = *in.ModifierID
		}
		if in.CountryTag != nil {
			body["country_tag"] = *in.CountryTag
		}
		if in.Value != nil {
			body["value"] = *in.Value
		}
		if in.Decay != nil {
			body["decay"] = *in.Decay
		}
		if in.Days != nil {
			body["days"] = *in.Days
		}
		if in.Months != nil {
			body["months"] = *in.Months
		}
		if in.Years != nil {
			body["years"] = *in.Years
		}
		if in.MinTrust != nil {
			body["min_trust"] = *in.MinTrust
		}
		if in.MaxTrust != nil {
			body["max_trust"] = *in.MaxTrust
		}
		if in.Trade != nil {
			body["trade"] = *in.Trade
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/opinion-modifiers/%d", in.ProjectID, in.OpinionModifierID), body)
		if err != nil {
			return nil, opinionModifierOut{}, err
		}
		out := opinionModifierOut{
			OpinionModifierID: int64Field(resp, "id"),
			ModifierID:        stringField(resp, "modifier_id"),
			CountryTag:        stringField(resp, "country_tag"),
		}
		return nil, out, nil
	})
}

type createModifierDefinitionIn struct {
	ProjectID  int64    `json:"project_id"`
	ModifierID string   `json:"modifier_id" jsonschema:"must start with a letter or underscore, then letters/digits/underscores only"`
	ColorType  string   `json:"color_type,omitempty" jsonschema:"good | bad | neutral; default bad"`
	ValueType  string   `json:"value_type,omitempty" jsonschema:"number | percentage | percentage_in_hundred | yes_no; default number"`
	Precision  *int     `json:"precision,omitempty" jsonschema:"decimal places, 0-3; default 2"`
	Postfix    string   `json:"postfix,omitempty" jsonschema:"none | days | hours | daily; default none"`
	Categories []string `json:"categories,omitempty" jsonschema:"e.g. all/country/state/army/naval/air/politics/ai/autonomy/... ; default [all]"`
}

type updateModifierDefinitionIn struct {
	ProjectID            int64     `json:"project_id"`
	ModifierDefinitionID int64     `json:"modifier_definition_id" jsonschema:"id returned by create_modifier_definition"`
	ModifierID           *string   `json:"modifier_id,omitempty"`
	ColorType            *string   `json:"color_type,omitempty"`
	ValueType            *string   `json:"value_type,omitempty"`
	Precision            *int      `json:"precision,omitempty"`
	Postfix              *string   `json:"postfix,omitempty"`
	Categories           *[]string `json:"categories,omitempty"`
}

type modifierDefinitionOut struct {
	ModifierDefinitionID int64  `json:"modifier_definition_id"`
	ModifierID           string `json:"modifier_id"`
}

func registerCreateModifierDefinition(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_modifier_definition",
		Description: "Define how a custom modifier id (used in country/state/unit-leader modifier blocks " +
			"elsewhere) renders in HOI4's tooltips: color (good/bad/neutral), value formatting " +
			"(number/percentage/percentage_in_hundred/yes_no), decimal precision, unit postfix " +
			"(none/days/hours/daily), and which modifier categories it belongs to.",
	}, func(ctx context.Context, _ *CallToolRequest, in createModifierDefinitionIn) (*CallToolResult, modifierDefinitionOut, error) {
		body := map[string]any{"modifier_id": in.ModifierID}
		if in.ColorType != "" {
			body["color_type"] = in.ColorType
		}
		if in.ValueType != "" {
			body["value_type"] = in.ValueType
		}
		if in.Precision != nil {
			body["precision"] = *in.Precision
		}
		if in.Postfix != "" {
			body["postfix"] = in.Postfix
		}
		if len(in.Categories) > 0 {
			body["categories"] = in.Categories
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/modifier-definitions", in.ProjectID), body)
		if err != nil {
			return nil, modifierDefinitionOut{}, err
		}
		out := modifierDefinitionOut{
			ModifierDefinitionID: int64Field(resp, "id"),
			ModifierID:           stringField(resp, "modifier_id"),
		}
		return nil, out, nil
	})
}

func registerUpdateModifierDefinition(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_modifier_definition",
		Description: "Edit a modifier definition in place. Only supplied fields change.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateModifierDefinitionIn) (*CallToolResult, modifierDefinitionOut, error) {
		body := map[string]any{}
		if in.ModifierID != nil {
			body["modifier_id"] = *in.ModifierID
		}
		if in.ColorType != nil {
			body["color_type"] = *in.ColorType
		}
		if in.ValueType != nil {
			body["value_type"] = *in.ValueType
		}
		if in.Precision != nil {
			body["precision"] = *in.Precision
		}
		if in.Postfix != nil {
			body["postfix"] = *in.Postfix
		}
		if in.Categories != nil {
			body["categories"] = *in.Categories
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/modifier-definitions/%d", in.ProjectID, in.ModifierDefinitionID), body)
		if err != nil {
			return nil, modifierDefinitionOut{}, err
		}
		out := modifierDefinitionOut{
			ModifierDefinitionID: int64Field(resp, "id"),
			ModifierID:           stringField(resp, "modifier_id"),
		}
		return nil, out, nil
	})
}

type createWargoalIn struct {
	ProjectID   int64             `json:"project_id"`
	WargoalKey  string            `json:"wargoal_key" jsonschema:"single HOI4 token, e.g. take_state_wg"`
	WarName     string            `json:"war_name,omitempty" jsonschema:"loc key or literal string shown as the war's name"`
	SafeScalars map[string]string `json:"safe_scalars,omitempty" jsonschema:"keys: generate_base_cost, generate_per_state_cost, take_states_limit, take_states_cost, puppet_cost, force_government_cost, expire, threat, take_states_threat_factor"`
	Allowed     string            `json:"allowed,omitempty" jsonschema:"raw HOI4 trigger body without outer braces, e.g. always = yes"`
	Available   string            `json:"available,omitempty" jsonschema:"raw trigger body without outer braces gating when this wargoal can be added to a war"`
	TakeStates  string            `json:"take_states,omitempty" jsonschema:"raw trigger body without outer braces gating which states this wargoal can take"`
	Puppet      string            `json:"puppet,omitempty" jsonschema:"raw trigger body without outer braces gating puppet-creation use of this wargoal"`
	Liberate    string            `json:"liberate,omitempty" jsonschema:"raw trigger body without outer braces gating liberation use of this wargoal"`
}

type updateWargoalIn struct {
	ProjectID   int64              `json:"project_id"`
	WargoalID   int64              `json:"wargoal_id" jsonschema:"id returned by create_wargoal"`
	WargoalKey  *string            `json:"wargoal_key,omitempty"`
	WarName     *string            `json:"war_name,omitempty"`
	SafeScalars *map[string]string `json:"safe_scalars,omitempty"`
	Allowed     *string            `json:"allowed,omitempty"`
	Available   *string            `json:"available,omitempty"`
	TakeStates  *string            `json:"take_states,omitempty"`
	Puppet      *string            `json:"puppet,omitempty"`
	Liberate    *string            `json:"liberate,omitempty"`
}

type wargoalOut struct {
	WargoalID  int64  `json:"wargoal_id"`
	WargoalKey string `json:"wargoal_key"`
}

func registerCreateWargoal(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_wargoal",
		Description: "Create a wargoal type (a war-goal players/AI can pursue in a war, e.g. take a state, " +
			"install a puppet, liberate a country). allowed/available/take_states/puppet/liberate are HOI4 " +
			"trigger bodies without outer braces (e.g. always = yes); the exporter supplies the braces. safe_scalars carries the numeric cost/threat " +
			"overrides (generate_base_cost, take_states_cost, puppet_cost, expire, threat, etc.).",
	}, func(ctx context.Context, _ *CallToolRequest, in createWargoalIn) (*CallToolResult, wargoalOut, error) {
		body := map[string]any{"wargoal_key": in.WargoalKey}
		if in.WarName != "" {
			body["war_name"] = in.WarName
		}
		if len(in.SafeScalars) > 0 {
			body["safe_scalars"] = in.SafeScalars
		}
		if in.Allowed != "" {
			body["allowed"] = in.Allowed
		}
		if in.Available != "" {
			body["available"] = in.Available
		}
		if in.TakeStates != "" {
			body["take_states"] = in.TakeStates
		}
		if in.Puppet != "" {
			body["puppet"] = in.Puppet
		}
		if in.Liberate != "" {
			body["liberate"] = in.Liberate
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/wargoals", in.ProjectID), body)
		if err != nil {
			return nil, wargoalOut{}, err
		}
		out := wargoalOut{
			WargoalID:  int64Field(resp, "id"),
			WargoalKey: stringField(resp, "wargoal_key"),
		}
		return nil, out, nil
	})
}

func registerUpdateWargoal(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_wargoal",
		Description: "Edit a wargoal type in place. Only supplied fields change.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateWargoalIn) (*CallToolResult, wargoalOut, error) {
		body := map[string]any{}
		if in.WargoalKey != nil {
			body["wargoal_key"] = *in.WargoalKey
		}
		if in.WarName != nil {
			body["war_name"] = *in.WarName
		}
		if in.SafeScalars != nil {
			body["safe_scalars"] = *in.SafeScalars
		}
		if in.Allowed != nil {
			body["allowed"] = *in.Allowed
		}
		if in.Available != nil {
			body["available"] = *in.Available
		}
		if in.TakeStates != nil {
			body["take_states"] = *in.TakeStates
		}
		if in.Puppet != nil {
			body["puppet"] = *in.Puppet
		}
		if in.Liberate != nil {
			body["liberate"] = *in.Liberate
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/wargoals/%d", in.ProjectID, in.WargoalID), body)
		if err != nil {
			return nil, wargoalOut{}, err
		}
		out := wargoalOut{
			WargoalID:  int64Field(resp, "id"),
			WargoalKey: stringField(resp, "wargoal_key"),
		}
		return nil, out, nil
	})
}

type createAiStrategyPlanIn struct {
	ProjectID         int64    `json:"project_id"`
	PlanID            string   `json:"plan_id" jsonschema:"unique within the project, one safe script token"`
	Name              string   `json:"name,omitempty"`
	Desc              string   `json:"desc,omitempty"`
	Allowed           []string `json:"allowed,omitempty" jsonschema:"trigger blocks gating when this plan is considered; each element is a full { ... } block"`
	Abort             []string `json:"abort,omitempty" jsonschema:"trigger blocks gating when this plan is dropped; each a full { ... } block"`
	AiNationalFocuses []string `json:"ai_national_focuses,omitempty" jsonschema:"blocks listing the national focus ids this plan drives the AI toward"`
	AiStrategy        []string `json:"ai_strategy,omitempty" jsonschema:"ai_strategy directive blocks (e.g. type = alliance, id = TAG, value = 1)"`
	Weight            []string `json:"weight,omitempty" jsonschema:"factor blocks controlling how strongly the AI favors this plan, e.g. {\n\tfactor = 1\n}"`
}

type updateAiStrategyPlanIn struct {
	ProjectID         int64     `json:"project_id"`
	AiStrategyPlanID  int64     `json:"ai_strategy_plan_id" jsonschema:"id returned by create_ai_strategy_plan"`
	PlanID            *string   `json:"plan_id,omitempty"`
	Name              *string   `json:"name,omitempty"`
	Desc              *string   `json:"desc,omitempty"`
	Allowed           *[]string `json:"allowed,omitempty"`
	Abort             *[]string `json:"abort,omitempty"`
	AiNationalFocuses *[]string `json:"ai_national_focuses,omitempty"`
	AiStrategy        *[]string `json:"ai_strategy,omitempty"`
	Weight            *[]string `json:"weight,omitempty"`
}

type aiStrategyPlanOut struct {
	AiStrategyPlanID int64  `json:"ai_strategy_plan_id"`
	PlanID           string `json:"plan_id"`
}

func registerCreateAiStrategyPlan(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_ai_strategy_plan",
		Description: "Create an AI strategy plan — a named bundle of triggers/weights that steers how the " +
			"AI-controlled country picks national focuses and diplomatic strategy. allowed/abort gate when the " +
			"plan applies; ai_national_focuses lists the focus ids it drives; weight controls how strongly the " +
			"AI favors it. Every block string must be a full { ... } HOI4 block.",
	}, func(ctx context.Context, _ *CallToolRequest, in createAiStrategyPlanIn) (*CallToolResult, aiStrategyPlanOut, error) {
		body := map[string]any{"plan_id": in.PlanID}
		if in.Name != "" {
			body["name"] = in.Name
		}
		if in.Desc != "" {
			body["desc"] = in.Desc
		}
		if len(in.Allowed) > 0 {
			body["allowed"] = in.Allowed
		}
		if len(in.Abort) > 0 {
			body["abort"] = in.Abort
		}
		if len(in.AiNationalFocuses) > 0 {
			body["ai_national_focuses"] = in.AiNationalFocuses
		}
		if len(in.AiStrategy) > 0 {
			body["ai_strategy"] = in.AiStrategy
		}
		if len(in.Weight) > 0 {
			body["weight"] = in.Weight
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/ai-strategy-plans", in.ProjectID), body)
		if err != nil {
			return nil, aiStrategyPlanOut{}, err
		}
		out := aiStrategyPlanOut{
			AiStrategyPlanID: int64Field(resp, "id"),
			PlanID:           stringField(resp, "plan_id"),
		}
		return nil, out, nil
	})
}

func registerUpdateAiStrategyPlan(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_ai_strategy_plan",
		Description: "Edit an AI strategy plan in place. Only supplied fields change; list fields REPLACE (send the full set).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateAiStrategyPlanIn) (*CallToolResult, aiStrategyPlanOut, error) {
		body := map[string]any{}
		if in.PlanID != nil {
			body["plan_id"] = *in.PlanID
		}
		if in.Name != nil {
			body["name"] = *in.Name
		}
		if in.Desc != nil {
			body["desc"] = *in.Desc
		}
		if in.Allowed != nil {
			body["allowed"] = *in.Allowed
		}
		if in.Abort != nil {
			body["abort"] = *in.Abort
		}
		if in.AiNationalFocuses != nil {
			body["ai_national_focuses"] = *in.AiNationalFocuses
		}
		if in.AiStrategy != nil {
			body["ai_strategy"] = *in.AiStrategy
		}
		if in.Weight != nil {
			body["weight"] = *in.Weight
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/ai-strategy-plans/%d", in.ProjectID, in.AiStrategyPlanID), body)
		if err != nil {
			return nil, aiStrategyPlanOut{}, err
		}
		out := aiStrategyPlanOut{
			AiStrategyPlanID: int64Field(resp, "id"),
			PlanID:           stringField(resp, "plan_id"),
		}
		return nil, out, nil
	})
}

type aiTargetTemplateIn struct {
	ID   string `json:"id" jsonschema:"target template id, unique within this role, e.g. infantry_default"`
	Body string `json:"body" jsonschema:"full HOI4 block, e.g. {\n\tupgrade_prio = {\n\t\tfactor = 1\n\t}\n\ttarget_template = {\n\t\tregiments = {\n\t\t\tinfantry = 6\n\t\t}\n\t}\n}"`
}

type createAiTemplateIn struct {
	ProjectID       int64                `json:"project_id"`
	TemplateID      string               `json:"template_id" jsonschema:"unique within the project, one safe script token"`
	Role            string               `json:"role,omitempty" jsonschema:"e.g. infantry, armor, cas; default infantry"`
	UpgradePrio     string               `json:"upgrade_prio,omitempty" jsonschema:"factor block ranking this template vs. others for the same role; default {\n\tfactor = 1\n}"`
	Enable          string               `json:"enable,omitempty" jsonschema:"trigger block gating when this template is usable"`
	TargetTemplates []aiTargetTemplateIn `json:"target_templates,omitempty" jsonschema:"the actual unit compositions (regiments/support) for this role; at least one required"`
}

type updateAiTemplateIn struct {
	ProjectID       int64                 `json:"project_id"`
	AiTemplateID    int64                 `json:"ai_template_id" jsonschema:"id returned by create_ai_template"`
	TemplateID      *string               `json:"template_id,omitempty"`
	Role            *string               `json:"role,omitempty"`
	UpgradePrio     *string               `json:"upgrade_prio,omitempty"`
	Enable          *string               `json:"enable,omitempty"`
	TargetTemplates *[]aiTargetTemplateIn `json:"target_templates,omitempty" jsonschema:"REPLACES the full set — send every target template you want kept"`
}

type aiTemplateOut struct {
	AiTemplateID int64  `json:"ai_template_id"`
	TemplateID   string `json:"template_id"`
}

func targetTemplatesToBody(items []aiTargetTemplateIn) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{"id": item.ID, "body": item.Body})
	}
	return out
}

func registerCreateAiTemplate(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_ai_template",
		Description: "Create an AI division template — the unit composition the AI builds for a given role " +
			"(infantry/armor/cas/...). target_templates carries the actual regiments/support blocks; " +
			"upgrade_prio ranks this template against sibling templates for the same role.",
	}, func(ctx context.Context, _ *CallToolRequest, in createAiTemplateIn) (*CallToolResult, aiTemplateOut, error) {
		body := map[string]any{"template_id": in.TemplateID}
		if in.Role != "" {
			body["role"] = in.Role
		}
		if in.UpgradePrio != "" {
			body["upgrade_prio"] = in.UpgradePrio
		}
		if in.Enable != "" {
			body["enable"] = in.Enable
		}
		if len(in.TargetTemplates) > 0 {
			body["target_templates"] = targetTemplatesToBody(in.TargetTemplates)
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/ai-templates", in.ProjectID), body)
		if err != nil {
			return nil, aiTemplateOut{}, err
		}
		out := aiTemplateOut{
			AiTemplateID: int64Field(resp, "id"),
			TemplateID:   stringField(resp, "template_id"),
		}
		return nil, out, nil
	})
}

func registerUpdateAiTemplate(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_ai_template",
		Description: "Edit an AI division template in place. Only supplied fields change; target_templates REPLACES the full set.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateAiTemplateIn) (*CallToolResult, aiTemplateOut, error) {
		body := map[string]any{}
		if in.TemplateID != nil {
			body["template_id"] = *in.TemplateID
		}
		if in.Role != nil {
			body["role"] = *in.Role
		}
		if in.UpgradePrio != nil {
			body["upgrade_prio"] = *in.UpgradePrio
		}
		if in.Enable != nil {
			body["enable"] = *in.Enable
		}
		if in.TargetTemplates != nil {
			body["target_templates"] = targetTemplatesToBody(*in.TargetTemplates)
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/ai-templates/%d", in.ProjectID, in.AiTemplateID), body)
		if err != nil {
			return nil, aiTemplateOut{}, err
		}
		out := aiTemplateOut{
			AiTemplateID: int64Field(resp, "id"),
			TemplateID:   stringField(resp, "template_id"),
		}
		return nil, out, nil
	})
}
