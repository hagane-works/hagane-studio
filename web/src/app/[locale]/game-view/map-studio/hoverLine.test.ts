import { describe, expect, it } from "vitest";
import {
  parseWorldMap,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import { describeHover, type HoverWords } from "./hoverLine";

const payload: WorldMapPayload = {
  provinces: [
    [101, 0x010101, 0, 0, "plains", 1, 1, 1],
    [102, 0x020202, 0, 0, "forest", 1, 1, 1],
    [103, 0x030303, 1, 1, "ocean", 0, 0, 2],
  ],
  states: [[1, "One", "AAA"]],
  regions: [
    [1, "Land"],
    [2, "Sea"],
  ],
  continents: ["Europe"],
  colors: { AAA: [100, 0, 0] },
};
const map = parseWorldMap(payload);
const words: HoverWords = {
  say: (key, values) => `${key} ${JSON.stringify(values)}`,
  ownerLabel: (key) => key || "(erased)",
  terrainName: (terrain) => terrain,
};
const none = new Map<number, string>();

describe("地图创作 hover line", () => {
  it("names the state and owner of the province under the pointer", () => {
    expect(describeHover(map, 1, "countries", none, none, words)).toBe(
      'hovering {"state":"One","owner":"AAA"}',
    );
    expect(
      describeHover(map, 1, "countries", new Map([[1, "BBB"]]), none, words),
    ).toBe('hoverPainted {"state":"One","from":"AAA","to":"BBB"}');
    expect(
      describeHover(map, 2, "terrain", none, new Map([[2, "hills"]]), words),
    ).toBe('hoverPainted {"state":"One","from":"forest","to":"hills"}');
  });

  it("says nothing over the sea, off the map, or with no map", () => {
    expect(describeHover(map, 3, "countries", none, none, words)).toBeNull();
    expect(describeHover(map, 0, "countries", none, none, words)).toBeNull();
    expect(describeHover(null, 1, "countries", none, none, words)).toBeNull();
  });

  it("a number the map has no province for is left out, in every mode", () => {
    const beyond = map.provinces.length + 4000;
    for (const mode of ["countries", "terrain", "rivers", "land"]) {
      expect(() =>
        describeHover(map, beyond, mode, none, none, words),
      ).not.toThrow();
      expect(describeHover(map, beyond, mode, none, none, words)).toBeNull();
    }
  });
});
