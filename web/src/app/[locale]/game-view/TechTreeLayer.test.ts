import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  backgroundNames,
  centring,
  linkFrame,
  linkSegments,
  pageGridboxes,
  slotOf,
  clampScroll,
  pageContentName,
  scrollLimits,
  scrollTechTree,
  techTreeWindow,
} from "./TechTreeLayer";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "containerwindowtype",
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

const gridbox = (name: string, x: number, raw: Record<string, unknown> = {}) =>
  node({ type: "gridboxtype", name, position: { x, y: 60 }, raw });

describe("item backgrounds", () => {
  it("tries the page's own, then the game's plain one", () => {
    expect(backgroundNames("infantry_folder", "available", false)).toEqual([
      "GFX_technology_infantry_folder_available_item_bg",
      "GFX_technology_available_item_bg",
    ]);
  });

  it("gives a special project's technology its riveted frame", () => {
    expect(
      backgroundNames("support_folder", "unavailable", true, true),
    ).toEqual([
      "GFX_technology_support_folder_special_project_small_unavailable_item_bg",
      "GFX_technology_support_folder_special_project_unavailable_item_bg",
    ]);
    expect(
      backgroundNames("mtgnavalsupportfolder", "unavailable", false, true),
    ).toEqual([
      "GFX_technology_mtgnavalsupportfolder_special_project_unavailable_item_bg",
    ]);
  });
});

describe("link colours", () => {
  it("is green between researched technologies", () => {
    expect(linkFrame("researched", "researched")).toBe(4);
  });

  it("is tan from a researched technology to an open one", () => {
    expect(linkFrame("researched", "available")).toBe(3);
  });

  it("stays grey into one still locked, and out of one not yet researched", () => {
    expect(linkFrame("researched", "unavailable")).toBe(2);
    expect(linkFrame("available", "unavailable")).toBe(2);
  });
});

describe("link routing", () => {
  it("runs straight along a row, either way", () => {
    expect(linkSegments({ x: 0, y: 0 }, { x: 140, y: 0 }, "x")).toEqual({
      runs: [{ x: 35, y: 27, length: 140, vertical: false }],
      corners: [],
    });

    expect(linkSegments({ x: 210, y: 0 }, { x: 0, y: 0 }, "x").runs).toEqual([
      { x: 35, y: 27, length: 210, vertical: false },
    ]);
  });

  it("turns at the next column for a child further on", () => {
    expect(linkSegments({ x: 0, y: 0 }, { x: 280, y: 70 }, "x")).toEqual({
      runs: [
        { x: 35, y: 27, length: 70, vertical: false },
        { x: 97, y: 35, length: 70, vertical: true },
        { x: 105, y: 97, length: 210, vertical: false },
      ],
      corners: [
        { x: 97, y: 27 },
        { x: 97, y: 97 },
      ],
    });
  });

  it("goes straight down to a child in the same column", () => {
    expect(linkSegments({ x: 0, y: 0 }, { x: 0, y: 70 }, "x").runs).toEqual([
      { x: 27, y: 35, length: 70, vertical: true },
    ]);
  });

  it("goes down the child's own column to one a column back", () => {
    expect(linkSegments({ x: 70, y: 0 }, { x: 0, y: 140 }, "x").runs).toEqual([
      { x: 35, y: 27, length: 70, vertical: false },
      { x: 27, y: 35, length: 140, vertical: true },
    ]);
  });

  it("turns the whole route when the years run down the page", () => {
    expect(linkSegments({ x: 0, y: 0 }, { x: 0, y: 140 }, "y").runs).toEqual([
      { x: 27, y: 35, length: 140, vertical: true },
    ]);
  });
});

describe("chain gridboxes", () => {
  it("centres a chain's first row on its gridbox", () => {
    expect(centring(0)).toBe(35);
    expect(centring(140)).toBe(-35);
    expect(centring(210)).toBe(-70);
    expect(centring(140, 35)).toBe(-52.5);
  });

  it("steps by the gridbox's own slot", () => {
    expect(
      slotOf(
        gridbox("basic_battery_tree", 150, {
          slotsize: { width: 70, height: 35 },
        }),
      ),
    ).toEqual({
      width: 70,
      height: 35,
    });
    expect(slotOf(gridbox("tech_support_tree", 100))).toEqual({
      width: 70,
      height: 70,
    });
  });

  it("finds a chain on the open page, not a same-named one elsewhere", () => {
    const old = node({
      name: "naval_folder",
      children: [gridbox("early_ship_hull_light_tree", 150)],
    });
    const mtg = node({
      name: "mtgnavalfolder",
      children: [gridbox("early_ship_hull_light_tree", 70)],
    });
    node({ children: [old, mtg] });
    expect(
      pageGridboxes(mtg).get("early_ship_hull_light_tree")?.position.x,
    ).toBe(70);
  });
});

