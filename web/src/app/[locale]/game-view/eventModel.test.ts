import { describe, expect, it } from "vitest";
import type {
  GuiBitmapFont,
  GuiNode,
  GuiSprite,
  GuiWindow,
} from "@/lib/guiRenderApi";
import { midsectionHeight, stretchEventWindow } from "./eventModel";

const font = {
  line: 16,
  base: 12,
  atlas: "",
  atlas_w: 0,
  atlas_h: 0,
  tint: "alpha",
  chars: { "32": [0, 0, 0, 0, 0, 0, 10], "97": [0, 0, 10, 16, 0, 0, 10] },
} as unknown as GuiBitmapFont;

const node = (partial: Partial<GuiNode>): GuiNode =>
  ({
    type: "containerwindowtype",
    name: null,
    position: { x: 0, y: 0 },
    size: {},
    children: [],
    ...partial,
  }) as GuiNode;

const eventWindow = (): GuiWindow =>
  ({
    name: "EventWindow",
    node: node({
      name: "EventWindow",
      size: { width: 581, height: 427 },
      children: [
        node({ name: "top_Window", size: { width: 581, height: 121 } }),
        node({
          name: "midsection",
          position: { x: 0, y: 121 },
          size: { width: 580, height: 100 },
        }),
        node({
          name: "bottom_Window",
          position: { x: 0, y: 221 },
          size: { width: 581, height: 206 },
          children: [
            node({
              type: "icontype",
              name: "event_picture",
              sprite: "GFX_report_event_001",
            }),
          ],
        }),
      ],
    }),
    sprites: {},
    fonts: {},
    texts: {},
  }) as unknown as GuiWindow;

describe("midsectionHeight", () => {
  it("keeps the declared 100 for a short description", () => {
    expect(midsectionHeight(font, "aaaa", 512)).toBe(100);
    expect(midsectionHeight(font, "", 512)).toBe(100);
    expect(midsectionHeight(undefined, "aaaa", 512)).toBe(100);
  });

  it("grows with the wrapped text, up to the box's maxHeight", () => {
    const words = Array.from({ length: 10 }, () => "aaaa").join(" ");
    expect(midsectionHeight(font, words, 90)).toBe(100);

    const lines = Array.from({ length: 20 }, () => "aaaa").join("\n");
    expect(midsectionHeight(font, lines, 512)).toBe(240);

    expect(
      midsectionHeight(
        font,
        Array.from({ length: 7 }, () => "a").join("\n"),
        512,
      ),
    ).toBe(122);
  });
});

describe("stretchEventWindow", () => {
  const sprite = {
    resolved: true,
    width: 210,
    height: 176,
    frames: 1,
  } as GuiSprite;

  it("sizes the midsection, drops the bottom window under it and grows the window", () => {
    const win = stretchEventWindow(eventWindow(), 180);
    const [top, mid, bottom] = win.node.children;
    expect(top.size).toEqual({ width: 581, height: 121 });
    expect(mid.size?.height).toBe(180);
    expect(bottom.position.y).toBe(121 + 180);
    expect(win.node.size?.height).toBe(121 + 180 + 206);
  });

  it("puts the event's own picture in the slot and ships its sprite", () => {
    const win = stretchEventWindow(eventWindow(), 100, {
      name: "GFX_report_event_airplane_crash",
      sprite,
    });
    const picture = win.node.children[2].children[0];
    expect(picture.sprite).toBe("GFX_report_event_airplane_crash");
    expect(win.sprites["GFX_report_event_airplane_crash"]).toBe(sprite);
  });

  it("leaves the shipped placeholder when the event has no picture it can name", () => {
    const win = stretchEventWindow(eventWindow(), 100, null);
    expect(win.node.children[2].children[0].sprite).toBe(
      "GFX_report_event_001",
    );
  });
});
