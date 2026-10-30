import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { warGoalValues } from "./WarGoalEditPanel";
import {
  NO_WARGOAL,
  pickType,
  toggleState,
  warGoalCost,
  warGoalCostText,
  warGoalWindow,
  type GuiWarGoals,
} from "./WarGoalLayer";
import { OPENS_OVER, matchNamePath } from "./gameNavigation";

const node = (name: string, type: string, children: GuiNode[] = []): GuiNode =>
  ({ name, type, children }) as GuiNode;

const view = (): GuiWindow =>
  ({
    name: "diplomacy_action_info_window",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("diplomacy_action_info_window", "containerwindowtype", [
      node("generate_wargoal", "containerwindowtype", [
        node("title", "instanttextboxtype"),
        node("wargoal_warning", "instanttextboxtype"),
        node("wargoal_name", "instanttextboxtype"),
        node("description", "instanttextboxtype"),
        node("threat", "containerwindowtype", [
          node("threat_value", "instanttextboxtype"),
        ]),
        node("wargoals", "containerwindowtype", [
          node("wargoals_grid", "gridboxtype"),
        ]),
      ]),
      node("add_wargoal", "containerwindowtype", [
        node("wargoal_name", "instanttextboxtype"),
        node("wargoals", "containerwindowtype", [
          node("wargoals_grid", "gridboxtype"),
        ]),
      ]),
      node("send_volunteers", "containerwindowtype"),
    ]),
  }) as unknown as GuiWindow;

const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

const conquer = {
  id: "take_state",
  label: "Conquer",
  base_cost: 200,
  per_state_cost: 50,
  limit: 5,
  threat: 2,
  states: [
    { id: 10, name: "Warszawa" },
    { id: 85, name: "Danzig" },
    { id: 86, name: "Poznań" },
    { id: 87, name: "Łódz" },
    { id: 88, name: "Kraków" },
    { id: 89, name: "Kielce" },
  ],
};

const goals: GuiWarGoals = {
  tag: "GER",
  target: "POL",
  types: [conquer],
  daily_pp: 0.2,
  texts: {
    title: "Justify War Goal",
    no_wargoal: "No war goal selected",
    select_type: "Select a war goal type",
    cost: "War goal will cost you $DAILYCOST|Y$ Political Power and last $DAYS|Y$ days",
  },
};

describe("the Justify War Goal dialog (intel_end 1:06–1:11)", () => {
  it("costs as recorded: one state 250 days and 50.00, four 400 and 80.00", () => {
    expect(warGoalCost(conquer, 1, 0.2)).toEqual({ days: 250, pp: 50 });
    expect(warGoalCost(conquer, 4, 0.2)).toEqual({ days: 400, pp: 80 });
    expect(warGoalCostText(goals.texts.cost, 400, 80)).toBe(
      "War goal will cost you §Y80.00§! Political Power and last §Y400§! days",
    );
    expect(warGoalCostText(undefined, 400, 80)).toBe("");
  });

  it("picks the first state with the kind, adds up to the limit and keeps the last", () => {
    const one = pickType(goals, "take_state");
    expect(one).toEqual({ type: "take_state", states: [10] });
    let pick = one;
    for (const id of [85, 86, 87, 88, 89])
      pick = toggleState(pick, conquer, id);
    expect(pick.states).toEqual([10, 85, 86, 87, 88]);
    expect(toggleState(pick, conquer, 85).states).toEqual([10, 86, 87, 88]);
    expect(toggleState(one, conquer, 10)).toBe(one);
  });

  it("writes the dialog's texts into its own boxes and puts the other dialogs aside", () => {
    const empty = warGoalWindow(view(), goals, NO_WARGOAL);
    const id = (out: typeof empty, path: string) =>
      idOf(matchNamePath(out.win.node, path)[0]);
    expect(empty.hidden).toContain(id(empty, "add_wargoal"));
    expect(empty.hidden).toContain(id(empty, "send_volunteers"));
    expect(empty.hidden).not.toContain(id(empty, "generate_wargoal"));
    expect(empty.texts[id(empty, "generate_wargoal/wargoal_warning")]).toBe(
      "No war goal selected",
    );
    expect(empty.texts[id(empty, "generate_wargoal/wargoal_name")]).toBe(
      "Select a war goal type",
    );
    expect(empty.texts[id(empty, "generate_wargoal/threat/threat_value")]).toBe(
      "0%",
    );
    expect(empty.texts[id(empty, "add_wargoal/wargoal_name")]).toBeUndefined();

    const four = warGoalWindow(view(), goals, {
      type: "take_state",
      states: [10, 85, 86, 87],
    });
    expect(four.texts[id(four, "generate_wargoal/wargoal_name")]).toBe(
      "Conquer Warszawa, Danzig, Poznań, Łódz",
    );
    expect(four.texts[id(four, "generate_wargoal/description")]).toBe(
      "War goal will cost you §Y80.00§! Political Power and last §Y400§! days",
    );
    expect(four.texts[id(four, "generate_wargoal/wargoal_warning")]).toBe("");
  });

  it("opens over the diplomacy panel from Justify War goal", () => {
    expect(OPENS_OVER.diplomacy_action_info_window).toBe(
      "countrydiplomacyview",
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain('if (id !== "generate_wargoal") return;');
    expect(page).toContain("warGoalWindow(win, warGoals.data, warGoalPick)");
  });
});

describe("a kind of war goal's edit panel", () => {
  it("sends only the numbers that changed and refuses what is not one", () => {
    expect(warGoalValues(conquer, {})).toEqual({});
    expect(
      warGoalValues(conquer, { generate_base_cost: "150", threat: "2" }),
    ).toEqual({ generate_base_cost: 150 });
    expect(warGoalValues(conquer, { take_states_limit: "" })).toBeNull();
    expect(warGoalValues(conquer, { threat: "-1" })).toBeNull();
    expect(warGoalValues(conquer, { generate_per_state_cost: "x" })).toBeNull();
  });

  it("opens from a kind's row in edit mode, the diplomacy edit panel giving way", () => {
    const layer = readFileSync(join(__dirname, "WarGoalLayer.tsx"), "utf8");
    expect(layer).toContain('label={tw("hit", { name: entry.label })}');
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "{editMode && !countryPanelOpen && panel === WARGOAL_WINDOW && editedWarGoal && !selection && (",
    );
    expect(page).toContain(
      "diplomacyPanelOpen && panel !== WARGOAL_WINDOW && !selection",
    );
    const panel = readFileSync(join(__dirname, "WarGoalEditPanel.tsx"), "utf8");
    expect(panel).toContain("/gui-render/wargoal-save");
  });

  it("shows the war goal workbench's own editor for the project's own war goal", () => {
    const workbench = readFileSync(
      join(__dirname, "..", "project", "[id]", "wargoals", "page.tsx"),
      "utf8",
    );
    expect(workbench).toContain("<WarGoalEditor");
    const panel = readFileSync(join(__dirname, "WarGoalEditPanel.tsx"), "utf8");
    expect(panel).toContain("<WarGoalEditor");
    expect(panel).toContain('wide={row ? "workbench" : false}');
  });
});
