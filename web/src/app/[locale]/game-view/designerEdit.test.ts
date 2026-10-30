import { describe, expect, it } from "vitest";
import {
  EMPTY_DRAFT,
  designerButtonsOff,
  designerSavePayload,
  encodeCells,
  filledRegiment,
  lineGroups,
  pickerEntries,
  placed,
  regimentSlots,
  regimentalSupportSlots,
  removed,
  removedRegiment,
  sameDraft,
  saveCost,
  swappedAlone,
  supportSlots,
  type DesignerDraft,
  type DesignerUnit,
} from "./designerEdit";

const UNITS: DesignerUnit[] = [
  { type: "anti_air", group: "support", divisional: true },
  { type: "anti_air_brigade", group: "combat_support" },
  { type: "mot_anti_air_brigade", group: "mobile_combat_support" },
  { type: "armored_car", group: "mobile" },
  { type: "artillery", group: "support", divisional: true },
  { type: "artillery_brigade", group: "combat_support" },
  { type: "cavalry", group: "mobile" },
  { type: "engineer", group: "support", divisional: true },
  { type: "field_guns", group: "support", divisional: false },
  { type: "heavy_armor", group: "armor" },
  { type: "infantry", group: "infantry" },
  { type: "mountaineers", group: "infantry" },
  { type: "motorized", group: "mobile" },
  { type: "light_armor", group: "armor" },
];
const COSTS = { group: 20, change: 5, support: 10 };

const infantry = (x: number, y: number) => ({ type: "infantry", x, y });

const INFANTERIE: DesignerDraft = {
  regiments: [0, 1, 2].flatMap((x) => [0, 1, 2].map((y) => infantry(x, y))),
  support: [
    { type: "engineer", x: 0, y: 0 },
    { type: "artillery", x: 0, y: 1 },
  ],
};

const kinds = (slots: { x: number; y: number; kind: string }[]) =>
  slots.map((slot) => `${slot.x},${slot.y},${slot.kind}`);

describe("the designer's slots", () => {
  it("opens an empty template on one '+' and locks (recording 0:00)", () => {
    expect(kinds(regimentSlots(EMPTY_DRAFT))).toEqual([
      "0,0,add",
      "0,1,locked",
      "1,0,locked",
      "2,0,locked",
      "3,0,locked",
      "4,0,locked",
    ]);
    expect(kinds(supportSlots(EMPTY_DRAFT))).toEqual(["0,0,add", "0,1,locked"]);
    expect(
      kinds(regimentalSupportSlots(EMPTY_DRAFT)).every((slot) =>
        slot.endsWith("locked"),
      ),
    ).toBe(true);
  });

  it("draws Infanterie-Division as the game does (recording 0:07)", () => {
    const slots = kinds(regimentSlots(INFANTERIE));

    expect(slots.filter((slot) => slot.startsWith("0,"))).toEqual([
      "0,0,unit",
      "0,1,unit",
      "0,2,unit",
      "0,3,add",
      "0,4,blocked",
    ]);
    expect(slots.filter((slot) => slot.startsWith("3,"))).toEqual([
      "3,0,add",
      "3,1,locked",
    ]);
    expect(slots.filter((slot) => slot.startsWith("4,"))).toEqual([
      "4,0,locked",
    ]);
    expect(kinds(supportSlots(INFANTERIE))).toEqual([
      "0,0,unit",
      "0,1,unit",
      "0,2,add",
      "0,3,locked",
    ]);

    expect(kinds(regimentalSupportSlots(INFANTERIE))).toEqual([
      "0,0,add",
      "1,0,add",
      "2,0,add",
      "3,0,locked",
      "4,0,locked",
    ]);
  });

  it("encodes a template for the service", () => {
    expect(encodeCells(EMPTY_DRAFT)).toBe("-");
    expect(
      encodeCells({
        regiments: [infantry(0, 0)],
        support: [{ type: "engineer", x: 0, y: 0 }],
      }),
    ).toBe("r:0:0:infantry,s:0:0:engineer");
  });
});

