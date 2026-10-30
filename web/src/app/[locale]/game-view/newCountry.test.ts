import { describe, expect, it } from "vitest";
import { matchStates } from "./NewCountryControl";

const STATES = [
  { id: 1, name: "Corsica", owner: "FRA" },
  { id: 64, name: "Brandenburg", owner: "GER" },
  { id: 640, name: "Kansai", owner: "JAP" },
  { id: 1082, name: "Atlantis", owner: "" },
];

describe("finding states to give a new country", () => {
  it("finds by name, by number (with or without #) and by owner", () => {
    expect(matchStates(STATES, "bran").map((s) => s.id)).toEqual([64]);
    expect(matchStates(STATES, "#64").map((s) => s.id)).toEqual([64]);
    expect(matchStates(STATES, "1082").map((s) => s.id)).toEqual([1082]);
    expect(matchStates(STATES, "ger").map((s) => s.id)).toEqual([64]);
  });

  it("finds nothing for an empty search and keeps to the limit", () => {
    expect(matchStates(STATES, "  ")).toEqual([]);
    expect(matchStates(STATES, "a", 2)).toHaveLength(2);
  });
});
