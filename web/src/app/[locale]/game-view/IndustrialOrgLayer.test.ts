import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  MIO_LIST_WINDOW,
  MIO_SLOTS,
  filterOrgs,
  mioListWindow,
  opensIndustrialOrgDetails,
  toggleFilter,
  type GuiIndustrialOrg,
  type MioCategory,
} from "./IndustrialOrgLayer";

const node = (name: string, id: string, children: GuiNode[] = []): GuiNode =>
  ({ name, _id: id, frame: 1, children }) as unknown as GuiNode;

const listWindow = (): GuiWindow =>
  ({
    name: MIO_LIST_WINDOW,
    node: node(MIO_LIST_WINDOW, "root", [
      node("sorting_options", "so", [
        node("sort_size", "s1"),
        node("sort_name", "s2"),
        node("sort_cost", "s3"),
        node("sort_capacity", "s4"),
      ]),
      node("filter_options", "fo", [
        node("materiel_filter", "m", [node("materiel_filter_button", "mb")]),
        node("tank_filter", "t", [node("tank_filter_button", "tb")]),
        node("plane_filter", "p", [node("plane_filter_button", "pb")]),
        node("ship_filter", "s", [node("ship_filter_button", "sb")]),
      ]),
    ]),
    sprites: {},
    texts: {},
    fonts: {},
  }) as unknown as GuiWindow;

const org = (token: string, category: MioCategory): GuiIndustrialOrg => ({
  token,
  name: token,
  category,
  size: 0,
  equipment: [],
});

function frameOf(root: GuiNode, name: string): unknown {
  let found: unknown;
  const walk = (at: GuiNode) => {
    if (at.name === name) found = at.frame;
    for (const child of at.children ?? []) walk(child);
  };
  walk(root);
  return found;
}

describe("MIO_SLOTS", () => {
  it("opens details from either the organisation row or its details button", () => {
    expect(opensIndustrialOrgDetails("background_button")).toBe(true);
    expect(opensIndustrialOrgDetails("details_button")).toBe(true);
    expect(opensIndustrialOrgDetails("filter_button")).toBe(false);
  });

  it("opens each roster on its own filter", () => {
    expect(MIO_SLOTS).toEqual({
      materiel_manufacturer: "materiel",
      tank_manufacturer: "tank",
      aircraft_manufacturer: "plane",
      naval_manufacturer: "ship",
    });
  });
});

describe("filterOrgs", () => {
  const items = [
    org("porsche", "tank"),
    org("junkers", "plane"),
    org("mauser", "materiel"),
  ];

  it("keeps the companies of the ticked filters, in their order", () => {
    expect(
      filterOrgs(items, new Set<MioCategory>(["tank", "plane"])).map(
        (item) => item.token,
      ),
    ).toEqual(["porsche", "junkers"]);
  });

  it("lists every company when no filter is ticked", () => {
    expect(filterOrgs(items, new Set()).map((item) => item.token)).toEqual([
      "porsche",
      "junkers",
      "mauser",
    ]);
  });
});

describe("toggleFilter", () => {
  it("ticks a filter and unticks it again, leaving the others", () => {
    const tank = new Set<MioCategory>(["tank"]);
    expect([...toggleFilter(tank, "plane")].sort()).toEqual(["plane", "tank"]);
    expect([...toggleFilter(tank, "tank")]).toEqual([]);
    expect([...tank]).toEqual(["tank"]);
  });
});

describe("mioListWindow", () => {
  it("draws the ticked filters on their checkbox's second frame", () => {
    const shown = mioListWindow(listWindow(), new Set<MioCategory>(["tank"]))
      .win.node;
    expect(frameOf(shown, "tank_filter_button")).toBe(2);
    expect(frameOf(shown, "plane_filter_button")).toBe(1);
    expect(frameOf(shown, "materiel_filter_button")).toBe(1);
  });

  it("hides the sort buttons the recording does not show, and leaves the window it was given alone", () => {
    const original = listWindow();
    const { hidden } = mioListWindow(original, new Set<MioCategory>(["tank"]));
    expect(hidden.sort()).toEqual(["s3", "s4"]);
    expect(frameOf(original.node, "tank_filter_button")).toBe(1);
  });
});
