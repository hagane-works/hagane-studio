import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { GuiNode } from "@/lib/guiRenderApi";
import {
  ALERT_TARGETS,
  DECELERATED,
  RUNTIME_HIDDEN,
  START_CLOCK,
  TOPBAR_SPRITE_SWAPS,
  TOPBAR_TARGETS,
  cubicBezier,
  hotkeyOf,
  nextClock,
  resolveActivation,
  movesBetweenScreens,
  restPlacement,
  shortcutsOf,
  topbarButtonFor,
  topbarControlOf,
} from "./gameNavigation";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "buttontype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    ...partial,
  };
}

describe("resolveActivation", () => {
  it("opens the panel a topbar button names", () => {
    expect(
      resolveActivation("topbar", node({ name: "technology_button" }), null),
    ).toEqual({
      kind: "open",
      window: "countrytechnologyview",
    });
  });

  it("switches panels directly, as the game does, without closing first", () => {
    expect(
      resolveActivation(
        "topbar",
        node({ name: "production_button" }),
        "countrypoliticsview",
      ),
    ).toEqual({
      kind: "open",
      window: "countryproductionlineview",
    });
  });

  it("toggles the open panel closed when its own button is pressed again", () => {
    expect(
      resolveActivation(
        "topbar",
        node({ name: "player_flag" }),
        "countrypoliticsview",
      ),
    ).toEqual({
      kind: "close",
    });
  });

  it("opens the focus tree from the politics panel's focus button", () => {
    expect(
      resolveActivation(
        "countrypoliticsview",
        node({ name: "add_national_goal_button" }),
        "countrypoliticsview",
      ),
    ).toEqual({ kind: "open", window: "nationalfocusview" });
  });

  it("opens the faction window from every politics-screen faction entry point", () => {
    for (const name of [
      "new_faction_button",
      "view_factions_button",
      "open_faction_button",
    ]) {
      expect(
        resolveActivation(
          "countrypoliticsview",
          node({ name }),
          "countrypoliticsview",
        ),
      ).toEqual({
        kind: "open",
        window: "countryfactionview",
      });
    }
  });

  it.each([
    ["countrypoliticsview", "manage_occupied_button", "countryoccupationview"],
    [
      "countrypoliticsview",
      "show_exiles_collaborations_button",
      "exiles_window",
    ],
    ["countrypoliticsview", "show_power_balance", "powerbalanceview"],
    [
      "countryinternationalmarketview",
      "diplo_view_button",
      "countrydiplomacyview",
    ],
    ["countrydiplomacyview", "show_national_goal_button", "nationalfocusview"],
    [
      "countryintelligenceagencyview",
      "agency_logo",
      "name_logo_agency_selection_window",
    ],
    [
      "countryproductionlineview",
      "infantry_button",
      "production_equipment_window_military",
    ],
    [
      "countryproductionlineview",
      "tank_button",
      "production_equipment_window_military",
    ],
    [
      "countryproductionlineview",
      "aircraft_button",
      "production_equipment_window_military",
    ],
    [
      "countryproductionlineview",
      "naval_button",
      "production_equipment_window",
    ],
    ["countryarmyview", "details_open", "equipment_overview_window"],
    ["countryarmyview", "naval_hq_open", "building_roster_window"],
    ["countryarmyview", "destroyed_ships_open", "navallosseswindow"],
    [
      "countryofficercorpview",
      "commander_roster_division",
      "divisioncommanderwindow",
    ],
    [
      "countryofficercorpview",
      "button_captured_generals",
      "capturedarmyleaderwindow",
    ],
  ])("opens the known %s/%s destination", (owner, button, window) => {
    expect(resolveActivation(owner, node({ name: button }), owner)).toEqual({
      kind: "open",
      window,
    });
  });

  it("opens the standalone achievements window from its topbar trophy", () => {
    expect(
      resolveActivation("topbar", node({ name: "achievements_button" }), null),
    ).toEqual({
      kind: "open",
      window: "achievements_window",
    });
  });

  it("closes on a panel's close button, recognised by name or by its ESCAPE shortcut", () => {
    expect(
      resolveActivation(
        "countrytechnologyview",
        node({ name: "close_button" }),
        "countrytechnologyview",
      ),
    ).toEqual({
      kind: "close",
    });
    expect(
      resolveActivation(
        "nationalfocusview",
        node({ name: "x", raw: { shortcut: "ESCAPE" } }),
        "nationalfocusview",
      ),
    ).toEqual({ kind: "close" });
    expect(
      resolveActivation(
        "ingamemenu",
        node({ name: "menu_close_button" }),
        "countrypoliticsview",
      ),
    ).toEqual({
      kind: "close",
    });
    expect(
      resolveActivation(
        "intelligence_agency_upgrades_window",
        node({ name: "btn_close" }),
        "intelligence_agency_upgrades_window",
      ),
    ).toEqual({ kind: "open", window: "countryintelligenceagencyview" });
  });

  it.each([
    ["manage_occupied_window", "countrypoliticsview"],
    ["exiles_window", "countrypoliticsview"],
    ["powerbalanceview", "countrypoliticsview"],
    ["production_equipment_window_military", "countryproductionlineview"],
    ["production_equipment_window", "countryproductionlineview"],
    ["equipment_overview_window", "countryarmyview"],
    ["building_roster_window", "countryarmyview"],
    ["navallosseswindow", "countryarmyview"],
    ["divisioncommanderwindow", "countryofficercorpview"],
    ["capturedarmyleaderwindow", "countryofficercorpview"],
  ])("returns from the %s pop-up to %s", (owner, under) => {
    expect(
      resolveActivation(owner, node({ name: "close_button" }), owner),
    ).toEqual({ kind: "open", window: under });
  });

  it("reports buttons with no known destination instead of failing silently or guessing", () => {
    expect(
      resolveActivation("topbar", node({ name: "music_next_button" }), null),
    ).toEqual({
      kind: "unsupported",
      control: "music_next_button",
    });
    expect(
      resolveActivation(
        "countrypoliticsview",
        node({ name: "dismantle_faction" }),
        null,
      ),
    ).toEqual({
      kind: "unsupported",
      control: "dismantle_faction",
    });
  });
});

