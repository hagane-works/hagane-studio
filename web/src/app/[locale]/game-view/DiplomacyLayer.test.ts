import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "../gui-renderer/GuiCanvas";
import { diplomacyListHidden } from "./DiplomacyLayer";

const source = readFileSync(join(__dirname, "DiplomacyLayer.tsx"), "utf8");
const navigation = readFileSync(join(__dirname, "gameNavigation.ts"), "utf8");

describe("diplomacy layer", () => {
  it("stacks the game's own row into the country list", () => {
    expect(source).toContain(
      'const ROW = "diplomacy_country_list_country_entry"',
    );
    expect(source).toContain("const ROW_H = 45");
  });

  it("says which faction a country stands in, since that is the file's answer", () => {
    expect(source).toContain("country.faction ?");
  });

  it("leaves opinion alone", () => {
    expect(navigation).toContain('"our_opinion_value", "their_opinion_value"');
    expect(source).toContain('child.name === "our_opinion"');
  });

  it("blanks the header placeholders it paints over", () => {
    expect(navigation).toContain(
      '"country_name", "faction_name", "leader_name"',
    );
    expect(navigation).toContain('"stability_value", "war_support_value"');
  });

  it("names an ideology badge the way the game does", () => {
    expect(source).toContain("`GFX_ideology_${party}_group`");
  });
});

describe("diplomacy list", () => {
  const el = (name: string, children: GuiNode[] = []): GuiNode =>
    ({
      type: "containerwindowtype",
      name,
      position: { x: 0, y: 0 },
      size: null,
      children,
    }) as unknown as GuiNode;

  it("hides an open country's bottom bar and buttons, which now land on the screen", () => {
    const root = el("countrydiplomacyview", [
      el("country_info", [
        el("diplomacy_bottom"),
        el("back_button"),
        el("logistics_button"),
        el("relations_info"),
      ]),
      el("country_list", [el("countries")]),
    ]);
    assignIds(root);
    const win = {
      name: "countrydiplomacyview",
      node: root,
      sprites: {},
    } as unknown as GuiWindow;
    const id = (node: GuiNode) => (node as GuiNode & { _id: string })._id;
    const info = root.children[0];
    expect(diplomacyListHidden(win).sort()).toEqual(
      [id(info.children[0]), id(info.children[1]), id(info.children[2])].sort(),
    );
  });

  it("ends the list inside its box, less the box's bottom margin", () => {
    expect(source).toContain('const listBox = rects["name:countries"]');
    expect(source).toContain("height: Math.max(0, listBottom - grid.y)");
  });
});