describe("the battalion picker", () => {
  it("files the groups in the order the files first name one (recording 0:13)", () => {
    expect(lineGroups(UNITS)).toEqual([
      "combat_support",
      "mobile_combat_support",
      "mobile",
      "armor",
      "infantry",
    ]);
  });

  it("starts an empty regiment on its groups, free", () => {
    const entries = pickerEntries(
      EMPTY_DRAFT,
      { grid: "regiment", x: 0, y: 0, group: null },
      UNITS,
      COSTS,
    );
    expect(
      entries.map((entry) =>
        entry.kind === "group" ? `${entry.group}:${entry.cost}` : entry.kind,
      ),
    ).toEqual([
      "combat_support:null",
      "mobile_combat_support:null",
      "mobile:null",
      "armor:null",
      "infantry:null",
    ]);
  });

  it("lists a group's battalions at 5 to add", () => {
    const entries = pickerEntries(
      EMPTY_DRAFT,
      { grid: "regiment", x: 0, y: 0, group: "infantry" },
      UNITS,
      COSTS,
    );
    expect(entries).toEqual([
      { kind: "unit", unit: UNITS[10], line: "add", cost: 5 },
      { kind: "unit", unit: UNITS[11], line: "add", cost: 5 },
    ]);
  });

  it("leads a filled slot with Remove and offers the other groups as a new group (recording 0:17)", () => {
    const one: DesignerDraft = { regiments: [infantry(0, 0)], support: [] };
    const entries = pickerEntries(
      one,
      { grid: "regiment", x: 0, y: 0, group: null },
      UNITS,
      COSTS,
    );
    expect(entries[0]).toEqual({ kind: "remove", cost: -5 });
    expect(
      entries
        .slice(1)
        .map((entry) =>
          entry.kind === "group" ? `${entry.group}:${entry.cost}` : entry.kind,
        ),
    ).toEqual([
      "combat_support:25",
      "mobile_combat_support:25",
      "mobile:25",
      "armor:25",
      "infantry:null",
    ]);
    const own = pickerEntries(
      one,
      { grid: "regiment", x: 0, y: 0, group: "infantry" },
      UNITS,
      COSTS,
    );
    expect(
      own
        .slice(1)
        .map((entry) =>
          entry.kind === "unit"
            ? `${entry.unit.type}:${entry.line}:${entry.cost}`
            : "",
        ),
    ).toEqual(["infantry:selected:0", "mountaineers:replace:0"]);
  });

  it("goes straight to the regiment's own battalions from a '+' under them", () => {
    const one: DesignerDraft = { regiments: [infantry(0, 0)], support: [] };
    const entries = pickerEntries(
      one,
      { grid: "regiment", x: 0, y: 1, group: null },
      UNITS,
      COSTS,
    );
    expect(
      entries.map((entry) =>
        entry.kind === "unit" ? entry.unit.type : entry.kind,
      ),
    ).toEqual(["infantry", "mountaineers"]);
  });

  it("offers the support companies not already in, at 10, and never a regiment-only one", () => {
    const entries = pickerEntries(
      INFANTERIE,
      { grid: "support", x: 0, y: 2, group: null },
      UNITS,
      COSTS,
    );
    expect(
      entries.map((entry) =>
        entry.kind === "unit" ? `${entry.unit.type}:${entry.cost}` : entry.kind,
      ),
    ).toEqual(["anti_air:10"]);
    const swap = pickerEntries(
      INFANTERIE,
      { grid: "support", x: 0, y: 0, group: null },
      UNITS,
      COSTS,
    );
    expect(swap[0]).toEqual({ kind: "remove", cost: -10 });
  });
});

