package toolapi

import (
	"context"
	"fmt"
)

func emptyIfNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func emptyIfNilMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

type createSubUnitIn struct {
	ProjectID              int64          `json:"project_id"`
	SubUnitID              string         `json:"sub_unit_id" jsonschema:"the unit TOKEN division templates reference; ASCII, mod-prefixed — AUTO-SUFFIXES on collision, read back the response"`
	CountryTag             string         `json:"country_tag,omitempty"`
	NameEn                 string         `json:"name_en,omitempty" jsonschema:"fill both name_en and name_zh, or the recruitment UI shows the raw token"`
	NameZh                 string         `json:"name_zh,omitempty"`
	DescEn                 string         `json:"desc_en,omitempty"`
	DescZh                 string         `json:"desc_zh,omitempty"`
	Abbreviation           string         `json:"abbreviation,omitempty"`
	Sprite                 string         `json:"sprite,omitempty"`
	MapIconCategory        string         `json:"map_icon_category,omitempty"`
	Group                  string         `json:"group,omitempty" jsonschema:"the recruitment-tab group"`
	Transport              string         `json:"transport,omitempty"`
	Priority               *int           `json:"priority,omitempty"`
	AiPriority             *int           `json:"ai_priority,omitempty"`
	Active                 *bool          `json:"active,omitempty" jsonschema:"true = recruitable without a tech unlock; otherwise unlock via a technology's enable_subunits"`
	Cavalry                *bool          `json:"cavalry,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	SpecialForces          *bool          `json:"special_forces,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	Marines                *bool          `json:"marines,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	Mountaineers           *bool          `json:"mountaineers,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	CanBeParachuted        *bool          `json:"can_be_parachuted,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	CanExfiltrateFromCoast *bool          `json:"can_exfiltrate_from_coast,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	AffectsSpeed           *bool          `json:"affects_speed,omitempty" jsonschema:"tri-state: omit = engine default, false = explicit no"`
	UnitTypes              []string       `json:"unit_types,omitempty" jsonschema:"emits as HOI4 type = { ... }"`
	Categories             []string       `json:"categories,omitempty" jsonschema:"unit-category tags"`
	Essential              []string       `json:"essential,omitempty"`
	Need                   map[string]any `json:"need,omitempty" jsonschema:"{equipment_archetype: amount}"`
	Stats                  map[string]any `json:"stats,omitempty" jsonschema:"catch-all numeric stats: maxspeed, max_strength, max_organisation, default_morale, combat_width, manpower, training_time, supply_consumption, suppression, ..."`
	TerrainModifiers       map[string]any `json:"terrain_modifiers,omitempty" jsonschema:"{terrain: {modifier: value}} — standard terrain keys only"`
}

type createSubUnitOut struct {
	SubUnitDBID int64  `json:"sub_unit_db_id"`
	SubUnitID   string `json:"sub_unit_id"`
	CountryTag  string `json:"country_tag"`
	Group       string `json:"group"`
}

func registerCreateSubUnit(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_sub_unit",
		Description: "Create a custom battalion/company type (a common/units/*.txt sub-unit). " +
			"sub_unit_id auto-suffixes on collision — read back the response before referencing it " +
			"from a division template.",
	}, func(ctx context.Context, _ *CallToolRequest, in createSubUnitIn) (*CallToolResult, createSubUnitOut, error) {
		body := map[string]any{
			"sub_unit_id":               in.SubUnitID,
			"country_tag":               in.CountryTag,
			"name_en":                   in.NameEn,
			"name_zh":                   in.NameZh,
			"desc_en":                   in.DescEn,
			"desc_zh":                   in.DescZh,
			"abbreviation":              in.Abbreviation,
			"sprite":                    in.Sprite,
			"map_icon_category":         in.MapIconCategory,
			"group":                     in.Group,
			"transport":                 in.Transport,
			"priority":                  in.Priority,
			"ai_priority":               in.AiPriority,
			"active":                    in.Active,
			"cavalry":                   in.Cavalry,
			"special_forces":            in.SpecialForces,
			"marines":                   in.Marines,
			"mountaineers":              in.Mountaineers,
			"can_be_parachuted":         in.CanBeParachuted,
			"can_exfiltrate_from_coast": in.CanExfiltrateFromCoast,
			"affects_speed":             in.AffectsSpeed,
			"unit_types":                emptyIfNilStrings(in.UnitTypes),
			"categories":                emptyIfNilStrings(in.Categories),
			"essential":                 emptyIfNilStrings(in.Essential),
			"need":                      emptyIfNilMap(in.Need),
			"stats":                     emptyIfNilMap(in.Stats),
			"terrain_modifiers":         emptyIfNilMap(in.TerrainModifiers),
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/sub-units", in.ProjectID), body)
		if err != nil {
			return nil, createSubUnitOut{}, err
		}

		return nil, createSubUnitOut{
			SubUnitDBID: int64Field(resp, "id"),
			SubUnitID:   stringField(resp, "sub_unit_id"),
			CountryTag:  stringField(resp, "country_tag"),
			Group:       stringField(resp, "group"),
		}, nil
	})
}

