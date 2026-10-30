import type { WorldMap } from "@/components/workbench/world-map/worldMapModel";

export type Overrides = ReadonlyMap<number, string>;

export type Neighbours = ReadonlyMap<number, ReadonlySet<number>>;

export function stateOwnerOf(map: WorldMap, index: number): string {
  const p = map.provinces[index];
  return p && p.kind === "land" && p.state
    ? (map.states.get(p.state)?.owner ?? "")
    : "";
}

export function ownerOf(
  map: WorldMap,
  overrides: Overrides,
  index: number,
): string {
  return overrides.get(index) ?? stateOwnerOf(map, index);
}

export function landOf(map: WorldMap, indexes: readonly number[]): number[] {
  const out = new Set<number>();
  for (const i of indexes) {
    const p = map.provinces[i];
    if (p && p.kind === "land" && p.state && map.states.has(p.state))
      out.add(i);
  }
  return [...out];
}

export function paintedMap(
  map: WorldMap,
  overrides: Overrides,
  colors: Record<string, [number, number, number]>,
): WorldMap {
  if (
    overrides.size === 0 &&
    Object.keys(colors).every((tag) => map.colors[tag])
  )
    return map;
  return {
    ...map,
    provinceOwners: overrides,
    colors: { ...colors, ...map.colors },
  };
}

export function connectedArea(
  map: WorldMap,
  overrides: Overrides,
  neighbours: Neighbours,
  start: number,
): number[] {
  if (landOf(map, [start]).length === 0) return [];
  const owner = ownerOf(map, overrides, start);
  const seen = new Set([start]);
  const queue = [start];
  while (queue.length) {
    const at = queue.pop()!;
    for (const next of neighbours.get(at) ?? []) {
      if (
        !seen.has(next) &&
        landOf(map, [next]).length &&
        ownerOf(map, overrides, next) === owner
      ) {
        seen.add(next);
        queue.push(next);
      }
    }
  }
  return [...seen];
}

export function countryProvinces(
  map: WorldMap,
  overrides: Overrides,
  owner: string,
): number[] {
  const out: number[] = [];
  for (let i = 1; i < map.provinces.length; i++) {
    const p = map.provinces[i];
    if (p.kind === "land" && p.state && ownerOf(map, overrides, i) === owner)
      out.push(i);
  }
  return out;
}

export function resolveErased(
  map: WorldMap,
  overrides: Overrides,
  neighbours: Neighbours,
): Map<number, string> {
  const out = new Map(overrides);
  let pending = [...out]
    .filter(([, tag]) => tag === "")
    .map(([i]) => i)
    .sort((a, b) => a - b);
  while (pending.length) {
    const settled: [number, string][] = [];
    for (const i of pending) {
      const votes = new Map<string, number>();
      for (const n of neighbours.get(i) ?? []) {
        if (!landOf(map, [n]).length) continue;
        const owner = out.get(n) ?? stateOwnerOf(map, n);
        if (owner) votes.set(owner, (votes.get(owner) ?? 0) + 1);
      }
      let best = "";
      let most = 0;
      for (const [tag, n] of [...votes].sort(([a], [b]) =>
        a.localeCompare(b),
      )) {
        if (n > most) {
          best = tag;
          most = n;
        }
      }
      if (best) settled.push([i, best]);
    }
    if (settled.length === 0) break;

    for (const [i, tag] of settled) out.set(i, tag);
    pending = pending.filter((i) => out.get(i) === "");
  }
  for (const i of pending) out.set(i, stateOwnerOf(map, i));
  return out;
}

export interface ProvinceChange {
  index: number;

  id: number;
  from: string;
  to: string;
}

export function pendingChanges(
  map: WorldMap,
  overrides: Overrides,
  neighbours: Neighbours | null,
): ProvinceChange[] {
  const final = neighbours
    ? resolveErased(map, overrides, neighbours)
    : overrides;
  const out: ProvinceChange[] = [];
  for (const [index, to] of final) {
    const from = stateOwnerOf(map, index);
    if (to && to !== from)
      out.push({ index, id: map.provinces[index].id, from, to });
  }
  return out.sort((a, b) => a.id - b.id);
}

export function balance(
  changes: readonly ProvinceChange[],
): [string, { gain: number; loss: number }][] {
  const out = new Map<string, { gain: number; loss: number }>();
  for (const c of changes) {
    const to = out.get(c.to) ?? { gain: 0, loss: 0 };
    to.gain++;
    out.set(c.to, to);
    const from = out.get(c.from) ?? { gain: 0, loss: 0 };
    from.loss++;
    out.set(c.from, from);
  }
  return [...out].sort(([, a], [, b]) => b.gain + b.loss - a.gain - a.loss);
}

export function statesTouched(
  map: WorldMap,
  changes: readonly ProvinceChange[],
): number {
  return new Set(changes.map((c) => map.provinces[c.index].state)).size;
}

export function restateProvinces(
  changes: readonly ProvinceChange[],
  tagOf: (key: string) => string,
): Record<string, string> {
  return Object.fromEntries(changes.map((c) => [String(c.id), tagOf(c.to)]));
}

export function freeTag(taken: ReadonlySet<string>): string {
  for (const letter of "ZQXW") {
    for (let n = 1; n < 100; n++) {
      const tag = `${letter}${String(n).padStart(2, "0")}`;
      if (!taken.has(tag)) return tag;
    }
  }
  throw new Error("no free tag");
}
