import { describe, expect, it } from "vitest";

import {
  GRID_COLS,
  GRID_ROWS,
  SUPPORT_SLOTS,
  buildRegimentGrid,
  buildSupportColumn,
  firstFreeCell,
} from "./divisionGridModel";

const reg = (token: string, x: number, y: number) => ({ token, x, y });

describe("division designer grid model", () => {
  it("places regiments by their native x/y coordinates", () => {
    const grid = buildRegimentGrid([
      reg("infantry", 0, 0),
      reg("artillery_brigade", 1, 2),
    ]);
    expect(grid.cells[0][0]).toBe(0);
    expect(grid.cells[1][2]).toBe(1);
    expect(grid.overflow).toEqual([]);
  });

  it("overflows out-of-bounds and colliding coordinates instead of dropping them", () => {
    const grid = buildRegimentGrid([
      reg("infantry", 0, 0),
      reg("infantry", 0, 0),
      reg("infantry", GRID_COLS, 0),
      reg("infantry", 0, GRID_ROWS),
      reg("infantry", -1, 0),
    ]);
    expect(grid.cells[0][0]).toBe(0);
    expect(grid.overflow).toEqual([1, 2, 3, 4]);
  });

  it("fills a column before opening the next one", () => {
    const grid = buildRegimentGrid([
      reg("infantry", 0, 0),
      reg("infantry", 0, 1),
    ]);
    expect(firstFreeCell(grid)).toEqual({ x: 0, y: 2 });
    const fullColumn = buildRegimentGrid(
      Array.from({ length: GRID_ROWS }, (_, y) => reg("infantry", 0, y)),
    );
    expect(firstFreeCell(fullColumn)).toEqual({ x: 1, y: 0 });
  });

  it("reports a full grid as having no free cell", () => {
    const full = buildRegimentGrid(
      Array.from({ length: GRID_COLS * GRID_ROWS }, (_, n) =>
        reg("infantry", Math.floor(n / GRID_ROWS), n % GRID_ROWS),
      ),
    );
    expect(firstFreeCell(full)).toBeNull();
    expect(full.overflow).toEqual([]);
  });

  it("support fills slots in array order and overflows past the fifth", () => {
    const column = buildSupportColumn(
      Array.from({ length: SUPPORT_SLOTS + 2 }, (_, n) =>
        reg("engineer", 0, n),
      ),
    );
    expect(column.slots).toEqual([0, 1, 2, 3, 4]);
    expect(column.overflow).toEqual([5, 6]);
  });
});
