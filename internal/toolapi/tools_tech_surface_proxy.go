package toolapi

import (
	"context"
	"fmt"
)

type getTechnologySurfaceEnvironmentIn struct {
	ProjectID int64 `json:"project_id"`
}

type confirmTechnologySurfaceEnvironmentIn struct {
	ProjectID         int64  `json:"project_id"`
	CompatibilityMode string `json:"compatibility_mode" jsonschema:"all_players (recommended) or requires_dlc"`
	NoStepBack        bool   `json:"no_step_back"`
	ByBloodAlone      bool   `json:"by_blood_alone"`
	ManTheGuns        bool   `json:"man_the_guns"`
}

type technologySurfaceProjectIn struct {
	ProjectID int64 `json:"project_id"`
}

type technologySurfaceBranchIn struct {
	ProjectID int64  `json:"project_id"`
	BranchID  string `json:"branch_id" jsonschema:"branch id returned by create_technology_surface_branch"`
}

type technologySurfaceContentIn struct {
	TechID                  string            `json:"tech_id"`
	CountryTag              string            `json:"country_tag,omitempty"`
	Name                    string            `json:"name,omitempty"`
	ResearchCost            *float64          `json:"research_cost,omitempty"`
	Categories              []string          `json:"categories,omitempty"`
	Modifiers               map[string]any    `json:"modifiers,omitempty"`
	AIWillDo                string            `json:"ai_will_do,omitempty"`
	NameEn                  string            `json:"name_en,omitempty"`
	NameZh                  string            `json:"name_zh,omitempty"`
	DescEn                  string            `json:"desc_en,omitempty"`
	DescZh                  string            `json:"desc_zh,omitempty"`
	DisplayNameLocs         map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs         map[string]string `json:"description_locs,omitempty"`
	IsBanner                bool              `json:"is_banner,omitempty"`
	ForceUseSmallTechLayout bool              `json:"force_use_small_tech_layout,omitempty"`
	EnableEquipments        []string          `json:"enable_equipments,omitempty"`
	EnableSubunits          []string          `json:"enable_subunits,omitempty"`
	EnableEquipmentModules  []string          `json:"enable_equipment_modules,omitempty"`
	EnableBuilding          string            `json:"enable_building,omitempty"`
	EnableTactic            string            `json:"enable_tactic,omitempty"`
	Xor                     []string          `json:"xor,omitempty"`
	Dependencies            map[string]any    `json:"dependencies,omitempty"`
	SubTechnologies         []string          `json:"sub_technologies,omitempty"`
	Doctrine                bool              `json:"doctrine,omitempty"`
	Allow                   string            `json:"allow,omitempty"`
	AllowBranch             string            `json:"allow_branch,omitempty"`
	OnResearchComplete      string            `json:"on_research_complete,omitempty"`
	OnResearchCompleteLimit string            `json:"on_research_complete_limit,omitempty"`
	XPResearchType          string            `json:"xp_research_type,omitempty"`
	XPBoostCost             int               `json:"xp_boost_cost,omitempty"`
	XPResearchBonus         map[string]any    `json:"xp_research_bonus,omitempty"`
}

type createTechnologySurfaceBranchIn struct {
	ProjectID int64                      `json:"project_id"`
	EntryID   string                     `json:"entry_id" jsonschema:"one of the four fixed P0 entrances returned by get_technology_surface_environment"`
	TargetID  string                     `json:"target_id" jsonschema:"certified named target returned under the selected entrance"`
	Root      technologySurfaceContentIn `json:"root"`
}

type addTechnologySurfaceNodeIn struct {
	ProjectID          int64                      `json:"project_id"`
	BranchID           string                     `json:"branch_id"`
	ParentTechnologyID string                     `json:"parent_technology_id"`
	Technology         technologySurfaceContentIn `json:"technology"`
}

type updateTechnologySurfaceBranchIn struct {
	ProjectID        int64              `json:"project_id"`
	BranchID         string             `json:"branch_id"`
	Name             *string            `json:"name,omitempty"`
	NameEn           *string            `json:"name_en,omitempty"`
	NameZh           *string            `json:"name_zh,omitempty"`
	DescEn           *string            `json:"desc_en,omitempty"`
	DescZh           *string            `json:"desc_zh,omitempty"`
	DisplayNameLocs  *map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs  *map[string]string `json:"description_locs,omitempty"`
	EnableEquipments *[]string          `json:"enable_equipments,omitempty"`
}

type listTechnologySurfaceBranchesOut struct {
	Branches []map[string]any `json:"branches"`
}

