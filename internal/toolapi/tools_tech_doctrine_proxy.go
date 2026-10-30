package toolapi

import (
	"context"
	"fmt"
	"strings"
)

type createTechnologyIn struct {
	ProjectID           int64             `json:"project_id"`
	Name                string            `json:"name" jsonschema:"set explicitly -- a blank name rides in as the literal bareword New Technology"`
	TechID              string            `json:"tech_id,omitempty" jsonschema:"blank -> auto-derived from name (prefixed by tech_line), auto-suffixed _1.._1000 on collision; READ BACK the returned tech_id"`
	TechLine            string            `json:"tech_line,omitempty" jsonschema:"default infantry"`
	Folder              string            `json:"folder,omitempty" jsonschema:"blank derives from tech_line; armour uses the BRITISH spelling armour_folder"`
	ResearchCost        *float64          `json:"research_cost,omitempty" jsonschema:"default 1.0"`
	StartYear           *int              `json:"start_year,omitempty" jsonschema:"default 1936"`
	PositionX           int               `json:"position_x,omitempty"`
	PositionY           int               `json:"position_y,omitempty"`
	PositionYIsExplicit bool              `json:"position_y_is_explicit,omitempty" jsonschema:"false derives the row from start_year; true emits stored position_y, including row 0"`
	Prerequisites       []string          `json:"prerequisites,omitempty"`
	Categories          []string          `json:"categories,omitempty"`
	CountryModifier     string            `json:"country_modifier,omitempty" jsonschema:"RAW multi-line string, e.g. 'army_attack_factor = 0.05\\nresearch_speed_factor = 0.1' -- packed into modifiers['__modifier']"`
	Modifiers           map[string]any    `json:"modifiers,omitempty" jsonschema:"nested per-unit form {archetype: {stat: value}} ONLY -- a flat {stat:value} dict is silently ignored by the exporter"`
	EnableEquipments    []string          `json:"enable_equipments,omitempty"`
	Doctrine            bool              `json:"doctrine,omitempty"`
	OnResearchComplete  string            `json:"on_research_complete,omitempty"`
	CountryTag          string            `json:"country_tag,omitempty" jsonschema:"ownership is inferred from the tech_id PREFIX (e.g. USA_...), not this column"`
	NameEn              string            `json:"name_en,omitempty"`
	NameZh              string            `json:"name_zh,omitempty"`
	DescEn              string            `json:"desc_en,omitempty"`
	DescZh              string            `json:"desc_zh,omitempty"`
	DisplayNameLocs     map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs     map[string]string `json:"description_locs,omitempty"`
	AIWillDo            string            `json:"ai_will_do,omitempty" jsonschema:"default: factor = 1"`
}

type createTechnologyOut struct {
	TechDBID int64    `json:"tech_db_id"`
	TechID   string   `json:"tech_id"`
	Warnings []string `json:"warnings"`
}

func registerCreateTechnology(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_technology",
		Description: "Create a research technology node (custom or overriding). " +
			"A tech's country-wide bonus MUST be a RAW multi-line string in country_modifier " +
			"(packed into modifiers['__modifier'] for you); modifiers is ONLY for the nested " +
			"per-unit form {archetype: {stat: value}} -- a flat {stat:value} dict there is " +
			"silently ignored by the exporter.",
	}, func(ctx context.Context, _ *CallToolRequest, in createTechnologyIn) (*CallToolResult, createTechnologyOut, error) {
		techLine := in.TechLine
		if techLine == "" {
			techLine = "infantry"
		}
		aiWillDo := in.AIWillDo
		if aiWillDo == "" {
			aiWillDo = "factor = 1"
		}
		researchCost := techdocFloatOr(in.ResearchCost, 1.0)
		startYear := intOr(in.StartYear, 1936)

		mods := map[string]any{}
		for k, v := range in.Modifiers {
			mods[k] = v
		}
		if strings.TrimSpace(in.CountryModifier) != "" {
			mods["__modifier"] = in.CountryModifier
		}

		body := map[string]any{
			"tech_id":                in.TechID,
			"name":                   in.Name,
			"tech_line":              techLine,
			"folder":                 in.Folder,
			"research_cost":          researchCost,
			"start_year":             startYear,
			"position_x":             in.PositionX,
			"position_y":             in.PositionY,
			"position_y_is_explicit": in.PositionYIsExplicit,
			"prerequisites":          nonNilSlice(in.Prerequisites),
			"categories":             nonNilSlice(in.Categories),
			"modifiers":              mods,
			"enable_equipments":      nonNilSlice(in.EnableEquipments),
			"doctrine":               in.Doctrine,
			"on_research_complete":   in.OnResearchComplete,
			"country_tag":            in.CountryTag,
			"name_en":                in.NameEn,
			"name_zh":                in.NameZh,
			"desc_en":                in.DescEn,
			"desc_zh":                in.DescZh,
			"ai_will_do":             aiWillDo,
		}
		if in.DisplayNameLocs != nil {
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/technologies", in.ProjectID), body)
		if err != nil {
			return nil, createTechnologyOut{}, err
		}

		out := createTechnologyOut{
			TechID:   stringField(resp, "tech_id"),
			Warnings: techdocStringSliceField(resp, "warnings"),
		}
		out.TechDBID = techdocInt64Field(resp, "id")
		return nil, out, nil
	})
}

