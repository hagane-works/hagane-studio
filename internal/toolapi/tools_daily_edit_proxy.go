package toolapi

import (
	"context"
	"fmt"
	"strings"
)

func applyEntityCreate(ctx context.Context, proxy *RESTProxy, kindName string,
	projectID int64, body map[string]any) (updateEntityOut, error) {
	kind, err := lookupEntityKind(kindName)
	if err != nil {
		return nil, err
	}
	resp, err := proxyAuthedCall(ctx, proxy, "POST",
		fmt.Sprintf("/api/projects/%d/%s", projectID, kind.Path), body)
	if err != nil {
		return nil, err
	}
	id := int64Field(resp, "id")
	key := entityKey(kind, resp)
	out := updateEntityOut{
		"id": id, "kind": kind.Kind, "key": key,
	}
	if kind.KeyField != "" {
		out[kind.KeyField] = key
	}
	return out, nil
}

type updateDecisionCategoryIn struct {
	ProjectID         int64             `json:"project_id"`
	ID                int64             `json:"id" jsonschema:"numeric row id from list_entities kind=decision_category"`
	CategoryID        *string           `json:"category_id,omitempty"`
	Icon              *string           `json:"icon,omitempty"`
	Picture           *string           `json:"picture,omitempty"`
	Priority          *int              `json:"priority,omitempty"`
	Allowed           *string           `json:"allowed,omitempty" jsonschema:"raw trigger block"`
	Visible           *string           `json:"visible,omitempty" jsonschema:"raw trigger block"`
	OnMapArea         *string           `json:"on_map_area,omitempty"`
	CountryTag        *string           `json:"country_tag,omitempty"`
	ScopeKind         *string           `json:"scope_kind,omitempty" jsonschema:"own | shared | global"`
	SharedCountryTags []string          `json:"shared_country_tags,omitempty"`
	NameEn            *string           `json:"name_en,omitempty"`
	NameZh            *string           `json:"name_zh,omitempty"`
	DescEn            *string           `json:"desc_en,omitempty"`
	DescZh            *string           `json:"desc_zh,omitempty"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs   map[string]string `json:"description_locs,omitempty"`
	Visibility        *string           `json:"visibility,omitempty"`
}

func registerUpdateDecisionCategory(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_decision_category",
		Description: "Partially update a decision category. Read it first; omitted fields remain " +
			"unchanged and supplied list/map fields replace their current value.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateDecisionCategoryIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "category_id", in.CategoryID)
		set(body, "icon", in.Icon)
		set(body, "picture", in.Picture)
		set(body, "priority", in.Priority)
		set(body, "allowed", in.Allowed)
		set(body, "visible", in.Visible)
		set(body, "on_map_area", in.OnMapArea)
		set(body, "country_tag", in.CountryTag)
		set(body, "scope_kind", in.ScopeKind)
		set(body, "name_en", in.NameEn)
		set(body, "name_zh", in.NameZh)
		set(body, "desc_en", in.DescEn)
		set(body, "desc_zh", in.DescZh)
		set(body, "visibility", in.Visibility)
		if in.SharedCountryTags != nil {
			body["shared_country_tags"] = in.SharedCountryTags
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}
		out, err := applyEntityUpdate(ctx, proxy, "decision_category", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateEventNamespaceIn struct {
	ProjectID         int64    `json:"project_id"`
	ID                int64    `json:"id" jsonschema:"numeric row id from list_entities kind=event_namespace"`
	Namespace         *string  `json:"namespace,omitempty"`
	CountryTag        *string  `json:"country_tag,omitempty"`
	ScopeKind         *string  `json:"scope_kind,omitempty" jsonschema:"own | shared | global"`
	SharedCountryTags []string `json:"shared_country_tags,omitempty"`
}

func registerUpdateEventNamespace(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_event_namespace",
		Description: "Partially update an event namespace. Renaming a namespace can affect event ids; " +
			"read its events and validate references afterwards.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateEventNamespaceIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "namespace", in.Namespace)
		set(body, "country_tag", in.CountryTag)
		set(body, "scope_kind", in.ScopeKind)
		if in.SharedCountryTags != nil {
			body["shared_country_tags"] = in.SharedCountryTags
		}
		out, err := applyEntityUpdate(ctx, proxy, "event_namespace", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateMioIn struct {
	ProjectID          int64             `json:"project_id"`
	ID                 int64             `json:"id" jsonschema:"numeric row id from list_entities kind=mio"`
	MioID              *string           `json:"mio_id,omitempty"`
	Include            *string           `json:"include,omitempty"`
	Icon               *string           `json:"icon,omitempty"`
	EquipmentType      []string          `json:"equipment_type,omitempty"`
	ResearchCategories []string          `json:"research_categories,omitempty"`
	Allowed            *string           `json:"allowed,omitempty"`
	Available          *string           `json:"available,omitempty"`
	Visible            *string           `json:"visible,omitempty"`
	Traits             []map[string]any  `json:"traits,omitempty" jsonschema:"REPLACES the complete MIO trait tree when supplied"`
	RawExtras          map[string]any    `json:"raw_extras,omitempty"`
	CountryTag         *string           `json:"country_tag,omitempty"`
	ScopeKind          *string           `json:"scope_kind,omitempty"`
	SharedCountryTags  []string          `json:"shared_country_tags,omitempty"`
	NameEn             *string           `json:"name_en,omitempty"`
	NameZh             *string           `json:"name_zh,omitempty"`
	DescEn             *string           `json:"desc_en,omitempty"`
	DescZh             *string           `json:"desc_zh,omitempty"`
	DisplayNameLocs    map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs    map[string]string `json:"description_locs,omitempty"`
}

func registerUpdateMio(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_mio",
		Description: "Partially update a Military Industrial Organization. `traits` replaces the " +
			"whole trait list; use add_mio_trait only for a simple append.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateMioIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "mio_id", in.MioID)
		set(body, "include", in.Include)
		set(body, "icon", in.Icon)
		set(body, "allowed", in.Allowed)
		set(body, "available", in.Available)
		set(body, "visible", in.Visible)
		set(body, "country_tag", in.CountryTag)
		set(body, "scope_kind", in.ScopeKind)
		set(body, "name_en", in.NameEn)
		set(body, "name_zh", in.NameZh)
		set(body, "desc_en", in.DescEn)
		set(body, "desc_zh", in.DescZh)
		if in.EquipmentType != nil {
			body["equipment_type"] = in.EquipmentType
		}
		if in.ResearchCategories != nil {
			body["research_categories"] = in.ResearchCategories
		}
		if in.Traits != nil {
			body["traits"] = in.Traits
		}
		if in.RawExtras != nil {
			body["raw_extras"] = in.RawExtras
		}
		if in.SharedCountryTags != nil {
			body["shared_country_tags"] = in.SharedCountryTags
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}
		out, err := applyEntityUpdate(ctx, proxy, "mio", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateIntelligenceAgencyIn struct {
	ProjectID         int64             `json:"project_id"`
	ID                int64             `json:"id" jsonschema:"numeric row id from list_entities kind=intelligence_agency"`
	AgencyID          *string           `json:"agency_id,omitempty"`
	Picture           *string           `json:"picture,omitempty"`
	Names             []string          `json:"names,omitempty"`
	DefaultTrigger    *string           `json:"default_trigger,omitempty"`
	Available         *string           `json:"available,omitempty"`
	UpgradeBranches   []map[string]any  `json:"upgrade_branches,omitempty" jsonschema:"REPLACES all upgrade branches when supplied"`
	RawExtras         map[string]any    `json:"raw_extras,omitempty"`
	CountryTag        *string           `json:"country_tag,omitempty"`
	ScopeKind         *string           `json:"scope_kind,omitempty"`
	SharedCountryTags []string          `json:"shared_country_tags,omitempty"`
	NameEn            *string           `json:"name_en,omitempty"`
	NameZh            *string           `json:"name_zh,omitempty"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
}

