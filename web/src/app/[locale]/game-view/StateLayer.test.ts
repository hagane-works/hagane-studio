import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const source = readFileSync(join(__dirname, "StateLayer.tsx"), "utf8");
const navigation = readFileSync(join(__dirname, "gameNavigation.ts"), "utf8");

describe("state layer", () => {
  it("stacks the game's own entry into each of the three building grids", () => {
    expect(source).toContain('const STATE_ENTRY = "state_building_entry"');
    expect(source).toContain(
      'const SHARED_ENTRY = "state_shared_slot_building_entry"',
    );
    expect(source).toContain(
      'const PROVINCE_ENTRY = "province_building_entry"',
    );
  });

  it("keeps each grid's own slot size", () => {
    expect(source).toContain("const STATE_SLOT = { w: 60, h: 52 }");
    expect(source).toContain("const SHARED_SLOT = { w: 61, h: 50 }");
    expect(source).toContain("const PROVINCE_SLOT = { w: 61, h: 84 }");
  });

  it("picks every icon by the frame the game's own files give", () => {
    expect(source).toContain('const STRIP = "GFX_buildings_strip"');
    expect(source).toContain('const RESOURCE_STRIP = "GFX_resources_strip"');
    expect(source).toContain("item.icon_frame ?? 1");
  });

  it("blanks the placeholders the screen ships", () => {
    expect(navigation).toContain('"template_need_text"');
    expect(navigation).toContain('"state_name"');
  });
});