type updateTechnologyIn struct {
	ProjectID           int64             `json:"project_id"`
	TechDBID            int64             `json:"tech_db_id" jsonschema:"numeric TechnologyModel.id from create_technology's tech_db_id -- NOT the string tech_id"`
	Name                *string           `json:"name,omitempty"`
	TechLine            *string           `json:"tech_line,omitempty"`
	Folder              *string           `json:"folder,omitempty"`
	ResearchCost        *float64          `json:"research_cost,omitempty"`
	StartYear           *int              `json:"start_year,omitempty"`
	PositionX           *int              `json:"position_x,omitempty"`
	PositionY           *int              `json:"position_y,omitempty"`
	PositionYIsExplicit *bool             `json:"position_y_is_explicit,omitempty"`
	Prerequisites       []string          `json:"prerequisites,omitempty"`
	Categories          []string          `json:"categories,omitempty"`
	CountryModifier     *string           `json:"country_modifier,omitempty" jsonschema:"RAW string packed into modifiers['__modifier'] -- only takes effect if modifiers is ALSO supplied or this is non-empty"`
	Modifiers           map[string]any    `json:"modifiers,omitempty"`
	EnableEquipments    []string          `json:"enable_equipments,omitempty"`
	Doctrine            *bool             `json:"doctrine,omitempty"`
	OnResearchComplete  *string           `json:"on_research_complete,omitempty"`
	CountryTag          *string           `json:"country_tag,omitempty"`
	RawExtras           map[string]any    `json:"raw_extras,omitempty" jsonschema:"re-emits preserved raw blocks verbatim -- a tech's country modifier still lives in modifiers['__modifier'], NOT here"`
	NameEn              *string           `json:"name_en,omitempty"`
	NameZh              *string           `json:"name_zh,omitempty"`
	DescEn              *string           `json:"desc_en,omitempty"`
	DescZh              *string           `json:"desc_zh,omitempty"`
	DisplayNameLocs     map[string]string `json:"display_name_locs,omitempty"`
	DescriptionLocs     map[string]string `json:"description_locs,omitempty"`
	AIWillDo            *string           `json:"ai_will_do,omitempty"`
}

type updateTechnologyOut struct {
	TechDBID int64  `json:"tech_db_id"`
	TechID   string `json:"tech_id"`
}