func registerUpdateIntelligenceAgency(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_intelligence_agency",
		Description: "Partially update an intelligence agency. `upgrade_branches` replaces the " +
			"complete branch list; use add_agency_upgrade only for a simple append.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateIntelligenceAgencyIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "agency_id", in.AgencyID)
		set(body, "picture", in.Picture)
		set(body, "default_trigger", in.DefaultTrigger)
		set(body, "available", in.Available)
		set(body, "country_tag", in.CountryTag)
		set(body, "scope_kind", in.ScopeKind)
		set(body, "name_en", in.NameEn)
		set(body, "name_zh", in.NameZh)
		if in.Names != nil {
			body["names"] = in.Names
		}
		if in.UpgradeBranches != nil {
			body["upgrade_branches"] = in.UpgradeBranches
		}
		if in.RawExtras != nil {
			body["raw_extras"] = in.RawExtras
		}
		if in.SharedCountryTags != nil {
			body["shared_country_tags"] = in.SharedCountryTags
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		out, err := applyEntityUpdate(ctx, proxy, "intelligence_agency", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type specializationFields struct {
	SpecializationID   *string           `json:"specialization_id,omitempty"`
	CountryTag         *string           `json:"country_tag,omitempty"`
	NameEn             *string           `json:"name_en,omitempty"`
	NameZh             *string           `json:"name_zh,omitempty"`
	DescEn             *string           `json:"desc_en,omitempty"`
	DescZh             *string           `json:"desc_zh,omitempty"`
	DisplayNameLocs    map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs    map[string]string `json:"description_locs,omitempty"`
	Color              []int             `json:"color,omitempty"`
	BlueprintImage     *string           `json:"blueprint_image,omitempty"`
	ProgramBackground  *string           `json:"program_background,omitempty"`
	FacilityModel      *string           `json:"facility_model,omitempty"`
	SubSpecializations []string          `json:"sub_specializations,omitempty"`
	RawExtras          map[string]any    `json:"raw_extras,omitempty"`
}

func (fields specializationFields) body() map[string]any {
	body := map[string]any{}
	set(body, "specialization_id", fields.SpecializationID)
	set(body, "country_tag", fields.CountryTag)
	set(body, "name_en", fields.NameEn)
	set(body, "name_zh", fields.NameZh)
	set(body, "desc_en", fields.DescEn)
	set(body, "desc_zh", fields.DescZh)
	set(body, "blueprint_image", fields.BlueprintImage)
	set(body, "program_background", fields.ProgramBackground)
	set(body, "facility_model", fields.FacilityModel)
	if fields.DisplayNameLocs != nil {
		body["display_name_locs"] = fields.DisplayNameLocs
	}
	if fields.DescriptionLocs != nil {
		body["description_locs"] = fields.DescriptionLocs
	}
	if fields.Color != nil {
		body["color"] = fields.Color
	}
	if fields.SubSpecializations != nil {
		body["sub_specializations"] = fields.SubSpecializations
	}
	if fields.RawExtras != nil {
		body["raw_extras"] = fields.RawExtras
	}
	return body
}

type createSpecializationIn struct {
	ProjectID            int64 `json:"project_id"`
	specializationFields `json:",inline"`
}

func registerCreateSpecialization(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_specialization",
		Description: "Create a special-project specialization/track before adding DAG projects to it. " +
			"specialization_id is required and unique within the project.",
	}, func(ctx context.Context, _ *CallToolRequest, in createSpecializationIn) (*CallToolResult, updateEntityOut, error) {
		body := in.specializationFields.body()
		if in.SpecializationID == nil || strings.TrimSpace(*in.SpecializationID) == "" {
			return nil, nil, fmt.Errorf("specialization_id is required")
		}
		out, err := applyEntityCreate(ctx, proxy, "specialization", in.ProjectID, body)
		return nil, out, err
	})
}

type updateSpecializationIn struct {
	ProjectID            int64 `json:"project_id"`
	ID                   int64 `json:"id" jsonschema:"numeric row id from list_entities kind=specialization"`
	specializationFields `json:",inline"`
}

func registerUpdateSpecialization(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{Name: "update_specialization", Description: "Partially update a special-project specialization/track."},
		func(ctx context.Context, _ *CallToolRequest, in updateSpecializationIn) (*CallToolResult, updateEntityOut, error) {
			out, err := applyEntityUpdate(ctx, proxy, "specialization", in.ProjectID, in.ID, in.specializationFields.body())
			return nil, out, err
		})
}

