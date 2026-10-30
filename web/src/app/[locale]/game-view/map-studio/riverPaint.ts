export const RIVER_SOURCE = 0;
export const RIVER_JOIN = 1;
export const RIVER_SPLIT = 2;
export const RIVER_NONE = 255;

export const RIVER_WIDTHS = [3, 4, 6, 7, 9, 10, 11] as const;

export type RiverSize = "stream" | "river" | "large";
export const RIVER_SIZES: Record<RiverSize, number> = {
  stream: 4,
  river: 7,
  large: 11,
};

export const isRiver = (v: number | undefined): boolean =>
  v !== undefined && v >= 0 && v <= 11;
const isWidth = (v: number | undefined) => v !== undefined && v >= 3 && v <= 11;

export interface RiverPayload {
  width: number;
  height: number;
  at: number[];
  values: number[];
}

export interface RiverMap {
  width: number;
  height: number;
  base: ReadonlyMap<number, number>;
}

export function parseRivers(payload: RiverPayload): RiverMap {
  const base = new Map<number, number>();
  let at = 0;
  payload.at.forEach((step, i) => {
    at += step;
    base.set(at, payload.values[i]);
  });
  return { width: payload.width, height: payload.height, base };
}

export type RiverEdits = ReadonlyMap<number, number>;

export function riverAt(
  rivers: RiverMap,
  edits: RiverEdits,
  p: number,
): number {
  return edits.get(p) ?? rivers.base.get(p) ?? RIVER_NONE;
}

export type Ground = (p: number) => boolean;

function sides(p: number, width: number, height: number): number[] {
  const x = p % width;
  const y = (p - x) / width;
  const out: number[] = [];
  if (x > 0) out.push(p - 1);
  if (x + 1 < width) out.push(p + 1);
  if (y > 0) out.push(p - width);
  if (y + 1 < height) out.push(p + width);
  return out;
}

export interface RiverDraft {
  path: number[];

  end: "open" | "sea" | "joined" | "blocked";

  joins?: number;
}

export function startRiver(
  rivers: RiverMap,
  edits: RiverEdits,
  ground: Ground,
  p: number,
): RiverDraft | null {
  if (
    p < 0 ||
    p >= rivers.width * rivers.height ||
    !ground(p) ||
    isRiver(riverAt(rivers, edits, p))
  )
    return null;
  if (
    sides(p, rivers.width, rivers.height).some((q) =>
      isRiver(riverAt(rivers, edits, q)),
    )
  )
    return null;
  return { path: [p], end: "open" };
}

export function extendRiver(
  draft: RiverDraft,
  rivers: RiverMap,
  edits: RiverEdits,
  ground: Ground,
  tx: number,
  ty: number,
): boolean {
  const { width, height } = rivers;
  if (draft.end !== "open") return false;
  const onPath = new Set(draft.path);
  let last = draft.path[draft.path.length - 1];
  let grew = false;
  tx = Math.min(width - 1, Math.max(0, Math.floor(tx)));
  ty = Math.min(height - 1, Math.max(0, Math.floor(ty)));
  for (let guard = 0; guard < 20000; guard++) {
    const x = last % width;
    const y = (last - x) / width;
    if (x === tx && y === ty) break;
    const dx = tx - x;
    const dy = ty - y;
    const q =
      Math.abs(dx) >= Math.abs(dy)
        ? last + Math.sign(dx)
        : last + Math.sign(dy) * width;
    if (onPath.has(q)) break;
    if (!ground(q)) {
      draft.end = "sea";
      break;
    }
    const around = sides(q, width, height);
    if (isRiver(riverAt(rivers, edits, q))) {
      draft.end = "blocked";
      break;
    }
    if (around.some((n) => n !== last && onPath.has(n))) break;
    const touching = around.filter((n) => isRiver(riverAt(rivers, edits, n)));
    if (touching.length) {
      const into = touching[0];
      const meeting = sides(into, width, height).filter((n) =>
        isRiver(riverAt(rivers, edits, n)),
      ).length;
      if (
        touching.length === 1 &&
        draft.path.length >= 2 &&
        isWidth(riverAt(rivers, edits, into)) &&
        meeting <= 2
      ) {
        draft.path.push(q);
        draft.end = "joined";
        draft.joins = riverAt(rivers, edits, into);
        return true;
      }
      draft.end = "blocked";
      break;
    }
    draft.path.push(q);
    onPath.add(q);
    last = q;
    grew = true;
  }
  return grew;
}

