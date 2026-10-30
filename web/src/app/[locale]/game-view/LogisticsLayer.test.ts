import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { usageFraction } from "./LogisticsLayer";

const source = readFileSync(join(__dirname, "LogisticsLayer.tsx"), "utf8");

describe("logistics layer", () => {
  it("uses the game's own row window for each arm", () => {
    expect(source).toContain(
      'const LAND_ROW = "logistics_overview_land_equipment_entry"',
    );
    expect(source).toContain(
      'const NAVAL_ROW = "logistics_overview_naval_equipment_entry"',
    );
    expect(source).toContain(
      'const AIR_ROW = "logistics_overview_air_equipment_entry"',
    );
    expect(source).toContain("capital_ships: NAVAL_ROW");
  });

  it("blanks every placeholder the file ships", () => {
    expect(source).toContain(
      '["equipment_type", "needs", "produced", "balance", "in_stock"]',
    );
    expect(source).toContain('["value"]');
  });

  it("scales a picture into the box its slot marks out", () => {
    expect(source).toContain("const ICON_BOX = { w: 147, h: 69 }");
    expect(source).toContain(
      "Math.min(1, ICON_BOX.w / icon.width, ICON_BOX.h / icon.height)",
    );
  });

  it("fills only the column a scenario's files can answer", () => {
    expect(source).toContain("String(item.needs)");
    expect(source).not.toContain('rects["name:produced"]');
    expect(source).not.toContain('rects["name:in_stock"]');
  });
});

describe("the logistics screen's bottom (hoi4-game -process 4:16)", () => {
  it("fills a usage bar by factories on lines over factories owned", () => {
    expect(usageFraction(20, 28)).toBeCloseTo(20 / 28);
    expect(usageFraction(13, 10)).toBe(1);
    expect(usageFraction(0, 0)).toBe(0);
    expect(usageFraction(undefined, 10)).toBe(0);
  });

  it("is laid out with its sizes pinned, so its bottom boxes sit on the screen's edge", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toMatch(/PINNED_SIZES = new Set\(\[[^\]]*LOGISTICS[^\]]*\]\)/);
    expect(page).toMatch(
      /PINNED_NEGATIVE = new Set\(\[[^\]]*LOGISTICS[^\]]*\]\)/,
    );
  });
});
