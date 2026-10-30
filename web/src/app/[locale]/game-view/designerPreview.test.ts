import { describe, expect, it } from "vitest";
import type { GuiDesigner } from "@/lib/guiRenderApi";
import { changeArrow, deltaTooltip, lowerIsBetter } from "./designerPreview";
import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";

setReferenceTables(REFERENCE_TABLES);

const EMPTY = { stats: {}, equipment: [] } as unknown as GuiDesigner;

const INFANTRY = {
  stats: {
    recovery_rate: 0.3,
    defense: 22,
    breakthrough: 3,
    soft_attack: 6,
    hard_attack: 1,
    supply_use: 0.06,
    suppression: 1.5,
    max_organisation: 60,
    max_strength: 25,
    speed: 4,
    ap_attack: 4,
    reliability: 0.9,
    weight: 0.5,
    production_cost: 50,
    combat_width: 2,
    manpower: 1000,
    training_time: 90,
  },
  equipment: [
    {
      archetype: "infantry_equipment",
      amount: 100,
      short_label: "Infantry Eq.",
    },
  ],
} as unknown as GuiDesigner;

const TEXTS: Record<string, string> = {
  STAT_ARMY_DEFAULT_MORALE: "Recovery Rate: ",
  STAT_ARMY_DEFENSE: "Defense: ",
  STAT_ARMY_BREAKTHROUGH: "Breakthrough: ",
  STAT_ARMY_SOFT_ATTACK: "Soft Attack: ",
  STAT_ARMY_HARD_ATTACK: "Hard Attack: ",
  STAT_ARMY_SUPPLY_CONSUMPTION: "Supply Use: ",
  STAT_ARMY_SUPRESSION: "Suppression: ",
  STAT_COMMON_MAX_ORG: "Organization: ",
  STAT_COMMON_MAX_STRENGTH: "HP: ",
  STAT_COMMON_MAXIMUM_SPEED: "Max Speed: ",
  STAT_COMMON_PIERCING: "Piercing: ",
  STAT_COMMON_RELIABILITY: "Reliability: ",
  STAT_COMMON_WEIGHT: "Weight: ",
  STAT_COMMON_BUILD_COST_IC: "£production_cost  Production Cost: ",
  DESIGNER_COMBATWIDTH: "Combat Width: ",
  DESIGNER_MANPOWER: "Manpower: ",
  DESIGNER_TRAINING_TIME: "Training Time: ",
};

describe("the picker's preview", () => {
  it("points the arrow the way the number went, green when that is better (designe 0:14)", () => {
    expect(changeArrow(0, 4, lowerIsBetter("speed"))).toBe(
      "£increase_texticon",
    );
    expect(changeArrow(0, 0.5, lowerIsBetter("weight"))).toBe(
      "£decrease_reversed_texticon",
    );
    expect(changeArrow(4, 2, lowerIsBetter("speed"))).toBe(
      "£decrease_texticon",
    );
    expect(changeArrow(2, 1, lowerIsBetter("combat_width"))).toBe(
      "£increase_reversed_texticon",
    );
    expect(changeArrow(3, 3, false)).toBe("");
  });

  it("lists what one rifle battalion adds, line for line as the recording's tooltip", () => {
    expect(
      deltaTooltip("Infantry", EMPTY, INFANTRY, TEXTS).split("\n"),
    ).toEqual([
      "§YInfantry§!",
      "Recovery Rate:  §G+0.30§!",
      "Defense:  §G+22.0§!",
      "Breakthrough:  §G+3.0§!",
      "Soft Attack:  §G+6.0§!",
      "Hard Attack:  §G+1.0§!",
      "Supply Use:  §R+0.06§!",
      "Suppression:  §G+1.5§!",
      "Organization:  §G+60.0§!",
      "HP:  §G+25.0§!",
      "Max Speed:  §G+4.0§! km/h",
      "Piercing:  §G+4.0§!",
      "Reliability:  §G+90.0%§!",
      "Weight:  §R+0.5§!",
      "£production_cost  Production Cost:  §R+50.00§!",
      "Combat Width:  §R+2.0§!",
      "Manpower:  §Y1000§!",
      "Training Time:  §R+90§!",
      "Infantry Eq.:  §Y100§!",
    ]);
  });

  it("says nothing of what does not move", () => {
    expect(deltaTooltip("Infantry", INFANTRY, INFANTRY, TEXTS)).toBe(
      "§YInfantry§!",
    );
  });
});