describe("hotkeys", () => {
  it("spells keys the way topbar.gui does", () => {
    const k = (key: string, mods: Partial<KeyboardEvent> = {}) =>
      hotkeyOf({
        key,
        shiftKey: false,
        ctrlKey: false,
        altKey: false,
        metaKey: false,
        ...mods,
      });
    expect(k("q")).toBe("q");
    expect(k("Q", { shiftKey: true })).toBe("shift+q");
    expect(k("q", { ctrlKey: true })).toBe("ctrl+q");
    expect(k("Escape")).toBe("ESCAPE");
  });

  it("spells the clock's keys the way time_control_window does", () => {
    const k = (key: string, code: string) =>
      hotkeyOf({
        key,
        code,
        shiftKey: false,
        ctrlKey: false,
        altKey: false,
        metaKey: false,
      });
    expect(k(" ", "Space")).toBe("space");
    expect(k("+", "NumpadAdd")).toBe("kp_plus");
    expect(k("-", "NumpadSubtract")).toBe("kp_minus");

    expect(k("-", "Minus")).toBe("-");
    expect(k("q", "KeyQ")).toBe("q");
  });

  it("reads shortcuts from the window's own .gui instead of a hard-coded table", () => {
    const flag = node({ name: "player_flag", raw: { shortcut: "q" } });
    const decisions = node({
      name: "decisionview_button",
      raw: { shortcut: "shift+q" },
    });
    const root = node({
      type: "containerwindowtype",
      name: "topbar",
      children: [flag, node({ children: [decisions] })],
    });
    const map = shortcutsOf(root);
    expect(map.get("q")).toBe(flag);
    expect(map.get("shift+q")).toBe(decisions);
  });
});