func registerUpdateTechnology(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_technology",
		Description: "Patch a technology by its INTEGER DB id (tech_db_id, not the string tech_id). " +
			"Only supplied fields change. Same modifier rule as create: pass the country-wide bonus as " +
			"a RAW string in country_modifier (packed into modifiers['__modifier']).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateTechnologyIn) (*CallToolResult, updateTechnologyOut, error) {
		body := map[string]any{}
		if in.Name != nil {
			body["name"] = *in.Name
		}
		if in.TechLine != nil {
			body["tech_line"] = *in.TechLine
		}
		if in.Folder != nil {
			body["folder"] = *in.Folder
		}
		if in.ResearchCost != nil {
			body["research_cost"] = *in.ResearchCost
		}
		if in.StartYear != nil {
			body["start_year"] = *in.StartYear
		}
		if in.PositionX != nil {
			body["position_x"] = *in.PositionX
		}
		if in.PositionY != nil {
			body["position_y"] = *in.PositionY
		}
		if in.PositionYIsExplicit != nil {
			body["position_y_is_explicit"] = *in.PositionYIsExplicit
		}
		if in.Prerequisites != nil {
			body["prerequisites"] = in.Prerequisites
		}
		if in.Categories != nil {
			body["categories"] = in.Categories
		}
		if in.EnableEquipments != nil {
			body["enable_equipments"] = in.EnableEquipments
		}
		if in.Doctrine != nil {
			body["doctrine"] = *in.Doctrine
		}
		if in.OnResearchComplete != nil {
			body["on_research_complete"] = *in.OnResearchComplete
		}
		if in.CountryTag != nil {
			body["country_tag"] = *in.CountryTag
		}
		if in.RawExtras != nil {
			body["raw_extras"] = in.RawExtras
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
			body["display_name_locs"] = in.DisplayNameLocs
		}
		if in.DescriptionLocs != nil {
			body["description_locs"] = in.DescriptionLocs
		}
		if in.AIWillDo != nil {
			body["ai_will_do"] = *in.AIWillDo
		}

		countryModNonEmpty := in.CountryModifier != nil && *in.CountryModifier != ""
		if in.Modifiers != nil || countryModNonEmpty {
			mods := map[string]any{}
			for k, v := range in.Modifiers {
				mods[k] = v
			}
			if countryModNonEmpty {
				mods["__modifier"] = *in.CountryModifier
			}
			body["modifiers"] = mods
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/technologies/%d", in.ProjectID, in.TechDBID), body)
		if err != nil {
			return nil, updateTechnologyOut{}, err
		}

		out := updateTechnologyOut{
			TechID: stringField(resp, "tech_id"),
		}
		out.TechDBID = techdocInt64Field(resp, "id")
		return nil, out, nil
	})
}

type createEquipmentIn struct {
	ProjectID     int64          `json:"project_id"`
	Name          string         `json:"name,omitempty"`
	EquipmentID   string         `json:"equipment_id,omitempty" jsonschema:"auto-suffixed _1.._1000 on collision -- READ BACK the returned equipment_id"`
	EquipmentType string         `json:"equipment_type,omitempty"`
	IsArchetype   *bool          `json:"is_archetype,omitempty" jsonschema:"default true -- set false for a standalone piece, or use create_equipment_variant for a concrete archetype variant"`
	Parent        string         `json:"parent,omitempty" jsonschema:"STRING equipment-id reference, not an int FK"`
	CountryTag    string         `json:"country_tag,omitempty" jsonschema:"uppercased on store; blank = global"`
	Year          *int           `json:"year,omitempty"`
	Picture       string         `json:"picture,omitempty"`
	IsBuildable   *bool          `json:"is_buildable,omitempty"`
	CanLicense    *bool          `json:"can_license,omitempty"`
	SoftAttack    *float64       `json:"soft_attack,omitempty" jsonschema:"None = inherit from archetype, NOT zero -- never send 0 to mean inherit"`
	HardAttack    *float64       `json:"hard_attack,omitempty"`
	APAttack      *float64       `json:"ap_attack,omitempty"`
	AirAttack     *float64       `json:"air_attack,omitempty"`
	Breakthrough  *float64       `json:"breakthrough,omitempty"`
	Defense       *float64       `json:"defense,omitempty"`
	ArmorValue    *float64       `json:"armor_value,omitempty"`
	Hardness      *float64       `json:"hardness,omitempty"`
	MaxStrength   *float64       `json:"max_strength,omitempty"`
	BuildCostIC   *float64       `json:"build_cost_ic,omitempty"`
	Reliability   *float64       `json:"reliability,omitempty"`
	MaximumSpeed  *float64       `json:"maximum_speed,omitempty"`
	Manpower      *int           `json:"manpower,omitempty"`
	Resources     map[string]any `json:"resources,omitempty"`
	NameEn        string         `json:"name_en,omitempty"`
}

