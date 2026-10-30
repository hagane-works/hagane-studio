import { useEffect, useState } from "react";
import { requestGameIndexed } from "@/lib/api/gameIndex";

interface VanillaCollisions {
  techIds: Set<string>;
  focusTreeIds: Set<string>;
  decisionCategoryIds: Set<string>;
  dynamicModifierIds: Set<string>;
  loaded: boolean;
}

const EMPTY: VanillaCollisions = {
  techIds: new Set(),
  focusTreeIds: new Set(),
  decisionCategoryIds: new Set(),
  dynamicModifierIds: new Set(),
  loaded: false,
};

let cache: VanillaCollisions | null = null;
let inflight: Promise<VanillaCollisions> | null = null;

async function fetchVanillaCollisions(): Promise<VanillaCollisions> {
  if (cache) return cache;
  if (inflight) return inflight;
  inflight = (async () => {
    try {
      const data = await requestGameIndexed<{
        tech_ids?: string[];
        focus_tree_ids?: string[];
        decision_category_ids?: string[];
        dynamic_modifier_ids?: string[];
      }>("/go/vanilla/id-collisions");
      const result: VanillaCollisions = {
        techIds: new Set(data.tech_ids ?? []),
        focusTreeIds: new Set(data.focus_tree_ids ?? []),
        decisionCategoryIds: new Set(data.decision_category_ids ?? []),
        dynamicModifierIds: new Set(data.dynamic_modifier_ids ?? []),
        loaded: true,
      };
      cache = result;
      return result;
    } catch {
      return {
        techIds: new Set<string>(),
        focusTreeIds: new Set<string>(),
        decisionCategoryIds: new Set<string>(),
        dynamicModifierIds: new Set<string>(),
        loaded: true,
      };
    } finally {
      inflight = null;
    }
  })();
  return inflight;
}

export function useVanillaCollisions(): VanillaCollisions {
  const [data, setData] = useState<VanillaCollisions>(cache ?? EMPTY);
  useEffect(() => {
    if (cache) return;
    let cancelled = false;
    fetchVanillaCollisions().then((d) => {
      if (!cancelled) setData(d);
    });
    return () => {
      cancelled = true;
    };
  }, []);
  return data;
}

export function suggestNonCollidingTechId(
  techId: string,
  countryTag: string,
): string {
  const prefix = countryTag ? `${countryTag.toUpperCase()}_` : "MOD_";
  if (techId.startsWith(prefix)) return techId;
  return `${prefix}${techId}`;
}

export const suggestNonCollidingId = suggestNonCollidingTechId;
