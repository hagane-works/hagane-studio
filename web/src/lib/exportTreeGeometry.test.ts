import { describe, expect, it } from "vitest";
import { computeExportGeometry } from "./exportTreePng";

describe("computeExportGeometry", () => {
  it("a normal tree exports 1:1 — no downscale at all", () => {
    const g = computeExportGeometry({ width: 2400, height: 1400 });
    expect(g.scale).toBe(1);
    expect(g.width).toBe(2400 + 240);
    expect(g.height).toBe(1400 + 240);
  });

  it("REGRESSION · a wide tree is no longer squeezed to 0.1 zoom", () => {
    const g = computeExportGeometry({ width: 9000, height: 1400 });
    expect(g.scale).toBeGreaterThan(0.5);
    expect(g.width).toBeLessThanOrEqual(12000);
  });

  it("tiny trees are padded up to the minimum, not left postage-stamp sized", () => {
    const g = computeExportGeometry({ width: 100, height: 80 });
    expect(g.width).toBe(800);
    expect(g.height).toBe(800);
    expect(g.scale).toBe(1);
  });

  it("never exceeds either the dimension cap or the area cap", () => {
    for (const b of [
      { width: 50000, height: 400 },
      { width: 400, height: 50000 },
      { width: 30000, height: 30000 },
      { width: 12000, height: 9000 },
    ]) {
      const g = computeExportGeometry(b);
      expect(g.width).toBeLessThanOrEqual(12000);
      expect(g.height).toBeLessThanOrEqual(12000);
      expect(g.width * g.height).toBeLessThanOrEqual(40_000_000 + 1);
      expect(g.scale).toBeGreaterThan(0);
      expect(g.scale).toBeLessThanOrEqual(1);
    }
  });

  it("spends leftover headroom on pixelRatio 2, and backs off when full", () => {
    const small = computeExportGeometry({ width: 1200, height: 800 });
    expect(small.pixelRatio).toBe(2);

    const huge = computeExportGeometry({ width: 30000, height: 30000 });
    expect(huge.pixelRatio).toBe(1);

    expect(huge.width * huge.height * huge.pixelRatio ** 2).toBeLessThanOrEqual(
      40_000_000 + 1,
    );
  });

  it("aspect ratio is preserved when it has to downscale", () => {
    const b = { width: 24000, height: 12000 };
    const g = computeExportGeometry(b);
    const naturalRatio = (b.width + 240) / (b.height + 240);
    expect(g.width / g.height).toBeCloseTo(naturalRatio, 2);
  });

  it("degenerate bounds do not produce a zero or NaN canvas", () => {
    for (const b of [
      { width: 0, height: 0 },
      { width: 0, height: 500 },
    ]) {
      const g = computeExportGeometry(b);
      expect(Number.isFinite(g.width)).toBe(true);
      expect(Number.isFinite(g.height)).toBe(true);
      expect(g.width).toBeGreaterThan(0);
      expect(g.height).toBeGreaterThan(0);
    }
  });
});
