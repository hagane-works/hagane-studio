import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import {
  lineHidden,
  lineTexts,
  percentText,
  productionTexts,
  usedText,
  type GuiQueueLine,
} from "./ProductionQueue";
import { matchNamePath } from "./gameNavigation";

const node = (name: string, type = "instanttextboxtype"): GuiNode =>
  ({
    name,
    type,
    children: [],
    position: { x: 0, y: 0 },
    background: null,
  }) as unknown as GuiNode;

describe("the production queue (hoi4-game-process-en 3:30)", () => {
  it("colours factories in use: gold while some stand idle, green when all work", () => {
    expect(usedText(20, 28)).toBe("§Y20/28§!");
    expect(usedText(10, 10)).toBe("§G10/10§!");
  });

  it("prints the modifiers as the header does", () => {
    expect(percentText(0.174)).toBe("§G+17%§!");
    expect(percentText(0)).toBe("+0%");
    expect(percentText(0.5, false)).toBe("50%");
  });

  it("writes the header's readouts into the window", () => {
    const root = {
      ...node("countryproductionlineview", "containerwindowtype"),
      children: [
        "operational_factories_value",
        "operational_naval_factories_value",
        "production_output_value",
        "efficiency_cap_value",
        "factories_used_equipment",
        "factories_used_other_ships",
      ].map((name) => node(name)),
    } as GuiNode;
    assignIds(root);
    const win = {
      name: "countryproductionlineview",
      node: root,
      sprites: {},
      texts: {},
      fonts: {},
    } as unknown as GuiWindow;
    const texts = productionTexts(win, {
      lines: [],
      military_used: 20,
      military_total: 28,
      naval_used: 10,
      naval_total: 10,
      categories: { infantry: 15, other_ships: 4 },
      modifiers: { production_output: 0.174, efficiency_cap: 0.5 },
    });
    const at = (name: string) =>
      texts[(matchNamePath(root, name)[0] as GuiNode & { _id: string })._id];
    expect(at("operational_factories_value")).toBe("§Y20/28§!");
    expect(at("operational_naval_factories_value")).toBe("§G10/10§!");
    expect(at("production_output_value")).toBe("§G+17%§!");
    expect(at("efficiency_cap_value")).toBe("50%");
    expect(at("factories_used_equipment")).toBe("15");
    expect(at("factories_used_other_ships")).toBe("4");
  });

  const rifles: GuiQueueLine = {
    priority: 1,
    naval: false,
    equipment: "infantry_equipment_1",
    factories: 10,
    requested: 10,
    name: "Karabiner 98k",
    general_name: "Infantry Equipment I",
    speed: "41.09/day",
    stock: 41,
  };
  const subs: GuiQueueLine = {
    priority: 9,
    naval: true,
    equipment: "ship_hull_submarine_1",
    factories: 1,
    requested: 1,
    name: "Type II",
    general_name: "Early Submarine",
    speed: "17 Feb 1936",
    amount: 2,
  };

  it("fills a line's window", () => {
    expect(lineTexts(rifles, "Auto")).toMatchObject({
      priority: "1",
      name: "Karabiner 98k",
      general_name: "Infantry Equipment I",
      speed: "41.09/day",
      factories_used_txt: "10",
      total_need: "41",
    });
    expect(lineTexts(subs, "Auto")).toMatchObject({
      amount: "2",
      deploy_name: "Auto",
      speed: "17 Feb 1936",
    });
    expect(
      lineTexts(
        { ...subs, factories: 0, requested: 1, speed: "§RNone§!" },
        "Auto",
      ).factories_used_txt,
    ).toBe("0/1");
  });

  it("leaves out the buttons by fives and tens, and a ship's name it cannot know", () => {
    expect(lineHidden(rifles)).toContain("factories_add_10x");
    expect(lineHidden(rifles)).not.toContain("total_need");
    expect(lineHidden(subs)).toContain("name_list_btn");
    expect(lineHidden({ ...rifles, stock: undefined })).toContain("total_need");
  });
});
