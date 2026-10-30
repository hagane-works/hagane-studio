import type { WorldMap } from "@/components/workbench/world-map/worldMapModel";
import type { Neighbours } from "./ownerPaint";

export const TERRAIN_BRUSHES = [
  "plains",
  "forest",
  "hills",
  "mountain",
  "desert",
  "marsh",
  "jungle",
  "urban",
] as const;
export type TerrainBrush = (typeof TERRAIN_BRUSHES)[number];

export type TerrainOverrides = ReadonlyMap<number, string>;

export function terrainLand(
  map: WorldMap,
  indexes: readonly number[],
): number[] {
  const out = new Set<number>();
  for (const i of indexes) {
    const p = map.provinces[i];
    if (i > 0 && p && p.kind === "land") out.add(i);
  }
  return [...out];
}

export function terrainOf(
  map: WorldMap,
  overrides: TerrainOverrides,
  index: number,
): string {
  return overrides.get(index) ?? map.provinces[index]?.terrain ?? "";
}

export function terrainArea(
  map: WorldMap,
  overrides: TerrainOverrides,
  neighbours: Neighbours,
  start: number,
): number[] {
  if (terrainLand(map, [start]).length === 0) return [];
  const terrain = terrainOf(map, overrides, start);
  const seen = new Set([start]);
  const queue = [start];
  while (queue.length) {
    const at = queue.pop()!;
    for (const next of neighbours.get(at) ?? []) {
      if (
        !seen.has(next) &&
        terrainLand(map, [next]).length &&
        terrainOf(map, overrides, next) === terrain
      ) {
        seen.add(next);
        queue.push(next);
      }
    }
  }
  return [...seen];
}

export interface TerrainChange {
  index: number;
  id: number;
  from: string;
  to: string;
}

export function terrainChanges(
  map: WorldMap,
  overrides: TerrainOverrides,
): TerrainChange[] {
  const out: TerrainChange[] = [];
  for (const [index, to] of overrides) {
    const from = map.provinces[index]?.terrain ?? "";
    if (to && to !== from)
      out.push({ index, id: map.provinces[index].id, from, to });
  }
  return out.sort((a, b) => a.id - b.id);
}

export function terrainCounts(
  changes: readonly TerrainChange[],
): [string, number][] {
  const out = new Map<string, number>();
  for (const c of changes) out.set(c.to, (out.get(c.to) ?? 0) + 1);
  return [...out].sort(([a, x], [b, y]) => y - x || a.localeCompare(b));
}

export function terrainRequest(
  changes: readonly TerrainChange[],
): Record<string, string> {
  return Object.fromEntries(changes.map((c) => [String(c.id), c.to]));
}
