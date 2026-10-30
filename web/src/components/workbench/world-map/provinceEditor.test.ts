import { describe, expect, it } from "vitest";
import { draftOf, provinceChanges } from "./ProvinceEditor";
import type { Province } from "./worldMapModel";

const brandenburg: Province = {
  id: 375,
  rgb: 0,
  kind: "land",
  coastal: false,
  terrain: "plains",
  continent: 1,
  state: 64,
  region: 8,
};

describe("a province edit", () => {
  it("sends nothing for an unchanged draft", () => {
    expect(provinceChanges(brandenburg, draftOf(brandenburg))).toEqual({});
  });

  it("sends only the fields that changed", () => {
    const draft = { ...draftOf(brandenburg), terrain: "forest", state: 65 };
    expect(provinceChanges(brandenburg, draft)).toEqual({
      terrain: "forest",
      state: 65,
    });
  });

  it("never sends an empty state or region", () => {
    const draft = { ...draftOf(brandenburg), state: 0, region: 0 };
    expect(provinceChanges(brandenburg, draft)).toEqual({});
  });
});