type createEquipmentOut struct {
	EquipmentDBID int64    `json:"equipment_db_id"`
	EquipmentID   string   `json:"equipment_id"`
	IsArchetype   bool     `json:"is_archetype"`
	Archetype     string   `json:"archetype"`
	Warnings      []string `json:"warnings"`
}

func registerCreateEquipment(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_equipment",
		Description: "Create an equipment ARCHETYPE (the parent, default) or a standalone piece " +
			"(is_archetype=false). Use create_equipment_variant for a concrete variant of an existing " +
			"archetype. Every numeric stat is Optional and None means 'inherit from the archetype', " +
			"NOT zero -- never send 0 to mean inherit.",
	}, func(ctx context.Context, _ *CallToolRequest, in createEquipmentIn) (*CallToolResult, createEquipmentOut, error) {
		isArchetype := boolOr(in.IsArchetype, true)
		resources := in.Resources
		if resources == nil {
			resources = map[string]any{}
		}

		body := map[string]any{
			"equipment_id":   in.EquipmentID,
			"name":           in.Name,
			"equipment_type": in.EquipmentType,
			"is_archetype":   isArchetype,
			"archetype":      "",
			"parent":         in.Parent,
			"country_tag":    in.CountryTag,
			"picture":        in.Picture,
			"resources":      resources,
			"name_en":        in.NameEn,
		}
		addEquipmentStatFields(body, equipmentStatFields{
			Year: in.Year, IsBuildable: in.IsBuildable, CanLicense: in.CanLicense,
			SoftAttack: in.SoftAttack, HardAttack: in.HardAttack, APAttack: in.APAttack,
			AirAttack: in.AirAttack, Breakthrough: in.Breakthrough, Defense: in.Defense,
			ArmorValue: in.ArmorValue, Hardness: in.Hardness, MaxStrength: in.MaxStrength,
			BuildCostIC: in.BuildCostIC, Reliability: in.Reliability, MaximumSpeed: in.MaximumSpeed,
			Manpower: in.Manpower,
		})

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/equipment", in.ProjectID), body)
		if err != nil {
			return nil, createEquipmentOut{}, err
		}

		out := createEquipmentOut{
			EquipmentID: stringField(resp, "equipment_id"),
			Archetype:   stringField(resp, "archetype"),
			Warnings:    techdocStringSliceField(resp, "warnings"),
		}
		out.EquipmentDBID = techdocInt64Field(resp, "id")
		out.IsArchetype = techdocBoolField(resp, "is_archetype")
		return nil, out, nil
	})
}

type createEquipmentVariantIn struct {
	ProjectID     int64          `json:"project_id"`
	Archetype     string         `json:"archetype" jsonschema:"the PARENT archetype's equipment_id STRING -- the parent row must already exist (create it first with create_equipment, is_archetype=true)"`
	EquipmentID   string         `json:"equipment_id,omitempty"`
	Name          string         `json:"name,omitempty"`
	EquipmentType string         `json:"equipment_type,omitempty"`
	Parent        string         `json:"parent,omitempty"`
	CountryTag    string         `json:"country_tag,omitempty"`
	Year          *int           `json:"year,omitempty"`
	Picture       string         `json:"picture,omitempty"`
	IsBuildable   *bool          `json:"is_buildable,omitempty"`
	CanLicense    *bool          `json:"can_license,omitempty"`
	SoftAttack    *float64       `json:"soft_attack,omitempty" jsonschema:"set ONLY the stats this variant overrides -- leave others unset to INHERIT; sending 0 persists an explicit 0 and breaks inheritance"`
	HardAttack    *float64       `json:"hard_attack,omitempty"`
	APAttack      *float64       `json:"ap_attack,omitempty"`
	AirAttack     *float64       `json:"air_attack,omitempty"`
	Breakthrough  *float64       `json:"breakthrough,omitempty"`
	Defense       *float64       `json:"defense,omitempty"`
	ArmorValue    *float64       `json:"armor_value,omitempty"`
	Hardness      *float64       `json:"hardness,omitempty"`
	MaxStrength   *float64       `json:"max_strength,omitempty"`
	BuildCostIC   *float64       `json:"build_cost_ic,omitempty"`
	Reliability   *float64       `json:"reliability,omitempty"`
	MaximumSpeed  *float64       `json:"maximum_speed,omitempty"`
	Manpower      *int           `json:"manpower,omitempty"`
	Resources     map[string]any `json:"resources,omitempty"`
	NameEn        string         `json:"name_en,omitempty"`
}

