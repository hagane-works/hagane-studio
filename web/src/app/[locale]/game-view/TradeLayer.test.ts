import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";
import { tradeCells } from "./TradeLayer";

const TEXT_COLORS = REFERENCE_TABLES.text_colors.global;
setReferenceTables(REFERENCE_TABLES);

const source = readFileSync(join(__dirname, "TradeLayer.tsx"), "utf8");

describe("trade layer", () => {
  it("stacks the game's own three windows into the three grids", () => {
    expect(source).toContain('const COLUMN = "resources_info_entry"');
    expect(source).toContain('const CHIP = "resources_filter_entry"');
    expect(source).toContain('const ROW = "country_trade_entry"');
  });

  it("places the filter row from its parent, because the name repeats", () => {
    expect(source).toContain('rects["name:filters"]');
    expect(source).toContain('rects["name:resources"]');
    expect(source).not.toContain('rects["name:resources_grid"]');
  });

  it("keeps the grids' own slot sizes", () => {
    expect(source).toContain("const COLUMN_W = 79");
    expect(source).toContain("const ROW_H = 45");
    expect(source).toContain("const ROW_W = 648");
  });

  it("prints the table's rows as the recording does (hoi4-game-process-en 3:28)", () => {
    const cells = tradeCells({
      resource: "steel",
      frame: 5,
      produced: 301,
      exported: 75,
      imported: 0,
      projects: 0,
      production: 49,
      surplus: 177,
    });
    expect(cells.balance).toEqual({ text: "177", color: TEXT_COLORS.G });
    expect(cells.produced).toEqual({ text: "301", color: TEXT_COLORS.G });
    expect(cells.exported).toEqual({ text: "-75", color: TEXT_COLORS.R });
    expect(cells.production_line).toEqual({
      text: "-49",
      color: TEXT_COLORS.R,
    });
    expect(cells.imported.text).toBe("0");
    expect(cells.needed.text).toBe("177");

    const rubber = tradeCells({
      resource: "rubber",
      frame: 3,
      produced: 0,
      exported: 0,
      production: 5,
      surplus: -5,
    });
    expect(rubber.balance).toEqual({ text: "-5", color: TEXT_COLORS.R });
    expect(rubber.produced.color).toBeUndefined();
  });

  it("hides the one runtime button the column window carries", () => {
    expect(source).toContain('RUNTIME_ONLY = new Set(["cancel_imports"])');
  });

  it("lets a resource chip filter and re-sort the country list", () => {
    expect(source).toContain("aria-label={`filter_${entry.resource}`}");
    expect(source).toContain("guiRecord(seller.available)[picked]");
  });
});