describe("building a template", () => {
  it("puts a battalion in, swaps it, takes it out", () => {
    const one = placed(
      EMPTY_DRAFT,
      { grid: "regiment", x: 0, y: 0 },
      "infantry",
      UNITS,
    );
    expect(one.regiments).toEqual([infantry(0, 0)]);
    const swapped = placed(
      one,
      { grid: "regiment", x: 0, y: 0 },
      "mountaineers",
      UNITS,
    );
    expect(swapped.regiments).toEqual([{ type: "mountaineers", x: 0, y: 0 }]);
    expect(
      removed(swapped, { grid: "regiment", x: 0, y: 0 }).regiments,
    ).toEqual([]);
  });

  it("changes a whole regiment when a battalion of another group goes in", () => {
    const two: DesignerDraft = {
      regiments: [infantry(0, 0), infantry(0, 1)],
      support: [],
    };
    const armoured = placed(
      two,
      { grid: "regiment", x: 0, y: 0 },
      "light_armor",
      UNITS,
    );
    expect(armoured.regiments.map((cell) => cell.type)).toEqual([
      "light_armor",
      "light_armor",
    ]);
  });

  it("swaps one slot alone for the picker's tooltip, where the pick turns the regiment (1:48)", () => {
    const two: DesignerDraft = {
      regiments: [infantry(0, 0), infantry(0, 1)],
      support: [],
    };
    expect(
      swappedAlone(
        two,
        { grid: "regiment", x: 0, y: 0 },
        "light_armor",
      ).regiments.map((cell) => cell.type),
    ).toEqual(["light_armor", "infantry"]);
    expect(
      swappedAlone(two, { grid: "regiment", x: 0, y: 5 }, "light_armor"),
    ).toBe(two);
  });

  it("moves the regiments on its right over when one is emptied", () => {
    const two: DesignerDraft = {
      regiments: [infantry(0, 0), { type: "cavalry", x: 1, y: 0 }],
      support: [],
    };
    expect(removed(two, { grid: "regiment", x: 0, y: 0 }).regiments).toEqual([
      { type: "cavalry", x: 0, y: 0 },
    ]);
  });

  it("costs what the recording's Save button says: 5, then 15 with a support company", () => {
    const one = placed(
      EMPTY_DRAFT,
      { grid: "regiment", x: 0, y: 0 },
      "infantry",
      UNITS,
    );
    expect(saveCost(EMPTY_DRAFT, one, UNITS, COSTS)).toBe(5);

    const tank = placed(
      one,
      { grid: "regiment", x: 0, y: 0 },
      "heavy_armor",
      UNITS,
    );
    expect(saveCost(EMPTY_DRAFT, tank, UNITS, COSTS)).toBe(5);
    const recon = placed(
      tank,
      { grid: "support", x: 0, y: 0 },
      "engineer",
      UNITS,
    );
    expect(saveCost(EMPTY_DRAFT, recon, UNITS, COSTS)).toBe(15);

    const armoured = placed(
      INFANTERIE,
      { grid: "regiment", x: 0, y: 0 },
      "light_armor",
      UNITS,
    );
    expect(saveCost(INFANTERIE, armoured, UNITS, COSTS)).toBe(35);
    expect(saveCost(INFANTERIE, INFANTERIE, UNITS, COSTS)).toBe(0);
  });

  it("counts a saved template's battalions against it: AOK (recording 1:42–2:00)", () => {
    const hq: DesignerUnit[] = [
      { type: "hq_infantry", group: "infantry" },
      { type: "hq_armored_car", group: "mobile" },
      { type: "hq_light_armor", group: "armor" },
      { type: "hq_medium_armor", group: "armor" },
      { type: "hq_engineer", group: "support", divisional: true },
    ];
    const tank = (y: number) => ({ type: "hq_light_armor", x: 0, y });
    const aok: DesignerDraft = { regiments: [tank(0), tank(1)], support: [] };
    const lead = { grid: "regiment" as const, x: 0, y: 0 };

    const groups = pickerEntries(aok, { ...lead, group: null }, hq, COSTS, aok);
    expect(groups[0]).toEqual({ kind: "remove", cost: 5 });
    const armour = pickerEntries(
      aok,
      { ...lead, group: "armor" },
      hq,
      COSTS,
      aok,
    );
    expect(
      armour
        .slice(1)
        .map((entry) =>
          entry.kind === "unit"
            ? `${entry.unit.type}:${entry.line}:${entry.cost}`
            : "",
        ),
    ).toEqual(["hq_light_armor:selected:0", "hq_medium_armor:replace:10"]);
    const mobile = pickerEntries(
      aok,
      { ...lead, group: "mobile" },
      hq,
      COSTS,
      aok,
    );
    expect(mobile[1]).toMatchObject({
      kind: "unit",
      line: "replace",
      cost: 30,
    });

    const cars = placed(aok, lead, "hq_armored_car", hq);
    expect(saveCost(aok, cars, hq, COSTS)).toBe(30);
    expect(
      pickerEntries(cars, { ...lead, group: null }, hq, COSTS, aok)[0],
    ).toEqual({ kind: "remove", cost: 0 });
    const three = placed(
      cars,
      { grid: "regiment", x: 0, y: 2 },
      "hq_armored_car",
      hq,
    );
    const four = placed(
      three,
      { grid: "regiment", x: 0, y: 3 },
      "hq_armored_car",
      hq,
    );
    expect([
      saveCost(aok, three, hq, COSTS),
      saveCost(aok, four, hq, COSTS),
    ]).toEqual([35, 40]);
    const staff = placed(
      four,
      { grid: "support", x: 0, y: 0 },
      "hq_engineer",
      hq,
    );
    expect(saveCost(aok, staff, hq, COSTS)).toBe(50);
  });
});