type createEquipmentVariantOut struct {
	EquipmentDBID int64  `json:"equipment_db_id"`
	EquipmentID   string `json:"equipment_id"`
	Archetype     string `json:"archetype"`
	IsArchetype   bool   `json:"is_archetype"`
}

func registerCreateEquipmentVariant(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_equipment_variant",
		Description: "Create a concrete VARIANT of an existing equipment archetype (is_archetype=false, " +
			"archetype = the parent's equipment_id string). The parent archetype row must ALREADY EXIST. " +
			"Set ONLY the stats this variant overrides -- leave every other stat unset so it INHERITS from " +
			"the archetype; sending 0 persists an explicit 0 and breaks inheritance.",
	}, func(ctx context.Context, _ *CallToolRequest, in createEquipmentVariantIn) (*CallToolResult, createEquipmentVariantOut, error) {
		resources := in.Resources
		if resources == nil {
			resources = map[string]any{}
		}

		body := map[string]any{
			"equipment_id":   in.EquipmentID,
			"name":           in.Name,
			"equipment_type": in.EquipmentType,
			"is_archetype":   false,
			"archetype":      in.Archetype,
			"parent":         in.Parent,
			"country_tag":    in.CountryTag,
			"picture":        in.Picture,
			"resources":      resources,
			"name_en":        in.NameEn,
		}
		addEquipmentStatFields(body, equipmentStatFields{
			Year: in.Year, IsBuildable: in.IsBuildable, CanLicense: in.CanLicense,
			SoftAttack: in.SoftAttack, HardAttack: in.HardAttack, APAttack: in.APAttack,
			AirAttack: in.AirAttack, Breakthrough: in.Breakthrough, Defense: in.Defense,
			ArmorValue: in.ArmorValue, Hardness: in.Hardness, MaxStrength: in.MaxStrength,
			BuildCostIC: in.BuildCostIC, Reliability: in.Reliability, MaximumSpeed: in.MaximumSpeed,
			Manpower: in.Manpower,
		})

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/equipment", in.ProjectID), body)
		if err != nil {
			return nil, createEquipmentVariantOut{}, err
		}

		out := createEquipmentVariantOut{
			EquipmentID: stringField(resp, "equipment_id"),
			Archetype:   stringField(resp, "archetype"),
		}
		out.EquipmentDBID = techdocInt64Field(resp, "id")
		out.IsArchetype = techdocBoolField(resp, "is_archetype")
		return nil, out, nil
	})
}

type equipmentStatFields struct {
	Year                                                      *int
	IsBuildable, CanLicense                                   *bool
	SoftAttack, HardAttack, APAttack, AirAttack, Breakthrough *float64
	Defense, ArmorValue, Hardness, MaxStrength                *float64
	BuildCostIC, Reliability, MaximumSpeed                    *float64
	Manpower                                                  *int
}

func addEquipmentStatFields(body map[string]any, f equipmentStatFields) {
	if f.Year != nil {
		body["year"] = *f.Year
	}
	if f.IsBuildable != nil {
		body["is_buildable"] = *f.IsBuildable
	}
	if f.CanLicense != nil {
		body["can_license"] = *f.CanLicense
	}
	if f.SoftAttack != nil {
		body["soft_attack"] = *f.SoftAttack
	}
	if f.HardAttack != nil {
		body["hard_attack"] = *f.HardAttack
	}
	if f.APAttack != nil {
		body["ap_attack"] = *f.APAttack
	}
	if f.AirAttack != nil {
		body["air_attack"] = *f.AirAttack
	}
	if f.Breakthrough != nil {
		body["breakthrough"] = *f.Breakthrough
	}
	if f.Defense != nil {
		body["defense"] = *f.Defense
	}
	if f.ArmorValue != nil {
		body["armor_value"] = *f.ArmorValue
	}
	if f.Hardness != nil {
		body["hardness"] = *f.Hardness
	}
	if f.MaxStrength != nil {
		body["max_strength"] = *f.MaxStrength
	}
	if f.BuildCostIC != nil {
		body["build_cost_ic"] = *f.BuildCostIC
	}
	if f.Reliability != nil {
		body["reliability"] = *f.Reliability
	}
	if f.MaximumSpeed != nil {
		body["maximum_speed"] = *f.MaximumSpeed
	}
	if f.Manpower != nil {
		body["manpower"] = *f.Manpower
	}
}

