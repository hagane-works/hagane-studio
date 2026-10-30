package toolapi

import (
	"context"
	"fmt"
)

type createCharacterIn struct {
	ProjectID                  int64             `json:"project_id" jsonschema:"the project to create the character in"`
	Name                       string            `json:"name" jsonschema:"character display name"`
	CountryTag                 string            `json:"country_tag,omitempty" jsonschema:"defaults to the project's primary country"`
	RoleType                   string            `json:"role_type,omitempty" jsonschema:"general (default) | field_marshal | admiral | advisor | scientist | operative | country_leader"`
	Description                string            `json:"description,omitempty"`
	Tag                        string            `json:"tag,omitempty" jsonschema:"character block id; auto-derived from country_tag+name if omitted"`
	Skill                      *int              `json:"skill,omitempty" jsonschema:"default 1"`
	Traits                     []string          `json:"traits,omitempty" jsonschema:"unit-leader traits (or pure-advisor traits — NOT advisor_traits, see hoi4-build-characters)"`
	Ideology                   string            `json:"ideology,omitempty"`
	Gender                     string            `json:"gender,omitempty"`
	HasAdvisorRole             bool              `json:"has_advisor_role,omitempty"`
	AdvisorSlot                string            `json:"advisor_slot,omitempty"`
	AdvisorTraits              []string          `json:"advisor_traits,omitempty"`
	AdvisorCost                *int              `json:"advisor_cost,omitempty" jsonschema:"default 150"`
	CanBeCaptured              *bool             `json:"can_be_captured,omitempty" jsonschema:"default true"`
	AlwaysShowOnActionsTooltip *bool             `json:"always_show_on_actions_tooltip,omitempty"`
	DisplayNameLocs            map[string]string `json:"display_name_locs,omitempty" jsonschema:"ContentLocale code -> display name (en/zh/ru/de/es/fr/ja/ko/pl/pt)"`
	DescriptionLocs            map[string]string `json:"description_locs,omitempty" jsonschema:"ContentLocale code -> description"`
}

type createCharacterOut struct {
	CharacterID    int64    `json:"character_id"`
	Tag            string   `json:"tag"`
	RoleType       string   `json:"role_type"`
	HasAdvisorRole bool     `json:"has_advisor_role"`
	Warnings       []string `json:"warnings"`
}

func intOr(p *int, fallback int) int {
	if p == nil {
		return fallback
	}
	return *p
}

func boolOr(p *bool, fallback bool) bool {
	if p == nil {
		return fallback
	}
	return *p
}

func registerCreateCharacter(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_character",
		Description: "Create a character (leader/general/admiral/advisor…). role_type = military role; " +
			"set role_type='country_leader' together with ideology for a country/party leader; " +
			"use that structured character as the canonical target for workbench and portrait operations. " +
			"Inspect and reconcile any existing legacy/raw country leader before claiming the exported MOD and " +
			"the game view agree. Do not create or patch a parallel history create_country_leader merely to set " +
			"its picture. " +
			"has_advisor_role + advisor_slot for advisors. traits = unit-leader traits; advisor_traits = " +
			"advisor traits (do not cross them). display_name_locs/description_locs accept any of the 10 " +
			"ContentLocale codes (en/zh/ru/de/es/fr/ja/ko/pl/pt). Submod (base-mod-dependent) projects " +
			"are supported: the program's " +
			"usual base-mod tag-collision protection applies automatically.",
	}, func(ctx context.Context, _ *CallToolRequest, in createCharacterIn) (*CallToolResult, createCharacterOut, error) {
		roleType := in.RoleType
		if roleType == "" {
			roleType = "general"
		}

		input := CharacterCreateInput{
			Name:                       in.Name,
			CountryTag:                 in.CountryTag,
			RoleType:                   roleType,
			Description:                in.Description,
			Tag:                        in.Tag,
			Skill:                      intOr(in.Skill, 1),
			Traits:                     in.Traits,
			Ideology:                   in.Ideology,
			Gender:                     in.Gender,
			HasAdvisorRole:             in.HasAdvisorRole,
			AdvisorSlot:                in.AdvisorSlot,
			AdvisorTraits:              in.AdvisorTraits,
			AdvisorCost:                intOr(in.AdvisorCost, 150),
			CanBeCaptured:              boolOr(in.CanBeCaptured, true),
			AlwaysShowOnActionsTooltip: in.AlwaysShowOnActionsTooltip,
			DisplayNameLocs:            in.DisplayNameLocs,
			DescriptionLocs:            in.DescriptionLocs,
		}
		return createCharacterViaProxy(ctx, proxy, in.ProjectID, input)
	})
}

func createCharacterViaProxy(ctx context.Context, proxy *RESTProxy, projectID int64, input CharacterCreateInput) (*CallToolResult, createCharacterOut, error) {
	body := map[string]any{
		"name":              input.Name,
		"country_tag":       input.CountryTag,
		"role_type":         input.RoleType,
		"description":       input.Description,
		"tag":               input.Tag,
		"skill":             input.Skill,
		"traits":            nonNilSlice(input.Traits),
		"ideology":          input.Ideology,
		"gender":            input.Gender,
		"has_advisor_role":  input.HasAdvisorRole,
		"advisor_slot":      input.AdvisorSlot,
		"advisor_traits":    nonNilSlice(input.AdvisorTraits),
		"advisor_cost":      input.AdvisorCost,
		"can_be_captured":   input.CanBeCaptured,
		"display_name_locs": nonNilMap(input.DisplayNameLocs),
		"description_locs":  nonNilMap(input.DescriptionLocs),
	}
	if input.AlwaysShowOnActionsTooltip != nil {
		body["always_show_on_actions_tooltip"] = *input.AlwaysShowOnActionsTooltip
	}

	resp, err := proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/characters", projectID), body)
	if err != nil {
		return nil, createCharacterOut{}, err
	}

	out := createCharacterOut{
		Tag:      stringField(resp, "tag"),
		RoleType: stringField(resp, "role_type"),
	}
	if id, ok := resp["id"].(float64); ok {
		out.CharacterID = int64(id)
	}
	if hasAdvisor, ok := resp["has_advisor_role"].(bool); ok {
		out.HasAdvisorRole = hasAdvisor
	}

	if rawWarnings, ok := resp["warnings"].([]any); ok {
		for _, w := range rawWarnings {
			if s, ok := w.(string); ok {
				out.Warnings = append(out.Warnings, s)
			}
		}
	}
	return nil, out, nil
}

func nonNilSlice(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilMap(v map[string]string) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	return v
}
