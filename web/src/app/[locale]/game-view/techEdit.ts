import type { Technology } from "@/types";
import { draftText } from "./focusEdit";

export interface TechDraft {
  name: string;
  desc: string;
  cost: number;
}

export function techDraft(
  row: Technology,
  shown: { label?: string; cost?: number },
): TechDraft {
  return {
    name: draftText(
      row.display_name_locs,
      row.name_en || undefined,
      shown.label ?? row.tech_id,
    ),
    desc: draftText(row.description_locs, row.desc_en || undefined, ""),
    cost: row.research_cost ?? shown.cost ?? 0,
  };
}

export function techPatch(
  row: Technology,
  draft: TechDraft,
  start: TechDraft,
): Partial<Technology> {
  const patch: Partial<Technology> = {};
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
  if (draft.cost !== start.cost) patch.research_cost = draft.cost;
  return patch;
}

export function findTechRow(
  rows: readonly Technology[] | undefined,
  techId: string,
  tag: string,
): Technology | null {
  const want = techId.toLowerCase();
  const matches = (rows ?? []).filter(
    (row) => row.tech_id.toLowerCase() === want,
  );
  const own = matches.find(
    (row) => (row.country_tag ?? "").toUpperCase() === tag.toUpperCase(),
  );
  return own ?? matches.find((row) => !row.country_tag) ?? matches[0] ?? null;
}

export function withResearched(
  ids: readonly string[],
  techId: string,
  researched: boolean,
): string[] {
  const want = techId.toLowerCase();
  const out = ids.filter((id) => id.toLowerCase() !== want);
  if (researched) out.push(techId);
  return out;
}

export function startToggleReaches(
  ids: readonly string[],
  techId: string,
  shownResearched: boolean,
): boolean {
  const want = techId.toLowerCase();
  return !shownResearched || ids.some((id) => id.toLowerCase() === want);
}

export function researchDays(
  cost: number,
  baseTechCost: number | undefined,
): number | null {
  if (!baseTechCost || !Number.isFinite(cost)) return null;
  return Math.round(cost * baseTechCost);
}
