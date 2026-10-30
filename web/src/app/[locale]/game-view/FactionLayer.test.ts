import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  applyMargins,
  factionTabOf,
  factionWindow,
  goalCard,
  nestedSlot,
  type GuiFaction,
} from "./FactionLayer";
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

const tabs = ["goals", "rules", "research", "army", "intelligence"].flatMap(
  (tab) => [
    node(`${tab}_tab_button`, "buttontype", [], {
      frame: tab === "research" ? 1 : 2,
    }),
    node(`${tab}_tab_button_icon`, "icontype"),
  ],
);

const view = (): GuiWindow =>
  ({
    name: "countryfactionview",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("countryfactionview", "containerwindowtype", [
      node("header_section", "containerwindowtype", [
        node("faction_icon", "icontype"),
        node("faction_name", "instanttextboxtype"),
        node("manifest_name", "instanttextboxtype"),
        node("manifest_progress", "icontype"),
        node("power_projection_value", "instanttextboxtype"),
      ]),
      node("faction_initiative", "containerwindowtype", [
        node("faction_initiative_value", "instanttextboxtype"),
      ]),
      node("tabs", "containerwindowtype", tabs),
      node(
        "tab_content_parent_view",
        "containerwindowtype",
        [
          node("goals_section", "containerwindowtype", [], {
            position: { x: 0, y: 10 },
          }),
          node("rules_section", "containerwindowtype"),
          node("research_section", "containerwindowtype"),
          node("army_section", "containerwindowtype"),
          node("intelligence_section", "containerwindowtype", [
            node("intelligence_sizer", "containerwindowtype", [
              node("header_window", "containerwindowtype", [
                node("intelligence_sharing_bonus", "instanttextboxtype"),
              ]),
              node("opacity_window", "containerwindowtype", [], {
                position: { x: 14, y: 38 },
              }),
            ]),
          ]),
        ],
        { raw: { margin: { top: 15, left: 0 } } },
      ),
    ]),
  }) as unknown as GuiWindow;

const axis: GuiFaction = {
  tag: "GER",
  in_faction: true,
  name: "Axis",
  icon: "GFX_faction_logo_axis",
  manifest: "Conquest and Domination",
  initiative: 0,
  rank_text: "Global Rank: §H4§!",
  intel: { sharing_bonus: "§H0%§!" },
};

const idOf = (win: GuiWindow, path: string) =>
  (matchNamePath(win.node, path)[0] as GuiNode & { _id?: string })._id ?? "";

describe("the faction screen (hoi4-game-process-en 1:51–2:00)", () => {
  it("opens from the topbar and switches tabs on their buttons", () => {
    expect(factionTabOf("intelligence_tab_button")).toBe("intelligence");
    expect(factionTabOf("goals_tab_button")).toBe("goals");
    expect(factionTabOf("intelligence_tab_button_icon")).toBeNull();
    expect(factionTabOf(null)).toBeNull();
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("factionWindow(win, faction.data, factionTab)");
    expect(page).toContain("<FactionLayer");
  });

  it("lights the open tab (frame 1) and shows only its section", () => {
    const out = factionWindow(view(), axis, "intelligence");
    expect(
      matchNamePath(out.win.node, "tabs/intelligence_tab_button")[0].frame,
    ).toBe(1);
    expect(
      matchNamePath(out.win.node, "tabs/intelligence_tab_button_icon")[0].frame,
    ).toBe(1);

    expect(
      matchNamePath(out.win.node, "tabs/research_tab_button")[0].frame,
    ).toBe(2);
    expect(out.hidden).toContain(
      idOf(out.win, "tab_content_parent_view/goals_section"),
    );
    expect(out.hidden).not.toContain(
      idOf(out.win, "tab_content_parent_view/intelligence_section"),
    );
  });

  it("writes the header, the sharing bonus, and takes the shade out to draw it over the positions", () => {
    const out = factionWindow(view(), axis, "intelligence");
    expect(out.texts[idOf(out.win, "header_section/faction_name")]).toBe(
      "Axis",
    );
    expect(out.texts[idOf(out.win, "header_section/manifest_name")]).toBe(
      "§HConquest and Domination§!",
    );
    expect(
      out.texts[idOf(out.win, "header_section/power_projection_value")],
    ).toBe("Global Rank: §H4§!");
    expect(
      out.texts[idOf(out.win, "intelligence_sizer/intelligence_sharing_bonus")],
    ).toBe("§H0%§!");
    expect(out.hidden).toContain(
      idOf(out.win, "intelligence_sizer/opacity_window"),
    );

    const goals = factionWindow(view(), axis, "goals");
    expect(goals.hidden).not.toContain(
      idOf(goals.win, "intelligence_sizer/opacity_window"),
    );
  });

  it("insets a container's children by its margin, as the game does", () => {
    const root = view().node;
    applyMargins(root);
    expect(
      matchNamePath(root, "tab_content_parent_view/goals_section")[0].position,
    ).toEqual({ x: 0, y: 25 });
  });

  it("centres a goal's name on its label's line", () => {
    const card = {
      name: "goal_item_container",
      sprites: {},
      texts: {},
      fonts: {},
      node: node("goal_item_container", "containerwindowtype", [
        node("assigned_goal_item_container", "containerwindowtype", [
          node("label", "instanttextboxtype", [], {
            position: { x: 10, y: 38 },
            maxheight: 60,
          }),
        ]),
      ]),
    } as unknown as GuiWindow;
    const label = matchNamePath(
      goalCard(card).node,
      "assigned_goal_item_container/label",
    )[0];
    expect(label.position).toEqual({ x: 10, y: 8 });
    expect(label.raw?.vertical_alignment).toBe("center");
  });

  it("puts a slot one level down, its backdrop kept, the empty card at box_list", () => {
    const slot = {
      name: "intelligence_advisor_slot_spymaster",
      sprites: {},
      texts: {},
      fonts: {},
      node: node(
        "intelligence_advisor_slot_spymaster",
        "containerwindowtype",
        [
          node("operative_title", "instanttextboxtype"),
          node("box_list", "gridboxtype", [], { position: { x: 0, y: 45 } }),
        ],
        {
          position: { x: 0, y: 16 },
          background: node("background", "background", [], {
            position: { x: 0, y: 42 },
          }),
        },
      ),
    } as unknown as GuiWindow;
    const out = nestedSlot(slot, "GFX_idea_slot_political_advisor");
    expect(out.node.background).toBeNull();
    const inner = out.node.children[0];
    expect(inner.name).toBe("intelligence_advisor_slot_spymaster");
    expect(inner.background?.position).toEqual({ x: 0, y: 42 });
    expect(inner.children.map((child) => child.name)).toEqual([
      "operative_title",
      "box_list",
      "vacant_portrait",
    ]);
    const card = inner.children[2];
    expect(card.sprite).toBe("GFX_idea_slot_political_advisor");
    expect(card.position).toEqual({ x: 0, y: 45 });

    expect(nestedSlot(slot).node.children[0].children).toHaveLength(2);
  });
});
