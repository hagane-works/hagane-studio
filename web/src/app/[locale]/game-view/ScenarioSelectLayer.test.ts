import { describe, expect, it } from "vitest";
import { scenarioRowLayout } from "./ScenarioSelectLayer";

const ROOM = 579 - 2 * 43;

describe("scenarioRowLayout", () => {
  it("draws the base game's two scenarios at full size", () => {
    expect(scenarioRowLayout(2, ROOM)).toEqual({ scale: 1, scroll: false });
  });

  it("shrinks a mod's three scenarios so the third stays inside the window", () => {
    const row = scenarioRowLayout(3, ROOM);
    expect(row.scroll).toBe(false);
    expect(row.scale).toBeLessThan(1);

    expect((2 * 260 + 232) * row.scale).toBeLessThanOrEqual(ROOM);
  });

  it("scrolls rather than shrinking tiles past legibility", () => {
    expect(scenarioRowLayout(8, ROOM)).toEqual({ scale: 0.55, scroll: true });
  });
});
