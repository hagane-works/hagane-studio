import { describe, expect, it } from "vitest";
import { frameBetween, moveFrame } from "./WorldMapCanvas";

const MAP: [number, number] = [5632, 2048];

describe("the new land's frame", () => {
  it("is drawn from the press to the pointer, either way round", () => {
    expect(frameBetween([100, 200], [160, 230], null, MAP)).toEqual([
      100, 200, 60, 30,
    ]);
    expect(frameBetween([160, 230], [100, 200], null, MAP)).toEqual([
      100, 200, 60, 30,
    ]);
  });

  it("keeps the sketch's shape", () => {
    expect(frameBetween([100, 200], [160, 210], 2, MAP)).toEqual([
      100, 200, 60, 30,
    ]);
    expect(frameBetween([100, 200], [110, 240], 2, MAP)).toEqual([
      100, 200, 80, 40,
    ]);
  });

  it("stays inside the map and is never a speck", () => {
    expect(frameBetween([5620, 2040], [5700, 2100], null, MAP)).toEqual([
      5552, 1988, 80, 60,
    ]);
    expect(frameBetween([10, 10], [11, 11], null, MAP)).toEqual([10, 10, 8, 8]);
  });

  it("moves inside the map", () => {
    expect(moveFrame([100, 200, 60, 30], 15.4, -10.6, MAP)).toEqual([
      115, 189, 60, 30,
    ]);
    expect(moveFrame([100, 200, 60, 30], -500, 5000, MAP)).toEqual([
      0, 2018, 60, 30,
    ]);
  });
});
