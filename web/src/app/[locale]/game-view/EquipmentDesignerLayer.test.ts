import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import {
  DESIGNER_VIEWS,
  PLANE_DESIGNER,
  SHIP_DESIGNER,
  TANK_DESIGNER,
  blueprintHighlights,
  blueprintUnfitted,
  designerCostText,
  designerSaveText,
  equipmentDesignerWindow,
  formatDesignerStat,
  lockedSlots,
  slotPlace,
  statEntryWindow,
  type GuiEquipmentDesign,
} from "./EquipmentDesignerLayer";
import { OPENS_OVER, matchNamePath } from "./gameNavigation";

const node = (
  name: string,
  type: string,
  extra: Partial<GuiNode> = {},
  children: GuiNode[] = [],
): GuiNode => ({ name, type, children, ...extra }) as GuiNode;

const view = (): GuiWindow =>
  ({
    name: TANK_DESIGNER,
    sprites: {},
    texts: {},
    fonts: {},
    node: node(TANK_DESIGNER, "containerwindowtype", {}, [
      node("designer_title", "instanttextboxtype"),
      node("production_value", "instanttextboxtype"),
      node("production_resources", "instanttextboxtype"),
      node("refit_cost", "instanttextboxtype"),
      node("experience", "instanttextboxtype"),
      node("save_button", "buttontype"),
      node("select_model", "icontype"),
      node("border_container", "containerwindowtype"),
      node("select3d_button", "buttontype"),
      node("icon_selection_container", "containerwindowtype", {}, [
        node("btn_equipment_icon", "buttontype"),
      ]),
      node("blueprint_type_name", "instanttextboxtype"),
      node("equipments", "dropdownboxtype", {}, [
        node("name", "instanttextboxtype"),
        node("expanded_window", "containerwindowtype"),
      ]),
      node("dropdown_tank_roles", "dropdownboxtype", {}, [
        node("role_text", "instanttextboxtype"),
      ]),
      node(
        "equipment_modules",
        "containerwindowtype",
        { position: { x: 13, y: 146 } },
        [
          node("pos_custom_module_slot_window_0", "positiontype", {
            position: { x: 1, y: 1 },
          }),
          node("pos_custom_module_slot_window_2", "positiontype", {
            position: { x: 1, y: 300 },
          }),
          node("equipment_upgrade_0", "containerwindowtype", {}, [
            node("name", "instanttextboxtype"),
            node("level", "instanttextboxtype"),
          ]),
          node("equipment_upgrade_1", "containerwindowtype", {}, [
            node("name", "instanttextboxtype"),
            node("level", "instanttextboxtype"),
          ]),
        ],
      ),
      node("equipment_select_model", "containerwindowtype"),
      node("module_selector_window", "containerwindowtype"),
    ]),
  }) as unknown as GuiWindow;

const design = (
  extra: Partial<GuiEquipmentDesign> = {},
): GuiEquipmentDesign => ({
  found: true,
  name: "Inter-War Medium Tank",
  type: "medium_tank_chassis_0",
  archetype: "medium_tank_chassis",
  type_label: "Inter-War Medium Tank",
  year: 1934,
  role: "Medium Tank",
  texts: {
    title: "Tank Designer",
    cost: "Production Cost: £production_cost $VALUE|2$",
    save: "Save £$XPICON$ $XPCOST|Y0$",
  },
  experience: 0,
  icon: "GFX_medium_tank_equipment_0_medium",
  slots: [],
  upgrades: [{ name: "tank_nsb_engine_upgrade", label: "Engine", level: 0 }],
  columns: { base: [], combat: [], misc: [] },
  cost: 4.6000000000000005,
  resources: { steel: 1 },
  blueprint: "equipment_designer_medium_tank_chassis",
  sprites: {
    GFX_medium_tank_equipment_0_medium: { width: 60, height: 30 } as never,
  },
  ...extra,
});

const find = (win: GuiWindow, path: string) => matchNamePath(win.node, path)[0];
const idOf = (n: GuiNode | undefined) =>
  (n as GuiNode & { _id?: string })?._id ?? "";

