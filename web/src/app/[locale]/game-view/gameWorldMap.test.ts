import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { parseWorldMap } from "@/components/workbench/world-map/worldMapModel";
import {
  homeProvince,
  mapClickTarget,
  overGameWindow,
  unitCounters,
} from "./GameWorldMap";

const map = parseWorldMap({
  provinces: [
    [1, 0x010101, 0, 0, "plains", 1, 10, 1],
    [2, 0x020202, 0, 0, "plains", 1, 20, 1],
    [3, 0x030303, 1, 0, "ocean", 0, 0, 2],
    [4, 0x040404, 0, 0, "plains", 1, 0, 1],
    [5, 0x050505, 0, 0, "plains", 1, 30, 1],
    [6, 0x060606, 0, 0, "plains", 1, 20, 1],
  ],
  states: [
    [10, "Brandenburg", "GER"],
    [20, "Ile de France", "FRA"],
    [30, "Nowhere", ""],
  ],
  regions: [
    [1, "Europe"],
    [2, "North Sea"],
  ],
  points: [
    [2, 1, "Orleans"],
    [6, 20, "Paris"],
  ],
  continents: ["europe"],
  colors: {},
});
const indexOf = (id: number) => map.provinces.findIndex((p) => p.id === id);

describe("a click on the game view's map", () => {
  it("opens the state view on the country's own state", () => {
    expect(mapClickTarget(map, indexOf(1), "ger")).toEqual({ state: 10 });
  });
  it("opens diplomacy on another country's land", () => {
    expect(mapClickTarget(map, indexOf(2), "GER")).toEqual({ country: "FRA" });
    expect(mapClickTarget(map, indexOf(1), null)).toEqual({ country: "GER" });
  });
  it("opens nothing on sea, stateless or unowned land, or off the map", () => {
    expect(mapClickTarget(map, indexOf(3), "GER")).toBeNull();
    expect(mapClickTarget(map, indexOf(4), "GER")).toBeNull();
    expect(mapClickTarget(map, indexOf(5), "GER")).toBeNull();
    expect(mapClickTarget(map, 0, "GER")).toBeNull();
  });
});

describe("where the game view's map opens", () => {
  it("on the country's biggest victory point", () => {
    expect(homeProvince(map, "FRA")).toBe(indexOf(6));
  });
  it("on any of its land without one, nowhere for a country with none", () => {
    expect(homeProvince(map, "GER")).toBe(indexOf(1));
    expect(homeProvince(map, "ZQX")).toBe(0);
    expect(homeProvince(map, null)).toBe(0);
  });
});

describe("a click on a game window", () => {
  const box = (left: number, top: number, width: number, height: number) => ({
    getBoundingClientRect: () => ({
      left,
      top,
      right: left + width,
      bottom: top + height,
      width,
      height,
    }),
  });
  const root = {
    querySelectorAll: () => [box(0, 80, 540, 1000), box(0, 0, 0, 0)],
  } as unknown as ParentNode;
  it("is not on the map", () => {
    expect(overGameWindow(260, 340, root)).toBe(true);
    expect(overGameWindow(900, 400, root)).toBe(false);
  });
  it("is checked before the map does anything with it", () => {
    const mapFile = readFileSync(
      join(__dirname, "GameWorldMap.tsx"),
      "utf8",
    ).replace(/\s+/g, " ");
    expect(mapFile).toContain("if (at && overGameWindow(at.x, at.y)) return;");
    const canvas = readFileSync(
      join(__dirname, "..", "gui-renderer", "GuiCanvas.tsx"),
      "utf8",
    );
    expect(canvas).toContain('data-window-bg=""');
    expect(canvas).toContain(
      "pushSprite(ctx, `bg${ctx.keySeq.n++}`, bref, bmeta, bx, by, bw, bh, bg.frame, undefined, true);",
    );
  });
});

describe("the map in 编辑", () => {
  const flat = (file: string) =>
    readFileSync(join(__dirname, file), "utf8").replace(/\s+/g, " ");
  it("picks the province in 编辑 only, the map's own picking first", () => {
    const page = flat("page.tsx");
    expect(page).toContain(
      "onProvince={ editMode && !uiEdit ? (index) => { if (mapPick.current?.(index)) return;",
    );
    expect(page).toContain("<ProvinceEditPanel");
    expect(flat("GameWorldMap.tsx")).toContain(
      "if (onProvince) { if (index) onProvince(index); return; }",
    );
  });
  it("edits the clicked state with the 地块 workbench's own editor, any country's", () => {
    const panel = flat("ProvinceEditPanel.tsx");
    expect(panel).toContain("<StateWorkbenchForm");
    expect(panel).toContain("<ProvinceEditor");
    const form = flat("StateWorkbenchForm.tsx");
    expect(form).toContain(
      "<StateEditor state={row} countryTags={countryTags ?? knownTags} onSave={save} />",
    );

    expect(flat("StateEditPanel.tsx")).toContain("<StateWorkbenchForm");

    expect(form).toContain(
      "useStateStore.getState().updateState(projectId, id, data)",
    );

    expect(form).toContain(
      "statesApi.seedFromVanilla(projectId, { state_ids: [stateId] })",
    );
  });
});

describe("unitCounters · the country's divisions on the map", () => {
  it("counts the divisions in each province; one with no province is not on the map", () => {
    const counters = unitCounters([
      { location: 6521 },
      { location: 6521 },
      { location: 11666 },
      { location: 0 },
      {},
    ]);
    expect(counters).toEqual([
      { province: 6521, count: 2 },
      { province: 11666, count: 1 },
    ]);
    expect(unitCounters(undefined)).toEqual([]);
  });
});