type updateSpecialProjectIn struct {
	ProjectID               int64             `json:"project_id"`
	ID                      int64             `json:"id" jsonschema:"numeric row id from list_entities kind=special_project"`
	ProjectToken            *string           `json:"project_token,omitempty"`
	CountryTag              *string           `json:"country_tag,omitempty"`
	Specialization          *string           `json:"specialization,omitempty" jsonschema:"when set, a non-empty specialization/track id"`
	NameEn                  *string           `json:"name_en,omitempty"`
	NameZh                  *string           `json:"name_zh,omitempty"`
	DescEn                  *string           `json:"desc_en,omitempty"`
	DescZh                  *string           `json:"desc_zh,omitempty"`
	DisplayNameLocs         map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs         map[string]string `json:"description_locs,omitempty"`
	Icon                    *string           `json:"icon,omitempty"`
	BlueprintImage          *string           `json:"blueprint_image,omitempty"`
	Complexity              *string           `json:"complexity,omitempty" jsonschema:"one positive integer or script constant"`
	PrototypeTime           *string           `json:"prototype_time,omitempty" jsonschema:"one positive integer or script constant"`
	ProjectTags             []string          `json:"project_tags,omitempty"`
	SpecialProjectParent    []string          `json:"special_project_parent,omitempty" jsonschema:"REPLACES parent project tokens (DAG prerequisites)"`
	Allowed                 *string           `json:"allowed,omitempty"`
	Visible                 *string           `json:"visible,omitempty"`
	Available               *string           `json:"available,omitempty"`
	ResourceCost            *string           `json:"resource_cost,omitempty" jsonschema:"HOI4 block body containing resources = { ... }"`
	ProjectOutput           *string           `json:"project_output,omitempty" jsonschema:"validated HOI4 output/effect block body"`
	GenericPrototypeRewards []map[string]any  `json:"generic_prototype_rewards,omitempty"`
	X                       *int              `json:"x,omitempty"`
	Y                       *int              `json:"y,omitempty"`
	RawExtras               map[string]any    `json:"raw_extras,omitempty"`
}

