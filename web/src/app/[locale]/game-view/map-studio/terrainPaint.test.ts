import { describe, expect, it } from "vitest";
import {
  layerTable,
  parseWorldMap,
  terrainColour,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import {
  terrainArea,
  terrainChanges,
  terrainCounts,
  terrainLand,
  terrainOf,
  terrainRequest,
} from "./terrainPaint";

const payload: WorldMapPayload = {
  provinces: [
    [101, 0x010101, 0, 0, "plains", 1, 1, 1],
    [102, 0x020202, 0, 0, "plains", 1, 1, 1],
    [103, 0x030303, 0, 0, "forest", 1, 2, 1],
    [104, 0x040404, 0, 0, "plains", 1, 2, 1],
    [105, 0x050505, 0, 1, "plains", 1, 3, 1],
    [106, 0x060606, 1, 1, "ocean", 0, 0, 2],
    [107, 0x070707, 2, 0, "lakes", 1, 0, 1],
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
  [5, new Set([4, 6, 7])],
  [6, new Set([5])],
  [7, new Set([5])],
]);

describe("painting terrain", () => {
  it("takes land only, and reads terrains through the painting", () => {
    expect(terrainLand(map, [0, 1, 1, 5, 6, 7])).toEqual([1, 5]);
    const overrides = new Map([[2, "mountain"]]);
    expect(terrainOf(map, overrides, 2)).toBe("mountain");
    expect(terrainOf(map, overrides, 3)).toBe("forest");
  });

  it("fills the joined land of one terrain, not across another or the water", () => {
    expect(terrainArea(map, new Map(), neighbours, 1).sort()).toEqual([1, 2]);
    expect(terrainArea(map, new Map(), neighbours, 4).sort()).toEqual([4, 5]);

    expect(
      terrainArea(
        map,
        new Map([
          [4, "forest"],
          [5, "forest"],
        ]),
        neighbours,
        3,
      ).sort(),
    ).toEqual([3, 4, 5]);
    expect(terrainArea(map, new Map(), neighbours, 6)).toEqual([]);
  });

  it("saves only what differs from the map, by province id", () => {
    const overrides = new Map([
      [4, "mountain"],
      [2, "mountain"],
      [3, "forest"],
      [5, "desert"],
    ]);
    const changes = terrainChanges(map, overrides);
    expect(changes.map((c) => [c.id, c.from, c.to])).toEqual([
      [102, "plains", "mountain"],
      [104, "plains", "mountain"],
      [105, "plains", "desert"],
    ]);
    expect(terrainCounts(changes)).toEqual([
      ["mountain", 2],
      ["desert", 1],
    ]);
    expect(terrainRequest(changes)).toEqual({
      "102": "mountain",
      "104": "mountain",
      "105": "desert",
    });
  });

  it("colours the terrain layer by the painted terrain, borders between terrains", () => {
    const painted = { ...map, provinceTerrains: new Map([[2, "mountain"]]) };
    const { colors, groups } = layerTable(painted, "terrain", { province: 0 });
    expect(colors[1]).toBe(terrainColour("plains"));
    expect(colors[2]).toBe(terrainColour("mountain"));
    expect(colors[3]).toBe(terrainColour("forest"));
    expect(groups[1]).toBe(groups[4]);
    expect(groups[1]).not.toBe(groups[2]);
    expect(terrainColour("something_modded")).toBe(
      terrainColour("another_modded"),
    );
  });
});
