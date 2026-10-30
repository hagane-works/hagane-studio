package toolapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func mapSliceField(m map[string]any, key string) []map[string]any {
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mm, ok := item.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

func boolPtrField(m map[string]any, key string) *bool {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	if b, ok := v.(bool); ok {
		return &b
	}
	return nil
}

type listRawTemplatesIn struct {
	ProjectID int64 `json:"project_id"`
}

type rawTemplateListingEntry struct {
	Key          string            `json:"key"`
	LabelEn      string            `json:"label_en"`
	LabelZh      string            `json:"label_zh"`
	DescEn       string            `json:"desc_en"`
	DescZh       string            `json:"desc_zh"`
	Labels       map[string]string `json:"labels"`
	Descriptions map[string]string `json:"descriptions"`
	MainParam    string            `json:"main_param"`
	RoutePath    string            `json:"route_path"`
}

type listRawTemplatesOut struct {
	ProjectID int64                     `json:"project_id"`
	Templates []rawTemplateListingEntry `json:"templates"`
}

func registerListRawTemplates(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "list_raw_templates",
		Description: "List the raw-file starting-point templates the program can generate (e.g. " +
			"scripted_gui overview panel, on_actions hooks, game_rules, ai_strategy, GFX sprite, defines " +
			"tweaks, balance-of-power, music station, peace conference). Each entry has: key (pass it as " +
			"`template` to create_raw_from_template), eight-locale labels/descriptions maps " +
			"(en/zh/ru/de/es/fr/ja/ko), legacy label_en/label_zh + desc_en/desc_zh fields, and main_param " +
			"(the naming-seed parameter the template derives names from). Every template writes to FIXED, " +
			"program-chosen mod-relative paths derived from your naming seed — you never choose the path.",
	}, func(ctx context.Context, _ *CallToolRequest, in listRawTemplatesIn) (*CallToolResult, listRawTemplatesOut, error) {
		resp, err := proxyAuthedCall(ctx, proxy, "GET",
			fmt.Sprintf("/api/projects/%d/raw-files/templates", in.ProjectID), nil)
		if err != nil {
			return nil, listRawTemplatesOut{}, err
		}

		rawEntries := mapSliceField(resp, "templates")
		out := listRawTemplatesOut{
			ProjectID: in.ProjectID,
			Templates: make([]rawTemplateListingEntry, 0, len(rawEntries)),
		}
		for _, t := range rawEntries {
			out.Templates = append(out.Templates, rawTemplateListingEntry{
				Key:     stringField(t, "key"),
				LabelEn: stringField(t, "label_en"),
				LabelZh: stringField(t, "label_zh"),
				DescEn:  stringField(t, "desc_en"),
				DescZh:  stringField(t, "desc_zh"),

				Labels:       orEmptyMap(stringMapField(t, "labels")),
				Descriptions: orEmptyMap(stringMapField(t, "descriptions")),
				MainParam:    stringField(t, "main_param"),
				RoutePath:    stringField(t, "route_path"),
			})
		}
		return nil, out, nil
	})
}

type listUiPanelsIn struct {
	ProjectID int64 `json:"project_id"`
}

type uiPanelListingEntry struct {
	PanelID     int64  `json:"panel_id"`
	Token       string `json:"token"`
	ContextType string `json:"context_type"`
	CountryTag  string `json:"country_tag"`
	NameEn      string `json:"name_en"`
	NameZh      string `json:"name_zh"`
	WindowName  string `json:"window_name"`
}

type listUiPanelsOut struct {
	ProjectID int64                 `json:"project_id"`
	Panels    []uiPanelListingEntry `json:"panels"`
	Total     int64                 `json:"total"`
}