func registerUpdateSpecialProject(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_special_project",
		Description: "Partially update a special-project DAG node. Read it first and run " +
			"validate_special_projects afterwards. Required specialization, scalar constants, resource cost, " +
			"and project output are syntax-checked before the change is saved.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateSpecialProjectIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "project_token", in.ProjectToken)
		set(body, "country_tag", in.CountryTag)
		set(body, "specialization", in.Specialization)
		set(body, "name_en", in.NameEn)
		set(body, "name_zh", in.NameZh)
		set(body, "desc_en", in.DescEn)
		set(body, "desc_zh", in.DescZh)
		set(body, "icon", in.Icon)
		set(body, "blueprint_image", in.BlueprintImage)
		set(body, "complexity", in.Complexity)
		set(body, "prototype_time", in.PrototypeTime)
		set(body, "allowed", in.Allowed)
		set(body, "visible", in.Visible)
		set(body, "available", in.Available)
		set(body, "resource_cost", in.ResourceCost)
		set(body, "project_output", in.ProjectOutput)
		set(body, "x", in.X)
		set(body, "y", in.Y)
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}
		if in.ProjectTags != nil {
			body["project_tags"] = in.ProjectTags
		}
		if in.SpecialProjectParent != nil {
			body["special_project_parent"] = in.SpecialProjectParent
		}
		if in.GenericPrototypeRewards != nil {
			body["generic_prototype_rewards"] = in.GenericPrototypeRewards
		}
		if in.RawExtras != nil {
			body["raw_extras"] = in.RawExtras
		}
		out, err := applyEntityUpdate(ctx, proxy, "special_project", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateAirWingGroupIn struct {
	ProjectID  int64            `json:"project_id"`
	ID         int64            `json:"id"`
	OobFile    *string          `json:"oob_file,omitempty"`
	SourceFile *string          `json:"source_file,omitempty"`
	CountryTag *string          `json:"country_tag,omitempty"`
	Location   *string          `json:"location,omitempty" jsonschema:"air-base state id as a string token"`
	Ordering   *int             `json:"ordering,omitempty"`
	Layout     []map[string]any `json:"layout,omitempty"`
}

func registerUpdateAirWingGroup(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{Name: "update_air_wing_group", Description: "Partially update a starting air-base wing group."},
		func(ctx context.Context, _ *CallToolRequest, in updateAirWingGroupIn) (*CallToolResult, updateEntityOut, error) {
			body := map[string]any{}
			set(body, "oob_file", in.OobFile)
			set(body, "source_file", in.SourceFile)
			set(body, "country_tag", in.CountryTag)
			set(body, "location", in.Location)
			set(body, "ordering", in.Ordering)
			if in.Layout != nil {
				body["layout"] = in.Layout
			}
			out, err := applyEntityUpdate(ctx, proxy, "air_wing_group", in.ProjectID, in.ID, body)
			return nil, out, err
		})
}

