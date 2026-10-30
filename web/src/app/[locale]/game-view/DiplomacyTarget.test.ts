import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  actionCostText,
  diplomacyTargetWindow,
  type GuiDiplomacy,
} from "./DiplomacyLayer";
import { matchNamePath } from "./gameNavigation";

const node = (name: string, type: string, children: GuiNode[] = []): GuiNode =>
  ({ name, type, children }) as GuiNode;

const view = (): GuiWindow =>
  ({
    name: "countrydiplomacyview",
    sprites: {},
    texts: {},
    fonts: {},
    node: node("countrydiplomacyview", "containerwindowtype", [
      node("country_info", "containerwindowtype", [
        node("info_tab_button", "buttontype"),
        node("diplomacy_tab_top", "containerwindowtype", [
          node("active_national_focus_info", "containerwindowtype", [
            node("national_focus_label", "instanttextboxtype"),
            node("progress", "icontype"),
            node("progress_frame", "icontype"),
          ]),
        ]),
        node("ideas_info", "containerwindowtype"),
        node("trade_info", "containerwindowtype"),
      ]),
      node("player_info_box", "containerwindowtype"),
      node("country_list", "containerwindowtype"),
      node("intel_ledger_container", "containerwindowtype"),
    ]),
  }) as unknown as GuiWindow;

const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

describe("another country's diplomacy panel (intel_end 1:04)", () => {
  it("prints an action's cost as its row does", () => {
    expect(
      actionCostText({
        id: "guarantee",
        label: "Guarantee independence",
        pp: 15,
      }),
    ).toBe("£pol_power 15");
    expect(
      actionCostText({
        id: "send_attache",
        label: "Send attaché",
        pp: 100,
        cp: 50,
      }),
    ).toBe("£command_power 50 £pol_power 100");
    expect(actionCostText({ id: "declare_war", label: "Declare war" })).toBe(
      "",
    );
  });

  it("puts the list aside, names the second tab the intel ledger and shows the focus box as F2 does", () => {
    const target = {
      tag: "POL",
      self: { tag: "POL" },
      countries: [],
      texts: {
        intel_ledger: "Intel Ledger",
        unknown_focus: "Unknown Focus",
        select_focus: "Select a National Focus",
      },
    } as GuiDiplomacy;
    const out = diplomacyTargetWindow(view(), target);
    const id = (path: string) => idOf(matchNamePath(out.win.node, path)[0]);
    for (const path of [
      "country_list",
      "player_info_box",
      "ideas_info",
      "trade_info",
      "intel_ledger_container",
    ]) {
      expect(out.hidden).toContain(id(path));
    }

    expect(out.hidden).not.toContain(id("active_national_focus_info/progress"));
    expect(out.hidden).not.toContain(id("diplomacy_tab_top"));
    expect(out.texts[id("info_tab_button")]).toBe("Intel Ledger");
    expect(out.texts[id("national_focus_label")]).toBe(
      "Select a National Focus",
    );

    const older = diplomacyTargetWindow(view(), {
      ...target,
      texts: { unknown_focus: "Unknown Focus" },
    });
    expect(
      older.texts[
        idOf(matchNamePath(older.win.node, "national_focus_label")[0])
      ],
    ).toBe("Unknown Focus");
  });

  it("opens from a row of the list and goes back with Open Country List", () => {
    const layer = readFileSync(join(__dirname, "DiplomacyLayer.tsx"), "utf8");
    expect(layer).toContain(
      "onOpen={onTarget ? () => onTarget(country.tag) : undefined}",
    );
    expect(layer).toContain('aria-label="Open Country List"');
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("diplomacyTargetWindow(win, viewedDiplomacy.data)");
  });
});
