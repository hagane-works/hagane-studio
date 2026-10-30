import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type {
  GuiCountryStats,
  GuiNode,
  GuiSprite,
  GuiWindow,
} from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import {
  buildPoliticsModel,
  pieSlicePaths,
  politicsExtraSprites,
} from "./politicsModel";
import { matchNamePath } from "./gameNavigation";
import { pinFullSizes } from "./topbarLayout";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "icontype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    ...partial,
  };
}

const sprite: GuiSprite = {
  resolved: true,
  frames: 1,
  width: 32,
  height: 32,
  effect: null,
  effect_kind: null,
  cornered: false,
  bordersize: null,
  vanilla_url: "http://store/gfx/interface/faction_logo_axis_miniature.png",
};

function politicsWindow(): GuiWindow {
  const text = (name: string, value: string) =>
    node({ type: "instanttextboxtype", name, text: value, font: "hoi_18mbs" });
  const container = (name: string, children: GuiNode[]) =>
    node({ type: "containerwindowtype", name, children });
  const root = container("countrypoliticsview", [
    text("ideology", "ideology"),
    text("elections", "elections"),
    container("faction", [
      container("no_faction", [text("view_factions_text", "CREATE_FACTION")]),
      container("original_in_faction", [text("faction_name", "faction")]),
      container("in_faction", [
        node({
          name: "faction_icon",
          sprite: "GFX_faction_logo_generic_democratic",
        }),
        text("faction_name", ""),
      ]),
    ]),
  ]);
  assignIds(root);
  return {
    name: "countrypoliticsview",
    file: "interface/countrypoliticsview.gui",
    node: root,
    sprites: {},
    texts: {},
    fonts: {
      hoi_18mbs: { bitmap: { line: 20 } },
    } as unknown as GuiWindow["fonts"],
    overlap_groups: [],
  };
}

const idOf = (n: GuiNode) => (n as GuiNode & { _id?: string })._id ?? "";
const one = (root: GuiNode, path: string) => matchNamePath(root, path)[0];

const GERMANY: GuiCountryStats = {
  tag: "GER",
  faction: {
    name: "Axis",
    icon: "GFX_faction_logo_axis_miniature",
    initiative: 0,
  },
  politics: {
    ruling_party: "fascism",
    ideology: "Fascist",
    elections: "No Elections",
  },
} as GuiCountryStats;

describe("buildPoliticsModel", () => {
  it("writes the ruling ideology and the election note over the placeholders", () => {
    const win = politicsWindow();
    const model = buildPoliticsModel({ win, stats: GERMANY, extraSprites: {} });
    expect(model.textOverrides[idOf(one(win.node, "ideology"))]).toBe(
      "Fascist",
    );
    expect(model.textOverrides[idOf(one(win.node, "elections"))]).toBe(
      "No Elections",
    );
  });

  it("never shows the placeholder words, even before the numbers arrive", () => {
    const win = politicsWindow();
    const model = buildPoliticsModel({
      win,
      stats: undefined,
      extraSprites: {},
    });
    expect(model.textOverrides[idOf(one(win.node, "ideology"))]).toBe("");
    expect(model.textOverrides[idOf(one(win.node, "elections"))]).toBe("");
  });

  it("shows the faction a country starts in instead of Create Faction", () => {
    const win = politicsWindow();
    const model = buildPoliticsModel({
      win,
      stats: GERMANY,
      extraSprites: { GFX_faction_logo_axis_miniature: sprite },
    });
    expect(model.hidden.has(idOf(one(win.node, "no_faction")))).toBe(true);
    expect(model.shown.has(idOf(one(win.node, "faction/in_faction")))).toBe(
      true,
    );

    expect(model.shown.has(idOf(one(win.node, "original_in_faction")))).toBe(
      false,
    );
    const name = one(model.win.node, "in_faction/faction_name");
    expect(name.text).toBe("Axis");

    expect(name.position).toEqual({ x: 40, y: 4 });
    expect(one(model.win.node, "in_faction/faction_icon").sprite).toBe(
      "GFX_faction_logo_axis_miniature",
    );
    expect(model.win.sprites.GFX_faction_logo_axis_miniature).toBe(sprite);
  });

  it("keeps Create Faction for a country outside every faction", () => {
    const win = politicsWindow();
    const model = buildPoliticsModel({
      win,
      stats: { ...GERMANY, faction: null },
      extraSprites: {},
    });
    expect(model.hidden.size).toBe(0);
    expect(model.shown.size).toBe(0);
    expect(politicsExtraSprites({ ...GERMANY, faction: null })).toEqual([]);
  });

  it("draws the 32px miniature, not the .gui's 200x100 placeholder emblem", () => {
    expect(politicsExtraSprites(GERMANY)).toEqual([
      "GFX_faction_logo_axis_miniature",
    ]);
  });
});

