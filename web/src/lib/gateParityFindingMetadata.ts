import type { FindingMeta } from "@/lib/exportFindingsMap";

export const GATE_PARITY_FINDINGS: Record<string, FindingMeta> = {
  "bookmark.filter_registry_missing": {
    tier: "health",
    surface: "export",
    i18nKey: "bookmark_filter_registry_missing",
  },
  "bookmark.default_multiple": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_default_multiple",
  },
  "bookmark.default_country_not_interesting": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_default_country_not_interesting",
  },
  "bookmark.country_label_invalid": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_country_label_invalid",
  },
  "bookmark.country_label_legacy_invalid_preserved": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_country_label_invalid",
  },
  "bookmark.country_labels_filter_missing": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_country_labels_filter_missing",
  },
  "bookmark.filter_invalid_for_target": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_filter_invalid_for_target",
  },
  "bookmark.filter_legacy_unknown_preserved": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_filter_invalid_for_target",
  },
  "bookmark.filter_country_label_semantics": {
    tier: "attn",
    surface: "export",
    i18nKey: "bookmark_filter_country_label_semantics",
  },
  focus_layout_missing_focus_id: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_missing_focus_id",
  },
  focus_layout_duplicate_focus_id: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_duplicate_focus_id",
  },
  focus_layout_relative_cycle: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_relative_cycle",
  },
  focus_layout_missing_anchor: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_missing_anchor",
  },
  focus_layout_replacement_global_shift: {
    tier: "attn",
    surface: "export",
    i18nKey: "focus_layout_replacement_global_shift",
  },
  focus_layout_unknown_mode: {
    tier: "health",
    surface: "export",
    i18nKey: "focus_layout_unknown_mode",
  },
  "timeline.base_timeline_source_unregistered": {
    tier: "attn",
    surface: "export",
    i18nKey: "timeline_base_timeline_source_unregistered",
  },
};

export function gateParityRepairRoute(
  code: string,
  entityRefType: string,
): string | null {
  if (
    entityRefType === "bookmark_set" &&
    code === "bookmark.default_multiple"
  ) {
    return "bookmarks";
  }
  if (
    entityRefType === "focus_policy" &&
    ((code in GATE_PARITY_FINDINGS && code.startsWith("focus_layout_")) ||
      code === "focus_layout_same_cell" ||
      code === "focus_layout_adjacent_same_row")
  ) {
    return "focus-tree";
  }
  if (
    entityRefType === "timeline_plan" &&
    code === "timeline.base_timeline_source_unregistered"
  ) {
    return "bookmarks";
  }
  return null;
}