type createDoctrineIn struct {
	ProjectID             int64          `json:"project_id"`
	EntityKey             string         `json:"entity_key" jsonschema:"REQUIRED (blank -> 400); unique within (project, entity_type, entity_key) -> 409 on collision, NO auto-suffix"`
	EntityType            string         `json:"entity_type,omitempty" jsonschema:"folder | grand_doctrine (default) | track | subdoctrine"`
	FolderKey             string         `json:"folder_key,omitempty"`
	TrackKey              string         `json:"track_key,omitempty"`
	Name                  string         `json:"name,omitempty"`
	Description           string         `json:"description,omitempty"`
	Icon                  string         `json:"icon,omitempty"`
	XPCost                int            `json:"xp_cost,omitempty"`
	XPType                string         `json:"xp_type,omitempty"`
	Xor                   *string        `json:"xor,omitempty" jsonschema:"1.19 SUBDOCTRINE-ONLY -- RAW inner list of subdoctrine keys, e.g. 'other_sub_a other_sub_b' (export wraps it as xor = { ... })"`
	AllowInMultipleTracks *bool          `json:"allow_in_multiple_tracks,omitempty" jsonschema:"1.19 SUBDOCTRINE-ONLY -- the export skips this key for any other entity_type"`
	RawExtras             map[string]any `json:"raw_extras,omitempty" jsonschema:"a doctrine node's actual bonus content (modifier/milestones/rewards/mastery) -- there is NO typed modifier field. GET an existing doctrine node first to learn the exact shape"`
	SourceFile            string         `json:"source_file,omitempty" jsonschema:"relative to common/doctrines/, e.g. folders/my_doctrines.txt -- NOT a mod-relative path"`
}

type createDoctrineOut struct {
	DoctrineDBID int64  `json:"doctrine_db_id"`
	EntityType   string `json:"entity_type"`
	EntityKey    string `json:"entity_key"`
	FolderKey    string `json:"folder_key"`
	TrackKey     string `json:"track_key"`
}

func registerCreateDoctrine(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_doctrine",
		Description: "Create a modern-doctrine node in the one discriminated doctrine table. " +
			"entity_key is REQUIRED and unique within (project, entity_type, entity_key) -- 409 on " +
			"collision with NO auto-suffix. There is NO typed modifier field: a doctrine's actual bonus " +
			"content (modifier/milestones/rewards/mastery) rides in raw_extras and round-trips verbatim.",
	}, func(ctx context.Context, _ *CallToolRequest, in createDoctrineIn) (*CallToolResult, createDoctrineOut, error) {
		entityType := in.EntityType
		if entityType == "" {
			entityType = "grand_doctrine"
		}
		rawExtras := in.RawExtras
		if rawExtras == nil {
			rawExtras = map[string]any{}
		}

		body := map[string]any{
			"entity_type": entityType,
			"entity_key":  in.EntityKey,
			"folder_key":  in.FolderKey,
			"track_key":   in.TrackKey,
			"name":        in.Name,
			"description": in.Description,
			"icon":        in.Icon,
			"xp_cost":     in.XPCost,
			"xp_type":     in.XPType,
			"raw_extras":  rawExtras,
			"source_file": in.SourceFile,
		}
		if in.Xor != nil {
			body["xor"] = *in.Xor
		}
		if in.AllowInMultipleTracks != nil {
			body["allow_in_multiple_tracks"] = *in.AllowInMultipleTracks
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/doctrines", in.ProjectID), body)
		if err != nil {
			return nil, createDoctrineOut{}, err
		}

		out := createDoctrineOut{
			EntityType: stringField(resp, "entity_type"),
			EntityKey:  stringField(resp, "entity_key"),
			FolderKey:  stringField(resp, "folder_key"),
			TrackKey:   stringField(resp, "track_key"),
		}
		out.DoctrineDBID = techdocInt64Field(resp, "id")
		return nil, out, nil
	})
}

