import type { Equipment } from "@/types";

export interface GameEquipment {
  name: string;
  found: boolean;
  label?: string;

  country_key?: string;
  country_label?: string;
  file?: string;
  archetype?: string;
  archetype_label?: string;
  is_archetype?: boolean;
  year?: number;
  parent?: string;
  numbers?: Record<string, number>;
  resources?: Record<string, number>;
  inherited?: Record<string, number>;
  inherited_resources?: Record<string, number>;
}

export interface EquipmentDraft {
  label: string;

  countryLabel: string;
  year: string;

  numbers: Record<string, string>;
  resources: Record<string, string>;
}

export const STAT_ORDER: readonly string[] = [
  "soft_attack",
  "hard_attack",
  "ap_attack",
  "air_attack",
  "defense",
  "breakthrough",
  "armor_value",
  "hardness",
  "maximum_speed",
  "reliability",
  "fuel_capacity",
  "fuel_consumption",
  "build_cost_ic",
  "lend_lease_cost",
  "manpower",
  "air_defence",
  "air_range",
  "air_agility",
  "air_ground_attack",
  "air_bombing",
  "naval_strike_attack",
  "naval_strike_targetting",
  "naval_speed",
  "lg_attack",
  "lg_armor_piercing",
  "hg_attack",
  "hg_armor_piercing",
  "torpedo_attack",
  "anti_air_attack",
  "surface_detection",
  "sub_detection",
  "surface_visibility",
  "sub_visibility",
];

export function statKeys(game: GameEquipment): string[] {
  const keys = new Set([
    ...Object.keys(game.inherited ?? {}),
    ...Object.keys(game.numbers ?? {}),
  ]);
  return [...keys].sort((a, b) => {
    const ia = STAT_ORDER.indexOf(a);
    const ib = STAT_ORDER.indexOf(b);
    if (ia >= 0 || ib >= 0)
      return (
        (ia < 0 ? STAT_ORDER.length : ia) - (ib < 0 ? STAT_ORDER.length : ib)
      );
    return a.localeCompare(b);
  });
}

export function resourceKeys(game: GameEquipment): string[] {
  return [
    ...new Set([
      ...Object.keys(game.inherited_resources ?? {}),
      ...Object.keys(game.resources ?? {}),
    ]),
  ].sort();
}

export function equipmentDraft(game: GameEquipment): EquipmentDraft {
  const numbers: Record<string, string> = {};
  for (const key of statKeys(game))
    numbers[key] =
      key in (game.numbers ?? {}) ? String(game.numbers?.[key]) : "";
  const resources: Record<string, string> = {};
  for (const key of resourceKeys(game))
    resources[key] =
      key in (game.resources ?? {}) ? String(game.resources?.[key]) : "";
  return {
    label: game.label ?? game.name,
    countryLabel: game.country_label ?? "",
    year: game.year ? String(game.year) : "",
    numbers,
    resources,
  };
}

const NUMBER = /^-?\d+(\.\d+)?$/;

export type EquipmentProblem =
  | { key: "nameRequired" | "countryNameKept" | "yearFormat" }
  | { key: "notNumber"; field: string };

export function equipmentProblem(
  draft: EquipmentDraft,
  game?: GameEquipment,
): EquipmentProblem | null {
  if (!draft.label.trim()) return { key: "nameRequired" };
  if (game?.country_label && !draft.countryLabel.trim())
    return { key: "countryNameKept" };
  if (draft.year.trim() && !/^\d{4}$/.test(draft.year.trim()))
    return { key: "yearFormat" };
  for (const [key, value] of [
    ...Object.entries(draft.numbers),
    ...Object.entries(draft.resources),
  ]) {
    if (value.trim() && !NUMBER.test(value.trim()))
      return { key: "notNumber", field: key };
  }
  return null;
}

export function equipmentPatch(draft: EquipmentDraft, start: EquipmentDraft) {
  const values: Record<string, number | null> = {};
  for (const [key, value] of Object.entries(draft.numbers)) {
    if (value.trim() !== (start.numbers[key] ?? "").trim())
      values[key] = value.trim() ? Number(value) : null;
  }
  if (draft.year.trim() !== start.year.trim() && draft.year.trim())
    values.year = Number(draft.year);
  const resources: Record<string, number | null> = {};
  for (const [key, value] of Object.entries(draft.resources)) {
    if (value.trim() !== (start.resources[key] ?? "").trim())
      resources[key] = value.trim() ? Number(value) : null;
  }
  const countryLabel = draft.countryLabel.trim();
  return {
    values,
    resources,
    label: draft.label.trim() !== start.label ? draft.label.trim() : null,
    countryLabel:
      countryLabel && countryLabel !== start.countryLabel ? countryLabel : null,
  };
}

export function projectEquipmentRows(
  rows: readonly Equipment[] | undefined,
  name: string,
): { own: Equipment | null; imported: boolean } {
  const want = name.toLowerCase();
  const same =
    rows?.filter((row) => row.equipment_id.toLowerCase() === want) ?? [];
  return {
    own: same.find((row) => row.source !== "imported") ?? null,
    imported: same.some((row) => row.source === "imported"),
  };
}

export type NewEquipmentProblem =
  | { key: "tokenRequired" | "tokenInvalid" }
  | { key: "tokenTaken"; token: string };

export function newEquipmentProblem(
  token: string,
  taken: readonly string[],
): NewEquipmentProblem | null {
  const id = token.trim();
  if (!id) return { key: "tokenRequired" };
  if (!/^[a-z_][a-z0-9_]*$/.test(id)) return { key: "tokenInvalid" };
  if (taken.includes(id)) return { key: "tokenTaken", token: id };
  return null;
}
