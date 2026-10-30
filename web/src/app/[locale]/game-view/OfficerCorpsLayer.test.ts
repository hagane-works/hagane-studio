import { describe, it, expect } from "vitest";
import { thumbPlace } from "./PageScrollbar";
import type { GuiNode, GuiSprite, GuiWindow } from "@/lib/guiRenderApi";
import {
  OPENS_OVER,
  RUNTIME_HIDDEN,
  basePanelOf,
  opensOverWith,
  resolveActivation,
} from "./gameNavigation";
import {
  OFFICER_CORPS,
  OFFICER_GRIDS,
  centredOn,
  officerCorpsWindow,
  officerSeats,
  statPlaces,
  type GuiOfficerCorps,
} from "./OfficerCorpsLayer";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";

setReferenceTables(REFERENCE_TABLES);

const beck = {
  token: "GER_ludwig_beck",
  name: "Ludwig Beck",
  slot: "army_chief",
};
const halder = {
  token: "GER_franz_halder",
  name: "Franz Halder",
  slot: "army_chief",
};

describe("officer corps seats", () => {
  it("fills all five grids the screen declares, the high command and theorist too", () => {
    expect(OFFICER_GRIDS.map((grid) => grid.grid)).toEqual([
      "high_command_advisors",
      "theorist_advisors",
      "army_chief_advisors",
      "navy_chief_advisors",
      "air_chief_advisors",
    ]);
  });

  it("puts nobody in a seat they do not hold: 1936 Germany's are all empty plates", () => {
    const corps: GuiOfficerCorps = {
      tag: "GER",
      slots: { army_chief: [beck, halder] },
      seats: { high_command: 3, army_chief: 1 },
      spirits: [],
    };
    expect(officerSeats(corps, "high_command")).toEqual([null, null, null]);
    expect(officerSeats(corps, "army_chief")).toEqual([null]);
  });

  it("seats whoever holds a post first", () => {
    const corps: GuiOfficerCorps = {
      tag: "GER",
      slots: { army_chief: [beck, { ...halder, hired: true }] },
      seats: { army_chief: 1 },
      spirits: [],
    };
    expect(
      officerSeats(corps, "army_chief").map((person) => person?.token),
    ).toEqual(["GER_franz_halder"]);
  });
});