describe("a swap's tooltip", () => {
  it("puts hardness and fuel where the Heavy Tank's does (designe 0:24)", () => {
    const light = {
      stats: { hardness: 0.8, fuel_capacity: 10, fuel_use: 0.5, weight: 1 },
      equipment: [],
    } as unknown as GuiDesigner;
    const heavy = {
      stats: { hardness: 0.95, fuel_capacity: 20, fuel_use: 2.8, weight: 2 },
      equipment: [],
    } as unknown as GuiDesigner;
    const texts = {
      STAT_ARMY_HARDNESS: "Hardness: ",
      STAT_ARMY_FUEL_CAPACITY: "Fuel Capacity: ",
      STAT_COMMON_WEIGHT: "Weight: ",
      STAT_COMMON_FUEL_CONSUMPTION: "Fuel Usage: ",
    };
    expect(deltaTooltip("Heavy Tank", light, heavy, texts).split("\n")).toEqual(
      [
        "§YHeavy Tank§!",
        "Hardness:  §G+15.0%§!",
        "Fuel Capacity:  §G+10.00§!",
        "Weight:  §R+1.0§!",
        "Fuel Usage:  §R+2.30§!",
      ],
    );
  });
});

describe("the grid's slot tooltips", () => {
  const SLOT_TEXTS: Record<string, string> = {
    DESIGNER_ADD: "§GClick§! to add",
    DESIGNER_ADD_COPY_ABOVE:
      "§GShift + Click§! to add a copy of the slot above",
    DESIGNER_ADD_FILL_REGIMENT:
      "§GCtrl + Click§! to fill the rest of the regiment with copies of the slot above",
    DESIGNER_ADD_FILL_REGIMENT_FROM_PICK:
      "§GCtrl + Click§! to fill the entire regiment with the battalion you select",
    DESIGNER_ADD_COPY_LEFT_REGIMENT:
      "§GShift + Click§! to add a copy of the first battalion from the regiment to the left",
    DESIGNER_ADD_FILL_REGIMENT_FROM_LEFT:
      "§GCtrl + Shift + Click§! to fill the regiment with copies of the first battalion from the regiment to the left",
    DESIGNER_CHANGE: "§GClick§! to change",
    DESIGNER_CHANGE_REMOVE: "§GShift + Click§! to remove",
    DESIGNER_REMOVE_REGIMENT:
      "§GCtrl + Click§! to remove every battalion in this regiment",
    DESIGNER_BLOCKED_REGIMENTAL_SUPPORT:
      "This Regimental Support slot is §Rlocked§!",
    DESIGNER_BLOCKED_BY_REGIMENT_BATTALIONS:
      "§R- Requires at least §!§H$NUM_BATTALIONS|H$ Line Battalions§!§R in the regiment.§!",
  };

  it("offers the left regiment's first battalion on a new regiment's '+' (designe 0:35)", async () => {
    const { slotHints } = await import("./designerPreview");
    expect(
      slotHints({ kind: "add", x: 1, y: 0 }, "regiment", true, SLOT_TEXTS),
    ).toEqual([
      SLOT_TEXTS.DESIGNER_ADD,
      SLOT_TEXTS.DESIGNER_ADD_FILL_REGIMENT_FROM_PICK,
      SLOT_TEXTS.DESIGNER_ADD_COPY_LEFT_REGIMENT,
      SLOT_TEXTS.DESIGNER_ADD_FILL_REGIMENT_FROM_LEFT,
    ]);
    expect(
      slotHints({ kind: "add", x: 0, y: 1 }, "regiment", false, SLOT_TEXTS)[1],
    ).toBe(SLOT_TEXTS.DESIGNER_ADD_COPY_ABOVE);
    expect(
      slotHints({ kind: "add", x: 0, y: 2 }, "support", false, SLOT_TEXTS),
    ).toEqual([SLOT_TEXTS.DESIGNER_ADD]);
  });

  it("says a locked regimental support slot wants three line battalions (1:06)", async () => {
    const { slotHints } = await import("./designerPreview");
    expect(
      slotHints(
        { kind: "locked", x: 0, y: 0 },
        "regimental",
        false,
        SLOT_TEXTS,
      ),
    ).toEqual([
      "This Regimental Support slot is §Rlocked§!",

      "§R- Requires at least §!§H§H3§! Line Battalions§!§R in the regiment.§!",
    ]);
  });

  it("names a battalion, its clicks, then what it brings (0:19)", async () => {
    const { slotHints, slotTooltip } = await import("./designerPreview");
    const hints = slotHints(
      { kind: "unit", x: 0, y: 0 },
      "regiment",
      false,
      SLOT_TEXTS,
    );
    const tip = slotTooltip("Infantry", hints, INFANTRY, TEXTS).split("\n");
    expect(tip.slice(0, 5)).toEqual([
      "§YInfantry§!",
      "§GClick§! to change",
      "§GShift + Click§! to remove",
      "§GCtrl + Click§! to remove every battalion in this regiment",
      "",
    ]);
    expect(tip[5]).toBe("Recovery Rate:  §G+0.30§!");
  });
});

