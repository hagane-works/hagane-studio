import { beforeEach, describe, expect, it } from "vitest";
import type { GuiCountryStats, GuiNode } from "@/lib/guiRenderApi";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";
import {
  amountColor,
  nodeNamed,
  stripEntries,
  stripSlot,
} from "./resourceStrip";

const TEXT_COLORS = REFERENCE_TABLES.text_colors.global;
beforeEach(() => setReferenceTables(REFERENCE_TABLES));

const frames = {
  oil: 1,
  aluminium: 2,
  rubber: 3,
  tungsten: 4,
  steel: 5,
  chromium: 6,
  coal: 7,
};

describe("the resource strip", () => {
  it("lists every resource in frame order at its surplus, shortfalls included", () => {
    const stats = {
      tag: "GER",
      resource_surplus: {
        coal: 195,
        oil: 3,
        rubber: -5,
        steel: 177,
        aluminium: 104,
        tungsten: 3,
        chromium: 1,
      },
    } as GuiCountryStats;
    expect(stripEntries(frames, stats)).toEqual([
      ["oil", 3],
      ["aluminium", 104],
      ["rubber", -5],
      ["tungsten", 3],
      ["steel", 177],
      ["chromium", 1],
      ["coal", 195],
    ]);
  });

  it("falls back to what the states hold when a service has no surplus yet", () => {
    const stats = {
      tag: "GER",
      resources: { steel: 160, rubber: 0 },
    } as GuiCountryStats;
    expect(stripEntries(frames, stats)).toEqual([["steel", 160]]);
  });

  it("colours a shortfall red and a surplus green", () => {
    expect(amountColor(-5)).toBe(TEXT_COLORS.R);
    expect(amountColor(195)).toBe(TEXT_COLORS.G);
    expect(amountColor(0)).toBeUndefined();
  });

  it("prints every amount plain when the colour table is empty", () => {
    setReferenceTables({});
    expect(amountColor(-5)).toBeUndefined();
    expect(amountColor(195)).toBeUndefined();
  });
});

const grid = (raw: Record<string, unknown>): GuiNode =>
  ({ type: "gridboxtype", name: "resources_grid", raw }) as unknown as GuiNode;

describe("stripSlot", () => {
  it("keeps the base game's one row of seven, 70 apart", () => {
    const vanilla = grid({
      slotsize: { width: 70, height: 31 },
      max_slots: { x: 7, y: 1 },
      add_horizontal: "no",
    });
    expect([0, 1, 6].map((at) => stripSlot(vanilla, at, 70))).toEqual([
      { x: 0, y: 0 },
      { x: 70, y: 0 },
      { x: 420, y: 0 },
    ]);
  });

  it("fills a column first in a grid two rows deep: a mod's ten resources stay on the panel", () => {
    const mod = grid({
      slotsize: { width: 70, height: 31 },
      max_slots: { x: 7, y: 2 },
      add_horizontal: "no",
    });
    expect([0, 1, 2, 9].map((at) => stripSlot(mod, at, 70))).toEqual([
      { x: 0, y: 0 },
      { x: 0, y: 31 },
      { x: 70, y: 0 },
      { x: 280, y: 31 },
    ]);
  });

  it("fills a row first and wraps with add_horizontal = yes", () => {
    const rows = grid({
      slotsize: { width: 40, height: 20 },
      max_slots_horizontal: 2,
      add_horizontal: "yes",
    });
    expect([0, 1, 2].map((at) => stripSlot(rows, at, 70))).toEqual([
      { x: 0, y: 0 },
      { x: 40, y: 0 },
      { x: 0, y: 20 },
    ]);
  });

  it("falls back to the given width in one row when the grid says nothing", () => {
    expect(stripSlot(undefined, 3, 70)).toEqual({ x: 210, y: 0 });
  });
});

describe("nodeNamed", () => {
  it("finds a node at any depth", () => {
    const root = {
      name: "a",
      children: [{ name: "b", children: [{ name: "resources_grid" }] }],
    } as unknown as GuiNode;
    expect(nodeNamed(root, "resources_grid")?.name).toBe("resources_grid");
    expect(nodeNamed(root, "missing")).toBeUndefined();
  });
});