export function riverValues(
  draft: RiverDraft,
  widest: number,
  preview = false,
): Map<number, number> {
  const out = new Map<number, number>();
  const { path } = draft;
  if (path.length < (preview ? 1 : 3)) return out;
  const cap =
    draft.end === "joined" && draft.joins
      ? Math.min(widest, draft.joins)
      : widest;
  const widths = RIVER_WIDTHS.filter((w) => w <= cap);
  const ramp = (i: number, of: number) =>
    widths[
      Math.min(
        widths.length - 1,
        Math.floor((i * widths.length) / Math.max(1, of)),
      )
    ];
  if (draft.end === "joined") {
    const run = path.length - 1;
    for (let i = 0; i < run; i++) out.set(path[i], ramp(i, run));
    out.set(path[run], RIVER_JOIN);
  } else {
    out.set(path[0], RIVER_SOURCE);
    const run = path.length - 1;
    for (let i = 1; i < path.length; i++) out.set(path[i], ramp(i - 1, run));
  }
  return out;
}

export interface RiverTree {
  branchOf: Map<number, number>;
  pixels: number[][];
  parent: number[];
}

export function riverTree(rivers: RiverMap, edits: RiverEdits): RiverTree {
  const { width, height } = rivers;
  const shown = new Map<number, number>();
  for (const [p, v] of rivers.base) if (!edits.has(p)) shown.set(p, v);
  for (const [p, v] of edits) if (isRiver(v)) shown.set(p, v);
  const tree: RiverTree = { branchOf: new Map(), pixels: [], parent: [] };
  const children: [number, number][] = [];
  const walk = (start: number, parent: number) => {
    const id = tree.pixels.length;
    const pixels: number[] = [];
    tree.pixels.push(pixels);
    tree.parent.push(parent);
    tree.branchOf.set(start, id);
    const stack = [start];
    while (stack.length) {
      const p = stack.pop()!;
      pixels.push(p);
      for (const n of sides(p, width, height)) {
        const v = shown.get(n);
        if (v === undefined || tree.branchOf.has(n)) continue;
        if (v === RIVER_JOIN || v === RIVER_SPLIT) {
          tree.branchOf.set(n, -1);
          children.push([n, id]);
          continue;
        }
        tree.branchOf.set(n, id);
        stack.push(n);
      }
    }
  };

  const drain = () => {
    for (let next = children.shift(); next; next = children.shift())
      walk(next[0], next[1]);
  };
  for (const [p, v] of shown)
    if (v === RIVER_SOURCE && !tree.branchOf.has(p)) walk(p, -1);
  drain();
  for (const p of shown.keys()) {
    if (tree.branchOf.has(p)) continue;
    walk(p, -1);
    drain();
  }
  return tree;
}

export function withTributaries(
  tree: RiverTree,
  branches: Iterable<number>,
): Set<number> {
  const kids = new Map<number, number[]>();
  tree.parent.forEach((parent, id) => {
    if (parent >= 0) kids.set(parent, [...(kids.get(parent) ?? []), id]);
  });
  const out = new Set<number>();
  const queue = [...branches];
  while (queue.length) {
    const id = queue.pop()!;
    if (out.has(id)) continue;
    out.add(id);
    queue.push(...(kids.get(id) ?? []));
  }
  return out;
}

export function riversNear(
  tree: RiverTree,
  width: number,
  x0: number,
  y0: number,
  x1: number,
  y1: number,
  radius: number,
): Set<number> {
  const out = new Set<number>();
  const dx = x1 - x0;
  const dy = y1 - y0;
  const length2 = dx * dx + dy * dy;
  const r2 = radius * radius;
  for (const [p, id] of tree.branchOf) {
    if (id < 0 || out.has(id)) continue;
    const px = (p % width) + 0.5;
    const py = Math.floor(p / width) + 0.5;
    const t = length2
      ? Math.max(0, Math.min(1, ((px - x0) * dx + (py - y0) * dy) / length2))
      : 0;
    const ex = x0 + dx * t - px;
    const ey = y0 + dy * t - py;
    if (ex * ex + ey * ey <= r2) out.add(id);
  }
  return out;
}

export function riverChanges(
  rivers: RiverMap,
  edits: RiverEdits,
): [number, number][] {
  const out: [number, number][] = [];
  for (const [p, v] of edits)
    if (v !== (rivers.base.get(p) ?? RIVER_NONE)) out.push([p, v]);
  return out.sort((a, b) => a[0] - b[0]);
}

export function riverCounts(
  rivers: RiverMap,
  changes: readonly [number, number][],
): { drawn: number; erased: number } {
  let drawn = 0;
  let erased = 0;
  for (const [p, v] of changes) {
    const was = rivers.base.get(p);
    if ((v === RIVER_SOURCE || v === RIVER_JOIN) && !isRiver(was)) drawn++;
    if (
      v === RIVER_NONE &&
      (was === RIVER_SOURCE || was === RIVER_JOIN || was === RIVER_SPLIT)
    )
      erased++;
  }
  return { drawn, erased };
}

export function riverColour(v: number): [number, number, number] {
  const width = isWidth(v) ? v : 3;
  const t = (width - 3) / 8;
  return [
    Math.round(120 - 90 * t),
    Math.round(186 - 100 * t),
    Math.round(240 - 40 * t),
  ];
}
