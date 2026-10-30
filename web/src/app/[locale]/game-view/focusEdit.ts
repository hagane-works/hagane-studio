import type { FocusNode as DbFocusNode } from "@/types";

const PURE_REFERENCE = /^\s*\$[A-Za-z0-9_.-]+\$\s*$/;

export function isPureReference(value: string | undefined): boolean {
  return !!value && PURE_REFERENCE.test(value);
}

export function draftText(
  locs: Record<string, string> | undefined,
  flat: string | undefined,
  shown: string,
): string {
  const own = locs?.en || flat || "";
  return own && !isPureReference(own) ? own : shown;
}

export interface FocusDraft {
  name: string;
  desc: string;
  cost: number;

  script?: FocusScript;
}

export const FOCUS_SCRIPT_FIELDS = [
  "completion_reward",
  "available",
  "bypass",
  "ai_will_do",
] as const;
export type FocusScriptField = (typeof FOCUS_SCRIPT_FIELDS)[number];
export type FocusScript = Record<FocusScriptField, string>;

export function focusScriptOf(node: DbFocusNode): FocusScript {
  const out = {} as FocusScript;
  for (const field of FOCUS_SCRIPT_FIELDS)
    out[field] = String(
      (node as unknown as Record<string, unknown>)[field] ?? "",
    );
  return out;
}

export function focusPatch(
  node: DbFocusNode,
  draft: FocusDraft,
  start: FocusDraft,
): Partial<DbFocusNode> {
  const patch: Partial<DbFocusNode> = {};
  if (draft.name !== start.name)
    patch.display_name_locs = {
      ...(node.display_name_locs ?? {}),
      en: draft.name,
    };
  if (draft.desc !== start.desc)
    patch.description_locs = {
      ...(node.description_locs ?? {}),
      en: draft.desc,
    };
  if (draft.cost !== start.cost) patch.cost = draft.cost;
  if (draft.script && start.script) {
    for (const field of FOCUS_SCRIPT_FIELDS) {
      if (draft.script[field] !== start.script[field]) {
        (patch as Record<string, unknown>)[field] = draft.script[field];
      }
    }
  }
  return patch;
}

export function focusChanged(draft: FocusDraft, start: FocusDraft): boolean {
  if (
    draft.name !== start.name ||
    draft.desc !== start.desc ||
    draft.cost !== start.cost
  )
    return true;
  return FOCUS_SCRIPT_FIELDS.some(
    (field) => (draft.script?.[field] ?? "") !== (start.script?.[field] ?? ""),
  );
}

export function relativeDescendants(
  focuses: readonly { id: string; relative_to?: string }[],
  id: string,
): Set<string> {
  const children = new Map<string, string[]>();
  for (const focus of focuses) {
    if (!focus.relative_to) continue;
    const list = children.get(focus.relative_to) ?? [];
    list.push(focus.id);
    children.set(focus.relative_to, list);
  }
  const out = new Set<string>();
  const queue = [...(children.get(id) ?? [])];
  while (queue.length) {
    const next = queue.shift() as string;
    if (out.has(next) || next === id) continue;
    out.add(next);
    queue.push(...(children.get(next) ?? []));
  }
  return out;
}

export function storedPosition(
  focuses: readonly { id: string; x: number; y: number }[],
  focus: { relative_to?: string },
  x: number,
  y: number,
): { x: number; y: number } {
  const anchor = focus.relative_to
    ? focuses.find((f) => f.id === focus.relative_to)
    : undefined;
  return anchor ? { x: x - anchor.x, y: y - anchor.y } : { x, y };
}

export function cellAt(
  px: number,
  py: number,
  min: { x: number; y: number },
  spacing: { x: number; y: number },
): { x: number; y: number } {
  return {
    x: Math.round((px - spacing.x / 2) / spacing.x) + min.x,
    y: Math.floor(py / spacing.y) + min.y,
  };
}

export function freeCell(
  focuses: readonly { x: number; y: number }[],
  x: number,
  y: number,
): { x: number; y: number } {
  const taken = new Set(focuses.map((f) => `${f.x},${f.y}`));
  const clear = (cx: number, cy: number) =>
    !taken.has(`${cx - 1},${cy}`) &&
    !taken.has(`${cx},${cy}`) &&
    !taken.has(`${cx + 1},${cy}`);
  const ROW_COST = 3;
  const REACH = 8;
  const before = (a: readonly number[], b: readonly number[]) => {
    for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return a[i] < b[i];
    return false;
  };
  let best: { x: number; y: number; rank: number[] } | null = null;
  for (let row = 0; row <= focuses.length; row++) {
    if (best && row * ROW_COST > best.rank[0]) break;
    for (let dx = -REACH; dx <= REACH; dx++) {
      if (!clear(x + dx, y + row)) continue;
      const rank = [
        Math.abs(dx) + row * ROW_COST,
        Math.abs(dx),
        dx < 0 ? 1 : 0,
      ];
      if (!best || before(rank, best.rank))
        best = { x: x + dx, y: y + row, rank };
    }
  }
  return best ? { x: best.x, y: best.y } : { x, y };
}

type Linked = {
  id: string;
  y?: number;
  x?: number;
  prereq: string[][] | null;
  mutex: string[] | null;
};

export function requires(
  focuses: readonly Linked[],
  from: string,
  target: string,
): boolean {
  const byId = new Map(focuses.map((f) => [f.id, f]));
  const seen = new Set<string>();
  const queue = [from];
  while (queue.length) {
    const id = queue.shift() as string;
    if (seen.has(id)) continue;
    seen.add(id);
    for (const group of byId.get(id)?.prereq ?? []) {
      for (const parent of group) {
        if (parent === target) return true;
        queue.push(parent);
      }
    }
  }
  return false;
}

export type FocusLinkProblem =
  "self" | "alreadyMutex" | "alreadyPrereq" | "loop";

export function linkProblem(
  focuses: readonly Linked[],
  focus: Linked,
  other: string,
  kind: "prereq" | "mutex",
): FocusLinkProblem | null {
  if (other === focus.id) return "self";
  if (kind === "mutex") {
    return (focus.mutex ?? []).includes(other) ? "alreadyMutex" : null;
  }
  if ((focus.prereq ?? []).some((group) => group.includes(other)))
    return "alreadyPrereq";
  if (requires(focuses, other, focus.id)) return "loop";
  return null;
}

export function withPrereq(
  groups: readonly string[][] | null,
  id: string,
  into?: number,
): string[][] {
  const next = (groups ?? []).map((group) => [...group]);
  if (into !== undefined && next[into]) next[into].push(id);
  else next.push([id]);
  return next;
}

export function withoutPrereq(
  groups: readonly string[][] | null,
  from: number,
  id: string,
): string[][] {
  return (groups ?? [])
    .map((group, i) =>
      i === from ? group.filter((p) => p !== id) : [...group],
    )
    .filter((group) => group.length > 0);
}

export function exclusivePairs<T extends Linked & { x: number; y: number }>(
  focuses: readonly T[],
): [T, T][] {
  const byId = new Map(focuses.map((f) => [f.id, f]));
  const pairs: [T, T][] = [];
  const done = new Set<string>();
  for (const focus of focuses) {
    for (const otherId of focus.mutex ?? []) {
      const other = byId.get(otherId);
      if (!other || other.id === focus.id || other.y !== focus.y) continue;
      const key = [focus.id, other.id].sort().join("\u0000");
      if (done.has(key)) continue;
      done.add(key);
      pairs.push(focus.x <= other.x ? [focus, other] : [other, focus]);
    }
  }
  return pairs;
}
