import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  constructionWindow,
  speedText,
  type GuiBuilding,
  type GuiConstruction,
} from "./ConstructionLayer";
import { matchNamePath } from "./gameNavigation";

const node = (
  name: string,
  type: string,
  children: GuiNode[] = [],
  extra: Partial<GuiNode> = {},
): GuiNode =>
  ({
    name,
    type,
    children,
    position: { x: 0, y: 0 },
    background: null,
    ...extra,
  }) as GuiNode;

const section = (name: string, grid: string, y: number) =>
  node(
    name,
    "containerwindowtype",
    [node(grid, "gridboxtype", [], { position: { x: 28, y: 52 } })],
    { position: { x: 0, y } },
  );

const view = (): GuiWindow =>
  ({
    name: "countryconstructionsview",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("countryconstructionsview", "containerwindowtype", [
      node("count_container", "containerwindowtype", [
        node("factories_used_txt", "instanttextboxtype"),
        ...[
          "factories_sub_1x",
          "factories_sub_5x",
          "factories_sub_10x",
          "factories_add_1x",
          "factories_add_5x",
          "factories_add_10x",
        ].map((name) => node(name, "buttontype")),
      ]),
      node("production_win_top", "containerwindowtype", [
        node("available_factories_value", "instanttextboxtype"),
        node("traded_factories_value", "instanttextboxtype"),
        node("local_factories_value", "instanttextboxtype"),
        node("cic_bank_icon", "icontype"),
        node("cic_bank_value", "instanttextboxtype"),
        node("speed_container", "containerwindowtype", [
          node("construction_speed_value", "instanttextboxtype"),
        ]),
      ]),
      node(
        "production_lines",
        "containerwindowtype",
        [node("item_grid", "gridboxtype", [], { position: { x: 1, y: 5 } })],
        {
          position: { x: 0, y: 130 },
          raw: { margin: { top: 13, left: 13 } },
        },
      ),
      node("possible_constructions", "containerwindowtype", [
        section("possible_state_constructions", "non_shared_item_grid", 0),
        section("possible_shared_constructions", "shared_item_grid", 50),
        section("possible_province_constructions", "province_item_grid", 100),
        section("possible_facility_constructions", "facility_item_grid", 150),
      ]),
      node("minimize", "buttontype"),
      node("expand", "buttontype"),
    ]),
  }) as unknown as GuiWindow;

const building = (name: string, group: string): GuiBuilding => ({
  name,
  group,
  icon_frame: 1,
  base_cost: 0,
  max_level: 1,
  is_province: false,
  built: 0,
});

const buildings = [
  ...["infrastructure", "air_base", "anti_air_building", "radar_station"].map(
    (name) => building(name, "state"),
  ),
  ...Array.from({ length: 8 }, (_, k) => building(`shared_${k}`, "shared")),
  building("rocket_site", "facility"),
];

const germany: GuiConstruction = {
  tag: "GER",
  civilian: 35,
  used: 17,
  from_trade: 0,
  speed: 0.05,
  rows: [{ kind: "consumer_goods", factories: 17 }],
};

const idOf = (win: GuiWindow, path: string) =>
  (matchNamePath(win.node, path)[0] as GuiNode & { _id?: string })._id ?? "";

describe("the construction screen (hoi4-game -process 2:57)", () => {
  it("is drawn through the page's window chain", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "constructionWindow(win, constructionData.data, constructionBuildings.data?.buildings)",
    );
    expect(page).toContain("constructionView?.win");
  });

  it("stacks the sections under one another and leaves an empty one out", () => {
    const out = constructionWindow(view(), germany, buildings);
    const top = (name: string) =>
      matchNamePath(out.win.node, `possible_constructions/${name}`)[0].position
        .y;
    expect(top("possible_state_constructions")).toBe(0);

    expect(top("possible_shared_constructions")).toBe(96);
    expect(top("possible_facility_constructions")).toBe(96 + 184);
    expect(out.hidden).toContain(
      idOf(out.win, "possible_constructions/possible_province_constructions"),
    );
    expect(out.hidden).not.toContain(
      idOf(out.win, "possible_constructions/possible_facility_constructions"),
    );
  });

  it("keeps the file's places until the catalogue is known", () => {
    const out = constructionWindow(view(), germany, undefined);
    expect(
      matchNamePath(
        out.win.node,
        "possible_constructions/possible_shared_constructions",
      )[0].position.y,
    ).toBe(50);
  });

  it("writes the header and shows the count at nothing with only its ±1 buttons", () => {
    const out = constructionWindow(view(), germany, buildings);
    expect(
      out.texts[idOf(out.win, "production_win_top/available_factories_value")],
    ).toBe("§Y17/35§!");
    expect(
      out.texts[idOf(out.win, "production_win_top/traded_factories_value")],
    ).toBe("0");
    expect(
      out.texts[idOf(out.win, "production_win_top/local_factories_value")],
    ).toBe("35");
    expect(
      out.texts[idOf(out.win, "speed_container/construction_speed_value")],
    ).toBe("§G+5%§!");
    expect(out.texts[idOf(out.win, "count_container/factories_used_txt")]).toBe(
      "0",
    );
    for (const name of [
      "factories_sub_5x",
      "factories_sub_10x",
      "factories_add_5x",
      "factories_add_10x",
      "expand",
      "cic_bank_icon",
      "cic_bank_value",
    ]) {
      expect(out.hidden).toContain(idOf(out.win, name));
    }
    for (const name of ["factories_sub_1x", "factories_add_1x", "minimize"])
      expect(out.hidden).not.toContain(idOf(out.win, name));
  });

  it("puts the list's rows inside its margin", () => {
    const out = constructionWindow(view(), germany, buildings);
    expect(
      matchNamePath(out.win.node, "production_lines/item_grid")[0].position,
    ).toEqual({ x: 14, y: 18 });
  });

  it("prints construction speed as the game does", () => {
    expect(speedText(0.05)).toBe("§G+5%§!");
    expect(speedText(-0.1)).toBe("§R-10%§!");
    expect(speedText(0)).toBe("0%");
  });
});
