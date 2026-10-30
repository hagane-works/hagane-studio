import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  CELL_H,
  CELL_W,
  MIO_DETAIL_WINDOW,
  bonusRows,
  exclusivePairs,
  mioDetailWindow,
  treeLines,
  type GuiIndustrialOrgDetail,
  type GuiIndustrialOrgTrait,
} from "./IndustrialOrgDetailLayer";
import { basePanelOf } from "./gameNavigation";

const trait = (
  token: string,
  x: number,
  y: number,
  more: Partial<GuiIndustrialOrgTrait> = {},
): GuiIndustrialOrgTrait => ({
  token,
  label: token,
  x,
  y,
  special: false,
  bonuses: [],
  parents: [],
  any_parent: [],
  exclusive: [],
  ...more,
});

describe("treeLines", () => {
  it("runs from the parent's foot along its bus to the child's column, then down into it", () => {
    const lines = treeLines([
      trait("wide", 0, 0),
      trait("gear", 2, 2, { parents: ["wide"] }),
    ]);
    const bus = 102;
    expect(lines).toEqual([
      { x: 42, y: bus - 12, length: 12, vertical: true, dotted: false },
      { x: 42, y: bus, length: 2 * CELL_W, vertical: false, dotted: false },
      {
        x: 2 * CELL_W + 42,
        y: bus,
        length: 2 * CELL_H + 20 - bus,
        vertical: true,
        dotted: false,
      },
    ]);
  });

  it("draws an any_parent link dotted, and nothing to a trait that is not shown", () => {
    const lines = treeLines([
      trait("a", 6, 2),
      trait("b", 6, 4, { any_parent: ["a", "gone"] }),
    ]);
    expect(lines.every((line) => line.dotted)).toBe(true);

    expect(lines.filter((line) => !line.vertical)).toEqual([]);
  });
});

describe("exclusivePairs", () => {
  it("marks each pair on a row once, left trait first", () => {
    const cupola = trait("cupola", 6, 2, { exclusive: ["layout"] });
    const layout = trait("layout", 8, 2, { exclusive: ["cupola"] });
    expect(
      exclusivePairs([layout, cupola]).map(([left, right]) => [
        left.token,
        right.token,
      ]),
    ).toEqual([["cupola", "layout"]]);
  });
});

const detail: GuiIndustrialOrgDetail = {
  token: "GER_porsche_organization",
  name: "Porsche",
  icon: "GFX_idea_porsche",
  icon_sprite: {
    resolved: true,
    frames: 1,
    width: 64,
    height: 64,
    effect: null,
    effect_kind: null,
    cornered: false,
    bordersize: null,
  },
  category: "tank",
  size: 0,
  points: 0,
  background: "GFX_MIO_details_background_tank",
  headers: [],
  traits: [],
  sprites: {},
  research_bonus: 0.1,
  assign_cost: 0.1,
  texts: {
    trait_points: "Trait points: §Y0§!",
    attributes: "Attributes",
    equipment_bonuses: "Equipment Bonuses",
    research_bonus: "Research Bonus",
    assign_cost: "Design Team Assign Cost",
  },
  initial: {
    label: "Heavy Tank Designer",
    equipment: [
      {
        group: "mio_cat_eq_all_heavy_tank",
        icon: "GFX_mio_cat_eq_all_heavy_tank",
        design: 1,
        production: 0,
        level: 1,
      },
    ],
  },
};

describe("bonusRows", () => {
  it("lists the column the way the recording shows Porsche's", () => {
    expect(
      bonusRows(detail).map((row) => [
        row.kind,
        row.text ?? row.equipment?.group,
        row.top,
      ]),
    ).toEqual([
      ["separator", "Attributes", 0],
      ["modifier", "Research Bonus: §G10%§!", 29],
      ["modifier", "Design Team Assign Cost:\n£pol_power §Y0.10§!/day", 47],
      ["separator", "Equipment Bonuses", 83],
      ["modifier", "§YHeavy Tank Designer§!", 112],
      ["equipment", "mio_cat_eq_all_heavy_tank", 130],
    ]);
  });
});

