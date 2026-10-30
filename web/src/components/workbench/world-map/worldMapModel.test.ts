import { describe, expect, it } from "vitest";
import {
  LAKE,
  SEA,
  UNOWNED,
  changedRects,
  groupPlaces,
  layerTable,
  mix,
  pack,
  paintLayer,
  paintLayerRect,
  parseWorldMap,
  pixelIndex,
  provinceBoxes,
  provinceCentroids,
  provinceNeighbours,
  provincesInCircle,
  searchNames,
  type WorldMapPayload,
} from "./worldMapModel";

const payload: WorldMapPayload = {
  provinces: [
    [11, 0x0a141e, 0, 0, "plains", 1, 1, 7],
    [12, 0x28323c, 0, 1, "forest", 1, 1, 7],
    [13, 0x464f5a, 0, 1, "hills", 1, 2, 8],
    [14, 0x000080, 1, 1, "ocean", 0, 0, 9],
    [15, 0x6e78a0, 2, 0, "lakes", 1, 0, 7],
  ],
  states: [
    [1, "Inland", "AAA"],
    [2, "Coast", "BBB"],
  ],
  regions: [
    [7, "Heartland"],
    [8, "Marches"],
    [9, "Sea"],
  ],
  points: [[12, 10, "Capital City"]],
  continents: ["Europe"],
  colors: { AAA: [100, 0, 0], BBB: [0, 100, 0] },
};
const map = parseWorldMap(payload);

function picture(rows: number[][]): Uint8ClampedArray {
  const byId = new Map(payload.provinces.map((p) => [p[0], p[1]]));
  const out = new Uint8ClampedArray(rows.length * rows[0].length * 4);
  rows.flat().forEach((id, p) => {
    const rgb = byId.get(id) ?? 0xffffff;
    out[p * 4] = rgb >> 16;
    out[p * 4 + 1] = (rgb >> 8) & 255;
    out[p * 4 + 2] = rgb & 255;
    out[p * 4 + 3] = 255;
  });
  return out;
}