type updateAirWingIn struct {
	ProjectID     int64            `json:"project_id"`
	ID            int64            `json:"id"`
	GroupID       *int64           `json:"group_id,omitempty"`
	EquipmentType *string          `json:"equipment_type,omitempty"`
	Name          *string          `json:"name,omitempty"`
	Owner         *string          `json:"owner,omitempty"`
	Amount        *string          `json:"amount,omitempty" jsonschema:"plane count as a string token"`
	Ordering      *int             `json:"ordering,omitempty"`
	BodyAST       []map[string]any `json:"body_ast,omitempty"`
}

func registerUpdateAirWing(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{Name: "update_air_wing", Description: "Partially update one starting air wing."},
		func(ctx context.Context, _ *CallToolRequest, in updateAirWingIn) (*CallToolResult, updateEntityOut, error) {
			body := map[string]any{}
			set(body, "group_id", in.GroupID)
			set(body, "equipment_type", in.EquipmentType)
			set(body, "name", in.Name)
			set(body, "owner", in.Owner)
			set(body, "amount", in.Amount)
			set(body, "ordering", in.Ordering)
			if in.BodyAST != nil {
				body["body_ast"] = in.BodyAST
			}
			out, err := applyEntityUpdate(ctx, proxy, "air_wing", in.ProjectID, in.ID, body)
			return nil, out, err
		})
}

type updateFleetIn struct {
	ProjectID  int64            `json:"project_id"`
	ID         int64            `json:"id"`
	OobFile    *string          `json:"oob_file,omitempty"`
	SourceFile *string          `json:"source_file,omitempty"`
	CountryTag *string          `json:"country_tag,omitempty"`
	Name       *string          `json:"name,omitempty"`
	NavalBase  *string          `json:"naval_base,omitempty" jsonschema:"home naval-base province id as a string token"`
	Ordering   *int             `json:"ordering,omitempty"`
	Layout     []map[string]any `json:"layout,omitempty"`
}

