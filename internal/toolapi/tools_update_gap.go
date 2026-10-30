package toolapi

import (
	"context"
	"fmt"
	"strings"

	"haganeworks/internal/workbench"
)

type updateEntityOut = map[string]any

func applyEntityUpdate(ctx context.Context, proxy *RESTProxy, kindName string,
	projectID, id int64, body map[string]any) (updateEntityOut, error) {

	kind, err := lookupEntityKind(kindName)
	if err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, fmt.Errorf(
			"update_%s needs the row id (the `id` field from list_entities), not the game identifier", kindName)
	}
	if len(body) == 0 {

		return nil, fmt.Errorf(
			"update_%s was called with no fields to change — supply at least one", kindName)
	}
	path := fmt.Sprintf("/api/projects/%d/%s/%d", projectID, kind.Path, id)
	resp, err := proxyAuthedCall(ctx, proxy, "PUT", path, body)
	if err != nil {
		return nil, err
	}
	key := entityKey(kind, resp)
	out := updateEntityOut{
		"id": id, "kind": kind.Kind, "key": key,
	}
	if kind.KeyField != "" {
		out[kind.KeyField] = key
	}
	if diagnostics := saveDiagnosticsField(resp); diagnostics != nil {
		out["diagnostics"] = *diagnostics
	}
	if diagnostics := namedSaveDiagnosticsField(resp, "tree_diagnostics"); diagnostics != nil {
		out["tree_diagnostics"] = *diagnostics
	}
	return out, nil
}

func set[T any](body map[string]any, key string, value *T) {
	if value != nil {
		body[key] = *value
	}
}

type locFields struct {
	DisplayNameLocs map[string]string `json:"display_name_locs,omitempty" jsonschema:"canonical localised names, e.g. {\"en\":\"Mandate of Heaven\",\"zh\":\"天命\"} — prefer this over name_en"`
	DescriptionLocs map[string]string `json:"description_locs,omitempty"`
}

func (l locFields) apply(body map[string]any) {
	if len(l.DisplayNameLocs) > 0 {
		body["display_name_locs"] = l.DisplayNameLocs
	}
	if len(l.DescriptionLocs) > 0 {
		body["description_locs"] = l.DescriptionLocs
	}
}

type updateSuperEventIn struct {
	ProjectID         int64 `json:"project_id"`
	ID                int64 `json:"id" jsonschema:"row id from list_entities kind=super_event"`
	locFields         `json:",inline"`
	Number            *int64           `json:"number,omitempty" jsonschema:"4-digit, unique per country_tag"`
	TitleEn           *string          `json:"title_en,omitempty"`
	DescEn            *string          `json:"desc_en,omitempty"`
	QuoteEn           *string          `json:"quote_en,omitempty"`
	BtnEn             *string          `json:"btn_en,omitempty"`
	SongToken         *string          `json:"song_token,omitempty"`
	AutoFireOnStartup *bool            `json:"auto_fire_on_startup,omitempty"`
	AutoFireAfterDays *int64           `json:"auto_fire_after_days,omitempty"`
	Options           []map[string]any `json:"options,omitempty" jsonschema:"REPLACES the existing option list when supplied"`
}

