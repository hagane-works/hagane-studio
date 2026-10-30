package toolapi

import (
	"context"
	"fmt"
)

type updateCharacterIn struct {
	ProjectID                  int64    `json:"project_id"`
	CharacterID                int64    `json:"character_id" jsonschema:"numeric character id from create_character"`
	Name                       *string  `json:"name,omitempty"`
	Tag                        *string  `json:"tag,omitempty"`
	RoleType                   *string  `json:"role_type,omitempty"`
	Description                *string  `json:"description,omitempty"`
	Skill                      *int     `json:"skill,omitempty"`
	Traits                     []string `json:"traits,omitempty" jsonschema:"REPLACES the whole list"`
	Ideology                   *string  `json:"ideology,omitempty"`
	Gender                     *string  `json:"gender,omitempty"`
	CountryTag                 *string  `json:"country_tag,omitempty"`
	CanBeCaptured              *bool    `json:"can_be_captured,omitempty" jsonschema:"1.19: false emits can_be_captured = no"`
	AlwaysShowOnActionsTooltip *bool    `json:"always_show_on_actions_tooltip,omitempty" jsonschema:"1.19: true emits the advisor-block key; leave unset to omit it"`
}

type updateCharacterOut struct {
	CharacterID int64  `json:"character_id"`
	Tag         string `json:"tag"`
	RoleType    string `json:"role_type"`
}

func registerUpdateCharacter(s *Server, proxy *RESTProxy, apiPrefix string) {
	AddTool(s, &Tool{
		Name: "update_character",
		Description: "Patch a character by numeric character_id. Only supplied fields change. traits REPLACES. " +
			"1.19: can_be_captured=False emits `can_be_captured = no`; always_show_on_actions_tooltip=True emits " +
			"the advisor-block key (leave unset to omit it).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateCharacterIn) (*CallToolResult, updateCharacterOut, error) {
		body := map[string]any{}
		if in.Name != nil {
			body["name"] = *in.Name
		}
		if in.Tag != nil {
			body["tag"] = *in.Tag
		}
		if in.RoleType != nil {
			body["role_type"] = *in.RoleType
		}
		if in.Description != nil {
			body["description"] = *in.Description
		}
		if in.Skill != nil {
			body["skill"] = *in.Skill
		}
		if in.Traits != nil {
			body["traits"] = in.Traits
		}
		if in.Ideology != nil {
			body["ideology"] = *in.Ideology
		}
		if in.Gender != nil {
			body["gender"] = *in.Gender
		}
		if in.CountryTag != nil {
			body["country_tag"] = *in.CountryTag
		}
		if in.CanBeCaptured != nil {
			body["can_be_captured"] = *in.CanBeCaptured
		}
		if in.AlwaysShowOnActionsTooltip != nil {
			body["always_show_on_actions_tooltip"] = *in.AlwaysShowOnActionsTooltip
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("%s/projects/%d/characters/%d", apiPrefix, in.ProjectID, in.CharacterID), body)
		if err != nil {
			return nil, updateCharacterOut{}, err
		}

		out := updateCharacterOut{
			Tag:      stringField(resp, "tag"),
			RoleType: stringField(resp, "role_type"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.CharacterID = int64(id)
		}
		return nil, out, nil
	})
}

type assignCharacterRoleIn struct {
	ProjectID      int64    `json:"project_id"`
	CharacterID    int64    `json:"character_id"`
	RoleType       *string  `json:"role_type,omitempty"`
	Skill          *int     `json:"skill,omitempty"`
	Traits         []string `json:"traits,omitempty"`
	HasAdvisorRole *bool    `json:"has_advisor_role,omitempty"`
	AdvisorSlot    *string  `json:"advisor_slot,omitempty"`
	AdvisorTraits  []string `json:"advisor_traits,omitempty"`
	AdvisorCost    *int     `json:"advisor_cost,omitempty"`
	Ideology       *string  `json:"ideology,omitempty"`
}

type assignCharacterRoleOut struct {
	CharacterID    int64  `json:"character_id"`
	Tag            string `json:"tag"`
	RoleType       string `json:"role_type"`
	HasAdvisorRole bool   `json:"has_advisor_role"`
	AdvisorSlot    string `json:"advisor_slot"`
}

func registerAssignCharacterRole(s *Server, proxy *RESTProxy, apiPrefix string) {
	AddTool(s, &Tool{
		Name: "assign_character_role",
		Description: "Assign/configure a character's role (reuses update). Military role → role_type " +
			"(+skill/traits); advisor → has_advisor_role=True + advisor_slot (+advisor_traits); country leader → " +
			"ideology. Roles stack (a general can also be an advisor). To add a country-leader role, set " +
			"role_type='country_leader' and ideology together.",
	}, func(ctx context.Context, _ *CallToolRequest, in assignCharacterRoleIn) (*CallToolResult, assignCharacterRoleOut, error) {
		body := map[string]any{}
		if in.RoleType != nil {
			body["role_type"] = *in.RoleType
		}
		if in.Skill != nil {
			body["skill"] = *in.Skill
		}
		if in.Traits != nil {
			body["traits"] = in.Traits
		}
		if in.HasAdvisorRole != nil {
			body["has_advisor_role"] = *in.HasAdvisorRole
		}
		if in.AdvisorSlot != nil {
			body["advisor_slot"] = *in.AdvisorSlot
		}
		if in.AdvisorTraits != nil {
			body["advisor_traits"] = in.AdvisorTraits
		}
		if in.AdvisorCost != nil {
			body["advisor_cost"] = *in.AdvisorCost
		}
		if in.Ideology != nil {
			body["ideology"] = *in.Ideology
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("%s/projects/%d/characters/%d", apiPrefix, in.ProjectID, in.CharacterID), body)
		if err != nil {
			return nil, assignCharacterRoleOut{}, err
		}

		out := assignCharacterRoleOut{
			Tag:         stringField(resp, "tag"),
			RoleType:    stringField(resp, "role_type"),
			AdvisorSlot: stringField(resp, "advisor_slot"),
		}
		if id, ok := resp["id"].(float64); ok {
			out.CharacterID = int64(id)
		}
		if hasAdvisor, ok := resp["has_advisor_role"].(bool); ok {
			out.HasAdvisorRole = hasAdvisor
		}
		return nil, out, nil
	})
}
