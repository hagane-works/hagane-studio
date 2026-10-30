import type { Technology } from "@/types";
import type { GuiTechnology } from "./TechTreeLayer";
import { slugOf } from "./personCreate";

export const TECH_YEAR_ROWS: readonly { year: number; row: number }[] = [
  { year: 1936, row: 2 },
  { year: 1938, row: 4 },
  { year: 1939, row: 6 },
  { year: 1940, row: 8 },
  { year: 1941, row: 10 },
  { year: 1942, row: 12 },
  { year: 1943, row: 14 },
  { year: 1944, row: 16 },
  { year: 1945, row: 18 },
  { year: 1946, row: 20 },
];

export interface NewTechDraft {
  name: string;
  year: number;

  x: number;
  cost: number;
}

export function newTechDraft(parent: GuiTechnology): NewTechDraft {
  const next =
    TECH_YEAR_ROWS.find((column) => column.year > parent.start_year) ??
    TECH_YEAR_ROWS[TECH_YEAR_ROWS.length - 1];
  return { name: "", year: next.year, x: parent.x, cost: parent.cost };
}

export function rowOfYear(year: number): number {
  return TECH_YEAR_ROWS.find((column) => column.year === year)?.row ?? 2;
}

export function techToken(name: string): string {
  return slugOf(name) || "new_technology";
}

export type NewTechProblem =
  | {
      key: "nameRequired" | "nameInvalid" | "xInvalid" | "costInvalid";
      values?: undefined;
    }
  | { key: "idTaken"; values: { id: string } };

export function newTechProblem(
  draft: NewTechDraft,
  taken: ReadonlySet<string>,
): NewTechProblem | null {
  const name = draft.name.trim();
  if (!name) return { key: "nameRequired" };
  if (/["\r\n]/.test(name)) return { key: "nameInvalid" };
  if (taken.has(techToken(name)))
    return { key: "idTaken", values: { id: techToken(name) } };
  if (!Number.isInteger(draft.x) || draft.x < 0 || draft.x > 20)
    return { key: "xInvalid" };
  if (!(draft.cost > 0)) return { key: "costInvalid" };
  return null;
}

export function parentFolder(parent: GuiTechnology, row: Technology): string {
  const placements = row.folder_placements ?? [];
  const here = placements.find(
    (p) => Number(p.x) === parent.x && Number(p.y) === parent.y,
  );
  return String(here?.folder ?? row.folder ?? "");
}

export function newTechRow(
  parent: GuiTechnology,
  parentRow: Technology,
  draft: NewTechDraft,
): Partial<Technology> {
  const name = draft.name.trim();
  const folder = parentFolder(parent, parentRow);
  const y = rowOfYear(draft.year);
  return {
    tech_id: techToken(name),
    name,
    name_en: name,
    display_name_locs: { en: name },
    tech_line: parentRow.tech_line,
    country_tag: parentRow.country_tag,
    research_cost: draft.cost,
    start_year: draft.year,
    folder,
    position_x: draft.x,
    position_y: y,
    position_y_is_explicit: true,
    folder_placements: [{ folder, x: draft.x, y }],
    prerequisites: [parent.name],
    categories: [...(parentRow.categories ?? [])],
  };
}

export function withPathTo(
  path: Technology["path"],
  techId: string,
): NonNullable<Technology["path"]> {
  const edges = [...(path ?? [])];
  if (!edges.some((edge) => edge.leads_to_tech === techId))
    edges.push({ leads_to_tech: techId, research_cost_coeff: 1 });
  return edges;
}

export function withoutReferences(
  rows: readonly Technology[],
  techId: string,
): { id: number; patch: Partial<Technology> }[] {
  const out: { id: number; patch: Partial<Technology> }[] = [];
  for (const row of rows) {
    if (row.tech_id === techId) continue;
    const patch: Partial<Technology> = {};
    if ((row.path ?? []).some((edge) => edge.leads_to_tech === techId)) {
      patch.path = (row.path ?? []).filter(
        (edge) => edge.leads_to_tech !== techId,
      );
    }
    if ((row.prerequisites ?? []).includes(techId)) {
      patch.prerequisites = row.prerequisites.filter((id) => id !== techId);
    }
    if (Object.keys(patch).length) out.push({ id: row.id, patch });
  }
  return out;
}

export function shownNewTech(row: Technology): GuiTechnology {
  return {
    name: row.tech_id,
    label: row.display_name_locs?.en || row.name,
    x: row.position_x,
    y: row.position_y,
    cost: row.research_cost,
    start_year: row.start_year,
    doctrine: false,
    researched: false,
    added_by_project: true,
  };
}