func registerListUiPanels(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "list_ui_panels",
		Description: "List the project's ui-panels (visual scripted_gui builder): panel_id, token, " +
			"context_type and the derived window_name (every emitted .gui/.txt/loc name derives from the " +
			"token by construction).",
	}, func(ctx context.Context, _ *CallToolRequest, in listUiPanelsIn) (*CallToolResult, listUiPanelsOut, error) {
		resp, err := proxyAuthedCall(ctx, proxy, "GET",
			fmt.Sprintf("/api/projects/%d/ui-panels", in.ProjectID), nil)
		if err != nil {
			return nil, listUiPanelsOut{}, err
		}

		rawPanels := mapSliceField(resp, "panels")
		out := listUiPanelsOut{
			ProjectID: in.ProjectID,
			Panels:    make([]uiPanelListingEntry, 0, len(rawPanels)),
			Total:     int64Field(resp, "total"),
		}
		for _, p := range rawPanels {
			windowName := uiPanelWindowName(p)
			out.Panels = append(out.Panels, uiPanelListingEntry{
				PanelID:     int64Field(p, "id"),
				Token:       stringField(p, "token"),
				ContextType: stringField(p, "context_type"),
				CountryTag:  stringField(p, "country_tag"),
				NameEn:      stringField(p, "name_en"),
				NameZh:      stringField(p, "name_zh"),
				WindowName:  windowName,
			})
		}
		return nil, out, nil
	})
}

type getSubUnitsIn struct {
	ProjectID  int64  `json:"project_id"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"filter to one country's own sub-units; omit for the whole project"`
}

type subUnitListingEntry struct {
	SubUnitDBID int64  `json:"sub_unit_db_id"`
	SubUnitID   string `json:"sub_unit_id"`
	NameEn      string `json:"name_en"`
	NameZh      string `json:"name_zh"`
	Group       string `json:"group"`
	Active      *bool  `json:"active"`
	CountryTag  string `json:"country_tag"`
}

type getSubUnitsOut struct {
	ProjectID int64                 `json:"project_id"`
	Total     int                   `json:"total"`
	SubUnits  []subUnitListingEntry `json:"sub_units"`
}

func registerGetSubUnits(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_sub_units",
		Description: "List the project's custom sub-units (tokens + names + db ids). Use this to " +
			"resolve the exact sub_unit_id tokens available for a division template's regiments/support " +
			"grids, or to check what a previous create auto-suffixed to. Vanilla battalion tokens " +
			"(infantry, artillery_brigade, ...) are always usable in templates and are NOT listed here — " +
			"this is only the project's own definitions.",
	}, func(ctx context.Context, _ *CallToolRequest, in getSubUnitsIn) (*CallToolResult, getSubUnitsOut, error) {
		path := fmt.Sprintf("/api/projects/%d/sub-units", in.ProjectID)
		if in.CountryTag != "" {
			path += "?country_tag=" + url.QueryEscape(in.CountryTag)
		}
		rows, err := proxyAuthedCallList(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, getSubUnitsOut{}, err
		}

		out := getSubUnitsOut{
			ProjectID: in.ProjectID,
			Total:     len(rows),
			SubUnits:  make([]subUnitListingEntry, 0, len(rows)),
		}
		for _, r := range rows {
			out.SubUnits = append(out.SubUnits, subUnitListingEntry{
				SubUnitDBID: int64Field(r, "id"),
				SubUnitID:   stringField(r, "sub_unit_id"),
				NameEn:      stringField(r, "name_en"),
				NameZh:      stringField(r, "name_zh"),
				Group:       stringField(r, "group"),
				Active:      boolPtrField(r, "active"),
				CountryTag:  stringField(r, "country_tag"),
			})
		}
		return nil, out, nil
	})
}

const oobSnapshotRowCap = 200

type oobAirWingGroupRow struct {
	ID         int64  `json:"id"`
	OobFile    string `json:"oob_file"`
	CountryTag string `json:"country_tag"`
	Location   string `json:"location"`
	Ordering   int64  `json:"ordering"`
}

type oobAirWingRow struct {
	ID            int64  `json:"id"`
	GroupID       int64  `json:"group_id"`
	EquipmentType string `json:"equipment_type"`
	Amount        string `json:"amount"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
}

type oobFleetRow struct {
	ID         int64  `json:"id"`
	OobFile    string `json:"oob_file"`
	CountryTag string `json:"country_tag"`
	Name       string `json:"name"`
	NavalBase  string `json:"naval_base"`
}

type oobTaskForceRow struct {
	ID       int64  `json:"id"`
	FleetID  int64  `json:"fleet_id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type oobShipRow struct {
	ID            int64  `json:"id"`
	TaskForceID   int64  `json:"task_force_id"`
	Name          string `json:"name"`
	Definition    string `json:"definition"`
	EquipmentType string `json:"equipment_type"`
	Amount        string `json:"amount"`
	Owner         string `json:"owner"`
	VersionName   string `json:"version_name"`
}

