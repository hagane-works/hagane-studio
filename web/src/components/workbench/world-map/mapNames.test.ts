import { describe, expect, it } from "vitest";
import { nameChanges, nameKey, nameLanguages } from "./mapNames";

describe("renaming places on the world map", () => {
  it("names the game's localisation keys", () => {
    expect(nameKey("state", 64)).toBe("STATE_64");
    expect(nameKey("city", 6521)).toBe("VICTORY_POINTS_6521");
    expect(nameKey("region", 8)).toBe("STRATEGICREGION_8");
  });

  it("asks for English first, then the project's other export languages once", () => {
    expect(nameLanguages(undefined)).toEqual(["en"]);
    expect(nameLanguages(["zh", "EN", "zh", " de "])).toEqual([
      "en",
      "zh",
      "de",
    ]);
  });

  it("writes only what changed, and an emptied name goes back to the game's", () => {
    expect(
      nameChanges(
        { en: "Neu-Berlin ", zh: "", de: "Berlin" },
        { en: undefined, zh: "柏林", de: "Berlin" },
      ),
    ).toEqual({
      en: "Neu-Berlin",
      zh: null,
    });
    expect(nameChanges({ en: "" }, {})).toEqual({});
  });
});
