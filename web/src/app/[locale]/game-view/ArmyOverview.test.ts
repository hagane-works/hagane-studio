import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  ARMY_OVERVIEW,
  armyOverviewWindow,
  overviewMode,
  type GuiAirOverview,
  type GuiArmyOverview,
  type GuiNavyOverview,
} from "./ArmyOverviewLayer";
import { RUNTIME_SHOWN, TOPBAR_TARGETS, matchNamePath } from "./gameNavigation";
import { divisionChanges } from "./DivisionEditPanel";
import {
  oobUnitChanges,
  oobUnitTarget,
  type OOBUnit,
} from "./OOBUnitEditPanel";

const node = (
  name: string,
  type: string,
  children: GuiNode[] = [],
  extra: Partial<GuiNode> = {},
): GuiNode =>
  ({ name, type, children, position: { x: 0, y: 0 }, ...extra }) as GuiNode;

const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

const window = (): GuiWindow =>
  ({
    name: ARMY_OVERVIEW,
    sprites: {},
    texts: {},
    fonts: {},
    node: node(ARMY_OVERVIEW, "containerwindowtype", [
      node("army_title", "instanttextboxtype"),
      node("naval_hq_open", "buttontype"),
      node("destroyed_ships_open", "buttontype"),
      node("details_open", "buttontype"),
      node(
        "info",
        "containerwindowtype",
        [
          node("army_overview_top", "icontype"),
          node("unit_count", "instanttextboxtype"),
          node("battalion_count", "instanttextboxtype"),
          node("special_forces_icon", "icontype"),
          node("special_forces_cap", "instanttextboxtype"),
        ],
        {
          raw: { margin: { top: 13, left: 13, right: 13 } },
        },
      ),
      node(
        "army_header",
        "containerwindowtype",
        [
          node("modifiers_grid", "gridboxtype"),
          node("sub_units_grid", "gridboxtype"),
        ],
        { raw: { margin: { top: 13, left: 13, right: 25 } } },
      ),
      node(
        "army_units",
        "containerwindowtype",
        [node("item_grid", "gridboxtype")],
        { raw: { margin: { top: 13, left: 13, bottom: 40, right: 25 } } },
      ),
      node(
        "navy_header",
        "containerwindowtype",
        [
          node("modifiers_grid", "gridboxtype"),
          node("ships_count_grid", "gridboxtype"),
        ],
        { raw: { margin: { top: 13, left: 13, right: 25 } } },
      ),
      node(
        "navy_units",
        "containerwindowtype",
        [node("item_grid", "gridboxtype")],
        { raw: { margin: { top: 13, left: 13, bottom: 40, right: 25 } } },
      ),
      node(
        "air_header",
        "containerwindowtype",
        [
          node("count_grid", "gridboxtype"),
          node("modifiers_grid", "gridboxtype"),
        ],
        { raw: { margin: { top: 13, left: 13, right: 25 } } },
      ),
      node(
        "air_units",
        "containerwindowtype",
        [node("item_grid", "gridboxtype")],
        { raw: { margin: { top: 13, left: 13, bottom: 40, right: 25 } } },
      ),
      node("theatres_count", "instanttextboxtype"),
    ]),
  }) as unknown as GuiWindow;

const germany: GuiArmyOverview = {
  tag: "GER",
  divisions: [],
  pills: [],
  templates: [],
  special_forces_cap: 24,
  texts: {
    title: "Division Overview",
    divisions: "30 Divisions",
    battalions: "315 Battalions",
    theatres: "Active in 0 Theater(s)",
    special_cap: "24",
  },
  sprites: {},
};

describe("the Division Overview", () => {
  it("opens on the army's section alone, inside its margins, with its counts", () => {
    const out = armyOverviewWindow(window(), "army", germany);
    for (const gone of [
      "navy_header",
      "navy_units",
      "air_header",
      "air_units",
      "naval_hq_open",
      "destroyed_ships_open",
    ]) {
      expect(matchNamePath(out.win.node, gone)).toHaveLength(0);
    }
    expect(matchNamePath(out.win.node, "details_open")).toHaveLength(1);

    expect(
      matchNamePath(out.win.node, "army_units/army_units_content")[0].position,
    ).toEqual({ x: 13, y: 13 });
    const text = (path: string) =>
      out.texts[idOf(matchNamePath(out.win.node, path)[0])];
    expect(text("army_title")).toBe("Division Overview");
    expect(text("unit_count")).toBe("30 Divisions");
    expect(text("battalion_count")).toBe("315 Battalions");
    expect(text("special_forces_cap")).toBe("24");
    expect(text("theatres_count")).toBe("Active in 0 Theater(s)");

    expect(out.hidden).toEqual([
      idOf(matchNamePath(out.win.node, "modifiers_grid")[0]),
    ]);
    expect(RUNTIME_SHOWN[ARMY_OVERVIEW]).toEqual(
      expect.arrayContaining([
        "info",
        "army_header",
        "army_units",
        "navy_header",
        "navy_units",
      ]),
    );
    expect(matchNamePath(out.win.node, "army_overview_top")[0].frame).toBe(1);
  });

  it("draws a pill a division type and a row a division, and is wired from the page", () => {
    const layer = readFileSync(
      join(__dirname, "ArmyOverviewLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain(
      'section === "army" ? { sub_units_count: String(count) } : section === "navy" ? { ships_count: String(count) } : { count: String(count) }',
    );
    expect(layer).toContain("frames={{ level: division.level + 1 }}");
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "armyOverviewWindow(win, overview, armyOverview.data, navyOverview.data, airOverview.data)",
    );
    expect(page).toContain("mode={overview}");
    expect(page).toContain("onEditDivision={onEditDivision}");
    expect(layer).toContain("{onEditDivision && (");
  });
});

