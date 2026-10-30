import { describe, expect, it } from "vitest";
import {
  ERASED,
  layerTable,
  pack,
  parseWorldMap,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import {
  balance,
  connectedArea,
  countryProvinces,
  freeTag,
  landOf,
  ownerOf,
  paintedMap,
  pendingChanges,
  resolveErased,
  restateProvinces,
  statesTouched,
} from "./ownerPaint";

const payload: WorldMapPayload = {
  provinces: [
    [101, 0x010101, 0, 0, "plains", 1, 1, 1],
    [102, 0x020202, 0, 0, "plains", 1, 1, 1],
    [103, 0x030303, 0, 0, "plains", 1, 2, 1],
    [104, 0x040404, 0, 0, "plains", 1, 2, 1],
    [105, 0x050505, 0, 1, "plains", 1, 3, 1],
    [106, 0x060606, 1, 1, "ocean", 0, 0, 2],
  ],
  states: [
    [1, "One", "AAA"],
    [2, "Two", "BBB"],
    [3, "Three", "AAA"],
  ],
  regions: [
    [1, "Land"],
    [2, "Sea"],
  ],
  continents: ["Europe"],
  colors: { AAA: [100, 0, 0], BBB: [0, 100, 0] },
};
const map = parseWorldMap(payload);
const neighbours = new Map<number, Set<number>>([
  [1, new Set([2])],
  [2, new Set([1, 3])],
  [3, new Set([2, 4])],
  [4, new Set([3, 5])],
  [5, new Set([4, 6])],
  [6, new Set([5])],
]);

describe("painting provinces", () => {
  it("reads owners through the painting, and draws a painted or erased province on its own", () => {
    const overrides = new Map([
      [2, "CCC"],
      [4, ""],
    ]);
    expect(ownerOf(map, overrides, 2)).toBe("CCC");
    expect(ownerOf(map, overrides, 1)).toBe("AAA");
    const shown = paintedMap(map, overrides, { CCC: [0, 0, 100] });
    const table = layerTable(shown, "political", { province: 0 });
    expect(table.colors[2]).toBe(pack(0, 0, 100));
    expect(table.colors[1]).toBe(pack(100, 0, 0));
    expect(table.colors[4]).toBe(ERASED);

    expect(table.groups[2]).not.toBe(table.groups[1]);
    expect(paintedMap(map, new Map(), {})).toBe(map);
  });

  it("finds land only, the joined area of one owner, and a whole country", () => {
    expect(landOf(map, [1, 6, 1]).sort()).toEqual([1]);
    expect(connectedArea(map, new Map(), neighbours, 1).sort()).toEqual([1, 2]);
    expect(
      connectedArea(map, new Map([[3, "AAA"]]), neighbours, 1).sort(),
    ).toEqual([1, 2, 3]);
    expect(connectedArea(map, new Map(), neighbours, 6)).toEqual([]);
    expect(countryProvinces(map, new Map(), "AAA").sort()).toEqual([1, 2, 5]);
  });

  it("gives erased land to the countries around it, from the edge in, and an unreached island back", () => {
    const given = resolveErased(
      map,
      new Map([
        [3, ""],
        [4, ""],
      ]),
      neighbours,
    );
    expect(given.get(3)).toBe("AAA");
    expect(given.get(4)).toBe("AAA");

    const all = resolveErased(
      map,
      new Map([1, 2, 3, 4, 5].map((i) => [i, ""] as [number, string])),
      neighbours,
    );
    expect([...all.values()]).toEqual(["AAA", "AAA", "BBB", "BBB", "AAA"]);
  });

  it("saves only real changes, by province id, new countries by the tag they got", () => {
    const overrides = new Map([
      [1, "AAA"],
      [3, "NEW1"],
      [4, ""],
    ]);
    const changes = pendingChanges(map, overrides, neighbours);
    expect(changes).toEqual([
      { index: 3, id: 103, from: "BBB", to: "NEW1" },
      { index: 4, id: 104, from: "BBB", to: "AAA" },
    ]);
    expect(statesTouched(map, changes)).toBe(1);
    expect(
      restateProvinces(changes, (key) => (key === "NEW1" ? "Z01" : key)),
    ).toEqual({ "103": "Z01", "104": "AAA" });
    expect(balance(changes)).toEqual([
      ["BBB", { gain: 0, loss: 2 }],
      ["NEW1", { gain: 1, loss: 0 }],
      ["AAA", { gain: 1, loss: 0 }],
    ]);
  });

  it("finds a free tag for a new country", () => {
    expect(freeTag(new Set())).toBe("Z01");
    expect(freeTag(new Set(["Z01", "Z02"]))).toBe("Z03");
  });
});