describe("the world map model", () => {
  it("reads provinces, states and regions", () => {
    expect(map.provinces).toHaveLength(6);
    expect(map.provinces[3]).toMatchObject({
      id: 13,
      kind: "land",
      coastal: true,
      state: 2,
      region: 8,
    });
    expect(map.provinces[4].kind).toBe("sea");
    expect(map.provinces[5].kind).toBe("lake");
    expect(map.states.get(2)).toEqual({ name: "Coast", owner: "BBB" });
    expect(map.regions.get(7)).toBe("Heartland");
  });

  it("knows the province of every pixel, and none for a colour the map does not have", () => {
    const index = pixelIndex(
      picture([
        [11, 11, 12],
        [13, 99, 14],
      ]),
      map.indexOfColor,
    );
    expect(Array.from(index).map((i) => map.provinces[i].id)).toEqual([
      11, 11, 12, 13, 0, 14,
    ]);
  });

  it("colours land by owner, state or region, and water as water", () => {
    const political = layerTable(map, "political", { province: 0 });
    expect(political.colors[1]).toBe(pack(100, 0, 0));
    expect(political.colors[3]).toBe(pack(0, 100, 0));
    expect(political.colors[4]).toBe(SEA);
    expect(political.colors[5]).toBe(LAKE);

    expect(political.groups[1]).toBe(political.groups[2]);
    expect(political.groups[1]).not.toBe(political.groups[3]);

    const states = layerTable(map, "state", { province: 0 });
    expect(states.colors[1]).toBe(states.colors[2]);
    expect(states.colors[1]).not.toBe(states.colors[3]);

    const unowned = parseWorldMap({
      ...payload,
      states: [
        [1, "Inland", ""],
        [2, "Coast", "BBB"],
      ],
    });
    expect(layerTable(unowned, "political", { province: 0 }).colors[1]).toBe(
      UNOWNED,
    );
  });

  it("brightens the picked province most and the rest of its state less", () => {
    const plain = layerTable(map, "political", { province: 0 });
    const picked = layerTable(map, "political", { province: 1 });
    expect(picked.colors[1]).toBe(mix(plain.colors[1], [255, 255, 255], 0.6));
    expect(picked.colors[2]).toBe(mix(plain.colors[2], [255, 255, 255], 0.3));
    expect(picked.colors[3]).toBe(plain.colors[3]);
  });

  it("lights the whole state, province or strategic region picked, and outlines that kind of block", () => {
    const plain = layerTable(map, "political", { province: 0 });
    const lit = (n: number) => mix(plain.colors[n], [255, 255, 255], 0.5);
    const state = layerTable(map, "political", { province: 1, scope: "state" });
    expect([state.colors[1], state.colors[2], state.colors[3]]).toEqual([
      lit(1),
      lit(2),
      plain.colors[3],
    ]);

    expect(state.groups[1]).toBe(state.groups[2]);
    expect(state.groups[1]).not.toBe(state.groups[3]);
    const province = layerTable(map, "political", {
      province: 1,
      scope: "province",
    });
    expect([province.colors[1], province.colors[2]]).toEqual([
      lit(1),
      plain.colors[2],
    ]);

    const region = layerTable(map, "political", {
      province: 1,
      scope: "region",
    });
    expect([region.colors[2], region.colors[3], region.colors[5]]).toEqual([
      lit(2),
      plain.colors[3],
      mix(plain.colors[5], [255, 255, 255], 0.5),
    ]);
    expect(region.groups[1]).toBe(region.groups[5]);
    expect(region.groups[1]).not.toBe(region.groups[3]);

    expect(
      layerTable(map, "political", { province: 0, scope: "state" }).groups[3],
    ).toBe(2);
  });

  it("draws strong borders between owners and faint ones between provinces of one owner", () => {
    const index = pixelIndex(
      picture([
        [11, 12, 13],
        [11, 12, 13],
      ]),
      map.indexOfColor,
    );
    const table = layerTable(map, "political", { province: 0 });
    const out = new Uint32Array(index.length);
    paintLayer(index, 3, table, out);
    const black: [number, number, number] = [0, 0, 0];

    expect(out[0]).toBe(mix(table.colors[1], black, 0.14));
    expect(out[1]).toBe(mix(table.colors[2], black, 0.55));
    expect(out[2]).toBe(table.colors[3]);
  });
});

describe("names on the world map", () => {
  it("finds states, cities and regions by any part of their name", () => {
    expect(
      searchNames(map, "cap").map((hit) => [
        hit.kind,
        hit.id,
        map.provinces[hit.province].id,
      ]),
    ).toEqual([["city", 12, 12]]);
    expect(searchNames(map, "land").map((hit) => hit.name)).toEqual([
      "Inland",
      "Heartland",
    ]);
    expect(searchNames(map, "  ")).toEqual([]);
  });

  it("knows where each province, state and sea is", () => {
    const index = pixelIndex(
      picture([
        [11, 11, 12],
        [13, 14, 14],
      ]),
      map.indexOfColor,
    );
    const centroids = provinceCentroids(index, 3, map.provinces.length);
    expect([centroids.x[1], centroids.y[1], centroids.count[1]]).toEqual([
      0.5, 0, 2,
    ]);
    const states = groupPlaces(
      map,
      centroids,
      (p) => (p.kind === "land" ? p.state : 0),
      3,
    );
    expect(states.get(1)).toMatchObject({ size: 3, x: 1, y: 0 });
    const seas = groupPlaces(
      map,
      centroids,
      (p) => (p.kind === "sea" ? p.region : 0),
      3,
    );
    expect(seas.get(9)).toMatchObject({ size: 2, x: 1.5, y: 1 });
  });

  it("centres a sea across the map's east-west edge on the edge", () => {
    const wrapped = parseWorldMap({
      ...payload,
      provinces: [
        [11, 0x0a141e, 1, 0, "ocean", 0, 0, 9],
        [12, 0x28323c, 1, 0, "ocean", 0, 0, 9],
        [13, 0x464f5a, 0, 0, "plains", 1, 2, 8],
      ],
    });
    const edge = pixelIndex(
      picture([[11, 13, 13, 13, 13, 13, 13, 13, 13, 12]]),
      wrapped.indexOfColor,
    );
    const seaPlaces = groupPlaces(
      wrapped,
      provinceCentroids(edge, 10, wrapped.provinces.length),
      (p) => (p.kind === "sea" ? p.region : 0),
      10,
    );

    expect(seaPlaces.get(9)!.x).toBeCloseTo(9.5);
  });
});

