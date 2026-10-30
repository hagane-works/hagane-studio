import { describe, expect, it } from "vitest";
import { terrainFrame, terrainSprite } from "./DesignerTerrain";

describe("the terrain panel", () => {
  it("picks the strip's frame the way its art is drawn: 1 better, 2 worse, 3 no change", () => {
    expect(terrainFrame(0.2)).toBe(1);
    expect(terrainFrame(-0.05)).toBe(2);
    expect(terrainFrame(0)).toBe(3);
  });

  it("draws each terrain with its own adjuster picture", () => {
    expect(terrainSprite("hills")).toBe("GFX_adjuster_hills_bg");
  });
});
