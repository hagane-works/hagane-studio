import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import type { MIOTraitEntry } from "@/lib/api/military";
import type { GuiIndustrialOrgTrait } from "./IndustrialOrgDetailLayer";
import {
  fieldText,
  liftField,
  parseBonus,
  storedTrait,
  traitEntryIndex,
  traitStart,
  traitsWithEntry,
} from "./mioTraitEdit";

const traits = (): MIOTraitEntry[] => [
  {
    kind: "override",
    token: "generic_mio_trait_wide_tracks",
    raw_extras: { position: "x = 0\ny = 0" },
  },
  {
    kind: "add",
    token: "GER_mio_trait_anti_tank_specialization",
    name: "GER_mio_trait_anti_tank_specialization",
    relative_position_id: "generic_mio_trait_one_piece_hull_side_coverings",
    raw_extras: {
      position: "x = 1\ny = 1",
      equipment_bonus: "hard_attack = 0.05\nap_attack = 0.05",
      available: "\n",
      visible: "FROM = {\n\toriginal_tag = GER\n}",
    },
  },
];

const shown = (
  extra: Partial<GuiIndustrialOrgTrait> = {},
): GuiIndustrialOrgTrait => ({
  token: "GER_mio_trait_anti_tank_specialization",
  label: "Anti-Tank Specialization",
  x: 4,
  y: 1,
  special: true,
  bonuses: ["design"],
  parents: [],
  any_parent: [],
  exclusive: [],
  values: {
    equipment_bonus: [
      { key: "hard_attack", value: 0.05 },
      { key: "ap_attack", value: 0.05 },
    ],
  },
  relative_to: "generic_mio_trait_one_piece_hull_side_coverings",
  ...extra,
});

const OWN = "GER_mio_trait_anti_tank_specialization";
const ANCHOR = { x: 3, y: 0 };

