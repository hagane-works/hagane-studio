import type { CtdIssue } from "@/lib/api/export";

export const IMPASSABLE_CATEGORY_NOTICE_CODE = "impassable_category_mismatch";

export const VANILLA_IMPASSABLE_CATEGORIES: ReadonlySet<string> = new Set([
  "wasteland",
  "enclave",
]);

export interface ImpassableNoticeState {
  state_id: number;
  impassable?: boolean | null;
  state_category?: string | null;
}

export function selectImpassableCategoryMismatches(
  states: readonly ImpassableNoticeState[] | null | undefined,
): number[] {
  if (!Array.isArray(states) || states.length === 0) return [];
  const hits = new Set<number>();
  for (const state of states) {
    if (!state || typeof state !== "object") continue;
    if (state.impassable !== true) continue;
    const id = state.state_id;
    if (typeof id !== "number" || !Number.isFinite(id)) continue;
    const category = (state.state_category ?? "").trim().toLowerCase();
    if (VANILLA_IMPASSABLE_CATEGORIES.has(category)) continue;
    hits.add(id);
  }

  return [...hits].sort((a, b) => a - b);
}

export function mismatchedCategories(
  states: readonly ImpassableNoticeState[] | null | undefined,
): string[] {
  if (!Array.isArray(states)) return [];
  const cats = new Set<string>();
  for (const state of states) {
    if (!state || state.impassable !== true) continue;
    const raw = (state.state_category ?? "").trim();
    if (VANILLA_IMPASSABLE_CATEGORIES.has(raw.toLowerCase())) continue;
    cats.add(raw === "" ? "(empty)" : raw);
  }
  return [...cats].sort();
}

export function buildImpassableCategoryNotice(
  states: readonly ImpassableNoticeState[] | null | undefined,
): CtdIssue | null {
  const stateIds = selectImpassableCategoryMismatches(states);
  if (stateIds.length === 0) return null;
  const categories = mismatchedCategories(states);
  return {
    code: IMPASSABLE_CATEGORY_NOTICE_CODE,
    entity_ref: null,
    severity: "info",

    message:
      `${stateIds.length} impassable states use a category vanilla never ` +
      "pairs with impassable",

    fix_hint: "",
    detail: { count: stateIds.length },
    evidence: { state_ids: stateIds, categories },
  };
}