describe("the bottom bar", () => {
  it("switches Reset, Duplicate and Save the way the recording shows them", () => {
    expect(
      designerButtonsOff({ dirty: false, fresh: true, cost: 0, experience: 0 }),
    ).toEqual(["reset_button", "duplicate_button", "save_button"]);

    expect(
      designerButtonsOff({
        dirty: false,
        fresh: false,
        cost: 0,
        experience: 0,
      }),
    ).toEqual(["reset_button", "save_button"]);

    expect(
      designerButtonsOff({ dirty: true, fresh: true, cost: 5, experience: 0 }),
    ).toEqual(["duplicate_button", "save_button"]);
    expect(
      designerButtonsOff({
        dirty: true,
        fresh: false,
        cost: 5,
        experience: 12,
      }),
    ).toEqual([]);

    expect(
      designerButtonsOff({
        dirty: true,
        fresh: true,
        cost: 5,
        experience: 0,
        editing: true,
      }),
    ).toEqual(["duplicate_button"]);

    expect(
      designerButtonsOff({
        dirty: false,
        fresh: true,
        cost: 0,
        experience: 0,
        empty: true,
      })[0],
    ).toBe("division_equipment_btn");
  });

  it("does not count the order battalions went in as a change", () => {
    const a: DesignerDraft = {
      regiments: [infantry(0, 0), infantry(0, 1)],
      support: [],
    };
    const b: DesignerDraft = {
      regiments: [infantry(0, 1), infantry(0, 0)],
      support: [],
    };
    expect(sameDraft(a, b)).toBe(true);
    expect(sameDraft(a, EMPTY_DRAFT)).toBe(false);

    expect(sameDraft({ ...a, priority: 1 }, a)).toBe(true);
    expect(sameDraft({ ...a, priority: 0 }, a)).toBe(false);
    expect(sameDraft({ ...a, namesGroup: "GER_LTARM_01" }, a)).toBe(false);
  });
});

describe("what the slots' Shift and Ctrl clicks do", () => {
  it("empties a whole regiment and moves the next one over (Ctrl + Click)", () => {
    const two: DesignerDraft = {
      regiments: [
        infantry(0, 0),
        infantry(0, 1),
        { type: "cavalry", x: 1, y: 0 },
      ],
      support: [],
    };
    expect(removedRegiment(two, 0).regiments).toEqual([
      { type: "cavalry", x: 0, y: 0 },
    ]);
  });

  it("fills the rest of a regiment with copies of the slot above, or all of it", () => {
    const one: DesignerDraft = { regiments: [infantry(0, 0)], support: [] };
    expect(filledRegiment(one, 0, "infantry", UNITS, 1).regiments).toEqual(
      [0, 1, 2, 3].map((y) => infantry(0, y)),
    );
    const mixed = filledRegiment(one, 1, "mountaineers", UNITS);
    expect(
      mixed.regiments.filter((cell) => cell.x === 1).map((cell) => cell.type),
    ).toEqual(["mountaineers", "mountaineers", "mountaineers", "mountaineers"]);
  });
});