describe("the officer corps", () => {
  it("shows the Army HQ ribbon, the preferred tactic and both rosters, as the designer recording does (1:36)", () => {
    for (const name of [
      "hq_ribbon",
      "preferred_tactic_container",
      "commander_roster_division",
      "commander_roster_ship",
    ]) {
      expect(RUNTIME_HIDDEN[OFFICER_CORPS]).not.toContain(name);
    }
    expect(RUNTIME_HIDDEN[OFFICER_CORPS]).toContain(
      "icon_prefered_tactic_type",
    );
    expect(RUNTIME_HIDDEN[OFFICER_CORPS]).not.toContain("high_command");
    expect(RUNTIME_HIDDEN[OFFICER_CORPS]).not.toContain("theorist");
  });

  it("writes the ribbon's tooltips the way the game does (designe 1:36–1:37)", async () => {
    const { officerCorpsTooltipFor } = await import("./OfficerCorpsLayer");
    const texts = {
      CAPTURED_GENERALS_TOOLTIP:
        "§HCaptured Commanders§!$CLICK_HINTS$\n\nOur missing Commanders: $OWN_COUNT|H$\nCommanders captured by us: $ENEMIES_COUNT|H$",
      CAPTURED_GENERALS_TOOLTIP_CLICK:
        "\n§GClick§! to see our missing Commanders",
      CAPTURED_GENERALS_TOOLTIP_RIGHT_CLICK:
        "\n§GRight-click§! to see Commanders captured by us",
      OFFICER_CORP_PREFERRED_TACTICS_SELECTION: "Preferred Tactics Selection\n",
      OFFICER_CORP_NO_PREFERRED_TACTIC: "Currently no tactic is selected.",
    };
    expect(
      officerCorpsTooltipFor({ name: "button_captured_generals" }, texts),
    ).toBe(
      "§HCaptured Commanders§!\n§GClick§! to see our missing Commanders\n§GRight-click§! to see Commanders captured by us" +
        "\n\nOur missing Commanders: §H0§!\nCommanders captured by us: §H0§!",
    );
    expect(
      officerCorpsTooltipFor(
        { name: "button_prefered_tactic_selection" },
        texts,
      ),
    ).toBe("Preferred Tactics Selection\nCurrently no tactic is selected.");
  });

  it("scales a stamped window as its view does", async () => {
    const { scaledTree } = await import("./OfficerCorpsLayer");
    const node = {
      type: "containerwindowtype",
      name: "army_spirit_entry",
      position: { x: 59, y: -56 },
      size: { width: 150, height: 66 },
      children: [
        {
          type: "buttontype",
          name: "button",
          sprite: "GFX_officer_corp_spirit_bg_enabled",
          position: { x: 94, y: 46 },
          children: [],
        },
      ],
    } as unknown as GuiNode;
    const scaled = scaledTree(node, 0.9);
    expect(scaled.position).toEqual({ x: 53.1, y: -50.4 });
    expect(scaled.size).toEqual({ width: 135, height: 59.4 });
    expect(scaled.children?.[0].scale).toBe(0.9);
  });

  const screen = () =>
    ({
      type: "containerwindowtype",
      name: OFFICER_CORPS,
      size: { width: 550 },
      children: [
        {
          type: "containerwindowtype",
          name: "officer_core_view",
          position: { x: 0, y: 173 },
          raw: { margin: { top: 13, left: 13, right: 13 } },
          children: [
            {
              type: "positiontype",
              name: "no_taog_offset",
              position: { x: 0, y: -57 },
              children: [],
            },
            {
              type: "containerwindowtype",
              name: "army_header",
              position: { x: 0, y: 5 },
              size: { width: "98%", height: 200 },
              children: [
                {
                  type: "containerwindowtype",
                  name: "land_doctrine_button",
                  position: { x: 260, y: -11 },
                  children: [
                    {
                      type: "icontype",
                      name: "icon",
                      sprite: "GFX_land_doctrine_medium",
                      children: [],
                    },
                  ],
                },
              ],
            },
          ],
        },
      ],
    }) as unknown as GuiNode;

  it("without Army HQs, moves the commands up by the .gui's no_taog_offset and into the view's margins", () => {
    const node = screen();
    const moved = officerCorpsWindow(
      { name: OFFICER_CORPS, node, sprites: {} } as unknown as GuiWindow,
      undefined,
      false,
    );
    const view = moved.node.children?.[0];
    expect(view?.position).toEqual({ x: 0, y: 116 });

    const army = view?.children?.[1];
    expect(army?.position).toEqual({ x: 13, y: 18 });
    expect(army?.size?.width).toBe(514);

    expect(node.children?.[0].position).toEqual({ x: 0, y: 173 });
  });

  it("puts the branch's grand doctrine in its ring", () => {
    const picture = {
      width: 64,
      height: 64,
      frames: 1,
      resolved: true,
    } as GuiSprite;
    const corps: GuiOfficerCorps = {
      tag: "GER",
      slots: {},
      spirits: [],
      doctrines: {
        land: {
          token: "new_mobile_warfare",
          icon: "GFX_doctrine_mobile_warfare_medium",
          icon_sprite: picture,
        },
      },
    };
    const moved = officerCorpsWindow(
      {
        name: OFFICER_CORPS,
        node: screen(),
        sprites: {},
      } as unknown as GuiWindow,
      corps,
    );
    const ring = moved.node.children?.[0].children?.[1].children?.[0];
    expect(ring?.children?.[0].sprite).toBe(
      "GFX_doctrine_mobile_warfare_medium",
    );
    expect(moved.sprites["GFX_doctrine_mobile_warfare_medium"]).toBe(picture);

    const plain = officerCorpsWindow({
      name: OFFICER_CORPS,
      node: screen(),
      sprites: {},
    } as unknown as GuiWindow);
    expect(
      plain.node.children?.[0].children?.[1].children?.[0].children?.[0].sprite,
    ).toBe("GFX_land_doctrine_medium");
  });

  it("centres the empty-seat card where the seat's plate was", () => {
    const plate = { width: 61, height: 83, frames: 1 } as GuiSprite;
    const card = { width: 42, height: 54, frames: 1 } as GuiSprite;
    expect(centredOn({ x: -30, y: 50 }, plate, card)).toEqual({
      x: -20,
      y: 65,
    });
  });
});

