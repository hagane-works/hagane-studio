import { describe, expect, it } from "vitest";
import type { GuiNode } from "@/lib/guiRenderApi";
import {
  NAVY_ROSTER,
  ROSTERS,
  newLeaderText,
  opensCommanderDetails,
} from "./CommanderRosterLayer";
import { OPENS_OVER, PANEL_TARGETS, restPlacement } from "./gameNavigation";

describe("newLeaderText", () => {
  it("puts the country's cost where the text leaves the engine's number", () => {
    expect(newLeaderText("New Commander £command_power $VALUE|Y0$", 80)).toBe(
      "New Commander £command_power §Y80§!",
    );
  });
});

describe("the commander roster", () => {
  it("opens details from the row, traits, and promote controls in view mode", () => {
    expect(
      ["leader_button", "traits_btn", "promote_btn"].every(
        opensCommanderDetails,
      ),
    ).toBe(true);
    expect(opensCommanderDetails("close_button")).toBe(false);
  });

  it("opens from the officer corps' Commanders button, over the officer corps", () => {
    expect(PANEL_TARGETS["countryofficercorpview/commander_roster_army"]).toBe(
      "armyleaderwindow",
    );
    expect(OPENS_OVER.armyleaderwindow).toBe("countryofficercorpview");
  });

  it("sits at its roster place, measured from the top left, not its centred one", () => {
    const win = {
      type: "containerwindowtype",
      name: "armyleaderwindow",
      position: { x: -225, y: -295 },
      size: { width: 550, height: 590 },
      orientation: "center",
      raw: {},
      children: [
        {
          type: "positiontype",
          name: "pos_roster_overview_orientation_top_left",
          position: { x: 540, y: 80 },
          children: [],
        },
      ],
    } as unknown as GuiNode;
    expect(restPlacement(win, 1920, 1080)).toMatchObject({ x: 540, y: 80 });
  });
});

describe("the admirals' roster", () => {
  it("opens from the officer corps' Admirals button, over the officer corps, at the same roster place", () => {
    expect(PANEL_TARGETS["countryofficercorpview/commander_roster_navy"]).toBe(
      NAVY_ROSTER,
    );
    expect(OPENS_OVER[NAVY_ROSTER]).toBe("countryofficercorpview");
  });

  it("uses the navy's own row, portrait and four skills", () => {
    expect(ROSTERS[NAVY_ROSTER]).toMatchObject({
      entry: "navyleaderentry",
      kind: "navy",
      portrait: "leader_icon",
    });
    expect(ROSTERS[NAVY_ROSTER].stats).toEqual([
      "attack_skill",
      "defense_skill",
      "maneuvering_skill",
      "coordination_skill",
    ]);
  });
});
