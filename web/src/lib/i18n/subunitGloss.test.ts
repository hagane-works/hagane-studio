import { describe, expect, it } from "vitest";
import { subunitOptionLabel } from "./subunitGloss";

const NAMES = {
  infantry: { en: "Infantry", zh: "步兵", ru: "Пехота" },
  light_armor: { en: "Light Tank", zh: "轻型坦克" },
};

describe("subunitGloss", () => {
  it("glosses tokens with the game's name in the UI language", () => {
    expect(subunitOptionLabel("infantry", "zh", NAMES)).toBe("infantry · 步兵");
    expect(subunitOptionLabel("light_armor", "zh", NAMES)).toBe(
      "light_armor · 轻型坦克",
    );
    expect(subunitOptionLabel("infantry", "ru", NAMES)).toBe(
      "infantry · Пехота",
    );
  });

  it("returns tokens the game does not name unchanged", () => {
    expect(subunitOptionLabel("tno_super_infantry", "zh", NAMES)).toBe(
      "tno_super_infantry",
    );
    expect(subunitOptionLabel("", "zh", NAMES)).toBe("");

    expect(subunitOptionLabel("light_armor", "ko", NAMES)).toBe("light_armor");
    expect(subunitOptionLabel("infantry", "zh", {})).toBe("infantry");
  });

  it("returns the bare token in the en locale", () => {
    expect(subunitOptionLabel("infantry", "en", NAMES)).toBe("infantry");
    expect(subunitOptionLabel("light_armor", "en", NAMES)).toBe("light_armor");
  });
});