describe("the advisor picker over the officer corps", () => {
  const close = { name: "close_button" } as GuiNode;

  it("opens over the politics screen unless a seat elsewhere opened it", () => {
    expect(OPENS_OVER.political_ideas_window).toBe("countrypoliticsview");
    expect(basePanelOf("political_ideas_window", opensOverWith(null))).toBe(
      "countrypoliticsview",
    );
    expect(
      basePanelOf("political_ideas_window", opensOverWith(OFFICER_CORPS)),
    ).toBe(OFFICER_CORPS);
  });

  it("closes back to the officer corps it opened over", () => {
    const over = opensOverWith(OFFICER_CORPS);
    expect(
      resolveActivation(
        "political_ideas_window",
        close,
        "political_ideas_window",
        over,
      ),
    ).toEqual({
      kind: "open",
      window: OFFICER_CORPS,
    });
  });
});

describe("the special forces ring", () => {
  const header = () =>
    ({
      type: "containerwindowtype",
      name: OFFICER_CORPS,
      size: { width: 550 },
      children: [
        {
          type: "containerwindowtype",
          name: "special_forces_header",
          children: [
            {
              type: "containerwindowtype",
              name: "special_forces_doctrine_button",
              children: [],
            },
          ],
        },
      ],
    }) as unknown as GuiNode;

  it("waits for a special forces doctrine: the recording's Germany could take one but shows no ring", () => {
    const none = officerCorpsWindow({
      name: OFFICER_CORPS,
      node: header(),
      sprites: {},
    } as unknown as GuiWindow);
    expect(none.node.children?.[0].children).toEqual([]);
    const picture = {
      width: 1,
      height: 1,
      frames: 1,
      resolved: true,
    } as GuiSprite;
    const corps: GuiOfficerCorps = {
      tag: "GER",
      slots: {},
      spirits: [],
      doctrines: {
        special_forces: {
          token: "special_forces_quality",
          icon: "GFX_sf",
          icon_sprite: picture,
        },
      },
    };
    const chosen = officerCorpsWindow(
      {
        name: OFFICER_CORPS,
        node: header(),
        sprites: {},
      } as unknown as GuiWindow,
      corps,
    );
    expect(
      chosen.node.children?.[0].children?.map((child) => child.name),
    ).toEqual(["special_forces_doctrine_button"]);
  });
});

describe("the stat rows", () => {
  it("fill the grid left to right, seven to a row, at its slot size", () => {
    const grid = {
      slotsize: { width: 72, height: 33 },
      max_slots_horizontal: 7,
    };
    const places = statPlaces(10, grid);
    expect(places[0]).toEqual({ x: 0, y: 0 });
    expect(places[6]).toEqual({ x: 432, y: 0 });
    expect(places[7]).toEqual({ x: 0, y: 33 });
    expect(places[9]).toEqual({ x: 144, y: 33 });
  });
});

describe("the page scrollbar's thumb", () => {
  it("is as long as the part of the page on screen, and moves along the track", () => {
    expect(thumbPlace(800, 800, 0, 0)).toEqual({ size: 800, offset: 0 });
    expect(thumbPlace(800, 800, 200, 0)).toEqual({ size: 640, offset: 0 });
    expect(thumbPlace(800, 800, 200, 200)).toEqual({ size: 640, offset: 160 });
    expect(thumbPlace(800, 800, 200, 100)).toEqual({ size: 640, offset: 80 });
  });
});
