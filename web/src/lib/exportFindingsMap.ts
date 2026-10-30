import { GATE_PARITY_FINDINGS } from "@/lib/gateParityFindingMetadata";

export type FindingTier = "attn" | "auto" | "pass" | "health";
export type FindingSurface =
  "export" | "playtest" | "import" | "ai" | "workbench";

export interface FindingMeta {
  tier: FindingTier;
  surface: FindingSurface;
  i18nKey: string;
}

export const EXPORT_FINDINGS: Record<string, FindingMeta> = {
  "focus.display_name.missing_created": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_display_name_missing_created",
  },
  "focus.display_name.missing_enabled_locale": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_display_name_missing_enabled_locale",
  },
  "focus.initial_position.unknown_kind": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_initial_position_unknown_kind",
  },
  "focus.reward.modifier_in_effect_position": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_reward_modifier_in_effect_position",
  },
  "focus.reward.scalar_effect_block_value": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_reward_scalar_effect_block_value",
  },
  "focus.reward.trigger_in_effect_position": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_reward_trigger_in_effect_position",
  },
  "focus.reward.unsupported_scope": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_reward_unsupported_scope",
  },
  focus_base_default_provider_unresolved: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_base_default_provider_unresolved",
  },
  focus_effective_default_missing: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_effective_default_missing",
  },
  focus_file_policy_applied: {
    tier: "pass",
    surface: "export",
    i18nKey: "focus_file_policy_applied",
  },
  focus_file_policy_cohort_incomplete: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_file_policy_cohort_incomplete",
  },
  focus_file_policy_drift: {
    tier: "auto",
    surface: "export",
    i18nKey: "focus_file_policy_drift",
  },
  focus_file_policy_duplicate_membership: {
    tier: "health",
    surface: "export",
    i18nKey: "focus_file_policy_duplicate_membership",
  },
  focus_file_policy_missing: {
    tier: "auto",
    surface: "export",
    i18nKey: "focus_file_policy_missing",
  },
  focus_initial_position_target_missing: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_initial_position_target_missing",
  },
  focus_raw_file_parse_failed: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_raw_file_parse_failed",
  },
  focus_registry_unavailable: {
    tier: "health",
    surface: "export",
    i18nKey: "focus_registry_unavailable",
  },
  "timeline.bookmark_path_shadow_date_mismatch": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_bookmark_path_shadow_date_mismatch",
  },
  "timeline.custom_start_without_bookmark": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_custom_start_without_bookmark",
  },
  "timeline.project_earliest_bookmark_mismatch": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_project_earliest_bookmark_mismatch",
  },
  "timeline.project_start_date_unknown": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_project_start_date_unknown",
  },
  "timeline.raw_bookmark_date_unknown": {
    tier: "auto",
    surface: "export",
    i18nKey: "timeline_raw_bookmark_date_unknown",
  },
  "timeline.raw_bookmark_hash_mismatch": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_raw_bookmark_hash_mismatch",
  },
  "timeline.raw_defines_start_date_conflict": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_raw_defines_start_date_conflict",
  },
  "timeline.start_date_ok": {
    tier: "pass",
    surface: "export",
    i18nKey: "timeline_start_date_ok",
  },
  "timeline.structured_bookmark_date_unknown": {
    tier: "auto",
    surface: "export",
    i18nKey: "timeline_structured_bookmark_date_unknown",
  },
  "clausewitz.parse_error": {
    tier: "attn",
    surface: "export",
    i18nKey: "clausewitz_parse_error",
  },
  "clausewitz.unclosed_block": {
    tier: "attn",
    surface: "export",
    i18nKey: "clausewitz_unclosed_block",
  },
  "clausewitz.stray_closing_brace": {
    tier: "attn",
    surface: "export",
    i18nKey: "clausewitz_stray_closing_brace",
  },
  "clausewitz.unterminated_quote": {
    tier: "attn",
    surface: "export",
    i18nKey: "clausewitz_unterminated_quote",
  },

  "raw_content.literal_escape_newline": {
    tier: "attn",
    surface: "export",
    i18nKey: "raw_content_literal_escape_newline",
  },
  "clausewitz.missing_assignment_value": {
    tier: "attn",
    surface: "export",
    i18nKey: "clausewitz_missing_assignment_value",
  },
  "clausewitz.analysis_ast_mismatch": {
    tier: "health",
    surface: "export",
    i18nKey: "clausewitz_analysis_ast_mismatch",
  },
  "clausewitz.analysis_token_alignment": {
    tier: "health",
    surface: "export",
    i18nKey: "clausewitz_analysis_token_alignment",
  },
  "country.capital_missing": {
    tier: "attn",
    surface: "export",
    i18nKey: "country_capital_missing",
  },
  "country.capital_invalid_state": {
    tier: "attn",
    surface: "export",
    i18nKey: "country_capital_invalid_state",
  },
  "country.capital_not_owned": {
    tier: "attn",
    surface: "export",
    i18nKey: "country_capital_not_owned",
  },

  "map.state_no_land": {
    tier: "attn",
    surface: "export",
    i18nKey: "map_state_no_land",
  },
  "country.history_final_winner_ambiguous": {
    tier: "attn",
    surface: "export",
    i18nKey: "country_history_final_winner_ambiguous",
  },
  "country.history_final_winner_inferred": {
    tier: "auto",
    surface: "export",
    i18nKey: "country_history_final_winner_inferred",
  },
  "country.research_slots_missing": {
    tier: "attn",
    surface: "export",
    i18nKey: "country_research_slots_missing",
  },
  "country.research_slots_invalid": {
    tier: "attn",
    surface: "export",
    i18nKey: "country_research_slots_invalid",
  },
  "playtest.crash": {
    tier: "attn",
    surface: "playtest",
    i18nKey: "playtest_crash",
  },
  "playtest.start_date_mismatch": {
    tier: "attn",
    surface: "playtest",
    i18nKey: "playtest_start_date_mismatch",
  },
  "playtest.runtime_diagnostics_input_invalid": {
    tier: "attn",
    surface: "playtest",
    i18nKey: "playtest_runtime_diagnostics_input_invalid",
  },
  "playtest.loaded_start_date": {
    tier: "pass",
    surface: "playtest",
    i18nKey: "playtest_loaded_start_date",
  },
  "playtest.enabled_mods": {
    tier: "pass",
    surface: "playtest",
    i18nKey: "playtest_enabled_mods",
  },
  "playtest.start_date_ok": {
    tier: "pass",
    surface: "playtest",
    i18nKey: "playtest_start_date_ok",
  },
  oob_air_location_missing: {
    tier: "attn",
    surface: "export",
    i18nKey: "oob_air_location_missing",
  },
  oob_air_location_invalid: {
    tier: "attn",
    surface: "export",
    i18nKey: "oob_air_location_invalid",
  },
  oob_air_equipment_type_missing: {
    tier: "attn",
    surface: "export",
    i18nKey: "oob_air_equipment_type_missing",
  },
  oob_air_equipment_type_invalid: {
    tier: "attn",
    surface: "export",
    i18nKey: "oob_air_equipment_type_invalid",
  },
  building_state_coordinate_mismatch: {
    tier: "attn",
    surface: "export",
    i18nKey: "building_state_coordinate_mismatch",
  },
  building_coordinate_membership_missing: {
    tier: "attn",
    surface: "export",
    i18nKey: "building_coordinate_membership_missing",
  },
  building_reference_province_invalid: {
    tier: "attn",
    surface: "export",
    i18nKey: "building_reference_province_invalid",
  },
  duplicate_expected_province: {
    tier: "health",
    surface: "export",
    i18nKey: "duplicate_expected_province",
  },
  duplicate_province_within_state: {
    tier: "attn",
    surface: "export",
    i18nKey: "duplicate_province_within_state",
  },
  duplicate_state_input: {
    tier: "attn",
    surface: "export",
    i18nKey: "duplicate_state_input",
  },
  province_in_multiple_states: {
    tier: "attn",
    surface: "export",
    i18nKey: "province_in_multiple_states",
  },
  province_without_state: {
    tier: "attn",
    surface: "export",
    i18nKey: "province_without_state",
  },
  unknown_province: {
    tier: "attn",
    surface: "export",
    i18nKey: "unknown_province",
  },
  "bookmark.raw_no_bookmark": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_raw_no_bookmark",
  },
  "bookmark.raw_path_case_collision": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_raw_path_case_collision",
  },
  "descriptor.imported_replace_paths_require_full_export": {
    tier: "attn",
    surface: "export",
    i18nKey: "descriptor_imported_replace_paths_require_full_export",
  },
  "descriptor.replace_path_empty": {
    tier: "attn",
    surface: "export",
    i18nKey: "descriptor_replace_path_empty",
  },
  ideology_root_invalid: {
    tier: "attn",
    surface: "export",
    i18nKey: "ideology_root_invalid",
  },
  ideology_root_submod_blocked: {
    tier: "attn",
    surface: "export",
    i18nKey: "ideology_root_submod_blocked",
  },
  raid_path_unsafe: {
    tier: "attn",
    surface: "import",
    i18nKey: "raid_path_unsafe",
  },
  raid_deps_not_carried: {
    tier: "attn",
    surface: "import",
    i18nKey: "raid_deps_not_carried",
  },
  unit_path_unsafe: {
    tier: "attn",
    surface: "import",
    i18nKey: "unit_path_unsafe",
  },
  factions_path_unsafe: {
    tier: "attn",
    surface: "import",
    i18nKey: "factions_path_unsafe",
  },
  loc_entry_unpreserved: {
    tier: "auto",
    surface: "import",
    i18nKey: "loc_entry_unpreserved",
  },
  "state.raw_parse_failed": {
    tier: "attn",
    surface: "export",
    i18nKey: "state_raw_parse_failed",
  },
  sub_unit_classification_invalid: {
    tier: "attn",
    surface: "export",
    i18nKey: "sub_unit_classification_invalid",
  },
  sub_unit_source_file_collision: {
    tier: "attn",
    surface: "export",
    i18nKey: "sub_unit_source_file_collision",
  },
  base_id_collision_rerouted: {
    tier: "auto",
    surface: "ai",
    i18nKey: "base_id_collision_rerouted",
  },
  non_additive_base_history_override: {
    tier: "attn",
    surface: "ai",
    i18nKey: "non_additive_base_history_override",
  },
  gateway_tool_passthrough_failed_sdk_emit: {
    tier: "attn",
    surface: "ai",
    i18nKey: "gateway_tool_passthrough_failed_sdk_emit",
  },
  "content_language.export_languages_missing": {
    tier: "attn",
    surface: "export",
    i18nKey: "content_language_export_languages_missing",
  },
  display_name_placeholder: {
    tier: "attn",
    surface: "export",
    i18nKey: "display_name_placeholder",
  },
  effect_typo: { tier: "attn", surface: "export", i18nKey: "effect_typo" },
  preserved_raw: { tier: "pass", surface: "export", i18nKey: "preserved_raw" },
  research_bonus_as_effect: {
    tier: "attn",
    surface: "export",
    i18nKey: "research_bonus_as_effect",
  },
  "script.position.swap_ideas_child_outside_parent": {
    tier: "attn",
    surface: "export",
    i18nKey: "script_position_swap_ideas_child_outside_parent",
  },
  unknown_tech_category: {
    tier: "attn",
    surface: "export",
    i18nKey: "unknown_tech_category",
  },

  vanilla_seed_dlc_effects_absent: {
    tier: "attn",
    surface: "export",
    i18nKey: "vanilla_seed_dlc_effects_absent",
  },

  impassable_category_mismatch: {
    tier: "auto",
    surface: "export",
    i18nKey: "impassable_category_mismatch",
  },

  impassable_without_force_link: {
    tier: "auto",
    surface: "export",
    i18nKey: "impassable_without_force_link",
  },

  ideology_undefined_standalone: {
    tier: "attn",
    surface: "export",
    i18nKey: "ideology_undefined_standalone",
  },
  ideology_registry_parent_missing: {
    tier: "attn",
    surface: "export",
    i18nKey: "ideology_registry_parent_missing",
  },
  focus_layout_same_cell: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_same_cell",
  },
  focus_layout_adjacent_same_row: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_adjacent_same_row",
  },
  "focus.display_name.unproven_fallback": {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_display_name_unproven_fallback",
  },
  idea_no_modifier: {
    tier: "attn",
    surface: "export",
    i18nKey: "idea_no_modifier",
  },
  character_no_military_traits: {
    tier: "attn",
    surface: "export",
    i18nKey: "character_no_military_traits",
  },
  "bookmark.selectable_ideology_missing": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_selectable_ideology_missing",
  },
  focus_tree_no_nodes: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_tree_no_nodes",
  },
  localisation_reference_not_in_project: {
    tier: "attn",
    surface: "export",
    i18nKey: "localisation_reference_not_in_project",
  },
  mod_folder_name: {
    tier: "auto",
    surface: "export",
    i18nKey: "mod_folder_name",
  },
  province_double_mapping: {
    tier: "attn",
    surface: "export",
    i18nKey: "province_double_mapping",
  },
  doctrine_track_unresolved: {
    tier: "attn",
    surface: "export",
    i18nKey: "doctrine_track_unresolved",
  },
  event_namespace_mismatch: {
    tier: "attn",
    surface: "export",
    i18nKey: "event_namespace_mismatch",
  },
  event_no_options: {
    tier: "attn",
    surface: "export",
    i18nKey: "event_no_options",
  },
  tech_circular_prereq: {
    tier: "attn",
    surface: "export",
    i18nKey: "tech_circular_prereq",
  },
  tech_folder_unknown: {
    tier: "attn",
    surface: "export",
    i18nKey: "tech_folder_unknown",
  },
  focus_effective_default_multiple: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_effective_default_multiple",
  },
  focus_created_primary_id_collision: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_created_primary_id_collision",
  },
  focus_shared_pool_scope_ignored: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_shared_pool_scope_ignored",
  },
  ...GATE_PARITY_FINDINGS,
};
