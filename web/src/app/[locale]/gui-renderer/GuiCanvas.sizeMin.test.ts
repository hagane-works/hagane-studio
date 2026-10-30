import { describe, expect, it } from "vitest";
import type { GuiNode } from "@/lib/guiRenderApi";
import { measureWindow, sizeMin } from "./GuiCanvas";

function node(partial: Partial<GuiNode>): GuiNode {
  return {
    type: "containerwindowtype",
    name: null,
    position: { x: 0, y: 0 },
    size: null,
    orientation: null,
    origo: null,
    scale: null,
    sprite: null,
    frame: null,
    text: null,
    text_source: null,
    font: null,
    format: null,
    maxwidth: null,
    maxheight: null,
    text_color: null,
    centerposition: false,
    alwaystransparent: false,
    background: null,
    children: [],
    ...partial,
  };
}

const stripes = () =>
  node({
    name: "techtree_stripes",
    size: { width: 1400, height: 1275 },
    point_tokens: {
      size: {
        __block__: [
          { k: "width", v: 1400 },
          { k: "height", v: 1275 },
          {
            k: "min",
            v: {
              __block__: [
                { k: "width", v: "100%%" },
                { k: "height", v: "100%%" },
              ],
            },
          },
        ],
      },
    },
  } as Partial<GuiNode>);

describe("a size's min", () => {
  it("reads the floor out of the verbatim tokens", () => {
    expect(sizeMin(stripes())).toEqual({ width: "100%%", height: "100%%" });
    expect(sizeMin(node({ size: { width: 10, height: 10 } }))).toBeNull();
  });

  it("makes the box at least its parent", () => {
    const root = node({
      size: { width: 1882, height: 1000 },
      children: [stripes()],
    });
    const rects = measureWindow(root, {}, 1920, 1080);
    expect(rects["name:techtree_stripes"]).toMatchObject({ w: 1882, h: 1275 });
  });
});
