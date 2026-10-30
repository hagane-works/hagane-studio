import type { DivisionRegiment } from "@/types";

export const GRID_COLS = 5;
export const GRID_ROWS = 5;
export const SUPPORT_SLOTS = 5;

export interface RegimentGrid {
  cells: (number | null)[][];

  overflow: number[];
}

export function buildRegimentGrid(regiments: DivisionRegiment[]): RegimentGrid {
  const cells: (number | null)[][] = Array.from({ length: GRID_COLS }, () =>
    Array.from({ length: GRID_ROWS }, () => null),
  );
  const overflow: number[] = [];
  regiments.forEach((regiment, index) => {
    const { x, y } = regiment;
    const inBounds =
      Number.isInteger(x) &&
      Number.isInteger(y) &&
      x >= 0 &&
      x < GRID_COLS &&
      y >= 0 &&
      y < GRID_ROWS;
    if (!inBounds || cells[x][y] !== null) {
      overflow.push(index);
      return;
    }
    cells[x][y] = index;
  });
  return { cells, overflow };
}

export interface SupportColumn {
  slots: (number | null)[];
  overflow: number[];
}

export function buildSupportColumn(support: DivisionRegiment[]): SupportColumn {
  const slots: (number | null)[] = Array.from(
    { length: SUPPORT_SLOTS },
    () => null,
  );
  const overflow: number[] = [];
  support.forEach((_, index) => {
    if (index < SUPPORT_SLOTS) slots[index] = index;
    else overflow.push(index);
  });
  return { slots, overflow };
}

export function firstFreeCell(
  grid: RegimentGrid,
): { x: number; y: number } | null {
  for (let x = 0; x < GRID_COLS; x++) {
    for (let y = 0; y < GRID_ROWS; y++) {
      if (grid.cells[x][y] === null) return { x, y };
    }
  }
  return null;
}
