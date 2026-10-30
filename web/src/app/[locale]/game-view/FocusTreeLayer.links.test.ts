import { describe, expect, it } from "vitest";
import { focusLinkBars } from "./FocusTreeLayer";

const PLATE_BOTTOM = 120;
const ROW = 130;

describe("focus links", () => {
  it("show between neighbouring rows, from under the parent's plate into the child's icon", () => {
    const [down, into] = focusLinkBars(
      { cx: 100, cy: 0 },
      { cx: 100, cy: ROW },
    );

    expect(down.top).toBeLessThan(PLATE_BOTTOM);
    expect(into.top + into.height).toBeGreaterThan(ROW);
    expect(into.top + into.height - PLATE_BOTTOM).toBeGreaterThanOrEqual(20);
  });

  it("branch on a bar just below the parent, as the engine draws it", () => {
    const near = focusLinkBars({ cx: 0, cy: 0 }, { cx: 192, cy: ROW });
    expect(near).toHaveLength(3);
    expect(near[1]).toMatchObject({ left: 0, width: 192 });
    const far = focusLinkBars({ cx: 0, cy: 0 }, { cx: 192, cy: 3 * ROW });

    expect(far[1].top).toBe(near[1].top);
    expect(far[2].height).toBeGreaterThan(2 * ROW);
  });
});