func registerUpdateFleet(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{Name: "update_fleet", Description: "Partially update a starting navy fleet."},
		func(ctx context.Context, _ *CallToolRequest, in updateFleetIn) (*CallToolResult, updateEntityOut, error) {
			body := map[string]any{}
			set(body, "oob_file", in.OobFile)
			set(body, "source_file", in.SourceFile)
			set(body, "country_tag", in.CountryTag)
			set(body, "name", in.Name)
			set(body, "naval_base", in.NavalBase)
			set(body, "ordering", in.Ordering)
			if in.Layout != nil {
				body["layout"] = in.Layout
			}
			out, err := applyEntityUpdate(ctx, proxy, "fleet", in.ProjectID, in.ID, body)
			return nil, out, err
		})
}

type updateTaskForceIn struct {
	ProjectID int64            `json:"project_id"`
	ID        int64            `json:"id"`
	FleetID   *int64           `json:"fleet_id,omitempty"`
	Name      *string          `json:"name,omitempty"`
	Location  *string          `json:"location,omitempty" jsonschema:"starting province id as a string token"`
	Ordering  *int             `json:"ordering,omitempty"`
	Layout    []map[string]any `json:"layout,omitempty"`
}

func registerUpdateTaskForce(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{Name: "update_task_force", Description: "Partially update a starting navy task force."},
		func(ctx context.Context, _ *CallToolRequest, in updateTaskForceIn) (*CallToolResult, updateEntityOut, error) {
			body := map[string]any{}
			set(body, "fleet_id", in.FleetID)
			set(body, "name", in.Name)
			set(body, "location", in.Location)
			set(body, "ordering", in.Ordering)
			if in.Layout != nil {
				body["layout"] = in.Layout
			}
			out, err := applyEntityUpdate(ctx, proxy, "task_force", in.ProjectID, in.ID, body)
			return nil, out, err
		})
}

type updateShipIn struct {
	ProjectID             int64            `json:"project_id"`
	ID                    int64            `json:"id"`
	TaskForceID           *int64           `json:"task_force_id,omitempty"`
	Name                  *string          `json:"name,omitempty"`
	Definition            *string          `json:"definition,omitempty"`
	StartExperienceFactor *string          `json:"start_experience_factor,omitempty"`
	EquipmentType         *string          `json:"equipment_type,omitempty"`
	Amount                *string          `json:"amount,omitempty"`
	Owner                 *string          `json:"owner,omitempty"`
	VersionName           *string          `json:"version_name,omitempty"`
	Ordering              *int             `json:"ordering,omitempty"`
	BodyAST               []map[string]any `json:"body_ast,omitempty"`
}

func registerUpdateShip(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{Name: "update_ship", Description: "Partially update one ship in a starting task force."},
		func(ctx context.Context, _ *CallToolRequest, in updateShipIn) (*CallToolResult, updateEntityOut, error) {
			body := map[string]any{}
			set(body, "task_force_id", in.TaskForceID)
			set(body, "name", in.Name)
			set(body, "definition", in.Definition)
			set(body, "start_experience_factor", in.StartExperienceFactor)
			set(body, "equipment_type", in.EquipmentType)
			set(body, "amount", in.Amount)
			set(body, "owner", in.Owner)
			set(body, "version_name", in.VersionName)
			set(body, "ordering", in.Ordering)
			if in.BodyAST != nil {
				body["body_ast"] = in.BodyAST
			}
			out, err := applyEntityUpdate(ctx, proxy, "ship", in.ProjectID, in.ID, body)
			return nil, out, err
		})
}

type createRawFileIn struct {
	ProjectID           int64   `json:"project_id"`
	DestPath            string  `json:"dest_path" jsonschema:"safe MOD-relative destination path"`
	Content             string  `json:"content" jsonschema:"complete UTF-8 text file content"`
	GenerateReplacePath bool    `json:"generate_replace_path,omitempty"`
	RawKind             string  `json:"raw_kind,omitempty" jsonschema:"default verbatim_override"`
	TargetLayer         *string `json:"target_layer,omitempty"`
	TargetSHA256        *string `json:"target_sha256,omitempty"`
	Note                string  `json:"note,omitempty"`
}

