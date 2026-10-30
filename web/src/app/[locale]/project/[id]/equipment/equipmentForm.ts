import type { Equipment } from "@/types";

export type TriState = "unset" | "yes" | "no";

export const OFFENSIVE_FIELDS = [
  "soft_attack",
  "hard_attack",
  "ap_attack",
  "air_attack",
  "breakthrough",
] as const;

export const DEFENSIVE_FIELDS = [
  "defense",
  "armor_value",
  "hardness",
  "max_strength",
] as const;

export const ECONOMY_FIELDS = [
  "build_cost_ic",
  "manpower",
  "reliability",
  "maximum_speed",
] as const;

export const NAVY_FIELDS = [
  "naval_speed",
  "fire_range",
  "lg_armor_piercing",
  "lg_attack",
  "hg_armor_piercing",
  "hg_attack",
  "torpedo_attack",
  "anti_air_attack",
  "shore_bombardment",
  "evasion",
  "surface_detection",
  "sub_attack",
  "sub_detection",
  "surface_visibility",
  "sub_visibility",
  "naval_range",
  "port_capacity_usage",
  "search_and_destroy_coordination",
  "convoy_raiding_coordination",
] as const;

export const AIR_FIELDS = [
  "air_defence",
  "air_range",
  "air_agility",
  "air_ground_attack",
  "air_bombing",
  "air_superiority",
  "naval_strike_attack",
  "naval_strike_targetting",
  "carrier_size",
  "default_carrier_composition_weight",
] as const;

export type EquipmentNumericKey =
  | (typeof OFFENSIVE_FIELDS)[number]
  | (typeof DEFENSIVE_FIELDS)[number]
  | (typeof ECONOMY_FIELDS)[number]
  | (typeof NAVY_FIELDS)[number]
  | (typeof AIR_FIELDS)[number];

export const EQUIPMENT_NUMERIC_FIELDS: readonly EquipmentNumericKey[] = [
  ...OFFENSIVE_FIELDS,
  ...DEFENSIVE_FIELDS,
  ...ECONOMY_FIELDS,
  ...NAVY_FIELDS,
  ...AIR_FIELDS,
];

export interface EquipmentDraft {
  equipment_id: string;
  equipment_type: string;
  year: string;
  picture: string;
  archetype: string;
  parent: string;
  source_file: string;
  is_archetype: boolean;
  is_buildable: TriState;
  can_license: TriState;
  carrier_capable: TriState;
  name_en: string;
  name_zh: string;
  display_name_locs: Record<string, string>;
  stats: Record<EquipmentNumericKey, string>;
  resourcePairs: { key: string; value: string }[];
}

export const triStateFromValue = (value: boolean | null): TriState =>
  value === null || value === undefined ? "unset" : value ? "yes" : "no";

export const valueFromTriState = (value: TriState): boolean | null =>
  value === "unset" ? null : value === "yes";

const numberString = (value: number | null | undefined): string =>
  value === null || value === undefined ? "" : String(value);

const nullableNumber = (value: string): number | null =>
  value.trim() === "" ? null : Number(value);

export function equipmentToDraft(equipment: Equipment): EquipmentDraft {
  const stats = {} as Record<EquipmentNumericKey, string>;
  for (const field of EQUIPMENT_NUMERIC_FIELDS) {
    stats[field] = numberString(equipment[field]);
  }
  return {
    equipment_id: equipment.equipment_id ?? "",
    equipment_type: equipment.equipment_type ?? "",
    year: numberString(equipment.year),
    picture: equipment.picture ?? "",
    archetype: equipment.archetype ?? "",
    parent: equipment.parent ?? "",
    source_file: equipment.source_file ?? "",
    is_archetype: Boolean(equipment.is_archetype),
    is_buildable: triStateFromValue(equipment.is_buildable),
    can_license: triStateFromValue(equipment.can_license),
    carrier_capable: triStateFromValue(equipment.carrier_capable),
    name_en: equipment.name_en ?? "",
    name_zh: equipment.name_zh ?? "",
    display_name_locs: { ...(equipment.display_name_locs ?? {}) },
    stats,
    resourcePairs: Object.entries(equipment.resources ?? {}).map(
      ([key, value]) => ({
        key,
        value: String(value),
      }),
    ),
  };
}

export function equipmentDraftToPayload(
  draft: EquipmentDraft,
): Partial<Equipment> {
  const payload: Record<string, unknown> = {
    equipment_id: draft.equipment_id.trim(),
    equipment_type: draft.equipment_type.trim(),
    year: draft.year.trim() === "" ? null : Math.trunc(Number(draft.year)),
    picture: draft.picture.trim(),
    archetype: draft.archetype.trim(),
    parent: draft.parent.trim(),
    source_file: draft.source_file.trim().replaceAll("\\", "/"),
    is_archetype: draft.is_archetype,
    is_buildable: valueFromTriState(draft.is_buildable),
    can_license: valueFromTriState(draft.can_license),
    carrier_capable: valueFromTriState(draft.carrier_capable),
    name_en: draft.name_en,
    name_zh: draft.name_zh,
    display_name_locs: { ...draft.display_name_locs },
  };

  for (const field of EQUIPMENT_NUMERIC_FIELDS) {
    const value = nullableNumber(draft.stats[field]);
    payload[field] =
      field === "manpower" && value !== null ? Math.trunc(value) : value;
  }

  const resources: Record<string, unknown> = {};
  for (const pair of draft.resourcePairs) {
    const key = pair.key.trim();
    if (!key) continue;
    const value = pair.value.trim();
    resources[key] =
      value === "" ? 0 : Number.isFinite(Number(value)) ? Number(value) : value;
  }
  payload.resources = resources;
  return payload as Partial<Equipment>;
}