describe("the Army HQ Designer's tooltips (recording 1:48–1:58)", () => {
  const HQ_TEXTS: Record<string, string> = {
    ...TEXTS,
    DESIGNER_SUBUNIT_DEPLOYMENT_COST:
      "£command_power HQ deployment cost: $COST|H1$",
    DESIGNER_DEPLOYED_LEADER_MODIFIERS: "§HCommander bonuses:§!",
    ENABLES_ABILITY: "Enables Ability: $NAME|H$",
  };
  const STAFF = {
    type: "hq_engineer",
    group: "support",
    label: "Engineer Staff",
    deployment_cost: 1,
    commander_bonuses: [
      "Invasion Defense: §G+5%§!",
      "Fort Damage from Combat: §G+20.0%§!",
    ],
    abilities: ["Prepare Fallback Positions"],
  };

  it("puts the deployment cost under the name, then the commander's bonuses and abilities", async () => {
    const { hqPickerTooltip } = await import("./designerPreview");
    const tip = hqPickerTooltip(
      deltaTooltip("Engineer Staff", EMPTY, INFANTRY, HQ_TEXTS),
      STAFF,
      HQ_TEXTS,
    ).split("\n");
    expect(tip.slice(0, 3)).toEqual([
      "§YEngineer Staff§!",
      "£command_power HQ deployment cost: §H1.0§!",
      "Recovery Rate:  §G+0.30§!",
    ]);
    expect(tip.slice(-7)).toEqual([
      "",
      "§HCommander bonuses:§!",
      "Invasion Defense: §G+5%§!",
      "Fort Damage from Combat: §G+20.0%§!",
      "",
      "§HEnables Ability:§!",
      "§G+Prepare Fallback Positions§!",
    ]);
  });

  it("gives an HQ battalion's slot its deployment cost, and the '+' the copy's numbers without a name", async () => {
    const { deploymentLine, slotTooltip } = await import("./designerPreview");
    const lead = [
      deploymentLine(
        { type: "hq_armored_car", group: "mobile", deployment_cost: 3 },
        HQ_TEXTS,
      ) ?? "",
    ];
    const remove = slotTooltip(
      "Armored Car HQ Security",
      ["§GClick§! to remove"],
      INFANTRY,
      HQ_TEXTS,
      lead,
    ).split("\n");
    expect(remove.slice(0, 5)).toEqual([
      "§YArmored Car HQ Security§!",
      "§GClick§! to remove",
      "",
      "£command_power HQ deployment cost: §H3.0§!",
      "Recovery Rate:  §G+0.30§!",
    ]);
    const add = slotTooltip(
      null,
      ["§GClick§! to add another §HArmored Car HQ Security§! battalion"],
      INFANTRY,
      HQ_TEXTS,
    ).split("\n");
    expect(add.slice(0, 3)).toEqual([
      "§GClick§! to add another §HArmored Car HQ Security§! battalion",
      "",
      "Recovery Rate:  §G+0.30§!",
    ]);
  });
});
