import { describe, expect, it } from "vitest";
import type { GuiNode, GuiWindow } from "@/lib/guiRenderApi";
import { resourceEntryLayout, textBox } from "./StateLayer";

const node = (
  name: string,
  x: number,
  y: number,
  extra: Partial<GuiNode> = {},
): GuiNode =>
  ({
    type: "instanttextboxtype",
    name,
    position: { x, y },
    size: null,
    maxwidth: null,
    format: null,
    font: null,
    children: [],
    ...extra,
  }) as GuiNode;
const PLATE = { width: 48 };

describe("the state screen writes each number in its own box", () => {
  it("centres a building's level over its plate, on both kinds of entry", () => {
    for (const entry of [
      {
        level: node("level", 5, 54, { maxwidth: 60, format: "center" }),
        plateX: 8,
      },

      {
        level: node("level", -16, 52, { maxwidth: 100, format: "center" }),
        plateX: 8,
      },
    ]) {
      const box = textBox(entry.level, { width: 60, align: "center" });
      expect(box.align).toBe("center");
      const middle = Number(entry.level.position.x) + box.width / 2;
      const plateMiddle = entry.plateX + PLATE.width / 2;
      expect(Math.abs(middle - plateMiddle)).toBeLessThanOrEqual(4);

      if (entry.level.position.x === -16)
        expect(Math.abs(-16 + 44 / 2 - plateMiddle)).toBeGreaterThan(20);
    }
  });

  it("puts a resource's icon and amount where state_resource_entry does, inside the resources panel", () => {
    const entry = {
      node: {
        children: [
          node("resource", 53, 11, { type: "icontype" }),
          node("amount", 70, 11, {
            maxwidth: 37,
            format: "center",
            font: "hoi_20b",
          }),
        ],
      },
      fonts: {},
    } as unknown as GuiWindow;
    const slot = resourceEntryLayout(entry);
    expect(slot.icon).toEqual({ x: 53, y: 11 });
    expect(slot.amount).toEqual({ x: 70, y: 11 });
    expect(slot).toMatchObject({ width: 37, align: "center" });

    const grid = -44;
    expect(grid + slot.icon.x).toBeGreaterThanOrEqual(0);
    expect(grid + slot.amount.x + slot.width).toBeLessThanOrEqual(135);

    expect(resourceEntryLayout(undefined)).toMatchObject({
      icon: { x: 53, y: 11 },
      amount: { x: 70, y: 11 },
      width: 37,
    });
  });
});
