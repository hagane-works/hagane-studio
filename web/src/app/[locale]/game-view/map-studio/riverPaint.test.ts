import { describe, expect, it } from "vitest";
import {
  extendRiver,
  isRiver,
  parseRivers,
  RIVER_JOIN,
  RIVER_NONE,
  RIVER_SOURCE,
  riverAt,
  riverChanges,
  riverCounts,
  riversNear,
  riverTree,
  riverValues,
  startRiver,
  withTributaries,
  type RiverDraft,
  type RiverMap,
} from "./riverPaint";

const W = 20;
const H = 10;
const at = (x: number, y: number) => y * W + x;
const ground = (p: number) => p % W < 17;
const mainRiver = new Map<number, number>();
for (let x = 2; x <= 16; x++)
  mainRiver.set(at(x, 5), x === 2 ? RIVER_SOURCE : 6);
const steps = [...mainRiver.keys()].sort((a, b) => a - b);
const rivers: RiverMap = parseRivers({
  width: W,
  height: H,
  at: steps.map((p, i) => p - (steps[i - 1] ?? 0)),
  values: steps.map((p) => mainRiver.get(p)!),
});

function faults(edits: Map<number, number>): number {
  const v = (p: number) => riverAt(rivers, edits, p);
  let n = 0;
  for (let p = 0; p < W * H; p++) {
    if (!isRiver(v(p))) continue;
    const x = p % W;
    const y = Math.floor(p / W);
    const sides = [
      x > 0 ? p - 1 : -1,
      x + 1 < W ? p + 1 : -1,
      y > 0 ? p - W : -1,
      y + 1 < H ? p + W : -1,
    ].filter((q) => q >= 0);
    const deg = sides.filter((q) => isRiver(v(q))).length;
    if (v(p) === RIVER_SOURCE && deg !== 1) n++;
    else if ((v(p) === 1 || v(p) === 2) && deg !== 2) n++;
    else if (deg > 3) n++;
    else if (deg === 3 && !sides.some((q) => v(q) === 1 || v(q) === 2)) n++;
    if (
      x + 1 < W &&
      y + 1 < H &&
      isRiver(v(p + 1)) &&
      isRiver(v(p + W)) &&
      isRiver(v(p + W + 1))
    )
      n++;
  }
  return n;
}

function draw(
  edits: Map<number, number>,
  from: [number, number],
  to: [number, number],
): RiverDraft | null {
  const draft = startRiver(rivers, edits, ground, at(...from));
  if (draft) extendRiver(draft, rivers, edits, ground, to[0], to[1]);
  return draft;
}