func registerTechSurfaceProxyTools(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "get_technology_surface_environment",
		Description: "Read the observed technology GUI/tag baseline, DLC-confirmation state, active certified profile, and fixed authoring entrances. Geometry is intentionally not returned.",
	}, func(ctx context.Context, _ *CallToolRequest, in getTechnologySurfaceEnvironmentIn) (*CallToolResult, map[string]any, error) {
		out, err := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/technologies/surface", in.ProjectID), nil)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "confirm_technology_surface_environment",
		Description: "Choose player compatibility (all players or selected DLC required) and confirm the author's owned DLCs for preview. The program selects certified Base/DLC placements; callers cannot submit hashes or geometry.",
	}, func(ctx context.Context, _ *CallToolRequest, in confirmTechnologySurfaceEnvironmentIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{"compatibility_mode": in.CompatibilityMode, "dlc": map[string]any{
			"no_step_back": in.NoStepBack, "by_blood_alone": in.ByBloodAlone, "man_the_guns": in.ManTheGuns,
		}}
		out, err := proxyAuthedCall(ctx, proxy, "PUT", fmt.Sprintf("/api/projects/%d/technologies/surface/environment", in.ProjectID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "list_technology_surface_branches",
		Description: "List all controlled technology branches in the project. Returned engine geometry is read-only evidence, never an update vocabulary.",
	}, func(ctx context.Context, _ *CallToolRequest, in technologySurfaceProjectIn) (*CallToolResult, listTechnologySurfaceBranchesOut, error) {
		rows, err := proxyAuthedCallList(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/technologies/surface/branches", in.ProjectID), nil)
		return nil, listTechnologySurfaceBranchesOut{Branches: rows}, err
	})

	AddTool(s, &Tool{
		Name:        "get_technology_surface_branch",
		Description: "Read one controlled technology branch and its profile-owned placement evidence.",
	}, func(ctx context.Context, _ *CallToolRequest, in technologySurfaceBranchIn) (*CallToolResult, map[string]any, error) {
		out, err := proxyAuthedCall(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/technologies/surface/branches/%s", in.ProjectID, in.BranchID), nil)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "create_technology_surface_branch",
		Description: "Create a technology branch under a fixed entrance and certified named target. The branch identity follows root.tech_id; surface/folder/slot/year/x/y/GRIDBOX format/slot size/path are compiler-owned and cannot be supplied.",
	}, func(ctx context.Context, _ *CallToolRequest, in createTechnologySurfaceBranchIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{
			"id": in.Root.TechID, "entry_id": in.EntryID, "target_id": in.TargetID, "root": in.Root,
		}
		out, err := proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/technologies/surface/branches", in.ProjectID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "update_technology_surface_branch",
		Description: "Update only author-owned branch-root names, descriptions, localisation maps, or equipment unlocks. Profile-owned geometry and identity fields are rejected by the tool schema.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateTechnologySurfaceBranchIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{}
		if in.Name != nil {
			body["name"] = *in.Name
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
		if in.EnableEquipments != nil {
			body["enable_equipments"] = *in.EnableEquipments
		}
		out, err := proxyAuthedCall(ctx, proxy, "PUT", fmt.Sprintf("/api/projects/%d/technologies/surface/branches/%s", in.ProjectID, in.BranchID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "delete_technology_surface_branch",
		Description: "Delete one controlled technology branch through the canonical lifecycle command. External references remain deterministic blockers.",
	}, func(ctx context.Context, _ *CallToolRequest, in technologySurfaceBranchIn) (*CallToolResult, map[string]any, error) {
		out, err := proxyAuthedCall(ctx, proxy, "DELETE", fmt.Sprintf("/api/projects/%d/technologies/surface/branches/%s", in.ProjectID, in.BranchID), nil)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "add_technology_surface_node",
		Description: "Append a semantic technology node after a named parent in a controlled branch. The certified profile assigns year, position, folder, and visual path.",
	}, func(ctx context.Context, _ *CallToolRequest, in addTechnologySurfaceNodeIn) (*CallToolResult, map[string]any, error) {
		body := map[string]any{"parent_technology_id": in.ParentTechnologyID, "technology": in.Technology}
		out, err := proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/technologies/surface/branches/%s/technologies", in.ProjectID, in.BranchID), body)
		return nil, out, err
	})

	AddTool(s, &Tool{
		Name:        "repair_technology_surface_timeline",
		Description: "Explicitly repair legacy controlled branches to their certified profile timeline. Accepts no coordinates and reports exactly which technology IDs were repaired.",
	}, func(ctx context.Context, _ *CallToolRequest, in technologySurfaceProjectIn) (*CallToolResult, map[string]any, error) {
		out, err := proxyAuthedCall(ctx, proxy, "POST", fmt.Sprintf("/api/projects/%d/technologies/surface/repair", in.ProjectID), nil)
		return nil, out, err
	})
}
