package toolapi

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type createCustomTraitIn struct {
	ProjectID        int64            `json:"project_id"`
	TraitID          string           `json:"trait_id" jsonschema:"snake_case key, required"`
	TraitType        string           `json:"trait_type,omitempty" jsonschema:"default country_leader; country_leader | unit_leader | scientist"`
	CountryTag       string           `json:"country_tag,omitempty" jsonschema:"blank inherits Project.country_tag"`
	NameEn           string           `json:"name_en,omitempty"`
	DescEn           string           `json:"desc_en,omitempty"`
	Modifier         map[string]any   `json:"modifier,omitempty" jsonschema:"raw HOI4 modifier dict"`
	AiWillDo         string           `json:"ai_will_do,omitempty"`
	TargetedModifier []map[string]any `json:"targeted_modifier,omitempty" jsonschema:"list of {tag, modifier} dicts"`
}

type createCustomTraitOut struct {
	CustomTraitID int64  `json:"custom_trait_id"`
	TraitID       string `json:"trait_id"`
	TraitType     string `json:"trait_type"`
}

func registerCreateCustomTrait(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_custom_trait",
		Description: "Create a custom character trait. trait_type = country_leader | unit_leader | " +
			"scientist. trait_id = snake_case key (required). modifier = raw HOI4 modifier dict. " +
			"Duplicate (type+id) errors (NO auto-suffix) — pick a unique trait_id.",
	}, func(ctx context.Context, _ *CallToolRequest, in createCustomTraitIn) (*CallToolResult, createCustomTraitOut, error) {
		traitType := in.TraitType
		if traitType == "" {
			traitType = "country_leader"

		}
		modifier := in.Modifier
		if modifier == nil {
			modifier = map[string]any{}

		}
		targetedModifier := in.TargetedModifier
		if targetedModifier == nil {
			targetedModifier = []map[string]any{}
		}
		body := map[string]any{
			"trait_id": in.TraitID, "trait_type": traitType, "country_tag": in.CountryTag,
			"name_en": in.NameEn, "desc_en": in.DescEn, "modifier": modifier,
			"ai_will_do": in.AiWillDo, "targeted_modifier": targetedModifier,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/custom-traits", in.ProjectID), body)
		if err != nil {
			return nil, createCustomTraitOut{}, err
		}

		out := createCustomTraitOut{
			CustomTraitID: int64Field(resp, "id"),
			TraitID:       stringField(resp, "trait_id"),
			TraitType:     stringField(resp, "trait_type"),
		}
		return nil, out, nil
	})
}

type createDynamicModifierIn struct {
	ProjectID     int64          `json:"project_id"`
	DmID          string         `json:"dm_id" jsonschema:"snake_case, required, auto-suffixes on collision"`
	CountryTag    string         `json:"country_tag,omitempty"`
	NameEn        string         `json:"name_en,omitempty"`
	Icon          string         `json:"icon,omitempty"`
	RemoveTrigger string         `json:"remove_trigger,omitempty" jsonschema:"raw trigger block"`
	Enable        string         `json:"enable,omitempty" jsonschema:"raw trigger block"`
	Modifier      map[string]any `json:"modifier,omitempty" jsonschema:"raw HOI4 modifier dict"`
}

type createDynamicModifierOut struct {
	DmDbID   int64    `json:"dm_db_id"`
	DmID     string   `json:"dm_id"`
	Warnings []string `json:"warnings,omitempty"`
}

func registerCreateDynamicModifier(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_dynamic_modifier",
		Description: "Create a dynamic modifier. dm_id = snake_case (required, auto-suffixes on " +
			"collision). modifier = raw HOI4 modifier dict. enable/remove_trigger = raw trigger blocks.",
	}, func(ctx context.Context, _ *CallToolRequest, in createDynamicModifierIn) (*CallToolResult, createDynamicModifierOut, error) {
		modifier := in.Modifier
		if modifier == nil {
			modifier = map[string]any{}

		}
		body := map[string]any{
			"dm_id": in.DmID, "country_tag": in.CountryTag, "name_en": in.NameEn,
			"icon": in.Icon, "remove_trigger": in.RemoveTrigger, "enable": in.Enable,
			"modifier": modifier,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/dynamic-modifiers", in.ProjectID), body)
		if err != nil {
			return nil, createDynamicModifierOut{}, err
		}

		out := createDynamicModifierOut{
			DmDbID:   int64Field(resp, "id"),
			DmID:     stringField(resp, "dm_id"),
			Warnings: stringSliceField(resp, "warnings"),
		}
		return nil, out, nil
	})
}