describe("pieSlicePaths", () => {
  const pie = [
    {
      party: "democratic",
      value: 35,
      color: [0, 0, 255] as [number, number, number],
    },
    {
      party: "communism",
      value: 10,
      color: [255, 0, 0] as [number, number, number],
    },
    {
      party: "fascism",
      value: 40,
      color: [150, 75, 0] as [number, number, number],
    },
    {
      party: "neutrality",
      value: 15,
      color: [124, 124, 124] as [number, number, number],
    },
  ];

  it("starts at 3 o'clock and runs counter-clockwise in the given order", () => {
    const [democratic, communism] = pieSlicePaths(pie, 100, 100, 10);

    expect(democratic.d.startsWith("M 100 100 L 110 100 A 10 10 0 0 0")).toBe(
      true,
    );
    const end = democratic.d.split(" ").slice(-3, -1).map(Number);
    expect(end[1]).toBeLessThan(100);

    expect(communism.d).toContain(`L ${end[0]} ${end[1]}`);
    expect(communism.fill).toBe("rgb(255, 0, 0)");
  });

  it("uses the large arc only for a slice over half the pie", () => {
    const big = pieSlicePaths(
      [
        { party: "a", value: 70, color: [0, 0, 0] },
        { party: "b", value: 30, color: [0, 0, 0] },
      ],
      0,
      0,
      10,
    );
    expect(big[0].d).toContain("A 10 10 0 1 0");
    expect(big[1].d).toContain("A 10 10 0 0 0");
  });

  it("draws a lone family as a whole circle and skips empty shares", () => {
    const whole = pieSlicePaths(
      [
        { party: "n", value: 100, color: [1, 2, 3] },
        { party: "c", value: 0, color: [0, 0, 0] },
      ],
      50,
      50,
      5,
    );
    expect(whole).toHaveLength(1);
    expect(whole[0].d).toContain("a 5 5 0 1 0 10 0");
  });

  it("draws nothing for no support at all", () => {
    expect(pieSlicePaths([], 0, 0)).toEqual([]);
  });
});

describe("the politics screen's lower box", () => {
  it("ends on the screen's bottom edge, its frame shown, as the game lays it out", () => {
    const root = node({
      type: "containerwindowtype",
      name: "countrypoliticsview",
      position: { x: -6, y: 78 },
      size: { width: 550, height: "100%%" },
      children: [
        node({
          type: "containerwindowtype",
          name: "ideas",
          position: { x: 0, y: 425 },
          size: { width: "100%%", height: "100%%" },
        }),
      ],
    });
    const pinned = pinFullSizes(root, 1920, 1080);
    const ideas = pinned.children[0];
    expect(pinned.size?.height).toBe(1002);
    expect(78 + 425 + Number(ideas.size?.height)).toBe(1080);
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toMatch(/PINNED_SIZES = new Set\(\[[^\]]*POLITICS[^\]]*\]\)/);

    expect(page).toMatch(
      /PINNED_SIZES = new Set\(\[[^\]]*\.\.\.Object\.values\(TOPBAR_TARGETS\)[^\]]*\]\)/,
    );
  });
});
