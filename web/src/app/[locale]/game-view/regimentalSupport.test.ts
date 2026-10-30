import { describe, expect, it } from "vitest";
import {
  designerSavePayload,
  draftOf,
  encodeCells,
  pickerEntries,
  placed,
  removed,
  removedRegiment,
  regimentalSupportSlots,
  sameDraft,
  saveCost,
  type DesignerDraft,
  type DesignerUnit,
} from "./designerEdit";

const units: DesignerUnit[] = [
  { type: "infantry", group: "infantry" },
  { type: "field_guns", group: "support", regimental: true, divisional: false },
  {
    type: "fire_support",
    group: "support",
    regimental: true,
    divisional: false,
  },
  { type: "engineer", group: "support", regimental: false, divisional: true },
];
const costs = { group: 20, change: 5, support: 10 };
const column = (x: number, n = 3) =>
  Array.from({ length: n }, (_, y) => ({ type: "infantry", x, y }));
const initial: DesignerDraft = {
  regiments: [...column(0), ...column(1)],
  support: [],
};
const target = { grid: "regimental" as const, x: 0, y: 0, group: null };
const add = (draft = initial, type = "field_guns", x = 0) =>
  placed(draft, { ...target, x }, type, units);

describe("regimental support end-to-end draft contract", () => {
  it("opens the proper picker, without offering line or division-only units", () => {
    expect(
      pickerEntries(initial, target, units, costs).map(
        (entry) => entry.kind === "unit" && entry.unit.type,
      ),
    ).toEqual(["field_guns", "fire_support"]);
    expect(
      pickerEntries(initial, { ...target, grid: "support" }, units, costs).map(
        (entry) => entry.kind === "unit" && entry.unit.type,
      ),
    ).toEqual(["engineer"]);
  });

  it("adds under its own regiment, leaving both other grids untouched", () => {
    const next = add();
    expect(next.regiments).toEqual(initial.regiments);
    expect(next.support).toEqual([]);
    expect(next.regimentalSupport).toEqual([
      { type: "field_guns", x: 0, y: 0 },
    ]);
    expect(regimentalSupportSlots(next)[0]).toMatchObject({
      kind: "unit",
      cell: { type: "field_guns" },
    });
    expect(sameDraft(initial, next)).toBe(false);
  });

  it("replaces and removes the company, not the first line battalion", () => {
    const next = add(add(), "fire_support");
    expect(next.regimentalSupport).toEqual([
      { type: "fire_support", x: 0, y: 0 },
    ]);
    const entries = pickerEntries(next, target, units, costs, initial);
    expect(entries[0]).toMatchObject({ kind: "remove", cost: -10 });
    expect(entries[2]).toMatchObject({ kind: "unit", line: "selected" });
    const empty = removed(next, target);
    expect(empty.regimentalSupport).toEqual([]);
    expect(empty.regiments).toEqual(initial.regiments);
    expect(sameDraft(initial, empty)).toBe(true);
  });

  it("permits the same company in different regiments", () => {
    expect(add(add(), "field_guns", 1).regimentalSupport).toEqual([
      { type: "field_guns", x: 0, y: 0 },
      { type: "field_guns", x: 1, y: 0 },
    ]);
  });

  it("does not open a locked slot or accept an ineligible company", () => {
    const small: DesignerDraft = { regiments: column(0, 2), support: [] };
    expect(regimentalSupportSlots(small)[0].kind).toBe("locked");
    expect(pickerEntries(small, target, units, costs)).toEqual([]);
    expect(add(small)).toBe(small);
    expect(add(initial, "engineer")).toBe(initial);
    expect(add(initial, "infantry")).toBe(initial);
    expect(add(initial, "field_guns", 5)).toBe(initial);
  });

  it("removes support when its regiment drops below three line battalions", () => {
    const next = removed(add(), { grid: "regiment", x: 0, y: 1 });
    expect(next.regimentalSupport).toEqual([]);
    expect(regimentalSupportSlots(next)[0].kind).toBe("locked");
  });

  it("moves support with its owning regiment when the preceding regiment is removed", () => {
    const next = removedRegiment(add(initial, "field_guns", 1), 0);
    expect(next.regiments).toEqual(column(0));
    expect(next.regimentalSupport).toEqual([
      { type: "field_guns", x: 0, y: 0 },
    ]);
  });

  it("uses a separate preview prefix and save field, and reads it back", () => {
    const next = add();
    expect(encodeCells(next)).toContain("rs:0:0:field_guns");
    const payload = designerSavePayload(next, {
      tag: "GER",
      template: "Infantry",
      name: "Infantry",
    });
    expect(payload.regimental_support).toEqual(next.regimentalSupport);
    expect(payload.regiments).toEqual(initial.regiments);
    expect(payload.support).toEqual([]);
    const reopened = draftOf(payload);
    expect(reopened.regimentalSupport).toEqual(next.regimentalSupport);
    expect(sameDraft(reopened, next)).toBe(true);
    expect(
      designerSavePayload(removed(next, target), {
        tag: "GER",
        template: "Infantry",
        name: "Infantry",
      }).regimental_support,
    ).toEqual([]);
  });

  it("includes additions, replacements and removals in the experience cost", () => {
    const original = add();
    expect(saveCost(initial, original, units, costs)).toBe(10);
    expect(saveCost(original, original, units, costs)).toBe(0);
    expect(
      saveCost(original, add(original, "fire_support"), units, costs),
    ).toBe(20);
    expect(saveCost(original, removed(original, target), units, costs)).toBe(
      10,
    );
  });
});