describe("painting only what changed (地图创作)", () => {
  const rows = [
    [11, 11, 12, 12, 14, 14],
    [11, 11, 12, 13, 14, 14],
    [15, 15, 13, 13, 13, 14],
  ];
  const index = pixelIndex(picture(rows), map.indexOfColor);
  const width = rows[0].length;
  const withOwner = (state: number, owner: string) => ({
    ...map,
    states: new Map(
      [...map.states].map(([id, s]) => [
        id,
        id === state ? { ...s, owner } : s,
      ]),
    ),
    colors: { ...map.colors, CCC: [0, 0, 100] as [number, number, number] },
  });

  it("knows each province's box", () => {
    const boxes = provinceBoxes(index, width, map.provinces.length);
    expect(Array.from(boxes.slice(3 * 4, 3 * 4 + 4))).toEqual([2, 1, 5, 3]);
    expect(boxes[0 * 4 + 2]).toBe(0);
  });

  it("paints a rectangle exactly as the whole picture would", () => {
    const table = layerTable(map, "political", { province: 0 });
    const whole = new Uint32Array(index.length);
    paintLayer(index, width, table, whole);
    const part = new Uint32Array(index.length);
    paintLayerRect(index, width, table, part, [1, 0, 4, 2]);
    for (let y = 0; y < 3; y++)
      for (let x = 0; x < width; x++)
        expect(part[y * width + x]).toBe(
          x >= 1 && x < 4 && y < 2 ? whole[y * width + x] : 0,
        );
  });

  it("repaints only the provinces of a state given away, and their border pixels, to the same picture", () => {
    const boxes = provinceBoxes(index, width, map.provinces.length);
    const before = layerTable(map, "political", { province: 0 });
    const after = layerTable(withOwner(2, "CCC"), "political", { province: 0 });
    const rects = changedRects(before, after, boxes);
    expect(rects).toEqual([[1, 0, 6, 4]]);
    const picture = new Uint32Array(index.length);
    paintLayer(index, width, before, picture);
    for (const rect of rects!)
      paintLayerRect(index, width, after, picture, rect);
    const fresh = new Uint32Array(index.length);
    paintLayer(index, width, after, fresh);
    expect(Array.from(picture)).toEqual(Array.from(fresh));
  });

  it("keeps an owner's border group when other owners come and go, and gives up past the limit", () => {
    const a = layerTable(map, "political", { province: 0 });
    const b = layerTable(withOwner(2, "CCC"), "political", { province: 0 });
    expect(b.groups[1]).toBe(a.groups[1]);
    expect(
      changedRects(a, b, provinceBoxes(index, width, map.provinces.length), 0),
    ).toBeNull();
  });

  it("finds the provinces under a brush, joining east and west", () => {
    expect(
      provincesInCircle(index, width, 0.5, 0.5, 0.5).map(
        (i) => map.provinces[i].id,
      ),
    ).toEqual([11]);
    const wide = provincesInCircle(index, width, 3, 1, 3)
      .map((i) => map.provinces[i].id)
      .sort();
    expect(wide).toEqual([11, 12, 13, 14, 15]);
    expect(
      provincesInCircle(index, width, 5.5, 0.5, 1.2)
        .map((i) => map.provinces[i].id)
        .sort(),
    ).toContain(11);
  });

  it("knows which provinces touch", () => {
    const touching = provinceNeighbours(index, width);
    const ids = (i: number) =>
      [...(touching.get(i) ?? [])].map((j) => map.provinces[j].id).sort();
    expect(ids(1)).toEqual([12, 14, 15]);
    expect(ids(3)).toEqual([12, 14, 15]);
  });
});