var rawTemplateRoutes = map[string]struct {
	RoutePath string
	Field     string
}{
	"scripted_gui":     {"", "panel_key"},
	"on_actions":       {"/on-actions", "panel_key"},
	"game_rules":       {"/game-rules", "panel_key"},
	"ai_strategy":      {"/ai-strategy", "panel_key"},
	"gfx_sprite":       {"/gfx", "panel_key"},
	"defines":          {"/defines", "key"},
	"bop":              {"/bop", "key"},
	"music":            {"/music", "key"},
	"peace_conference": {"/peace-conference", "key"},
}

var rawTemplateKeyRE = regexp.MustCompile(`^[a-z0-9_]{2,40}$`)

type createRawFromTemplateIn struct {
	ProjectID int64  `json:"project_id"`
	Template  string `json:"template" jsonschema:"a key from list_raw_templates"`
	Key       string `json:"key" jsonschema:"naming seed, 2-40 chars of lowercase a-z/0-9/_, TAG-prefixed recommended"`
}

type rawTemplateFileOut struct {
	RawFileID int64  `json:"raw_file_id"`
	DestPath  string `json:"dest_path"`
}

type createRawFromTemplateOut struct {
	Template  string               `json:"template"`
	Files     []rawTemplateFileOut `json:"files"`
	FileCount int                  `json:"file_count"`
	Note      string               `json:"note"`
}

func registerCreateRawFromTemplate(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_raw_from_template",
		Description: "Generate starting-point raw override file(s) from a built-in template (see " +
			"list_raw_templates for the key list) and store them in the project — they ship verbatim " +
			"in the export. key = your naming seed, 2-40 chars of lowercase a-z/0-9/_ (TAG-prefix it, " +
			"e.g. 'fng_tweaks', so generated filenames never collide with the base mod). Idempotent: " +
			"re-running with the same template+key OVERWRITES the previously generated file(s) (upsert " +
			"by dest_path), never duplicates. Only the whitelisted templates are reachable here — an " +
			"arbitrary-path raw write is create_raw_file (or the Export page's raw-files manager).",
	}, func(ctx context.Context, _ *CallToolRequest, in createRawFromTemplateIn) (*CallToolResult, createRawFromTemplateOut, error) {
		route, ok := rawTemplateRoutes[strings.TrimSpace(in.Template)]
		if !ok {
			keys := make([]string, 0, len(rawTemplateRoutes))
			for k := range rawTemplateRoutes {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return nil, createRawFromTemplateOut{}, fmt.Errorf(
				"unknown template %q. Valid templates: %s. Call list_raw_templates for labels + descriptions.",
				in.Template, strings.Join(keys, ", "))
		}
		if !rawTemplateKeyRE.MatchString(in.Key) {
			return nil, createRawFromTemplateOut{}, fmt.Errorf(
				"key must be 2-40 chars of lowercase a-z, 0-9 or _ (it becomes part of the " +
					"generated filename), e.g. 'my_tweaks'")
		}

		body := map[string]any{route.Field: in.Key}
		files, err := proxyAuthedCallList(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/raw-files/from-template%s", in.ProjectID, route.RoutePath), body)
		if err != nil {
			return nil, createRawFromTemplateOut{}, err
		}

		out := createRawFromTemplateOut{
			Template:  in.Template,
			Files:     make([]rawTemplateFileOut, 0, len(files)),
			FileCount: len(files),
			Note: "Starting-point file(s) generated — edit the content with update_raw_file (or the " +
				"Export page's raw-files manager) and verify in-game.",
		}
		for _, f := range files {
			out.Files = append(out.Files, rawTemplateFileOut{
				RawFileID: int64Field(f, "id"),
				DestPath:  stringField(f, "dest_path"),
			})
		}
		return nil, out, nil
	})
}