describe("restPlacement", () => {
  it("rests a side panel at show_position and slides in from position, like the game", () => {
    const panel = node({
      type: "containerwindowtype",
      position: { x: -606, y: 78 },
      raw: { show_position: { x: -6, y: 78 }, animation_time: 300 },
    });
    expect(restPlacement(panel)).toEqual({
      x: -6,
      y: 78,
      fromX: -606,
      fromY: 78,
      ms: 300,
    });
  });

  it("centres a window whose orientation and origo both say centre", () => {
    const picker = node({
      type: "containerwindowtype",
      position: { x: 0, y: 0 },
      size: { width: 1225, height: 717 },
      orientation: "CENTER",
      origo: "center",
    });
    const place = restPlacement(picker, 1920, 1080);
    expect([place.x, place.y]).toEqual([(1920 - 1225) / 2, (1080 - 717) / 2]);
  });

  it("puts a centre-oriented window without origo by its top-left", () => {
    const event = node({
      type: "containerwindowtype",
      position: { x: -282, y: -310 },
      size: { width: 581, height: 427 },
      orientation: "CENTER",
    });
    const place = restPlacement(event, 1920, 1080);
    expect([place.x, place.y]).toEqual([960 - 282, 540 - 310]);
  });

  it("rests a window that names its place in position_default there", () => {
    const list = node({
      type: "containerwindowtype",
      size: { width: 550, height: 667 },
      children: [
        node({
          type: "positiontype",
          name: "position_default",
          position: { x: 540, y: 80 },
        }),
        node({
          type: "positiontype",
          name: "position_equipment_designer",
          position: { x: 265, y: 80 },
        }),
      ],
    });
    expect(restPlacement(list, 1920, 1080)).toMatchObject({
      x: 540,
      y: 80,
      ms: 0,
    });
  });

  it("leaves an upper-left window on its own coordinates", () => {
    const panel = node({
      type: "containerwindowtype",
      position: { x: 12, y: 40 },
      orientation: "UPPER_LEFT",
    });
    expect(restPlacement(panel, 1920, 1080)).toMatchObject({ x: 12, y: 40 });
  });

  it("measures a lower-left window from the bottom edge, not from bottom minus its height", () => {
    const state = node({
      type: "containerwindowtype",
      position: { x: -416, y: -549 },
      size: { width: 436, height: 496 },
      orientation: "lower_left",
      raw: { show_position: { x: -6, y: -655 }, animation_time: 300 },
    });
    const place = restPlacement(state, 1920, 1080);
    expect([place.x, place.y]).toEqual([-6, 1080 - 655]);
  });

  it("places a window without a slide-in where it is declared", () => {
    expect(restPlacement(node({ position: { x: -3, y: 78 } }))).toEqual({
      x: -3,
      y: 78,
      fromX: -3,
      fromY: 78,
      ms: 0,
    });
  });
});

describe("RUNTIME_HIDDEN", () => {
  it("hides the on-demand focus-tree overlays an idle game never shows", async () => {
    const { RUNTIME_HIDDEN } = await import("./gameNavigation");
    expect(RUNTIME_HIDDEN.nationalfocusview).toEqual([
      "continuous_focus_window",
      "search_tooltip",
    ]);
  });

  it("hides the topbar's count badges, which are all 0 at a start", () => {
    for (const name of [
      "decisionview_amount_timeout_bg",
      "decisionview_amount_timeout_items",
      "intelview_ops_prepared_or_completed_bg",
      "intelview_ops_prepared_or_completed_items",
      "intelview_ops_available_bg",
      "intelview_ops_available_items",
      "prototype_rewards_aquired_bg",
      "prototype_rewards_aquired",
      "non_delivering_contracts",
      "delivering_contracts",
    ]) {
      expect(RUNTIME_HIDDEN.topbar).toContain(name);
    }
    expect(RUNTIME_HIDDEN.topbar).not.toContain(
      "decisionview_amount_to_take_items",
    );
  });
});