func registerUpdateSuperEvent(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_super_event",
		Description: "Change an existing SUPEREVENT (the full-screen cinematic set piece). Not an " +
			"ordinary event (update_event) and not an id prefix. Only the fields you supply change. " +
			"Note: a superevent nothing references is dropped from the export — keep it auto-firing, " +
			"or make sure some event, decision, focus or file calls its id.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateSuperEventIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "number", in.Number)
		set(body, "title_en", in.TitleEn)
		set(body, "desc_en", in.DescEn)
		set(body, "quote_en", in.QuoteEn)
		set(body, "btn_en", in.BtnEn)
		set(body, "song_token", in.SongToken)
		set(body, "auto_fire_on_startup", in.AutoFireOnStartup)
		set(body, "auto_fire_after_days", in.AutoFireAfterDays)
		if in.Options != nil {
			body["options"] = in.Options
		}
		out, err := applyEntityUpdate(ctx, proxy, "super_event", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateBookmarkIn struct {
	ProjectID           int64 `json:"project_id"`
	ID                  int64 `json:"id"`
	locFields           `json:",inline"`
	BookmarkID          *string          `json:"bookmark_id,omitempty"`
	NameEn              *string          `json:"name_en,omitempty"`
	DescEn              *string          `json:"desc_en,omitempty"`
	Date                *string          `json:"date,omitempty"`
	Picture             *string          `json:"picture,omitempty"`
	DefaultCountry      *string          `json:"default_country,omitempty"`
	SelectableCountries []map[string]any `json:"selectable_countries,omitempty" jsonschema:"REPLACES the existing list when supplied"`
	Filters             []string         `json:"filters,omitempty"`
}

func registerUpdateBookmark(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_bookmark",
		Description: "Change an existing bookmark (a start-date scenario on the game's launch " +
			"screen). Only the fields you supply change.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateBookmarkIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "bookmark_id", in.BookmarkID)
		set(body, "name_en", in.NameEn)
		set(body, "desc_en", in.DescEn)
		set(body, "date", in.Date)
		set(body, "picture", in.Picture)
		set(body, "default_country", in.DefaultCountry)
		if in.SelectableCountries != nil {
			body["selectable_countries"] = in.SelectableCountries
		}
		if in.Filters != nil {
			body["filters"] = in.Filters
		}
		out, err := applyEntityUpdate(ctx, proxy, "bookmark", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateCustomTraitIn struct {
	ProjectID        int64 `json:"project_id"`
	ID               int64 `json:"id"`
	locFields        `json:",inline"`
	TraitID          *string          `json:"trait_id,omitempty"`
	TraitType        *string          `json:"trait_type,omitempty" jsonschema:"country_leader | unit_leader | scientist"`
	NameEn           *string          `json:"name_en,omitempty"`
	DescEn           *string          `json:"desc_en,omitempty"`
	Modifier         map[string]any   `json:"modifier,omitempty" jsonschema:"REPLACES the existing modifier dict when supplied"`
	AiWillDo         *string          `json:"ai_will_do,omitempty"`
	TargetedModifier []map[string]any `json:"targeted_modifier,omitempty"`
}

func registerUpdateCustomTrait(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_custom_trait",
		Description: "Change an existing custom trait (a leader / commander / scientist trait). " +
			"Only the fields you supply change; `modifier` replaces the whole dict when given.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateCustomTraitIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "trait_id", in.TraitID)
		set(body, "trait_type", in.TraitType)
		set(body, "name_en", in.NameEn)
		set(body, "desc_en", in.DescEn)
		set(body, "ai_will_do", in.AiWillDo)
		if in.Modifier != nil {
			body["modifier"] = in.Modifier
		}
		if in.TargetedModifier != nil {
			body["targeted_modifier"] = in.TargetedModifier
		}
		out, err := applyEntityUpdate(ctx, proxy, "custom_trait", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateDynamicModifierIn struct {
	ProjectID     int64 `json:"project_id"`
	ID            int64 `json:"id"`
	locFields     `json:",inline"`
	DmID          *string        `json:"dm_id,omitempty"`
	NameEn        *string        `json:"name_en,omitempty"`
	Icon          *string        `json:"icon,omitempty"`
	RemoveTrigger *string        `json:"remove_trigger,omitempty" jsonschema:"raw trigger block"`
	Enable        *string        `json:"enable,omitempty" jsonschema:"raw trigger block"`
	Modifier      map[string]any `json:"modifier,omitempty" jsonschema:"REPLACES the existing modifier dict when supplied"`
}

func registerUpdateDynamicModifier(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_dynamic_modifier",
		Description: "Change an existing dynamic modifier. Only the fields you supply change.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateDynamicModifierIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "dm_id", in.DmID)
		set(body, "name_en", in.NameEn)
		set(body, "icon", in.Icon)
		set(body, "remove_trigger", in.RemoveTrigger)
		set(body, "enable", in.Enable)
		if in.Modifier != nil {
			body["modifier"] = in.Modifier
		}
		out, err := applyEntityUpdate(ctx, proxy, "dynamic_modifier", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateScriptedSnippetIn struct {
	ProjectID  int64   `json:"project_id"`
	ID         int64   `json:"id"`
	Name       *string `json:"name,omitempty" jsonschema:"the dispatch symbol other content calls"`
	Kind       *string `json:"kind,omitempty" jsonschema:"effect | trigger"`
	Body       *string `json:"body,omitempty" jsonschema:"RAW HOI4 script — the verbatim content inside the block braces"`
	SourceFile *string `json:"source_file,omitempty"`
}

func registerUpdateScriptedSnippet(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_scripted_snippet",
		Description: "Change an existing scripted effect or trigger. Renaming it does NOT update " +
			"the places that call it — find them with validate_project afterwards, or the calls " +
			"silently stop working.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateScriptedSnippetIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "name", in.Name)
		set(body, "kind", in.Kind)
		set(body, "body", in.Body)
		set(body, "source_file", in.SourceFile)
		out, err := applyEntityUpdate(ctx, proxy, "scripted_snippet", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateScriptedLocalisationIn struct {
	ProjectID  int64   `json:"project_id"`
	ID         int64   `json:"id"`
	Name       *string `json:"name,omitempty"`
	Body       *string `json:"body,omitempty" jsonschema:"RAW HOI4 script — the verbatim inner content of defined_text = { ... }"`
	SourceFile *string `json:"source_file,omitempty"`
}

func registerUpdateScriptedLocalisation(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_scripted_localisation",
		Description: "Change an existing scripted localisation (a defined_text block).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateScriptedLocalisationIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "name", in.Name)
		set(body, "body", in.Body)
		set(body, "source_file", in.SourceFile)
		out, err := applyEntityUpdate(ctx, proxy, "scripted_localisation", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateAutonomyStateIn struct {
	ProjectID         int64 `json:"project_id"`
	ID                int64 `json:"id"`
	locFields         `json:",inline"`
	AutonomyID        *string        `json:"autonomy_id,omitempty"`
	NameEn            *string        `json:"name_en,omitempty"`
	IsPuppet          *bool          `json:"is_puppet,omitempty"`
	IsDefault         *bool          `json:"is_default,omitempty"`
	ManpowerInfluence *float64       `json:"manpower_influence,omitempty"`
	MinFreedomLevel   *int           `json:"min_freedom_level,omitempty" jsonschema:"stored as freedom x 1000 (freedom 0.25 -> 250)"`
	Rule              *string        `json:"rule,omitempty" jsonschema:"RAW autonomy-rule block"`
	Modifier          map[string]any `json:"modifier,omitempty"`
	AiWeights         map[string]any `json:"ai_weights,omitempty"`
}

func registerUpdateAutonomyState(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_autonomy_state",
		Description: "Change an existing autonomy state (a subject level such as puppet or dominion).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateAutonomyStateIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "autonomy_id", in.AutonomyID)
		set(body, "name_en", in.NameEn)
		set(body, "is_puppet", in.IsPuppet)
		set(body, "is_default", in.IsDefault)
		set(body, "manpower_influence", in.ManpowerInfluence)
		set(body, "min_freedom_level", in.MinFreedomLevel)
		set(body, "rule", in.Rule)
		if in.Modifier != nil {
			body["modifier"] = in.Modifier
		}
		if in.AiWeights != nil {
			body["ai_weights"] = in.AiWeights
		}
		out, err := applyEntityUpdate(ctx, proxy, "autonomy_state", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateDivisionTemplateIn struct {
	ProjectID          int64            `json:"project_id"`
	ID                 int64            `json:"id"`
	TemplateName       *string          `json:"template_name,omitempty"`
	OobFile            *string          `json:"oob_file,omitempty"`
	DivisionNamesGroup *string          `json:"division_names_group,omitempty"`
	Regiments          []map[string]any `json:"regiments,omitempty" jsonschema:"REPLACES the line battalions when supplied: list of {token, x, y}"`
	Support            []map[string]any `json:"support,omitempty" jsonschema:"REPLACES the support companies when supplied"`
	IsLocked           *bool            `json:"is_locked,omitempty"`
	Priority           *int             `json:"priority,omitempty"`
}

func registerUpdateDivisionTemplate(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_division_template",
		Description: "Change an existing division template. Renaming it does NOT re-point the " +
			"deployments that use it — they reference the NAME, not the row, so update those too " +
			"or they will spawn nothing.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateDivisionTemplateIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "template_name", in.TemplateName)
		set(body, "oob_file", in.OobFile)
		set(body, "division_names_group", in.DivisionNamesGroup)
		set(body, "is_locked", in.IsLocked)
		set(body, "priority", in.Priority)
		if in.Regiments != nil {
			body["regiments"] = in.Regiments
		}
		if in.Support != nil {
			body["support"] = in.Support
		}
		out, err := applyEntityUpdate(ctx, proxy, "division_template", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateDivisionDeploymentIn struct {
	ProjectID             int64    `json:"project_id"`
	ID                    int64    `json:"id"`
	DivisionTemplate      *string  `json:"division_template,omitempty" jsonschema:"the template's NAME, not its row id"`
	OobFile               *string  `json:"oob_file,omitempty"`
	Location              *int64   `json:"location,omitempty" jsonschema:"province id where the division spawns"`
	Count                 *int     `json:"count,omitempty"`
	DivisionName          *string  `json:"division_name,omitempty"`
	StartExperienceFactor *float64 `json:"start_experience_factor,omitempty"`
	StartEquipmentFactor  *float64 `json:"start_equipment_factor,omitempty"`
}

func registerUpdateDivisionDeployment(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name:        "update_division_deployment",
		Description: "Change an existing division deployment (which template spawns where, and how many).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateDivisionDeploymentIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		set(body, "division_template", in.DivisionTemplate)
		set(body, "oob_file", in.OobFile)
		set(body, "location", in.Location)
		set(body, "count", in.Count)
		set(body, "division_name", in.DivisionName)
		set(body, "start_experience_factor", in.StartExperienceFactor)
		set(body, "start_equipment_factor", in.StartEquipmentFactor)
		out, err := applyEntityUpdate(ctx, proxy, "division_deployment", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateEquipmentIn struct {
	ProjectID     int64 `json:"project_id"`
	ID            int64 `json:"id"`
	locFields     `json:",inline"`
	Name          *string  `json:"name,omitempty"`
	EquipmentID   *string  `json:"equipment_id,omitempty"`
	EquipmentType *string  `json:"equipment_type,omitempty"`
	IsArchetype   *bool    `json:"is_archetype,omitempty"`
	Parent        *string  `json:"parent,omitempty" jsonschema:"equipment-id string reference, not a row id"`
	Year          *int     `json:"year,omitempty"`
	Picture       *string  `json:"picture,omitempty"`
	IsBuildable   *bool    `json:"is_buildable,omitempty"`
	CanLicense    *bool    `json:"can_license,omitempty"`
	SoftAttack    *float64 `json:"soft_attack,omitempty" jsonschema:"omit to inherit from the archetype — sending 0 means an explicit zero, not inherit"`
	HardAttack    *float64 `json:"hard_attack,omitempty"`
	APAttack      *float64 `json:"ap_attack,omitempty"`
	AirAttack     *float64 `json:"air_attack,omitempty"`
	Breakthrough  *float64 `json:"breakthrough,omitempty"`
	Defense       *float64 `json:"defense,omitempty"`
	ArmorValue    *float64 `json:"armor_value,omitempty"`
}

func registerUpdateEquipment(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_equipment",
		Description: "Change an existing equipment archetype or variant. Numeric stats you omit " +
			"keep inheriting from the archetype; sending 0 sets an explicit zero instead.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateEquipmentIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "name", in.Name)
		set(body, "equipment_id", in.EquipmentID)
		set(body, "equipment_type", in.EquipmentType)
		set(body, "is_archetype", in.IsArchetype)
		set(body, "parent", in.Parent)
		set(body, "year", in.Year)
		set(body, "picture", in.Picture)
		set(body, "is_buildable", in.IsBuildable)
		set(body, "can_license", in.CanLicense)
		set(body, "soft_attack", in.SoftAttack)
		set(body, "hard_attack", in.HardAttack)
		set(body, "ap_attack", in.APAttack)
		set(body, "air_attack", in.AirAttack)
		set(body, "breakthrough", in.Breakthrough)
		set(body, "defense", in.Defense)
		set(body, "armor_value", in.ArmorValue)
		out, err := applyEntityUpdate(ctx, proxy, "equipment", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

type updateCountryIn struct {
	ProjectID int64 `json:"project_id"`
	ID        int64 `json:"id"`
	locFields `json:",inline"`
	Name      *string `json:"name,omitempty"`
	Tag       *string `json:"tag,omitempty" jsonschema:"2-5 chars, first A-Z"`
	Ideology  *string `json:"ideology,omitempty"`
}

func registerUpdateCountry(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_country",
		Description: "Change an existing country. Changing its tag does NOT re-point the content " +
			"that references the old tag — expect to fix those too; validate_project finds them. " +
			"A base-game or imported country takes its ruling party from its start history, not from " +
			"this row: changing its ideology here also writes that party to its start history " +
			"(the same as set_country_history set_politics_ruling_party).",
	}, func(ctx context.Context, _ *CallToolRequest, in updateCountryIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "name", in.Name)
		set(body, "tag", in.Tag)
		set(body, "ideology", in.Ideology)
		party, err := rulingPartyChange(ctx, proxy, in)
		if err != nil {
			return nil, nil, err
		}
		out, err := applyEntityUpdate(ctx, proxy, "country", in.ProjectID, in.ID, body)
		if err != nil || party == nil {
			return nil, out, err
		}
		if !party.changed {
			out["ruling_party"] = map[string]any{"written": false, "note": "ideology is what the row " +
				"already held, so the start history was left alone; to set this country's ruling " +
				"party use set_country_history set_politics_ruling_party"}
			return nil, out, nil
		}
		historyID, created, err := setHistoryRulingParty(ctx, proxy, in.ProjectID, party.tag, party.ideology)
		if err != nil {
			return nil, out, fmt.Errorf("the country row was updated, but its ruling party could not be "+
				"written to its start history (retry with set_country_history): %w", err)
		}
		out["ruling_party"] = map[string]any{
			"written": true, "history_file_id": historyID, "created": created,
			"note": "written to the country's start history: a base-game or imported country " +
				"takes its ruling party from there, not from the country row",
		}
		return nil, out, nil
	})
}

type countryPartyChange struct {
	tag, ideology string
	changed       bool
}

func rulingPartyChange(ctx context.Context, proxy *RESTProxy, in updateCountryIn) (*countryPartyChange, error) {
	if in.Ideology == nil || strings.TrimSpace(*in.Ideology) == "" || in.ID <= 0 {
		return nil, nil
	}
	rows, err := proxyAuthedCallList(ctx, proxy, "GET", fmt.Sprintf("/api/projects/%d/countries", in.ProjectID), nil)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if id, _ := row["id"].(float64); int64(id) != in.ID {
			continue
		}
		tag := strings.ToUpper(strings.TrimSpace(stringField(row, "tag")))
		if in.Tag != nil && strings.TrimSpace(*in.Tag) != "" {
			tag = strings.ToUpper(strings.TrimSpace(*in.Tag))
		}
		imported, _ := row["auto_added"].(bool)
		if !workbench.VanillaCountryTag(tag) && !imported {
			return nil, nil
		}
		ideology := strings.TrimSpace(*in.Ideology)
		return &countryPartyChange{tag: tag, ideology: ideology,
			changed: ideology != strings.TrimSpace(stringField(row, "ideology"))}, nil
	}

	return nil, nil
}

type updateFocusTreeIn struct {
	ProjectID   int64   `json:"project_id"`
	ID          int64   `json:"id,omitempty" jsonschema:"row id from list_entities; focus_tree_id from create_focus_tree is accepted as an alias"`
	FocusTreeID *int64  `json:"focus_tree_id,omitempty" jsonschema:"numeric id from create_focus_tree; alias for id"`
	TreeID      *string `json:"tree_id,omitempty"`
	CountryTag  *string `json:"country_tag,omitempty"`
	TreeKind    *string `json:"tree_kind,omitempty" jsonschema:"primary | shared_pool"`
}

func registerUpdateFocusTree(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_focus_tree",

		Description: "Change the focus TREE ITSELF — its id (use id from list_entities or focus_tree_id from create_focus_tree), which country it belongs to, whether " +
			"it is a shared pool. This does NOT touch any focus inside it; to change one focus use " +
			"update_focus, which takes a focus id rather than a tree id. Tree metadata has no localised title; edit node names with update_focus.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateFocusTreeIn) (*CallToolResult, updateEntityOut, error) {
		id := in.ID
		if in.FocusTreeID != nil {
			if *in.FocusTreeID <= 0 {
				return nil, updateEntityOut{}, fmt.Errorf("focus_tree_id must be positive")
			}
			if id > 0 && id != *in.FocusTreeID {
				return nil, updateEntityOut{}, fmt.Errorf("id and focus_tree_id must identify the same focus tree")
			}
			if id <= 0 {
				id = *in.FocusTreeID
			}
		}
		if id <= 0 {
			return nil, updateEntityOut{}, fmt.Errorf("id or focus_tree_id must be positive")
		}
		body := map[string]any{}
		set(body, "tree_id", in.TreeID)
		set(body, "country_tag", in.CountryTag)
		set(body, "tree_kind", in.TreeKind)
		out, err := applyEntityUpdate(ctx, proxy, "focus_tree", in.ProjectID, id, body)
		return nil, out, err
	})
}

type updateSubUnitIn struct {
	ProjectID       int64 `json:"project_id"`
	ID              int64 `json:"id"`
	locFields       `json:",inline"`
	SubUnitID       *string `json:"sub_unit_id,omitempty" jsonschema:"the token division templates reference"`
	NameEn          *string `json:"name_en,omitempty"`
	NameZh          *string `json:"name_zh,omitempty"`
	Abbreviation    *string `json:"abbreviation,omitempty"`
	Sprite          *string `json:"sprite,omitempty"`
	MapIconCategory *string `json:"map_icon_category,omitempty"`
	Group           *string `json:"group,omitempty" jsonschema:"the recruitment-tab group"`
	Transport       *string `json:"transport,omitempty"`
	Priority        *int    `json:"priority,omitempty"`
	AiPriority      *int    `json:"ai_priority,omitempty"`
	Active          *bool   `json:"active,omitempty"`
	Cavalry         *bool   `json:"cavalry,omitempty"`
	SpecialForces   *bool   `json:"special_forces,omitempty"`
	Marines         *bool   `json:"marines,omitempty"`
}

func registerUpdateSubUnit(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "update_sub_unit",
		Description: "Change an existing sub-unit (a battalion or company type). Renaming its token " +
			"does NOT re-point the division templates that use it.",
	}, func(ctx context.Context, _ *CallToolRequest, in updateSubUnitIn) (*CallToolResult, updateEntityOut, error) {
		body := map[string]any{}
		in.locFields.apply(body)
		set(body, "sub_unit_id", in.SubUnitID)
		set(body, "name_en", in.NameEn)
		set(body, "name_zh", in.NameZh)
		set(body, "abbreviation", in.Abbreviation)
		set(body, "sprite", in.Sprite)
		set(body, "map_icon_category", in.MapIconCategory)
		set(body, "group", in.Group)
		set(body, "transport", in.Transport)
		set(body, "priority", in.Priority)
		set(body, "ai_priority", in.AiPriority)
		set(body, "active", in.Active)
		set(body, "cavalry", in.Cavalry)
		set(body, "special_forces", in.SpecialForces)
		set(body, "marines", in.Marines)
		out, err := applyEntityUpdate(ctx, proxy, "sub_unit", in.ProjectID, in.ID, body)
		return nil, out, err
	})
}

func registerUpdateGapTools(s *Server, proxy *RESTProxy) {
	registerUpdateSuperEvent(s, proxy)
	registerUpdateBookmark(s, proxy)
	registerUpdateCustomTrait(s, proxy)
	registerUpdateDynamicModifier(s, proxy)
	registerUpdateScriptedSnippet(s, proxy)
	registerUpdateScriptedLocalisation(s, proxy)
	registerUpdateAutonomyState(s, proxy)
	registerUpdateDivisionTemplate(s, proxy)
	registerUpdateDivisionDeployment(s, proxy)
	registerUpdateEquipment(s, proxy)
	registerUpdateCountry(s, proxy)
	registerUpdateFocusTree(s, proxy)
	registerUpdateSubUnit(s, proxy)
}
