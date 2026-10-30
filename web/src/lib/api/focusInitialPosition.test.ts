import { describe, expect, it } from "vitest";
import {
  buildInitialShowPositionUpdate,
  isAmbiguousLegacyInitialShowPosition,
} from "./focus-tech";

describe("focus initial_show_position frontend contract", () => {
  it("does not guess numeric legacy strings", () => {
    expect(isAmbiguousLegacyInitialShowPosition("93")).toBe(true);
    expect(isAmbiguousLegacyInitialShowPosition("-12")).toBe(true);
    expect(
      isAmbiguousLegacyInitialShowPosition("BEL_perpetual_neutrality"),
    ).toBe(false);
  });

  it("writes the tagged union and retires the legacy field", () => {
    expect(
      buildInitialShowPositionUpdate({
        kind: "coordinates",
        x: 40,
        y: 53,
        offsets: [{ x: 2, y: -1, trigger: 'has_dlc = "Arms Against Tyranny"' }],
      }),
    ).toEqual({
      initial_show_position: {
        kind: "coordinates",
        x: 40,
        y: 53,
        offsets: [{ x: 2, y: -1, trigger: 'has_dlc = "Arms Against Tyranny"' }],
      },
      initial_show_focus: "",
    });
  });
});