describe("a design company's trait in the game view", () => {
  it("finds the company's own trait, or its override of an inherited one", () => {
    const row = { traits: traits() };
    expect(traitEntryIndex(row, OWN)).toEqual({ index: 1, own: true });
    expect(traitEntryIndex(row, "generic_mio_trait_wide_tracks")).toEqual({
      index: 0,
      own: false,
    });
    expect(traitEntryIndex(row, "generic_mio_trait_fully_cast_turret")).toEqual(
      { index: -1, own: false },
    );
  });

  it("reads a block the editor has a field for, and leaves the rest as text", () => {
    expect(parseBonus("soft_attack = 0.05\n\n reliability = -0.02 ")).toEqual({
      soft_attack: 0.05,
      reliability: -0.02,
    });
    expect(parseBonus("soft_attack 0.05")).toBeNull();
    expect(liftField("position", "x = 9\ny = -1")).toEqual({ x: 9, y: -1 });
    expect(liftField("position", "x = 9\ny = @offset")).toBeUndefined();
    expect(
      liftField("equipment_bonus", "if = { limit = { a = b } }"),
    ).toBeUndefined();
    expect(liftField("all_parents", "trait_a\n trait_b")).toEqual([
      "trait_a",
      "trait_b",
    ]);
    expect(liftField("available", "\n has_mio_size > 2 \n")).toBe(
      "has_mio_size > 2",
    );
    expect(liftField("limit_to_equipment_type", "tank")).toBeUndefined();

    expect(fieldText("all_parents", ["trait_a"])).toBe("trait_a\n");
    expect(fieldText("production_bonus", {})).toBe("\n");
  });

  it("starts the editor from the own trait's blocks, lifted into its fields", () => {
    const start = traitStart(traits()[1], OWN, shown(), ANCHOR);
    expect(start.trait).toMatchObject({
      position: { x: 1, y: 1 },
      equipment_bonus: { hard_attack: 0.05, ap_attack: 0.05 },
      available: "",
      visible: "FROM = {\n\toriginal_tag = GER\n}",
      raw_extras: {},
    });

    expect(start.inherited).toEqual({});
  });

  it("puts each block back where the file had it: as written when unchanged, as text when changed", () => {
    const start = traitStart(traits()[1], OWN, shown(), ANCHOR);
    expect(storedTrait(start.trait, start)).toEqual(traits()[1]);
    expect(
      traitsWithEntry(
        { traits: traits() },
        OWN,
        storedTrait(start.trait, start),
      ),
    ).toEqual(traits());

    const edited = {
      ...start.trait,
      position: { x: 2, y: 1 },
      equipment_bonus: { hard_attack: 0.1 },
      available: "has_mio_size > 2",
    };
    const stored = storedTrait(edited, start);
    expect(stored.raw_extras).toEqual({
      position: "x = 2\ny = 1",
      equipment_bonus: "hard_attack = 0.1",
      available: "has_mio_size > 2",
      visible: "FROM = {\n\toriginal_tag = GER\n}",
    });
    expect(Object.keys(stored.raw_extras ?? {})).toEqual([
      "position",
      "equipment_bonus",
      "available",
      "visible",
    ]);
    for (const field of ["position", "equipment_bonus", "available", "visible"])
      expect(stored).not.toHaveProperty(field);
    const out = traitsWithEntry({ traits: traits() }, OWN, stored);
    expect(out[1]).toBe(stored);
    expect(out[0]).toEqual(traits()[0]);
  });

  it("keeps an emptied bonus as an empty block, and drops a cleared script", () => {
    const start = traitStart(traits()[1], OWN, shown(), ANCHOR);
    const edited: MIOTraitEntry = { ...start.trait };
    delete edited.equipment_bonus;
    delete edited.visible;
    const stored = storedTrait(edited, start);
    expect(stored.raw_extras?.equipment_bonus).toBe("\n");
    expect(stored.raw_extras).not.toHaveProperty("visible");
  });

  it("starts an inherited trait's override from what the tree shows, and names only what changed", () => {
    const turret = shown({
      token: "generic_mio_trait_fully_cast_turret",
      x: 4,
      y: 0,
      relative_to: undefined,
      values: {
        production_bonus: [{ key: "production_capacity_factor", value: 0.05 }],
      },
    });
    const start = traitStart(undefined, turret.token, turret, null);
    expect(start.trait).toEqual({
      kind: "override",
      token: "generic_mio_trait_fully_cast_turret",
      position: { x: 4, y: 0 },
      production_bonus: { production_capacity_factor: 0.05 },
    });

    expect(
      traitsWithEntry(
        { traits: traits() },
        turret.token,
        storedTrait(start.trait, start),
      ),
    ).toEqual(traits());

    const moved = traitsWithEntry(
      { traits: traits() },
      turret.token,
      storedTrait({ ...start.trait, position: { x: 6, y: 0 } }, start),
    );
    expect(moved[2]).toEqual({
      kind: "override",
      token: "generic_mio_trait_fully_cast_turret",
      position: { x: 6, y: 0 },
    });

    const more = storedTrait(
      {
        ...start.trait,
        production_bonus: {
          production_capacity_factor: 0.05,
          production_cost_factor: -0.1,
        },
      },
      start,
    );
    expect(more).toEqual({
      kind: "override",
      token: "generic_mio_trait_fully_cast_turret",
      production_bonus: {
        production_capacity_factor: 0.05,
        production_cost_factor: -0.1,
      },
    });
    const emptied: MIOTraitEntry = { ...start.trait };
    delete emptied.production_bonus;
    expect(storedTrait(emptied, start)).toEqual({
      kind: "override",
      token: "generic_mio_trait_fully_cast_turret",
      raw_extras: { production_bonus: "\n" },
    });
  });

  it("fills what an existing override does not name, its own cell lifted", () => {
    const tracks = shown({
      token: "generic_mio_trait_wide_tracks",
      x: 0,
      y: 0,
      relative_to: undefined,
      values: { equipment_bonus: [{ key: "reliability", value: 0.1 }] },
    });
    const start = traitStart(traits()[0], tracks.token, tracks, null);
    expect(start.trait).toEqual({
      kind: "override",
      token: "generic_mio_trait_wide_tracks",
      raw_extras: {},
      position: { x: 0, y: 0 },
      equipment_bonus: { reliability: 0.1 },
    });
    const stored = storedTrait(
      { ...start.trait, position: { x: 0, y: 2 } },
      start,
    );
    expect(stored).toEqual({
      kind: "override",
      token: "generic_mio_trait_wide_tracks",
      raw_extras: { position: "x = 0\ny = 2" },
    });
  });

  it("counts an inherited cell from its anchor, and leaves it out without one", () => {
    const inherited = shown({
      token: "generic_mio_trait_sloped_armor",
      values: {},
    });
    expect(
      traitStart(undefined, inherited.token, inherited, ANCHOR).trait.position,
    ).toEqual({ x: 1, y: 1 });
    expect(
      traitStart(undefined, inherited.token, inherited, null).trait,
    ).not.toHaveProperty("position");
  });

  it("opens from a trait in the details window in edit mode", () => {
    const layer = readFileSync(
      join(__dirname, "IndustrialOrgDetailLayer.tsx"),
      "utf8",
    );
    expect(layer).toContain(
      'label={tLayer("editTrait", { name: item.label })}',
    );
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain(
      "editMode && !countryPanelOpen && panel === MIO_DETAIL_WINDOW && tag && mioDetail && mioTraitPick && !selection",
    );
    expect(page).toContain("onEditTrait={onEditMioTrait}");
  });
});