type createUiPanelIn struct {
	ProjectID         int64             `json:"project_id"`
	Token             string            `json:"token" jsonschema:"REQUIRED ASCII identifier naming seed, TAG-prefixed recommended, e.g. EA_legitimacy"`
	CountryTag        string            `json:"country_tag,omitempty"`
	ContextType       string            `json:"context_type,omitempty" jsonschema:"default decision_category; player_context | selected_country_context | selected_state_context | decision_category | diplomatic_action | national_focus_context | country_mapicon | state_mapicon"`
	ParentWindowToken string            `json:"parent_window_token,omitempty" jsonschema:"the game screen the panel is mounted on (HOI4 token): politics_tab, decision_tab, technology_tab, diplomacy_tab, trade_tab, construction_tab, production_tab, deployment_tab, logistics_tab, national_focus, top_bar …; empty = the main window"`
	VisibleTrigger    string            `json:"visible_trigger,omitempty"`
	WindowProps       map[string]any    `json:"window_props,omitempty" jsonschema:"window geometry and presentation: x, y, width, height, optional background_sprite/orientation/origo/moveable"`
	Controls          []map[string]any  `json:"controls,omitempty" jsonschema:"complete initial control tree; kinds: container/icon/text/progressbar/button. text is a string; multilingual text belongs in text_locs as locale-to-string object. A locale-to-string object supplied as text is preserved as text_locs; conflicting duplicate locale values are rejected. Use apply_ui_panel_operations after creation"`
	RawExtras         map[string]any    `json:"raw_extras,omitempty"`
	NameEn            string            `json:"name_en,omitempty"`
	NameZh            string            `json:"name_zh,omitempty"`
	DisplayNameLocs   map[string]string `json:"display_name_locs,omitempty"`
}

func registerCreateUiPanel(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_ui_panel",
		Description: "Create a ui-panel (visual scripted_gui builder — a custom in-game window " +
			"compiled to interface/*.gui + common/scripted_guis/*.txt + loc on export). token = " +
			"REQUIRED snake_case naming seed (TAG-prefixed recommended); all emitted names derive " +
			"from it. On a project-scoped token collision the program auto-suffixes — check the " +
			"returned token + warnings. Accepts full window_props, controls, visibility, parent, raw extras and localisation; old shell-only calls remain valid. " +
			"For text/button controls, text is a string and text_locs holds multilingual locale-to-string values. A locale object passed as text is preserved as text_locs; conflicting duplicate locales are rejected. " +
			"To put components on a game screen (\"in the politics screen\"), mount the panel on it: " +
			"parent_window_token = that screen's token (countrypoliticsview politics_tab, countrydecisionview decision_tab, countrytechnologyview technology_tab, " +
			"countrydiplomacyview diplomacy_tab, countrytradeview trade_tab, countryconstructionsview construction_tab, countryproductionlineview production_tab, " +
			"countrydeploymentview deployment_tab, countrylogisticsview logistics_tab, nationalfocusview national_focus) with context_type = player_context; " +
			"a state's view countrystateview mounts on selected_state_view with context_type = selected_state_context, and the top bar on every screen is top_bar (player_context); " +
			"window_props x/y then count from that screen window's top-left (verified in game: politics_tab at 40,600 sits under Research & Production) " +
			"and background_sprite \"\" keeps it see-through. The game view draws a mounted panel on its screen. Each screen has at most one such panel, " +
			"token <TAG>_<token>_ui (the game view's component bar adds to it): when list_ui_panels shows it, add controls to it with apply_ui_panel_operations " +
			"instead of creating another.",
	}, func(ctx context.Context, _ *CallToolRequest, in createUiPanelIn) (*CallToolResult, map[string]any, error) {
		if in.ProjectID <= 0 {
			return nil, nil, fmt.Errorf("project_id must be a positive integer")
		}
		contextType := in.ContextType
		if contextType == "" {
			contextType = "decision_category"

		}
		candidate := map[string]any{
			"token": in.Token, "country_tag": in.CountryTag,
			"context_type": contextType, "parent_window_token": in.ParentWindowToken,
			"visible_trigger": in.VisibleTrigger, "window_props": in.WindowProps,
			"controls": in.Controls, "raw_extras": in.RawExtras,
			"name_en": in.NameEn, "name_zh": in.NameZh,
			"display_name_locs": copyStringMap(in.DisplayNameLocs),
		}
		normalized, diagnostics := validateUIPanelData(candidate)
		if diagnosticsHaveErrors(diagnostics) {
			return nil, nil, diagnosticsError("ui-panel create rejected", diagnostics)
		}
		body := map[string]any{}
		for _, field := range []string{
			"token", "country_tag", "context_type", "parent_window_token",
			"visible_trigger", "window_props", "controls", "raw_extras",
			"name_en", "name_zh", "display_name_locs",
		} {
			body[field] = normalized[field]
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/ui-panels", in.ProjectID), body)
		if err != nil {
			return nil, nil, err
		}

		out := uiPanelReadback(resp, in.ProjectID)
		out["diagnostics"] = diagnostics
		return nil, out, nil
	})
}

