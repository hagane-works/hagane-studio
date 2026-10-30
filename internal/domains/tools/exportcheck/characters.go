package exportcheck

import (
	"errors"
)

var ErrProjectMissing = errors.New("Project not found")

type Character struct {
	ID                         int64             `json:"id"`
	ProjectID                  int64             `json:"project_id"`
	Name                       string            `json:"name"`
	Tag                        string            `json:"tag"`
	LocalisationKey            string            `json:"localisation_key"`
	CountryTag                 string            `json:"country_tag"`
	Description                string            `json:"description"`
	RoleType                   string            `json:"role_type"`
	HasAdvisorRole             bool              `json:"has_advisor_role"`
	IsReplacement              bool              `json:"is_replacement"`
	ReplacesVanillaID          bool              `json:"replaces_vanilla_id"`
	Skill                      int               `json:"skill"`
	AttackSkill                int               `json:"attack_skill"`
	DefenseSkill               int               `json:"defense_skill"`
	PlanningSkill              int               `json:"planning_skill"`
	LogisticsSkill             int               `json:"logistics_skill"`
	ManeuveringSkill           int               `json:"maneuvering_skill"`
	CoordinationSkill          int               `json:"coordination_skill"`
	MilitaryVisible            string            `json:"military_visible"`
	Traits                     []string          `json:"traits"`
	AdvisorTraits              []string          `json:"advisor_traits"`
	AdvisorSlot                string            `json:"advisor_slot"`
	AdvisorIdeaToken           string            `json:"advisor_idea_token"`
	AdvisorCost                int               `json:"advisor_cost"`
	AdvisorAvailable           string            `json:"advisor_available"`
	AdvisorVisible             string            `json:"advisor_visible"`
	AdvisorRemovalCost         int               `json:"advisor_removal_cost"`
	AdvisorModifier            string            `json:"advisor_modifier"`
	AdvisorOnAdd               string            `json:"advisor_on_add"`
	AdvisorOnRemove            string            `json:"advisor_on_remove"`
	Gender                     string            `json:"gender"`
	CanBeCaptured              bool              `json:"can_be_captured"`
	AlwaysShowOnActionsTooltip *bool             `json:"always_show_on_actions_tooltip"`
	ScientistData              map[string]any    `json:"scientist_data"`
	ScientistVisible           string            `json:"scientist_visible"`
	OperativeSkillLevel        int               `json:"operative_skill_level"`
	OperativeTraits            []string          `json:"operative_traits"`
	OperativeNationalities     []string          `json:"operative_nationalities"`
	OperativeAvailable         string            `json:"operative_available"`
	OperativeBg                string            `json:"operative_bg"`
	Ideology                   string            `json:"ideology"`
	Expire                     string            `json:"expire"`
	LargePortrait              string            `json:"large_portrait"`
	SmallPortrait              string            `json:"small_portrait"`
	RawExtras                  map[string]any    `json:"raw_extras"`
	NameEn                     string            `json:"name_en"`
	NameZh                     string            `json:"name_zh"`
	DescEn                     string            `json:"desc_en"`
	DescZh                     string            `json:"desc_zh"`
	DisplayNameLocs            map[string]string `json:"display_name_locs"`
	DescriptionLocs            map[string]string `json:"description_locs"`
	Source                     string            `json:"source"`
	SourceFile                 string            `json:"source_file"`
	IsPortraitModified         bool              `json:"is_portrait_modified"`
	Warnings                   []string          `json:"warnings"`
}
