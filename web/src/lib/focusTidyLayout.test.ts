import { describe, expect, it } from "vitest";
import { bfsLayoutFocusNodes } from "./focusTreeGraph";

type Spec = {
  id: number;
  focus_id: string;
  prerequisites?: string[][];
  relative_position_id?: string | null;
};

const mk = (specs: Spec[]): any[] =>
  specs.map((s) => ({
    id: s.id,
    focus_id: s.focus_id,
    prerequisites: s.prerequisites ?? [],
    relative_position_id: s.relative_position_id ?? null,
    x: 0,
    y: 0,
  }));

function layout(specs: Spec[]) {
  const out = bfsLayoutFocusNodes(mk(specs));
  const byFocus = new Map<string, { x: number; y: number }>();
  for (const s of specs) {
    const p = out.get(s.id);
    if (p) byFocus.set(s.focus_id, p);
  }
  return byFocus;
}

describe("bfsLayoutFocusNodes · tidy tree", () => {
  it("REGRESSION · a child is centred under its parent, not packed at col 0", () => {
    const p = layout([
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "a", prerequisites: [["root"]] },
      { id: 3, focus_id: "b", prerequisites: [["root"]] },
    ]);
    const root = p.get("root")!;
    const a = p.get("a")!;
    const b = p.get("b")!;
    expect(root.y).toBe(0);
    expect(a.y).toBe(1);
    expect(b.y).toBe(1);
    expect(root.x).toBe((a.x + b.x) / 2);
    expect(a.x).not.toBe(b.x);
  });

  it("sibling branches occupy disjoint horizontal bands", () => {
    const p = layout([
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "L", prerequisites: [["root"]] },
      { id: 3, focus_id: "R", prerequisites: [["root"]] },
      { id: 4, focus_id: "L1", prerequisites: [["L"]] },
      { id: 5, focus_id: "L2", prerequisites: [["L"]] },
      { id: 6, focus_id: "R1", prerequisites: [["R"]] },
      { id: 7, focus_id: "R2", prerequisites: [["R"]] },
    ]);
    const leftMax = Math.max(p.get("L1")!.x, p.get("L2")!.x, p.get("L")!.x);
    const rightMin = Math.min(p.get("R1")!.x, p.get("R2")!.x, p.get("R")!.x);
    expect(leftMax).toBeLessThan(rightMin);

    expect(p.get("L")!.x).toBe((p.get("L1")!.x + p.get("L2")!.x) / 2);
    expect(p.get("R")!.x).toBe((p.get("R1")!.x + p.get("R2")!.x) / 2);
  });

  it("no two emitted nodes share a cell", () => {
    const p = layout([
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "a", prerequisites: [["root"]] },
      { id: 3, focus_id: "b", prerequisites: [["root"]] },
      { id: 4, focus_id: "c", prerequisites: [["root"]] },
      { id: 5, focus_id: "d", prerequisites: [["a"]] },
      { id: 6, focus_id: "e", prerequisites: [["a"]] },
      { id: 7, focus_id: "f", prerequisites: [["b"]] },
      { id: 8, focus_id: "deep", prerequisites: [["c"], ["f"]] },
    ]);
    const seen = new Set<string>();
    for (const [, v] of p) {
      const key = `${v.x}:${v.y}`;
      expect(seen.has(key)).toBe(false);
      seen.add(key);
    }
  });

  it("a focus always sits strictly below every one of its prerequisites", () => {
    const p = layout([
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "mid", prerequisites: [["root"]] },
      { id: 3, focus_id: "late", prerequisites: [["mid"]] },

      { id: 4, focus_id: "join", prerequisites: [["root"], ["late"]] },
    ]);
    expect(p.get("join")!.y).toBeGreaterThan(p.get("root")!.y);
    expect(p.get("join")!.y).toBeGreaterThan(p.get("late")!.y);
  });

  it("anchored nodes are placed relative to their anchor, and push nobody", () => {
    const specs = [
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "a", prerequisites: [["root"]] },
      { id: 3, focus_id: "anchored", relative_position_id: "a" },
    ];
    const p = layout(specs);
    expect(p.has("anchored")).toBe(true);

    const without = layout(specs.slice(0, 2));
    expect(p.get("root")).toEqual(without.get("root"));
    expect(p.get("a")).toEqual(without.get("a"));
  });

  it("leftmost node is normalised to x = 0 and layout is deterministic", () => {
    const specs: Spec[] = [
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "a", prerequisites: [["root"]] },
      { id: 3, focus_id: "b", prerequisites: [["root"]] },
      { id: 4, focus_id: "c", prerequisites: [["b"]] },
    ];
    const first = layout(specs);
    const second = layout(specs);
    expect(Math.min(...[...first.values()].map((v) => v.x))).toBe(0);
    expect([...first.entries()]).toEqual([...second.entries()]);
  });

  it("PARITY FIXTURE · exact coords, mirrored by the backend smoke", () => {
    const p = layout([
      { id: 1, focus_id: "root" },
      { id: 2, focus_id: "L", prerequisites: [["root"]] },
      { id: 3, focus_id: "R", prerequisites: [["root"]] },
      { id: 4, focus_id: "L1", prerequisites: [["L"]] },
      { id: 5, focus_id: "L2", prerequisites: [["L"]] },
      { id: 6, focus_id: "R1", prerequisites: [["R"]] },
      { id: 7, focus_id: "R2", prerequisites: [["R"]] },
    ]);
    expect(Object.fromEntries(p)).toEqual({
      root: { x: 6, y: 0 },
      L: { x: 2, y: 1 },
      R: { x: 10, y: 1 },
      L1: { x: 0, y: 2 },
      L2: { x: 4, y: 2 },
      R1: { x: 8, y: 2 },
      R2: { x: 12, y: 2 },
    });
  });

  it("empty input and a lone root are handled", () => {
    expect(bfsLayoutFocusNodes([])).toEqual(new Map());
    const solo = layout([{ id: 1, focus_id: "only" }]);
    expect(solo.get("only")).toEqual({ x: 0, y: 0 });
  });

  it("a prerequisite cycle does not hang or throw", () => {
    const p = layout([
      { id: 1, focus_id: "x", prerequisites: [["y"]] },
      { id: 2, focus_id: "y", prerequisites: [["x"]] },
    ]);

    expect(p.size).toBe(2);
  });
});
