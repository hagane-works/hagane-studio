import { describe, expect, it } from "vitest";
import {
  parseWorldMap,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import {
  seaRegionSplits,
  statesAcrossRegions,
  statesWithoutLand,
} from "./seaRegions";

const payload = (regionOf105: number, island: boolean): WorldMapPayload => ({
  provinces: [
    [101, 0x010101, 0, 1, "plains", 1, 1, 10],
    [102, 0x020202, 1, 1, "ocean", 0, 0, 20],
    [103, 0x030303, 1, 0, "ocean", 0, 0, 20],
    [104, 0x040404, 0, 1, "plains", 1, 2, 10],
    [105, 0x050505, 1, 1, "ocean", 0, 0, regionOf105],
    [106, 0x060606, 1, 0, "ocean", 0, 0, 40],
    ...(island
      ? [
          [
            107,
            0x070707,
            0,
            1,
            "plains",
            1,
            3,
            20,
          ] as WorldMapPayload["provinces"][number],
        ]
      : []),
  ],
  states: [
    [1, "West", "AAA"],
    [2, "Middle", "AAA"],
    [3, "Isle", "AAA"],
  ],
  regions: [
    [10, "Land"],
    [20, "Cut Sea"],
    [40, "East Sea"],
  ],
  continents: ["Europe"],
  colors: {},
});
const neighbours = new Map<number, Set<number>>([
  [1, new Set([2, 6])],
  [2, new Set([1, 3])],
  [3, new Set([2, 4])],
  [4, new Set([3, 5])],
  [5, new Set([4, 6, 7])],
  [6, new Set([5, 1])],
  [7, new Set([5])],
]);

describe("sea regions in pieces (mod 232)", () => {
  it("names the part the land cut off", () => {
    expect(
      seaRegionSplits(parseWorldMap(payload(20, false)), neighbours),
    ).toEqual([{ region: 20, provinces: [105] }]);
  });

  it("counts an island with the sea it stands in", () => {
    expect(
      seaRegionSplits(parseWorldMap(payload(20, true)), neighbours),
    ).toEqual([{ region: 20, provinces: [105, 107] }]);
  });

  it("finds nothing once the part has joined the sea beside it", () => {
    expect(
      seaRegionSplits(parseWorldMap(payload(40, false)), neighbours),
    ).toEqual([]);
  });

  it("leaves a land-only piece alone: the base game joins such islands by a strait", () => {
    expect(
      seaRegionSplits(parseWorldMap(payload(40, true)), neighbours),
    ).toEqual([]);
  });
});

describe("states across strategic regions", () => {
  it("names a state whose provinces sit in two regions", () => {
    const map = parseWorldMap(payload(20, false));
    expect(statesAcrossRegions(map)).toEqual([]);

    const across = payload(20, false);
    across.provinces[0] = [101, 0x010101, 0, 1, "plains", 1, 2, 10];
    across.provinces[3] = [104, 0x040404, 0, 1, "plains", 1, 2, 20];
    expect(statesAcrossRegions(parseWorldMap(across))).toEqual([2]);
  });
});

describe("states made only of lake provinces", () => {
  it("names a state left with only its lake, with the lake", () => {
    expect(statesWithoutLand(parseWorldMap(payload(20, false)))).toEqual([]);

    const lakes = payload(20, false);
    lakes.provinces.push(
      [108, 0x080808, 2, 0, "lakes", 0, 3, 10],
      [109, 0x090909, 2, 0, "lakes", 0, 1, 10],
    );
    expect(statesWithoutLand(parseWorldMap(lakes))).toEqual([
      { state: 3, provinces: [108] },
    ]);
  });
});
