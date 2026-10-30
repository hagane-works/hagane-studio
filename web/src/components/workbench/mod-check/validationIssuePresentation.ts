import type { CtdIssue } from "@/lib/api/export";

export type PlayerIssue = Omit<
  Pick<
    CtdIssue,
    | "code"
    | "entity_ref"
    | "detail"
    | "source"
    | "evidence"
    | "fix_hint"
    | "message"
    | "severity"
  >,
  "severity"
> & { severity?: CtdIssue["severity"] };

export type PlayerRepairMode = "automatic" | "guided" | "manual";

export interface PlayerIssuePresentation {
  family: string;
  groupKey: string;
  message: string;
  impact: string;
  hint: string;
  route: string | null;
  repairMode: PlayerRepairMode;
}

export type PlayerTranslator = (
  key: string,
  values?: Record<string, string | number>,
) => string;

export const TYPE_TO_ROUTE: Record<string, string> = {
  state: "states",
  history_file: "history-files",
  country: "countries",
  character: "characters",
  focus: "focus-tree",
  focus_tree: "focus-tree",
  idea: "ideas",
  event: "events",
  decision: "decisions",
  technology: "technology",
  doctrine: "doctrine",
  division_template: "division-templates",
  bookmark: "bookmarks",
  bookmark_country: "bookmarks",
  project: "settings",
  ideology: "ideologies",
};

export const MESSAGE_KEY_BY_CODE: Record<string, string> = {
  mod_folder_name: "msg_mod_folder_name",
  character_missing_name: "msg_character_missing_name",
  character_no_military_traits: "msg_character_no_military_traits",
  character_advisor_traits_fallback: "msg_character_advisor_traits_fallback",
  focus_tree_no_nodes: "msg_focus_tree_no_nodes",
  focus_missing_id: "msg_focus_missing_id",
  focus_prereq_not_found: "msg_focus_prereq_not_found",
  focus_mutex_not_found: "msg_focus_mutex_not_found",
  focus_prereq_mutex_conflict: "msg_focus_prereq_mutex_conflict",
  tech_missing_id: "msg_tech_missing_id",
  tech_prereq_not_found: "msg_tech_prereq_not_found",
  idea_missing_id: "msg_idea_missing_id",
  idea_no_modifier: "msg_idea_no_modifier",
  event_missing_id: "msg_event_missing_id",
  event_no_options: "msg_event_no_options",
  event_namespace_mismatch: "msg_event_namespace_mismatch",
  event_number_mismatch: "msg_event_number_mismatch",
  decision_missing_id: "msg_decision_missing_id",
  tech_circular_prereq: "msg_tech_circular_prereq",
  capital_not_owned: "msg_capital_not_owned",
  province_double_mapping: "msg_province_double_mapping",
  ideology_undefined_standalone: "msg_ideology_undefined_standalone",
  ideology_registry_parent_missing: "msg_ideology_registry_parent_missing",
  base_mod_id_collision: "msg_base_mod_id_collision",
  submod_ideology_undefined: "msg_submod_ideology_undefined",
  history_writes_base_tag: "msg_history_writes_base_tag",
  division_stem_base_collision: "msg_division_stem_base_collision",
  recruit_pure_advisor_base: "msg_recruit_pure_advisor_base",
  tech_folder_unknown: "msg_tech_folder_unknown",
  doctrine_track_unresolved: "msg_doctrine_track_unresolved",
  localisation_reference_not_in_project:
    "msg_localisation_reference_not_in_project",
  "bookmark.filter_registry_missing": "msg_bookmark_filter_registry_missing",
  "bookmark.default_missing": "msg_bookmark_default_missing",
  "bookmark.default_multiple": "msg_bookmark_default_multiple",
  "bookmark.default_country_not_interesting":
    "msg_bookmark_default_country_not_interesting",
  "bookmark.country_label_legacy_invalid_preserved":
    "msg_bookmark_country_label_legacy_invalid_preserved",
  "bookmark.country_label_invalid": "msg_bookmark_country_label_invalid",
  "bookmark.country_labels_filter_missing":
    "msg_bookmark_country_labels_filter_missing",
  "bookmark.filter_legacy_unknown_preserved":
    "msg_bookmark_filter_legacy_unknown_preserved",
  "bookmark.filter_invalid_for_target":
    "msg_bookmark_filter_invalid_for_target",
  "bookmark.filter_country_label_semantics":
    "msg_bookmark_filter_country_label_semantics",
  "bookmark.selectable_ideology_missing":
    "msg_bookmark_selectable_ideology_missing",
  "focus.initial_position.legacy_ambiguous":
    "msg_focus_initial_position_legacy_ambiguous",
  "focus.initial_position.invalid_shape":
    "msg_focus_initial_position_invalid_shape",

  "focus.display_name.unproven_fallback":
    "msg_focus_display_name_unproven_fallback",
  focus_effective_default_multiple: "msg_focus_effective_default_multiple",
  focus_relative_position_not_found: "msg_focus_relative_position_not_found",
  focus_prereq_cycle: "msg_focus_prereq_cycle",
  focus_id_duplicate: "msg_focus_id_duplicate",
  politics_custom_root_no_random_subtype:
    "msg_politics_custom_root_no_random_subtype",
};