type getOobSnapshotIn struct {
	ProjectID int64  `json:"project_id"`
	OobFile   string `json:"oob_file,omitempty" jsonschema:"narrow to one group stem (the SAME stem used by division templates/deployments)"`
}

type getOobSnapshotOut struct {
	ProjectID          int64                `json:"project_id"`
	OobFile            string               `json:"oob_file"`
	AirWingGroupsTotal int                  `json:"air_wing_groups_total"`
	AirWingGroups      []oobAirWingGroupRow `json:"air_wing_groups"`
	AirWingsTotal      int                  `json:"air_wings_total"`
	AirWings           []oobAirWingRow      `json:"air_wings"`
	FleetsTotal        int                  `json:"fleets_total"`
	Fleets             []oobFleetRow        `json:"fleets"`
	TaskForcesTotal    int                  `json:"task_forces_total"`
	TaskForces         []oobTaskForceRow    `json:"task_forces"`
	ShipsTotal         int                  `json:"ships_total"`
	Ships              []oobShipRow         `json:"ships"`
}

func oobCappedRows[T any](full []map[string]any, build func(map[string]any) T) []T {
	n := len(full)
	if n > oobSnapshotRowCap {
		n = oobSnapshotRowCap
	}
	out := make([]T, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, build(full[i]))
	}
	return out
}

func registerGetOobSnapshot(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_oob_snapshot",
		Description: "Read the whole air+naval OOB hierarchy in one call (resolve int FKs). Returns " +
			"the five collections (air_wing_groups, air_wings, fleets, task_forces, ships) with their " +
			"INT ids and parent ids, so you can extend an existing OOB without guessing ids. Pass " +
			"oob_file to narrow to one group stem. Each collection is capped at 200 rows (per-collection " +
			"`*_total` fields carry the true counts).",
	}, func(ctx context.Context, _ *CallToolRequest, in getOobSnapshotIn) (*CallToolResult, getOobSnapshotOut, error) {
		path := fmt.Sprintf("/api/projects/%d/oob-units/snapshot", in.ProjectID)
		if in.OobFile != "" {
			path += "?oob_file=" + url.QueryEscape(in.OobFile)
		}
		resp, err := proxyAuthedCall(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, getOobSnapshotOut{}, err
		}

		groups := mapSliceField(resp, "air_wing_groups")
		wings := mapSliceField(resp, "air_wings")
		fleets := mapSliceField(resp, "fleets")
		taskForces := mapSliceField(resp, "task_forces")
		ships := mapSliceField(resp, "ships")

		out := getOobSnapshotOut{
			ProjectID:          in.ProjectID,
			OobFile:            in.OobFile,
			AirWingGroupsTotal: len(groups),
			AirWingGroups: oobCappedRows(groups, func(m map[string]any) oobAirWingGroupRow {
				return oobAirWingGroupRow{
					ID:         int64Field(m, "id"),
					OobFile:    stringField(m, "oob_file"),
					CountryTag: stringField(m, "country_tag"),
					Location:   stringField(m, "location"),
					Ordering:   int64Field(m, "ordering"),
				}
			}),
			AirWingsTotal: len(wings),
			AirWings: oobCappedRows(wings, func(m map[string]any) oobAirWingRow {
				return oobAirWingRow{
					ID:            int64Field(m, "id"),
					GroupID:       int64Field(m, "group_id"),
					EquipmentType: stringField(m, "equipment_type"),
					Amount:        stringField(m, "amount"),
					Owner:         stringField(m, "owner"),
					Name:          stringField(m, "name"),
				}
			}),
			FleetsTotal: len(fleets),
			Fleets: oobCappedRows(fleets, func(m map[string]any) oobFleetRow {
				return oobFleetRow{
					ID:         int64Field(m, "id"),
					OobFile:    stringField(m, "oob_file"),
					CountryTag: stringField(m, "country_tag"),
					Name:       stringField(m, "name"),
					NavalBase:  stringField(m, "naval_base"),
				}
			}),
			TaskForcesTotal: len(taskForces),
			TaskForces: oobCappedRows(taskForces, func(m map[string]any) oobTaskForceRow {
				return oobTaskForceRow{
					ID:       int64Field(m, "id"),
					FleetID:  int64Field(m, "fleet_id"),
					Name:     stringField(m, "name"),
					Location: stringField(m, "location"),
				}
			}),
			ShipsTotal: len(ships),
			Ships: oobCappedRows(ships, func(m map[string]any) oobShipRow {
				return oobShipRow{
					ID:            int64Field(m, "id"),
					TaskForceID:   int64Field(m, "task_force_id"),
					Name:          stringField(m, "name"),
					Definition:    stringField(m, "definition"),
					EquipmentType: stringField(m, "equipment_type"),
					Amount:        stringField(m, "amount"),
					Owner:         stringField(m, "owner"),
					VersionName:   stringField(m, "version_name"),
				}
			}),
		}
		return nil, out, nil
	})
}