type createAirWingGroupIn struct {
	ProjectID  int64  `json:"project_id"`
	Location   string `json:"location" jsonschema:"the air base's STATE id as a STRING token ('798', not 798) — validated on write"`
	OobFile    string `json:"oob_file,omitempty" jsonschema:"the same group stem used by division templates/deployments, e.g. 'USA_1936'"`
	CountryTag string `json:"country_tag,omitempty"`
	Ordering   int    `json:"ordering,omitempty"`
}

type createAirWingGroupOut struct {
	AirWingGroupID int64  `json:"air_wing_group_id"`
	Location       string `json:"location"`
	OobFile        string `json:"oob_file"`
	CountryTag     string `json:"country_tag"`
}

func registerCreateAirWingGroup(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_air_wing_group",
		Description: "Create one air base entry in a country's starting air OOB. Wings hang off the " +
			"returned air_wing_group_id via add_air_wing — the air/naval chain links by INT db id.",
	}, func(ctx context.Context, _ *CallToolRequest, in createAirWingGroupIn) (*CallToolResult, createAirWingGroupOut, error) {
		body := map[string]any{
			"location":    in.Location,
			"oob_file":    in.OobFile,
			"country_tag": in.CountryTag,
			"ordering":    in.Ordering,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/oob-units/air-wing-groups", in.ProjectID), body)
		if err != nil {
			return nil, createAirWingGroupOut{}, err
		}

		return nil, createAirWingGroupOut{
			AirWingGroupID: int64Field(resp, "id"),
			Location:       stringField(resp, "location"),
			OobFile:        stringField(resp, "oob_file"),
			CountryTag:     stringField(resp, "country_tag"),
		}, nil
	})
}

type addAirWingIn struct {
	ProjectID     int64  `json:"project_id"`
	GroupID       int64  `json:"group_id" jsonschema:"the air_wing_group_id from create_air_wing_group — 404 if not in this project"`
	EquipmentType string `json:"equipment_type" jsonschema:"the plane equipment token (e.g. 'fighter_equipment_0') — becomes the wing's block key"`
	Amount        string `json:"amount,omitempty" jsonschema:"number of planes as a STRING token, e.g. '100'"`
	Owner         string `json:"owner,omitempty" jsonschema:"country tag providing them; usually blank = the OOB's own country"`
	Name          string `json:"name,omitempty"`
	Ordering      int    `json:"ordering,omitempty"`
}

type addAirWingOut struct {
	AirWingID     int64  `json:"air_wing_id"`
	GroupID       int64  `json:"group_id"`
	EquipmentType string `json:"equipment_type"`
	Amount        string `json:"amount"`
}

func registerAddAirWing(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "add_air_wing",
		Description: "Add a plane wing to an air-wing group (create the group FIRST with create_air_wing_group).",
	}, func(ctx context.Context, _ *CallToolRequest, in addAirWingIn) (*CallToolResult, addAirWingOut, error) {
		body := map[string]any{
			"group_id":       in.GroupID,
			"equipment_type": in.EquipmentType,
			"amount":         in.Amount,
			"owner":          in.Owner,
			"name":           in.Name,
			"ordering":       in.Ordering,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/oob-units/air-wings", in.ProjectID), body)
		if err != nil {
			return nil, addAirWingOut{}, err
		}

		return nil, addAirWingOut{
			AirWingID:     int64Field(resp, "id"),
			GroupID:       int64Field(resp, "group_id"),
			EquipmentType: stringField(resp, "equipment_type"),
			Amount:        stringField(resp, "amount"),
		}, nil
	})
}

type createFleetIn struct {
	ProjectID  int64  `json:"project_id"`
	Name       string `json:"name,omitempty"`
	NavalBase  string `json:"naval_base,omitempty" jsonschema:"the PROVINCE id of the home naval base, as a STRING token ('9601')"`
	OobFile    string `json:"oob_file,omitempty" jsonschema:"the same group stem as the rest of the country's starting OOB"`
	CountryTag string `json:"country_tag,omitempty"`
	Ordering   int    `json:"ordering,omitempty"`
}

type createFleetOut struct {
	FleetID   int64  `json:"fleet_id"`
	Name      string `json:"name"`
	NavalBase string `json:"naval_base"`
	OobFile   string `json:"oob_file"`
}