describe("the topbar's own controls", () => {
  it("sends L to the army view, which holds the air section, and E to the market", () => {
    expect(TOPBAR_TARGETS.air_button).toBe("countryarmyview");
    expect(TOPBAR_TARGETS.diplomacy_button).toBe(
      "countryinternationalmarketview",
    );
    expect(TOPBAR_SPRITE_SWAPS.diplomacy_button).toBe(
      "GFX_topbar_international_market",
    );
  });

  it("lights the button that opened the panel, else the first that names it", () => {
    expect(topbarButtonFor("countryarmyview", "air_button")).toBe("air_button");
    expect(topbarButtonFor("countryarmyview", null)).toBe("army_button");
    expect(topbarButtonFor("countryconstructionsview", "air_button")).toBe(
      "construction_button",
    );
    expect(topbarButtonFor(null)).toBeNull();

    expect(
      topbarButtonFor(TOPBAR_TARGETS.technology_button, "technology_button"),
    ).toBeNull();
    expect(
      topbarButtonFor(
        TOPBAR_TARGETS.intel_agency_button,
        "intel_agency_button",
      ),
    ).toBeNull();
  });

  it("reads the clock, music and menu buttons as screen controls", () => {
    expect(topbarControlOf("pause_button_date_paused")).toEqual({
      kind: "pause",
    });
    expect(topbarControlOf("button_speedup")).toEqual({ kind: "speed", by: 1 });
    expect(topbarControlOf("button_speeddown")).toEqual({
      kind: "speed",
      by: -1,
    });
    expect(topbarControlOf("button_speedstep4")).toEqual({
      kind: "speedTo",
      speed: 4,
    });
    expect(topbarControlOf("playlist_button")).toEqual({ kind: "music" });
    expect(topbarControlOf("menu_button")).toEqual({
      kind: "menu",
      window: "ingamemenu",
    });
    expect(topbarControlOf("navy_button")).toBeNull();
  });

  it("keeps the clock paused at speed 1 to start, with speed held to 1..5", () => {
    expect(START_CLOCK).toEqual({ paused: true, speed: 1 });
    expect(nextClock(START_CLOCK, { kind: "pause" })).toEqual({
      paused: false,
      speed: 1,
    });
    expect(nextClock(START_CLOCK, { kind: "speed", by: -1 })).toEqual(
      START_CLOCK,
    );
    expect(
      nextClock({ paused: true, speed: 5 }, { kind: "speed", by: 1 }).speed,
    ).toBe(5);
    expect(nextClock(START_CLOCK, { kind: "speedTo", speed: 3 }).speed).toBe(3);
  });

  it("opens only windows the game has for an alert", () => {
    expect(ALERT_TARGETS.alert_no_research).toBe("countrytechnologyview");
    expect(ALERT_TARGETS.alert_select_focus).toBe("nationalfocusview");
  });

  it("eases like CSS cubic-bezier", () => {
    const linear = cubicBezier(0, 0, 1, 1);
    expect(linear(0.5)).toBeCloseTo(0.5, 5);
    expect(DECELERATED(0)).toBe(0);
    expect(DECELERATED(1)).toBe(1);

    expect(DECELERATED(0.5)).toBeGreaterThan(0.75);
  });
});

describe("engine runtime knowledge", () => {
  it("matches a name path through ancestors, not every node of that name", async () => {
    const { matchNamePath } = await import("./gameNavigation");
    const inBox = node({ name: "title", type: "instanttextboxtype" });
    const header = node({ name: "title", type: "instanttextboxtype" });
    const root = node({
      type: "containerwindowtype",
      name: "countrypoliticsview",
      children: [
        header,
        node({
          type: "containerwindowtype",
          name: "active_goal",
          children: [inBox],
        }),
      ],
    });
    expect(matchNamePath(root, "active_goal/title")).toEqual([inBox]);
    expect(matchNamePath(root, "title")).toEqual([header, inBox]);
  });

  it("treats glows as runtime highlights that an idle screen never shows", async () => {
    const { isRuntimeEffect } = await import("./gameNavigation");
    expect(isRuntimeEffect("infantry_button_glow")).toBe(true);
    expect(isRuntimeEffect("tutorial_glow_2")).toBe(true);
    expect(isRuntimeEffect("nat_spirit_glow_overlay")).toBe(true);
    expect(isRuntimeEffect("infantry_button")).toBe(false);
    expect(isRuntimeEffect(null)).toBe(false);
  });

  it("keeps the glows the engine always shows: each owned DLC's on the main menu", async () => {
    const { STEADY_GLOWS } = await import("./gameNavigation");
    expect(STEADY_GLOWS.mainmenu_dlc_ownership).toContain("glowup");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      'if (isRuntimeEffect(row.node.name) && !steady.includes(row.node.name ?? "")) hidden.add(row.id);',
    );
  });

  it("replaces the officer-corps Placeholder labels with the engine's keys", async () => {
    const { RUNTIME_TEXT } = await import("./gameNavigation");
    expect(RUNTIME_TEXT.countryofficercorpview.commander_roster_army).toBe(
      "COMMANDER_ROSTER_ARMY_BUTTON",
    );
    expect(RUNTIME_TEXT.countrypoliticsview["active_goal/title"]).toBe(
      "POLITICS_SELECT_FOCUS",
    );
  });

  it("opens Division Templates together with Recruit & Deploy", async () => {
    const { COMPANIONS } = await import("./gameNavigation");
    expect(COMPANIONS.countrydeploymentview).toEqual([
      "templatedeploymentwindow",
    ]);
  });
});