type createSuperEventIn struct {
	ProjectID         int64            `json:"project_id"`
	Number            int64            `json:"number" jsonschema:"4-digit, unique per country_tag"`
	CountryTag        string           `json:"country_tag,omitempty"`
	TitleEn           string           `json:"title_en,omitempty"`
	DescEn            string           `json:"desc_en,omitempty"`
	QuoteEn           string           `json:"quote_en,omitempty"`
	BtnEn             string           `json:"btn_en,omitempty"`
	SongToken         string           `json:"song_token,omitempty"`
	AutoFireOnStartup bool             `json:"auto_fire_on_startup,omitempty"`
	AutoFireAfterDays int64            `json:"auto_fire_after_days,omitempty"`
	Options           []map[string]any `json:"options,omitempty" jsonschema:"list of {name_en, name_zh, effect, ai_chance}; empty = single Continue button"`
}

type createSuperEventOut struct {
	SuperEventID int64  `json:"super_event_id"`
	Number       int64  `json:"number"`
	CountryTag   string `json:"country_tag"`
}

func registerCreateSuperEvent(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_super_event",

		Description: "Create a SUPEREVENT: a full-screen cinematic set piece with its own artwork, " +
			"shown edge-to-edge. NOT an ordinary popup event (use create_event) and NOT an id prefix " +
			"(use create_event_namespace). number = 4-digit, unique per country_tag " +
			"(duplicate -> 409, NO auto-suffix). A reachable trigger is required: set " +
			"auto_fire_on_startup=true or auto_fire_after_days to a positive number — a dormant call " +
			"is rejected.",
	}, func(ctx context.Context, _ *CallToolRequest, in createSuperEventIn) (*CallToolResult, createSuperEventOut, error) {
		if !in.AutoFireOnStartup && in.AutoFireAfterDays <= 0 {
			return nil, createSuperEventOut{}, fmt.Errorf(
				"super-events created through this tool require auto_fire_on_startup=true " +
					"or auto_fire_after_days>0 so the event is reachable in-game")
		}
		options := in.Options
		if options == nil {
			options = []map[string]any{}

		}
		body := map[string]any{
			"number": in.Number, "country_tag": in.CountryTag, "title_en": in.TitleEn,
			"desc_en": in.DescEn, "quote_en": in.QuoteEn, "btn_en": in.BtnEn,
			"song_token": in.SongToken, "auto_fire_on_startup": in.AutoFireOnStartup,
			"auto_fire_after_days": in.AutoFireAfterDays, "options": options,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/super-events", in.ProjectID), body)
		if err != nil {
			return nil, createSuperEventOut{}, err
		}

		out := createSuperEventOut{
			SuperEventID: int64Field(resp, "id"),
			Number:       int64Field(resp, "number"),
			CountryTag:   stringField(resp, "country_tag"),
		}
		return nil, out, nil
	})
}

type createScriptedLocalisationIn struct {
	ProjectID  int64          `json:"project_id"`
	Name       string         `json:"name" jsonschema:"the defined_text dispatch symbol, e.g. GetMyDynamicText; unique per project, required"`
	Body       string         `json:"body" jsonschema:"RAW HOI4 script — the verbatim inner content of defined_text = { ... }, re-including the name = X line; required, non-whitespace"`
	SourceFile string         `json:"source_file,omitempty" jsonschema:"complete mod-relative path under common/scripted_localisation/*.txt; blank = the program's default"`
	RawExtras  map[string]any `json:"raw_extras,omitempty" jsonschema:"leave unset — round-trip preservation bucket, not a modifier"`
}

