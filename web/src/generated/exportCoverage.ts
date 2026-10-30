export type ExportCoverageStatus =
  | "consumed_then_discarded"
  | "imported_not_exported"
  | "metadata_only"
  | "modeled_roundtrip"
  | "not_routed"
  | "parsed_partial"
  | "preserved_verbatim"
  | "scanner_blind";

export interface ExportCoverageRow {
  readonly path: string;
  readonly status: ExportCoverageStatus;
  readonly overlayBehavior: string;
  readonly category: string;
  readonly scannerRouted: boolean;
  readonly backendRouted: boolean;
  readonly reason: string;
  readonly evidence: Readonly<Record<string, string | number | boolean>>;
}

export const EXPORT_COVERAGE_ROWS = [
  {
    path: "common/national_focus/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "focus_trees",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Focus entities and their owned localisation were measured as complete; source filenames may be rewritten.",
    evidence: { measured_in: 99, measured_out: 7, measured_nodes: "1087/1087" },
  },
  {
    path: "common/technologies/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "technologies",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Technologies are modeled and regrouped by technology family on export.",
    evidence: { measured_in: 16, measured_out: 11 },
  },
  {
    path: "history/states/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "states",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "State fields survive, but most imported source filenames are not retained.",
    evidence: { measured_in: 2525, measured_out: 2525, renamed_files: 2334 },
  },
  {
    path: "common/characters/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "characters",
    scannerRouted: true,
    backendRouted: true,
    reason: "Character entities are modeled and emitted.",
    evidence: { measured_in: 1, measured_out: 1 },
  },
  {
    path: "common/decisions/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "decisions",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Decisions and categories are modeled; export consolidates source files.",
    evidence: { measured_in: 115, measured_out: 2, measured_entities: 580 },
  },
  {
    path: "common/ideas/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "ideas",
    scannerRouted: true,
    backendRouted: true,
    reason: "Ideas are modeled; export consolidates source files.",
    evidence: { measured_in: 11, measured_out: 1, measured_entities: 203 },
  },
  {
    path: "events/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "events",
    scannerRouted: true,
    backendRouted: true,
    reason: "Events and namespaces are modeled and emitted.",
    evidence: { measured_in: 9, measured_out: 6, measured_events: 56 },
  },
  {
    path: "history/countries/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "country_history",
    scannerRouted: true,
    backendRouted: true,
    reason: "Country history is modeled and emitted.",
    evidence: { measured_in: 6, measured_out: 6 },
  },
  {
    path: "common/bookmarks/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "bookmarks",
    scannerRouted: true,
    backendRouted: true,
    reason: "Bookmarks are modeled and emitted.",
    evidence: { measured_in: 1, measured_out: 1 },
  },
  {
    path: "common/country_leader/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "leader_traits",
    scannerRouted: true,
    backendRouted: true,
    reason: "Country leader traits are modeled and emitted.",
    evidence: { measured_in: 13, measured_out: 1 },
  },
  {
    path: "common/raids/**",
    status: "preserved_verbatim",
    overlayBehavior: "overwritten",
    category: "raids",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Raid files use whole-file custody at their original relative paths.",
    evidence: { measured_in: 3, measured_out: 3 },
  },
  {
    path: "common/scripted_effects/**",
    status: "parsed_partial",
    overlayBehavior: "coexists",
    category: "scripted_effects",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Scripted effects are parsed and re-emitted; measured top-level coverage is below byte or block identity.",
    evidence: {
      measured_in: 198,
      measured_parsed: 12741,
      measured_total_blocks: 12754,
      measured_out: 196,
    },
  },
  {
    path: "common/scripted_triggers/**",
    status: "parsed_partial",
    overlayBehavior: "coexists",
    category: "scripted_triggers",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Scripted triggers are parsed and re-emitted with a measured block gap.",
    evidence: {
      measured_in: 49,
      measured_parsed: 986,
      measured_total_blocks: 991,
      measured_out: 49,
    },
  },
  {
    path: "common/scripted_localisation/**",
    status: "parsed_partial",
    overlayBehavior: "coexists",
    category: "scripted_localisation",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Scripted localisation is routed, but the measured parser gap is 7.6 percent.",
    evidence: {
      measured_in: 122,
      measured_parsed: 6343,
      measured_total_blocks: 6868,
      measured_out: 119,
      measured_gap_pct: 7.6,
    },
  },
  {
    path: "common/dynamic_modifiers/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "dynamic_modifiers",
    scannerRouted: true,
    backendRouted: true,
    reason: "Dynamic modifiers are modeled and emitted.",
    evidence: { measured_in: 64, measured_parsed: 517, measured_out: 1 },
  },
  {
    path: "common/autonomous_states/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "autonomy_states",
    scannerRouted: true,
    backendRouted: true,
    reason: "Autonomy states are modeled and emitted.",
    evidence: { measured_in: 23, measured_parsed: 23, measured_out: 1 },
  },
  {
    path: "common/opinion_modifiers/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "opinion_modifiers",
    scannerRouted: true,
    backendRouted: true,
    reason: "Opinion modifiers are modeled and emitted.",
    evidence: { measured_in: 4, measured_parsed: 672, measured_out: 1 },
  },
  {
    path: "common/modifier_definitions/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "modifier_definitions",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Modifier definitions are modeled; incompatible files are preserved verbatim as raw files.",
    evidence: {},
  },
  {
    path: "common/doctrines/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "doctrines",
    scannerRouted: true,
    backendRouted: true,
    reason: "Recognized doctrine subdirectories are modeled and emitted.",
    evidence: { measured_in: 26, measured_parsed: 61, measured_out: 18 },
  },
  {
    path: "common/intelligence_agencies/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "intelligence_agencies",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Intelligence agencies are modeled and emitted; upgrade siblings are outside this row.",
    evidence: { measured_in: 1, measured_parsed: 29, measured_out: 1 },
  },
  {
    path: "common/unit_leader/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "unit_leader_traits",
    scannerRouted: true,
    backendRouted: true,
    reason: "Unit leader traits are modeled and consolidated on export.",
    evidence: { measured_in: 12, measured_out: 1 },
  },
  {
    path: "common/scientist_traits/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "scientist_traits",
    scannerRouted: true,
    backendRouted: true,
    reason: "Scientist traits are modeled and emitted.",
    evidence: { measured_in: 1, measured_parsed: 28, measured_out: 1 },
  },
  {
    path: "common/countries/**",
    status: "metadata_only",
    overlayBehavior: "survives_in_base",
    category: "country_metadata",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Country files feed display metadata but are not round-tripped as files.",
    evidence: { measured_in: 54, measured_out: 0 },
  },
  {
    path: "common/scripted_guis/**",
    status: "imported_not_exported",
    overlayBehavior: "survives_in_base",
    category: "scripted_guis",
    scannerRouted: false,
    backendRouted: true,
    reason:
      "Scripted GUI sources may be copied for GUI analysis but are not generally replayed by export.",
    evidence: { measured_in: 133, measured_parsed: 0, measured_out: 0 },
  },
  {
    path: "localisation/**",
    status: "modeled_roundtrip",
    overlayBehavior: "coexists",
    category: "localisation",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Imported localisation is parsed into key-level custody rows; modeled keys take structured precedence while otherwise-unmodeled creator keys are replayed from their original files.",
    evidence: {
      custody_model: "project_loc_entries",
      precedence: "replace_then_source_order",
    },
  },
  {
    path: "common/on_actions/**",
    status: "preserved_verbatim",
    overlayBehavior: "sibling_preserved",
    category: "on_actions",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Imported hook files now use whole-file custody; Hagane Works-authored hooks retain the sibling collision policy.",
    evidence: { measured_in: 45 },
  },
  {
    path: "history/units/**",
    status: "preserved_verbatim",
    overlayBehavior: "survives_in_base",
    category: "orders_of_battle",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Imported OOB files now use whole-file custody and retain their paths.",
    evidence: { measured_in: 851 },
  },
  {
    path: "common/units/**",
    status: "preserved_verbatim",
    overlayBehavior: "survives_in_base",
    category: "units",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Imported unit and equipment trees now use whole-file custody; structured emitters remain authoritative on collisions.",
    evidence: { measured_in: 317 },
  },
  {
    path: "map/**",
    status: "not_routed",
    overlayBehavior: "survives_in_base",
    category: "map",
    scannerRouted: false,
    backendRouted: false,
    reason: "Custom geography is outside current import and export routing.",
    evidence: { measured_in: 307, measured_out: 0 },
  },
  {
    path: "common/factions/**",
    status: "preserved_verbatim",
    overlayBehavior: "overwritten",
    category: "factions",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Faction definition files use whole-directory raw custody and are replayed at their original relative paths; a creator-authored raw file wins a same-path import.",
    evidence: { measured_in: 7, measured_out: 7 },
  },
  {
    path: "common/ai_strategy/**",
    status: "not_routed",
    overlayBehavior: "survives_in_base",
    category: "ai_strategy",
    scannerRouted: false,
    backendRouted: false,
    reason: "AI strategy scripts are outside the modeled product surface.",
    evidence: { measured_in: 22 },
  },
  {
    path: "common/ai_strategy_plans/**",
    status: "modeled_roundtrip",
    overlayBehavior: "overwritten",
    category: "ai_strategy_plans",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "Strict whole-file parsing provides CRUD and source-aware replay; unsupported files move atomically to raw-file custody.",
    evidence: { raw_fallback: "whole_file" },
  },
  {
    path: "common/ai_templates/**",
    status: "modeled_roundtrip",
    overlayBehavior: "overwritten",
    category: "ai_templates",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "AI template roles and targets provide CRUD and source-aware replay; unsupported files move atomically to raw-file custody.",
    evidence: { raw_fallback: "whole_file" },
  },
  {
    path: "common/wargoals/**",
    status: "modeled_roundtrip",
    overlayBehavior: "overwritten",
    category: "wargoals",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "War-goal wrappers provide CRUD and ordered source-aware replay; malformed files move atomically to raw-file custody.",
    evidence: { raw_fallback: "whole_file" },
  },
  {
    path: "common/special_projects/**",
    status: "parsed_partial",
    overlayBehavior: "survives_in_base",
    category: "special_projects",
    scannerRouted: false,
    backendRouted: true,
    reason:
      "Only specialization and projects children are backend-routed; tag and reward siblings remain outside the model.",
    evidence: { measured_in: 14 },
  },
  {
    path: "common/technology_tags/**",
    status: "imported_not_exported",
    overlayBehavior: "survives_in_base",
    category: "technology_profiles",
    scannerRouted: true,
    backendRouted: true,
    reason:
      "The technology folder registry is captured in the project GUI-source snapshot for audited profile selection; the base registry remains authoritative on export.",
    evidence: { audited_source: "00_technology.txt" },
  },
  {
    path: "common/{abilities,aces,ai_areas,ai_equipment,ai_faction_theaters,ai_focuses,ai_navy,achievements,bop,buildings,collections,continuous_focus,country_tag_aliases,defines,difficulty_settings,frontend,game_rules,idea_tags,ideologies,intelligence_agency_upgrades,map_modes,medals,modifiers,names,occupation_laws,operation_phases,operations,peace_conference,profile_pictures,resistance_activity,resistance_compliance_modifiers,resources,ribbons,scorers,script_constants,scripted_diplomatic_actions,state_category,synchronized_dynamic_tokens,technology_sharing,terrain,timed_activities,unit_medals,unit_tags}/**",
    status: "not_routed",
    overlayBehavior: "survives_in_base",
    category: "engine_and_balance_tail",
    scannerRouted: false,
    backendRouted: false,
    reason:
      "Engine, balance, AI, and registry directories in this measured tail have no custody route.",
    evidence: { measured_in: 169, measured_directories: 44 },
  },
  {
    path: "history/general/**",
    status: "not_routed",
    overlayBehavior: "survives_in_base",
    category: "global_history",
    scannerRouted: false,
    backendRouted: false,
    reason: "Global history scripts have no route.",
    evidence: { measured_in: 1 },
  },
  {
    path: "country_metadata/**",
    status: "not_routed",
    overlayBehavior: "survives_in_base",
    category: "custom_metadata",
    scannerRouted: false,
    backendRouted: false,
    reason: "This corpus-specific nonstandard directory has no route.",
    evidence: { measured_in: 1 },
  },
  {
    path: "portraits/**",
    status: "not_routed",
    overlayBehavior: "survives_in_base",
    category: "portrait_declarations",
    scannerRouted: false,
    backendRouted: false,
    reason:
      "This corpus-specific portrait declaration directory has no text route.",
    evidence: { measured_in: 7 },
  },
] as const satisfies readonly ExportCoverageRow[];

export const LOSSY_EXPORT_STATUSES = new Set<ExportCoverageStatus>([
  "parsed_partial",
  "metadata_only",
  "imported_not_exported",
  "consumed_then_discarded",
  "scanner_blind",
  "not_routed",
]);

export const UNSUPPORTED_OR_LOSSY_EXPORT_COVERAGE = EXPORT_COVERAGE_ROWS.filter(
  (row) => LOSSY_EXPORT_STATUSES.has(row.status),
);

export const UNSUPPORTED_OR_LOSSY_EXPORT_PATHS =
  UNSUPPORTED_OR_LOSSY_EXPORT_COVERAGE.map((row) => row.path);