describe("the tank designer", () => {
  it("prints the numbers as the game does (recording 3:08)", () => {
    expect(formatDesignerStat(5.175, "speed")).toBe("5.1 km/h");
    expect(formatDesignerStat(0.75, "percent")).toBe("75.0%");
    expect(formatDesignerStat(0.25, "rate")).toBe("0.25");
    expect(formatDesignerStat(2, "rate")).toBe("2.00");
    expect(formatDesignerStat(11.5, "")).toBe("11.5");
    expect(formatDesignerStat(20, "")).toBe("20.0");
    expect(
      designerCostText(
        "Production Cost: £production_cost $VALUE|2$",
        4.6000000000000005,
      ),
    ).toBe("Production Cost: £production_cost 4.60");
    expect(designerSaveText("Save £$XPICON$ $XPCOST|Y0$")).toBe(
      "Save £army_experience §Y0§!",
    );
  });

  it("writes the design in and keeps its pop-ups for their buttons", () => {
    const out = equipmentDesignerWindow(view(), design());
    const text = (path: string) => out.texts[idOf(find(out.win, path))];
    expect(text("designer_title")).toBe("Tank Designer");
    expect(text("equipments/name")).toBe("Inter-War Medium Tank");
    expect(text("dropdown_tank_roles/role_text")).toBe("Medium Tank");
    expect(text("production_value")).toBe(
      "Production Cost: £production_cost 4.60",
    );
    expect(text("production_resources")).toBe("1");
    expect(text("experience")).toBe("£army_experience 0");
    expect(text("equipment_modules/equipment_upgrade_0/name")).toBe("Engine");

    expect(out.hidden).toContain(
      idOf(find(out.win, "equipment_modules/equipment_upgrade_1")),
    );
    for (const popup of [
      "equipment_select_model",
      "module_selector_window",
      "equipments/expanded_window",
      "refit_cost",
    ]) {
      expect(out.hidden).toContain(idOf(find(out.win, popup)));
    }

    for (const part of [
      "select_model",
      "border_container",
      "select3d_button",
    ]) {
      expect(out.hidden).toContain(idOf(find(out.win, part)));
    }

    expect(
      find(out.win, "icon_selection_container/btn_equipment_icon").sprite,
    ).toBe("GFX_medium_tank_equipment_0_medium");
  });

  it("places the slots at their positions and leaves the drawing's highlights out", () => {
    expect(slotPlace(view(), 0)).toEqual({ x: 14, y: 147 });
    expect(slotPlace(view(), 2)).toEqual({ x: 14, y: 446 });
    expect(slotPlace(view(), 7)).toBeNull();
    const drawing = node(
      "equipment_designer_medium_tank_chassis",
      "containerwindowtype",
      {},
      [
        node("module_slots", "containerwindowtype", {}, [
          node("turret_type_slot", "containerwindowtype", {}, [
            node("turret_type_slot", "containerwindowtype"),
            node("turret_type_slot@highlight", "containerwindowtype"),
          ]),
        ]),
      ],
    );
    expect(blueprintHighlights(drawing)).toEqual([
      "turret_type_slot@highlight",
    ]);
  });

  it("right-aligns a stat row's number across the row", () => {
    const entry = {
      name: "equipment_designer_stat_entry",
      sprites: {},
      texts: {},
      fonts: {},
      node: node("equipment_designer_stat_entry", "containerwindowtype", {}, [
        node("label", "instanttextboxtype", { position: { x: 0, y: 6 } }),
        node("count", "instanttextboxtype", {
          position: { x: 126, y: 6 },
          maxwidth: 120,
        } as Partial<GuiNode>),
      ]),
    } as unknown as GuiWindow;
    const count = find(statEntryWindow(entry), "count") as GuiNode & {
      maxwidth?: number;
    };
    expect(count.position).toEqual({ x: 0, y: 6 });
    expect(count.maxwidth).toBe(155);
  });

  it("opens over the production screen from a tank or airframe row's variant button", () => {
    expect(OPENS_OVER[TANK_DESIGNER]).toBe("countryproductionlineview");
    expect(OPENS_OVER[PLANE_DESIGNER]).toBe("countryproductionlineview");
    expect(DESIGNER_VIEWS.tank).toBe(TANK_DESIGNER);
    expect(DESIGNER_VIEWS.plane).toBe(PLANE_DESIGNER);
    expect(DESIGNER_VIEWS.ship).toBe(SHIP_DESIGNER);
    expect(OPENS_OVER[SHIP_DESIGNER]).toBe("countryproductionlineview");
    const layer = readFileSync(join(__dirname, "ProductionLayer.tsx"), "utf8");
    expect(layer).toContain(
      "onOpenDesigner(item.name, item.designer as string)",
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("setPanel(DESIGNER_VIEWS[kind] ?? TANK_DESIGNER)");
    expect(page).toContain("<EquipmentDesignerLayer");
  });
});

const planeView = (): GuiWindow =>
  ({
    name: PLANE_DESIGNER,
    sprites: {},
    texts: {},
    fonts: {},
    node: node(PLANE_DESIGNER, "containerwindowtype", {}, [
      node("designer_title", "instanttextboxtype"),
      node("experience", "instanttextboxtype"),
      node("save_button", "buttontype"),
      node("equipments", "dropdownboxtype", {}, [
        node("name", "instanttextboxtype"),
        node("name_reset", "buttontype"),
      ]),
      node("position_version", "instanttextboxtype"),
      node("plane_role", "containerwindowtype", {}, [
        node("role_icon", "icontype"),
        node("role_text", "instanttextboxtype"),
      ]),
      node(
        "equipment_modules",
        "containerwindowtype",
        { position: { x: 13, y: 146 } },
        [
          ...[1, 74, 147, 220, 293].map((x, at) =>
            node(`pos_fixed_module_slot_window_${at}`, "positiontype", {
              position: { x, y: 1 },
            }),
          ),
          ...[1, 74, 147, 220, 293, 366, 439].map((x, at) =>
            node(`pos_custom_module_slot_window_${at}`, "positiontype", {
              position: { x, y: 300 },
            }),
          ),
        ],
      ),
    ]),
  }) as unknown as GuiWindow;

describe("the plane designer", () => {
  const plane = design({
    kind: "plane",
    name: "He 111",
    type: "medium_plane_airframe_1",
    archetype: "medium_plane_airframe",
    type_label: "Basic Medium Airframe",
    role: "Add main armament to give role",
    texts: { title: "Plane Designer", save: "Save £$XPICON$ $XPCOST|Y0$" },
    experience: 0,
    experience_icon: "air_experience",
    version: "B",
    upgrades: [],
    slots: [
      ...[
        "fixed_main_weapon_slot",
        "fixed_auxiliary_weapon_slot_1",
        "fixed_auxiliary_weapon_slot_2",
        "fixed_auxiliary_weapon_slot_3",
      ].map((slot, index) => ({
        slot,
        row: "fixed" as const,
        index,
        required: index === 0,
        module: "",
      })),
      {
        slot: "engine_type_slot",
        row: "custom",
        index: 0,
        required: true,
        module: "engine_1_2x",
        icon: "GFX_EMI_engine_1_2x",
      },
      ...[1, 2, 3, 4].map((n) => ({
        slot: `special_type_slot_${n}`,
        row: "custom" as const,
        index: n,
        required: false,
        module: "",
      })),
    ],
  });

  it("prints a range in whole kilometres (recording 3:22)", () => {
    expect(formatDesignerStat(325, "speed")).toBe("325.0 km/h");
    expect(formatDesignerStat(900, "km")).toBe("900 km");
    expect(formatDesignerStat(0.28, "rate")).toBe("0.28");
    expect(formatDesignerStat(35.5, "")).toBe("35.5");
    expect(formatDesignerStat(0.8, "percent")).toBe("80.0%");
    expect(
      designerSaveText("Save £$XPICON$ $XPCOST|Y0$", 0, "air_experience"),
    ).toBe("Save £air_experience §Y0§!");
  });

  it("writes the role in its own box, paid in air experience", () => {
    const out = equipmentDesignerWindow(planeView(), plane);
    const text = (path: string) => out.texts[idOf(find(out.win, path))];
    expect(text("designer_title")).toBe("Plane Designer");
    expect(text("equipments/name")).toBe("He 111");
    expect(text("plane_role/role_text")).toBe("Add main armament to give role");
    expect(text("experience")).toBe("£air_experience 0");
    expect(text("save_button")).toBe("Save £air_experience §Y0§!");
    expect(text("position_version")).toBe("B");
    expect(out.hidden).toContain(idOf(find(out.win, "plane_role/role_icon")));
    expect(out.hidden).toContain(idOf(find(out.win, "equipments/name_reset")));
  });

  it("puts the fixed slots on top, the rest below, and locks the places left over", () => {
    expect(slotPlace(planeView(), 0, "fixed")).toEqual({ x: 14, y: 147 });
    expect(slotPlace(planeView(), 3, "fixed")).toEqual({ x: 233, y: 147 });
    expect(slotPlace(planeView(), 0, "custom")).toEqual({ x: 14, y: 446 });

    expect(lockedSlots(planeView(), plane.slots)).toEqual([
      { row: "fixed", index: 4 },
      { row: "custom", index: 5 },
      { row: "custom", index: 6 },
    ]);

    expect(
      lockedSlots(view(), [
        { slot: "turret_type_slot", index: 0, required: true, module: "" },
        { slot: "suspension_type_slot", index: 2, required: true, module: "" },
      ]),
    ).toEqual([]);
  });
});

const shipView = (): GuiWindow =>
  ({
    name: SHIP_DESIGNER,
    sprites: {},
    texts: {},
    fonts: {},
    node: node(SHIP_DESIGNER, "containerwindowtype", {}, [
      node("designer_title", "instanttextboxtype"),
      node("production_value", "instanttextboxtype", {
        orientation: "center_right",
      } as Partial<GuiNode>),
      node("production_resources", "instanttextboxtype"),
      node("equipments", "dropdownboxtype", {}, [
        node("name", "instanttextboxtype"),
      ]),
      node("equipment_names", "dropdownboxtype", {}, [
        node("name", "instanttextboxtype"),
        node("expanded_window", "containerwindowtype"),
      ]),
      node("position_version", "instanttextboxtype"),
      node("info", "containerwindowtype", {}, [
        node("ship_designer_naval_dominance_value", "instanttextboxtype"),
      ]),
    ]),
  }) as unknown as GuiWindow;

describe("the ship designer", () => {
  it("prints knots, whole men and a whole percentage (recording 3:36)", () => {
    expect(formatDesignerStat(32.400000000000006, "kn")).toBe("32.4 kn");
    expect(formatDesignerStat(3500, "km")).toBe("3500 km");
    expect(formatDesignerStat(4300, "int")).toBe("4300");
    expect(formatDesignerStat(1, "pct0")).toBe("100%");
    expect(formatDesignerStat(2000 / 36.2, "")).toBe("55.2");
    expect(
      designerCostText(
        "Production Cost: £production_cost $VALUE|2$",
        3824.9999999999995,
        0,
      ),
    ).toBe("Production Cost: £production_cost 3825");
  });

  it("draws only the fitted modules' pictures on the hull, and a tank's slot pictures always", () => {
    const hull = node(
      "equipment_designer_ship_hull_heavy",
      "containerwindowtype",
      {},
      [
        node("module_slots", "containerwindowtype", {}, [
          node("fixed_ship_battery_slot", "containerwindowtype", {}, [
            node("ship_heavy_battery", "containerwindowtype"),
            node("ship_heavy_battery@highlight", "containerwindowtype"),
            node("ship_super_heavy_battery", "containerwindowtype"),
          ]),
          node("fixed_ship_engine_slot", "containerwindowtype", {}, [
            node("heavy_ship_engine", "containerwindowtype"),
            node("heavy_ship_nuclear_engine", "containerwindowtype"),
          ]),
          node("fixed_ship_armor_slot", "containerwindowtype", {}, [
            node("ship_armor_bb", "containerwindowtype"),
            node("ship_heavy_armor", "containerwindowtype"),
          ]),
        ]),
      ],
    );
    const slots = [
      {
        slot: "fixed_ship_battery_slot",
        row: "fixed" as const,
        index: 0,
        required: true,
        module: "",
      },
      {
        slot: "fixed_ship_engine_slot",
        row: "fixed" as const,
        index: 4,
        required: true,
        module: "heavy_ship_engine_2",
        category: "heavy_ship_engine",
      },
      {
        slot: "fixed_ship_armor_slot",
        row: "fixed" as const,
        index: 6,
        required: true,
        module: "ship_armor_bb_1",
        category: "ship_heavy_armor",
      },
    ];
    expect(blueprintUnfitted(hull, slots).sort()).toEqual([
      "fixed_ship_armor_slot/ship_heavy_armor",
      "fixed_ship_battery_slot/ship_heavy_battery",
      "fixed_ship_battery_slot/ship_super_heavy_battery",
      "fixed_ship_engine_slot/heavy_ship_nuclear_engine",
    ]);
    const chassis = node(
      "equipment_designer_medium_tank_chassis",
      "containerwindowtype",
      {},
      [
        node("module_slots", "containerwindowtype", {}, [
          node("main_armament_slot", "containerwindowtype", {}, [
            node("main_armament_slot", "containerwindowtype"),
          ]),
        ]),
      ],
    );
    expect(
      blueprintUnfitted(chassis, [
        { slot: "main_armament_slot", index: 1, required: true, module: "" },
      ]),
    ).toEqual([]);
  });

  it("writes the name list, the version and the dominance cut to a whole number", () => {
    const out = equipmentDesignerWindow(
      shipView(),
      design({
        kind: "ship",
        name: "Basic Heavy Ship",
        type: "ship_hull_heavy_2",
        archetype: "ship_hull_heavy",
        texts: {
          title: "Ship Designer",
          cost: "Production Cost: £production_cost $VALUE|2$",
        },
        cost: 3824.9999999999995,
        cost_decimals: 0,
        version: "A",
        names: "None",
        dominance: 589.75,
        resources: { chromium: 1, steel: 4 },
        resource_order: ["steel", "chromium"],
        upgrades: [],
      }),
    );
    const text = (path: string) => out.texts[idOf(find(out.win, path))];
    expect(text("designer_title")).toBe("Ship Designer");
    expect(text("equipment_names/name")).toBe("None");
    expect(text("position_version")).toBe("A");
    expect(text("info/ship_designer_naval_dominance_value")).toBe("589");
    expect(text("production_value")).toBe(
      "Production Cost: £production_cost 3825",
    );
    expect(text("production_resources")).toBe("4 1");
    expect(out.hidden).toContain(
      idOf(find(out.win, "equipment_names/expanded_window")),
    );
    expect(
      (find(out.win, "production_value") as GuiNode & { orientation?: string })
        .orientation,
    ).toBeUndefined();
  });
});
