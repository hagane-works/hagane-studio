package toolapi

type entityKind struct {
	Kind string

	Path string

	KeyField string

	NoKeyReason string

	Summary []string

	CountryFilter bool

	CountryPath string

	RequireCountry string

	ReadFromList bool

	ParentPath string

	ChildCount *childCount

	ParentLookup *parentLookup
}

var entityKinds = []entityKind{
	{Kind: "ai_strategy_plan", Path: "ai-strategy-plans", KeyField: "plan_id",
		Summary: []string{"desc"}, CountryFilter: true, ReadFromList: true},
	{Kind: "ai_template", Path: "ai-templates", KeyField: "template_id",
		Summary: []string{"role"}, CountryFilter: true, ReadFromList: true},
	{Kind: "air_wing", Path: "oob-units/air-wings",
		NoKeyReason: "air wings have no game-script identifier — reference by id",
		Summary:     []string{"equipment_type", "amount", "owner", "group_id"}},
	{Kind: "air_wing_group", Path: "oob-units/air-wing-groups",
		NoKeyReason: "air wing groups have no game-script identifier — reference by id",
		Summary:     []string{"oob_file", "location", "country_tag", "wing_count"},
		ChildCount: &childCount{Field: "wing_count", ChildKind: "air_wing",
			ChildLink: "group_id", ParentLink: "id"}},
	{Kind: "autonomy_state", Path: "autonomy-states", KeyField: "autonomy_id"},
	{Kind: "bookmark", Path: "bookmarks", KeyField: "bookmark_id",
		Summary: []string{"is_default"}, CountryFilter: true},
	{Kind: "character", Path: "characters", KeyField: "tag",
		Summary: []string{"role_type", "has_advisor_role"}, CountryFilter: true},
	{Kind: "country", Path: "countries", KeyField: "tag", ReadFromList: true,
		Summary: []string{"has_flag"}},
	{Kind: "custom_trait", Path: "custom-traits", KeyField: "trait_id",
		Summary: []string{"trait_type", "advanced_extras_count"}, CountryFilter: true},
	{Kind: "decision", Path: "decisions", KeyField: "decision_id",

		Summary: []string{"category_key", "cost"}, CountryFilter: true,
		ParentLookup: &parentLookup{Field: "category_key", ParentKind: "decision_category",
			ChildLink: "category_id", ParentValue: "category_id"}},
	{Kind: "decision_category", Path: "decisions/categories", KeyField: "category_id",
		Summary: []string{"decision_count"}, CountryFilter: true,
		ChildCount: &childCount{Field: "decision_count", ChildKind: "decision",
			ChildLink: "category_id", ParentLink: "id"}},
	{Kind: "division_deployment", Path: "division-deployments",
		NoKeyReason: "deployments have no game-script identifier — reference by id",
		Summary: []string{"oob_file", "division_template", "location", "count",
			"has_actionable_warning"}},
	{Kind: "division_template", Path: "division-templates", KeyField: "template_name",
		Summary: []string{"oob_file"}},
	{Kind: "doctrine", Path: "doctrines", KeyField: "entity_key",
		Summary: []string{"entity_type", "source"}, CountryFilter: true},
	{Kind: "dynamic_modifier", Path: "dynamic-modifiers", KeyField: "dm_id",
		CountryFilter: true},
	{Kind: "equipment", Path: "equipment", KeyField: "equipment_id",
		Summary: []string{"is_archetype"}},
	{Kind: "event", Path: "events", KeyField: "event_id",
		Summary: []string{"namespace", "hidden", "major"}, CountryFilter: true},
	{Kind: "event_namespace", Path: "events/namespaces", KeyField: "namespace", ReadFromList: true,
		Summary: []string{"event_count"}, CountryFilter: true,
		ChildCount: &childCount{Field: "event_count", ChildKind: "event",
			ChildLink: "namespace_id", ParentLink: "id"}},
	{Kind: "fleet", Path: "oob-units/fleets",
		NoKeyReason: "fleets are identified by name, and names may repeat — reference by id",
		Summary:     []string{"naval_base", "oob_file", "country_tag", "task_force_count"},
		ChildCount: &childCount{Field: "task_force_count", ChildKind: "task_force",
			ChildLink: "fleet_id", ParentLink: "id"}},

	{Kind: "focus", Path: "nodes", ParentPath: "focus-trees", KeyField: "focus_id",
		Summary: []string{"focus_tree_id", "cost", "prerequisites"}},
	{Kind: "focus_tree", Path: "focus-trees", KeyField: "tree_id",
		Summary: []string{"tree_kind", "node_count", "shared_ref_count"}, CountryFilter: true},
	{Kind: "history_file", Path: "history-files", KeyField: "country_tag",
		Summary: []string{"configured_fields"}},
	{Kind: "idea", Path: "ideas", KeyField: "idea_id",
		Summary: []string{"idea_type"}, CountryFilter: true},
	{Kind: "intelligence_agency", Path: "intelligence-agencies", KeyField: "agency_id",
		CountryFilter: true},
	{Kind: "mio", Path: "mios", KeyField: "mio_id",
		Summary: []string{"include"}, CountryFilter: true},
	{Kind: "modifier_definition", Path: "modifier-definitions", KeyField: "modifier_id",
		Summary: []string{"value_type", "category_count"}},
	{Kind: "opinion_modifier", Path: "opinion-modifiers", KeyField: "modifier_id",
		Summary: []string{"source"}},
	{Kind: "raw_file", Path: "raw-files", KeyField: "dest_path",

		Summary: []string{"raw_kind", "is_binary", "size", "generate_replace_path", "references"}},
	{Kind: "scripted_localisation", Path: "scripted-localisations", KeyField: "name",
		Summary: []string{"source"}},
	{Kind: "scripted_snippet", Path: "scripted-snippets", KeyField: "name",
		Summary: []string{"snippet_kind", "source"}},
	{Kind: "ship", Path: "oob-units/ships",
		NoKeyReason: "ships are identified by name, and names may repeat — reference by id",
		Summary: []string{"definition", "equipment_type", "version_name", "task_force_id",
			"task_force_name", "pride_of_the_fleet"},

		ParentLookup: &parentLookup{Field: "task_force_name", ParentKind: "task_force",
			ChildLink: "task_force_id", ParentValue: "name"}},
	{Kind: "special_project", Path: "special-projects/projects", KeyField: "project_token", ReadFromList: true,

		Summary: []string{"specialization", "complexity", "special_project_parent"}},
	{Kind: "specialization", Path: "special-projects/specializations", KeyField: "specialization_id", ReadFromList: true,
		Summary: []string{"sub_specialization_count", "project_count"},
		ChildCount: &childCount{Field: "project_count", ChildKind: "special_project",
			ChildLink: "specialization", ParentLink: "specialization_id"}},
	{Kind: "state", Path: "states", KeyField: "state_id", CountryPath: "states/by-owner/{country_tag}",
		Summary: []string{"owner_tag", "loc_key"}, CountryFilter: true,

		RequireCountry: "a project can hold well over a thousand states; pass country_tag " +
			"to list one country's, or use get_entity when you already know the id"},
	{Kind: "sub_unit", Path: "sub-units", KeyField: "sub_unit_id",
		Summary: []string{"group"}},

	{Kind: "super_event", Path: "super-events", KeyField: "",

		NoKeyReason: "composed from country_tag and number — supplied as key",
		Summary:     []string{"number", "title"}, CountryFilter: true},
	{Kind: "task_force", Path: "oob-units/task-forces",
		NoKeyReason: "task forces are identified by name, and names may repeat — reference by id",
		Summary:     []string{"fleet_id", "fleet_name", "location", "ship_count"},
		ChildCount: &childCount{Field: "ship_count", ChildKind: "ship",
			ChildLink: "task_force_id", ParentLink: "id"},
		ParentLookup: &parentLookup{Field: "fleet_name", ParentKind: "fleet",
			ChildLink: "fleet_id", ParentValue: "name"}},
	{Kind: "technology", Path: "technologies", KeyField: "tech_id",
		Summary: []string{"tech_line", "cost", "start_year"}, CountryFilter: true},
	{Kind: "wargoal", Path: "wargoals", KeyField: "wargoal_key",
		Summary: []string{"war_name"}, CountryFilter: true},
}

var entityKindByName = func() map[string]entityKind {
	index := make(map[string]entityKind, len(entityKinds))
	for _, kind := range entityKinds {
		index[kind.Kind] = kind
	}
	return index
}()

func entityKindNames() []string {
	names := make([]string, 0, len(entityKinds))
	for _, kind := range entityKinds {
		names = append(names, kind.Kind)
	}
	return names
}