const getStatesRowCap = 200

type getStatesIn struct {
	ProjectID  int64  `json:"project_id"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"return only states this country OWNS"`
	NameQuery  string `json:"name_query,omitempty" jsonschema:"case-insensitive contains match on the display name in ANY language the project stores, or on the raw loc key, plus exact state_id match"`
}

type stateListingEntry struct {
	StateDBID     int64             `json:"state_db_id"`
	StateID       int64             `json:"state_id"`
	Name          string            `json:"name" jsonschema:"the raw HOI4 loc key (STATE_123) -- display_name is what a human reads"`
	DisplayName   map[string]string `json:"display_name" jsonschema:"{lang: name} for every language this state has a name in; the author's rename wins over the built-in vanilla name"`
	Owner         string            `json:"owner"`
	Controller    string            `json:"controller"`
	ProvinceCount int               `json:"province_count"`
}

type getStatesOut struct {
	ProjectID  int64               `json:"project_id"`
	CountryTag string              `json:"country_tag"`
	NameQuery  string              `json:"name_query"`
	Total      int                 `json:"total"`
	Returned   int                 `json:"returned"`
	States     []stateListingEntry `json:"states"`
	Note       string              `json:"note,omitempty"`
}

func registerGetStates(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "get_states",
		Description: "List the project's states so you can resolve state_db_id + see owners. " +
			"edit_state keys on state_db_id (the surrogate PK) — read it here first. Pass country_tag " +
			"to return only the states that country OWNS. name_query narrows by NAME: case-insensitive " +
			"contains match on the display name in ANY stored language or on the raw loc key, plus exact state_id match " +
			"(\"613\"). Both filters combine (tag first, then name). Output is CAPPED at 200 rows AFTER " +
			"filtering: `total` is the true count and a `note` flags truncation. If `total` is 0 this " +
			"project has no states yet — run seed_states_from_vanilla first (standalone project) or " +
			"import a mod.",
	}, func(ctx context.Context, _ *CallToolRequest, in getStatesIn) (*CallToolResult, getStatesOut, error) {
		tag := strings.TrimSpace(in.CountryTag)
		var path string
		if tag != "" {
			path = fmt.Sprintf("/api/projects/%d/states/by-owner/%s", in.ProjectID, url.PathEscape(tag))
		} else {
			path = fmt.Sprintf("/api/projects/%d/states", in.ProjectID)
		}
		rows, err := proxyAuthedCallList(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, getStatesOut{}, err
		}

		q := strings.ToLower(strings.TrimSpace(in.NameQuery))
		filtered := make([]stateListingEntry, 0, len(rows))
		for _, r := range rows {
			disp := orEmptyMap(stringMapField(r, "display_name"))
			name := stringField(r, "name")
			stateID := int64Field(r, "state_id")

			if q != "" {

				match := strings.Contains(strings.ToLower(name), q) ||
					q == strconv.FormatInt(stateID, 10)
				for _, text := range disp {
					if match {
						break
					}
					match = strings.Contains(strings.ToLower(text), q)
				}
				if !match {
					continue
				}
			}

			provinces, _ := r["provinces"].([]any)
			filtered = append(filtered, stateListingEntry{
				StateDBID:     int64Field(r, "id"),
				StateID:       stateID,
				Name:          name,
				DisplayName:   disp,
				Owner:         stringField(r, "owner"),
				Controller:    stringField(r, "controller"),
				ProvinceCount: len(provinces),
			})
		}

		total := len(filtered)
		shown := filtered
		if len(shown) > getStatesRowCap {
			shown = shown[:getStatesRowCap]
		}

		out := getStatesOut{
			ProjectID:  in.ProjectID,
			CountryTag: tag,
			NameQuery:  strings.TrimSpace(in.NameQuery),
			Total:      total,
			Returned:   len(shown),
			States:     shown,
		}

		switch {
		case total == 0 && q != "":
			note := fmt.Sprintf("No states match name_query '%s'", q)
			if tag != "" {
				note += fmt.Sprintf(" among states owned by %s", strings.ToUpper(tag))
			}
			note += " (try a shorter substring, the other language, or drop the filter)."
			out.Note = note
		case total == 0 && tag != "":
			out.Note = fmt.Sprintf("No states owned by %s (check the tag, or seed states first).", strings.ToUpper(tag))
		case total == 0:
			out.Note = "No states in this project. On a fresh standalone project run " +
				"seed_states_from_vanilla (all_states=True) once before editing; imported-mod projects " +
				"already have their states."
		case total > getStatesRowCap:
			out.Note = fmt.Sprintf("Showing the first %d of %d states. Pass country_tag or name_query to narrow the list.",
				getStatesRowCap, total)
		}
		return nil, out, nil
	})
}