describe("the Navy Overview", () => {
  const navy: GuiNavyOverview = {
    tag: "GER",
    task_forces: [],
    pills: [],
    texts: {
      title: "Navy Overview",
      fleets: "5 Fleets",
      theatres: "Active in 3 Strategic area(s)",
    },
    sprites: {},
  };

  it("opens from the Navy button on the navy's section, its title buttons and ship picture, its counts", () => {
    expect(TOPBAR_TARGETS.navy_button).toBe(ARMY_OVERVIEW);
    expect(TOPBAR_TARGETS.air_button).toBe(ARMY_OVERVIEW);
    expect([
      overviewMode("navy_button"),
      overviewMode("air_button"),
      overviewMode("army_button"),
      overviewMode(null),
    ]).toEqual(["navy", "air", "army", "army"]);
    const out = armyOverviewWindow(window(), "navy", undefined, navy);
    for (const gone of ["army_header", "army_units", "air_header", "air_units"])
      expect(matchNamePath(out.win.node, gone)).toHaveLength(0);
    for (const kept of [
      "naval_hq_open",
      "destroyed_ships_open",
      "navy_header",
      "navy_units",
    ])
      expect(matchNamePath(out.win.node, kept)).toHaveLength(1);
    expect(matchNamePath(out.win.node, "army_overview_top")[0].frame).toBe(2);
    const text = (path: string) =>
      out.texts[idOf(matchNamePath(out.win.node, path)[0])];
    expect(text("army_title")).toBe("Navy Overview");
    expect(text("unit_count")).toBe("5 Fleets");
    expect(text("theatres_count")).toBe("Active in 3 Strategic area(s)");

    for (const part of [
      "battalion_count",
      "special_forces_icon",
      "special_forces_cap",
      "navy_header/modifiers_grid",
    ]) {
      expect(out.hidden).toContain(idOf(matchNamePath(out.win.node, part)[0]));
    }
  });

  it("switches between the overviews rather than closing, and draws the pride of the fleet in gold", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "overviewMode(openedBy) !== overviewMode(node.name)",
    );
    const layer = readFileSync(
      join(__dirname, "ArmyOverviewLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain("frames={{ icon: force.pride ? 2 : 1 }}");
  });
});

