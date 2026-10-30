import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiAlert, GuiNode } from "@/lib/guiRenderApi";
import {
  VANILLA_ALERT_POSITIONS,
  alertEditNode,
  alertSlots,
  alertTooltip,
  alertsShown,
} from "./AlertsLayer";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";

setReferenceTables(REFERENCE_TABLES);

describe("an alert in UI 编辑", () => {
  const template = {
    type: "containerwindowtype",
    name: "global_alerticon_window",
    children: [
      { type: "buttontype", name: "icon", sprite: "GFX_global_alert_icons" },
      {
        type: "icontype",
        name: "red_alert_glow",
        sprite: "GFX_alert_red_glow",
      },
      {
        type: "icontype",
        name: "yellow_alert_glow",
        sprite: "GFX_alert_yellow_glow",
      },
    ],
  } as unknown as GuiNode;

  it("is the template on the alert's own frame, with only its category's glow", () => {
    const high = alertEditNode(template, 12, true);
    expect(high.children?.map((c) => c.name)).toEqual([
      "icon",
      "red_alert_glow",
    ]);
    expect(high.children?.[0].frame).toBe(12);
    const other = alertEditNode(template, 43, false);
    expect(other.children?.map((c) => c.name)).toEqual([
      "icon",
      "yellow_alert_glow",
    ]);

    expect(template.children?.length).toBe(3);
    expect(template.children?.[0].frame).toBeUndefined();
  });

  it("is drawn on a pickable canvas in UI 编辑 and does not open the alert's screen", () => {
    const layer = readFileSync(join(__dirname, "AlertsLayer.tsx"), "utf8");
    expect(layer).toContain(
      "const { editMode: uiEdit, swaps } = useContext(EditContext);",
    );
    expect(layer).toContain("if (uiEdit) {");
    expect(layer).toContain("<AlertEditTile");
    expect(layer).toContain(
      "{...edit(`${ALERT_TEMPLATE}:${alert.type ?? alert.name}:${alert.frame}`, win, node, win.sprites)}",
    );
  });

  it("gives every visible alert a response even when its destination is not implemented", () => {
    const layer = readFileSync(join(__dirname, "AlertsLayer.tsx"), "utf8");
    expect(layer).toContain("onUnavailable?: (control: string) => void");
    expect(layer).toContain("? () => onUnavailable(alert.name)");
  });
});

describe("alertSlots", () => {
  it("starts the row where the game draws its first icon and steps 48 px", () => {
    const slots = alertSlots(3, VANILLA_ALERT_POSITIONS, 1924, 43);

    expect(slots).toEqual([
      { x: 718, y: 38 },
      { x: 766, y: 38 },
      { x: 814, y: 38 },
    ]);
  });

  it("breaks the row before an icon would cross alerticon_endposition from the right edge", () => {
    const slots = alertSlots(70, VANILLA_ALERT_POSITIONS, 1924, 43);
    expect(slots[17]).toEqual({ x: 1534, y: 38 });
    expect(slots[18]).toEqual({ x: 718, y: 82 });
    const rows = new Map<number, number>();
    for (const slot of slots) rows.set(slot.y, (rows.get(slot.y) ?? 0) + 1);
    expect([...rows.values()]).toEqual([18, 18, 18, 16]);
  });

  it("uses the plain start position for a bar without the extended background", () => {
    expect(alertSlots(1, VANILLA_ALERT_POSITIONS, 1924, 43, false)[0]).toEqual({
      x: 662,
      y: 38,
    });
  });
});

describe("alertsShown", () => {
  const slotsFor = (count: number) =>
    alertSlots(count + 1, VANILLA_ALERT_POSITIONS, 1920, 40);

  it("folds to the first line, its last slot the toggle that counts the rest", () => {
    const slots = slotsFor(76);
    const firstLine = slots.filter((slot) => slot.y === slots[0].y).length;
    const view = alertsShown(slots, 76, false);
    expect(view.shown).toBe(firstLine - 1);
    expect(view.hidden).toBe(76 - (firstLine - 1));
    expect(view.toggle).toEqual(slots[firstLine - 1]);
  });

  it("unfolded, draws every alert and puts the toggle after the last", () => {
    const slots = slotsFor(76);
    expect(alertsShown(slots, 76, true)).toEqual({
      shown: 76,
      toggle: slots[76],
      hidden: 0,
    });
  });

  it("needs no toggle when every alert fits on the first line", () => {
    expect(alertsShown(slotsFor(3), 3, false)).toEqual({
      shown: 3,
      toggle: null,
      hidden: 0,
    });
    expect(alertsShown(slotsFor(3), 3, true)).toEqual({
      shown: 3,
      toggle: null,
      hidden: 0,
    });
  });
});

describe("alertTooltip", () => {
  const research: GuiAlert = {
    name: "alert_no_research",
    frame: 1,
    instant: "§YResearch Slot Available§!",
    delayed:
      "You have $COUNT|Y$ Research Slot(s) available.\n§GRight-click to dismiss§!",
  };

  it("shows the instant text and fills the free research slots into the delayed one", () => {
    expect(alertTooltip(research, { tag: "GER", research_slots: 4 })).toEqual({
      text: "§YResearch Slot Available§!",
      delayed:
        "You have §Y4§! Research Slot(s) available.\n§GRight-click to dismiss§!",
    });
  });

  it("leaves other run-time counts as slots, for the tooltip to drop", () => {
    const decisions: GuiAlert = {
      name: "alert_decision_new",
      frame: 43,
      instant: "§HDecisions Available§!",
      delayed:
        "Next decisions are ready to be taken:\n$LIST$§GRight-click to dismiss§!",
    };
    expect(
      alertTooltip(decisions, { tag: "GER", research_slots: 4 }).delayed,
    ).toBe(decisions.delayed);
  });
});