describe("the division designer opens over Recruit & Deploy", () => {
  const DESIGNER = "countrydivisiondesignerview";
  const DEPLOY = "countrydeploymentview";

  it("keeps the deployment panel underneath", async () => {
    const { OPENS_OVER, basePanelOf } = await import("./gameNavigation");
    expect(OPENS_OVER[DESIGNER]).toBe(DEPLOY);
    expect(basePanelOf(DESIGNER)).toBe(DEPLOY);

    expect(basePanelOf("countrytechnologyview")).toBe("countrytechnologyview");
    expect(basePanelOf(null)).toBeNull();
  });

  it("goes back to the panel under it on its own X, and the panel's X closes everything", () => {
    expect(
      resolveActivation(DESIGNER, node({ name: "close_button" }), DESIGNER),
    ).toEqual({
      kind: "open",
      window: DEPLOY,
    });
    expect(
      resolveActivation(DEPLOY, node({ name: "close_button" }), DESIGNER),
    ).toEqual({ kind: "close" });
  });

  it("treats the pop-up as its panel for the topbar toggle", () => {
    expect(
      resolveActivation(
        "topbar",
        node({ name: "deployment_button" }),
        DESIGNER,
      ),
    ).toEqual({ kind: "close" });
    expect(
      resolveActivation(
        "topbar",
        node({ name: "technology_button" }),
        DESIGNER,
      ),
    ).toEqual({
      kind: "open",
      window: "countrytechnologyview",
    });
  });

  it("draws it as a pop-up over both panels, taking the mouse, and closes it one Esc at a time", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");

    expect(page).toContain("const base = basePanelOf(panel, opensOver);");
    expect(page).toContain("key={base}");

    expect(page.indexOf("{popup && popupWin.data && (")).toBeGreaterThan(
      page.indexOf("{companionName && companion.data && ("),
    );

    expect(page).toMatch(
      /\{popup && popupWin\.data && \([\s\S]{0,200}\s+modal\s/,
    );
    expect(page).toContain('pointerEvents: "auto",');
    expect(page).toContain("setPanel((at) => (at && opensOver[at]) || null);");
  });

  it("hides the Army HQ row's placeholders the way it hides a division's", () => {
    const hidden = RUNTIME_HIDDEN.countrydivisiondesignerview;
    for (const name of [
      "hq_view/div_templ_symbol_button",
      "hq_view/equipment_warning",
      "hq_view/division_priority_icon",
      "hq_view/priority_button_1",
      "hq_overlay_images/combat_width_view_hq/combat_width_text",
    ]) {
      expect(hidden).toContain(name);
    }
  });

  it("shows the designer's row 1 before hiding its runtime state", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    const shown = page.indexOf(
      "for (const namePath of [...(RUNTIME_SHOWN[win.name] ?? []), ...alsoShown])",
    );
    expect(shown).toBeGreaterThan(-1);
    expect(
      page.indexOf("for (const namePath of RUNTIME_HIDDEN[win.name] ?? [])"),
    ).toBeGreaterThan(shown);
  });
});

describe("movesBetweenScreens", () => {
  it("keeps the buttons that move between screens working in edit mode", () => {
    expect(
      movesBetweenScreens(
        "topbar",
        node({ name: "technology_button" }),
        "countrypoliticsview",
      ),
    ).toBe(true);
    expect(
      movesBetweenScreens("topbar", node({ name: "navy_button" }), null),
    ).toBe(true);
    expect(
      movesBetweenScreens(
        "countrypoliticsview",
        node({ name: "add_national_goal_button" }),
        "countrypoliticsview",
      ),
    ).toBe(true);
    expect(
      movesBetweenScreens(
        "countryofficercorpview",
        node({ name: "commander_roster_army" }),
        "countryofficercorpview",
      ),
    ).toBe(true);
    expect(
      movesBetweenScreens(
        "countrytechnologyview",
        node({ name: "close_button" }),
        "countrytechnologyview",
      ),
    ).toBe(true);
  });

  it("leaves everything else to be picked for editing", () => {
    expect(
      movesBetweenScreens("topbar", node({ name: "music_next_button" }), null),
    ).toBe(false);
    expect(
      movesBetweenScreens(
        "countrypoliticsview",
        node({ name: "dismantle_faction" }),
        "countrypoliticsview",
      ),
    ).toBe(false);
    expect(
      movesBetweenScreens(
        "countrydivisiondesignerview",
        node({ name: "save_button" }),
        "countrydivisiondesignerview",
      ),
    ).toBe(false);
  });
});
