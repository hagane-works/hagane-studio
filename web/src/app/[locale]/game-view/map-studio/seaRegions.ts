import type { WorldMap } from "@/components/workbench/world-map/worldMapModel";
import type { Neighbours } from "./ownerPaint";

export interface SeaRegionSplit {
  region: number;

  provinces: number[];
}

export function seaRegionSplits(
  map: WorldMap,
  neighbours: Neighbours,
): SeaRegionSplit[] {
  const members = new Map<number, number[]>();
  for (let i = 1; i < map.provinces.length; i++) {
    const region = map.provinces[i]?.region;
    if (!region) continue;
    const list = members.get(region);
    if (list) list.push(i);
    else members.set(region, [i]);
  }
  const isSea = (i: number) => map.provinces[i]?.kind === "sea";
  const seas = (part: number[]) => part.filter(isSea).length;
  const lowest = (part: number[]) =>
    Math.min(...part.map((i) => map.provinces[i].id));
  const out: SeaRegionSplit[] = [];
  for (const [region, list] of members) {
    if (!list.some(isSea)) continue;
    const inside = new Set(list);
    const seen = new Set<number>();
    const parts: number[][] = [];
    for (const first of list) {
      if (seen.has(first)) continue;
      seen.add(first);
      const part = [first];
      for (let k = 0; k < part.length; k++) {
        for (const next of neighbours.get(part[k]) ?? []) {
          if (inside.has(next) && !seen.has(next)) {
            seen.add(next);
            part.push(next);
          }
        }
      }
      if (part.some(isSea)) parts.push(part);
    }
    if (parts.length < 2) continue;
    parts.sort((a, b) => seas(b) - seas(a) || lowest(a) - lowest(b));
    const provinces = parts
      .slice(1)
      .flat()
      .map((i) => map.provinces[i].id)
      .sort((a, b) => a - b);
    out.push({ region, provinces });
  }
  return out.sort((a, b) => a.region - b.region);
}

export function statesAcrossRegions(map: WorldMap): number[] {
  const regions = new Map<number, Set<number>>();
  for (let i = 1; i < map.provinces.length; i++) {
    const p = map.provinces[i];
    if (!p?.state || !p.region) continue;
    const set = regions.get(p.state);
    if (set) set.add(p.region);
    else regions.set(p.state, new Set([p.region]));
  }
  return [...regions]
    .filter(([, set]) => set.size > 1)
    .map(([state]) => state)
    .sort((a, b) => a - b);
}

export interface LandlessState {
  state: number;

  provinces: number[];
}

export function statesWithoutLand(map: WorldMap): LandlessState[] {
  const members = new Map<number, number[]>();
  const landed = new Set<number>();
  for (let i = 1; i < map.provinces.length; i++) {
    const p = map.provinces[i];
    if (!p?.state) continue;
    if (p.kind === "land") landed.add(p.state);
    const list = members.get(p.state);
    if (list) list.push(p.id);
    else members.set(p.state, [p.id]);
  }
  return [...members]
    .filter(([state]) => !landed.has(state))
    .map(([state, provinces]) => ({
      state,
      provinces: provinces.sort((a, b) => a - b),
    }))
    .sort((a, b) => a.state - b.state);
}