func registerCreateRawFile(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_raw_file",
		Description: "Create a validated text override for a MOD path that has no structured editor. " +
			"Prefer typed entity tools whenever one exists; raw files still pass path, tombstone, and script safety gates.",
	}, func(ctx context.Context, _ *CallToolRequest, in createRawFileIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{"dest_path": in.DestPath, "content": in.Content,
			"generate_replace_path": in.GenerateReplacePath, "note": in.Note}
		if in.RawKind != "" {
			body["raw_kind"] = in.RawKind
		}
		if in.TargetLayer != nil {
			body["target_layer"] = *in.TargetLayer
		}
		if in.TargetSHA256 != nil {
			body["target_sha256"] = *in.TargetSHA256
		}
		out, err := applyEntityCreate(ctx, proxy, "raw_file", in.ProjectID, body)
		return nil, out, err
	})
}

type updateRawFileIn struct {
	ProjectID           int64   `json:"project_id"`
	ID                  int64   `json:"id"`
	DestPath            *string `json:"dest_path,omitempty"`
	Content             *string `json:"content,omitempty"`
	GenerateReplacePath *bool   `json:"generate_replace_path,omitempty"`
	RawKind             *string `json:"raw_kind,omitempty"`
	TargetLayer         *string `json:"target_layer,omitempty"`
	TargetSHA256        *string `json:"target_sha256,omitempty"`
	Note                *string `json:"note,omitempty"`
}

func registerUpdateRawFile(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_raw_file",
		Description: "Partially update a validated text override. Read it first; this is for content " +
			"without a structured editor, not a shortcut around typed MOD tools.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateRawFileIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "dest_path", in.DestPath)
		set(body, "content", in.Content)
		set(body, "generate_replace_path", in.GenerateReplacePath)
		set(body, "raw_kind", in.RawKind)
		set(body, "target_layer", in.TargetLayer)
		set(body, "target_sha256", in.TargetSHA256)
		set(body, "note", in.Note)
		out, err := applyEntityUpdate(ctx, proxy, "raw_file", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type batchEntityPatch struct {
	Kind     string         `json:"kind" jsonschema:"one of the kinds accepted by list_entities"`
	ID       int64          `json:"id" jsonschema:"numeric row id"`
	ParentID *int64         `json:"parent_id,omitempty" jsonschema:"required only for kind=focus: numeric focus-tree row id"`
	Fields   map[string]any `json:"fields,omitempty" jsonschema:"partial update fields; read the entity and its field reference (hw fields) first"`
}

type batchUpdateEntitiesIn struct {
	ProjectID   int64              `json:"project_id"`
	Operations  []batchEntityPatch `json:"operations" jsonschema:"1 to 200 stable-order partial updates"`
	StopOnError bool               `json:"stop_on_error,omitempty" jsonschema:"default false; successes are preserved and failures reported per item"`
}

type batchEntityPatchResult struct {
	Index          int                 `json:"index"`
	Kind           string              `json:"kind"`
	ID             int64               `json:"id"`
	Key            string              `json:"key"`
	OK             bool                `json:"ok"`
	ReferenceCheck string              `json:"reference_check,omitempty"`
	UnresolvedRefs []DanglingReference `json:"unresolved_references,omitempty"`
	Warnings       []string            `json:"warnings,omitempty"`
	Error          string              `json:"error,omitempty"`
}

type batchUpdateEntitiesOut struct {
	Requested int                      `json:"requested"`
	Attempted int                      `json:"attempted"`
	Updated   int                      `json:"updated"`
	Failed    int                      `json:"failed"`
	Results   []batchEntityPatchResult `json:"results"`
}

var forbiddenBatchFields = map[string]bool{
	"id": true, "project_id": true, "created_at": true, "updated_at": true,
	"source": true, "warnings": true, "revision": true,
}

func batchEntityPath(projectID int64, kind entityKind, operation batchEntityPatch) (string, error) {
	if operation.ID <= 0 {
		return "", fmt.Errorf("id must be a positive numeric row id")
	}
	if kind.ParentPath != "" {
		if operation.ParentID == nil || *operation.ParentID <= 0 {
			return "", fmt.Errorf("kind %s requires parent_id", kind.Kind)
		}
		return fmt.Sprintf("/api/projects/%d/%s/%d/%s/%d", projectID,
			kind.ParentPath, *operation.ParentID, kind.Path, operation.ID), nil
	}
	return fmt.Sprintf("/api/projects/%d/%s/%d", projectID, kind.Path, operation.ID), nil
}

func validateBatchFields(fields map[string]any) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields must contain at least one partial update")
	}
	for raw := range fields {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || forbiddenBatchFields[name] {
			return fmt.Errorf("field %q is program-owned and cannot be batch-updated", raw)
		}
	}
	return nil
}