func registerCreateFleet(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_fleet",
		Description: "Create a starting-navy fleet (top of the fleet -> task force -> ship chain). " +
			"Then: add_task_force(fleet_id=...) -> add_ship(task_force_id=...).",
	}, func(ctx context.Context, _ *CallToolRequest, in createFleetIn) (*CallToolResult, createFleetOut, error) {
		body := map[string]any{
			"name":        in.Name,
			"naval_base":  in.NavalBase,
			"oob_file":    in.OobFile,
			"country_tag": in.CountryTag,
			"ordering":    in.Ordering,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/oob-units/fleets", in.ProjectID), body)
		if err != nil {
			return nil, createFleetOut{}, err
		}

		return nil, createFleetOut{
			FleetID:   int64Field(resp, "id"),
			Name:      stringField(resp, "name"),
			NavalBase: stringField(resp, "naval_base"),
			OobFile:   stringField(resp, "oob_file"),
		}, nil
	})
}

type addTaskForceIn struct {
	ProjectID int64  `json:"project_id"`
	FleetID   int64  `json:"fleet_id" jsonschema:"the fleet_id from create_fleet — 404 if not in this project"`
	Name      string `json:"name,omitempty"`
	Location  string `json:"location,omitempty" jsonschema:"the province id where the task force starts, as a STRING token (usually the fleet's naval base)"`
	Ordering  int    `json:"ordering,omitempty"`
}

type addTaskForceOut struct {
	TaskForceID int64  `json:"task_force_id"`
	FleetID     int64  `json:"fleet_id"`
	Name        string `json:"name"`
	Location    string `json:"location"`
}

func registerAddTaskForce(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "add_task_force",
		Description: "Add a task force to a fleet (create the fleet FIRST with create_fleet).",
	}, func(ctx context.Context, _ *CallToolRequest, in addTaskForceIn) (*CallToolResult, addTaskForceOut, error) {
		body := map[string]any{
			"fleet_id": in.FleetID,
			"name":     in.Name,
			"location": in.Location,
			"ordering": in.Ordering,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/oob-units/task-forces", in.ProjectID), body)
		if err != nil {
			return nil, addTaskForceOut{}, err
		}

		return nil, addTaskForceOut{
			TaskForceID: int64Field(resp, "id"),
			FleetID:     int64Field(resp, "fleet_id"),
			Name:        stringField(resp, "name"),
			Location:    stringField(resp, "location"),
		}, nil
	})
}

type addShipIn struct {
	ProjectID             int64  `json:"project_id"`
	TaskForceID           int64  `json:"task_force_id" jsonschema:"the task_force_id from add_task_force — 404 if not in this project"`
	Name                  string `json:"name,omitempty"`
	Definition            string `json:"definition,omitempty" jsonschema:"the ship's hull class token (e.g. 'destroyer', 'battleship')"`
	EquipmentType         string `json:"equipment_type,omitempty" jsonschema:"the hull equipment token"`
	Amount                string `json:"amount,omitempty" jsonschema:"STRING token, e.g. '1'"`
	Owner                 string `json:"owner,omitempty"`
	VersionName           string `json:"version_name,omitempty" jsonschema:"display name of an equipment VARIANT (matches a variant in the Equipment workbench) — that's how a ship gets its stats"`
	StartExperienceFactor string `json:"start_experience_factor,omitempty" jsonschema:"STRING token, e.g. '0.25'"`
	Ordering              int    `json:"ordering,omitempty"`
}

type addShipOut struct {
	ShipID        int64  `json:"ship_id"`
	TaskForceID   int64  `json:"task_force_id"`
	Name          string `json:"name"`
	Definition    string `json:"definition"`
	EquipmentType string `json:"equipment_type"`
	VersionName   string `json:"version_name"`
}

func registerAddShip(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "add_ship",
		Description: "Add a ship to a task force (create the fleet + task force FIRST). Empty string on any scalar = the line is omitted from the emit.",
	}, func(ctx context.Context, _ *CallToolRequest, in addShipIn) (*CallToolResult, addShipOut, error) {
		body := map[string]any{
			"task_force_id":           in.TaskForceID,
			"name":                    in.Name,
			"definition":              in.Definition,
			"equipment_type":          in.EquipmentType,
			"amount":                  in.Amount,
			"owner":                   in.Owner,
			"version_name":            in.VersionName,
			"start_experience_factor": in.StartExperienceFactor,
			"ordering":                in.Ordering,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/oob-units/ships", in.ProjectID), body)
		if err != nil {
			return nil, addShipOut{}, err
		}

		return nil, addShipOut{
			ShipID:        int64Field(resp, "id"),
			TaskForceID:   int64Field(resp, "task_force_id"),
			Name:          stringField(resp, "name"),
			Definition:    stringField(resp, "definition"),
			EquipmentType: stringField(resp, "equipment_type"),
			VersionName:   stringField(resp, "version_name"),
		}, nil
	})
}
