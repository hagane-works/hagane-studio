import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const source = readFileSync(join(__dirname, "ProductionLayer.tsx"), "utf8");
const navigation = readFileSync(join(__dirname, "gameNavigation.ts"), "utf8");

describe("production layer", () => {
  it("stacks the game's own windows instead of laying out rows by hand", () => {
    expect(source).toContain(
      'const PANEL = "production_equipment_window_military"',
    );
    expect(source).toContain('const ROW = "military_equipment_info_entry"');
    expect(source).toContain(
      'const NEW_LINE = "production_military_new_line_entry"',
    );
    expect(source).toContain(
      'const ROW_RESOURCE = "production_type_resource_item"',
    );

    expect(source).toContain('const TOP_RESOURCE = "resource_strip_item"');
  });

  it("slides the catalogue out to the position the window itself names", () => {
    expect(source).toContain("raw?.show_position?.x");

    expect(source).toContain("left: showAt.x, top: showAt.y");
  });

  it("uses the grid's own slot height rather than the row window's height", () => {
    expect(source).toContain("const SLOT_H = 80");
    expect(source).toContain("top={at * SLOT_H}");
  });

  it("clips and scrolls the list the way the panel declares it", () => {
    expect(source).toContain('overflowY: "auto"');
    expect(source).toContain("items.length * SLOT_H");
    expect(source).toContain("1080 - showAt.y - grid.y");
  });

  it("blanks the placeholders the file ships before painting real values", () => {
    expect(source).toContain('["name", "general_name", "production_cost"]');
  });

  it("keeps the six category buttons as the screen's own filter", () => {
    for (const button of [
      "show_equipment",
      "show_armor",
      "show_air",
      "show_capital_ships",
      "show_screen_ships",
      "show_other_ships",
    ]) {
      expect(source).toContain(`button: "${button}"`);
    }
  });

  it("takes the equipment icon from the Go side rather than guessing a name", () => {
    expect(source).toContain("item.icon ? icons.data?.[item.icon]");
    expect(source).not.toContain('"GFX_" +');
  });

  it("writes the header's readouts and hides only the running game's counters", () => {
    const page = readFileSync(join(__dirname, "page.tsx"), "utf8");
    expect(page).toContain("productionTexts(win, productionData.data?.queue)");
    expect(navigation).not.toContain('"operational_factories_value"');
    expect(navigation).not.toContain('"docyards_output_value"');
    expect(navigation).not.toContain('"factories_used_equipment"');
    expect(navigation).toContain('"factories_used_stockpiling"');
  });
});