describe("the Air Forces Overview", () => {
  const air: GuiAirOverview = {
    tag: "GER",
    wings: [],
    pills: [],
    texts: {
      title: "Air Forces Overview",
      wings: "15 Air Wings",
      theatres: "Active in 0 Strategic area(s)",
    },
    sprites: {},
  };

  it("opens from the Air button on the air section, the planes picture and its counts", () => {
    const out = armyOverviewWindow(window(), "air", undefined, undefined, air);
    for (const gone of [
      "army_header",
      "navy_header",
      "naval_hq_open",
      "destroyed_ships_open",
    ])
      expect(matchNamePath(out.win.node, gone)).toHaveLength(0);
    expect(matchNamePath(out.win.node, "air_header")).toHaveLength(1);
    expect(matchNamePath(out.win.node, "army_overview_top")[0].frame).toBe(3);
    const text = (path: string) =>
      out.texts[idOf(matchNamePath(out.win.node, path)[0])];
    expect(text("army_title")).toBe("Air Forces Overview");
    expect(text("unit_count")).toBe("15 Air Wings");
    expect(text("theatres_count")).toBe("Active in 0 Strategic area(s)");
    expect(out.hidden).toContain(
      idOf(matchNamePath(out.win.node, "air_header/modifiers_grid")[0]),
    );
  });

  it("draws a pill a plane role on its frame of the strip, and a row a wing with its count of 100", () => {
    const layer = readFileSync(
      join(__dirname, "ArmyOverviewLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain(
      'frames={section === "air" && item.frame ? { icon: item.frame } : undefined}',
    );
    expect(layer).toContain(
      "count: String(wing.count), limit: String(wing.limit)",
    );
  });
});

describe("a division's edit panel", () => {
  const panzer = {
    name: "1. Panzer-Division",
    template: "Panzer-Division",
    icon: "GFX_unit_light_armor_icon_medium",
    level: 2,
    strength: 1,
    organisation: 1,
    oob: "GER_1936",
    ordinal: 26,
    experience: 0.3,
    equipment: 1,
  };

  it("sends only what changed, the shares from percent", () => {
    expect(
      divisionChanges(panzer, {
        name: "1. Panzer-Division",
        template: "Panzer-Division",
        experience: "30",
        equipment: "100",
      }),
    ).toEqual({});
    expect(
      divisionChanges(panzer, {
        name: "Panzer-Lehr-Division",
        experience: "75",
        equipment: "50",
      }),
    ).toEqual({
      name: "Panzer-Lehr-Division",
      experience: 0.75,
      equipment: 0.5,
    });
    expect(
      divisionChanges(panzer, { template: "Infanterie-Division" }),
    ).toEqual({ template: "Infanterie-Division" });
    for (const wrong of [
      { name: " " },
      { name: 'Die "Beste"' },
      { experience: "120" },
      { equipment: "" },
    ]) {
      expect(divisionChanges(panzer, wrong)).toBeNull();
    }
  });

  it("opens from a division's row in edit mode and saves its block of the battle order", () => {
    const layer = readFileSync(
      join(__dirname, "ArmyOverviewLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain('label={t("division", { name: division.name })}');
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      'panel === ARMY_OVERVIEW && overviewMode(openedBy) === "army" && editedDivision && !selection && (',
    );
    const panel = readFileSync(
      join(__dirname, "DivisionEditPanel.tsx"),
      "utf8",
    );
    expect(panel).toContain("/gui-render/division-save");
    expect(panel).toContain(
      "JSON.stringify({ oob: division.oob, ordinal: division.ordinal, ...body })",
    );
  });
});

describe("a task force's and an air wing's edit panel", () => {
  const force: OOBUnit = {
    kind: "task_force",
    unit: {
      name: "Hochseeflotte",
      fleet: "Kriegsmarine",
      region: "North-West Germany",
      size: 12,
      icon: "GFX_unit_battleship_icon_medium",
      class: "battleship",
      pride: true,
      strength: 1,
      organisation: 1,
      oob: "GER_1936_naval_mtg",
      ordinal: 0,
      raw_name: "Hochseeflotte",
    },
  };
  const wing: OOBUnit = {
    kind: "air_wing",
    unit: {
      name: "Küstenfliegergruppe 106",
      design: "Do 22",
      type: "small_plane_naval_bomber_airframe_1",
      role: "naval_bomber",
      icon: "",
      count: 72,
      limit: 100,
      base: "Holstein",
      state: 58,
      oob: "GER_1936_air_bba",
      ordinal: 14,
      state_index: 0,
      raw_name: "Küstenfliegergruppe 106",
    },
  };

  it("sends only what changed; a wing's count is a whole 1 to 1000, a task force has none", () => {
    expect(oobUnitChanges(force, { name: "Hochseeflotte" })).toEqual({});
    expect(oobUnitChanges(force, { name: " Flottenverband Nord " })).toEqual({
      name: "Flottenverband Nord",
    });
    expect(oobUnitChanges(force, { amount: "5" })).toEqual({});
    expect(
      oobUnitChanges(wing, { name: "Jagdgeschwader Richthofen", amount: "60" }),
    ).toEqual({ name: "Jagdgeschwader Richthofen", amount: 60 });
    expect(oobUnitChanges(wing, { amount: "72" })).toEqual({});
    for (const wrong of [
      { name: "" },
      { name: '"Nord"' },
      { amount: "0" },
      { amount: "12.5" },
      { amount: "1001" },
      { amount: " " },
    ]) {
      expect(oobUnitChanges(wing, wrong)).toBeNull();
    }
  });

  it("names the unit by its file, its place, and a wing's air base", () => {
    expect(oobUnitTarget(force)).toEqual({
      kind: "task_force",
      oob: "GER_1936_naval_mtg",
      ordinal: 0,
      raw_name: "Hochseeflotte",
    });
    expect(oobUnitTarget(wing)).toEqual({
      kind: "air_wing",
      oob: "GER_1936_air_bba",
      ordinal: 14,
      raw_name: "Küstenfliegergruppe 106",
      state: 58,
      state_index: 0,
    });
  });

  it("opens from a row in edit mode and saves through oob-unit-save", () => {
    const layer = readFileSync(
      join(__dirname, "ArmyOverviewLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain(
      'onClick={() => onEditUnit("task_force", force.oob, force.ordinal)}',
    );
    expect(layer).toContain(
      'onClick={() => onEditUnit("air_wing", wing.oob, wing.ordinal)}',
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      'overviewMode(openedBy) === (editedUnit.kind === "task_force" ? "navy" : "air")',
    );
    expect(page).toContain("onEditUnit={onEditUnit}");
    expect(layer).toContain("{onEditUnit && (");
    const panel = readFileSync(join(__dirname, "OOBUnitEditPanel.tsx"), "utf8");
    expect(panel).toContain("/gui-render/oob-unit-save");
  });
});