type updateDoctrineIn struct {
	ProjectID             int64          `json:"project_id"`
	DoctrineDBID          int64          `json:"doctrine_db_id" jsonschema:"numeric doctrine_db_id from create_doctrine"`
	EntityType            *string        `json:"entity_type,omitempty"`
	EntityKey             *string        `json:"entity_key,omitempty"`
	FolderKey             *string        `json:"folder_key,omitempty"`
	TrackKey              *string        `json:"track_key,omitempty"`
	Name                  *string        `json:"name,omitempty"`
	Description           *string        `json:"description,omitempty"`
	Icon                  *string        `json:"icon,omitempty"`
	XPCost                *int           `json:"xp_cost,omitempty"`
	XPType                *string        `json:"xp_type,omitempty"`
	Xor                   *string        `json:"xor,omitempty" jsonschema:"RAW inner list, e.g. 'other_sub_a other_sub_b' -- only emits for entity_type=subdoctrine"`
	AllowInMultipleTracks *bool          `json:"allow_in_multiple_tracks,omitempty" jsonschema:"only emits for entity_type=subdoctrine"`
	RawExtras             map[string]any `json:"raw_extras,omitempty" jsonschema:"REPLACES the whole dict -- GET the node first and send it back merged"`
}

type updateDoctrineOut struct {
	DoctrineDBID          int64  `json:"doctrine_db_id"`
	EntityType            string `json:"entity_type"`
	EntityKey             string `json:"entity_key"`
	Xor                   string `json:"xor,omitempty"`
	AllowInMultipleTracks bool   `json:"allow_in_multiple_tracks"`
}

func registerUpdateDoctrine(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_doctrine",
		Description: "Edit an EXISTING doctrine node by numeric doctrine_db_id. Only the fields you " +
			"supply are changed; raw_extras REPLACES the whole dict -- GET the node first and send it " +
			"back merged. xor / allow_in_multiple_tracks only emit for entity_type=subdoctrine.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateDoctrineIn) (*CallToolResult, updateDoctrineOut, error) {
		body := map[string]any{}
		if in.EntityType != nil {
			body["entity_type"] = *in.EntityType
		}
		if in.EntityKey != nil {
			body["entity_key"] = *in.EntityKey
		}
		if in.FolderKey != nil {
			body["folder_key"] = *in.FolderKey
		}
		if in.TrackKey != nil {
			body["track_key"] = *in.TrackKey
		}
		if in.Name != nil {
			body["name"] = *in.Name
		}
		if in.Description != nil {
			body["description"] = *in.Description
		}
		if in.Icon != nil {
			body["icon"] = *in.Icon
		}
		if in.XPCost != nil {
			body["xp_cost"] = *in.XPCost
		}
		if in.XPType != nil {
			body["xp_type"] = *in.XPType
		}
		if in.Xor != nil {
			body["xor"] = *in.Xor
		}
		if in.AllowInMultipleTracks != nil {
			body["allow_in_multiple_tracks"] = *in.AllowInMultipleTracks
		}
		if in.RawExtras != nil {
			body["raw_extras"] = in.RawExtras
		}

		resp, err := proxyAuthedCall(ctx, proxy, "PUT",
			fmt.Sprintf("/api/projects/%d/doctrines/%d", in.ProjectID, in.DoctrineDBID), body)
		if err != nil {
			return nil, updateDoctrineOut{}, err
		}

		out := updateDoctrineOut{
			EntityType:            stringField(resp, "entity_type"),
			EntityKey:             stringField(resp, "entity_key"),
			Xor:                   stringField(resp, "xor"),
			AllowInMultipleTracks: techdocBoolField(resp, "allow_in_multiple_tracks"),
		}
		out.DoctrineDBID = techdocInt64Field(resp, "id")
		return nil, out, nil
	})
}

func techdocFloatOr(p *float64, fallback float64) float64 {
	if p == nil {
		return fallback
	}
	return *p
}

func techdocInt64Field(m map[string]any, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func techdocStringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func techdocBoolField(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
