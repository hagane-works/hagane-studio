import { describe, expect, it } from "vitest";

import {
  DEFAULT_SUPER_EVENT_AI_CHANCE,
  parseSuperEventAiChance,
} from "./superEventAiChance";

describe("SuperEvent ai_chance precision", () => {
  it("preserves precision beyond two decimal places through JSON save and re-entry", () => {
    const input = "0.123456789";
    const saved = JSON.stringify({ ai_chance: parseSuperEventAiChance(input) });
    const reentered = JSON.parse(saved) as { ai_chance: number };

    expect(reentered.ai_chance).toBe(0.123456789);
    expect(saved).toContain("0.123456789");
  });

  it("keeps zero intentional and maps an empty or non-finite edit to the default", () => {
    expect(parseSuperEventAiChance("0")).toBe(0);
    expect(parseSuperEventAiChance("")).toBe(DEFAULT_SUPER_EVENT_AI_CHANCE);
    expect(parseSuperEventAiChance("Infinity")).toBe(
      DEFAULT_SUPER_EVENT_AI_CHANCE,
    );
  });
});