const node = (
  name: string,
  id: string,
  children: GuiNode[] = [],
  more: Partial<GuiNode> = {},
): GuiNode =>
  ({
    name,
    _id: id,
    frame: 1,
    children,
    position: { x: 0, y: 0 },
    ...more,
  }) as unknown as GuiNode;

const detailWindow = (): GuiWindow =>
  ({
    name: MIO_DETAIL_WINDOW,
    node: node(
      MIO_DETAIL_WINDOW,
      "root",
      [
        node("industrial_org_name", "title"),
        node(
          "tabs",
          "tabs",
          [node("traits_tab", "tt"), node("history_tab", "ht")],
          {
            position: { x: 480, y: 14 },
            size: { width: "100%%", height: 30 },
          } as Partial<GuiNode>,
        ),
        node("top_left_window", "tl", [
          node("icon", "icon"),
          node("points", "points"),
          node("size_icon_window", "sw", [node("size", "size")]),
        ]),
        node(
          "aggregated_bonuses_window",
          "agg",
          [
            node("open_queue_button", "queue", [], {
              position: { x: 15, y: 0 },
            }),
          ],
          {
            raw: { margin: { left: 20, top: 20 } },
          } as Partial<GuiNode>,
        ),
        node("tree_scrollbar_window", "tree"),
        node("history_window", "history"),
        node("open_policies_window", "pol", [
          node("open_policies_text", "pt"),
          node("click_to_open_policies_text", "ct"),
          node("attached_policy_icon", "pi"),
          node("attached_policy_name", "pn"),
        ]),
        node("toggle_auto_designs", "auto"),
      ],
      { size: { width: 1280, height: 720 } } as Partial<GuiNode>,
    ),
    sprites: {},
    texts: {},
    fonts: {},
  }) as unknown as GuiWindow;

function find(root: GuiNode, name: string): GuiNode | undefined {
  if (root.name === name) return root;
  for (const child of root.children ?? []) {
    const hit = find(child, name);
    if (hit) return hit;
  }
  return undefined;
}

describe("mioDetailWindow", () => {
  it("writes the company's name, points and size, and puts its picture in", () => {
    const shown = mioDetailWindow(detailWindow(), detail, "traits");
    expect(shown.texts).toEqual({
      title: "Porsche",
      points: "Trait points: §Y0§!",
      size: "0",
    });
    expect(find(shown.win.node, "icon")?.sprite).toBe("GFX_idea_porsche");
    expect(shown.win.sprites["GFX_idea_porsche"]).toBe(detail.icon_sprite);
  });

  it("lights the chosen tab and shows only its page", () => {
    const traits = mioDetailWindow(detailWindow(), detail, "traits");
    expect([
      find(traits.win.node, "traits_tab")?.frame,
      find(traits.win.node, "history_tab")?.frame,
    ]).toEqual([2, 1]);
    expect(traits.hidden).toContain("history");
    expect(traits.hidden).not.toContain("tree");
    expect(traits.shown).toEqual(["tree"]);
    const history = mioDetailWindow(detailWindow(), detail, "history");
    expect(history.hidden).toContain("tree");
    expect(history.hidden).not.toContain("history");
    expect(history.shown).toContain("history");
  });

  it("keeps only the Policies label on the policy button, and ticks the auto-update box", () => {
    const shown = mioDetailWindow(detailWindow(), detail, "traits");
    expect(shown.hidden).toEqual(expect.arrayContaining(["ct", "pi", "pn"]));
    expect(shown.hidden).not.toContain("pt");
    expect(find(shown.win.node, "toggle_auto_designs")?.frame).toBe(2);
  });

  it("moves the column's contents by its margin", () => {
    const shown = mioDetailWindow(detailWindow(), detail, "traits");
    expect(find(shown.win.node, "open_queue_button")?.position).toEqual({
      x: 35,
      y: 20,
    });
  });
});

describe("basePanelOf", () => {
  it("goes down a chain of pop-ups to the panel under them", () => {
    expect(basePanelOf(MIO_DETAIL_WINDOW)).toBe("countrypoliticsview");
    expect(basePanelOf("industrial_organisation_list_window")).toBe(
      "countrypoliticsview",
    );
    expect(basePanelOf("countrypoliticsview")).toBe("countrypoliticsview");
    expect(basePanelOf(null)).toBe(null);
  });
});
