import type {
  AirWing,
  AirWingCreate,
  AirWingGroup,
  AirWingGroupCreate,
  Fleet,
  FleetUpdate,
  Ship,
  ShipUpdate,
  TaskForce,
  TaskForceUpdate,
} from "@/types/oobUnits";
import type { Equipment } from "@/types";

export type OobEquipmentDomain = "air" | "naval";

const AIR_EQUIPMENT_TYPES = new Set([
  "fighter",
  "cas",
  "naval_bomber",
  "tactical_bomber",
  "strategic_bomber",
  "scout_plane",
]);
const NAVAL_EQUIPMENT_TYPES = new Set(["ship", "submarine", "convoy"]);

const AIR_EVIDENCE_FIELDS: Array<keyof Equipment> = [
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
];
const NAVAL_EVIDENCE_FIELDS: Array<keyof Equipment> = [
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
];

export function equipmentDomain(row: Equipment): OobEquipmentDomain | null {
  const evidence = new Set<OobEquipmentDomain>();
  if (AIR_EQUIPMENT_TYPES.has(row.equipment_type)) evidence.add("air");
  if (NAVAL_EQUIPMENT_TYPES.has(row.equipment_type)) evidence.add("naval");
  if (AIR_EVIDENCE_FIELDS.some((field) => typeof row[field] === "number"))
    evidence.add("air");
  if (NAVAL_EVIDENCE_FIELDS.some((field) => typeof row[field] === "number"))
    evidence.add("naval");
  return evidence.size === 1 ? [...evidence][0] : null;
}

export function buildOobEquipmentSuggestions(
  domain: OobEquipmentDomain,
  projectEquipment: Equipment[],
  vanillaArchetypes: string[],
): string[] {
  const catalog = new Map<string, OobEquipmentDomain | null>();
  for (const token of vanillaArchetypes) {
    if (token.trim()) catalog.set(token.trim(), null);
  }
  for (const row of projectEquipment) {
    const token = row.equipment_id?.trim();
    if (token) catalog.set(token, equipmentDomain(row));
  }
  return [...catalog.entries()]
    .filter(([, rowDomain]) => rowDomain == null || rowDomain === domain)
    .map(([token]) => token)
    .sort((left, right) => left.localeCompare(right));
}

export const isKnownEquipmentToken = (
  value: string,
  suggestions: string[],
): boolean => !value.trim() || suggestions.includes(value.trim());

export interface AirWingGroupDraft {
  oob_file: string;
  source_file: string;
  country_tag: string;
  location: string;
  ordering: string;
}

export interface AirWingDraft {
  group_id: string;
  equipment_type: string;
  name: string;
  owner: string;
  amount: string;
  ordering: string;
}

export interface FleetDraft {
  oob_file: string;
  source_file: string;
  country_tag: string;
  name: string;
  naval_base: string;
  ordering: string;
}

export interface TaskForceDraft {
  fleet_id: string;
  name: string;
  location: string;
  ordering: string;
}

export interface ShipDraft {
  task_force_id: string;
  name: string;
  definition: string;
  start_experience_factor: string;
  equipment_type: string;
  amount: string;
  owner: string;
  version_name: string;
  ordering: string;
}

export type OobEntityDraft =
  | { kind: "air-wing-group"; rowId: number; value: AirWingGroupDraft }
  | { kind: "air-wing"; rowId: number; value: AirWingDraft }
  | { kind: "fleet"; rowId: number; value: FleetDraft }
  | { kind: "task-force"; rowId: number; value: TaskForceDraft }
  | { kind: "ship"; rowId: number; value: ShipDraft };

const orderingValue = (value: string): number => {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : 0;
};

export const airWingGroupToDraft = (row: AirWingGroup): AirWingGroupDraft => ({
  oob_file: row.oob_file,
  source_file: row.source_file,
  country_tag: row.country_tag,
  location: row.location,
  ordering: String(row.ordering),
});

export const airWingToDraft = (row: AirWing): AirWingDraft => ({
  group_id: String(row.group_id),
  equipment_type: row.equipment_type,
  name: row.name,
  owner: row.owner,
  amount: row.amount,
  ordering: String(row.ordering),
});

export const fleetToDraft = (row: Fleet): FleetDraft => ({
  oob_file: row.oob_file,
  source_file: row.source_file,
  country_tag: row.country_tag,
  name: row.name,
  naval_base: row.naval_base,
  ordering: String(row.ordering),
});

export const taskForceToDraft = (row: TaskForce): TaskForceDraft => ({
  fleet_id: String(row.fleet_id),
  name: row.name,
  location: row.location,
  ordering: String(row.ordering),
});

export const shipToDraft = (row: Ship): ShipDraft => ({
  task_force_id: String(row.task_force_id),
  name: row.name,
  definition: row.definition,
  start_experience_factor: row.start_experience_factor,
  equipment_type: row.equipment_type,
  amount: row.amount,
  owner: row.owner,
  version_name: row.version_name,
  ordering: String(row.ordering),
});

export const airWingGroupUpdate = (
  draft: AirWingGroupDraft,
): AirWingGroupCreate => ({
  oob_file: draft.oob_file.trim(),
  source_file: draft.source_file.trim().replaceAll("\\", "/"),
  country_tag: draft.country_tag.trim().toUpperCase(),
  location: draft.location.trim(),
  ordering: orderingValue(draft.ordering),
});

export const airWingUpdate = (draft: AirWingDraft): AirWingCreate => ({
  group_id: orderingValue(draft.group_id),
  equipment_type: draft.equipment_type.trim(),
  name: draft.name.trim(),
  owner: draft.owner.trim(),
  amount: draft.amount.trim(),
  ordering: orderingValue(draft.ordering),
});

export const fleetUpdate = (draft: FleetDraft): FleetUpdate => ({
  oob_file: draft.oob_file.trim(),
  source_file: draft.source_file.trim().replaceAll("\\", "/"),
  country_tag: draft.country_tag.trim().toUpperCase(),
  name: draft.name.trim(),
  naval_base: draft.naval_base.trim(),
  ordering: orderingValue(draft.ordering),
});

export const taskForceUpdate = (draft: TaskForceDraft): TaskForceUpdate => ({
  fleet_id: orderingValue(draft.fleet_id),
  name: draft.name.trim(),
  location: draft.location.trim(),
  ordering: orderingValue(draft.ordering),
});

export const shipUpdate = (draft: ShipDraft): ShipUpdate => ({
  task_force_id: orderingValue(draft.task_force_id),
  name: draft.name.trim(),
  definition: draft.definition.trim(),
  start_experience_factor: draft.start_experience_factor.trim(),
  equipment_type: draft.equipment_type.trim(),
  amount: draft.amount.trim(),
  owner: draft.owner.trim(),
  version_name: draft.version_name.trim(),
  ordering: orderingValue(draft.ordering),
});