describe("the research window", () => {
  const win = (): GuiWindow =>
    ({
      name: "countrytechtreeview",
      node: node({
        name: "countrytechtreeview",
        children: [
          node({
            name: "mtgnavalfolder",
            raw: { margin: { top: 13, left: 13, bottom: 24, right: 25 } },
            children: [gridbox("early_ship_hull_light_tree", 70)],
          }),
          node({ name: "folder_tabs", raw: { margin: { top: 4, left: 13 } } }),
        ],
      }),
      sprites: {},
    }) as unknown as GuiWindow;

  it("puts each page's content in the box its margin leaves", () => {
    const page = techTreeWindow(win()).node.children[0];
    expect(page.children).toHaveLength(1);
    const content = page.children[0];
    expect(content.position).toEqual({ x: 13, y: 13 });

    expect(content.size).toEqual({ width: -38, height: -37 });
    expect(content.children[0].name).toBe("early_ship_hull_light_tree");
    expect(content.children[0].position).toEqual({ x: 70, y: 60 });
  });

  it("leaves what is not a page, and the window it was given, alone", () => {
    const original = win();
    const laid = techTreeWindow(original);
    expect(laid.node.children[1].children).toEqual([]);
    expect(original.node.children[0].children[0].name).toBe(
      "early_ship_hull_light_tree",
    );
  });
});

describe("scrolling a research page", () => {
  const view = { x: 13, y: 60, w: 1882, h: 942 };

  it("does not scroll a page that fits", () => {
    expect(
      scrollLimits(view, [{ x: 13, y: 60, w: 1400, h: 900 }], { x: 0, y: 0 }),
    ).toEqual({ x: 0, y: 0 });
  });

  it("scrolls as far as the farthest piece, measured where it rests", () => {
    const stripes = { x: 13, y: 60, w: 1400, h: 1275 };
    expect(scrollLimits(view, [stripes], { x: 0, y: 0 })).toEqual({
      x: 0,
      y: 333,
    });

    expect(
      scrollLimits(view, [{ ...stripes, y: stripes.y - 200 }], {
        x: 0,
        y: 200,
      }),
    ).toEqual({ x: 0, y: 333 });

    expect(
      scrollLimits(
        view,
        [
          stripes,
          { x: 300, y: 1400, w: 180, h: 80 },
          { x: 0, y: 5000, w: 0, h: 0 },
        ],
        { x: 0, y: 0 },
      ),
    ).toEqual({
      x: 0,
      y: 478,
    });
  });

  it("keeps the offset inside its limits, in whole pixels", () => {
    expect(clampScroll({ x: -4, y: 520.6 }, { x: 0, y: 333 })).toEqual({
      x: 0,
      y: 333,
    });
    expect(clampScroll({ x: 0, y: 120.4 }, { x: 0, y: 333 })).toEqual({
      x: 0,
      y: 120,
    });
  });

  const win = (): GuiWindow =>
    ({
      name: "countrytechtreeview",
      node: node({
        name: "countrytechtreeview",
        children: [
          node({
            name: "mtgnavalfolder",
            raw: { margin: { top: 13, left: 13, bottom: 24, right: 25 } },
            children: [gridbox("early_ship_hull_light_tree", 70)],
          }),
          node({ name: "folder_tabs", raw: { margin: { top: 4, left: 13 } } }),
        ],
      }),
      sprites: {},
    }) as unknown as GuiWindow;

  it("moves each page's content box and nothing else", () => {
    const laid = techTreeWindow(win());
    expect(scrollTechTree(laid, { x: 0, y: 0 })).toBe(laid);
    const scrolled = scrollTechTree(laid, { x: 0, y: 160 });
    const content = scrolled.node.children[0].children[0];
    expect(content.name).toBe(pageContentName("mtgnavalfolder"));
    expect(content.raw?.hagane_scroll).toEqual({ x: 0, y: 160 });
    expect(scrolled.node.children[0].raw?.hagane_scroll).toBeUndefined();
    expect(scrolled.node.children[1].raw?.hagane_scroll).toBeUndefined();

    expect(
      laid.node.children[0].children[0].raw?.hagane_scroll,
    ).toBeUndefined();
  });
});

describe("folder tabs", () => {
  const tab = (name: string, x: number) =>
    node({ type: "buttontype", name, position: { x, y: 0 } });
  const withTabs = (): GuiWindow =>
    ({
      name: "countrytechtreeview",
      file: "interface/countrytechtreeview.gui",
      node: node({
        name: "countrytechtreeview",
        children: [
          node({
            name: "folder_tabs",
            children: [
              tab("infantry_folder_tab", 0),
              tab("support_folder_tab", 90),
              tab("armor_folder_tab", 180),
              tab("naval_folder_tab", 270),
            ],
          }),
        ],
      }),
      sprites: {},
      texts: {},
      fonts: {},
    }) as unknown as GuiWindow;

  it("keeps a tab only for a folder the tree has", () => {
    const laid = techTreeWindow(withTabs(), [
      "infantry_folder",
      "ARMOR_FOLDER",
    ]);
    const names = laid.node.children[0].children.map((child) => child.name);
    expect(names).toEqual(["infantry_folder_tab", "armor_folder_tab"]);
  });

  it("keeps every tab while the folders are not known yet", () => {
    expect(techTreeWindow(withTabs()).node.children[0].children).toHaveLength(
      4,
    );
    expect(
      techTreeWindow(withTabs(), []).node.children[0].children,
    ).toHaveLength(4);
  });
});