type createScriptedLocalisationOut struct {
	LocDbID int64  `json:"loc_db_id"`
	Name    string `json:"name"`
}

func registerCreateScriptedLocalisation(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_scripted_localisation",
		Description: "Create a scripted localisation — a HOI4 defined_text block: a dynamic loc key " +
			"that resolves to different text depending on triggers. name = the defined_text dispatch " +
			"symbol (internal HOI4 symbol, NOT player-facing text), unique per project — a duplicate " +
			"is a 409 with NO auto-suffix. body = RAW HOI4 script, the verbatim inner content of the " +
			"block (re-including the name = X line), required and non-whitespace.",
	}, func(ctx context.Context, _ *CallToolRequest, in createScriptedLocalisationIn) (*CallToolResult, createScriptedLocalisationOut, error) {
		if strings.TrimSpace(in.Body) == "" {
			return nil, createScriptedLocalisationOut{}, fmt.Errorf(
				"body is required and must contain non-whitespace HOI4 script")
		}
		body := map[string]any{
			"name": in.Name, "body": in.Body, "source_file": in.SourceFile,
			"raw_extras": in.RawExtras,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/scripted-localisations", in.ProjectID), body)
		if err != nil {
			return nil, createScriptedLocalisationOut{}, err
		}

		out := createScriptedLocalisationOut{
			LocDbID: int64Field(resp, "id"),
			Name:    stringField(resp, "name"),
		}
		return nil, out, nil
	})
}

type createScriptedSnippetIn struct {
	ProjectID  int64          `json:"project_id"`
	Name       string         `json:"name" jsonschema:"dispatch symbol other entities call; unique per (project, kind), required"`
	Kind       string         `json:"kind,omitempty" jsonschema:"default effect; effect | trigger — any other value is a 400"`
	Body       string         `json:"body,omitempty" jsonschema:"RAW HOI4 script — verbatim content inside the block braces"`
	SourceFile string         `json:"source_file,omitempty" jsonschema:"complete mod-relative path under common/scripted_effects/ or common/scripted_triggers/"`
	RawExtras  map[string]any `json:"raw_extras,omitempty" jsonschema:"leave unset — round-trip bucket, not a modifier"`
}

type createScriptedSnippetOut struct {
	SnippetDbID int64  `json:"snippet_db_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
}

func registerCreateScriptedSnippet(s *Server, proxy *RESTProxy) {
	AddTool(s, &Tool{
		Name: "create_scripted_snippet",
		Description: "Create a scripted snippet — a reusable named scripted_effect or " +
			"scripted_trigger block, called by name from other effect/trigger fields (as `<name> = " +
			"yes`, both kinds). kind = effect | trigger (default effect). name is unique per " +
			"(project, kind) — a duplicate is a 409 with NO auto-suffix. A scripted block runs in the " +
			"CALLER's scope: write body scope-agnostic.",
	}, func(ctx context.Context, _ *CallToolRequest, in createScriptedSnippetIn) (*CallToolResult, createScriptedSnippetOut, error) {
		kind := in.Kind
		if kind == "" {
			kind = "effect"

		}
		body := map[string]any{
			"kind": kind, "name": in.Name, "body": in.Body, "source_file": in.SourceFile,
			"raw_extras": in.RawExtras,
		}

		resp, err := proxyAuthedCall(ctx, proxy, "POST",
			fmt.Sprintf("/api/projects/%d/scripted-snippets", in.ProjectID), body)
		if err != nil {
			return nil, createScriptedSnippetOut{}, err
		}

		out := createScriptedSnippetOut{
			SnippetDbID: int64Field(resp, "id"),
			Name:        stringField(resp, "name"),
			Kind:        stringField(resp, "kind"),
		}
		return nil, out, nil
	})
}

func proxyAuthedCallList(ctx context.Context, proxy *RESTProxy, method, path string, body any) ([]map[string]any, error) {
	return proxyAuthedCallAny(ctx, proxy, method, path, body)
}
