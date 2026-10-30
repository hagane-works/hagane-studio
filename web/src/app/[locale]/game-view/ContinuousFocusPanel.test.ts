import { describe, expect, it } from "vitest";
import { CONTINUOUS_LIST, continuousSlot } from "./ContinuousFocusPanel";
import { treePoint } from "./FocusTreeLayer";

describe("the continuous focus panel", () => {
  it("lays its focuses out four across, as the file's list does", () => {
    expect(CONTINUOUS_LIST).toMatchObject({
      left: 29,
      top: 113,
      slotW: 180,
      slotH: 120,
      across: 4,
    });
    expect(continuousSlot(0)).toEqual({ left: 29, top: 113 });
    expect(continuousSlot(3)).toEqual({ left: 569, top: 113 });
    expect(continuousSlot(4)).toEqual({ left: 29, top: 233 });
  });

  it("sits where the tree puts it, measured from the engine's grid corner", () => {
    expect(
      treePoint({ x: 500, y: 1300 }, { x: 0, y: 0 }, { x: 96, y: 130 }),
    ).toEqual({ left: 450, top: 1250 });
    expect(
      treePoint({ x: 500, y: 1300 }, { x: -2, y: 0 }, { x: 96, y: 130 }),
    ).toEqual({ left: 642, top: 1250 });
  });
});