describe("drawing rivers", () => {
  it("reads the rivers sent as steps", () => {
    expect(rivers.base.get(at(2, 5))).toBe(RIVER_SOURCE);
    expect(rivers.base.get(at(16, 5))).toBe(6);
    expect(rivers.base.size).toBe(15);
  });

  it("starts only on land with no river on it or beside it", () => {
    const none = new Map<number, number>();
    expect(startRiver(rivers, none, ground, at(5, 5))).toBeNull();
    expect(startRiver(rivers, none, ground, at(5, 4))).toBeNull();
    expect(startRiver(rivers, none, ground, at(18, 1))).toBeNull();
    expect(startRiver(rivers, none, ground, at(5, 2))?.path).toEqual([
      at(5, 2),
    ]);
  });

  it("runs to the sea: a source at the top, widening down to the mouth", () => {
    const edits = new Map<number, number>();
    const draft = draw(edits, [3, 1], [19, 1])!;
    expect(draft.end).toBe("sea");
    expect(draft.path.at(-1)).toBe(at(16, 1));
    const values = riverValues(draft, 7);
    expect(values.get(at(3, 1))).toBe(RIVER_SOURCE);
    expect(values.get(at(4, 1))).toBe(3);
    expect(values.get(at(16, 1))).toBe(7);
    const widths = draft.path.slice(1).map((p) => values.get(p)!);
    expect(widths).toEqual([...widths].sort((a, b) => a - b));
    for (const [p, v] of values) edits.set(p, v);
    expect(faults(edits)).toBe(0);
  });

  it("joins a river it runs into side by side: a tributary, no wider than that river", () => {
    const edits = new Map<number, number>();
    const draft = draw(edits, [8, 1], [8, 9])!;
    expect(draft.end).toBe("joined");
    expect(draft.path).toEqual([at(8, 1), at(8, 2), at(8, 3), at(8, 4)]);
    const values = riverValues(draft, 11);
    expect(values.get(at(8, 4))).toBe(RIVER_JOIN);
    expect(values.get(at(8, 1))).toBe(3);
    expect(
      Math.max(...[1, 2, 3].map((y) => values.get(at(8, y))!)),
    ).toBeLessThanOrEqual(6);
    for (const [p, v] of values) edits.set(p, v);
    expect(faults(edits)).toBe(0);

    expect(startRiver(rivers, edits, ground, at(9, 1))).toBeNull();
    const next = draw(edits, [8, 9], [8, 0])!;
    expect(next.end).toBe("blocked");
    expect(next.path.at(-1)).toBe(at(8, 7));
  });

  it("stops short of a river's source and never runs beside itself", () => {
    const edits = new Map<number, number>();
    const draft = draw(edits, [2, 1], [2, 9])!;
    expect(draft.end).toBe("blocked");
    expect(draft.path.at(-1)).toBe(at(2, 3));

    const turn = draw(edits, [5, 1], [10, 1])!;
    extendRiver(turn, rivers, edits, ground, 10, 2);
    extendRiver(turn, rivers, edits, ground, 6, 2);
    expect(turn.path.at(-1)).toBe(at(10, 2));
    for (const [p, v] of riverValues(turn, 7)) edits.set(p, v);
    expect(faults(edits)).toBe(0);
  });

  it("leaves out a river under three pixels, unless previewing", () => {
    const draft: RiverDraft = { path: [at(5, 1), at(6, 1)], end: "open" };
    expect(riverValues(draft, 7).size).toBe(0);
    expect(riverValues(draft, 7, true).size).toBe(2);
  });
});

describe("erasing rivers", () => {
  it("tells rivers apart, and takes a river away with the ones flowing into it", () => {
    const edits = new Map<number, number>();
    const tributary = draw(edits, [8, 1], [8, 9])!;
    for (const [p, v] of riverValues(tributary, 7)) edits.set(p, v);
    const tree = riverTree(rivers, edits);
    const main = tree.branchOf.get(at(10, 5))!;
    const side = tree.branchOf.get(at(8, 2))!;
    expect(main).not.toBe(side);
    expect(tree.branchOf.get(at(8, 4))).toBe(side);
    expect(tree.parent[side]).toBe(main);
    expect([...withTributaries(tree, [side])]).toEqual([side]);
    expect([...withTributaries(tree, [main])].sort()).toEqual(
      [main, side].sort(),
    );

    expect([...riversNear(tree, W, 8.5, 1.5, 8.5, 2.5, 1)]).toEqual([side]);
    const gone = withTributaries(
      tree,
      riversNear(tree, W, 14.5, 5.5, 14.5, 5.5, 1),
    );
    for (const id of gone)
      for (const p of tree.pixels[id]) edits.set(p, RIVER_NONE);
    expect(
      [...Array(W * H).keys()].filter((p) =>
        isRiver(riverAt(rivers, edits, p)),
      ),
    ).toEqual([]);
    expect(faults(edits)).toBe(0);
  });

  it("saves only what differs, and counts rivers by their sources and joins", () => {
    const edits = new Map<number, number>([
      [at(3, 1), RIVER_SOURCE],
      [at(4, 1), 3],
      [at(5, 1), 3],
      [at(2, 5), RIVER_NONE],
      [at(3, 5), RIVER_NONE],
      [at(9, 5), 6],
      [at(1, 1), RIVER_NONE],
    ]);
    const changes = riverChanges(rivers, edits);
    expect(changes).toEqual([
      [at(3, 1), RIVER_SOURCE],
      [at(4, 1), 3],
      [at(5, 1), 3],
      [at(2, 5), RIVER_NONE],
      [at(3, 5), RIVER_NONE],
    ]);
    expect(riverCounts(rivers, changes)).toEqual({ drawn: 1, erased: 1 });
  });
});