func registerBatchUpdateEntities(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "batch_update_entities",
		Description: "Apply 1-200 deterministic partial updates across existing MOD entities. Read the " +
			"complete target set first. Every operation is project-scoped, runs in stable order, rejects " +
			"program-owned fields, and verifies that every requested field exists in the saved response. " +
			"This is not a transaction: successful earlier rows remain saved and are reported if a later row fails.",
	}, func(ctx context.Context, _ *CallToolRequest, in batchUpdateEntitiesIn) (*CallToolResult, batchUpdateEntitiesOut, error) {
		if len(in.Operations) == 0 || len(in.Operations) > 200 {
			return nil, batchUpdateEntitiesOut{}, fmt.Errorf("operations must contain between 1 and 200 updates")
		}
		out := batchUpdateEntitiesOut{Requested: len(in.Operations), Results: make([]batchEntityPatchResult, 0, len(in.Operations))}
		for index, operation := range in.Operations {
			row := batchEntityPatchResult{Index: index, Kind: operation.Kind, ID: operation.ID}
			out.Attempted++
			kind, err := lookupEntityKind(operation.Kind)
			if err == nil {
				err = validateBatchFields(operation.Fields)
			}
			var path string
			if err == nil {
				path, err = batchEntityPath(in.ProjectID, kind, operation)
			}
			var resp map[string]any
			if err == nil {
				resp, err = proxyAuthedCall(ctx, proxy, "PUT", path, operation.Fields)
			}
			if err == nil {
				for field := range operation.Fields {
					if _, present := resp[field]; !present {
						err = fmt.Errorf("saved response did not contain requested field %q", field)
						break
					}
				}
			}
			if err != nil {
				row.Error = err.Error()
				out.Failed++
				out.Results = append(out.Results, row)
				if in.StopOnError {
					break
				}
				continue
			}
			row.OK = true
			row.Key = entityKey(kind, resp)
			if relationFieldsTouched(kind.Kind, operation.Fields) {
				row.ReferenceCheck, row.UnresolvedRefs, row.Warnings = checkEntityReferences(
					ctx, proxy, in.ProjectID, kind.Kind, operation.ID, true,
				)
			}
			out.Updated++
			out.Results = append(out.Results, row)
		}
		return nil, out, nil
	})
}

func registerDailyEditTools(s *Server, proxy *RESTProxy) {
	registerUpdateDecisionCategory(s, proxy)
	registerUpdateEventNamespace(s, proxy)
	registerUpdateMio(s, proxy)
	registerUpdateIntelligenceAgency(s, proxy)
	registerCreateSpecialization(s, proxy)
	registerUpdateSpecialization(s, proxy)
	registerUpdateSpecialProject(s, proxy)
	registerUpdateAirWingGroup(s, proxy)
	registerUpdateAirWing(s, proxy)
	registerUpdateFleet(s, proxy)
	registerUpdateTaskForce(s, proxy)
	registerUpdateShip(s, proxy)
	registerCreateRawFile(s, proxy)
	registerUpdateRawFile(s, proxy)
	registerBatchUpdateEntities(s, proxy)
}
