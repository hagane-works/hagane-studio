import { describe, expect, it } from "vitest";
import { SHORTCUT_GRID, shortcutSlotTop } from "./FocusShortcuts";
import { jumpView } from "./FocusTreeLayer";

describe("the focus view's jump buttons", () => {
  it("stack up from the grid's lower left, one slot each", () => {
    expect(SHORTCUT_GRID).toMatchObject({
      left: 1,
      bottom: 1024,
      slot: 68,
      max: 8,
    });
    expect(shortcutSlotTop(0)).toBe(956);
    expect(shortcutSlotTop(1)).toBe(888);
  });

  it("put the target focus in the middle of the frame at the shortcut's zoom", () => {
    const frame = { width: 1860, height: 850 };
    const treeW = 5000;
    const target = { x: 3000, y: 1200 };
    for (const zoom of [1, 0.485]) {
      const view = jumpView(frame, treeW, target, zoom);
      expect(view.zoom).toBe(zoom);

      const left = (frame.width - treeW * zoom) / 2 + view.x;
      expect(
        Math.abs(left + target.x * zoom - frame.width / 2),
      ).toBeLessThanOrEqual(1);
      expect(
        Math.abs(view.y + target.y * zoom - frame.height / 2),
      ).toBeLessThanOrEqual(1);
    }
  });
});
