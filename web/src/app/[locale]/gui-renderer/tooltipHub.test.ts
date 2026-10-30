import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  createTooltipHub,
  type TooltipElement,
  type TooltipHit,
} from "./tooltipHub";
import type { GameTooltipContent } from "./tooltipText";

class Fake implements TooltipElement {
  isConnected = true;
  constructor(
    readonly order: number,
    readonly parent: Fake | null,
    readonly box = { left: 0, top: 0, width: 1920, height: 1080 },
    readonly offsetWidth = 1920,
    readonly offsetHeight = 1080,
  ) {}
  getBoundingClientRect() {
    return this.box;
  }
  contains(other: TooltipElement | null): boolean {
    for (let at = other as Fake | null; at; at = at.parent)
      if (at === this) return true;
    return false;
  }
  compareDocumentPosition(other: TooltipElement): number {
    const o = other as Fake;
    if (o === this) return 0;
    if (this.contains(o)) return 4 | 16;
    if (o.contains(this)) return 2 | 8;
    return o.order > this.order ? 4 : 2;
  }
}

const tip = (text: string, delayed?: string): GameTooltipContent => ({
  text,
  delayed,
});

function source(
  id: string,
  el: Fake,
  hit: (x: number, y: number) => TooltipHit | null,
) {
  return { id, element: () => el, hitTest: hit };
}

beforeEach(() => {
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
});

describe("timing (TOOLTIP_SHOW/HIDE/DELAYED_DELAY)", () => {
  it("shows after 50 ms, keeps it while the pointer stays, hides 50 ms after it leaves", () => {
    const hub = createTooltipHub();
    const stage = new Fake(0, null);
    const canvas = new Fake(1, stage);
    hub.registerHost({ id: "host", explicit: true, element: () => stage });
    let over = true;
    hub.registerSource(
      source("c", canvas, () =>
        over
          ? {
              key: "n",
              content: tip("Navy"),
              rect: { x: 0, y: 0, w: 10, h: 10 },
            }
          : null,
      ),
    );
    hub.pointerMove(5, 5, stage);
    expect(hub.getSnapshot().shown).toBeNull();
    vi.advanceTimersByTime(49);
    expect(hub.getSnapshot().shown).toBeNull();
    vi.advanceTimersByTime(1);
    expect(hub.getSnapshot().shown?.content.text).toBe("Navy");

    hub.pointerMove(8, 9, stage);
    expect(hub.getSnapshot().shown?.cursor).toEqual({ x: 8, y: 9 });
    over = false;
    hub.pointerMove(500, 500, stage);
    vi.advanceTimersByTime(49);
    expect(hub.getSnapshot().shown).not.toBeNull();
    vi.advanceTimersByTime(1);
    expect(hub.getSnapshot().shown).toBeNull();
  });

  it("adds the delayed text 1000 ms after the pointer arrived", () => {
    const hub = createTooltipHub();
    const stage = new Fake(0, null);
    hub.registerHost({ id: "host", explicit: true, element: () => stage });
    hub.registerSource(
      source("c", new Fake(1, stage), () => ({
        key: "a",
        content: tip("§HNo Active Goal§!", "Pick one."),
        rect: { x: 0, y: 0, w: 1, h: 1 },
      })),
    );
    hub.pointerMove(1, 1, stage);
    vi.advanceTimersByTime(50);
    expect(hub.getSnapshot().shown?.delayed).toBe(false);
    vi.advanceTimersByTime(949);
    expect(hub.getSnapshot().shown?.delayed).toBe(false);
    vi.advanceTimersByTime(1);
    expect(hub.getSnapshot().shown?.delayed).toBe(true);
  });

  it("takes a tooltip down at once when its canvas goes away", () => {
    const hub = createTooltipHub();
    const stage = new Fake(0, null);
    hub.registerHost({ id: "host", explicit: true, element: () => stage });
    const off = hub.registerSource(
      source("c", new Fake(1, stage), () => ({
        key: "a",
        content: tip("x"),
        rect: { x: 0, y: 0, w: 1, h: 1 },
      })),
    );
    hub.pointerMove(1, 1, stage);
    vi.advanceTimersByTime(50);
    expect(hub.getSnapshot().shown).not.toBeNull();
    off();
    expect(hub.getSnapshot().shown).toBeNull();
  });
});

