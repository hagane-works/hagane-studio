import type { Decision, DecisionCategory } from "@/types";
import { draftText } from "./focusEdit";

function spriteName(token: string, prefix: string): string {
  const t = token.trim();
  if (!t) return "";
  if (t.startsWith("GFX_")) return t;
  if (t.startsWith("decision_")) return `GFX_${t}`;
  return prefix + t;
}

export const decisionIconSprite = (token: string) =>
  spriteName(token, "GFX_decision_");

export const categoryIconSprite = (token: string) =>
  spriteName(token, "GFX_decision_category_");

export function categoryIconToken(sprite: string): string {
  if (sprite.startsWith("GFX_decision_category_"))
    return sprite.slice("GFX_decision_category_".length);
  if (sprite.startsWith("GFX_decision_")) return sprite.slice("GFX_".length);
  return sprite;
}

export interface DecisionDraft {
  name: string;
  desc: string;
  cost: number;
  icon: string;

  script?: DecisionScript;

  daysRemove?: number;
}

export const DECISION_SCRIPT_FIELDS = [
  "visible",
  "available",
  "complete_effect",
  "remove_effect",
  "modifier",
  "ai_will_do",
] as const;
export type DecisionScriptField = (typeof DECISION_SCRIPT_FIELDS)[number];
export type DecisionScript = Record<DecisionScriptField, string>;

export function decisionScriptOf(row: Decision): DecisionScript {
  const out = {} as DecisionScript;
  for (const field of DECISION_SCRIPT_FIELDS)
    out[field] = String(
      (row as unknown as Record<string, unknown>)[field] ?? "",
    );
  return out;
}

export function decisionDeletable(row: Decision): boolean {
  return (row as unknown as { source?: string }).source === "created";
}

export function decisionDraft(
  row: Decision,
  shown: { label?: string; cost?: number },
  shownDesc: string,
): DecisionDraft {
  return {
    name: draftText(
      row.display_name_locs,
      row.name_en || undefined,
      shown.label ?? row.decision_id,
    ),
    desc: draftText(row.description_locs, row.desc_en || undefined, shownDesc),
    cost: row.cost ?? shown.cost ?? 0,
    icon: row.icon ?? "",
    script: decisionScriptOf(row),
    daysRemove: row.days_remove ?? 0,
  };
}

export function decisionPatch(
  row: Decision,
  draft: DecisionDraft,
  start: DecisionDraft,
): Partial<Decision> {
  const patch: Partial<Decision> = {};
  if (draft.name !== start.name)
    patch.display_name_locs = {
      ...(row.display_name_locs ?? {}),
      en: draft.name,
    };
  if (draft.desc !== start.desc)
    patch.description_locs = {
      ...(row.description_locs ?? {}),
      en: draft.desc,
    };
  if (draft.cost !== start.cost) patch.cost = draft.cost;
  if (draft.icon !== start.icon) patch.icon = draft.icon;
  if (draft.script && start.script) {
    for (const field of DECISION_SCRIPT_FIELDS) {
      if (draft.script[field] !== start.script[field]) {
        (patch as Record<string, unknown>)[field] = draft.script[field];
      }
    }
  }
  if (draft.daysRemove !== undefined && draft.daysRemove !== start.daysRemove)
    patch.days_remove = draft.daysRemove;
  return patch;
}

export function categoryForEditor(
  row: DecisionCategory,
  shownIcon: string | undefined,
): DecisionCategory {
  return row.icon || !shownIcon
    ? row
    : { ...row, icon: categoryIconToken(shownIcon) };
}

export function newDecisionId(tag: string, taken: readonly string[]): string {
  const used = new Set(taken.map((id) => id.toLowerCase()));
  const base = `${tag}_new_decision`;
  if (!used.has(base.toLowerCase())) return base;
  for (let i = 2; ; i++) {
    const id = `${base}_${i}`;
    if (!used.has(id.toLowerCase())) return id;
  }
}
