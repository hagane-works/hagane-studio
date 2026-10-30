import { useSyncExternalStore } from "react";
import { sessionGameData } from "@/lib/api/gameIndex";
import { vanillaDataApi } from "@/lib/api/vanilla";

const archetypes = sessionGameData(
  () =>
    vanillaDataApi
      .equipmentArchetypes()
      .then((data) => new Set<string>(data.archetypes)),
  new Set<string>(),
);

export function useVanillaEquipmentArchetypes(): ReadonlySet<string> {
  return useSyncExternalStore(
    archetypes.subscribe,
    archetypes.current,
    archetypes.current,
  );
}

export const EQUIPMENT_BONUS_RESERVED_KEYS: readonly string[] = ["instant"];

const RESERVED_SET = new Set<string>(EQUIPMENT_BONUS_RESERVED_KEYS);

export function isEquipmentBonusReservedKey(key: string): boolean {
  return RESERVED_SET.has(key);
}

export function detectInstanceArchetype(
  key: string,
  archetypes: ReadonlySet<string>,
): string | null {
  const m = key.match(/^(.+)_(\d+)$/);
  if (!m) return null;
  const prefix = m[1];
  return archetypes.has(prefix) ? prefix : null;
}

export const COMMON_ARCHETYPES: readonly string[] = [
  "infantry_equipment",
  "artillery_equipment",
  "support_equipment",
  "light_tank_chassis",
  "medium_tank_chassis",
  "fighter_equipment",
  "tac_bomber_equipment",
  "ship_hull_light",
];