describe("the Army HQ Designer's slots (designe 1:41)", () => {
  it("draws AOK: a lead tank, a tank, '+', a lock; three companies and '+'", async () => {
    const { hqRegimentSlots, hqSupportSlots } = await import("./designerEdit");
    const aok: DesignerDraft = {
      regiments: [
        { type: "hq_light_armor", x: 0, y: 0 },
        { type: "hq_light_armor", x: 0, y: 1 },
      ],
      support: [
        { type: "hq_support_company", x: 0, y: 0 },
        { type: "hq_air_liaison", x: 0, y: 1 },
        { type: "hq_recon", x: 0, y: 2 },
      ],
    };
    const { lead, row } = hqRegimentSlots(aok);
    expect(lead.map((slot) => slot.kind)).toEqual(["unit"]);
    expect(row.map((slot) => `${slot.kind}@${slot.at?.x}`)).toEqual([
      "unit@0",
      "add@1",
      "locked@2",
    ]);

    expect(row[1].y).toBe(2);
    expect(hqSupportSlots(aok).map((slot) => slot.kind)).toEqual([
      "unit",
      "unit",
      "unit",
      "add",
    ]);
  });

  it("opens an empty Army HQ on its lead's '+'", async () => {
    const { hqRegimentSlots } = await import("./designerEdit");
    const { lead, row } = hqRegimentSlots(EMPTY_DRAFT);
    expect(lead[0].kind).toBe("add");
    expect(row.every((slot) => slot.kind === "locked")).toBe(true);
  });
});

describe("saving a template from the designer", () => {
  it("lets edit mode save what the scenario's army experience could not pay for", () => {
    const base = { dirty: true, fresh: true, cost: 5, experience: 0 };
    expect(designerButtonsOff(base)).toContain("save_button");
    expect(designerButtonsOff({ ...base, editing: true })).not.toContain(
      "save_button",
    );
    expect(
      designerButtonsOff({ ...base, editing: true, dirty: false }),
    ).toContain("save_button");
  });

  it("counts a new name as a change, and the template's own as none", () => {
    expect(
      sameDraft({ ...INFANTERIE, name: "Leichte-Division" }, INFANTERIE),
    ).toBe(false);
    expect(sameDraft({ ...INFANTERIE, name: undefined }, INFANTERIE)).toBe(
      true,
    );
  });

  it("sends the cells, and the name, names group and priority it has or the template's own", () => {
    const own = {
      tag: "GER",
      date: "1936.1.1.12",
      template: "Infanterie-Division",
      name: "Infanterie-Division",
      namesGroup: "GER_INF_01",
      priority: 1,
    };
    const kept = designerSavePayload(INFANTERIE, own);
    expect(kept).toMatchObject({
      tag: "GER",
      template: "Infanterie-Division",
      name: "Infanterie-Division",
      names_group: "GER_INF_01",
      priority: 1,
    });
    expect(kept.regiments).toHaveLength(9);
    expect(kept.support[0]).toEqual({ type: "engineer", x: 0, y: 0 });
    const changed = designerSavePayload(
      {
        ...INFANTERIE,
        name: "  Leichte-Division ",
        namesGroup: null,
        priority: 2,
      },
      own,
    );
    expect(changed).toMatchObject({
      name: "Leichte-Division",
      names_group: "",
      priority: 2,
    });
    expect(
      designerSavePayload(INFANTERIE, { ...own, template: "", date: null }),
    ).not.toHaveProperty("date");
  });
});