type validateSpecialProjectsIn struct {
	ProjectID  int64  `json:"project_id"`
	CountryTag string `json:"country_tag,omitempty" jsonschema:"optionally limit validation to one country's graph"`
}

type specialProjectValidationSummary struct {
	Status          string `json:"status"`
	Total           int64  `json:"total"`
	ActionableCount int    `json:"actionable_count"`
	InfoCount       int    `json:"info_count"`
}

type validateSpecialProjectsOut struct {
	ProjectID  int64                           `json:"project_id"`
	CountryTag *string                         `json:"country_tag"`
	Actionable []map[string]any                `json:"actionable"`
	Info       []map[string]any                `json:"info"`
	Total      int64                           `json:"total"`
	Summary    specialProjectValidationSummary `json:"summary"`
}

func registerValidateSpecialProjects(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "validate_special_projects",
		Description: "Run the real Special Projects DAG validator used by the REST API. `actionable` " +
			"findings (missing parents or dependency cycles) must be fixed before export. `info` " +
			"findings describe non-blocking graph shape such as isolated nodes. `country_tag` optionally " +
			"limits the validation to one country's graph.",
	}, func(ctx context.Context, _ *CallToolRequest, in validateSpecialProjectsIn) (*CallToolResult, validateSpecialProjectsOut, error) {
		path := fmt.Sprintf("/api/projects/%d/special-projects/validate", in.ProjectID)
		if in.CountryTag != "" {
			path += "?country_tag=" + url.QueryEscape(in.CountryTag)
		}
		resp, err := proxyAuthedCall(ctx, proxy, "GET", path, nil)
		if err != nil {
			return nil, validateSpecialProjectsOut{}, err
		}

		actionable := mapSliceField(resp, "actionable")
		info := mapSliceField(resp, "info")
		total := int64Field(resp, "total")

		var countryTagOut *string
		if in.CountryTag != "" {
			upper := strings.ToUpper(in.CountryTag)
			countryTagOut = &upper
		}

		status := "ok"
		if len(actionable) > 0 {
			status = "action_required"
		}

		out := validateSpecialProjectsOut{
			ProjectID:  in.ProjectID,
			CountryTag: countryTagOut,
			Actionable: actionable,
			Info:       info,
			Total:      total,
			Summary: specialProjectValidationSummary{
				Status:          status,
				Total:           total,
				ActionableCount: len(actionable),
				InfoCount:       len(info),
			},
		}
		return nil, out, nil
	})
}