export const HINT_KEY_BY_CODE: Record<string, string> = {
  capital_not_owned: "hint_capital_not_owned",
  province_double_mapping: "hint_province_double_mapping",
  ideology_undefined_standalone: "hint_ideology_undefined_standalone",
  ideology_registry_parent_missing: "hint_ideology_registry_parent_missing",
  base_mod_id_collision: "hint_base_mod_id_collision",
  submod_ideology_undefined: "hint_submod_ideology_undefined",
  history_writes_base_tag: "hint_history_writes_base_tag",
  division_stem_base_collision: "hint_division_stem_base_collision",
  recruit_pure_advisor_base: "hint_recruit_pure_advisor_base",
  tech_folder_unknown: "hint_tech_folder_unknown",
  doctrine_track_unresolved: "hint_doctrine_track_unresolved",
  localisation_reference_not_in_project:
    "hint_localisation_reference_not_in_project",
  "bookmark.filter_registry_missing": "hint_bookmark_filter_registry_missing",
  "bookmark.default_missing": "hint_bookmark_default_missing",
  "bookmark.default_multiple": "hint_bookmark_default_multiple",
  "bookmark.default_country_not_interesting":
    "hint_bookmark_default_country_not_interesting",
  "bookmark.country_label_legacy_invalid_preserved":
    "hint_bookmark_country_label_invalid",
  "bookmark.country_label_invalid": "hint_bookmark_country_label_invalid",
  "bookmark.country_labels_filter_missing":
    "hint_bookmark_country_labels_filter_missing",
  "bookmark.filter_legacy_unknown_preserved": "hint_bookmark_filter_unknown",
  "bookmark.filter_invalid_for_target": "hint_bookmark_filter_unknown",
  "bookmark.filter_country_label_semantics":
    "hint_bookmark_filter_country_label_semantics",
  "bookmark.selectable_ideology_missing":
    "hint_bookmark_selectable_ideology_missing",
  "focus.initial_position.legacy_ambiguous":
    "hint_focus_initial_position_choose_kind",
  "focus.initial_position.invalid_shape":
    "hint_focus_initial_position_choose_kind",
  "focus.display_name.unproven_fallback":
    "hint_focus_display_name_unproven_fallback",
  focus_effective_default_multiple: "hint_focus_effective_default_multiple",
  event_namespace_mismatch: "hint_event_namespace_mismatch",
  event_number_mismatch: "hint_event_number_mismatch",
  focus_relative_position_not_found: "hint_focus_relative_position_not_found",
  focus_prereq_cycle: "hint_focus_prereq_cycle",
  focus_id_duplicate: "hint_focus_id_duplicate",
  politics_custom_root_no_random_subtype:
    "hint_politics_custom_root_no_random_subtype",
};

const SUMMARY_KEY_BY_CODE: Record<string, string> = {
  idea_no_modifier: "summary_idea_no_modifier",
  ideology_undefined_standalone: "summary_ideology_undefined",
  "focus.display_name.unproven_fallback":
    "summary_focus_display_name_unproven_fallback",
  focus_effective_default_multiple: "summary_focus_effective_default_multiple",
  event_namespace_mismatch: "summary_event_namespace_mismatch",
  event_number_mismatch: "summary_event_number_mismatch",
  focus_relative_position_not_found: "summary_focus_reference_problem",
  focus_prereq_not_found: "summary_focus_reference_problem",
  focus_mutex_not_found: "summary_focus_reference_problem",
  focus_prereq_cycle: "summary_focus_reference_problem",
  focus_id_duplicate: "summary_focus_id_duplicate",
  politics_custom_root_no_random_subtype:
    "summary_politics_custom_root_no_random_subtype",
};

const IMPACT_KEY_BY_CODE: Record<string, string> = {
  event_namespace_mismatch: "impact_event_namespace_mismatch",
  event_number_mismatch: "impact_event_number_mismatch",
  focus_relative_position_not_found: "impact_focus_reference_problem",
  focus_prereq_not_found: "impact_focus_reference_problem",
  focus_mutex_not_found: "impact_focus_reference_problem",
  focus_prereq_cycle: "impact_focus_reference_problem",
  focus_prereq_mutex_conflict: "impact_focus_reference_problem",
  focus_id_duplicate: "impact_focus_id_duplicate",
  "focus.display_name.unproven_fallback": "impact_focus_missing_translation",
  politics_custom_root_no_random_subtype:
    "impact_politics_custom_root_no_random_subtype",
};

