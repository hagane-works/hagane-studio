import { describe, expect, it } from "vitest";
import {
  parseWorldMap,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import { landArea, landSketch, strokePixels, unreachedSeas } from "./landPaint";
import { TERRAINS } from "./studioSketch";

describe("painting new land", () => {
  it("takes the pixels round a stroke, inside the map", () => {
    const one = strokePixels(10, 10, [5, 5], [5, 5], 1);
    expect(one.sort((a, b) => a - b)).toEqual([44, 45, 54, 55]);
    const edge = strokePixels(10, 10, [0, 0], [0, 0], 3);
    expect(edge.every((p) => p >= 0 && p < 100)).toBe(true);
    const line = strokePixels(100, 10, [10, 5], [60, 5], 2);
    expect(line).toContain(5 * 100 + 35);
    expect(line).not.toContain(5 * 100 + 70);
  });

  it("counts only painted land, not wiped pixels", () => {
    expect(
      landArea(
        new Map([
          [1, 1],
          [2, 0],
          [3, 4],
        ]),
      ),
    ).toBe(2);
  });

  it("sends the painted box with a margin in the legend's colours, sea round it", () => {
    const W = 50;
    const edits = new Map([
      [10 * W + 20, 1],
      [10 * W + 21, 4],
      [11 * W + 22, 0],
    ]);
    const sketch = landSketch(edits, W, 40)!;
    expect(sketch.rect).toEqual([18, 8, 6, 5]);
    const at = (x: number, y: number) => [
      ...sketch.rgba.slice(
        ((y - 8) * 6 + (x - 18)) * 4,
        ((y - 8) * 6 + (x - 18)) * 4 + 3,
      ),
    ];
    expect(at(20, 10)).toEqual([...TERRAINS[1].legend]);
    expect(at(21, 10)).toEqual([...TERRAINS[4].legend]);
    expect(at(22, 11)).toEqual([...TERRAINS[0].legend]);
    expect(landSketch(new Map([[5, 0]]), W, 40)).toBeNull();
  });

  it("keeps the box at least 4×4 and inside the map at its edges", () => {
    const sketch = landSketch(new Map([[0, 2]]), 50, 40)!;
    expect(sketch.rect).toEqual([0, 0, 4, 4]);
    const corner = landSketch(new Map([[39 * 50 + 49, 2]]), 50, 40)!;
    expect(corner.rect).toEqual([46, 36, 4, 4]);
  });
});

describe("making sea", () => {
  const payload: WorldMapPayload = {
    provinces: [
      [101, 0x010101, 0, 1, "plains", 1, 1, 1],
      [102, 0x020202, 0, 0, "plains", 1, 1, 1],
      [103, 0x030303, 0, 0, "plains", 1, 2, 1],
      [104, 0x040404, 0, 0, "plains", 1, 2, 1],
      [105, 0x050505, 1, 1, "ocean", 0, 0, 2],
      [106, 0x060606, 2, 0, "lakes", 0, 0, 1],
    ],
    states: [
      [1, "One", "AAA"],
      [2, "Two", "BBB"],
    ],
    regions: [
      [1, "Land"],
      [2, "Sea"],
    ],
    continents: ["Europe"],
    colors: {},
  };
  const map = parseWorldMap(payload);
  const neighbours = new Map<number, Set<number>>([
    [1, new Set([2, 5])],
    [2, new Set([1, 3])],
    [3, new Set([2, 4])],
    [4, new Set([3, 6])],
    [5, new Set([1])],
    [6, new Set([4])],
  ]);

  it("lets a group through when one of it touches the sea", () => {
    expect(unreachedSeas(map, neighbours, [1])).toEqual([]);
    expect(unreachedSeas(map, neighbours, [1, 2, 3])).toEqual([]);
  });

  it("names each group that reaches no sea by its lowest id; a lake does not count", () => {
    expect(unreachedSeas(map, neighbours, [4])).toEqual([104]);
    expect(unreachedSeas(map, neighbours, [3, 4])).toEqual([103]);
    expect(unreachedSeas(map, neighbours, [1, 3, 4])).toEqual([103]);
  });
});