describe("which canvas answers", () => {
  const stage = new Fake(0, null);
  const base = new Fake(1, stage);
  const layerControl = new Fake(2, stage);
  const topbar = new Fake(3, stage);
  const topbarButton = new Fake(4, topbar);
  const hub = createTooltipHub();
  hub.registerSource(
    source("base", base, (x, y) => ({
      key: `b${x},${y}`,
      content: tip("base"),
      rect: { x, y, w: 1, h: 1 },
    })),
  );
  hub.registerSource(
    source("topbar", topbar, (x) =>
      x < 100
        ? {
            key: "t",
            content: tip("topbar"),
            rect: { x: 0, y: 0, w: 100, h: 50 },
          }
        : null,
    ),
  );

  it("asks the canvas painted last first", () => {
    expect(hub.resolve(50, 20, stage)?.content.text).toBe("topbar");
  });

  it("falls through to the canvas below where the top one has no tooltip", () => {
    expect(hub.resolve(500, 500, stage)?.content.text).toBe("base");
  });

  it("stops at a React layer's control drawn over a canvas", () => {
    expect(hub.resolve(500, 500, layerControl)).toBeNull();
    expect(hub.resolve(50, 20, layerControl)?.content.text).toBe("topbar");
  });

  it("does not look through a canvas's own button", () => {
    expect(hub.resolve(500, 500, topbarButton)).toBeNull();
    expect(hub.resolve(50, 20, topbarButton)?.content.text).toBe("topbar");
  });
});

describe("scaled stage", () => {
  it("hit-tests in the canvas's own pixels and places in the host's", () => {
    const hub = createTooltipHub();

    const half = { left: 100, top: 50, width: 960, height: 540 };
    const stage = new Fake(0, null, half);
    const canvas = new Fake(1, stage, {
      left: 100,
      top: 50,
      width: 960,
      height: 540,
    });
    const seen: [number, number][] = [];
    hub.registerHost({ id: "host", explicit: true, element: () => stage });
    hub.registerSource(
      source("c", canvas, (x, y) => {
        seen.push([x, y]);
        return {
          key: "a",
          content: tip("x"),
          rect: { x: 564, y: 11, w: 40, h: 40 },
        };
      }),
    );
    hub.pointerMove(150, 60, stage);
    vi.advanceTimersByTime(50);
    expect(seen[0]).toEqual([100, 20]);
    const shown = hub.getSnapshot().shown;
    expect(shown?.cursor).toEqual({ x: 100, y: 20 });
    expect(shown?.anchorRect).toEqual({ x: 564, y: 11, w: 40, h: 40 });
    expect(shown?.bounds).toEqual({ w: 1920, h: 1080 });
  });
});

describe("hosts and layers", () => {
  it("draws in the page's host when there is one, else the biggest fallback", () => {
    const hub = createTooltipHub();
    const small = new Fake(
      5,
      null,
      { left: 0, top: 0, width: 300, height: 200 },
      300,
      200,
    );
    const stageSized = new Fake(6, null);
    hub.registerHost({ id: "small", explicit: false, element: () => small });
    hub.registerHost({
      id: "stage",
      explicit: false,
      element: () => stageSized,
    });
    expect(hub.getSnapshot().hostId).toBe("stage");
    const off = hub.registerHost({
      id: "page",
      explicit: true,
      element: () => stageSized,
    });
    expect(hub.getSnapshot().hostId).toBe("page");
    off();
    expect(hub.getSnapshot().hostId).toBe("stage");
  });

  it("lets a layer's own tooltip win until the pointer leaves it", () => {
    const hub = createTooltipHub();
    const stage = new Fake(0, null);
    hub.registerHost({ id: "host", explicit: true, element: () => stage });
    hub.registerSource(
      source("c", new Fake(1, stage), () => ({
        key: "a",
        content: tip("canvas"),
        rect: { x: 0, y: 0, w: 1, h: 1 },
      })),
    );
    hub.showExplicit("portrait", tip("Adolf Hitler"), 10, 10, null);
    hub.pointerMove(12, 12, stage);
    vi.advanceTimersByTime(50);
    expect(hub.getSnapshot().shown?.content.text).toBe("Adolf Hitler");
    hub.hideExplicit("portrait");
    hub.pointerMove(12, 12, stage);
    vi.advanceTimersByTime(50);
    expect(hub.getSnapshot().shown?.content.text).toBe("canvas");
  });
});