const GROUP_KEY_BY_CODE: Record<string, string> = {
  event_namespace_mismatch: "event.identity",
  event_number_mismatch: "event.identity",
  focus_relative_position_not_found: "focus.references",
  focus_prereq_not_found: "focus.references",
  focus_mutex_not_found: "focus.references",
  focus_prereq_cycle: "focus.references",
  focus_prereq_mutex_conflict: "focus.references",
  focus_id_duplicate: "focus.identity",
  "focus.display_name.unproven_fallback": "focus.translation",
  idea_no_modifier: "idea.effects",
  politics_custom_root_no_random_subtype: "politics.ideologies",
};

const FAMILY_ALIASES: Record<string, string> = {
  focus_tree: "focus",
  bookmark_country: "bookmark",
  history_file: "country",
  province: "state",
  tech: "technology",
  ideology: "country",
};

const PLAYER_FAMILIES = new Set([
  "state",
  "country",
  "character",
  "focus",
  "idea",
  "event",
  "decision",
  "technology",
  "doctrine",
  "division_template",
  "bookmark",
  "project",
]);

export function issueInterp(
  issue: PlayerIssue,
): Record<string, string | number> {
  const out: Record<string, string | number> = {};
  const merge = (src: Record<string, unknown> | null | undefined) => {
    if (!src) return;
    for (const [key, value] of Object.entries(src)) {
      if (value === null || value === undefined) continue;
      if (Array.isArray(value)) out[key] = value.join(", ");
      else if (typeof value !== "object") out[key] = value as string | number;
    }
  };
  merge(issue.source && typeof issue.source === "object" ? issue.source : null);
  merge(issue.evidence);
  merge(issue.entity_ref as Record<string, unknown> | null);
  merge(issue.detail);
  return out;
}

export function issueFamily(issue: PlayerIssue): string {
  const refType = String(issue.entity_ref?.type ?? "")
    .trim()
    .toLowerCase();
  if (refType) {
    const family = FAMILY_ALIASES[refType] ?? refType;
    return PLAYER_FAMILIES.has(family) ? family : "other";
  }
  const codePrefix = String(issue.code ?? "")
    .split(/[._-]/, 1)[0]
    .toLowerCase();
  if (codePrefix === "tech") return "technology";
  if (
    [
      "focus",
      "idea",
      "event",
      "decision",
      "bookmark",
      "character",
      "country",
      "state",
      "project",
    ].includes(codePrefix)
  ) {
    return codePrefix;
  }
  if (String(issue.code).includes("ideology")) return "country";
  if (String(issue.code).includes("province")) return "state";
  return "other";
}

function safeTranslate(
  t: PlayerTranslator,
  key: string | undefined,
  values: Record<string, string | number>,
  fallback: string,
): string {
  if (!key) return fallback;
  try {
    const translated = t(key, values);
    return translated && !translated.endsWith(key) ? translated : fallback;
  } catch {
    return fallback;
  }
}

export function playerIssueCopy(
  t: PlayerTranslator,
  issue: PlayerIssue,
  count = 1,
): PlayerIssuePresentation {
  const family = issueFamily(issue);
  const kind = safeTranslate(t, `kind_${family}`, {}, t("kind_other"));
  const values = { ...issueInterp(issue), kind, count };
  const genericMessage = t("genericIssueMessage", { kind });
  const genericHint = t("genericIssueHint", { kind });
  const message =
    count > 1
      ? safeTranslate(
          t,
          SUMMARY_KEY_BY_CODE[issue.code],
          values,
          t("groupIssueCount", { count, kind }),
        )
      : safeTranslate(
          t,
          MESSAGE_KEY_BY_CODE[issue.code],
          values,
          genericMessage,
        );
  const hint = safeTranslate(
    t,
    HINT_KEY_BY_CODE[issue.code],
    values,
    genericHint,
  );
  const impactFallback =
    issue.severity === "blocker"
      ? t("impactBlocker")
      : issue.severity === "info"
        ? t("impactInfo")
        : t("impactWarning");
  const impact = safeTranslate(
    t,
    IMPACT_KEY_BY_CODE[issue.code],
    values,
    impactFallback,
  );
  const route =
    TYPE_TO_ROUTE[String(issue.entity_ref?.type ?? "")] ??
    TYPE_TO_ROUTE[family] ??
    null;
  const groupKey = GROUP_KEY_BY_CODE[issue.code] ?? family;
  const repairMode: PlayerRepairMode =
    issue.code === "character_advisor_traits_fallback"
      ? "automatic"
      : route
        ? "guided"
        : "manual";
  return { family, groupKey, message, impact, hint, route, repairMode };
}
